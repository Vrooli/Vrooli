package main

import (
	"strings"
	"testing"
)

func TestCheckMakefileQuality_Canonical(t *testing.T) {
	content := `fmt: fmt-go fmt-ui ## Format all code

fmt-go:
	@if [ -d api ] && find api -name "*.go" | head -1 | grep -q .; then \
		echo "Formatting Go code..."; \
		if command -v gofumpt >/dev/null 2>&1; then \
			cd api && gofumpt -w .; \
		elif command -v gofmt >/dev/null 2>&1; then \
			cd api && gofmt -w .; \
		fi; \
		echo "$(GREEN)✓ Go code formatted$(RESET)"; \
	fi

fmt-ui:
	@echo "Formatting UI assets..."

lint: lint-go lint-ui ## Lint all code

lint-go:
	@if [ -d api ] && find api -name "*.go" | head -1 | grep -q .; then \
		echo "Linting Go code..."; \
		if command -v golangci-lint >/dev/null 2>&1; then \
			cd api && golangci-lint run ./...; \
		else \
			cd api && go vet ./...; \
		fi; \
		echo "$(GREEN)✓ Go code linted$(RESET)"; \
	fi

lint-ui:
	@echo "Linting UI code..."

check: fmt lint test ## Run full quality gates
`
	violations, err := CheckMakefileQuality(content, "Makefile")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(violations) != 0 {
		for _, v := range violations {
			t.Errorf("unexpected violation at line %d: %s", v.Line, v.Message)
		}
	}
}

func TestCheckMakefileQuality_MissingFmtGo(t *testing.T) {
	content := `fmt: fmt-ui ## Format all code

fmt-go:
	@if [ -d api ] && find api -name "*.go" | head -1 | grep -q .; then \
		echo "Formatting Go code..."; \
		if command -v gofumpt >/dev/null 2>&1; then \
			cd api && gofumpt -w .; \
		elif command -v gofmt >/dev/null 2>&1; then \
			cd api && gofmt -w .; \
		fi; \
		echo "$(GREEN)✓ Go code formatted$(RESET)"; \
	fi

fmt-ui:
	@echo "Formatting UI assets..."

lint: lint-go lint-ui

lint-go:
	@if [ -d api ] && find api -name "*.go" | head -1 | grep -q .; then \
		echo "Linting Go code..."; \
		if command -v golangci-lint >/dev/null 2>&1; then \
			cd api && golangci-lint run ./...; \
		else \
			cd api && go vet ./...; \
		fi; \
		echo "$(GREEN)✓ Go code linted$(RESET)"; \
	fi

lint-ui:
	@echo "Linting UI code..."

check: fmt lint test
`
	violations, err := CheckMakefileQuality(content, "Makefile")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(violations) != 1 {
		t.Fatalf("expected 1 violation, got %d", len(violations))
	}
	if !strings.Contains(violations[0].Message, "fmt target must depend on or invoke 'fmt-go'") {
		t.Errorf("unexpected message: %s", violations[0].Message)
	}
}

func TestCheckMakefileQuality_LintCliDirectory(t *testing.T) {
	// CLI-only scenarios should pass quality checks — both api/ and cli/ are
	// valid Go source directories.
	content := `fmt: fmt-go fmt-ui

fmt-go:
	@if [ -d cli ] && find cli -name "*.go" | head -1 | grep -q .; then \
		echo "Formatting Go code..."; \
		if command -v gofumpt >/dev/null 2>&1; then \
			cd cli && gofumpt -w .; \
		elif command -v gofmt >/dev/null 2>&1; then \
			cd cli && gofmt -w .; \
		fi; \
		echo "$(GREEN)✓ Go code formatted$(RESET)"; \
	fi

lint: lint-go lint-ui

lint-go:
	@if [ -d cli ] && find cli -name "*.go" | head -1 | grep -q .; then \
		echo "Linting Go code..."; \
		if command -v golangci-lint >/dev/null 2>&1; then \
			cd cli && golangci-lint run ./...; \
		else \
			cd cli && go vet ./...; \
		fi; \
		echo "$(GREEN)✓ Go code linted$(RESET)"; \
	fi

lint-ui:
	@echo "Linting UI code..."

check: fmt lint test
`
	violations, err := CheckMakefileQuality(content, "Makefile")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(violations) != 0 {
		for _, v := range violations {
			t.Errorf("unexpected violation at line %d: %s", v.Line, v.Message)
		}
	}
}

