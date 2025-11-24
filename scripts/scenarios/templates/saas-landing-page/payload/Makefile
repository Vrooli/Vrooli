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

.PHONY: help start stop status test logs clean build dev fmt fmt-go fmt-ui lint lint-go lint-ui check

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
	@echo "$(BLUE)Scenario Commands$(RESET)"
	@echo ""
	@echo "$(YELLOW)Usage:$(RESET)"
	@echo "  make <command>"
	@echo ""
	@echo "$(YELLOW)Commands:$(RESET)"
	@grep -E '^[a-zA-Z0-9_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  $(GREEN)%-15s$(RESET) %s\n", $$1, $$2}'
	@echo ""
	@echo "$(RED)⚠️  IMPORTANT:$(RESET)"
	@echo "    Never run ./api/landing-manager-api or ./ui/server directly!"
	@echo "    Always use 'make start' or 'vrooli scenario start $(SCENARIO_NAME)'"

start: ## Start this scenario (uses Vrooli lifecycle)
	@echo "$(BLUE)🚀 Starting $(SCENARIO_NAME) scenario...$(RESET)"
	@vrooli scenario start $(SCENARIO_NAME)

stop: ## Stop this scenario
	@echo "$(YELLOW)⏹️  Stopping $(SCENARIO_NAME) scenario...$(RESET)"
	@vrooli scenario stop $(SCENARIO_NAME)

status: ## Show scenario status
	@echo "$(BLUE)📊 Status of $(SCENARIO_NAME):$(RESET)"
	@vrooli scenario status $(SCENARIO_NAME)

test: ## Run all tests (API, UI, CLI)
	@echo "$(BLUE)🧪 Testing $(SCENARIO_NAME) scenario...$(RESET)"
	@vrooli scenario test $(SCENARIO_NAME)

logs: ## Show recent logs for this scenario
	@echo "$(BLUE)📜 Logs for $(SCENARIO_NAME):$(RESET)"
	@vrooli scenario logs $(SCENARIO_NAME) --tail 50

clean: ## Clean build artifacts
	@echo "$(YELLOW)🧹 Cleaning $(SCENARIO_NAME) build artifacts...$(RESET)"
	@rm -rf build/ dist/ *.log
	@rm -f api/$(SCENARIO_NAME) api/$(SCENARIO_NAME)-api
	@rm -rf ui/dist ui/build
	@find . -name "*.pyc" -delete 2>/dev/null || true
	@find . -name "__pycache__" -type d -exec rm -rf {} + 2>/dev/null || true
	@echo "$(GREEN)✓ Cleaned$(RESET)"

build: ## Build API and UI
	@echo "$(BLUE)🔨 Building $(SCENARIO_NAME)...$(RESET)"
	@cd api && go build -o landing-manager-api .
	@if [ -d ui ]; then cd ui && pnpm run build; fi
	@echo "$(GREEN)✓ Build complete$(RESET)"

dev: ## Start development mode (alias for start)
	@$(MAKE) start

fmt: fmt-go fmt-ui ## Format all code (Go + UI)

fmt-go: ## Format Go code
	@echo "$(BLUE)📝 Formatting Go code...$(RESET)"
	@if [ -d api ] && [ -n "$$(find api -name '*.go' 2>/dev/null)" ]; then \
		if command -v gofumpt >/dev/null 2>&1; then \
			cd api && gofumpt -w .; \
		else \
			cd api && go fmt ./...; \
		fi; \
		echo "$(GREEN)✓ Go code formatted$(RESET)"; \
	else \
		echo "$(YELLOW)No Go sources found, skipping$(RESET)"; \
	fi

fmt-ui: ## Format UI code
	@echo "$(BLUE)📝 Formatting UI code...$(RESET)"
	@if [ -d ui ]; then cd ui && pnpm run format 2>/dev/null || echo "No format script in package.json"; fi
	@echo "$(GREEN)✓ UI code formatted$(RESET)"

lint: lint-go lint-ui ## Lint all code (Go + UI)

lint-go: ## Lint Go code
	@echo "$(BLUE)🔍 Linting Go code...$(RESET)"
	@if [ -d api ] && [ -n "$$(find api -name '*.go' 2>/dev/null)" ]; then \
		if command -v golangci-lint >/dev/null 2>&1; then \
			cd api && golangci-lint run; \
		else \
			echo "$(YELLOW)golangci-lint not installed, skipping Go linting$(RESET)"; \
		fi; \
		echo "$(GREEN)✓ Go linting complete$(RESET)"; \
	else \
		echo "$(YELLOW)No Go sources found, skipping$(RESET)"; \
	fi

lint-ui: ## Lint UI code
	@echo "$(BLUE)🔍 Linting UI code...$(RESET)"
	@if [ -d ui ]; then cd ui && pnpm run lint 2>/dev/null || echo "No lint script in package.json"; fi
	@echo "$(GREEN)✓ UI linting complete$(RESET)"

check: fmt lint test ## Run formatting, linting, and tests
	@echo "$(GREEN)✓ All checks passed$(RESET)"
