// Package model defines the core data types shared across the application.
package model

import (
	"fmt"
	"time"
)

// LogEvent represents a parsed syslog message flowing through the worker pipeline.
type LogEvent struct {
	// Timestamp is the time the syslog message was generated.
	Timestamp time.Time

	// Hostname is the originating host reported in the syslog message.
	Hostname string

	// Tag is the syslog tag/app-name field.
	Tag string

	// Content is the actual log message body.
	Content string

	// Severity is the syslog severity level (0-7).
	Severity int

	// Facility is the syslog facility code.
	Facility int

	// IsDDoS indicates whether this event matched the configured DDoS prefix.
	IsDDoS bool
}

// FormatLogLine returns a structured single-line representation suitable for file logging.
func (e LogEvent) FormatLogLine() string {
	ddosMarker := ""
	if e.IsDDoS {
		ddosMarker = " [DDOS]"
	}
	return fmt.Sprintf("[%s] %s %s: %s%s",
		e.Timestamp.Format(time.RFC3339),
		e.Hostname,
		e.Tag,
		e.Content,
		ddosMarker,
	)
}

// FormatTelegramMessage returns a Markdown-formatted message for Telegram notifications.
func (e LogEvent) FormatTelegramMessage() string {
	return fmt.Sprintf(
		"🚨 *DDoS Alert Detected*\n\n"+
			"📅 *Time:* `%s`\n"+
			"🖥️ *Host:* `%s`\n"+
			"🏷️ *Tag:* `%s`\n"+
			"📝 *Details:*\n```\n%s\n```",
		e.Timestamp.Format(time.RFC3339),
		e.Hostname,
		e.Tag,
		e.Content,
	)
}
