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

.PHONY: help start stop run dev test logs status clean build install-deps fmt fmt-go fmt-ui lint lint-go lint-ui check

# Default target
.DEFAULT_GOAL := help

# Get scenario name from current directory
SCENARIO_NAME := $(notdir $(CURDIR))

# Standard colors (STRICT: only these are allowed)
GREEN := \033[1;32m
YELLOW := \033[1;33m
BLUE := \033[1;34m
RED := \033[1;31m
RESET := \033[0m

help: ## Show this help message
	@echo "$(BLUE)🏗️  $(SCENARIO_NAME) Scenario Commands$(RESET)"
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

start: ## Start this scenario (via Vrooli lifecycle)
	@echo "$(BLUE)🚀 Starting $(SCENARIO_NAME)...$(RESET)"
	@vrooli scenario start $(SCENARIO_NAME)

stop: ## Stop this scenario
	@echo "$(YELLOW)⏹️  Stopping $(SCENARIO_NAME)...$(RESET)"
	@vrooli scenario stop $(SCENARIO_NAME)

run: start ## Alias for start

dev: ## Start in development mode
	@echo "$(BLUE)🔧 Starting $(SCENARIO_NAME) in development mode...$(RESET)"
	@vrooli scenario develop $(SCENARIO_NAME)

test: ## Run all tests
	@echo "$(BLUE)🧪 Running tests for $(SCENARIO_NAME)...$(RESET)"
	@bash test/phases/test-structure.sh
	@bash test/phases/test-dependencies.sh
	@bash test/phases/test-integration.sh

logs: ## Show scenario logs
	@echo "$(BLUE)📜 Logs for $(SCENARIO_NAME):$(RESET)"
	@vrooli scenario logs $(SCENARIO_NAME) --tail 50

status: ## Check scenario status
	@echo "$(BLUE)📊 Status of $(SCENARIO_NAME):$(RESET)"
	@vrooli scenario status $(SCENARIO_NAME)

clean: ## Clean build artifacts
	@echo "$(YELLOW)🧹 Cleaning build artifacts...$(RESET)"
	@rm -rf build/ dist/ *.log
	@rm -f api/git-control-tower-api
	@rm -rf ui/dist ui/build
	@echo "$(GREEN)✓ Cleaned$(RESET)"

build: ## Build the scenario
	@echo "$(BLUE)🏗️  Building $(SCENARIO_NAME)...$(RESET)"
	@if [ -f api/go.mod ]; then \
		cd api && go build -o git-control-tower-api main.go; \
		echo "$(GREEN)✓ Built api/git-control-tower-api$(RESET)"; \
	fi

install-deps: ## Install dependencies
	@echo "$(BLUE)📦 Installing dependencies...$(RESET)"
	@if [ -f api/go.mod ]; then \
		cd api && go mod download; \
	fi
	@if [ -f ui/package.json ]; then \
		cd ui && npm install; \
	fi
	@echo "$(GREEN)✓ Dependencies installed$(RESET)"

fmt: fmt-go fmt-ui ## Format all code (Go + UI)

fmt-go: ## Format Go code
	@echo "$(BLUE)🎨 Formatting Go code...$(RESET)"
	@if [ -d api ] && find api -name "*.go" | head -1 | grep -q .; then \
		if command -v gofumpt >/dev/null 2>&1; then \
			cd api && gofumpt -w .; \
		elif command -v gofmt >/dev/null 2>&1; then \
			cd api && gofmt -w .; \
		fi; \
		echo "$(GREEN)✓ Go code formatted$(RESET)"; \
	fi

fmt-ui: ## Format UI code (no-op - no UI yet)
	@echo "$(BLUE)ℹ️  No UI to format$(RESET)"

lint: lint-go lint-ui ## Lint all code (Go + UI)

lint-go: ## Lint Go code
	@echo "$(BLUE)🔍 Linting Go code...$(RESET)"
	@if [ -d api ] && find api -name "*.go" | head -1 | grep -q .; then \
		if command -v golangci-lint >/dev/null 2>&1; then \
			cd api && golangci-lint run; \
		elif command -v go >/dev/null 2>&1; then \
			cd api && go vet ./...; \
		fi; \
		echo "$(GREEN)✓ Go code linted$(RESET)"; \
	fi

lint-ui: ## Lint UI code (no-op - no UI yet)
	@echo "$(BLUE)ℹ️  No UI to lint$(RESET)"

check: ## Format, lint, and test
	@echo "$(BLUE)✅ Running full check...$(RESET)"
	@$(MAKE) fmt
	@$(MAKE) lint
	@$(MAKE) test
