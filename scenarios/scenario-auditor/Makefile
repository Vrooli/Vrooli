# Scenario Makefile
#
# This Makefile ensures scenarios are always run through the Vrooli lifecycle system.
# NEVER run scenarios directly (./api/scenario-auditor-api). ALWAYS use these commands.
#
# Usage:
#   make       - Show help
#   make start - Start this scenario
#   make stop  - Stop this scenario
#   make test  - Run scenario tests
#   make logs  - Show scenario logs
#   make clean - Clean build artifacts

.PHONY: help start stop test logs status clean build dev install setup health run scan-current scan-all rules restart rebuild make-executable fmt fmt-go fmt-ui lint lint-go lint-ui install-cli test-unit test-integration test-structure test-deps test-business test-performance

.DEFAULT_GOAL := help

SCENARIO_NAME := $(notdir $(CURDIR))

GREEN := \033[1;32m
YELLOW := \033[1;33m
BLUE := \033[1;34m
RED := \033[1;31m
RESET := \033[0m

help: ## Show this help message
	@echo "$(BLUE)🔍 $(SCENARIO_NAME) Scenario Commands$(RESET)"
	@echo ""
	@echo "$(YELLOW)Usage:$(RESET)"
	@echo "  make <command>"
	@echo ""
	@echo "$(YELLOW)Commands:$(RESET)"
	@grep -E '^[a-zA-Z0-9_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  $(GREEN)%-16s$(RESET) %s\n", $$1, $$2}'
	@echo ""
	@echo "$(RED)⚠️  IMPORTANT:$(RESET) Never run ./api/$(SCENARIO_NAME)-api directly!"
	@echo "    Always use 'make start' or 'vrooli scenario start $(SCENARIO_NAME)'"
	@echo ""

# Lifecycle commands
start: ## Start this scenario
	@echo "$(BLUE)Starting $(SCENARIO_NAME)...$(RESET)"
	@vrooli scenario start $(SCENARIO_NAME)

stop: ## Stop this scenario
	@echo "$(BLUE)Stopping $(SCENARIO_NAME)...$(RESET)"
	@vrooli scenario stop $(SCENARIO_NAME) || true
	@echo "$(GREEN)✅ Stopped$(RESET)"

status: ## Show scenario status
	@vrooli scenario status $(SCENARIO_NAME)

logs: ## Show scenario logs
	@vrooli scenario logs $(SCENARIO_NAME)

# Installation
install: ## Install all dependencies
	@echo "📦 Installing dependencies..."
	@cd api && go mod download
	@if [ -f ui/package.json ] && command -v node >/dev/null 2>&1; then \
		cd ui && npm install --prefer-offline --no-audit --no-fund --loglevel error; \
	fi
	@echo "$(GREEN)✅ Dependencies installed$(RESET)"

# Build
build: ## Build API and CLI binaries
	@echo "🔨 Building binaries..."
	@cd api && go build -o scenario-auditor-api .
	@chmod +x cli/scenario-auditor
	@echo "$(GREEN)✅ Build complete$(RESET)"

# Install CLI globally
install-cli: ## Install CLI command globally
	@echo "📦 Installing CLI globally..."
	@cd cli && ./install.sh

# Setup (install + build)
setup: install build install-cli ## Full setup (install + build)
	@echo "$(GREEN)✅ Setup complete$(RESET)"

# Testing
test: ## Run comprehensive test suite
	@echo "🧪 Running full test suite..."
	@test-genie execute $(SCENARIO_NAME) --preset comprehensive

test-unit: ## Run unit tests only
	@echo "🧪 Running unit tests..."
	@test-genie execute $(SCENARIO_NAME) --phases unit

test-integration: ## Run integration tests only
	@echo "🧪 Running integration tests..."
	@test-genie execute $(SCENARIO_NAME) --phases integration

test-structure: ## Run structure validation
	@echo "🧪 Running structure tests..."
	@test-genie execute $(SCENARIO_NAME) --phases structure

test-deps: ## Check dependencies
	@echo "🧪 Checking dependencies..."
	@test-genie execute $(SCENARIO_NAME) --phases dependencies

test-business: ## Run business logic tests
	@echo "🧪 Running business logic tests..."
	@test-genie execute $(SCENARIO_NAME) --phases business

test-performance: ## Run performance tests
	@echo "🧪 Running performance tests..."
	@test-genie execute $(SCENARIO_NAME) --phases performance

# Development
dev: start ## Start development mode (alias for start)

run: ## Run scenario locally (for testing only)
	@echo "$(YELLOW)⚠️  Using local run mode (for testing only)$(RESET)"
	@echo "$(YELLOW)⚠️  For production, use 'make start' instead$(RESET)"
	@if [ ! -f api/scenario-auditor-api ]; then make build; fi
	@export API_PORT=$${API_PORT:-15001} && \
	export UI_PORT=$${UI_PORT:-35001} && \
	echo "Starting API on port $$API_PORT..." && \
	cd api && VROOLI_LIFECYCLE_MANAGED=true ./scenario-auditor-api & \
	echo "$$!" > ../api.pid && \
	sleep 3 && \
	if [ -f ../ui/package.json ] && command -v node >/dev/null 2>&1; then \
		echo "Starting UI on port $$UI_PORT..." && \
		cd ../ui && npm run dev & \
		echo "$$!" > ../ui.pid; \
	fi && \
	echo "$(GREEN)✅ Scenario Auditor is running:$(RESET)" && \
	echo "  Dashboard: http://localhost:$$UI_PORT" && \
	echo "  API: http://localhost:$$API_PORT/api/v1" && \
	echo "  Health: http://localhost:$$API_PORT/api/v1/health"

health: ## Check system health
	@echo "🏥 Checking health..."
	@export API_PORT=$${API_PORT:-15001} && \
	if curl -s "http://localhost:$$API_PORT/api/v1/health" >/dev/null 2>&1; then \
		echo "$(GREEN)✅ API is healthy$(RESET)"; \
	else \
		echo "$(RED)❌ API is not responding$(RESET)"; \
		exit 1; \
	fi

# Clean
clean: ## Clean build artifacts
	@echo "🧹 Cleaning build artifacts..."
	@rm -f api/scenario-auditor-api
	@rm -f cli/scenario-auditor
	@rm -f api.pid ui.pid
	@rm -rf ui/dist
	@rm -rf ui/node_modules/.vite
	@find test/artifacts -name "*.log" -type f -delete 2>/dev/null || true
	@echo "$(GREEN)✅ Clean complete$(RESET)"

# Formatting and Linting
fmt: fmt-go fmt-ui ## Format all code

fmt-go: ## Format Go code
	@if command -v gofumpt >/dev/null 2>&1; then \
		echo "$(BLUE)Formatting Go code with gofumpt...$(RESET)"; \
		cd api && gofumpt -w .; \
		cd cli && gofumpt -w .; \
		echo "$(GREEN)✅ Go code formatted$(RESET)"; \
	else \
		echo "$(YELLOW)⚠️  gofumpt not installed, using gofmt...$(RESET)"; \
		cd api && go fmt ./...; \
		cd cli && go fmt ./...; \
	fi

fmt-ui: ## Format UI code
	@if [ -f ui/package.json ] && command -v node >/dev/null 2>&1; then \
		echo "$(BLUE)Formatting UI code...$(RESET)"; \
		cd ui && npm run format 2>/dev/null || echo "$(YELLOW)⚠️  No format script in package.json$(RESET)"; \
	else \
		echo "$(YELLOW)⚠️  No UI or Node.js not installed$(RESET)"; \
	fi

lint: lint-go lint-ui ## Lint all code

lint-go: ## Lint Go code
	@if command -v golangci-lint >/dev/null 2>&1; then \
		echo "$(BLUE)Linting Go code...$(RESET)"; \
		cd api && golangci-lint run; \
		cd cli && golangci-lint run; \
		echo "$(GREEN)✅ Go code linted$(RESET)"; \
	else \
		echo "$(YELLOW)⚠️  golangci-lint not installed$(RESET)"; \
		echo "$(YELLOW)    Install: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest$(RESET)"; \
	fi

lint-ui: ## Lint UI code
	@if [ -f ui/package.json ] && command -v node >/dev/null 2>&1; then \
		echo "$(BLUE)Linting UI code...$(RESET)"; \
		cd ui && npm run lint 2>/dev/null || echo "$(YELLOW)⚠️  No lint script in package.json$(RESET)"; \
	else \
		echo "$(YELLOW)⚠️  No UI or Node.js not installed$(RESET)"; \
	fi

# Utilities
scan-current: ## Scan current scenario for violations
	@echo "🔍 Scanning current scenario..."
	@scenario-auditor scan $(SCENARIO_NAME) || echo "$(RED)❌ CLI not installed. Run 'make install-cli'$(RESET)"

scan-all: ## Scan all scenarios
	@echo "🔍 Scanning all scenarios..."
	@scenario-auditor scan || echo "$(RED)❌ CLI not installed. Run 'make install-cli'$(RESET)"

rules: ## List available rules
	@echo "📋 Listing available rules..."
	@scenario-auditor rules || echo "$(RED)❌ CLI not installed. Run 'make install-cli'$(RESET)"

restart: stop start ## Restart scenario

rebuild: clean build ## Clean and rebuild

make-executable: ## Make test scripts executable
	@chmod +x cli/install.sh
