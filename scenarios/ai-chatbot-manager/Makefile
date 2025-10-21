# AI Chatbot Manager Scenario Makefile
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

.PHONY: help start run stop test logs status clean build dev fmt fmt-go fmt-ui lint lint-go lint-ui check test-p1 health api-test ws-test widget-test cli-test ports errors restart

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

VROOLI_CMD := vrooli scenario

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

test: ## Run tests for this scenario
	@echo "$(BLUE)🧪 Testing $(SCENARIO_NAME) scenario...$(RESET)"
	@vrooli scenario test $(SCENARIO_NAME)
	@echo "$(GREEN)✅ All tests completed successfully!$(RESET)"

test-p1: ## Test P1 features (multi-tenant, A/B testing, CRM)
	@echo "$(BLUE)🧪 Testing P1 features...$(RESET)"
	@if [ -f test/test-p1-features.sh ]; then \
		bash test/test-p1-features.sh; \
	else \
		echo "$(YELLOW)⚠️  test/test-p1-features.sh not found$(RESET)"; \
	fi

api-test: ## Test API endpoints
	@echo "$(BLUE)🔌 Testing API endpoints...$(RESET)"
	@if [ -f test/test-api-endpoints.sh ]; then \
		bash test/test-api-endpoints.sh; \
	else \
		echo "$(YELLOW)⚠️  test/test-api-endpoints.sh not found$(RESET)"; \
	fi

ws-test: ## Test WebSocket functionality
	@echo "$(BLUE)💬 Testing WebSocket functionality...$(RESET)"
	@if [ -f test/test-websocket.sh ]; then \
		bash test/test-websocket.sh; \
	else \
		echo "$(YELLOW)⚠️  test/test-websocket.sh not found$(RESET)"; \
	fi

widget-test: ## Test widget generation
	@echo "$(BLUE)🎨 Testing widget generation...$(RESET)"
	@if [ -f test/test-widget-generation.sh ]; then \
		bash test/test-widget-generation.sh; \
	else \
		echo "$(YELLOW)⚠️  test/test-widget-generation.sh not found$(RESET)"; \
	fi

cli-test: ## Test CLI commands
	@echo "$(BLUE)🖥️  Testing CLI commands...$(RESET)"
	@if [ -f cli/ai-chatbot-manager.bats ]; then \
		cd cli && bats ai-chatbot-manager.bats; \
	else \
		echo "$(YELLOW)Testing basic CLI commands...$(RESET)"; \
		ai-chatbot-manager --help > /dev/null && echo "$(GREEN)✓ CLI help works$(RESET)" || echo "$(RED)❌ CLI not installed$(RESET)"; \
		ai-chatbot-manager version > /dev/null && echo "$(GREEN)✓ CLI version works$(RESET)" || echo "$(RED)❌ Version command failed$(RESET)"; \
	fi

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
	@if [ -d ui/dist ]; then rm -rf ui/dist; fi
	@if [ -d ui/build ]; then rm -rf ui/build; fi
	@echo "$(GREEN)✓ Cleaned$(RESET)"

build: ## Build the scenario (if applicable)
	@echo "$(BLUE)🏗️  Building $(SCENARIO_NAME)...$(RESET)"
	@if [ -f api/go.mod ]; then \
		cd api && go build -o $(SCENARIO_NAME)-api ./cmd/server/main.go 2>/dev/null || go build -o $(SCENARIO_NAME)-api main.go 2>/dev/null || echo "Could not build Go API"; \
	elif [ -f ui/package.json ]; then \
		cd ui && npm run build 2>/dev/null || echo "Could not build UI"; \
	else \
		echo "$(YELLOW)No build needed for this scenario$(RESET)"; \
	fi

fmt: ## Format all code
	@$(MAKE) fmt-go
	@$(MAKE) fmt-ui

fmt-go: ## Format Go code
	@echo "$(BLUE)🎨 Formatting Go code...$(RESET)"
	@if [ -f api/go.mod ]; then \
		cd api && gofumpt -w . && echo "$(GREEN)✓ Go code formatted$(RESET)"; \
	else \
		echo "$(YELLOW)No Go code to format$(RESET)"; \
	fi

fmt-ui: ## Format TypeScript/JavaScript code
	@echo "$(BLUE)🎨 Formatting UI code...$(RESET)"
	@if [ -f ui/package.json ]; then \
		cd ui && npm run format 2>/dev/null || echo "$(YELLOW)No format script defined$(RESET)"; \
	else \
		echo "$(YELLOW)No UI code to format$(RESET)"; \
	fi

