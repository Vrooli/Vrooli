# Scenario Auditor Makefile
# Comprehensive standards enforcement system for Vrooli scenarios

.PHONY: help install build test clean run stop logs health setup

# Default target
help:
	@echo "Scenario Auditor - Standards Enforcement System"
	@echo ""
	@echo "Available commands:"
	@echo "  install     Install all dependencies"
	@echo "  build       Build API and CLI binaries"
	@echo "  test        Run comprehensive test suite"
	@echo "  run         Start the scenario in development mode"
	@echo "  stop        Stop all running processes"
	@echo "  clean       Clean build artifacts"
	@echo "  health      Check system health"
	@echo "  setup       Full setup (install + build)"
	@echo ""
	@echo "Testing commands:"
	@echo "  test-unit         Run unit tests only"
	@echo "  test-integration  Run integration tests only"
	@echo "  test-structure    Run structure validation"
	@echo "  test-deps         Check dependencies"
	@echo "  test-business     Run business logic tests"
	@echo "  test-performance  Run performance tests"
	@echo ""

# Installation
install:
	@echo "📦 Installing dependencies..."
	@cd api && go mod download
	@cd cli && go mod download
	@if [ -f ui/package.json ] && command -v node >/dev/null 2>&1; then \
		cd ui && npm install --prefer-offline --no-audit --no-fund --loglevel error; \
	fi
	@echo "✅ Dependencies installed"

# Build
build:
	@echo "🔨 Building binaries..."
	@cd api && go build -o scenario-auditor-api .
	@echo "✅ Build complete"

# Install CLI globally
install-cli:
	@echo "📦 Installing CLI globally..."
	@cd cli && ./install.sh

# Setup (install + build)
setup: install build install-cli
	@echo "✅ Setup complete"

# Testing
test:
	@echo "🧪 Running full test suite..."
	@chmod +x test/run-tests.sh
	@./test/run-tests.sh

test-unit:
	@echo "🧪 Running unit tests..."
	@chmod +x test/phases/test-unit.sh
	@./test/phases/test-unit.sh

test-integration:
	@echo "🧪 Running integration tests..."
	@chmod +x test/phases/test-integration.sh
	@./test/phases/test-integration.sh

test-structure:
	@echo "🧪 Running structure tests..."
	@chmod +x test/phases/test-structure.sh
	@./test/phases/test-structure.sh

test-deps:
	@echo "🧪 Checking dependencies..."
	@chmod +x test/phases/test-dependencies.sh
	@./test/phases/test-dependencies.sh

test-business:
	@echo "🧪 Running business logic tests..."
	@chmod +x test/phases/test-business.sh
	@./test/phases/test-business.sh

test-performance:
	@echo "🧪 Running performance tests..."
	@chmod +x test/phases/test-performance.sh
	@./test/phases/test-performance.sh

# Development
run:
	@echo "🚀 Starting Scenario Auditor..."
	@if [ ! -f api/scenario-auditor-api ]; then make build; fi
	@export API_PORT=$${API_PORT:-15001} && \
	export UI_PORT=$${UI_PORT:-35001} && \
	echo "Starting API on port $$API_PORT..." && \
	cd api && VROOLI_LIFECYCLE_MANAGED=true ./scenario-auditor-api & \
	echo "$$!" > ../api.pid && \
	sleep 3 && \
	if [ -f ../ui/package.json ] && command -v node >/dev/null 2>&1; then \
		echo "Starting UI on port $$UI_PORT..." && \
		cd ../ui && npm run dev & \
		echo "$$!" > ../ui.pid; \
	fi && \
	echo "✅ Scenario Auditor is running:" && \
	echo "  Dashboard: http://localhost:$$UI_PORT" && \
	echo "  API: http://localhost:$$API_PORT/api/v1" && \
	echo "  Health: http://localhost:$$API_PORT/api/v1/health"

stop:
	@echo "🛑 Stopping Scenario Auditor..."
	@vrooli scenario stop scenario-auditor || true
	@echo "✅ Stopped"

logs:
	@echo "📋 Recent logs:"
	@if [ -d test/artifacts ]; then \
		ls -lt test/artifacts/*.log 2>/dev/null | head -5 || echo "No log files found"; \
	else \
		echo "No artifacts directory found"; \
	fi

health:
	@echo "🏥 Checking health..."
	@export API_PORT=$${API_PORT:-15001} && \
	if curl -s "http://localhost:$$API_PORT/api/v1/health" >/dev/null 2>&1; then \
		echo "✅ API is healthy"; \
	else \
		echo "❌ API is not responding"; \
		exit 1; \
	fi

# Clean
clean:
	@echo "🧹 Cleaning build artifacts..."
	@rm -f api/scenario-auditor-api
	@rm -f cli/scenario-auditor
	@rm -f api.pid ui.pid
	@rm -rf ui/dist
	@rm -rf ui/node_modules/.vite
	@find test/artifacts -name "*.log" -type f -delete 2>/dev/null || true
	@echo "✅ Clean complete"

# Utilities
scan-current:
	@echo "🔍 Scanning current scenario..."
	@if [ -f cli/scenario-auditor ]; then \
		cd cli && ./scenario-auditor scan current; \
	else \
		echo "❌ CLI not built. Run 'make build' first."; \
		exit 1; \
	fi

scan-all:
	@echo "🔍 Scanning all scenarios..."
	@if [ -f cli/scenario-auditor ]; then \
		cd cli && ./scenario-auditor scan; \
	else \
		echo "❌ CLI not built. Run 'make build' first."; \
		exit 1; \
	fi

rules:
	@echo "📋 Listing available rules..."
	@if [ -f cli/scenario-auditor ]; then \
		cd cli && ./scenario-auditor rules; \
	else \
		echo "❌ CLI not built. Run 'make build' first."; \
		exit 1; \
	fi

# Development shortcuts
dev: run

restart: stop run

rebuild: clean build

# Make sure test scripts are executable
.PHONY: make-executable
make-executable:
	@chmod +x test/run-tests.sh
	@chmod +x test/phases/*.sh
	@chmod +x cli/install.sh
