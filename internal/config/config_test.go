package config

import (
	"os"
	"testing"
)

func TestLoad_RequiredFields(t *testing.T) {
	// Clear any existing env vars.
	clearEnv()

	_, err := Load()
	if err == nil {
		t.Fatal("expected error when required fields are missing")
	}

	// Verify the error mentions both required fields.
	errStr := err.Error()
	if !contains(errStr, "TELEGRAM_BOT_TOKEN") {
		t.Errorf("expected error to mention TELEGRAM_BOT_TOKEN, got: %s", errStr)
	}
	if !contains(errStr, "TELEGRAM_CHAT_ID") {
		t.Errorf("expected error to mention TELEGRAM_CHAT_ID, got: %s", errStr)
	}
}

func TestLoad_Defaults(t *testing.T) {
	clearEnv()

	// Set only the required fields.
	os.Setenv("TELEGRAM_BOT_TOKEN", "test-token")
	os.Setenv("TELEGRAM_CHAT_ID", "123456")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assertEqual(t, "SyslogListenAddr", cfg.SyslogListenAddr, "0.0.0.0:1514")
	assertEqual(t, "SyslogFormat", cfg.SyslogFormat, "auto")
	assertEqual(t, "DDoSPrefix", cfg.DDoSPrefix, "DDOS-ALERT:")
	assertEqual(t, "LogFilePath", cfg.LogFilePath, "./logs/syslog.log")
	assertEqualInt(t, "TelegramThrottleSec", cfg.TelegramThrottleSec, 5)
	assertEqualInt(t, "LogMaxSizeMB", cfg.LogMaxSizeMB, 100)
	assertEqualInt(t, "LogMaxBackups", cfg.LogMaxBackups, 7)
	assertEqualInt(t, "LogMaxAgeDays", cfg.LogMaxAgeDays, 30)
	assertEqualInt(t, "WorkerBufferSize", cfg.WorkerBufferSize, 1000)
}

func TestLoad_CustomValues(t *testing.T) {
	clearEnv()

	os.Setenv("TELEGRAM_BOT_TOKEN", "custom-token")
	os.Setenv("TELEGRAM_CHAT_ID", "999")
	os.Setenv("SYSLOG_LISTEN_ADDR", "127.0.0.1:5514")
	os.Setenv("SYSLOG_FORMAT", "rfc5424")
	os.Setenv("DDOS_PREFIX", "ATTACK:")
	os.Setenv("TELEGRAM_THROTTLE_SEC", "10")
	os.Setenv("LOG_FILE_PATH", "/var/log/custom.log")
	os.Setenv("LOG_MAX_SIZE_MB", "50")
	os.Setenv("LOG_MAX_BACKUPS", "3")
	os.Setenv("LOG_MAX_AGE_DAYS", "14")
	os.Setenv("WORKER_BUFFER_SIZE", "500")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assertEqual(t, "TelegramBotToken", cfg.TelegramBotToken, "custom-token")
	assertEqual(t, "TelegramChatID", cfg.TelegramChatID, "999")
	assertEqual(t, "SyslogListenAddr", cfg.SyslogListenAddr, "127.0.0.1:5514")
	assertEqual(t, "SyslogFormat", cfg.SyslogFormat, "rfc5424")
	assertEqual(t, "DDoSPrefix", cfg.DDoSPrefix, "ATTACK:")
	assertEqualInt(t, "TelegramThrottleSec", cfg.TelegramThrottleSec, 10)
	assertEqual(t, "LogFilePath", cfg.LogFilePath, "/var/log/custom.log")
	assertEqualInt(t, "LogMaxSizeMB", cfg.LogMaxSizeMB, 50)
	assertEqualInt(t, "LogMaxBackups", cfg.LogMaxBackups, 3)
	assertEqualInt(t, "LogMaxAgeDays", cfg.LogMaxAgeDays, 14)
	assertEqualInt(t, "WorkerBufferSize", cfg.WorkerBufferSize, 500)
}

func TestLoad_InvalidIntegerValue(t *testing.T) {
	clearEnv()

	os.Setenv("TELEGRAM_BOT_TOKEN", "test-token")
	os.Setenv("TELEGRAM_CHAT_ID", "123456")
	os.Setenv("TELEGRAM_THROTTLE_SEC", "not-a-number")

	_, err := Load()
	if err == nil {
		t.Fatal("expected error for invalid integer value")
	}

	if !contains(err.Error(), "TELEGRAM_THROTTLE_SEC") {
		t.Errorf("expected error to mention TELEGRAM_THROTTLE_SEC, got: %s", err.Error())
	}
}

// --- helpers ---

func clearEnv() {
	envVars := []string{
		"TELEGRAM_BOT_TOKEN", "TELEGRAM_CHAT_ID",
		"SYSLOG_LISTEN_ADDR", "SYSLOG_FORMAT", "DDOS_PREFIX",
		"TELEGRAM_THROTTLE_SEC", "LOG_FILE_PATH",
		"LOG_MAX_SIZE_MB", "LOG_MAX_BACKUPS", "LOG_MAX_AGE_DAYS",
		"WORKER_BUFFER_SIZE",
	}
	for _, v := range envVars {
		os.Unsetenv(v)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsSubstr(s, substr))
}

func containsSubstr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func assertEqual(t *testing.T, field, got, want string) {
	t.Helper()
	if got != want {
		t.Errorf("%s = %q, want %q", field, got, want)
	}
}

func assertEqualInt(t *testing.T, field string, got, want int) {
	t.Helper()
	if got != want {
		t.Errorf("%s = %d, want %d", field, got, want)
	}
}
