// Package main is the entry point for the syslog receiver application.
// It wires together all components: configuration, syslog server, worker
// dispatcher, and event handlers (file logger, Telegram notifier).
package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/dhanu/syslog-receiver/internal/config"
	"github.com/dhanu/syslog-receiver/internal/server"
	"github.com/dhanu/syslog-receiver/internal/worker"
)

func main() {
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
	log.Println("=== MikroTik Syslog Receiver & DDoS Alerting ===")

	// ── 1. Load configuration ──────────────────────────────────────────
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load configuration: %v", err)
	}
	log.Println("[main] configuration loaded successfully")

	// ── 2. Create worker dispatcher ────────────────────────────────────
	dispatcher := worker.NewDispatcher(cfg.WorkerBufferSize)

	// ── 3. Register file logger handler ────────────────────────────────
	fileLogger, err := worker.NewFileLogger(
		cfg.LogFilePath,
		cfg.LogMaxSizeMB,
		cfg.LogMaxBackups,
		cfg.LogMaxAgeDays,
	)
	if err != nil {
		log.Fatalf("failed to create file logger: %v", err)
	}
	defer fileLogger.Close()
	dispatcher.Register(fileLogger)

	// ── 4. Register Telegram notifier handler ──────────────────────────
	telegramNotifier := worker.NewTelegramNotifier(
		cfg.TelegramBotToken,
		cfg.TelegramChatID,
		cfg.TelegramThrottleSec,
	)
	dispatcher.Register(telegramNotifier)

	// ── 5. Create and boot syslog server ───────────────────────────────
	srv, err := server.New(cfg, dispatcher.EventChannel())
	if err != nil {
		log.Fatalf("failed to create syslog server: %v", err)
	}

	if err := srv.Boot(); err != nil {
		log.Fatalf("failed to boot syslog server: %v", err)
	}

	// ── 6. Start dispatcher in background ──────────────────────────────
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go dispatcher.Start(ctx)

	log.Printf("[main] syslog receiver is running on %s", cfg.SyslogListenAddr)
	log.Println("[main] press Ctrl+C to stop")

	// ── 7. Wait for shutdown signal ────────────────────────────────────
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	sig := <-sigCh
	log.Printf("[main] received signal: %s, shutting down...", sig)

	// ── 8. Graceful shutdown ───────────────────────────────────────────
	cancel() // Signal dispatcher to stop.

	if err := srv.Kill(); err != nil {
		log.Printf("[main] error shutting down syslog server: %v", err)
	}

	log.Println("[main] shutdown complete")
}
