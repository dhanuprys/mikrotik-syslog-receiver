package worker

import (
	"context"
	"fmt"
	"log"

	"github.com/dhanu/syslog-receiver/internal/model"
	"gopkg.in/natefinch/lumberjack.v2"
)

// FileLogger writes all incoming log events to a file with automatic rotation.
// It uses lumberjack for log rotation based on file size, backup count, and age.
type FileLogger struct {
	logger *log.Logger
	writer *lumberjack.Logger
}

// NewFileLogger creates a new FileLogger that writes to the specified file path.
// Parameters control log rotation behavior:
//   - filePath: path to the log file (directories are created automatically by lumberjack)
//   - maxSizeMB: maximum size in megabytes before rotation
//   - maxBackups: maximum number of old log files to retain
//   - maxAgeDays: maximum number of days to retain old log files
func NewFileLogger(filePath string, maxSizeMB, maxBackups, maxAgeDays int) (*FileLogger, error) {
	lj := &lumberjack.Logger{
		Filename:   filePath,
		MaxSize:    maxSizeMB,
		MaxBackups: maxBackups,
		MaxAge:     maxAgeDays,
		LocalTime:  true,
		Compress:   true,
	}

	// Perform a write test to catch permission/path errors early.
	if _, err := lj.Write([]byte("")); err != nil {
		return nil, fmt.Errorf("unable to write to log file %s: %w", filePath, err)
	}

	return &FileLogger{
		logger: log.New(lj, "", 0), // No prefix/flags — we format our own lines.
		writer: lj,
	}, nil
}

// Name returns the handler name.
func (f *FileLogger) Name() string {
	return "file-logger"
}

// Handle writes the log event to the file. All events are logged, not just DDoS.
func (f *FileLogger) Handle(_ context.Context, event model.LogEvent) error {
	f.logger.Println(event.FormatLogLine())
	return nil
}

// Close gracefully closes the underlying log file writer.
func (f *FileLogger) Close() error {
	log.Println("[file-logger] closing log file")
	return f.writer.Close()
}
