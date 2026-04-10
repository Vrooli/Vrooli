# Full Scenario Template Makefile
# 
# This Makefile ensures scenarios are always run through the Vrooli lifecycle system.
# NEVER run scenarios directly (./api/binary). ALWAYS use these commands.
#
# Usage:
#   make           - Show help
#   make start     - Start this scenario
#   make stop      - Stop this scenario  
#   make dev       - Start in development mode with hot reload
#   make test      - Run all tests
#   make logs      - Show scenario logs
#   make clean     - Clean build artifacts
#   make db-reset  - Reset database (if applicable)

.PHONY: help start run stop dev test test-api test-ui logs status clean build db-reset db-migrate validate fmt fmt-go fmt-ui lint lint-go lint-ui check

# Default target - show help
.DEFAULT_GOAL := help

# Get scenario name from current directory
SCENARIO_NAME := $(notdir $(CURDIR))

# Colors for output
GREEN := \033[1;32m
YELLOW := \033[1;33m
BLUE := \033[1;34m
RED := \033[1;31m
CYAN := \033[1;36m
RESET := \033[0m

help: ## Show this help message
	@echo "$(BLUE)🏗️  $(SCENARIO_NAME) Template Commands$(RESET)"
	@echo ""
	@echo "$(YELLOW)Usage:$(RESET)"
	@echo "  make <command>"
	@echo ""
	@echo "$(YELLOW)Essential Commands:$(RESET)"
	@grep -E '^(start|run|stop|dev|test|logs|help):.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  $(GREEN)%-12s$(RESET) %s\n", $$1, $$2}'
	@echo ""
	@echo "$(YELLOW)Additional Commands:$(RESET)"
	@grep -E '^[a-zA-Z0-9_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		grep -v -E '^(start|run|stop|dev|test|logs|help):' | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  $(GREEN)%-12s$(RESET) %s\n", $$1, $$2}'
	@echo ""
	@echo "$(RED)⚠️  IMPORTANT:$(RESET) This is a TEMPLATE - copy to scenarios/your-name first!"
	@echo "    Always use 'make start' or 'vrooli scenario start <name>' for actual scenarios"

start: ## Start this scenario (uses Vrooli lifecycle)
	@echo "$(BLUE)🚀 Starting $(SCENARIO_NAME) scenario...$(RESET)"
	@echo "$(CYAN)Note: This is a comprehensive template for business applications$(RESET)"
	@vrooli scenario start $(SCENARIO_NAME)

stop: ## Stop this scenario
	@echo "$(YELLOW)⏹️  Stopping $(SCENARIO_NAME) scenario...$(RESET)"
	@vrooli scenario stop $(SCENARIO_NAME)

dev: ## Start in development mode with hot reload
	@echo "$(BLUE)🔧 Starting $(SCENARIO_NAME) in development mode...$(RESET)"
	@vrooli scenario start $(SCENARIO_NAME) --dev

test: ## Run all tests (API, UI, CLI)
	@echo "$(BLUE)🧪 Running all tests for $(SCENARIO_NAME)...$(RESET)"
	@$(MAKE) test-api
	@$(MAKE) test-ui
	@$(MAKE) test-cli

test-api: ## Run API tests
	@echo "$(BLUE)🧪 Testing API...$(RESET)"
	@if [ -f api/go.mod ]; then \
		cd api && go test ./... -v; \
	elif [ -f api/package.json ]; then \
		cd api && npm test; \
	else \
		echo "$(YELLOW)No API tests found$(RESET)"; \
	fi

test-ui: ## Run UI tests
	@echo "$(BLUE)🧪 Testing UI...$(RESET)"
	@if [ -f ui/package.json ]; then \
		cd ui && pnpm test; \
	else \
		echo "$(YELLOW)No UI tests found$(RESET)"; \
	fi

test-cli: ## Run CLI tests (BATS)
	@echo "$(BLUE)🧪 Testing CLI...$(RESET)"
	@if [ -f "cli/$(SCENARIO_NAME).bats" ]; then \
		( cd cli && bats "$(SCENARIO_NAME).bats" ); \
	else \
		echo "$(YELLOW)No CLI tests found$(RESET)"; \
	fi

logs: ## Show recent logs for this scenario
	@echo "$(BLUE)📜 Logs for $(SCENARIO_NAME):$(RESET)"
	@vrooli scenario logs $(SCENARIO_NAME) --tail 50

logs-follow: ## Follow logs in real-time
	@echo "$(BLUE)📜 Following logs for $(SCENARIO_NAME):$(RESET)"
	@vrooli scenario logs $(SCENARIO_NAME) --follow

status: ## Check if scenario is running
	@echo "$(BLUE)📊 Status of $(SCENARIO_NAME):$(RESET)"
	@vrooli scenario status $(SCENARIO_NAME)

clean: ## Clean build artifacts
	@echo "$(YELLOW)🧹 Cleaning $(SCENARIO_NAME) build artifacts...$(RESET)"
	@rm -rf build/ dist/ *.log
	@rm -f api/$(SCENARIO_NAME) api/$(SCENARIO_NAME)-api
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
		cd api && go build -o $(SCENARIO_NAME)-api ./cmd/server; \
		echo "$(GREEN)✓ Built api/$(SCENARIO_NAME)-api$(RESET)"; \
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
		cd ui && pnpm run build; \
		echo "$(GREEN)✓ Built UI$(RESET)"; \
	else \
		echo "$(YELLOW)No UI to build$(RESET)"; \
	fi

db-reset: ## Reset database (drop and recreate)
	@echo "$(RED)⚠️  Resetting database for $(SCENARIO_NAME)...$(RESET)"
	@if [ -f initialization/storage/postgres/schema.sql ]; then \
		vrooli scenario db-reset $(SCENARIO_NAME); \
		echo "$(GREEN)✓ Database reset$(RESET)"; \
	else \
		echo "$(YELLOW)No database to reset$(RESET)"; \
	fi

db-migrate: ## Run database migrations
	@echo "$(BLUE)🔄 Running database migrations...$(RESET)"
	@if [ -f initialization/storage/postgres/migrations ]; then \
		vrooli scenario db-migrate $(SCENARIO_NAME); \
		echo "$(GREEN)✓ Migrations complete$(RESET)"; \
	else \
		echo "$(YELLOW)No migrations to run$(RESET)"; \
	fi

validate: ## Validate scenario structure
	@echo "$(BLUE)✅ Validating $(SCENARIO_NAME) structure...$(RESET)"
	@vrooli scenario validate $(SCENARIO_NAME)

install-deps: ## Install dependencies
	@echo "$(BLUE)📦 Installing dependencies...$(RESET)"
	@if [ -f api/go.mod ]; then \
		cd api && go mod download; \
	fi
	@if [ -f ui/package.json ]; then \
		cd ui && pnpm install; \
	fi
	@echo "$(GREEN)✓ Dependencies installed$(RESET)"

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

# Development shortcuts
run: start
r: start
s: stop
d: dev
t: test
l: logs
c: clean
b: build
f: fmt
