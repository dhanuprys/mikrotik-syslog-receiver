// Package config provides centralized configuration management.
// It loads values from a .env file (if present) and environment variables,
// parses them into a typed Config struct, and validates required fields.
// Configuration is loaded once at startup and passed throughout the application.
package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

// Config holds all application configuration values.
type Config struct {
	// SyslogListenAddr is the UDP address to bind the syslog server (e.g. "0.0.0.0:1514").
	SyslogListenAddr string

	// SyslogFormat specifies the syslog format: "auto", "rfc3164", or "rfc5424".
	SyslogFormat string

	// DDoSPrefix is the MikroTik log-prefix string that identifies DDoS alerts.
	DDoSPrefix string

	// TelegramBotToken is the Telegram Bot API token from @BotFather.
	TelegramBotToken string

	// TelegramChatID is the target chat or group ID for notifications.
	TelegramChatID string

	// TelegramThrottleSec is the minimum interval (in seconds) between Telegram messages.
	TelegramThrottleSec int

	// LogFilePath is the file path where syslog events are persisted.
	LogFilePath string

	// LogMaxSizeMB is the maximum size of a single log file in megabytes before rotation.
	LogMaxSizeMB int

	// LogMaxBackups is the maximum number of old rotated log files to retain.
	LogMaxBackups int

	// LogMaxAgeDays is the maximum number of days to retain old log files.
	LogMaxAgeDays int

	// WorkerBufferSize is the capacity of the internal event channel buffer.
	WorkerBufferSize int
}

// Load reads configuration from the .env file and environment variables,
// parses them into a Config struct, and validates required fields.
// It returns an error if any required field is missing or a numeric field is invalid.
func Load() (*Config, error) {
	// Load .env file if present; ignore error if file doesn't exist (e.g. production).
	_ = godotenv.Load()

	cfg := &Config{}
	var errs []error

	// --- Required fields ---
	cfg.TelegramBotToken = os.Getenv("TELEGRAM_BOT_TOKEN")
	if cfg.TelegramBotToken == "" {
		errs = append(errs, fmt.Errorf("TELEGRAM_BOT_TOKEN is required"))
	}

	cfg.TelegramChatID = os.Getenv("TELEGRAM_CHAT_ID")
	if cfg.TelegramChatID == "" {
		errs = append(errs, fmt.Errorf("TELEGRAM_CHAT_ID is required"))
	}

	// --- Optional fields with defaults ---
	cfg.SyslogListenAddr = getEnvOrDefault("SYSLOG_LISTEN_ADDR", "0.0.0.0:1514")
	cfg.SyslogFormat = getEnvOrDefault("SYSLOG_FORMAT", "auto")
	cfg.DDoSPrefix = getEnvOrDefault("DDOS_PREFIX", "DDOS-ALERT:")
	cfg.LogFilePath = getEnvOrDefault("LOG_FILE_PATH", "./logs/syslog.log")

	// --- Numeric fields ---
	var err error

	cfg.TelegramThrottleSec, err = getEnvIntOrDefault("TELEGRAM_THROTTLE_SEC", 5)
	if err != nil {
		errs = append(errs, fmt.Errorf("TELEGRAM_THROTTLE_SEC: %w", err))
	}

	cfg.LogMaxSizeMB, err = getEnvIntOrDefault("LOG_MAX_SIZE_MB", 100)
	if err != nil {
		errs = append(errs, fmt.Errorf("LOG_MAX_SIZE_MB: %w", err))
	}

	cfg.LogMaxBackups, err = getEnvIntOrDefault("LOG_MAX_BACKUPS", 7)
	if err != nil {
		errs = append(errs, fmt.Errorf("LOG_MAX_BACKUPS: %w", err))
	}

	cfg.LogMaxAgeDays, err = getEnvIntOrDefault("LOG_MAX_AGE_DAYS", 30)
	if err != nil {
		errs = append(errs, fmt.Errorf("LOG_MAX_AGE_DAYS: %w", err))
	}

	cfg.WorkerBufferSize, err = getEnvIntOrDefault("WORKER_BUFFER_SIZE", 1000)
	if err != nil {
		errs = append(errs, fmt.Errorf("WORKER_BUFFER_SIZE: %w", err))
	}

	if len(errs) > 0 {
		return nil, fmt.Errorf("configuration errors: %w", errors.Join(errs...))
	}

	return cfg, nil
}

// getEnvOrDefault returns the environment variable value, or the provided default if unset.
func getEnvOrDefault(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

// getEnvIntOrDefault returns an integer from the environment variable, or the default if unset.
// Returns an error if the value is present but not a valid integer.
func getEnvIntOrDefault(key string, defaultVal int) (int, error) {
	val := os.Getenv(key)
	if val == "" {
		return defaultVal, nil
	}

	parsed, err := strconv.Atoi(val)
	if err != nil {
		return 0, fmt.Errorf("invalid integer value %q", val)
	}
	return parsed, nil
}
