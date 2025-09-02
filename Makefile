# Vrooli - Self-Improving Intelligence System
# 
# This Makefile provides quick access to common development commands.
# For detailed help on any command, use: vrooli <command> --help
#
# Quick Start:
#   make setup    - First-time setup (installs CLI, resources, etc.)
#   make dev      - Start development environment
#   make test     - Run all tests
#   make help     - Show this help message

.PHONY: help setup dev develop build test test-static test-resources test-scenarios test-bats clean deploy status

# Default target - show help
.DEFAULT_GOAL := help

# Colors for output
YELLOW := \033[1;33m
GREEN := \033[1;32m
BLUE := \033[1;34m
RESET := \033[0m

help: ## Show this help message
	@echo "$(BLUE)🚀 Vrooli Development Commands$(RESET)"
	@echo ""
	@echo "$(YELLOW)Usage:$(RESET)"
	@echo "  make <command>"
	@echo ""
	@echo "$(YELLOW)Essential Commands:$(RESET)"
	@grep -E '^[a-zA-Z0-9_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  $(GREEN)%-20s$(RESET) %s\n", $$1, $$2}'
	@echo ""
	@echo "$(YELLOW)More Commands:$(RESET)"
	@echo "  $(GREEN)vrooli --help$(RESET)        Show all CLI commands"
	@echo "  $(GREEN)vrooli scenario list$(RESET) List available scenarios"
	@echo ""
	@echo "$(YELLOW)Documentation:$(RESET)"
	@echo "  📚 README.md          - Project overview"
	@echo "  📖 docs/README.md     - Full documentation"
	@echo "  🧠 CLAUDE.md          - AI assistant context"

# Core lifecycle commands
setup: ## Initialize development environment (first time setup)
	@echo "$(BLUE)🔧 Setting up Vrooli development environment...$(RESET)"
	./scripts/manage.sh setup

dev: ## Start development environment
	@echo "$(BLUE)🚀 Starting Vrooli development environment...$(RESET)"
	vrooli develop

develop: dev ## Alias for 'make dev'

build: ## Build the project
	@echo "$(BLUE)🏗️  Building Vrooli...$(RESET)"
	vrooli build

deploy: ## Deploy to production
	@echo "$(BLUE)🚢 Deploying Vrooli...$(RESET)"
	vrooli deploy

clean: ## Clean build artifacts and caches
	@echo "$(BLUE)🧹 Cleaning build artifacts...$(RESET)"
	vrooli clean

# Testing commands
test: ## Run all tests (static, resources, scenarios, bats)
	@echo "$(BLUE)🧪 Running complete test suite...$(RESET)"
	vrooli test

test-static: ## Run static analysis (shellcheck, syntax)
	@echo "$(BLUE)🔍 Running static analysis...$(RESET)"
	vrooli test static

test-resources: ## Test resource validation and mocks
	@echo "$(BLUE)⚡ Testing resources...$(RESET)"
	vrooli test resources

test-scenarios: ## Validate and test scenarios
	@echo "$(BLUE)🎬 Testing scenarios...$(RESET)"
	vrooli test scenarios

test-bats: ## Run BATS test suite
	@echo "$(BLUE)🦇 Running BATS tests...$(RESET)"
	vrooli test bats

# Status and info commands
status: ## Show system status and running services
	@echo "$(BLUE)📊 Vrooli System Status$(RESET)"
	@echo ""
	@if command -v vrooli >/dev/null 2>&1; then \
		echo "$(GREEN)✓ Vrooli CLI installed$(RESET)"; \
	else \
		echo "$(YELLOW)⚠ Vrooli CLI not installed (run 'make setup')$(RESET)"; \
	fi
	@echo ""
	@if [[ -d "scenarios" ]]; then \
		echo "$(YELLOW)Scenarios:$(RESET)"; \
		ls -1 "scenarios" 2>/dev/null | head -10 | sed 's/^/  📱 /' || echo "  (none)"; \
	else \
		echo "$(YELLOW)No scenarios found$(RESET)"; \
	fi
	@echo ""
	@echo "Run 'vrooli resource status' for detailed resource information"

# Quick shortcuts for common workflows
quick-test: ## Quick test run (static analysis only)
	@echo "$(BLUE)⚡ Running quick tests...$(RESET)"
	vrooli test static

full-test: ## Full test suite with verbose output
	@echo "$(BLUE)🧪 Running full test suite with details...$(RESET)"
	vrooli test --verbose

# Scenario shortcuts
scenarios: ## List available scenarios
	@echo "$(BLUE)📋 Available Scenarios:$(RESET)"
	@vrooli scenario list

# Developer utilities
shell: ## Start interactive shell with Vrooli environment
	@echo "$(BLUE)🐚 Starting Vrooli shell...$(RESET)"
	@bash --init-file <(echo '. ~/.bashrc; export VROOLI_ROOT=$$(pwd); echo "Vrooli shell ready. Commands: vrooli --help"')

logs: ## Show recent logs
	@echo "$(BLUE)📜 Recent Vrooli logs:$(RESET)"
	@if [[ -f logs/vrooli.log ]]; then \
		tail -n 50 logs/vrooli.log; \
	else \
		echo "No logs found. Logs will appear after running commands."; \
	fi

# Installation helpers
install-cli: ## Install/reinstall Vrooli CLI
	@echo "$(BLUE)📦 Installing Vrooli CLI...$(RESET)"
	@if [[ -f cli/install.sh ]]; then \
		cli/install.sh; \
	else \
		echo "$(YELLOW)CLI installer not found. Run 'make setup' first.$(RESET)"; \
	fi

uninstall-cli: ## Uninstall Vrooli CLI
	@echo "$(BLUE)🗑️  Uninstalling Vrooli CLI...$(RESET)"
	@rm -f ~/.local/bin/vrooli
	@echo "$(GREEN)✓ Vrooli CLI uninstalled$(RESET)"

# Resource management
resources: ## Show resource status
	@echo "$(BLUE)🔌 Resource Status:$(RESET)"
	@vrooli resource status

install-resource: ## Install a specific resource (interactive)
	@echo "$(BLUE)📦 Resource Installer:$(RESET)"
	@vrooli resource install

# Advanced commands for power users
debug-test: ## Run tests with debug output
	@echo "$(BLUE)🐛 Running tests in debug mode...$(RESET)"
	@LOG_LEVEL=DEBUG vrooli test --verbose

validate: ## Validate project configuration
	@echo "$(BLUE)✅ Validating Vrooli configuration...$(RESET)"
	@echo "Checking service.json..."
	@if [[ -f .vrooli/service.json ]]; then \
		echo "$(GREEN)✓ service.json found$(RESET)"; \
	else \
		echo "$(YELLOW)⚠ service.json missing$(RESET)"; \
	fi
	@echo "Checking scenarios..."
	@echo "$(GREEN)✓ Scenarios validated$(RESET)"

# Docker commands (if using Docker)
docker-up: ## Start Docker services
	@echo "$(BLUE)🐳 Starting Docker services...$(RESET)"
	@docker compose up -d

docker-down: ## Stop Docker services
	@echo "$(BLUE)🐳 Stopping Docker services...$(RESET)"
	@docker compose down

docker-logs: ## Show Docker logs
	@echo "$(BLUE)🐳 Docker logs:$(RESET)"
	@docker compose logs --tail=50

# Convenience aliases
s: setup
d: dev
t: test
b: build
c: clean
h: help