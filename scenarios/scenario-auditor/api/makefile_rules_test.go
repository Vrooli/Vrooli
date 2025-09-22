package main

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestMakefileStructureRule(t *testing.T) {
	t.Setenv("VROOLI_ROOT", filepath.Join("..", "..", ".."))
	rule := loadMakefileRule(t, "makefile_structure", "makefile_structure.go", "makefile")

	valid := canonicalMakefile()
	violations, err := rule.Check(valid, "scenarios/demo/Makefile", "demo")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(violations) != 0 {
		t.Fatalf("expected no violations, got %d: %+v", len(violations), violations)
	}

	invalid := strings.Replace(valid, ".DEFAULT_GOAL := help", ".DEFAULT_GOAL := run", 1)
	violations, err = rule.Check(invalid, "scenarios/demo/Makefile", "demo")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(violations) == 0 {
		t.Fatalf("expected violations for incorrect default goal")
	}
}

func TestMakefileLifecycleRule(t *testing.T) {
	t.Setenv("VROOLI_ROOT", filepath.Join("..", "..", ".."))
	rule := loadMakefileRule(t, "makefile_lifecycle", "makefile_lifecycle.go", "makefile")

	valid := canonicalMakefile()
	violations, err := rule.Check(valid, "scenarios/demo/Makefile", "demo")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(violations) != 0 {
		t.Fatalf("expected no violations, got %d: %+v", len(violations), violations)
	}

	invalid := strings.Replace(valid, "@vrooli scenario start $(SCENARIO_NAME)", "@vrooli scenario run $(SCENARIO_NAME)", 1)
	violations, err = rule.Check(invalid, "scenarios/demo/Makefile", "demo")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(violations) == 0 {
		t.Fatalf("expected violations for legacy start command")
	}
}

func TestMakefileQualityRule(t *testing.T) {
	t.Setenv("VROOLI_ROOT", filepath.Join("..", "..", ".."))
	rule := loadMakefileRule(t, "makefile_quality", "makefile_quality.go", "makefile")

	valid := canonicalMakefile()
	violations, err := rule.Check(valid, "scenarios/demo/Makefile", "demo")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(violations) != 0 {
		t.Fatalf("expected no violations, got %d: %+v", len(violations), violations)
	}

	invalid := strings.Replace(valid, "cd api && golangci-lint run", "cd api && golangci-lint --fast", 1)
	violations, err = rule.Check(invalid, "scenarios/demo/Makefile", "demo")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(violations) == 0 {
		t.Fatalf("expected violations for modified lint-go body")
	}
}

func loadMakefileRule(t *testing.T, id, fileName, category string) RuleInfo {
	t.Helper()
	rulePath := filepath.Join("..", "rules", "config", fileName)
	info := RuleInfo{
		ID:       id,
		FilePath: rulePath,
		Category: category,
	}

	exec, status := compileGoRule(&info)
	if !status.Valid {
		t.Fatalf("expected rule %s to compile: %s", id, status.Error)
	}
	info.executor = exec
	return info
}

func canonicalMakefile() string {
	return `# Demo Scenario Makefile
# 
# This Makefile ensures scenarios are always run through the Vrooli lifecycle system.
# NEVER run scenarios directly (./api/demo-api). ALWAYS use these commands.
#
# Usage:
#   make       - Show help
#   make start - Start this scenario
#   make stop  - Stop this scenario
#   make test  - Run scenario tests
#   make logs  - Show scenario logs
#   make clean - Clean build artifacts

.PHONY: help start stop test logs status clean build dev fmt fmt-go fmt-ui lint lint-go lint-ui check

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
	@echo "$(RED)⚠️  IMPORTANT:$(RESET) Never run ./api/$(SCENARIO_NAME)-api directly!"
	@echo "    Always use 'make start' or 'vrooli scenario start $(SCENARIO_NAME)'"

start: ## Start this scenario (uses Vrooli lifecycle)
	@echo "$(BLUE)🚀 Starting $(SCENARIO_NAME) scenario...$(RESET)"
	@vrooli scenario start $(SCENARIO_NAME)

run: ## Legacy support - forwards to start
	@$(MAKE) start

stop: ## Stop this scenario
	@echo "$(YELLOW)⏹️  Stopping $(SCENARIO_NAME) scenario...$(RESET)"
	@vrooli scenario stop $(SCENARIO_NAME)

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
	@rm -rf build/ dist/ *.log api/$(SCENARIO_NAME)-api
	@if [ -d api ]; then cd api && go clean 2>/dev/null || true; fi
	@if [ -d ui/dist ]; then rm -rf ui/dist; fi
	@echo "$(GREEN)✓ Cleaned$(RESET)"

fmt: ## Format all code
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

lint: ## Lint all code
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

check: ## Format, lint, and test code
	@$(MAKE) fmt
	@$(MAKE) lint
	@$(MAKE) test
`
}
