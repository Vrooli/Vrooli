# Calendar Scenario Makefile
# 
# This Makefile ensures scenarios are always run through the Vrooli lifecycle system.
# NEVER run scenarios directly (./api/calendar-api). ALWAYS use these commands.
#
# Usage:
#   make       - Show help
#   make start - Start this scenario
#   make stop  - Stop this scenario
#   make test  - Run scenario tests
#   make logs  - Show scenario logs
#   make clean - Clean build artifacts
#   make run - Start this scenario in lifecycle-managed mode

.PHONY: help start run stop test logs status clean build dev fmt fmt-go fmt-ui lint lint-go lint-ui check schema migrate

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
	@echo "$(BLUE)📅 $(SCENARIO_NAME) Scenario Commands$(RESET)"
	@echo ""
	@echo "$(YELLOW)Usage:$(RESET)"
	@echo "  make <command>"
	@echo ""
	@echo "$(YELLOW)Commands:$(RESET)"
	@grep -E '^[a-zA-Z0-9_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  $(GREEN)%-12s$(RESET) %s\n", $$1, $$2}'
	@echo ""
	@echo "$(RED)⚠️  IMPORTANT:$(RESET) Never run ./api/calendar-api directly!"
	@echo "    Always use 'make start' or 'vrooli scenario start $(SCENARIO_NAME)'"

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

logs: ## Show recent logs for this scenario
	@echo "$(BLUE)📜 Logs for $(SCENARIO_NAME):$(RESET)"
	@vrooli scenario logs $(SCENARIO_NAME) --tail 50

status: ## Check if scenario is running
	@echo "$(BLUE)📊 Status of $(SCENARIO_NAME):$(RESET)"
	@vrooli scenario status $(SCENARIO_NAME)

clean: ## Clean build artifacts
	@echo "$(YELLOW)🧹 Cleaning $(SCENARIO_NAME) build artifacts...$(RESET)"
	@rm -rf build/ dist/ *.log api/calendar-api api/$(SCENARIO_NAME)
	@if [ -d api ]; then cd api && go clean 2>/dev/null || true; fi
	@if [ -d ui/dist ]; then rm -rf ui/dist; fi
	@if [ -d ui/build ]; then rm -rf ui/build; fi
	@if [ -d ui/node_modules/.cache ]; then rm -rf ui/node_modules/.cache; fi
	@echo "$(GREEN)✓ Cleaned$(RESET)"

build: ## Build the scenario components
	@echo "$(BLUE)🏗️  Building $(SCENARIO_NAME)...$(RESET)"
	@if [ -f api/go.mod ]; then \
		echo "Building Go API..."; \
		cd api && go build -o calendar-api .; \
	fi
	@if [ -f ui/package.json ]; then \
		echo "Building React UI..."; \
		cd ui && npm ci && npm run build; \
	fi
	@echo "$(GREEN)✓ Build complete$(RESET)"

# ============================================================================
# Database Management
# ============================================================================

schema: ## Apply database schema
	@echo "$(BLUE)🗃️  Applying database schema...$(RESET)"
	@if [ -f initialization/postgres/schema.sql ]; then \
		resource-postgres exec calendar-system < initialization/postgres/schema.sql; \
		echo "$(GREEN)✓ Schema applied$(RESET)"; \
	else \
		echo "$(RED)❌ Schema file not found: initialization/postgres/schema.sql$(RESET)"; \
		exit 1; \
	fi

migrate: ## Run database migrations
	@echo "$(BLUE)🔄 Running database migrations...$(RESET)"
	@if [ -d api/migrations ]; then \
		cd api && go run cmd/migrate/main.go up 2>/dev/null || echo "$(YELLOW)Migration tool not implemented yet$(RESET)"; \
	else \
		echo "$(YELLOW)No migrations configured yet$(RESET)"; \
	fi

# ============================================================================
# Code Quality Targets
# ============================================================================

fmt: ## Format all code
	@echo "$(BLUE)🎨 Formatting code...$(RESET)"
	@$(MAKE) fmt-go
	@$(MAKE) fmt-ui

fmt-go: ## Format Go code  
	@if [ -d api ] && find api -name "*.go" | head -1 | grep -q .; then \
		echo "Formatting Go code..."; \
		if command -v gofumpt >/dev/null 2>&1; then \
			cd api && gofumpt -w .; \
		elif command -v gofmt >/dev/null 2>&1; then \
			cd api && gofmt -w .; \
		fi; \
		echo "$(GREEN)✓ Go code formatted$(RESET)"; \
	fi

fmt-ui: ## Format TypeScript/JavaScript code
	@if [ -d ui ] && (find ui/src -name "*.ts" -o -name "*.tsx" -o -name "*.js" -o -name "*.jsx") | head -1 | grep -q .; then \
		echo "Formatting UI code..."; \
		cd ui && npm run format 2>/dev/null || \
		(command -v prettier >/dev/null 2>&1 && prettier --write "src/**/*.{ts,tsx,js,jsx,json,css}" 2>/dev/null) || \
		echo "$(YELLOW)Prettier not configured$(RESET)"; \
		echo "$(GREEN)✓ UI code formatted$(RESET)"; \
	fi

lint: ## Lint all code
	@echo "$(BLUE)🔍 Linting code...$(RESET)" 
	@$(MAKE) lint-go
	@$(MAKE) lint-ui

lint-go: ## Lint Go code
	@if [ -d api ] && find api -name "*.go" | head -1 | grep -q .; then \
		echo "Linting Go code..."; \
		if command -v golangci-lint >/dev/null 2>&1; then \
			cd api && golangci-lint run; \
		else \
			cd api && go vet ./...; \
		fi; \
		echo "$(GREEN)✓ Go code linted$(RESET)"; \
	fi

lint-ui: ## Lint TypeScript/JavaScript code
	@if [ -d ui ] && (find ui/src -name "*.ts" -o -name "*.tsx" -o -name "*.js" -o -name "*.jsx") | head -1 | grep -q .; then \
		echo "Linting UI code..."; \
		cd ui && (npm run lint 2>/dev/null || echo "$(YELLOW)ESLint not configured$(RESET)"); \
		echo "$(GREEN)✓ UI code linted$(RESET)"; \
	fi

typecheck: ## Type check TypeScript code
	@if [ -d ui ] && [ -f ui/tsconfig.json ]; then \
		echo "$(BLUE)🔍 Type checking...$(RESET)"; \
		cd ui && npm run typecheck 2>/dev/null || npx tsc --noEmit; \
		echo "$(GREEN)✓ Type check passed$(RESET)"; \
	fi

check: ## Format, lint, and test code (pre-commit workflow)
	@echo "$(BLUE)✅ Running full code quality check...$(RESET)"
	@$(MAKE) fmt
	@$(MAKE) lint
	@$(MAKE) typecheck
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
f: fmt
lint-all: lint
