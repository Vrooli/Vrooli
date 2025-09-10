# Scenario-to-Desktop Converter
# 
# This Makefile ensures the scenario is always run through the Vrooli lifecycle system.
# NEVER run scenarios directly (./api/scenario-to-desktop). ALWAYS use these commands.
#
# Usage:
#   make           - Show help
#   make run       - Start scenario-to-desktop converter
#   make stop      - Stop converter
#   make dev       - Start in development mode with hot reload
#   make test      - Run all tests
#   make build     - Build for current platform
#   make build-all - Build for all platforms
#   make logs      - Show scenario logs
#   make clean     - Clean build artifacts

.PHONY: help run stop dev test logs logs-follow status clean build build-all build-linux build-macos build-windows deps fmt vet check docker-build docker-run health docs version validate

# Default target - show help
.DEFAULT_GOAL := help

# Get scenario name
SCENARIO_NAME := scenario-to-desktop
BINARY_NAME := scenario-to-desktop-api
BUILD_DIR := api/bin
GO_FILES := $(shell find api -type f -name '*.go' -not -path "./vendor/*" 2>/dev/null)
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")

# Colors for output
GREEN := \033[1;32m
YELLOW := \033[1;33m
BLUE := \033[1;34m
RED := \033[1;31m
CYAN := \033[1;36m
RESET := \033[0m

help: ## Show this help message
	@echo "$(BLUE)🖥️  Scenario-to-Desktop Converter Commands$(RESET)"
	@echo ""
	@echo "$(YELLOW)Usage:$(RESET)"
	@echo "  make <command>"
	@echo ""
	@echo "$(YELLOW)Lifecycle Commands:$(RESET)"
	@grep -E '^(run|stop|dev|logs|status):.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  $(GREEN)%-15s$(RESET) %s\n", $$1, $$2}'
	@echo ""
	@echo "$(YELLOW)Build Commands:$(RESET)"
	@grep -E '^(build|build-all|test|clean):.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  $(GREEN)%-15s$(RESET) %s\n", $$1, $$2}'
	@echo ""
	@echo "$(YELLOW)Additional Commands:$(RESET)"
	@grep -E '^[a-zA-Z0-9_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		grep -v -E '^(run|stop|dev|logs|status|build|build-all|test|clean|help):' | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  $(GREEN)%-15s$(RESET) %s\n", $$1, $$2}'
	@echo ""
	@echo "$(RED)⚠️  IMPORTANT:$(RESET) Never run ./api/$(BINARY_NAME) directly!"
	@echo "    Always use 'make run' or 'vrooli scenario run $(SCENARIO_NAME)'"

# ============================================================================
# LIFECYCLE COMMANDS - Use Vrooli lifecycle system
# ============================================================================

run: ## Start scenario-to-desktop (via Vrooli lifecycle)
	@echo "$(BLUE)🚀 Starting Scenario-to-Desktop Converter...$(RESET)"
	@echo "$(CYAN)Converting scenarios to desktop applications$(RESET)"
	@vrooli scenario run $(SCENARIO_NAME)

stop: ## Stop scenario-to-desktop
	@echo "$(YELLOW)⏹️  Stopping Scenario-to-Desktop Converter...$(RESET)"
	@vrooli scenario stop $(SCENARIO_NAME)

dev: ## Start in development mode with hot reload
	@echo "$(BLUE)🔧 Starting in development mode...$(RESET)"
	@vrooli scenario run $(SCENARIO_NAME) --dev

logs: ## Show recent logs
	@echo "$(BLUE)📜 Recent Scenario-to-Desktop logs:$(RESET)"
	@vrooli scenario logs $(SCENARIO_NAME) --tail 50

logs-follow: ## Follow logs in real-time
	@echo "$(BLUE)📜 Following Scenario-to-Desktop logs...$(RESET)"
	@vrooli scenario logs $(SCENARIO_NAME) --follow

status: ## Check if scenario is running
	@echo "$(BLUE)📊 Scenario-to-Desktop Status:$(RESET)"
	@vrooli scenario status $(SCENARIO_NAME)

# ============================================================================
# BUILD COMMANDS - Cross-platform builds
# ============================================================================

build: deps ## Build for current platform
	@echo "$(BLUE)🏗️  Building $(BINARY_NAME)...$(RESET)"
	@mkdir -p $(BUILD_DIR)
	@cd api && CGO_ENABLED=0 go build \
		-ldflags "-X main.version=$(VERSION) -X main.buildTime=$(BUILD_TIME)" \
		-o bin/$(BINARY_NAME) \
		main.go
	@echo "$(GREEN)✓ Built $(BUILD_DIR)/$(BINARY_NAME)$(RESET)"

build-all: deps ## Build for all platforms (Linux, macOS, Windows)
	@echo "$(BLUE)🏗️  Building for all platforms...$(RESET)"
	@$(MAKE) build-linux
	@$(MAKE) build-macos
	@$(MAKE) build-windows
	@echo "$(GREEN)✓ All platforms built$(RESET)"

build-linux: ## Build for Linux (amd64 & arm64)
	@echo "$(CYAN)Building for Linux...$(RESET)"
	@mkdir -p $(BUILD_DIR)
	@cd api && GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build \
		-ldflags "-X main.version=$(VERSION) -X main.buildTime=$(BUILD_TIME)" \
		-o bin/$(BINARY_NAME)-linux-amd64 \
		main.go
	@cd api && GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build \
		-ldflags "-X main.version=$(VERSION) -X main.buildTime=$(BUILD_TIME)" \
		-o bin/$(BINARY_NAME)-linux-arm64 \
		main.go
	@echo "  $(GREEN)✓$(RESET) Linux builds complete"

build-macos: ## Build for macOS (Intel & Apple Silicon)
	@echo "$(CYAN)Building for macOS...$(RESET)"
	@mkdir -p $(BUILD_DIR)
	@cd api && GOOS=darwin GOARCH=amd64 CGO_ENABLED=0 go build \
		-ldflags "-X main.version=$(VERSION) -X main.buildTime=$(BUILD_TIME)" \
		-o bin/$(BINARY_NAME)-darwin-amd64 \
		main.go
	@cd api && GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 go build \
		-ldflags "-X main.version=$(VERSION) -X main.buildTime=$(BUILD_TIME)" \
		-o bin/$(BINARY_NAME)-darwin-arm64 \
		main.go
	@echo "  $(GREEN)✓$(RESET) macOS builds complete"

build-windows: ## Build for Windows (amd64)
	@echo "$(CYAN)Building for Windows...$(RESET)"
	@mkdir -p $(BUILD_DIR)
	@cd api && GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build \
		-ldflags "-X main.version=$(VERSION) -X main.buildTime=$(BUILD_TIME)" \
		-o bin/$(BINARY_NAME)-windows-amd64.exe \
		main.go
	@echo "  $(GREEN)✓$(RESET) Windows build complete"

# ============================================================================
# DEVELOPMENT COMMANDS
# ============================================================================

test: ## Run all tests
	@echo "$(BLUE)🧪 Running tests...$(RESET)"
	@cd api && go test -v ./...

deps: ## Install/update dependencies
	@echo "$(BLUE)📦 Installing dependencies...$(RESET)"
	@cd api && go mod download
	@cd api && go mod tidy
	@echo "$(GREEN)✓ Dependencies installed$(RESET)"

fmt: ## Format code
	@echo "$(BLUE)🎨 Formatting code...$(RESET)"
	@cd api && go fmt ./...
	@echo "$(GREEN)✓ Format complete$(RESET)"

vet: ## Vet code
	@echo "$(BLUE)🔍 Vetting code...$(RESET)"
	@cd api && go vet ./...
	@echo "$(GREEN)✓ Vet complete$(RESET)"

check: fmt vet test ## Run fmt, vet, and test
	@echo "$(GREEN)✓ All checks passed!$(RESET)"

clean: ## Clean build artifacts
	@echo "$(YELLOW)🧹 Cleaning build artifacts...$(RESET)"
	@rm -rf $(BUILD_DIR)
	@cd api && go clean
	@rm -f api/*.log
	@echo "$(GREEN)✓ Cleaned$(RESET)"

# ============================================================================
# DOCKER COMMANDS
# ============================================================================

docker-build: ## Build Docker image
	@echo "$(BLUE)🐳 Building Docker image...$(RESET)"
	@cd api && docker build -t $(SCENARIO_NAME):$(VERSION) .
	@echo "$(GREEN)✓ Docker image built: $(SCENARIO_NAME):$(VERSION)$(RESET)"

docker-run: docker-build ## Run in Docker container
	@echo "$(BLUE)🐳 Running Docker container...$(RESET)"
	@docker run --rm -p 3202:3202 $(SCENARIO_NAME):$(VERSION)

# ============================================================================
# UTILITY COMMANDS
# ============================================================================

health: ## Check API health
	@echo "$(BLUE)💓 Checking API health...$(RESET)"
	@curl -f http://localhost:3202/api/v1/health 2>/dev/null && echo "$(GREEN)✓ API is healthy$(RESET)" || echo "$(RED)✗ API not responding$(RESET)"

docs: ## Show API documentation
	@echo "$(BLUE)📚 API Endpoints:$(RESET)"
	@echo ""
	@echo "$(CYAN)Health & Status:$(RESET)"
	@echo "  GET  /api/v1/health                         - Health check"
	@echo "  GET  /api/v1/status                         - System status"
	@echo ""
	@echo "$(CYAN)Templates:$(RESET)"
	@echo "  GET  /api/v1/templates                      - List templates"
	@echo "  GET  /api/v1/templates/{type}               - Get template details"
	@echo ""
	@echo "$(CYAN)Desktop Generation:$(RESET)"
	@echo "  POST /api/v1/desktop/generate               - Generate desktop app"
	@echo "  GET  /api/v1/desktop/status/{build_id}      - Get build status"
	@echo "  POST /api/v1/desktop/build                  - Build desktop app"
	@echo "  POST /api/v1/desktop/test                   - Test desktop app"
	@echo "  POST /api/v1/desktop/package                - Package desktop app"
	@echo ""
	@echo "$(CYAN)Webhooks:$(RESET)"
	@echo "  POST /api/v1/desktop/webhook/build-complete - Build completion webhook"

version: ## Show version info
	@echo "$(CYAN)Version Information:$(RESET)"
	@echo "  Version:    $(VERSION)"
	@echo "  Build Time: $(BUILD_TIME)"
	@echo "  Go Version: $$(go version | cut -d' ' -f3)"

validate: ## Validate scenario structure
	@echo "$(BLUE)✅ Validating $(SCENARIO_NAME) structure...$(RESET)"
	@vrooli scenario validate $(SCENARIO_NAME)

install: build ## Install binary to GOPATH/bin
	@echo "$(BLUE)📦 Installing $(BINARY_NAME) to GOPATH/bin...$(RESET)"
	@cp $(BUILD_DIR)/$(BINARY_NAME) $$(go env GOPATH)/bin/
	@echo "$(GREEN)✓ Installed to $$(go env GOPATH)/bin/$(BINARY_NAME)$(RESET)"

dev-deps: ## Install development dependencies
	@echo "$(BLUE)📦 Installing development dependencies...$(RESET)"
	@go install github.com/cosmtrek/air@latest
	@go install github.com/tsenart/vegeta@latest
	@echo "$(GREEN)✓ Development dependencies installed$(RESET)"

# ============================================================================
# SHORTCUTS
# ============================================================================

r: run
s: stop
d: dev
t: test
l: logs
c: clean
b: build
ba: build-all