# Scenario Makefile
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

.PHONY: help start run stop test logs status clean build dev fmt fmt-go fmt-ui lint lint-go lint-ui check

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
	@echo "$(RED)⚠️  IMPORTANT:$(RESET) Never run ./api/$(SCENARIO_NAME) directly!"
	@echo "    Always use 'make start' or 'vrooli scenario start $(SCENARIO_NAME)'"

start: ## Start this scenario (uses Vrooli lifecycle)
	@echo "$(BLUE)🚀 Starting $(SCENARIO_NAME) scenario...$(RESET)"
	@vrooli scenario run $(SCENARIO_NAME)

run: start ## Alias for 'make start'

stop: ## Stop this scenario
	@echo "$(YELLOW)⏹️  Stopping $(SCENARIO_NAME) scenario...$(RESET)"
	@vrooli scenario stop $(SCENARIO_NAME)

dev: ## Start in development mode
	@echo "$(BLUE)🔧 Starting $(SCENARIO_NAME) in development mode...$(RESET)"
	@vrooli scenario run $(SCENARIO_NAME) --dev

test: ## Run tests for this scenario
	@echo "$(BLUE)🧪 Testing $(SCENARIO_NAME) scenario...$(RESET)"
	@vrooli scenario test $(SCENARIO_NAME)

logs: ## Show recent logs for this scenario
	@echo "$(BLUE)📜 Logs for $(SCENARIO_NAME):$(RESET)"
	@vrooli scenario logs $(SCENARIO_NAME) --tail 50

status: ## Check if scenario is running
	@echo "$(BLUE)📊 Status of $(SCENARIO_NAME):$(RESET)"
	@vrooli scenario status $(SCENARIO_NAME)

clean: ## Clean build artifacts
	@echo "$(YELLOW)🧹 Cleaning $(SCENARIO_NAME) build artifacts...$(RESET)"
	@rm -rf build/ dist/ *.log api/$(SCENARIO_NAME) api/$(SCENARIO_NAME)-api
	@if [ -d api ]; then cd api && go clean 2>/dev/null || true; fi
	@echo "$(GREEN)✓ Cleaned$(RESET)"

build: ## Build the scenario (if applicable)
	@echo "$(BLUE)🏗️  Building $(SCENARIO_NAME)...$(RESET)"
	@if [ -f api/go.mod ]; then \
		cd api && go build -o $(SCENARIO_NAME)-api ./cmd/server/main.go 2>/dev/null || go build -o $(SCENARIO_NAME)-api main.go 2>/dev/null || echo "Could not build Go API"; \
	else \
		echo "$(YELLOW)No build needed for this scenario$(RESET)"; \
	fi

# ============================================================================
# Code Quality Targets
# ============================================================================

fmt: ## Format all code
	@echo "$(BLUE)🎨 Formatting code...$(RESET)"
	@$(MAKE) fmt-go

fmt-go: ## Format Go code
	@if [ -d api ] && find api -name "*.go" | head -1 | grep -q .; then \
		if command -v gofumpt >/dev/null 2>&1; then \
			cd api && gofumpt -w .; \
		elif command -v gofmt >/dev/null 2>&1; then \
			cd api && gofmt -w .; \
		fi; \
	fi

fmt-ui: ## Format UI code (no-op: scenario has no UI)
	@echo "$(YELLOW)ℹ️  No UI to format$(RESET)"

lint: ## Lint all code
	@echo "$(BLUE)🔍 Linting code...$(RESET)"
	@$(MAKE) lint-go

lint-go: ## Lint Go code
	@if [ -d api ] && find api -name "*.go" | head -1 | grep -q .; then \
		if command -v golangci-lint >/dev/null 2>&1; then \
			cd api && golangci-lint run; \
		fi; \
	fi

lint-ui: ## Lint UI code (no-op: scenario has no UI)
	@echo "$(YELLOW)ℹ️  No UI to lint$(RESET)"

check: ## Format, lint, and test code (pre-commit workflow)
	@echo "$(BLUE)✅ Running full code quality check...$(RESET)"
	@$(MAKE) fmt
	@$(MAKE) lint
	@$(MAKE) test
	@echo "$(GREEN)✓ All checks passed!$(RESET)"

# ============================================================================
# Development shortcuts
# ============================================================================

r: run
s: stop
d: dev
t: test
l: logs
c: clean
b: build
