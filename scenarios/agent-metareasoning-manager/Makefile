# Agent Metareasoning Manager
# 
# This Makefile ensures the scenario is always run through the Vrooli lifecycle system.
# NEVER run scenarios directly (./api/agent-metareasoning-manager). ALWAYS use these commands.
#
# Usage:
#   make           - Show help
#   make run       - Start metareasoning manager
#   make stop      - Stop metareasoning manager
#   make dev       - Start in development mode with hot reload
#   make test      - Run all tests
#   make coverage  - Run tests with coverage report
#   make logs      - Show scenario logs
#   make clean     - Clean build artifacts

.PHONY: help run stop dev test test-verbose test-short test-integration coverage coverage-html logs logs-follow status clean build lint fmt deps benchmark profile check fmt-go fmt-ui lint-go lint-ui

# Default target - show help
.DEFAULT_GOAL := help

# Get scenario name
SCENARIO_NAME := agent-metareasoning-manager
BINARY_NAME := agent-metareasoning-manager-api

# Test configuration
COVERAGE_TARGET := 90
POSTGRES_PORT ?= 5432
API_PORT ?= 8090
TEST_DB_URL := postgres://postgres:postgres@localhost:$(POSTGRES_PORT)/metareasoning_test?sslmode=disable

# Colors for output
GREEN := \033[1;32m
YELLOW := \033[1;33m
BLUE := \033[1;34m
RED := \033[1;31m
CYAN := \033[1;36m
RESET := \033[0m

help: ## Show this help message
	@echo "$(BLUE)🧠 Agent Metareasoning Manager Commands$(RESET)"
	@echo ""
	@echo "$(YELLOW)Usage:$(RESET)"
	@echo "  make <command>"
	@echo ""
	@echo "$(YELLOW)Lifecycle Commands:$(RESET)"
	@grep -E '^(run|stop|dev|logs|status):.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  $(GREEN)%-15s$(RESET) %s\n", $$1, $$2}'
	@echo ""
	@echo "$(YELLOW)Development Commands:$(RESET)"
	@grep -E '^(test|coverage|lint|fmt|build|clean):.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  $(GREEN)%-15s$(RESET) %s\n", $$1, $$2}'
	@echo ""
	@echo "$(YELLOW)Additional Commands:$(RESET)"
	@grep -E '^[a-zA-Z0-9_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		grep -v -E '^(run|stop|dev|logs|status|test|coverage|lint|fmt|build|clean|help):' | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  $(GREEN)%-15s$(RESET) %s\n", $$1, $$2}'
	@echo ""
	@echo "$(RED)⚠️  IMPORTANT:$(RESET) Never run ./api/$(BINARY_NAME) directly!"
	@echo "    Always use 'make run' or 'vrooli scenario run $(SCENARIO_NAME)'"

# ============================================================================
# LIFECYCLE COMMANDS - Use Vrooli lifecycle system
# ============================================================================

run: ## Start metareasoning manager (via Vrooli lifecycle)
	@echo "$(BLUE)🚀 Starting Agent Metareasoning Manager...$(RESET)"
	@echo "$(CYAN)Coordinating meta-cognitive reasoning processes$(RESET)"
	@vrooli scenario run $(SCENARIO_NAME)

stop: ## Stop metareasoning manager
	@echo "$(YELLOW)⏹️  Stopping Agent Metareasoning Manager...$(RESET)"
	@vrooli scenario stop $(SCENARIO_NAME)

dev: ## Start in development mode with hot reload
	@echo "$(BLUE)🔧 Starting in development mode...$(RESET)"
	@vrooli scenario run $(SCENARIO_NAME) --dev

logs: ## Show recent logs
	@echo "$(BLUE)📜 Recent Metareasoning Manager logs:$(RESET)"
	@vrooli scenario logs $(SCENARIO_NAME) --tail 50

logs-follow: ## Follow logs in real-time
	@echo "$(BLUE)📜 Following Metareasoning Manager logs...$(RESET)"
	@vrooli scenario logs $(SCENARIO_NAME) --follow

status: ## Check if scenario is running
	@echo "$(BLUE)📊 Metareasoning Manager Status:$(RESET)"
	@vrooli scenario status $(SCENARIO_NAME)

# ============================================================================
# DEVELOPMENT COMMANDS - For testing and building
# ============================================================================

test: ## Run tests for this scenario
	@echo "$(BLUE)🧪 Testing $(SCENARIO_NAME) scenario...$(RESET)"
	@vrooli scenario test $(SCENARIO_NAME)

test-verbose: ## Run tests with verbose output
	@echo "$(BLUE)🧪 Running tests (verbose)...$(RESET)"
	@cd api && TEST_DATABASE_URL=$(TEST_DB_URL) go test -v -race -count=1 ./...

test-short: ## Run short tests only
	@echo "$(BLUE)🧪 Running short tests...$(RESET)"
	@cd api && TEST_DATABASE_URL=$(TEST_DB_URL) go test -short ./...

test-integration: ## Run integration tests (requires running services)
	@echo "$(BLUE)🧪 Running integration tests...$(RESET)"
	@cd api && TEST_DATABASE_URL=$(TEST_DB_URL) go test -v -tags=integration ./...

coverage: ## Run tests with coverage report
	@echo "$(BLUE)🧪 Running tests with coverage...$(RESET)"
	@cd api && TEST_DATABASE_URL=$(TEST_DB_URL) go test -v -race -coverprofile=coverage.out -covermode=atomic ./...
	@echo "\n$(CYAN)Coverage Report:$(RESET)"
	@cd api && go tool cover -func=coverage.out
	@echo "\nGenerating HTML report..."
	@cd api && go tool cover -html=coverage.out -o coverage.html
	@echo "$(GREEN)✓ HTML report saved to api/coverage.html$(RESET)"
	@echo "\nChecking coverage target ($(COVERAGE_TARGET)%)..."
	@coverage=$$(cd api && go tool cover -func=coverage.out | grep total | awk '{print $$3}' | sed 's/%//'); \
	if [ $$(echo "$$coverage >= $(COVERAGE_TARGET)" | bc 2>/dev/null || echo 0) -eq 1 ]; then \
		echo "$(GREEN)✓ Coverage target met: $$coverage%$(RESET)"; \
	else \
		echo "$(YELLOW)⚠ Coverage below target: $$coverage% < $(COVERAGE_TARGET)%$(RESET)"; \
	fi

coverage-html: coverage ## Open coverage report in browser
	@echo "$(BLUE)Opening coverage report...$(RESET)"
	@if command -v xdg-open > /dev/null; then \
		xdg-open api/coverage.html; \
	elif command -v open > /dev/null; then \
		open api/coverage.html; \
	else \
		echo "Please open api/coverage.html manually"; \
	fi

lint: ## Lint all code
	@echo "$(BLUE)🔍 Linting code...$(RESET)" 
	@$(MAKE) lint-go
	@$(MAKE) lint-ui

fmt: ## Format all code
	@echo "$(BLUE)🎨 Formatting code...$(RESET)"
	@$(MAKE) fmt-go
	@$(MAKE) fmt-ui

deps: ## Install/update dependencies
	@echo "$(BLUE)📦 Installing dependencies...$(RESET)"
	@cd api && go mod download
	@cd api && go mod tidy
	@echo "$(GREEN)✓ Dependencies installed$(RESET)"

build: ## Build the API binary
	@echo "$(BLUE)🏗️  Building $(BINARY_NAME)...$(RESET)"
	@cd api && go build -o $(BINARY_NAME) main.go
	@echo "$(GREEN)✓ Built api/$(BINARY_NAME)$(RESET)"

clean: ## Clean build artifacts
	@echo "$(YELLOW)🧹 Cleaning build artifacts...$(RESET)"
	@rm -f api/$(BINARY_NAME) api/coverage.out api/coverage.html api/*.prof
	@rm -rf api/build/ api/dist/ api/*.log
	@echo "$(GREEN)✓ Cleaned$(RESET)"

# ============================================================================
# PERFORMANCE & PROFILING
# ============================================================================

benchmark: ## Run benchmarks
	@echo "$(BLUE)⚡ Running benchmarks...$(RESET)"
	@cd api && go test -bench=. -benchmem ./...

benchmark-performance: ## Run performance benchmarks
	@echo "$(BLUE)⚡ Running performance benchmarks...$(RESET)"
	@cd api && go test -bench=BenchmarkHealthHandler -benchtime=10s
	@cd api && go test -bench=BenchmarkWriteJSON -benchtime=5s
	@cd api && go test -bench=BenchmarkPerformanceMiddleware -benchtime=5s

profile: ## Generate CPU and memory profiles
	@echo "$(BLUE)📊 Generating profiles...$(RESET)"
	@cd api && go test -cpuprofile=cpu.prof -memprofile=mem.prof -bench=. ./...
	@echo "$(GREEN)✓ Profiles generated: api/cpu.prof, api/mem.prof$(RESET)"
	@echo "View with: go tool pprof api/cpu.prof"

performance-test: ## Run performance monitoring
	@echo "$(BLUE)📊 Running performance tests...$(RESET)"
	@if [ -f api/monitor_performance.sh ]; then \
		cd api && ./monitor_performance.sh; \
	else \
		echo "$(YELLOW)Performance monitor script not found$(RESET)"; \
	fi

# ============================================================================
# DATABASE HELPERS
# ============================================================================

setup-test-db: ## Create test database
	@echo "$(BLUE)🗄️  Setting up test database...$(RESET)"
	@psql -U postgres -p $(POSTGRES_PORT) -c "CREATE DATABASE metareasoning_test;" 2>/dev/null || true
	@echo "$(GREEN)✓ Test database ready$(RESET)"

teardown-test-db: ## Drop test database
	@echo "$(YELLOW)🗑️  Dropping test database...$(RESET)"
	@psql -U postgres -p $(POSTGRES_PORT) -c "DROP DATABASE IF EXISTS metareasoning_test;"
	@echo "$(GREEN)✓ Test database dropped$(RESET)"

# ============================================================================
# DOCUMENTATION & CI
# ============================================================================

docs: ## Generate API documentation
	@echo "$(BLUE)📚 Generating documentation...$(RESET)"
	@if command -v godoc > /dev/null; then \
		echo "Documentation server starting at http://localhost:6060"; \
		cd api && godoc -http=:6060; \
	else \
		echo "$(YELLOW)Please install godoc: go install golang.org/x/tools/cmd/godoc@latest$(RESET)"; \
	fi

check: ## Format, lint, and test code (pre-commit workflow)
	@echo "$(BLUE)✅ Running full code quality check...$(RESET)"
	@$(MAKE) fmt
	@$(MAKE) lint
	@$(MAKE) test
	@echo "$(GREEN)✓ All checks passed!$(RESET)"

ci: deps lint test coverage ## Run CI pipeline
	@echo "$(GREEN)✓ CI pipeline complete$(RESET)"

validate: ## Validate scenario structure
	@echo "$(BLUE)✅ Validating $(SCENARIO_NAME) structure...$(RESET)"
	@vrooli scenario validate $(SCENARIO_NAME)

# ============================================================================
# DOCKER COMMANDS
# ============================================================================

docker-build: ## Build Docker image
	@echo "$(BLUE)🐳 Building Docker image...$(RESET)"
	@cd api && docker build -t $(BINARY_NAME):latest .

docker-run: ## Run API in Docker container
	@echo "$(BLUE)🐳 Running in Docker...$(RESET)"
	@docker run -p $(API_PORT):$(API_PORT) --env-file api/.env $(BINARY_NAME):latest

# ============================================================================
# LOAD TESTING
# ============================================================================

load-test: ## Run load tests (requires vegeta)
	@if command -v vegeta > /dev/null; then \
		echo "$(BLUE)💥 Running load tests...$(RESET)"; \
		echo "GET http://localhost:$(API_PORT)/health" | vegeta attack -duration=30s -rate=100 | vegeta report; \
	else \
		echo "$(YELLOW)Please install vegeta: go install github.com/tsenart/vegeta@latest$(RESET)"; \
	fi

# ============================================================================
# Code Quality Targets
# ============================================================================

fmt-go: ## Format Go code  
	@if [ -d api ] && find api -name "*.go" | head -1 | grep -q .; then \
		if command -v gofumpt >/dev/null 2>&1; then \
			cd api && gofumpt -w .; \
		elif command -v gofmt >/dev/null 2>&1; then \
			cd api && gofmt -w .; \
		fi; \
	fi

fmt-ui: ## Format TypeScript/JavaScript code
	@if [ -d ui ] && (find ui -name "*.ts" -o -name "*.tsx" -o -name "*.js" -o -name "*.jsx") | head -1 | grep -q .; then \
		if command -v prettier >/dev/null 2>&1; then \
			cd ui && prettier --write "**/*.{ts,tsx,js,jsx,json,css,md}" 2>/dev/null || true; \
		fi; \
	fi

lint-go: ## Lint Go code
	@if [ -d api ] && find api -name "*.go" | head -1 | grep -q .; then \
		if command -v golangci-lint >/dev/null 2>&1; then \
			cd api && golangci-lint run; \
		fi; \
	fi

lint-ui: ## Lint TypeScript/JavaScript code
	@if [ -d ui ] && (find ui -name "*.ts" -o -name "*.tsx" -o -name "*.js" -o -name "*.jsx") | head -1 | grep -q .; then \
		if [ -f ui/package.json ] && grep -q '"eslint"' ui/package.json; then \
			cd ui && npm run lint 2>/dev/null || true; \
		fi; \
	fi

# ============================================================================
# DEVELOPMENT SHORTCUTS
# ============================================================================

r: run
s: stop
d: dev
t: test
l: logs
c: clean
b: build