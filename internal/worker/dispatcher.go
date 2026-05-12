package worker

import (
	"context"
	"log"

	"github.com/dhanu/syslog-receiver/internal/model"
)

// Dispatcher receives log events from the syslog server and fans them out
// to all registered handlers. Each handler is invoked in its own goroutine
// to prevent slow handlers from blocking the pipeline.
type Dispatcher struct {
	eventCh  chan model.LogEvent
	handlers []Handler
}

// NewDispatcher creates a new Dispatcher with a buffered event channel.
// The bufferSize controls how many events can be queued before the syslog
// server goroutine blocks (providing backpressure).
func NewDispatcher(bufferSize int) *Dispatcher {
	return &Dispatcher{
		eventCh: make(chan model.LogEvent, bufferSize),
	}
}

// Register adds a handler to the dispatcher's fan-out list.
// Must be called before Start.
func (d *Dispatcher) Register(h Handler) {
	d.handlers = append(d.handlers, h)
	log.Printf("[dispatcher] registered handler: %s", h.Name())
}

// EventChannel returns the channel that the syslog server should send events to.
func (d *Dispatcher) EventChannel() chan<- model.LogEvent {
	return d.eventCh
}

// Start begins consuming events from the channel and dispatching them to all
// registered handlers. It blocks until the context is cancelled or the event
// channel is closed. Call this in a goroutine.
func (d *Dispatcher) Start(ctx context.Context) {
	log.Printf("[dispatcher] started with %d handler(s), buffer size: %d", len(d.handlers), cap(d.eventCh))

	for {
		select {
		case <-ctx.Done():
			log.Println("[dispatcher] shutting down: context cancelled")
			d.drain(ctx)
			return

		case event, ok := <-d.eventCh:
			if !ok {
				log.Println("[dispatcher] shutting down: event channel closed")
				return
			}
			d.dispatch(ctx, event)
		}
	}
}

// dispatch fans out a single event to all handlers concurrently.
func (d *Dispatcher) dispatch(ctx context.Context, event model.LogEvent) {
	for _, h := range d.handlers {
		go func(handler Handler) {
			if err := handler.Handle(ctx, event); err != nil {
				log.Printf("[dispatcher] handler %q error: %v", handler.Name(), err)
			}
		}(h)
	}
}

// drain processes any remaining events in the channel buffer after shutdown signal.
func (d *Dispatcher) drain(ctx context.Context) {
	remaining := len(d.eventCh)
	if remaining == 0 {
		return
	}

	log.Printf("[dispatcher] draining %d remaining event(s)", remaining)
	for i := 0; i < remaining; i++ {
		event, ok := <-d.eventCh
		if !ok {
			break
		}
		d.dispatch(ctx, event)
	}
	log.Println("[dispatcher] drain complete")
}
