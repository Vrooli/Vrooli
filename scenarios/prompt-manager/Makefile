# Prompt Manager Makefile
# Comprehensive management interface for the prompt-manager scenario

SCENARIO_NAME := prompt-manager
VROOLI_CMD := vrooli scenario

# Color codes for output
RED := \033[0;31m
GREEN := \033[0;32m
YELLOW := \033[1;33m
BLUE := \033[0;34m
NC := \033[0m # No Color

.PHONY: help
help: ## Show this help message
	@echo "$(BLUE)╔══════════════════════════════════════════════════════════════╗$(NC)"
	@echo "$(BLUE)║           📝 Prompt Manager - Make Commands 📝              ║$(NC)"
	@echo "$(BLUE)╚══════════════════════════════════════════════════════════════╝$(NC)"
	@echo ""
	@echo "$(GREEN)Available commands:$(NC)"
	@echo ""
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  $(YELLOW)%-15s$(NC) %s\n", $$1, $$2}'
	@echo ""
	@echo "$(BLUE)Examples:$(NC)"
	@echo "  make setup      # Initial setup of the scenario"
	@echo "  make run        # Start the scenario"
	@echo "  make test       # Run tests"
	@echo "  make logs       # View logs"
	@echo "  make stop       # Stop the scenario"

.PHONY: setup
setup: ## Initial setup and build
	@echo "$(GREEN)🔧 Setting up Prompt Manager...$(NC)"
	@$(VROOLI_CMD) setup $(SCENARIO_NAME)
	@echo "$(GREEN)✅ Setup complete!$(NC)"

.PHONY: run
run: ## Start the scenario
	@echo "$(GREEN)🚀 Starting Prompt Manager...$(NC)"
	@$(VROOLI_CMD) start $(SCENARIO_NAME)
	@echo ""
	@echo "$(GREEN)✅ Prompt Manager is running!$(NC)"
	@echo "$(BLUE)📍 Dashboard: http://localhost:$${UI_PORT:-35000}$(NC)"
	@echo "$(BLUE)📍 API: http://localhost:$${API_PORT:-15000}$(NC)"
	@echo "$(BLUE)📍 CLI: prompt-manager --help$(NC)"

.PHONY: start
start: run ## Alias for 'run'

.PHONY: stop
stop: ## Stop the scenario
	@echo "$(YELLOW)⏹️  Stopping Prompt Manager...$(NC)"
	@$(VROOLI_CMD) stop $(SCENARIO_NAME)
	@echo "$(GREEN)✅ Stopped$(NC)"

.PHONY: restart
restart: stop run ## Restart the scenario

.PHONY: status
status: ## Check scenario status
	@echo "$(BLUE)📊 Checking Prompt Manager status...$(NC)"
	@$(VROOLI_CMD) status $(SCENARIO_NAME) || true
	@echo ""
	@echo "$(BLUE)🔍 Process status:$(NC)"
	@ps aux | grep -E "prompt-manager-api|scenarios/prompt-manager.*server\.js" | grep -v grep || echo "  No processes running"

.PHONY: logs
logs: ## View scenario logs
	@echo "$(BLUE)📋 Prompt Manager logs:$(NC)"
	@$(VROOLI_CMD) logs $(SCENARIO_NAME)

.PHONY: test
test: ## Run tests for this scenario
	@echo "$(BLUE)🧪 Testing $(SCENARIO_NAME) scenario...$(RESET)"
	@vrooli scenario test $(SCENARIO_NAME)

.PHONY: clean
clean: ## Clean build artifacts and data
	@echo "$(YELLOW)🧹 Cleaning Prompt Manager...$(NC)"
	@rm -rf api/prompt-manager-api
	@rm -rf ui/node_modules
	@rm -rf data/campaigns/*
	@echo "$(GREEN)✅ Cleaned$(NC)"

.PHONY: reset
reset: stop clean setup ## Full reset (stop, clean, setup)
	@echo "$(GREEN)✅ Prompt Manager has been reset$(NC)"

.PHONY: build-api
build-api: ## Build API binary
	@echo "$(BLUE)🔨 Building API...$(NC)"
	@cd api && go mod download && go build -o prompt-manager-api .
	@echo "$(GREEN)✅ API built$(NC)"

.PHONY: build-ui
build-ui: ## Build UI
	@echo "$(BLUE)🔨 Building UI...$(NC)"
	@cd ui && npm install
	@echo "$(GREEN)✅ UI dependencies installed$(NC)"

.PHONY: dev
dev: ## Start in development mode with live logs
	@echo "$(GREEN)🔧 Starting Prompt Manager in development mode...$(NC)"
	@$(VROOLI_CMD) develop $(SCENARIO_NAME)

.PHONY: cli-test
cli-test: ## Test CLI commands
	@echo "$(BLUE)🧪 Testing CLI...$(NC)"
	@prompt-manager status || echo "$(YELLOW)⚠️  CLI not installed or service not running$(NC)"
	@prompt-manager list-campaigns || true
	@echo "$(GREEN)✅ CLI test complete$(NC)"

.PHONY: health
health: ## Check health endpoints
	@echo "$(BLUE)🏥 Checking health...$(NC)"
	@curl -sf http://localhost:$${API_PORT:-15000}/health | jq '.' || echo "$(RED)❌ API not responding$(NC)"
	@curl -sf http://localhost:$${UI_PORT:-35000}/health | jq '.' || echo "$(RED)❌ UI not responding$(NC)"

.PHONY: db-status
db-status: ## Check database connection
	@echo "$(BLUE)🗄️  Checking database status...$(NC)"
	@curl -sf http://localhost:$${API_PORT:-15000}/health | jq '.services.database' || echo "$(RED)❌ Cannot check database status$(NC)"

.PHONY: quick-test
quick-test: ## Quick API test
	@echo "$(BLUE)⚡ Quick API test...$(NC)"
	@curl -sf http://localhost:$${API_PORT:-15000}/api/v1/campaigns | jq '.' || echo "$(RED)❌ API test failed$(NC)"
	@echo "$(GREEN)✅ Quick test complete$(NC)"

# Default target
.DEFAULT_GOAL := help