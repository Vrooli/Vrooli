# Tidiness Manager Scenario Makefile
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

.PHONY: help start stop dev status test logs clean build fmt fmt-go fmt-ui lint lint-go lint-ui type check r s t l c b f

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
	@echo "$(BLUE)Scenario Commands$(RESET)"
	@echo ""
	@echo "$(YELLOW)Usage:$(RESET)"
	@echo "  make <command>"
	@echo ""
	@echo "$(YELLOW)Commands:$(RESET)"
	@grep -E '^[a-zA-Z0-9_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  $(GREEN)%-15s$(RESET) %s\n", $$1, $$2}'
	@echo ""
	@echo "$(RED)⚠️  IMPORTANT:$(RESET)"
	@echo "    Never run ./api/tidiness-manager-api or ./ui/server directly!"
	@echo "    Always use 'make start' or 'vrooli scenario start $(SCENARIO_NAME)'"

start: ## Start this scenario (uses Vrooli lifecycle)
	@echo "$(BLUE)🚀 Starting $(SCENARIO_NAME) scenario...$(RESET)"
	@vrooli scenario start $(SCENARIO_NAME)

stop: ## Stop this scenario
	@echo "$(YELLOW)⏹️  Stopping $(SCENARIO_NAME) scenario...$(RESET)"
	@vrooli scenario stop $(SCENARIO_NAME)

dev: ## Start in development mode
	@echo "$(BLUE)🔧 Starting $(SCENARIO_NAME) in development mode...$(RESET)"
	@vrooli scenario start $(SCENARIO_NAME)

status: ## Show scenario status
	@echo "$(BLUE)📊 Status of $(SCENARIO_NAME):$(RESET)"
	@vrooli scenario status $(SCENARIO_NAME)

test: ## Run all tests
	@echo "$(BLUE)🧪 Testing $(SCENARIO_NAME) scenario...$(RESET)"
	@vrooli scenario test $(SCENARIO_NAME)

logs: ## Show recent logs for this scenario
	@echo "$(BLUE)📜 Logs for $(SCENARIO_NAME):$(RESET)"
	@vrooli scenario logs $(SCENARIO_NAME) --tail 50

logs-follow: ## Follow logs in real-time
	@echo "$(BLUE)📜 Following logs for $(SCENARIO_NAME):$(RESET)"
	@vrooli scenario logs $(SCENARIO_NAME) --follow

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
		cd api && go build -o $(SCENARIO_NAME)-api .; \
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
		if [ -f ui/package.json ] && grep -q '"lint"' ui/package.json; then \
			cd ui && pnpm run lint 2>/dev/null || true; \
		fi; \
		echo "$(GREEN)✓ UI code linted$(RESET)"; \
	else \
		echo "$(YELLOW)No UI code to lint$(RESET)"; \
	fi

type: ## Run type checking
	@echo "$(BLUE)🔍 Type checking all code...$(RESET)"
	@if [ -d api ] && find api -name "*.go" | head -1 | grep -q .; then \
		echo "$(BLUE)Type checking Go code...$(RESET)"; \
		cd api && go vet ./... 2>&1; \
	fi
	@if [ -d ui ] && [ -f ui/package.json ] && grep -q '"type-check"' ui/package.json; then \
		echo "$(BLUE)Type checking UI code...$(RESET)"; \
		cd ui && pnpm run type-check 2>&1 || true; \
	fi

check: ## Format, lint, and test code (pre-commit workflow)
	@echo "$(BLUE)✅ Running full code quality check...$(RESET)"
	@$(MAKE) fmt
	@$(MAKE) lint
	@$(MAKE) test

# Development shortcuts
r: start
s: stop
t: test
l: logs
c: clean
b: build
f: fmt
