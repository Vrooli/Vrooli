# Scenario-to-Desktop API Makefile

.PHONY: build run clean test deps fmt vet check install

# Variables
BINARY_NAME=scenario-to-desktop-api
BUILD_DIR=./bin
GO_FILES=$(shell find . -type f -name '*.go' -not -path "./vendor/*")
VERSION=$(shell git describe --tags --always --dirty)
BUILD_TIME=$(shell date -u +"%Y-%m-%dT%H:%M:%SZ")

# Default target
all: check build

# Install dependencies
deps:
	@echo "Installing dependencies..."
	go mod download
	go mod tidy

# Format code
fmt:
	@echo "Formatting code..."
	go fmt ./...

# Vet code
vet:
	@echo "Vetting code..."
	go vet ./...

# Run tests
test:
	@echo "Running tests..."
	go test -v ./...

# Check code (fmt, vet, test)
check: fmt vet test

# Build the binary
build: deps
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 go build \
		-ldflags "-X main.version=$(VERSION) -X main.buildTime=$(BUILD_TIME)" \
		-o $(BUILD_DIR)/$(BINARY_NAME) \
		main.go

# Build for multiple platforms
build-all: deps
	@echo "Building for multiple platforms..."
	@mkdir -p $(BUILD_DIR)
	
	# Linux amd64
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build \
		-ldflags "-X main.version=$(VERSION) -X main.buildTime=$(BUILD_TIME)" \
		-o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 \
		main.go
	
	# Linux arm64
	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build \
		-ldflags "-X main.version=$(VERSION) -X main.buildTime=$(BUILD_TIME)" \
		-o $(BUILD_DIR)/$(BINARY_NAME)-linux-arm64 \
		main.go
	
	# macOS amd64
	GOOS=darwin GOARCH=amd64 CGO_ENABLED=0 go build \
		-ldflags "-X main.version=$(VERSION) -X main.buildTime=$(BUILD_TIME)" \
		-o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 \
		main.go
	
	# macOS arm64
	GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 go build \
		-ldflags "-X main.version=$(VERSION) -X main.buildTime=$(BUILD_TIME)" \
		-o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 \
		main.go
	
	# Windows amd64
	GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build \
		-ldflags "-X main.version=$(VERSION) -X main.buildTime=$(BUILD_TIME)" \
		-o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe \
		main.go

# Run the application
run: build
	@echo "Running $(BINARY_NAME)..."
	$(BUILD_DIR)/$(BINARY_NAME)

# Run in development mode with hot reload (requires air)
dev:
	@echo "Starting development server..."
	@if command -v air >/dev/null 2>&1; then \
		air; \
	else \
		echo "Air not installed. Install with: go install github.com/cosmtrek/air@latest"; \
		echo "Falling back to regular run..."; \
		make run; \
	fi

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	@rm -rf $(BUILD_DIR)
	@go clean

# Install the binary to GOPATH/bin
install: build
	@echo "Installing $(BINARY_NAME) to GOPATH/bin..."
	@cp $(BUILD_DIR)/$(BINARY_NAME) $(shell go env GOPATH)/bin/

# Show help
help:
	@echo "Available commands:"
	@echo "  deps      - Install dependencies"
	@echo "  fmt       - Format code"
	@echo "  vet       - Vet code"
	@echo "  test      - Run tests"
	@echo "  check     - Run fmt, vet, and test"
	@echo "  build     - Build binary for current platform"
	@echo "  build-all - Build binaries for all supported platforms"
	@echo "  run       - Build and run the application"
	@echo "  dev       - Run in development mode with hot reload"
	@echo "  clean     - Clean build artifacts"
	@echo "  install   - Install binary to GOPATH/bin"
	@echo "  help      - Show this help message"

# Development dependencies
dev-deps:
	@echo "Installing development dependencies..."
	go install github.com/cosmtrek/air@latest

# Docker build
docker-build:
	@echo "Building Docker image..."
	docker build -t scenario-to-desktop-api:$(VERSION) .

# Docker run
docker-run: docker-build
	@echo "Running Docker container..."
	docker run --rm -p 3202:3202 scenario-to-desktop-api:$(VERSION)

# Health check
health:
	@echo "Checking API health..."
	@curl -f http://localhost:3202/api/v1/health || echo "API not responding"

# API documentation
docs:
	@echo "API Endpoints:"
	@echo "GET    /api/v1/health                          - Health check"
	@echo "GET    /api/v1/status                          - System status"
	@echo "GET    /api/v1/templates                       - List templates"
	@echo "GET    /api/v1/templates/{type}                - Get template details"
	@echo "POST   /api/v1/desktop/generate                - Generate desktop app"
	@echo "GET    /api/v1/desktop/status/{build_id}       - Get build status"
	@echo "POST   /api/v1/desktop/build                   - Build desktop app"
	@echo "POST   /api/v1/desktop/test                    - Test desktop app"
	@echo "POST   /api/v1/desktop/package                 - Package desktop app"
	@echo "POST   /api/v1/desktop/webhook/build-complete  - Build completion webhook"

# Version info
version:
	@echo "Version: $(VERSION)"
	@echo "Build Time: $(BUILD_TIME)"