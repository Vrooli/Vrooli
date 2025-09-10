# Ecosystem Manager Scenario
# 
# This Makefile ensures the ecosystem-manager is always run through the Vrooli lifecycle system.
# NEVER run scenarios directly (./api/ecosystem-manager). ALWAYS use these commands.
#
# Usage:
#   make           - Show help
#   make run       - Start ecosystem manager
#   make stop      - Stop ecosystem manager  
#   make dev       - Start in development mode
#   make test      - Run all tests
#   make logs      - Show recent logs
#   make queue     - Show queue status
#   make clean     - Clean build artifacts

.PHONY: help run stop dev test test-api test-ui logs logs-follow status clean build queue queue-pending queue-progress validate

# Default target - show help
.DEFAULT_GOAL := help

# Get scenario name from current directory
SCENARIO_NAME := ecosystem-manager

# Colors for output
GREEN := \033[1;32m
YELLOW := \033[1;33m
BLUE := \033[1;34m
RED := \033[1;31m
CYAN := \033[1;36m
RESET := \033[0m

help: ## Show this help message
	@echo "$(BLUE)🌐 Ecosystem Manager Commands$(RESET)"
	@echo ""
	@echo "$(YELLOW)Usage:$(RESET)"
	@echo "  make <command>"
	@echo ""
	@echo "$(YELLOW)Essential Commands:$(RESET)"
	@grep -E '^(run|stop|dev|test|logs|queue|help):.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  $(GREEN)%-15s$(RESET) %s\n", $$1, $$2}'
	@echo ""
	@echo "$(YELLOW)Additional Commands:$(RESET)"
	@grep -E '^[a-zA-Z0-9_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		grep -v -E '^(run|stop|dev|test|logs|queue|help):' | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  $(GREEN)%-15s$(RESET) %s\n", $$1, $$2}'
	@echo ""
	@echo "$(RED)⚠️  IMPORTANT:$(RESET) Never run ./api/ecosystem-manager directly!"
	@echo "    Always use 'make run' or 'vrooli scenario run ecosystem-manager'"

run: ## Start ecosystem manager (uses Vrooli lifecycle)
	@echo "$(BLUE)🚀 Starting Ecosystem Manager...$(RESET)"
	@echo "$(CYAN)This manages resource and scenario generation/improvement$(RESET)"
	@vrooli scenario run $(SCENARIO_NAME)

stop: ## Stop ecosystem manager
	@echo "$(YELLOW)⏹️  Stopping Ecosystem Manager...$(RESET)"
	@vrooli scenario stop $(SCENARIO_NAME)

dev: ## Start in development mode with hot reload
	@echo "$(BLUE)🔧 Starting Ecosystem Manager in development mode...$(RESET)"
	@vrooli scenario run $(SCENARIO_NAME) --dev

test: ## Run all tests
	@echo "$(BLUE)🧪 Running all Ecosystem Manager tests...$(RESET)"
	@$(MAKE) test-api
	@$(MAKE) test-integration

test-api: ## Run API tests
	@echo "$(BLUE)🧪 Testing API...$(RESET)"
	@cd api && go test ./... -v

test-integration: ## Run integration tests
	@echo "$(BLUE)🧪 Running integration tests...$(RESET)"
	@if [ -f test/orchestrator-discovery-test.sh ]; then \
		./test/orchestrator-discovery-test.sh; \
	fi
	@if [ -f test/claude-integration-test.go ]; then \
		cd test && go test -v *.go; \
	fi

logs: ## Show recent logs
	@echo "$(BLUE)📜 Recent Ecosystem Manager logs:$(RESET)"
	@vrooli scenario logs $(SCENARIO_NAME) --tail 50

logs-follow: ## Follow logs in real-time
	@echo "$(BLUE)📜 Following Ecosystem Manager logs...$(RESET)"
	@vrooli scenario logs $(SCENARIO_NAME) --follow

status: ## Check if ecosystem manager is running
	@echo "$(BLUE)📊 Ecosystem Manager Status:$(RESET)"
	@vrooli scenario status $(SCENARIO_NAME)

queue: ## Show queue status (pending, in-progress, completed)
	@echo "$(CYAN)📋 Queue Status:$(RESET)"
	@echo ""
	@echo "$(YELLOW)Pending tasks:$(RESET)"
	@ls -1 queue/pending/*.yaml 2>/dev/null | wc -l | xargs -I {} echo "  {} tasks"
	@echo ""
	@echo "$(BLUE)In-progress tasks:$(RESET)"
	@ls -1 queue/in-progress/*.yaml 2>/dev/null | wc -l | xargs -I {} echo "  {} tasks"
	@echo ""
	@echo "$(GREEN)Completed tasks:$(RESET)"
	@ls -1 queue/completed/*.yaml 2>/dev/null | wc -l | xargs -I {} echo "  {} tasks"
	@echo ""
	@echo "$(RED)Failed tasks:$(RESET)"
	@ls -1 queue/failed/*.yaml 2>/dev/null | wc -l | xargs -I {} echo "  {} tasks"

queue-pending: ## List pending queue items
	@echo "$(YELLOW)📋 Pending Queue Items:$(RESET)"
	@ls -la queue/pending/*.yaml 2>/dev/null || echo "  No pending items"

queue-progress: ## Show in-progress tasks
	@echo "$(BLUE)⏳ In-Progress Tasks:$(RESET)"
	@ls -la queue/in-progress/*.yaml 2>/dev/null || echo "  No tasks in progress"

clean: ## Clean build artifacts and logs
	@echo "$(YELLOW)🧹 Cleaning Ecosystem Manager artifacts...$(RESET)"
	@rm -rf api/ecosystem-manager api/ecosystem-manager-api
	@rm -f *.log api/*.log ui/*.log
	@rm -rf ui/dist ui/build
	@echo "$(GREEN)✓ Cleaned$(RESET)"

build: ## Build the ecosystem manager
	@echo "$(BLUE)🏗️  Building Ecosystem Manager...$(RESET)"
	@$(MAKE) build-api
	@$(MAKE) build-ui

build-api: ## Build API
	@echo "$(BLUE)Building Go API...$(RESET)"
	@cd api && go build -o ecosystem-manager-api main.go
	@echo "$(GREEN)✓ Built api/ecosystem-manager-api$(RESET)"

build-ui: ## Build UI
	@echo "$(BLUE)Building UI...$(RESET)"
	@cd ui && npm run build
	@echo "$(GREEN)✓ Built UI$(RESET)"

validate: ## Validate scenario structure and prompts
	@echo "$(BLUE)✅ Validating Ecosystem Manager...$(RESET)"
	@vrooli scenario validate $(SCENARIO_NAME)
	@echo "$(CYAN)Checking prompt files...$(RESET)"
	@find prompts -name "*.md" -o -name "*.yaml" | wc -l | xargs -I {} echo "  {} prompt files found"

view-prompt: ## View assembled prompt sections
	@echo "$(CYAN)📝 Viewing assembled prompt:$(RESET)"
	@./view-prompt.sh

install-deps: ## Install dependencies
	@echo "$(BLUE)📦 Installing dependencies...$(RESET)"
	@cd api && go mod download
	@cd ui && npm install
	@echo "$(GREEN)✓ Dependencies installed$(RESET)"

# Quick shortcuts
r: run
s: stop
d: dev
t: test
l: logs
q: queue
c: clean
b: build