func TestCheckMakefileQuality_MissingGuards(t *testing.T) {
	content := `fmt: fmt-go

fmt-go:
	cd api && gofumpt -w .

lint: lint-go

lint-go:
	cd api && golangci-lint run ./...

check: fmt lint test
`
	violations, err := CheckMakefileQuality(content, "Makefile")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(violations) != 6 {
		t.Errorf("expected 6 violations, got %d", len(violations))
		for _, v := range violations {
			t.Logf("  violation: %s", v.Message)
		}
	}
}

func TestCheckMakefileQuality_MakeInvocationFlags(t *testing.T) {
	content := `fmt: ## Format all code
	@$(MAKE) --no-print-directory fmt-go
	@$(MAKE) --no-print-directory fmt-ui

fmt-go:
	@if [ -d api ] && [ -f api/go.mod ]; then \
		echo "Formatting Go code..."; \
		if command -v gofumpt >/dev/null 2>&1; then \
			cd api && gofumpt -w .; \
		else \
			cd api && gofmt -w .; \
		fi; \
	fi

fmt-ui:
	@echo "Formatting UI assets..."

lint: ## Lint all code
	@$(MAKE) --no-print-directory lint-go
	@$(MAKE) --no-print-directory lint-ui

lint-go:
	@if [ -d api ] && [ -f api/go.mod ]; then \
		echo "Linting Go code..."; \
		if command -v golangci-lint >/dev/null 2>&1; then \
			cd api && golangci-lint run ./...; \
		else \
			cd api && go vet ./...; \
		fi; \
	fi

lint-ui:
	@echo "Linting UI code..."

test:
	@echo "Running tests..."

check: ## Run full quality gates
	@$(MAKE) fmt
	@$(MAKE) lint
	@$(MAKE) test
`
	violations, err := CheckMakefileQuality(content, "Makefile")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(violations) != 0 {
		for _, v := range violations {
			t.Errorf("unexpected violation at line %d: %s", v.Line, v.Message)
		}
	}
}

func TestCheckMakefileQuality_CommentOnlyRecipe(t *testing.T) {
	// All quality patterns appear in comments only — this should NOT pass.
	content := `fmt: fmt-go fmt-ui

fmt-go:
	@# if [ -d api ] && find api -name "*.go" | head -1 | grep -q .; then
	@# cd api && gofumpt -w .
	@# gofmt -w .
	@echo "Go formatting disabled"

lint: lint-go lint-ui

lint-go:
	@# if [ -d api ] && find api -name "*.go" | head -1 | grep -q .
	@# cd api && golangci-lint run ./...
	@# go vet ./...
	@echo "Go linting disabled"

lint-ui:
	@echo "UI lint not configured"

check: fmt lint test
`
	violations, err := CheckMakefileQuality(content, "Makefile")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Should have violations because the patterns are only in comments.
	if len(violations) == 0 {
		t.Error("expected violations when patterns only appear in comments")
	}
}

func TestCheckMakefileQuality_CommentAndCode(t *testing.T) {
	// Both comments with patterns and actual code — should pass.
	content := `fmt: fmt-go fmt-ui

fmt-go:
	@# This target formats Go code using gofumpt
	@if [ -d api ] && find api -name "*.go" | head -1 | grep -q .; then \
		echo "Formatting Go code..."; \
		if command -v gofumpt >/dev/null 2>&1; then \
			cd api && gofumpt -w .; \
		elif command -v gofmt >/dev/null 2>&1; then \
			cd api && gofmt -w .; \
		fi; \
	fi

lint: lint-go lint-ui

lint-go:
	@# Lint Go code with golangci-lint, fallback to go vet
	@if [ -d api ] && find api -name "*.go" | head -1 | grep -q .; then \
		if command -v golangci-lint >/dev/null 2>&1; then \
			cd api && golangci-lint run ./...; \
		else \
			cd api && go vet ./...; \
		fi; \
	fi

lint-ui:
	@echo "UI lint not configured"

check: fmt lint test
`
	violations, err := CheckMakefileQuality(content, "Makefile")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(violations) != 0 {
		for _, v := range violations {
			t.Errorf("unexpected violation at line %d: %s", v.Line, v.Message)
		}
	}
}
