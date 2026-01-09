package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// [REQ:TM-LS-003] Test graceful handling when Makefile is missing
func TestNoMakefile_GracefulHandling(t *testing.T) {
	tmpDir := t.TempDir()

	// Create directory structure but NO Makefile
	apiDir := filepath.Join(tmpDir, "api")
	if err := os.MkdirAll(apiDir, 0755); err != nil {
		t.Fatalf("failed to create api directory: %v", err)
	}

	// Create test files
	testFile := filepath.Join(apiDir, "main.go")
	if err := os.WriteFile(testFile, []byte("package main\n\nfunc main() {}\n"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	scanner := NewLightScanner(tmpDir, 30*time.Second)
	ctx := context.Background()

	result, err := scanner.Scan(ctx)
	// Scan should succeed even without Makefile
	if err != nil {
		t.Fatalf("scan should succeed even without Makefile: %v", err)
	}

	// Verify Makefile detection flag
	if result.HasMakefile {
		t.Error("expected HasMakefile to be false when Makefile doesn't exist")
	}

	// Verify lint/type outputs are nil or skipped
	if result.LintOutput != nil && !result.LintOutput.Skipped {
		t.Error("expected LintOutput to be nil or skipped when no Makefile")
	}

	if result.TypeOutput != nil && !result.TypeOutput.Skipped {
		t.Error("expected TypeOutput to be nil or skipped when no Makefile")
	}

	// Verify file metrics still work
	if result.FileMetrics == nil {
		t.Error("expected FileMetrics to be collected even without Makefile")
	}

	if len(result.FileMetrics) == 0 {
		t.Error("expected at least one file in metrics")
	}

	// Verify total counts
	if result.TotalFiles == 0 {
		t.Error("expected TotalFiles to be greater than 0")
	}

	if result.TotalLines == 0 {
		t.Error("expected TotalLines to be greater than 0")
	}
}

// [REQ:TM-LS-003] Test scenario with empty Makefile (no targets)
func TestNoMakefile_EmptyMakefile(t *testing.T) {
	tmpDir := t.TempDir()

	// Create an empty Makefile
	makefileContent := `
# This Makefile has no targets
`
	if err := os.WriteFile(filepath.Join(tmpDir, "Makefile"), []byte(makefileContent), 0644); err != nil {
		t.Fatalf("failed to create Makefile: %v", err)
	}

	apiDir := filepath.Join(tmpDir, "api")
	if err := os.MkdirAll(apiDir, 0755); err != nil {
		t.Fatalf("failed to create api directory: %v", err)
	}

	testFile := filepath.Join(apiDir, "test.go")
	if err := os.WriteFile(testFile, []byte("package main\n"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	scanner := NewLightScanner(tmpDir, 30*time.Second)
	ctx := context.Background()

	result, err := scanner.Scan(ctx)
	if err != nil {
		t.Fatalf("scan should complete with empty Makefile: %v", err)
	}

	// Makefile exists but has no targets
	if !result.HasMakefile {
		t.Error("expected HasMakefile to be true (file exists)")
	}

	// Lint/type commands might fail or be skipped
	// Both outcomes are acceptable for an empty Makefile
	t.Logf("LintOutput: %v", result.LintOutput)
	t.Logf("TypeOutput: %v", result.TypeOutput)
}

// [REQ:TM-LS-003] Test scenario with Makefile but no lint/type targets
func TestNoMakefile_NoLintTypeTargets(t *testing.T) {
	tmpDir := t.TempDir()

	// Create Makefile with other targets but not lint/type
	makefileContent := `
.PHONY: build
build:
	@echo "Building..."

.PHONY: test
test:
	@echo "Testing..."
`
	if err := os.WriteFile(filepath.Join(tmpDir, "Makefile"), []byte(makefileContent), 0644); err != nil {
		t.Fatalf("failed to create Makefile: %v", err)
	}

	apiDir := filepath.Join(tmpDir, "api")
	if err := os.MkdirAll(apiDir, 0755); err != nil {
		t.Fatalf("failed to create api directory: %v", err)
	}

	testFile := filepath.Join(apiDir, "test.go")
	if err := os.WriteFile(testFile, []byte("package main\n"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	scanner := NewLightScanner(tmpDir, 30*time.Second)
	ctx := context.Background()

	result, err := scanner.Scan(ctx)
	if err != nil {
		t.Fatalf("scan should complete even when lint/type targets don't exist: %v", err)
	}

	// Makefile exists
	if !result.HasMakefile {
		t.Error("expected HasMakefile to be true")
	}

	// File metrics should still be collected
	if len(result.FileMetrics) == 0 {
		t.Error("expected file metrics to be collected")
	}
}

// [REQ:TM-LS-003] Test scenario with unreadable Makefile
func TestNoMakefile_UnreadableMakefile(t *testing.T) {
	tmpDir := t.TempDir()

	// Create Makefile with restricted permissions
	makefilePath := filepath.Join(tmpDir, "Makefile")
	makefileContent := `
.PHONY: lint
lint:
	@echo "lint"
`
	if err := os.WriteFile(makefilePath, []byte(makefileContent), 0000); err != nil {
		t.Fatalf("failed to create Makefile: %v", err)
	}
	defer os.Chmod(makefilePath, 0644) // Cleanup

	apiDir := filepath.Join(tmpDir, "api")
	if err := os.MkdirAll(apiDir, 0755); err != nil {
		t.Fatalf("failed to create api directory: %v", err)
	}

	testFile := filepath.Join(apiDir, "test.go")
	if err := os.WriteFile(testFile, []byte("package main\n"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	scanner := NewLightScanner(tmpDir, 30*time.Second)
	ctx := context.Background()

	result, err := scanner.Scan(ctx)
	// Should complete gracefully even if Makefile is unreadable
	if err != nil {
		t.Fatalf("scan should complete even with unreadable Makefile: %v", err)
	}

	// File metrics should still be collected
	if len(result.FileMetrics) == 0 {
		t.Error("expected file metrics to be collected despite Makefile issues")
	}
}

// [REQ:TM-LS-003] Test empty Makefile output handling
func TestNoMakefile_EmptyCommandOutput(t *testing.T) {
	tmpDir := t.TempDir()

	// Create Makefile with silent commands
	makefileContent := `
.PHONY: lint
lint:
	@# Silent command with no output
	@true

.PHONY: type
type:
	@# Silent command with no output
	@true
`
	if err := os.WriteFile(filepath.Join(tmpDir, "Makefile"), []byte(makefileContent), 0644); err != nil {
		t.Fatalf("failed to create Makefile: %v", err)
	}

	scanner := NewLightScanner(tmpDir, 30*time.Second)
	ctx := context.Background()

	result, err := scanner.Scan(ctx)
	if err != nil {
		t.Fatalf("scan failed: %v", err)
	}

	// Verify lint execution
	if result.LintOutput == nil {
		t.Fatal("expected LintOutput even with empty output")
	}

	if !result.LintOutput.Success {
		t.Error("expected lint to succeed with empty output")
	}

	// Check that stdout doesn't contain typical lint patterns
	// (file paths with line numbers, error messages)
	if strings.Contains(result.LintOutput.Stdout, ":") &&
		!strings.Contains(result.LintOutput.Stdout, "Entering directory") {
		t.Logf("Unexpected output in silent lint: %q", result.LintOutput.Stdout)
	}
}

// [REQ:TM-LS-003] Test scenario directory structure without source files
func TestNoMakefile_NoSourceFiles(t *testing.T) {
	tmpDir := t.TempDir()

	// Create directory structure but no source files
	apiDir := filepath.Join(tmpDir, "api")
	if err := os.MkdirAll(apiDir, 0755); err != nil {
		t.Fatalf("failed to create api directory: %v", err)
	}

	// Create only non-source files
	if err := os.WriteFile(filepath.Join(tmpDir, "README.md"), []byte("# Test"), 0644); err != nil {
		t.Fatalf("failed to create README: %v", err)
	}

	if err := os.WriteFile(filepath.Join(tmpDir, "package.json"), []byte("{}"), 0644); err != nil {
		t.Fatalf("failed to create package.json: %v", err)
	}

	scanner := NewLightScanner(tmpDir, 30*time.Second)
	ctx := context.Background()

	result, err := scanner.Scan(ctx)
	// Should complete without error
	if err != nil {
		t.Fatalf("scan should complete even without source files: %v", err)
	}

	// Should have no file metrics (or only non-source files if scanner includes them)
	if result.TotalFiles != len(result.FileMetrics) {
		t.Errorf("TotalFiles (%d) should match FileMetrics length (%d)",
			result.TotalFiles, len(result.FileMetrics))
	}
}

// [REQ:TM-LS-003] Test scenario with Makefile in subdirectory (not root)
func TestNoMakefile_MakefileInSubdirectory(t *testing.T) {
	tmpDir := t.TempDir()

	// Create Makefile in subdirectory, not root
	apiDir := filepath.Join(tmpDir, "api")
	if err := os.MkdirAll(apiDir, 0755); err != nil {
		t.Fatalf("failed to create api directory: %v", err)
	}

	makefileContent := `
.PHONY: lint
lint:
	@echo "lint from subdirectory"
`
	if err := os.WriteFile(filepath.Join(apiDir, "Makefile"), []byte(makefileContent), 0644); err != nil {
		t.Fatalf("failed to create Makefile in subdirectory: %v", err)
	}

	// Create test file
	testFile := filepath.Join(apiDir, "test.go")
	if err := os.WriteFile(testFile, []byte("package main\n"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	scanner := NewLightScanner(tmpDir, 30*time.Second)
	ctx := context.Background()

	result, err := scanner.Scan(ctx)
	if err != nil {
		t.Fatalf("scan failed: %v", err)
	}

	// Should NOT detect Makefile in subdirectory (only looks at root)
	if result.HasMakefile {
		t.Error("expected HasMakefile to be false (Makefile not in root)")
	}

	// File metrics should still work
	if len(result.FileMetrics) == 0 {
		t.Error("expected file metrics to be collected")
	}
}
