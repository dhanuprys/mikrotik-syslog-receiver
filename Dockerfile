# ============================================
# Stage 1: Build
# ============================================
FROM golang:1.22-alpine AS builder

WORKDIR /build

# Install git (required for fetching Go modules from VCS).
RUN apk add --no-cache git

# Cache dependencies first for faster rebuilds.
COPY go.mod go.sum ./
RUN go mod download

# Copy source and build a static binary.
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags="-s -w" -o syslog-receiver ./cmd/syslog-receiver

# ============================================
# Stage 2: Runtime (minimal image)
# ============================================
FROM alpine:3.20

# Add CA certs (needed for HTTPS calls to Telegram API)
# and tzdata (for proper log timestamps).
RUN apk add --no-cache ca-certificates tzdata

# Run as non-root for security.
RUN adduser -D -H -s /sbin/nologin appuser

WORKDIR /app

# Copy the binary from the builder stage.
COPY --from=builder /build/syslog-receiver .

# Create logs directory with proper ownership.
RUN mkdir -p /app/logs && chown -R appuser:appuser /app

USER appuser

# Expose the syslog UDP port.
EXPOSE 1514/udp

ENTRYPOINT ["./syslog-receiver"]
