package worker

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/dhanu/syslog-receiver/internal/model"
)

func TestTelegramNotifier_OnlyDDoSEvents(t *testing.T) {
	var callCount atomic.Int32

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount.Add(1)
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]bool{"ok": true})
	}))
	defer srv.Close()

	notifier := newTestNotifier(srv.URL, 0)

	// Non-DDoS event should not trigger a send.
	err := notifier.Handle(context.Background(), model.LogEvent{
		Timestamp: time.Now(),
		Content:   "normal log",
		IsDDoS:    false,
	})
	if err != nil {
		t.Fatalf("Handle() error: %v", err)
	}

	time.Sleep(100 * time.Millisecond)
	if callCount.Load() != 0 {
		t.Errorf("non-DDoS event triggered %d API call(s), want 0", callCount.Load())
	}
}

func TestTelegramNotifier_SendsDDoSEvent(t *testing.T) {
	var callCount atomic.Int32
	var receivedPayload telegramPayload

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount.Add(1)
		json.NewDecoder(r.Body).Decode(&receivedPayload)
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]bool{"ok": true})
	}))
	defer srv.Close()

	notifier := newTestNotifier(srv.URL, 0)

	event := model.LogEvent{
		Timestamp: time.Date(2026, 5, 12, 14, 0, 0, 0, time.UTC),
		Hostname:  "router",
		Tag:       "firewall",
		Content:   "DDOS-ALERT: attack",
		IsDDoS:    true,
	}

	err := notifier.Handle(context.Background(), event)
	if err != nil {
		t.Fatalf("Handle() error: %v", err)
	}

	time.Sleep(200 * time.Millisecond)
	if callCount.Load() != 1 {
		t.Errorf("DDoS event triggered %d API call(s), want 1", callCount.Load())
	}

	if receivedPayload.ParseMode != "Markdown" {
		t.Errorf("ParseMode = %q, want %q", receivedPayload.ParseMode, "Markdown")
	}

	if receivedPayload.ChatID != "test-chat" {
		t.Errorf("ChatID = %q, want %q", receivedPayload.ChatID, "test-chat")
	}
}

func TestTelegramNotifier_Throttling(t *testing.T) {
	var callCount atomic.Int32

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount.Add(1)
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]bool{"ok": true})
	}))
	defer srv.Close()

	// 2-second throttle window.
	notifier := newTestNotifier(srv.URL, 2)

	ddosEvent := model.LogEvent{
		Timestamp: time.Now(),
		Hostname:  "router",
		Content:   "DDOS-ALERT: attack",
		IsDDoS:    true,
	}

	// First event — should send immediately.
	err := notifier.Handle(context.Background(), ddosEvent)
	if err != nil {
		t.Fatalf("first Handle() error: %v", err)
	}

	// Rapid subsequent events — should be throttled (buffered).
	for i := 0; i < 3; i++ {
		_ = notifier.Handle(context.Background(), ddosEvent)
	}

	// Only the first should have sent immediately.
	time.Sleep(200 * time.Millisecond)
	if callCount.Load() != 1 {
		t.Errorf("after rapid events, got %d API calls, want 1 (throttled)", callCount.Load())
	}

	// Wait for throttle window to expire and flush.
	time.Sleep(2500 * time.Millisecond)
	if callCount.Load() != 2 {
		t.Errorf("after throttle window, got %d API calls, want 2 (initial + batch)", callCount.Load())
	}
}

func TestTelegramNotifier_Name(t *testing.T) {
	notifier := NewTelegramNotifier("token", "chat", 5)
	if notifier.Name() != "telegram-notifier" {
		t.Errorf("Name() = %q, want %q", notifier.Name(), "telegram-notifier")
	}
}

// newTestNotifier creates a TelegramNotifier pointing at the test server.
// It replaces the API URL template to use the test server's URL.
func newTestNotifier(serverURL string, throttleSec int) *TelegramNotifier {
	n := NewTelegramNotifier("test-token", "test-chat", throttleSec)
	// Override the bot token so the URL construction points at our test server.
	n.botToken = "test"
	// We need to override the send method's URL construction.
	// Since telegramAPIURL is a const, we work around it by setting botToken
	// such that the full URL becomes: serverURL + "/bottest/sendMessage"
	// But our const uses fmt.Sprintf, so we need a different approach.
	// Instead, we'll just replace the whole URL in the client transport.
	n.client = &http.Client{
		Timeout: 5 * time.Second,
		Transport: &rewriteTransport{
			base:      http.DefaultTransport,
			targetURL: serverURL + "/sendMessage",
		},
	}
	return n
}

// rewriteTransport redirects all requests to a target URL (for testing).
type rewriteTransport struct {
	base      http.RoundTripper
	targetURL string
}

func (t *rewriteTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Replace the URL but keep headers and body.
	newReq := req.Clone(req.Context())
	parsed, _ := http.NewRequest(req.Method, t.targetURL, req.Body)
	newReq.URL = parsed.URL
	newReq.Host = parsed.URL.Host
	return t.base.RoundTrip(newReq)
}
