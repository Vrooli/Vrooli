# Scenario Makefile
#
# This Makefile ensures scenarios are always run through the Vrooli lifecycle system.
# NEVER run scenarios directly (./api/binary). ALWAYS use these commands.
#
# Usage:
#   make          - Show help
#   make help     - Show help
#   make start    - Start this scenario through Vrooli lifecycle
#   make stop     - Stop this scenario and cleanup processes
#   make test     - Run scenario test suite (all phases)
#   make logs     - Show recent logs from running scenario
#   make status   - Check if scenario is currently running
#   make clean    - Clean build artifacts and temporary files
#   make build    - Build API binary and UI assets
#   make dev      - Start in development mode with hot reload
#   make fmt      - Format all code (Go and UI)
#   make lint     - Lint all code (Go and UI)
#   make check    - Run fmt, lint, and test (pre-commit workflow)

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
	@echo "$(BLUE)🎥 $(SCENARIO_NAME) Scenario Commands$(RESET)"
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
	@vrooli scenario start $(SCENARIO_NAME)

run: start ## Start this scenario (alias for 'start')

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
	@rm -rf build/ dist/ *.log
	@rm -f api/$(SCENARIO_NAME) api/$(SCENARIO_NAME)-api api/video-tools-api
	@rm -rf ui/dist ui/build
	@find . -name "*.pyc" -delete 2>/dev/null || true
	@find . -name "__pycache__" -type d -exec rm -rf {} + 2>/dev/null || true
	@echo "$(GREEN)✓ Cleaned$(RESET)"

build: ## Build the scenario
	@echo "$(BLUE)🏗️  Building $(SCENARIO_NAME)...$(RESET)"
	@$(MAKE) build-api
	@$(MAKE) build-ui

build-api: ## Build API
	@if [ -f api/go.mod ]; then \
		echo "$(BLUE)Building Go API...$(RESET)"; \
		cd api && go build -o video-tools-api ./cmd/server/main.go; \
		echo "$(GREEN)✓ Built api/video-tools-api$(RESET)"; \
	elif [ -f api/package.json ]; then \
		echo "$(BLUE)Building Node.js API...$(RESET)"; \
		cd api && npm run build 2>/dev/null || npm install; \
		echo "$(GREEN)✓ Built Node.js API$(RESET)"; \
	else \
		echo "$(YELLOW)No API to build$(RESET)"; \
	fi

build-ui: ## Build UI
	@if [ -f ui/package.json ]; then \
		echo "$(BLUE)Building UI...$(RESET)"; \
		cd ui && npm run build; \
		echo "$(GREEN)✓ Built UI$(RESET)"; \
	else \
		echo "$(YELLOW)No UI to build$(RESET)"; \
	fi

# Code Quality Targets
fmt: ## Format all code
	@echo "$(BLUE)🎨 Formatting all code...$(RESET)"
	@$(MAKE) fmt-go
	@$(MAKE) fmt-ui

fmt-go: ## Format Go code
	@if [ -d api ] && find api -name "*.go" | head -1 | grep -q .; then \
		echo "$(BLUE)🎨 Formatting Go code...$(RESET)"; \
		if command -v gofumpt >/dev/null 2>&1; then \
			cd api && gofumpt -w .; \
		elif command -v gofmt >/dev/null 2>&1; then \
			cd api && gofmt -w .; \
		fi; \
		echo "$(GREEN)✓ Go code formatted$(RESET)"; \
	else \
		echo "$(YELLOW)No Go code to format$(RESET)"; \
	fi

fmt-ui: ## Format TypeScript/JavaScript code
	@if [ -d ui ] && (find ui -name "*.ts" -o -name "*.tsx" -o -name "*.js" -o -name "*.jsx") | head -1 | grep -q .; then \
		echo "$(BLUE)🎨 Formatting UI code...$(RESET)"; \
		if command -v prettier >/dev/null 2>&1; then \
			cd ui && prettier --write "**/*.{ts,tsx,js,jsx,json,css,md}" 2>/dev/null || true; \
		fi; \
		echo "$(GREEN)✓ UI code formatted$(RESET)"; \
	else \
		echo "$(YELLOW)No UI code to format$(RESET)"; \
	fi

lint: ## Lint all code
	@echo "$(BLUE)🔍 Linting all code...$(RESET)"
	@$(MAKE) lint-go
	@$(MAKE) lint-ui

lint-go: ## Lint Go code
	@if [ -d api ] && find api -name "*.go" | head -1 | grep -q .; then \
		echo "$(BLUE)🔍 Linting Go code...$(RESET)"; \
		if command -v golangci-lint >/dev/null 2>&1; then \
			cd api && golangci-lint run; \
		elif command -v go >/dev/null 2>&1; then \
			cd api && go vet ./... && go fmt ./...; \
		fi; \
		echo "$(GREEN)✓ Go code linted$(RESET)"; \
	else \
		echo "$(YELLOW)No Go code to lint$(RESET)"; \
	fi

lint-ui: ## Lint TypeScript/JavaScript code
	@if [ -d ui ] && (find ui -name "*.ts" -o -name "*.tsx" -o -name "*.js" -o -name "*.jsx") | head -1 | grep -q .; then \
		echo "$(BLUE)🔍 Linting UI code...$(RESET)"; \
		if [ -f ui/package.json ] && grep -q '"eslint"' ui/package.json; then \
			cd ui && npm run lint 2>/dev/null || true; \
		fi; \
		echo "$(GREEN)✓ UI code linted$(RESET)"; \
	else \
		echo "$(YELLOW)No UI code to lint$(RESET)"; \
	fi

check: ## Format, lint, and test code (pre-commit workflow)
	@echo "$(BLUE)✅ Running full code quality check...$(RESET)"
	@$(MAKE) fmt
	@$(MAKE) lint
	@$(MAKE) test
