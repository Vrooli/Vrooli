# Scenario Makefile
#
# This Makefile ensures scenarios are always run through the Vrooli lifecycle system.
# NEVER run scenarios directly (./api/scenario-to-desktop-api). ALWAYS use these commands.
#
# Usage:
#   make       - Show help
#   make start - Start this scenario
#   make stop  - Stop this scenario
#   make test  - Run scenario tests
#   make logs  - Show scenario logs
#   make clean - Clean build artifacts

.PHONY: help start run stop dev test logs logs-follow status clean build build-all build-linux build-macos build-windows deps fmt fmt-go fmt-ui vet lint lint-go lint-ui check docker-build docker-run health docs version validate runtime-validate lint-templates typecheck-templates test-templates check-templates templates-deps

# Default target - show help
.DEFAULT_GOAL := help

# Get scenario name from directory
SCENARIO_NAME := $(notdir $(CURDIR))

# Color palette for output (standard set only)
GREEN := \033[1;32m
YELLOW := \033[1;33m
BLUE := \033[1;34m
RED := \033[1;31m
RESET := \033[0m

help: ## Show this help message
	@echo "$(BLUE)Scenario-to-Desktop Scenario Commands$(RESET)"
	@echo ""
	@echo "$(YELLOW)Usage:$(RESET)"
	@echo "  make <command>"
	@echo ""
	@echo "$(YELLOW)Commands:$(RESET)"
	@grep -E '^[a-zA-Z0-9_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  $(GREEN)%-18s$(RESET) %s\n", $$1, $$2}'
	@echo ""
	@echo "$(RED)⚠️  Always use 'make start' or 'vrooli scenario start $(SCENARIO_NAME)' to run this scenario.$(RESET)"
	@echo "$(RED)   Never run ./api/scenario-to-desktop-api directly!$(RESET)"

# Build configuration
BINARY_NAME := scenario-to-desktop-api
BUILD_DIR := api/bin
GO_FILES := $(shell find api -type f -name '*.go' -not -path "./vendor/*" 2>/dev/null)
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")

# ============================================================================
# LIFECYCLE COMMANDS - Use Vrooli lifecycle system
# ============================================================================

start: ## Start scenario-to-desktop (via Vrooli lifecycle)
	@echo "$(BLUE)🚀 Starting $(SCENARIO_NAME) scenario...$(RESET)"
	@vrooli scenario start $(SCENARIO_NAME)

run: start ## Alias for 'make start'

stop: ## Stop scenario-to-desktop
	@echo "$(YELLOW)⏹️  Stopping $(SCENARIO_NAME) scenario...$(RESET)"
	@vrooli scenario stop $(SCENARIO_NAME)

dev: ## Start in development mode with hot reload
	@echo "$(BLUE)🔧 Starting $(SCENARIO_NAME) in development mode...$(RESET)"
	@vrooli scenario start $(SCENARIO_NAME) --dev

logs: ## Show recent logs
	@echo "$(BLUE)📜 Logs for $(SCENARIO_NAME):$(RESET)"
	@vrooli scenario logs $(SCENARIO_NAME) --tail 50

logs-follow: ## Follow logs in real-time
	@echo "$(BLUE)📜 Following Scenario-to-Desktop logs...$(RESET)"
	@vrooli scenario logs $(SCENARIO_NAME) --follow

status: ## Check if scenario is running
	@echo "$(BLUE)📊 Status of $(SCENARIO_NAME):$(RESET)"
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
		.
	@echo "$(GREEN)✓ Built $(BUILD_DIR)/$(BINARY_NAME)$(RESET)"

build-all: deps ## Build for all platforms (Linux, macOS, Windows)
	@echo "$(BLUE)🏗️  Building for all platforms...$(RESET)"
	@$(MAKE) build-linux
	@$(MAKE) build-macos
	@$(MAKE) build-windows
	@echo "$(GREEN)✓ All platforms built$(RESET)"

build-linux: ## Build for Linux (amd64 & arm64)
	@echo "$(BLUE)Building for Linux...$(RESET)"
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
	@echo "$(BLUE)Building for macOS...$(RESET)"
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
	@echo "$(BLUE)Building for Windows...$(RESET)"
	@mkdir -p $(BUILD_DIR)
	@cd api && GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build \
		-ldflags "-X main.version=$(VERSION) -X main.buildTime=$(BUILD_TIME)" \
		-o bin/$(BINARY_NAME)-windows-amd64.exe \
		main.go
	@echo "  $(GREEN)✓$(RESET) Windows build complete"

# ============================================================================
# DEVELOPMENT COMMANDS
# ============================================================================

test: ## Run tests for this scenario
	@echo "$(BLUE)🧪 Testing $(SCENARIO_NAME) scenario...$(RESET)"
	@vrooli scenario test $(SCENARIO_NAME)

test-unit: ## Run unit tests only (fast, no external dependencies)
	@echo "$(BLUE)🧪 Running unit tests...$(RESET)"
	@cd api && go test -v -race ./smoketest/...
	@echo "$(GREEN)✓ Unit tests passed$(RESET)"

test-integration: ## Run integration tests (requires test fixture build)
	@echo "$(BLUE)🧪 Running integration tests...$(RESET)"
	@cd api && go test -tags=integration -timeout=5m -v ./smoketest/...
	@echo "$(GREEN)✓ Integration tests passed$(RESET)"

test-all: test-unit test-integration ## Run all tests (unit + integration)
	@echo "$(GREEN)✓ All tests passed$(RESET)"

runtime-validate: ## Run supervisor dry-run validation loop (fixture bundle)
	@echo "$(BLUE)🧪 Running runtime dry-run validation...$(RESET)"
	@cd runtime && go run ./cmd/runtime --manifest testdata/fixture-bundle/bundle.json --app-data /tmp/vrooli-runtime-fixture --dry-run

deps: ## Install/update dependencies
	@echo "$(BLUE)📦 Installing dependencies...$(RESET)"
	@cd api && go mod download
	@cd api && go mod tidy
	@echo "$(GREEN)✓ Dependencies installed$(RESET)"

fmt: fmt-go fmt-ui ## Format all code

fmt-go: ## Format Go code
	@echo "$(BLUE)🎨 Formatting Go code...$(RESET)"
	@if [ -d api ] && ls api/*.go &>/dev/null; then \
		if command -v gofumpt >/dev/null 2>&1; then \
			gofumpt -w api/; \
		else \
			cd api && go fmt ./...; \
		fi; \
	else \
		echo "$(YELLOW)ℹ No Go sources found, skipping$(RESET)"; \
	fi
	@echo "$(GREEN)✓ Go format complete$(RESET)"

fmt-ui: ## Format UI code (no-op if no UI formatting tool)
	@echo "$(BLUE)🎨 Formatting UI code...$(RESET)"
	@echo "$(YELLOW)ℹ UI formatting not configured$(RESET)"

lint: lint-go lint-ui ## Lint all code

lint-go: ## Lint Go code
	@echo "$(BLUE)🔍 Linting Go code...$(RESET)"
	@if [ -d api ] && ls api/*.go &>/dev/null; then \
		if command -v golangci-lint >/dev/null 2>&1; then \
			cd api && golangci-lint run ./...; \
		else \
			echo "$(YELLOW)ℹ golangci-lint not installed, skipping$(RESET)"; \
		fi; \
	else \
		echo "$(YELLOW)ℹ No Go sources found, skipping$(RESET)"; \
	fi
	@echo "$(GREEN)✓ Go lint complete$(RESET)"

lint-ui: ## Lint UI code (no-op if no UI linter)
	@echo "$(BLUE)🔍 Linting UI code...$(RESET)"
	@echo "$(YELLOW)ℹ UI linting not configured$(RESET)"

vet: ## Vet code
	@echo "$(BLUE)🔍 Vetting code...$(RESET)"
	@cd api && go vet ./...
	@echo "$(GREEN)✓ Vet complete$(RESET)"

check: fmt lint vet test ## Run fmt, lint, vet, and test
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
	@echo "$(YELLOW)Health & Status:$(RESET)"
	@echo "  GET  /api/v1/health                         - Health check"
	@echo "  GET  /api/v1/status                         - System status"
	@echo ""
	@echo "$(YELLOW)Templates:$(RESET)"
	@echo "  GET  /api/v1/templates                      - List templates"
	@echo "  GET  /api/v1/templates/{type}               - Get template details"
	@echo ""
	@echo "$(YELLOW)Desktop Generation:$(RESET)"
	@echo "  POST /api/v1/desktop/generate               - Generate desktop app"
	@echo "  GET  /api/v1/desktop/status/{build_id}      - Get build status"
	@echo "  POST /api/v1/desktop/build                  - Build desktop app"
	@echo "  POST /api/v1/desktop/test                   - Test desktop app"
	@echo "  POST /api/v1/desktop/package                - Package desktop app"
	@echo ""
	@echo "$(YELLOW)Webhooks:$(RESET)"
	@echo "  POST /api/v1/desktop/webhook/build-complete - Build completion webhook"

version: ## Show version info
	@echo "$(YELLOW)Version Information:$(RESET)"
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
# TEMPLATE LINTING & TYPE-CHECKING
# ============================================================================

TEMPLATES_DIR := templates/vanilla

lint-templates: ## Lint vanilla template TypeScript files
	@echo "$(BLUE)🔍 Linting vanilla templates...$(RESET)"
	@cd $(TEMPLATES_DIR) && npm run lint

typecheck-templates: ## Type-check vanilla template TypeScript files
	@echo "$(BLUE)🔍 Type-checking vanilla templates...$(RESET)"
	@cd $(TEMPLATES_DIR) && npm run typecheck

test-templates: ## Run vanilla template tests
	@echo "$(BLUE)🧪 Testing vanilla templates...$(RESET)"
	@cd $(TEMPLATES_DIR) && npm run test

check-templates: typecheck-templates lint-templates test-templates ## Full template validation
	@echo "$(GREEN)✓ All template checks passed$(RESET)"

templates-deps: ## Install template development dependencies
	@echo "$(BLUE)📦 Installing template dependencies...$(RESET)"
	@cd $(TEMPLATES_DIR) && npm install
	@echo "$(GREEN)✓ Template dependencies installed$(RESET)"

# ============================================================================
# SHORTCUTS
# ============================================================================

r: run
s: stop
d: dev
t: test
tu: test-unit
ti: test-integration
ta: test-all
l: logs
c: clean
b: build
ba: build-all
lt: lint-templates
tt: typecheck-templates
ct: check-templates
