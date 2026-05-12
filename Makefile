.PHONY: build run test lint clean

# Binary output directory
BIN_DIR := bin
BINARY  := $(BIN_DIR)/syslog-receiver

## build: Compile the application binary.
build:
	@echo "==> Building..."
	@mkdir -p $(BIN_DIR)
	go build -o $(BINARY) ./cmd/syslog-receiver
	@echo "==> Built: $(BINARY)"

## run: Build and run the application.
run: build
	@echo "==> Running..."
	./$(BINARY)

## test: Run all tests with race detection.
test:
	@echo "==> Running tests..."
	go test ./... -v -race -count=1

## lint: Run golangci-lint (must be installed).
lint:
	@echo "==> Linting..."
	golangci-lint run ./...

## clean: Remove build artifacts and logs.
clean:
	@echo "==> Cleaning..."
	rm -rf $(BIN_DIR) logs/
	@echo "==> Clean complete"

## help: Show available targets.
help:
	@echo "Available targets:"
	@grep -E '^## ' Makefile | sed 's/^## /  /'