lint: ## Lint all code
	@$(MAKE) lint-go
	@$(MAKE) lint-ui

lint-go: ## Lint Go code
	@echo "$(BLUE)🔍 Linting Go code...$(RESET)"
	@if [ -f api/go.mod ]; then \
		cd api && golangci-lint run && echo "$(GREEN)✓ Go code linted$(RESET)"; \
	else \
		echo "$(YELLOW)No Go code to lint$(RESET)"; \
	fi

lint-ui: ## Lint TypeScript/JavaScript code
	@echo "$(BLUE)🔍 Linting UI code...$(RESET)"
	@if [ -f ui/package.json ]; then \
		cd ui && npm run lint 2>/dev/null || echo "$(YELLOW)No lint script defined$(RESET)"; \
	else \
		echo "$(YELLOW)No UI code to lint$(RESET)"; \
	fi

check: ## Format, lint, and test code (pre-commit workflow)
	@echo "$(BLUE)🔍 Running pre-commit checks...$(RESET)"
	@$(MAKE) fmt
	@$(MAKE) lint
	@$(MAKE) test
	@echo "$(GREEN)✅ All checks passed!$(RESET)"

start: run  ## Alias for run

health: ## Check health of all components
	@echo "$(BLUE)🏥 Checking component health...$(RESET)"
	@API_PORT=$$($(VROOLI_CMD) port $(SCENARIO_NAME) API_PORT 2>/dev/null); \
	if [ -n "$$API_PORT" ]; then \
		echo "$(GREEN)API Health:$(RESET)"; \
		curl -sf http://localhost:$$API_PORT/health | jq '.' 2>/dev/null || echo "$(RED)❌ API not responding$(RESET)"; \
		UI_PORT=$$($(VROOLI_CMD) port $(SCENARIO_NAME) UI_PORT 2>/dev/null); \
		echo "$(GREEN)UI Health:$(RESET)"; \
		curl -sf http://localhost:$$UI_PORT/ > /dev/null && echo "$(GREEN)✓ UI responding on port $$UI_PORT$(RESET)" || echo "$(RED)❌ UI not responding$(RESET)"; \
		echo "$(GREEN)WebSocket Health:$(RESET)"; \
		echo "$(GREEN)✓ WebSocket endpoint available at ws://localhost:$$API_PORT/api/v1/ws/{chatbot_id}$(RESET)"; \
	else \
		echo "$(YELLOW)⚠️  Services not running. Start with: make run$(RESET)"; \
	fi

ports: ## Display allocated ports and access URLs
	@echo "$(BLUE)🔢 Allocated Ports:$(RESET)"
	@API_PORT=$$($(VROOLI_CMD) port $(SCENARIO_NAME) API_PORT 2>/dev/null); \
	UI_PORT=$$($(VROOLI_CMD) port $(SCENARIO_NAME) UI_PORT 2>/dev/null); \
	if [ -n "$$API_PORT" ]; then \
		echo "  API Port: $$API_PORT"; \
		echo "  UI Port:  $$UI_PORT"; \
		echo ""; \
		echo "$(GREEN)Access URLs:$(RESET)"; \
		echo "  Dashboard: http://localhost:$$UI_PORT"; \
		echo "  API:       http://localhost:$$API_PORT/api/v1"; \
		echo "  Health:    http://localhost:$$API_PORT/health"; \
		echo "  WebSocket: ws://localhost:$$API_PORT/api/v1/ws/{chatbot_id}"; \
	else \
		echo "$(YELLOW)⚠️  Services not running. Start with: make run$(RESET)"; \
	fi

restart: ## Restart all services
	@echo "$(YELLOW)🔄 Restarting $(SCENARIO_NAME)...$(RESET)"
	@$(MAKE) stop
	@sleep 2
	@$(MAKE) run

errors: ## Watch logs for errors
	@echo "$(RED)🔍 Watching for errors in logs...$(RESET)"
	@$(VROOLI_CMD) logs $(SCENARIO_NAME) | grep -E "(ERROR|FATAL|PANIC|error|Error|failed|Failed)" --color=always

# Shortcuts
r: run
s: stop
d: dev
t: test
l: logs
c: clean
b: build
h: health
p: ports
