package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// [REQ:TM-LS-001] Test Makefile lint execution
func TestLightScanner_MakefileLintExecution(t *testing.T) {
	// Create temporary test scenario with Makefile
	tmpDir := t.TempDir()

	// Create a simple Makefile with lint target
	makefileContent := `
.PHONY: lint
lint:
	@echo "Running lint checks..."
	@echo "src/file.go:10:5: unused variable [deadcode]"
	@exit 0
`
	err := os.WriteFile(filepath.Join(tmpDir, "Makefile"), []byte(makefileContent), 0644)
	if err != nil {
		t.Fatalf("failed to create Makefile: %v", err)
	}

	// Create api directory to scan
	apiDir := filepath.Join(tmpDir, "api")
	err = os.MkdirAll(apiDir, 0755)
	if err != nil {
		t.Fatalf("failed to create api directory: %v", err)
	}

	// Create a test file
	testFile := filepath.Join(apiDir, "test.go")
	err = os.WriteFile(testFile, []byte("package main\n\nfunc main() {}\n"), 0644)
	if err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	scanner := NewLightScanner(tmpDir, 30*time.Second)
	ctx := context.Background()

	result, err := scanner.Scan(ctx)
	if err != nil {
		t.Fatalf("scan failed: %v", err)
	}

	if !result.HasMakefile {
		t.Error("expected HasMakefile to be true")
	}

	if result.LintOutput == nil {
		t.Fatal("expected LintOutput to be populated")
	}

	if result.LintOutput.Command != "make lint" {
		t.Errorf("expected command 'make lint', got %q", result.LintOutput.Command)
	}

	if result.LintOutput.Skipped {
		t.Error("expected lint command to not be skipped")
	}

	if result.LintOutput.ExitCode != 0 {
		t.Errorf("expected exit code 0, got %d", result.LintOutput.ExitCode)
	}
}

// [REQ:TM-LS-002] Test Makefile type check execution
func TestLightScanner_MakefileTypeExecution(t *testing.T) {
	tmpDir := t.TempDir()

	makefileContent := `
.PHONY: type
type:
	@echo "Running type checks..."
	@echo "src/file.ts(10,5): error TS2304: Cannot find name 'unknown'"
	@exit 0
`
	err := os.WriteFile(filepath.Join(tmpDir, "Makefile"), []byte(makefileContent), 0644)
	if err != nil {
		t.Fatalf("failed to create Makefile: %v", err)
	}

	scanner := NewLightScanner(tmpDir, 30*time.Second)
	ctx := context.Background()

	result, err := scanner.Scan(ctx)
	if err != nil {
		t.Fatalf("scan failed: %v", err)
	}

	if result.TypeOutput == nil {
		t.Fatal("expected TypeOutput to be populated")
	}

	if result.TypeOutput.Command != "make type" {
		t.Errorf("expected command 'make type', got %q", result.TypeOutput.Command)
	}

	if result.TypeOutput.Skipped {
		t.Error("expected type command to not be skipped")
	}
}

// [REQ:TM-LS-003] Test graceful handling when Makefile is missing
func TestLightScanner_NoMakefile(t *testing.T) {
	tmpDir := t.TempDir()

	// Don't create a Makefile
	apiDir := filepath.Join(tmpDir, "api")
	err := os.MkdirAll(apiDir, 0755)
	if err != nil {
		t.Fatalf("failed to create api directory: %v", err)
	}

	scanner := NewLightScanner(tmpDir, 30*time.Second)
	ctx := context.Background()

	result, err := scanner.Scan(ctx)
	if err != nil {
		t.Fatalf("scan should succeed even without Makefile: %v", err)
	}

	if result.HasMakefile {
		t.Error("expected HasMakefile to be false")
	}

	if result.LintOutput != nil {
		t.Error("expected LintOutput to be nil when no Makefile")
	}

	if result.TypeOutput != nil {
		t.Error("expected TypeOutput to be nil when no Makefile")
	}

	// File metrics should still work
	if result.FileMetrics == nil {
		t.Error("expected FileMetrics to be collected even without Makefile")
	}
}

