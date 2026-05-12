package model

import (
	"strings"
	"testing"
	"time"
)

func TestLogEvent_FormatLogLine(t *testing.T) {
	ts := time.Date(2026, 5, 12, 14, 23, 57, 0, time.UTC)

	tests := []struct {
		name     string
		event    LogEvent
		contains []string
	}{
		{
			name: "regular event",
			event: LogEvent{
				Timestamp: ts,
				Hostname:  "router-gw",
				Tag:       "firewall",
				Content:   "user admin logged in",
				IsDDoS:    false,
			},
			contains: []string{"2026-05-12T14:23:57Z", "router-gw", "firewall", "user admin logged in"},
		},
		{
			name: "ddos event",
			event: LogEvent{
				Timestamp: ts,
				Hostname:  "router-gw",
				Tag:       "firewall",
				Content:   "DDOS-ALERT: SYN flood detected",
				IsDDoS:    true,
			},
			contains: []string{"2026-05-12T14:23:57Z", "router-gw", "DDOS-ALERT:", "[DDOS]"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			line := tt.event.FormatLogLine()
			for _, s := range tt.contains {
				if !strings.Contains(line, s) {
					t.Errorf("FormatLogLine() = %q, expected to contain %q", line, s)
				}
			}
		})
	}
}

func TestLogEvent_FormatLogLine_NoDDoSMarker(t *testing.T) {
	event := LogEvent{
		Timestamp: time.Now(),
		Hostname:  "test",
		Tag:       "test",
		Content:   "normal message",
		IsDDoS:    false,
	}

	line := event.FormatLogLine()
	if strings.Contains(line, "[DDOS]") {
		t.Errorf("non-DDoS event should not contain [DDOS] marker, got: %s", line)
	}
}

func TestLogEvent_FormatTelegramMessage(t *testing.T) {
	ts := time.Date(2026, 5, 12, 14, 23, 57, 0, time.UTC)

	event := LogEvent{
		Timestamp: ts,
		Hostname:  "router-gw",
		Tag:       "firewall",
		Content:   "DDOS-ALERT: SYN flood from 1.2.3.4",
		IsDDoS:    true,
	}

	msg := event.FormatTelegramMessage()

	expectedParts := []string{
		"🚨 *DDoS Alert Detected*",
		"router-gw",
		"firewall",
		"DDOS-ALERT: SYN flood from 1.2.3.4",
	}

	for _, part := range expectedParts {
		if !strings.Contains(msg, part) {
			t.Errorf("FormatTelegramMessage() missing %q, got:\n%s", part, msg)
		}
	}
}
