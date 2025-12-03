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
#   make run   - Start scenario through lifecycle run (legacy compat)

.PHONY: help start run stop dev test logs status clean build fmt fmt-go fmt-ui lint lint-go lint-ui check

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
	@echo "$(BLUE)🧪 $(SCENARIO_NAME) Scenario Commands$(RESET)"
	@echo ""
	@echo "$(YELLOW)Usage:$(RESET)"
	@echo "  make <command>"
	@echo ""
	@echo "$(YELLOW)Commands:$(RESET)"
	@grep -E '^[a-zA-Z0-9_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  $(GREEN)%-10s$(RESET) %s\n", $$1, $$2}'
	@echo ""
	@echo "$(RED)⚠️  IMPORTANT:$(RESET) Never run ./api/test-genie-api directly!"
	@echo "    Always use 'make start' or 'vrooli scenario start $(SCENARIO_NAME)'"

run: ## Start this scenario (uses Vrooli lifecycle run command)
	@echo "$(BLUE)🚀 Starting $(SCENARIO_NAME) scenario...$(RESET)"
	@vrooli scenario run $(SCENARIO_NAME)

start: ## Start this scenario
	@echo "$(BLUE)🚀 Starting $(SCENARIO_NAME) scenario...$(RESET)"
	@vrooli scenario start $(SCENARIO_NAME)

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
	@rm -rf build/ dist/ *.log api/test-genie-api
	@if [ -d ui/dist ]; then rm -rf ui/dist; fi
	@echo "$(GREEN)✓ Cleaned$(RESET)"

build: ## Build compiled assets when needed
	@echo "$(BLUE)🏗️  Building $(SCENARIO_NAME)...$(RESET)"
	@if [ -d api ] && [ -f api/go.mod ]; then \
		cd api && go build -o test-genie-api ./cmd/test-genie-api/; \
	fi
	@if [ -d cli ] && [ -f cli/go.mod ]; then \
		cd cli && go build -o test-genie .; \
	fi
	@if [ -d ui ] && [ -f ui/package.json ]; then \
		cd ui && pnpm run build; \
	fi

fmt: ## Format code (Go + UI)
	@$(MAKE) fmt-go
	@$(MAKE) fmt-ui

fmt-go: ## Format Go code
	@echo "$(BLUE)🎨 Formatting Go code...$(RESET)"
	@if [ -d api ]; then \
		if command -v gofumpt >/dev/null 2>&1; then \
			cd api && gofumpt -w .; \
		else \
			cd api && gofmt -w .; \
		fi; \
	fi

fmt-ui: ## Format TypeScript/JavaScript
	@echo "$(BLUE)🎨 Formatting UI code...$(RESET)"
	@if [ -d ui ] && command -v prettier >/dev/null 2>&1; then \
		cd ui && prettier --write "**/*.{ts,tsx,js,jsx,json,css,md}" 2>/dev/null || true; \
	fi

lint: ## Lint Go + UI code
	@$(MAKE) lint-go
	@$(MAKE) lint-ui

lint-go: ## Lint Go code
	@echo "$(BLUE)🔍 Linting Go code...$(RESET)"
	@if [ -d api ] && command -v golangci-lint >/dev/null 2>&1; then \
		cd api && golangci-lint run; \
	fi

lint-ui: ## Lint TypeScript/JavaScript code
	@echo "$(BLUE)🔍 Linting UI code...$(RESET)"
	@if [ -f ui/package.json ] && grep -q '"lint"' ui/package.json; then \
		cd ui && pnpm run lint; \
	fi

check: ## Format, lint, and test
	@$(MAKE) fmt
	@$(MAKE) lint
	@$(MAKE) test