// [REQ:TM-LS-005] Test file line counting
func TestLightScanner_FileMetrics(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test files with known line counts
	apiDir := filepath.Join(tmpDir, "api")
	err := os.MkdirAll(apiDir, 0755)
	if err != nil {
		t.Fatalf("failed to create api directory: %v", err)
	}

	// Create file with 10 non-empty lines
	file1 := filepath.Join(apiDir, "test1.go")
	content1 := "package main\n\nfunc test1() {\n  x := 1\n  y := 2\n  z := 3\n  return\n}\n\nfunc test2() {}\n"
	err = os.WriteFile(file1, []byte(content1), 0644)
	if err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	// Create file with 5 non-empty lines
	file2 := filepath.Join(apiDir, "test2.go")
	content2 := "package main\n\nfunc foo() {\n  println(\"hello\")\n}\n"
	err = os.WriteFile(file2, []byte(content2), 0644)
	if err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	scanner := NewLightScanner(tmpDir, 30*time.Second)
	ctx := context.Background()

	result, err := scanner.Scan(ctx)
	if err != nil {
		t.Fatalf("scan failed: %v", err)
	}

	if len(result.FileMetrics) != 2 {
		t.Errorf("expected 2 files, got %d", len(result.FileMetrics))
	}

	if result.TotalFiles != 2 {
		t.Errorf("expected TotalFiles=2, got %d", result.TotalFiles)
	}

	// Verify line counts (exact counts may vary based on empty line handling)
	for _, m := range result.FileMetrics {
		if m.Lines == 0 {
			t.Errorf("file %s should have non-zero line count", m.Path)
		}
		if m.Extension != ".go" {
			t.Errorf("expected extension .go, got %s", m.Extension)
		}
	}

	if result.TotalLines == 0 {
		t.Error("expected TotalLines to be greater than 0")
	}
}

// [REQ:TM-LS-006] Test long file threshold flagging
func TestLightScanner_LongFileDetection(t *testing.T) {
	tmpDir := t.TempDir()

	apiDir := filepath.Join(tmpDir, "api")
	err := os.MkdirAll(apiDir, 0755)
	if err != nil {
		t.Fatalf("failed to create api directory: %v", err)
	}

	// Create a file with >500 lines
	longFile := filepath.Join(apiDir, "long.go")
	var content string
	content += "package main\n\n"
	for i := 0; i < 600; i++ {
		content += "// Line comment to increase file length\n"
	}
	err = os.WriteFile(longFile, []byte(content), 0644)
	if err != nil {
		t.Fatalf("failed to create long file: %v", err)
	}

	// Create a normal file
	normalFile := filepath.Join(apiDir, "normal.go")
	err = os.WriteFile(normalFile, []byte("package main\n\nfunc main() {}\n"), 0644)
	if err != nil {
		t.Fatalf("failed to create normal file: %v", err)
	}

	scanner := NewLightScanner(tmpDir, 30*time.Second)
	ctx := context.Background()

	result, err := scanner.Scan(ctx)
	if err != nil {
		t.Fatalf("scan failed: %v", err)
	}

	if len(result.LongFiles) == 0 {
		t.Fatal("expected at least one long file to be detected")
	}

	foundLong := false
	for _, lf := range result.LongFiles {
		if filepath.Base(lf.Path) == "long.go" {
			foundLong = true
			if lf.Lines <= 500 {
				t.Errorf("expected long file to have >500 lines, got %d", lf.Lines)
			}
			if lf.Threshold != 500 {
				t.Errorf("expected threshold 500, got %d", lf.Threshold)
			}
		}
	}

	if !foundLong {
		t.Error("expected to find long.go in LongFiles list")
	}
}

// [REQ:TM-LS-007,TM-LS-008] Test scan performance and timeout handling
func TestLightScanner_Timeout(t *testing.T) {
	tmpDir := t.TempDir()

	// Create Makefile with slow target
	makefileContent := `
.PHONY: lint
lint:
	@sleep 10
	@echo "Done"
`
	err := os.WriteFile(filepath.Join(tmpDir, "Makefile"), []byte(makefileContent), 0644)
	if err != nil {
		t.Fatalf("failed to create Makefile: %v", err)
	}

	// Use very short timeout to force timeout
	scanner := NewLightScanner(tmpDir, 1*time.Second)
	ctx := context.Background()

	result, err := scanner.Scan(ctx)
	// Scan should still complete (file metrics succeed even if commands timeout)
	if err != nil {
		t.Fatalf("scan should complete despite command timeout: %v", err)
	}

	// Lint should have been attempted
	if result.LintOutput == nil {
		t.Error("expected LintOutput even if timed out")
	}
}
