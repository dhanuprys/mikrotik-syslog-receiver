package worker

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/dhanu/syslog-receiver/internal/model"
)

// mockHandler implements Handler for testing.
type mockHandler struct {
	name   string
	events []model.LogEvent
	mu     sync.Mutex
	count  atomic.Int32
}

func (m *mockHandler) Name() string { return m.name }

func (m *mockHandler) Handle(_ context.Context, event model.LogEvent) error {
	m.mu.Lock()
	m.events = append(m.events, event)
	m.mu.Unlock()
	m.count.Add(1)
	return nil
}

func (m *mockHandler) getEvents() []model.LogEvent {
	m.mu.Lock()
	defer m.mu.Unlock()
	result := make([]model.LogEvent, len(m.events))
	copy(result, m.events)
	return result
}

func TestDispatcher_FanOut(t *testing.T) {
	d := NewDispatcher(10)

	h1 := &mockHandler{name: "handler-1"}
	h2 := &mockHandler{name: "handler-2"}
	d.Register(h1)
	d.Register(h2)

	ctx, cancel := context.WithCancel(context.Background())
	go d.Start(ctx)

	event := model.LogEvent{
		Timestamp: time.Now(),
		Hostname:  "test-host",
		Content:   "test message",
	}

	d.EventChannel() <- event

	// Wait for handlers to process.
	deadline := time.After(2 * time.Second)
	for {
		if h1.count.Load() >= 1 && h2.count.Load() >= 1 {
			break
		}
		select {
		case <-deadline:
			t.Fatalf("timeout waiting for handlers: h1=%d, h2=%d", h1.count.Load(), h2.count.Load())
		default:
			time.Sleep(10 * time.Millisecond)
		}
	}

	cancel()

	// Verify both handlers received the event.
	events1 := h1.getEvents()
	events2 := h2.getEvents()

	if len(events1) != 1 {
		t.Errorf("handler-1 received %d events, want 1", len(events1))
	}
	if len(events2) != 1 {
		t.Errorf("handler-2 received %d events, want 1", len(events2))
	}

	if events1[0].Content != "test message" {
		t.Errorf("handler-1 got content %q, want %q", events1[0].Content, "test message")
	}
}

func TestDispatcher_MultipleEvents(t *testing.T) {
	d := NewDispatcher(10)

	h := &mockHandler{name: "handler"}
	d.Register(h)

	ctx, cancel := context.WithCancel(context.Background())
	go d.Start(ctx)

	eventCount := 5
	for i := 0; i < eventCount; i++ {
		d.EventChannel() <- model.LogEvent{
			Timestamp: time.Now(),
			Hostname:  "test-host",
			Content:   "message",
		}
	}

	// Wait for all events to be processed.
	deadline := time.After(2 * time.Second)
	for {
		if h.count.Load() >= int32(eventCount) {
			break
		}
		select {
		case <-deadline:
			t.Fatalf("timeout: handler received %d events, want %d", h.count.Load(), eventCount)
		default:
			time.Sleep(10 * time.Millisecond)
		}
	}

	cancel()

	events := h.getEvents()
	if len(events) != eventCount {
		t.Errorf("handler received %d events, want %d", len(events), eventCount)
	}
}
