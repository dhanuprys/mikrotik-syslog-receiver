package worker

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/dhanu/syslog-receiver/internal/model"
)

const telegramAPIURL = "https://api.telegram.org/bot%s/sendMessage"

// telegramPayload is the JSON body sent to the Telegram Bot API.
type telegramPayload struct {
	ChatID    string `json:"chat_id"`
	Text      string `json:"text"`
	ParseMode string `json:"parse_mode"`
}

// TelegramNotifier sends DDoS alert messages to a Telegram chat.
// It implements throttling to prevent exceeding Telegram's rate limits:
// only one message is sent per throttle interval. If multiple DDoS events
// arrive within the interval, they are counted and a summary is sent.
type TelegramNotifier struct {
	botToken    string
	chatID      string
	throttleSec int
	client      *http.Client

	mu         sync.Mutex
	lastSentAt time.Time
	pending    []model.LogEvent // buffered events during throttle window
}

// NewTelegramNotifier creates a new TelegramNotifier.
// throttleSec controls the minimum interval between sent messages.
func NewTelegramNotifier(botToken, chatID string, throttleSec int) *TelegramNotifier {
	return &TelegramNotifier{
		botToken:    botToken,
		chatID:      chatID,
		throttleSec: throttleSec,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// Name returns the handler name.
func (t *TelegramNotifier) Name() string {
	return "telegram-notifier"
}

// Handle processes a log event. Only DDoS events trigger notifications.
// If the throttle window has not elapsed, the event is buffered and a
// deferred flush is scheduled.
func (t *TelegramNotifier) Handle(ctx context.Context, event model.LogEvent) error {
	if !event.IsDDoS {
		return nil
	}

	t.mu.Lock()

	elapsed := time.Since(t.lastSentAt)
	throttle := time.Duration(t.throttleSec) * time.Second

	if elapsed >= throttle && len(t.pending) == 0 {
		// No throttle active and no pending events — send immediately.
		t.lastSentAt = time.Now()
		t.mu.Unlock()

		return t.send(ctx, event.FormatTelegramMessage())
	}

	// Throttle is active — buffer the event.
	t.pending = append(t.pending, event)
	shouldScheduleFlush := len(t.pending) == 1 // schedule only on first buffered event

	t.mu.Unlock()

	if shouldScheduleFlush {
		go t.scheduleFlush(ctx, throttle-elapsed)
	}

	return nil
}

// telegramMaxMessageLength is the maximum number of characters allowed in a single Telegram message.
const telegramMaxMessageLength = 4096

// scheduleFlush waits for the remaining throttle duration, then flushes all pending events.
func (t *TelegramNotifier) scheduleFlush(ctx context.Context, wait time.Duration) {
	if wait > 0 {
		select {
		case <-time.After(wait):
		case <-ctx.Done():
			return
		}
	}

	t.mu.Lock()
	events := t.pending
	t.pending = nil
	t.lastSentAt = time.Now()
	t.mu.Unlock()

	if len(events) == 0 {
		return
	}

	messages := t.buildMessages(events)
	for i, msg := range messages {
		if err := t.send(ctx, msg); err != nil {
			log.Printf("[telegram] failed to send message %d/%d: %v", i+1, len(messages), err)
		}
	}
}

// buildMessages splits events into one or more Telegram messages, each under the 4096 char limit.
func (t *TelegramNotifier) buildMessages(events []model.LogEvent) []string {
	if len(events) == 1 {
		return []string{events[0].FormatTelegramMessage()}
	}

	var messages []string
	header := fmt.Sprintf("🚨 *%d DDoS Alerts Detected*\n\n", len(events))
	current := header
	count := 0

	for i, e := range events {
		entry := fmt.Sprintf(
			"*#%d* — `%s`\n🖥️ `%s` | 🏷️ `%s`\n```\n%s\n```\n\n",
			i+1,
			e.Timestamp.Format(time.RFC3339),
			model.EscapeMarkdown(e.Hostname),
			model.EscapeMarkdown(e.Tag),
			model.EscapeMarkdown(e.Content),
		)

		// If adding this entry would exceed the limit, flush current message and start a new one.
		if len(current)+len(entry) > telegramMaxMessageLength && count > 0 {
			messages = append(messages, current)
			current = "🚨 *DDoS Alerts (continued)*\n\n"
			count = 0
		}

		current += entry
		count++
	}

	// Append the last chunk.
	if count > 0 {
		messages = append(messages, current)
	}

	return messages
}

// send posts a message to the Telegram Bot API.
func (t *TelegramNotifier) send(ctx context.Context, text string) error {
	payload := telegramPayload{
		ChatID:    t.chatID,
		Text:      text,
		ParseMode: "Markdown",
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal telegram payload: %w", err)
	}

	url := fmt.Sprintf(telegramAPIURL, t.botToken)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create telegram request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := t.client.Do(req)
	if err != nil {
		return fmt.Errorf("send telegram request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		log.Printf("[telegram] API error response (status %d): %s", resp.StatusCode, string(respBody))
		return fmt.Errorf("telegram API returned status %d: %s", resp.StatusCode, string(respBody))
	}

	log.Printf("[telegram] message sent successfully to chat %s", t.chatID)
	return nil
}
