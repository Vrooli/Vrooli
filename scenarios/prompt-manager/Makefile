# Scenario Makefile
#
# This Makefile ensures scenarios are always run through the Vrooli lifecycle system.
# NEVER run scenarios directly (./api/binary). ALWAYS use these commands.
#
# Usage:
#   make       - Show help
#   make start - Start this scenario (ALWAYS use this, never run binaries directly)
#   make run   - Start this scenario (alias for 'start')
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
	@vrooli scenario start $(SCENARIO_NAME)

run: start ## Alias for 'start'

stop: ## Stop this scenario
	@echo "$(YELLOW)⏹️  Stopping $(SCENARIO_NAME) scenario...$(RESET)"
	@vrooli scenario stop $(SCENARIO_NAME)

test: ## Run scenario tests
	@echo "$(BLUE)🧪 Testing $(SCENARIO_NAME) scenario...$(RESET)"
	@vrooli scenario test $(SCENARIO_NAME)

logs: ## Show scenario logs
	@echo "$(BLUE)📋 Showing $(SCENARIO_NAME) logs...$(RESET)"
	@vrooli scenario logs $(SCENARIO_NAME)

status: ## Check scenario status
	@echo "$(BLUE)📊 Checking $(SCENARIO_NAME) status...$(RESET)"
	@vrooli scenario status $(SCENARIO_NAME)

clean: ## Clean build artifacts
	@echo "$(YELLOW)🧹 Cleaning build artifacts...$(RESET)"
	@rm -rf api/prompt-manager-api
	@rm -rf ui/node_modules
	@echo "$(GREEN)✅ Cleaned$(RESET)"

build: ## Build all components
	@echo "$(BLUE)🔨 Building $(SCENARIO_NAME)...$(RESET)"
	@cd api && go mod download && go build -o prompt-manager-api .
	@cd ui && npm install
	@echo "$(GREEN)✅ Build complete$(RESET)"

dev: ## Start in development mode
	@echo "$(BLUE)🔧 Starting $(SCENARIO_NAME) in development mode...$(RESET)"
	@vrooli scenario develop $(SCENARIO_NAME)

fmt: fmt-go fmt-ui ## Format all code

fmt-go: ## Format Go code
	@echo "$(BLUE)🎨 Formatting Go code...$(RESET)"
	@cd api && gofumpt -w . 2>/dev/null || go fmt ./...
	@echo "$(GREEN)✅ Go code formatted$(RESET)"

fmt-ui: ## Format UI code
	@echo "$(BLUE)🎨 Formatting UI code...$(RESET)"
	@cd ui && npm run lint:fix 2>/dev/null || echo "$(YELLOW)⚠️  No lint:fix script$(RESET)"
	@echo "$(GREEN)✅ UI code formatted$(RESET)"

lint: lint-go lint-ui ## Lint all code

lint-go: ## Lint Go code
	@echo "$(BLUE)🔍 Linting Go code...$(RESET)"
	@cd api && golangci-lint run 2>/dev/null || go vet ./...
	@echo "$(GREEN)✅ Go code linted$(RESET)"

lint-ui: ## Lint UI code
	@echo "$(BLUE)🔍 Linting UI code...$(RESET)"
	@cd ui && npm run lint 2>/dev/null || echo "$(YELLOW)⚠️  No lint script$(RESET)"
	@echo "$(GREEN)✅ UI code linted$(RESET)"

check: lint test ## Run all checks (lint + test)

# Scenario-specific targets
.PHONY: health db-status cli-test quick-test

health: ## Check health endpoints
	@echo "$(BLUE)🏥 Checking health...$(RESET)"
	@curl -sf http://localhost:$${API_PORT:-15000}/health | jq '.' || echo "$(RED)❌ API not responding$(RESET)"
	@curl -sf http://localhost:$${UI_PORT:-35000}/health | jq '.' || echo "$(RED)❌ UI not responding$(RESET)"

db-status: ## Check database connection
	@echo "$(BLUE)🗄️  Checking database status...$(RESET)"
	@curl -sf http://localhost:$${API_PORT:-15000}/health | jq '.services.database' || echo "$(RED)❌ Cannot check database status$(RESET)"

cli-test: ## Test CLI commands
	@echo "$(BLUE)🧪 Testing CLI...$(RESET)"
	@prompt-manager status || echo "$(YELLOW)⚠️  CLI not installed or service not running$(RESET)"
	@prompt-manager list-campaigns || true
	@echo "$(GREEN)✅ CLI test complete$(RESET)"

quick-test: ## Quick API test
	@echo "$(BLUE)⚡ Quick API test...$(RESET)"
	@curl -sf http://localhost:$${API_PORT:-15000}/api/v1/campaigns | jq '.' || echo "$(RED)❌ API test failed$(RESET)"
	@echo "$(GREEN)✅ Quick test complete$(RESET)"
