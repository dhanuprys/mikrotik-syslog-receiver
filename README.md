# MikroTik Syslog Receiver & DDoS Alerting

A Go application that receives syslog messages from a MikroTik router, detects DDoS attack alerts based on a configurable log prefix, sends throttled notifications via Telegram Bot, and persists all logs to a rotating file for future analysis.

## Features

- **UDP Syslog Server** — Listens for incoming syslog messages using [go-syslog](https://github.com/mcuadros/go-syslog) with automatic format detection (RFC3164, RFC5424).
- **Non-blocking Pipeline** — Syslog reception is never blocked; events are dispatched to workers via a buffered channel.
- **DDoS Detection** — Matches incoming log messages against a configurable prefix (e.g. `DDOS-ALERT:`).
- **Telegram Notifications** — Sends Markdown-formatted alerts to a Telegram chat with smart throttling (batches rapid events).
- **Log Rotation** — Automatic file rotation based on size, backup count, and age via [lumberjack](https://github.com/natefinch/lumberjack).
- **Centralized Configuration** — All settings loaded from `.env` file via [godotenv](https://github.com/joho/godotenv).

## Architecture

```
MikroTik Router ──UDP──▶ Syslog Server ──channel──▶ Dispatcher ──▶ File Logger (all events)
                                                                ──▶ Telegram Notifier (DDoS only, throttled)
```

## Quick Start

### Prerequisites

- Go 1.22+
- A Telegram Bot token (from [@BotFather](https://t.me/BotFather))
- The chat ID of your target Telegram group/chat

### Setup

1. **Clone and install dependencies:**

   ```bash
   git clone <repository-url>
   cd syslog-receiver
   go mod download
   ```

2. **Configure environment:**

   ```bash
   cp .env.example .env
   # Edit .env with your Telegram bot token and chat ID
   ```

3. **Build and run:**

   ```bash
   make run
   ```

   Or without Make:

   ```bash
   go build -o bin/syslog-receiver ./cmd/syslog-receiver
   ./bin/syslog-receiver
   ```

### Testing with a manual syslog message

Send a test DDoS alert:

```bash
echo "<134>May 12 14:23:57 router-gw firewall: DDOS-ALERT: input: in:ether1, proto TCP, 1.2.3.4:12345->10.0.0.1:80" | nc -u -w1 127.0.0.1 1514
```

Send a regular (non-DDoS) log:

```bash
echo "<134>May 12 14:24:00 router-gw system: user admin logged in" | nc -u -w1 127.0.0.1 1514
```

## MikroTik Configuration

On your MikroTik router, configure the firewall rule with a log prefix:

```routeros
/ip firewall filter
add chain=input action=log log=yes log-prefix="DDOS-ALERT: " \
    protocol=tcp tcp-flags=syn connection-limit=30,32 \
    comment="Log potential SYN flood"
```

Then configure remote syslog:

```routeros
/system logging action
set remote bsd-syslog=yes remote=<server-ip> remote-port=1514

/system logging
add topics=firewall action=remote
```

## Configuration Reference

| Variable | Default | Description |
|---|---|---|
| `SYSLOG_LISTEN_ADDR` | `0.0.0.0:1514` | UDP address to bind |
| `SYSLOG_FORMAT` | `auto` | `auto` / `rfc3164` / `rfc5424` |
| `DDOS_PREFIX` | `DDOS-ALERT:` | MikroTik log prefix to match |
| `TELEGRAM_BOT_TOKEN` | *(required)* | Telegram Bot API token |
| `TELEGRAM_CHAT_ID` | *(required)* | Target chat/group ID |
| `TELEGRAM_THROTTLE_SEC` | `5` | Min seconds between messages |
| `LOG_FILE_PATH` | `./logs/syslog.log` | Log file path |
| `LOG_MAX_SIZE_MB` | `100` | Max file size before rotation |
| `LOG_MAX_BACKUPS` | `7` | Max rotated files to keep |
| `LOG_MAX_AGE_DAYS` | `30` | Max days to retain logs |
| `WORKER_BUFFER_SIZE` | `1000` | Event channel buffer size |

## Project Structure

```
syslog-receiver/
├── cmd/syslog-receiver/main.go   # Application entry point
├── internal/
│   ├── config/config.go          # Configuration loader
│   ├── model/event.go            # LogEvent data type
│   ├── server/server.go          # Syslog server wrapper
│   └── worker/
│       ├── handler.go            # Handler interface
│       ├── dispatcher.go         # Event fan-out dispatcher
│       ├── filelogger.go         # File logging handler
│       └── telegram.go           # Telegram notification handler
├── .env.example                  # Environment variable template
├── Makefile                      # Build shortcuts
└── README.md
```

## Development

```bash
make build    # Compile binary
make run      # Build and run
make test     # Run tests with race detection
make lint     # Run golangci-lint
make clean    # Remove build artifacts
```

## License

MIT
