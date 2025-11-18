# Secrets Manager Scenario Makefile
# Modernized to track the React + Vite template workflow

.PHONY: help run stop dev test test-api test-ui test-cli logs logs-follow status clean build build-api build-ui db-reset db-migrate validate install-deps fmt fmt-go fmt-ui lint lint-go lint-ui check

.DEFAULT_GOAL := help

SCENARIO_NAME := $(notdir $(CURDIR))

GREEN := \033[1;32m
YELLOW := \033[1;33m
BLUE := \033[1;34m
RED := \033[1;31m
CYAN := \033[1;36m
RESET := \033[0m

help: ## Show this help message
	@echo "$(BLUE)🏗️  $(SCENARIO_NAME) Commands$(RESET)"
	@echo ""
	@echo "$(YELLOW)Usage:$(RESET) make <command>"
	@echo ""
	@echo "$(YELLOW)Essential Commands:$(RESET)"
	@grep -E '^(run|stop|dev|test|logs|help):.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  $(GREEN)%-12s$(RESET) %s\n", $$1, $$2}'
	@echo ""
	@echo "$(YELLOW)Additional Commands:$(RESET)"
	@grep -E '^[a-zA-Z0-9_-]+:.*?## .*$$' $(MAKEFILE_LIST) | grep -v -E '^(run|stop|dev|test|logs|help):' | awk 'BEGIN {FS = ":.*?## "}; {printf "  $(GREEN)%-12s$(RESET) %s\n", $$1, $$2}'
	@echo ""
	@echo "$(RED)⚠️  IMPORTANT:$(RESET) Always start via 'make run' or 'vrooli scenario run $(SCENARIO_NAME)'"

run: ## Start this scenario (uses Vrooli lifecycle)
	@echo "$(BLUE)🚀 Starting $(SCENARIO_NAME) scenario...$(RESET)"
	@vrooli scenario run $(SCENARIO_NAME)

stop: ## Stop this scenario
	@echo "$(YELLOW)⏹️  Stopping $(SCENARIO_NAME) scenario...$(RESET)"
	@vrooli scenario stop $(SCENARIO_NAME)

dev: ## Start in development mode with hot reload
	@echo "$(BLUE)🔧 Starting $(SCENARIO_NAME) in development mode...$(RESET)"
	@vrooli scenario run $(SCENARIO_NAME) --dev

test: ## Run API + UI + CLI tests
	@echo "$(BLUE)🧪 Running all tests for $(SCENARIO_NAME)...$(RESET)"
	@$(MAKE) test-api
	@$(MAKE) test-ui
	@$(MAKE) test-cli

test-api: ## Run Go API tests
	@if [ -f api/go.mod ]; then \
		echo "$(BLUE)🧪 Testing API...$(RESET)"; \
		cd api && go test ./... -v; \
	else \
		echo "$(YELLOW)No API test suite found$(RESET)"; \
	fi

test-ui: ## Run Vite UI tests
	@if [ -f ui/package.json ]; then \
		echo "$(BLUE)🧪 Testing UI...$(RESET)"; \
		cd ui && pnpm test -- --run; \
	else \
		echo "$(YELLOW)No UI test suite found$(RESET)"; \
	fi

test-cli: ## Run CLI tests (BATS)
	@if [ -f "cli/$(SCENARIO_NAME).bats" ]; then \
		echo "$(BLUE)🧪 Testing CLI...$(RESET)"; \
		cd cli && bats "$(SCENARIO_NAME).bats"; \
	else \
		echo "$(YELLOW)No CLI tests found$(RESET)"; \
	fi

logs: ## Show recent logs
	@vrooli scenario logs $(SCENARIO_NAME) --tail 50

logs-follow: ## Follow logs
	@vrooli scenario logs $(SCENARIO_NAME) --follow

status: ## Check running status
	@vrooli scenario status $(SCENARIO_NAME)

clean: ## Clean build artifacts
	@echo "$(YELLOW)🧹 Cleaning $(SCENARIO_NAME)...$(RESET)"
	@rm -rf build/ dist/ *.log api/secrets-manager-api api/secrets-manager
	@rm -rf ui/dist ui/node_modules/.vite
	@find . -name "*.pyc" -delete 2>/dev/null || true
	@find . -name "__pycache__" -type d -exec rm -rf {} + 2>/dev/null || true
	@echo "$(GREEN)✓ Cleaned$(RESET)"

build: ## Build API and UI
	@$(MAKE) build-api
	@$(MAKE) build-ui

build-api: ## Build Go API
	@if [ -f api/go.mod ]; then \
		echo "$(BLUE)🏗️  Building API...$(RESET)"; \
		cd api && go build -o secrets-manager-api .; \
	else \
		echo "$(YELLOW)No Go API present$(RESET)"; \
	fi

build-ui: ## Build Vite UI
	@if [ -f ui/package.json ]; then \
		echo "$(BLUE)🏗️  Building UI...$(RESET)"; \
		cd ui && pnpm run build; \
	else \
		echo "$(YELLOW)No UI present$(RESET)"; \
	fi

db-reset: ## Drop and recreate scenario schema
	@if [ -f initialization/storage/postgres/schema.sql ]; then \
		vrooli scenario db-reset $(SCENARIO_NAME); \
	else \
		echo "$(YELLOW)No database artifacts defined$(RESET)"; \
	fi

db-migrate: ## Run migrations (if any)
	@if [ -d initialization/storage/postgres/migrations ]; then \
		vrooli scenario db-migrate $(SCENARIO_NAME); \
	else \
		echo "$(YELLOW)No migrations directory$(RESET)"; \
	fi

validate: ## Validate scenario structure
	@vrooli scenario validate $(SCENARIO_NAME)

install-deps: ## Install Go + pnpm deps
	@if [ -f api/go.mod ]; then cd api && go mod download; fi
	@if [ -f ui/package.json ]; then cd ui && pnpm install; fi

fmt: ## Format Go + UI code
	@$(MAKE) fmt-go
	@$(MAKE) fmt-ui

fmt-go: ## Format Go files
	@if [ -d api ]; then \
		if command -v gofumpt >/dev/null 2>&1; then cd api && gofumpt -w .; \
		elif command -v gofmt >/dev/null 2>&1; then cd api && gofmt -w .; fi; \
	fi

fmt-ui: ## Format TS/JS/CSS
	@if [ -d ui ]; then \
		if command -v pnpm >/dev/null 2>&1; then cd ui && pnpm dlx prettier --write "src/**/*.{ts,tsx,js,jsx,css}"; \
		elif command -v prettier >/dev/null 2>&1; then cd ui && prettier --write "src/**/*.{ts,tsx,js,jsx,css}"; fi; \
	fi

lint: ## Lint Go + UI code
	@$(MAKE) lint-go
	@$(MAKE) lint-ui

lint-go: ## Run golangci-lint if available
	@if [ -d api ]; then \
		if command -v golangci-lint >/dev/null 2>&1; then cd api && golangci-lint run; \
		else cd api && go vet ./...; fi; \
	fi

lint-ui: ## Run eslint via pnpm
	@if [ -f ui/package.json ]; then \
		if [ -f ui/node_modules/.bin/eslint ]; then cd ui && pnpm run lint || true; fi; \
	fi

check: ## Format + lint + test
	@$(MAKE) fmt
	@$(MAKE) lint
	@$(MAKE) test

r: run
s: stop
d: dev
t: test
l: logs
c: clean
b: build
f: fmt
