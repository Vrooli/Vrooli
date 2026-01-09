# Scenario To Cloud Scenario Makefile
#
# This Makefile ensures scenarios are always run through the Vrooli lifecycle system.
# NEVER run scenarios directly (./api/scenario-to-cloud-api). ALWAYS use these commands.
#
# Usage:
#   make       - Show help
#   make start - Start this scenario
#   make stop  - Stop this scenario
#   make test  - Run scenario tests
#   make logs  - Show scenario logs
#   make clean - Clean build artifacts

.PHONY: help start stop test logs status clean build dev fmt fmt-go fmt-ui lint lint-go lint-ui

.DEFAULT_GOAL := help

SCENARIO_NAME := $(notdir $(CURDIR))

GREEN := \033[1;32m
YELLOW := \033[1;33m
BLUE := \033[1;34m
RED := \033[1;31m
RESET := \033[0m

help: ## Show this help message
	@echo "$(BLUE)☁️  $(SCENARIO_NAME) Scenario Commands$(RESET)"
	@echo ""
	@echo "$(YELLOW)Usage:$(RESET)"
	@echo "  make <command>"
	@echo ""
	@echo "$(YELLOW)Commands:$(RESET)"
	@grep -E '^[a-zA-Z0-9_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  $(GREEN)%-12s$(RESET) %s\n", $$1, $$2}'
	@echo ""
	@echo "$(RED)⚠️  IMPORTANT:$(RESET) Never run ./api/scenario-to-cloud-api directly!"
	@echo "    Always use 'make start' or 'vrooli scenario start $(SCENARIO_NAME)'"

start: ## Start this scenario
	@echo "$(BLUE)🚀 Starting $(SCENARIO_NAME) scenario...$(RESET)"
	@vrooli scenario start $(SCENARIO_NAME)

stop: ## Stop this scenario
	@echo "$(YELLOW)⏹️  Stopping $(SCENARIO_NAME) scenario...$(RESET)"
	@vrooli scenario stop $(SCENARIO_NAME)

dev: ## Start development mode
	@echo "$(BLUE)🔧 Starting $(SCENARIO_NAME) in development mode...$(RESET)"
	@vrooli scenario run $(SCENARIO_NAME) --dev

test: ## Run scenario tests
	@echo "$(BLUE)🧪 Testing $(SCENARIO_NAME) scenario...$(RESET)"
	@vrooli scenario test $(SCENARIO_NAME)

logs: ## Show scenario logs
	@echo "$(BLUE)📜 Logs for $(SCENARIO_NAME):$(RESET)"
	@vrooli scenario logs $(SCENARIO_NAME) --tail 50

status: ## Show scenario status
	@echo "$(BLUE)📊 Status of $(SCENARIO_NAME):$(RESET)"
	@vrooli scenario status $(SCENARIO_NAME)

clean: ## Clean build artifacts
	@echo "$(YELLOW)🧹 Cleaning $(SCENARIO_NAME) artifacts...$(RESET)"
	@rm -f api/scenario-to-cloud-api
	@rm -rf ui/dist
	@echo "$(GREEN)✓ Cleaned$(RESET)"

build: ## Build the scenario
	@echo "$(BLUE)🏗️  Building $(SCENARIO_NAME)...$(RESET)"
	@vrooli scenario setup $(SCENARIO_NAME)

fmt: fmt-go fmt-ui ## Format all code

fmt-go: ## Format Go code
	@if [ -d api ] && find api -name "*.go" | head -1 | grep -q .; then \
		echo "$(BLUE)🎨 Formatting Go code...$(RESET)"; \
		if command -v gofumpt >/dev/null 2>&1; then \
			cd api && gofumpt -w .; \
		elif command -v gofmt >/dev/null 2>&1; then \
			cd api && gofmt -w .; \
		fi; \
		echo "$(GREEN)✓ Go code formatted$(RESET)"; \
	fi

fmt-ui: ## Format UI code
	@if command -v pnpm >/dev/null 2>&1; then \
		( cd ui && pnpm -s format || true ); \
	fi

lint: lint-go lint-ui ## Lint all code

lint-go: ## Lint Go code
	@if [ -d api ] && find api -name "*.go" | head -1 | grep -q .; then \
		echo "$(BLUE)🔍 Linting Go code...$(RESET)"; \
		if command -v golangci-lint >/dev/null 2>&1; then \
			cd api && golangci-lint run ./...; \
		else \
			cd api && go vet ./...; \
		fi; \
		echo "$(GREEN)✓ Go code linted$(RESET)"; \
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
r: run
s: stop
d: dev
t: test
l: logs
c: clean
b: build
f: fmt
