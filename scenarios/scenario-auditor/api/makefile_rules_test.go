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

	t.Run("valid canonical makefile", func(t *testing.T) {
		violations, err := rule.Check(valid, "scenarios/demo/Makefile", "demo")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(violations) != 0 {
			t.Fatalf("expected no violations, got %d: %+v", len(violations), violations)
		}
	})

	t.Run("invalid default goal", func(t *testing.T) {
		invalid := strings.Replace(valid, ".DEFAULT_GOAL := help", ".DEFAULT_GOAL := run", 1)
		violations, err := rule.Check(invalid, "scenarios/demo/Makefile", "demo")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !violationsContain(violations, ".DEFAULT_GOAL must be set to 'help'") {
			t.Fatalf("expected violation about default goal, got: %+v", violations)
		}
	})

	t.Run("requires phony block before other directives", func(t *testing.T) {
		needle := `.PHONY: help start stop test logs status clean build dev restart rebuild fmt fmt-go fmt-ui lint lint-go lint-ui check`
		reordered := strings.Replace(valid,
			needle+"\n\n# Default target - show help\n.DEFAULT_GOAL := help\n",
			".DEFAULT_GOAL := help\n\n"+needle+"\n\n# Default target - show help\n",
			1,
		)
		violations, err := rule.Check(reordered, "scenarios/demo/Makefile", "demo")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !violationsContain(violations, "First directive after header must be .PHONY target declaration") {
			t.Fatalf("expected .PHONY ordering violation, got: %+v", violations)
		}
	})

	t.Run("help target must be first", func(t *testing.T) {
		withEarly := strings.Replace(valid,
			"RESET := \\033[0m\n\nhelp:",
			"RESET := \\033[0m\n\nearly:\n\t@echo \"early\"\n\nhelp:",
			1,
		)
		violations, err := rule.Check(withEarly, "scenarios/demo/Makefile", "demo")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !violationsContain(violations, "help target must be defined before any other targets") {
			t.Fatalf("expected help ordering violation, got: %+v", violations)
		}
	})

	t.Run("shortcut section must terminate file", func(t *testing.T) {
		invalid := valid + "\n.PHONY: make-executable\nmake-executable:\n\t@echo \"fix\"\n"
		violations, err := rule.Check(invalid, "scenarios/demo/Makefile", "demo")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !violationsContain(violations, "Shortcut targets must be the final content in the Makefile") {
			t.Fatalf("expected shortcut terminal violation, got: %+v", violations)
		}
	})
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

func violationsContain(list []Violation, needle string) bool {
	for _, v := range list {
		if strings.Contains(v.Message, needle) || strings.Contains(v.Description, needle) {
			return true
		}
	}
	return false
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

.PHONY: help start stop test logs status clean build dev restart rebuild fmt fmt-go fmt-ui lint lint-go lint-ui check

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

# Development shortcuts
dev: start

restart: stop start

rebuild: clean build
`
}
