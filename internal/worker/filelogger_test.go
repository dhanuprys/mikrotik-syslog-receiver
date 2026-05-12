package worker

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/dhanu/syslog-receiver/internal/model"
)

func TestFileLogger_WritesEvents(t *testing.T) {
	// Create a temp directory for the test log file.
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "test.log")

	fl, err := NewFileLogger(logPath, 10, 1, 1)
	if err != nil {
		t.Fatalf("failed to create FileLogger: %v", err)
	}
	defer fl.Close()

	event := model.LogEvent{
		Timestamp: time.Date(2026, 5, 12, 14, 23, 57, 0, time.UTC),
		Hostname:  "router-gw",
		Tag:       "firewall",
		Content:   "DDOS-ALERT: SYN flood detected",
		IsDDoS:    true,
	}

	if err := fl.Handle(context.Background(), event); err != nil {
		t.Fatalf("Handle() error: %v", err)
	}

	// Read the log file and verify contents.
	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("failed to read log file: %v", err)
	}

	content := string(data)
	expectedParts := []string{
		"2026-05-12T14:23:57Z",
		"router-gw",
		"firewall",
		"DDOS-ALERT: SYN flood detected",
		"[DDOS]",
	}

	for _, part := range expectedParts {
		if !strings.Contains(content, part) {
			t.Errorf("log file missing %q, got:\n%s", part, content)
		}
	}
}

func TestFileLogger_NonDDoSEvent(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "test.log")

	fl, err := NewFileLogger(logPath, 10, 1, 1)
	if err != nil {
		t.Fatalf("failed to create FileLogger: %v", err)
	}
	defer fl.Close()

	event := model.LogEvent{
		Timestamp: time.Now(),
		Hostname:  "router-gw",
		Tag:       "system",
		Content:   "user admin logged in",
		IsDDoS:    false,
	}

	if err := fl.Handle(context.Background(), event); err != nil {
		t.Fatalf("Handle() error: %v", err)
	}

	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("failed to read log file: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "user admin logged in") {
		t.Error("non-DDoS events should be written to log")
	}
	if strings.Contains(content, "[DDOS]") {
		t.Error("non-DDoS events should not have [DDOS] marker")
	}
}

func TestFileLogger_Name(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "test.log")

	fl, err := NewFileLogger(logPath, 10, 1, 1)
	if err != nil {
		t.Fatalf("failed to create FileLogger: %v", err)
	}
	defer fl.Close()

	if fl.Name() != "file-logger" {
		t.Errorf("Name() = %q, want %q", fl.Name(), "file-logger")
	}
}
