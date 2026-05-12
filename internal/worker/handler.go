// Package worker provides the event processing pipeline for syslog events.
// It defines a Handler interface and a Dispatcher that fans out events
// to all registered handlers concurrently without blocking the syslog receiver.
package worker

import (
	"context"

	"github.com/dhanu/syslog-receiver/internal/model"
)

// Handler defines the interface for processing syslog events.
// Each handler is responsible for a specific action (e.g., file logging, Telegram notification).
type Handler interface {
	// Handle processes a single log event. Implementations should be safe for concurrent use.
	Handle(ctx context.Context, event model.LogEvent) error

	// Name returns a human-readable name for this handler, used in logging.
	Name() string
}
