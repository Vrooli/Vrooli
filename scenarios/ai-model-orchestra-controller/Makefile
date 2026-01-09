# AI Model Orchestra Controller Scenario Makefile
# 
# This Makefile ensures scenarios are always run through the Vrooli lifecycle system.
# NEVER run scenarios directly (./api/binary). ALWAYS use these commands.
#
# Usage:
#   make       - Show help
#   make start - Start this scenario
#   make stop  - Stop this scenario
#   make test  - Run scenario tests
#   make logs  - Show scenario logs
#   make clean - Clean build artifacts

.PHONY: help start run stop test logs status clean build dev fmt fmt-go fmt-ui lint lint-go lint-ui check install-cli models health

# Default target - show help
.DEFAULT_GOAL := help

# Get scenario name from current directory
SCENARIO_NAME := $(notdir $(CURDIR))

# Colors for output
GREEN := \033[1;32m
YELLOW := \033[1;33m
BLUE := \033[1;34m
RED := \033[1;31m
RESET := \033[0m

help: ## Show this help message
	@echo "$(BLUE)📱 $(SCENARIO_NAME) Scenario Commands$(RESET)"
	@echo ""
	@echo "$(YELLOW)Usage:$(RESET)"
	@echo "  make <command>"
	@echo ""
	@echo "$(YELLOW)Commands:$(RESET)"
	@grep -E '^[a-zA-Z0-9_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  $(GREEN)%-10s$(RESET) %s\n", $$1, $$2}'
	@echo ""
	@echo "$(RED)⚠️  IMPORTANT:$(RESET) Never run ./api/$(SCENARIO_NAME)-api directly!"
	@echo "    Always use 'make start' or 'vrooli scenario start $(SCENARIO_NAME)'"

# ============================================================================
# LIFECYCLE COMMANDS - Use Vrooli lifecycle system
# ============================================================================

run: ## Start this scenario (uses Vrooli lifecycle)
	@echo "$(BLUE)🚀 Starting $(SCENARIO_NAME) scenario...$(RESET)"
	@vrooli scenario run $(SCENARIO_NAME)

start: run ## Start this scenario

stop: ## Stop this scenario
	@echo "$(YELLOW)⏹️  Stopping $(SCENARIO_NAME) scenario...$(RESET)"
	@vrooli scenario stop $(SCENARIO_NAME)

dev: ## Start in development mode
	@echo "$(BLUE)🔧 Starting $(SCENARIO_NAME) in development mode...$(RESET)"
	@vrooli scenario run $(SCENARIO_NAME) --dev

logs: ## Show recent logs for this scenario
	@echo "$(BLUE)📜 Logs for $(SCENARIO_NAME):$(RESET)"
	@vrooli scenario logs $(SCENARIO_NAME) --tail 50

status: ## Check if scenario is running
	@echo "$(BLUE)📊 Status of $(SCENARIO_NAME):$(RESET)"
	@vrooli scenario status $(SCENARIO_NAME)

# ============================================================================
# DEVELOPMENT COMMANDS - For testing and building
# ============================================================================

test: ## Run tests for this scenario
	@echo "$(BLUE)🧪 Testing $(SCENARIO_NAME) scenario...$(RESET)"
	@vrooli scenario test $(SCENARIO_NAME)

build: ## Build the API binary
	@echo "$(BLUE)🏗️  Building $(SCENARIO_NAME)...$(RESET)"
	@if [ -d api ]; then \
		cd api && go mod tidy && go build -o $(SCENARIO_NAME)-api .; \
	else \
		echo "$(YELLOW)No API directory found, skipping build$(RESET)"; \
	fi

clean: ## Clean build artifacts
	@echo "$(YELLOW)🧹 Cleaning build artifacts...$(RESET)"
	@rm -f api/$(SCENARIO_NAME)-api api/coverage.out api/coverage.html api/*.prof
	@if [ -d api ]; then cd api && go clean 2>/dev/null || true; fi
	@echo "$(GREEN)✓ Cleaned$(RESET)"

fmt: ## Format all code
	@echo "$(BLUE)🎨 Formatting code...$(RESET)"
	@$(MAKE) fmt-go
	@$(MAKE) fmt-ui

fmt-go: ## Format Go code
	@if [ -d api ] && find api -name "*.go" | head -1 | grep -q .; then \
		if command -v gofumpt >/dev/null 2>&1; then \
			cd api && gofumpt -w .; \
		elif command -v gofmt >/dev/null 2>&1; then \
			cd api && gofmt -w .; \
		else \
			echo "$(YELLOW)No Go formatter available$(RESET)"; \
		fi; \
	else \
		echo "$(YELLOW)No Go files to format$(RESET)"; \
	fi

fmt-ui: ## Format TypeScript/JavaScript code
	@if [ -d ui ] && (find ui -name "*.ts" -o -name "*.tsx" -o -name "*.js" -o -name "*.jsx") | head -1 | grep -q .; then \
		if command -v prettier >/dev/null 2>&1; then \
			cd ui && prettier --write "**/*.{ts,tsx,js,jsx,json,css,md}" 2>/dev/null || true; \
		else \
			echo "$(YELLOW)Prettier not found$(RESET)"; \
		fi; \
	else \
		echo "$(YELLOW)No UI files to format$(RESET)"; \
	fi

lint: ## Lint all code
	@echo "$(BLUE)🔍 Linting code...$(RESET)"
	@$(MAKE) lint-go
	@$(MAKE) lint-ui

lint-go: ## Lint Go code
	@if [ -d api ] && find api -name "*.go" | head -1 | grep -q .; then \
		if command -v golangci-lint >/dev/null 2>&1; then \
			cd api && golangci-lint run; \
		else \
			echo "$(YELLOW)golangci-lint not installed$(RESET)"; \
		fi; \
	else \
		echo "$(YELLOW)No Go files to lint$(RESET)"; \
	fi

lint-ui: ## Lint TypeScript/JavaScript code
	@if [ -d ui ] && (find ui -name "*.ts" -o -name "*.tsx" -o -name "*.js" -o -name "*.jsx") | head -1 | grep -q .; then \
		if [ -f ui/package.json ] && grep -q '"eslint"' ui/package.json; then \
			cd ui && npm run lint 2>/dev/null || true; \
		else \
			echo "$(YELLOW)ESLint not configured$(RESET)"; \
		fi; \
	else \
		echo "$(YELLOW)No UI files to lint$(RESET)"; \
	fi

check: ## Format, lint, and test code (pre-commit workflow)
	@echo "$(BLUE)✅ Running full code quality check...$(RESET)"
	@$(MAKE) fmt
	@$(MAKE) lint
	@$(MAKE) test

# ============================================================================
# ORCHESTRATION UTILITIES
# ============================================================================

install-cli: ## Install the AI orchestra CLI helper
	@echo "$(BLUE)⚙️  Installing ai-orchestra CLI...$(RESET)"
	@if [ -d cli ] && [ -f cli/install.sh ]; then \
		cd cli && ./install.sh; \
	else \
		echo "$(YELLOW)CLI installer not found$(RESET)"; \
	fi

models: ## List registered AI models
	@echo "$(BLUE)🤖 Checking AI models...$(RESET)"
	@ai-orchestra models --include-metrics || echo "$(YELLOW)Install CLI first: make install-cli$(RESET)"

health: ## Check scenario health via CLI
	@echo "$(BLUE)🏥 Checking system health...$(RESET)"
	@ai-orchestra health || echo "$(YELLOW)Install CLI first: make install-cli$(RESET)"

# Shortcuts
r: run
s: stop
d: dev
t: test
l: logs
c: clean
b: build
f: fmt
h: health
m: models
