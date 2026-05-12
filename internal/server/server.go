// Package server manages the syslog server lifecycle.
// It listens for incoming syslog messages over UDP, parses them into LogEvent
// structs, and forwards them to the worker dispatcher's event channel.
package server

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/dhanu/syslog-receiver/internal/config"
	"github.com/dhanu/syslog-receiver/internal/model"
	"gopkg.in/mcuadros/go-syslog.v2"
	"gopkg.in/mcuadros/go-syslog.v2/format"
)

// Server wraps the go-syslog server and bridges incoming messages to the worker pipeline.
type Server struct {
	srv     *syslog.Server
	channel syslog.LogPartsChannel
	cfg     *config.Config
	eventCh chan<- model.LogEvent
}

// New creates a new syslog Server configured from the application config.
// eventCh is the dispatcher's input channel where parsed events are sent.
func New(cfg *config.Config, eventCh chan<- model.LogEvent) (*Server, error) {
	channel := make(syslog.LogPartsChannel)
	handler := syslog.NewChannelHandler(channel)

	srv := syslog.NewServer()
	srv.SetHandler(handler)

	// Configure the syslog format based on the config value.
	switch strings.ToLower(cfg.SyslogFormat) {
	case "rfc3164":
		srv.SetFormat(syslog.RFC3164)
	case "rfc5424":
		srv.SetFormat(syslog.RFC5424)
	case "rfc6587":
		srv.SetFormat(syslog.RFC6587)
	default:
		srv.SetFormat(syslog.Automatic)
	}

	// Listen on UDP.
	if err := srv.ListenUDP(cfg.SyslogListenAddr); err != nil {
		return nil, fmt.Errorf("failed to listen on UDP %s: %w", cfg.SyslogListenAddr, err)
	}

	log.Printf("[server] configured to listen on UDP %s (format: %s)", cfg.SyslogListenAddr, cfg.SyslogFormat)

	return &Server{
		srv:     srv,
		channel: channel,
		cfg:     cfg,
		eventCh: eventCh,
	}, nil
}

// Boot starts the syslog server and begins processing incoming messages.
// It spawns a goroutine that reads from the syslog channel, converts messages
// to LogEvent structs, and pushes them to the dispatcher's event channel.
// This method is non-blocking.
func (s *Server) Boot() error {
	if err := s.srv.Boot(); err != nil {
		return fmt.Errorf("failed to boot syslog server: %w", err)
	}

	log.Println("[server] syslog server booted successfully")

	go s.processMessages()

	return nil
}

// Wait blocks until the syslog server shuts down.
func (s *Server) Wait() {
	s.srv.Wait()
}

// Kill gracefully stops the syslog server.
func (s *Server) Kill() error {
	log.Println("[server] shutting down syslog server")
	return s.srv.Kill()
}

// processMessages reads from the go-syslog channel and converts each message
// into a LogEvent. It checks for the DDoS prefix and sends the event to
// the dispatcher channel. This runs in its own goroutine.
func (s *Server) processMessages() {
	log.Printf("[server] processing messages (DDoS prefix: %q)", s.cfg.DDoSPrefix)

	for logParts := range s.channel {
		event := s.parseLogParts(logParts)

		// Non-blocking send: if the channel is full, log a warning and drop.
		select {
		case s.eventCh <- event:
		default:
			log.Printf("[server] WARNING: event channel full, dropping event from %s", event.Hostname)
		}
	}
}

// parseLogParts converts a format.LogParts map into a structured LogEvent.
func (s *Server) parseLogParts(parts format.LogParts) model.LogEvent {
	event := model.LogEvent{
		Timestamp: extractTime(parts, "timestamp"),
		Hostname:  extractString(parts, "hostname"),
		Tag:       extractString(parts, "tag"),
		Content:   extractString(parts, "content"),
		Severity:  extractInt(parts, "severity"),
		Facility:  extractInt(parts, "facility"),
	}

	// If content is empty, try the "message" key (RFC5424 uses "message").
	if event.Content == "" {
		event.Content = extractString(parts, "message")
	}

	// If tag is empty, try "app_name" (RFC5424 field name).
	if event.Tag == "" {
		event.Tag = extractString(parts, "app_name")
	}

	// Check if the message content contains the DDoS prefix.
	if s.cfg.DDoSPrefix != "" && strings.Contains(event.Content, s.cfg.DDoSPrefix) {
		event.IsDDoS = true
	}

	return event
}

// extractString safely extracts a string value from the LogParts map.
func extractString(parts format.LogParts, key string) string {
	if val, ok := parts[key]; ok {
		if s, ok := val.(string); ok {
			return s
		}
	}
	return ""
}

// extractInt safely extracts an integer value from the LogParts map.
func extractInt(parts format.LogParts, key string) int {
	if val, ok := parts[key]; ok {
		if i, ok := val.(int); ok {
			return i
		}
	}
	return 0
}

// extractTime safely extracts a time.Time value from the LogParts map.
func extractTime(parts format.LogParts, key string) time.Time {
	if val, ok := parts[key]; ok {
		if t, ok := val.(time.Time); ok {
			return t
		}
	}
	return time.Now()
}
