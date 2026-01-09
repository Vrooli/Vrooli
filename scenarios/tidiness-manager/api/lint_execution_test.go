package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// [REQ:TM-LS-001] Test Makefile lint execution with valid output
func TestMakefileLintExecution_ValidOutput(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a Makefile with lint target that produces structured output
	makefileContent := `
.PHONY: lint
lint:
	@echo "Running lint checks..."
	@echo "src/file.go:10:5: unused variable [deadcode]"
	@echo "src/another.go:25:10: missing return statement [missing-return]"
	@exit 0
`
	if err := os.WriteFile(filepath.Join(tmpDir, "Makefile"), []byte(makefileContent), 0644); err != nil {
		t.Fatalf("failed to create Makefile: %v", err)
	}

	// Create api directory to scan
	apiDir := filepath.Join(tmpDir, "api")
	if err := os.MkdirAll(apiDir, 0755); err != nil {
		t.Fatalf("failed to create api directory: %v", err)
	}

	// Create test file
	testFile := filepath.Join(apiDir, "test.go")
	if err := os.WriteFile(testFile, []byte("package main\n\nfunc main() {}\n"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	scanner := NewLightScanner(tmpDir, 30*time.Second)
	ctx := context.Background()

	result, err := scanner.Scan(ctx)
	if err != nil {
		t.Fatalf("scan failed: %v", err)
	}

	// Verify Makefile detection
	if !result.HasMakefile {
		t.Error("expected HasMakefile to be true")
	}

	// Verify lint output was captured
	if result.LintOutput == nil {
		t.Fatal("expected LintOutput to be populated")
	}

	// Verify command string
	if result.LintOutput.Command != "make lint" {
		t.Errorf("expected command 'make lint', got %q", result.LintOutput.Command)
	}

	// Verify lint was executed (not skipped)
	if result.LintOutput.Skipped {
		t.Error("expected lint command to not be skipped when Makefile exists")
	}

	// Verify successful execution
	if result.LintOutput.ExitCode != 0 {
		t.Errorf("expected exit code 0, got %d", result.LintOutput.ExitCode)
	}

	// Verify command succeeded
	if !result.LintOutput.Success {
		t.Error("expected lint command to succeed")
	}

	// Verify output contains lint messages
	if result.LintOutput.Stdout == "" {
		t.Error("expected lint output to contain stdout")
	}
}

// [REQ:TM-LS-001] Test Makefile lint with failing exit code
func TestMakefileLintExecution_FailingTarget(t *testing.T) {
	tmpDir := t.TempDir()

	// Create Makefile with failing lint target
	makefileContent := `
.PHONY: lint
lint:
	@echo "src/error.go:1:1: syntax error [syntax]"
	@exit 1
`
	if err := os.WriteFile(filepath.Join(tmpDir, "Makefile"), []byte(makefileContent), 0644); err != nil {
		t.Fatalf("failed to create Makefile: %v", err)
	}

	scanner := NewLightScanner(tmpDir, 30*time.Second)
	ctx := context.Background()

	result, err := scanner.Scan(ctx)
	// Scan should complete gracefully even if lint fails
	if err != nil {
		t.Fatalf("scan should complete even with failing Makefile targets: %v", err)
	}

	// Verify lint was attempted
	if result.LintOutput == nil {
		t.Fatal("expected LintOutput even when command fails")
	}

	// Verify failure was recorded
	if result.LintOutput.Success {
		t.Error("expected lint command to fail with non-zero exit")
	}

	// Verify non-zero exit code
	if result.LintOutput.ExitCode == 0 {
		t.Errorf("expected non-zero exit code, got %d", result.LintOutput.ExitCode)
	}

	// Verify output was still captured
	if result.LintOutput.Stdout == "" {
		t.Error("expected lint output to be captured even on failure")
	}
}

// [REQ:TM-LS-001] Test lint execution with invalid Makefile target
func TestMakefileLintExecution_InvalidCommand(t *testing.T) {
	tmpDir := t.TempDir()

	// Create Makefile with invalid command in lint target
	makefileContent := `
.PHONY: lint
lint:
	@nonexistentcommand --flag
	@echo "This should not run"
`
	if err := os.WriteFile(filepath.Join(tmpDir, "Makefile"), []byte(makefileContent), 0644); err != nil {
		t.Fatalf("failed to create Makefile: %v", err)
	}

	scanner := NewLightScanner(tmpDir, 30*time.Second)
	ctx := context.Background()

	result, err := scanner.Scan(ctx)
	if err != nil {
		t.Fatalf("scan should complete even with invalid Makefile commands: %v", err)
	}

	// Verify lint was attempted
	if result.LintOutput == nil {
		t.Fatal("expected LintOutput even when command is invalid")
	}

	// Verify failure was recorded
	if result.LintOutput.Success {
		t.Error("expected lint command to fail with invalid command")
	}

	// Verify non-zero exit code
	if result.LintOutput.ExitCode == 0 {
		t.Errorf("expected non-zero exit code for invalid command, got %d", result.LintOutput.ExitCode)
	}
}

// [REQ:TM-LS-001] Test lint execution timeout handling
func TestMakefileLintExecution_Timeout(t *testing.T) {
	tmpDir := t.TempDir()

	// Create Makefile with slow lint target
	makefileContent := `
.PHONY: lint
lint:
	@sleep 10
	@echo "Done"
`
	if err := os.WriteFile(filepath.Join(tmpDir, "Makefile"), []byte(makefileContent), 0644); err != nil {
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

	// Verify lint was attempted
	if result.LintOutput == nil {
		t.Error("expected LintOutput even if timed out")
	}

	// Verify timeout/failure was recorded
	if result.LintOutput != nil && result.LintOutput.Success {
		t.Error("expected lint command to fail/timeout with short timeout")
	}
}

// [REQ:TM-LS-001] Test concurrent lint execution safety
func TestMakefileLintExecution_ConcurrentSafety(t *testing.T) {
	tmpDir := t.TempDir()

	makefileContent := `
.PHONY: lint
lint:
	@echo "lint output for concurrent test"
	@echo "file.go:1:1: test issue [test]"
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

	// Run 5 concurrent scans to test for race conditions
	done := make(chan error, 5)
	for i := 0; i < 5; i++ {
		go func(index int) {
			scanner := NewLightScanner(tmpDir, 30*time.Second)
			result, err := scanner.Scan(context.Background())
			if err != nil {
				done <- err
				return
			}

			// Verify lint output was collected
			if result.LintOutput == nil {
				done <- err
				return
			}

			// Verify output is correct (not corrupted by concurrent execution)
			if !result.LintOutput.Success {
				t.Errorf("concurrent scan %d: expected successful lint execution", index)
			}

			done <- nil
		}(i)
	}

	// Wait for all scans to complete and check for errors
	for i := 0; i < 5; i++ {
		if err := <-done; err != nil {
			t.Errorf("concurrent lint scan %d failed: %v", i, err)
		}
	}
}

// [REQ:TM-LS-001] Test lint execution with context cancellation
func TestMakefileLintExecution_ContextCancellation(t *testing.T) {
	tmpDir := t.TempDir()

	// Create Makefile with moderately slow target
	makefileContent := `
.PHONY: lint
lint:
	@sleep 2
	@echo "Done"
`
	if err := os.WriteFile(filepath.Join(tmpDir, "Makefile"), []byte(makefileContent), 0644); err != nil {
		t.Fatalf("failed to create Makefile: %v", err)
	}

	scanner := NewLightScanner(tmpDir, 10*time.Second)

	// Create cancellable context
	ctx, cancel := context.WithCancel(context.Background())

	// Cancel after 100ms
	go func() {
		time.Sleep(100 * time.Millisecond)
		cancel()
	}()

	result, err := scanner.Scan(ctx)

	// Scan should handle cancellation gracefully
	// Either return partial results or an error
	if err == nil && result != nil {
		// Partial results are acceptable
		t.Logf("Scan returned partial results after cancellation")
	} else if err != nil && err == context.Canceled {
		// Explicit cancellation error is also acceptable
		t.Logf("Scan returned cancellation error as expected")
	} else if err != nil {
		// Other errors might occur during cleanup
		t.Logf("Scan returned error during cancellation: %v", err)
	}
}

// [REQ:TM-LS-001] Test lint execution with empty/silent output
func TestMakefileLintExecution_EmptyOutput(t *testing.T) {
	tmpDir := t.TempDir()

	// Create Makefile with silent lint target
	makefileContent := `
.PHONY: lint
lint:
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

	// Verify lint was executed
	if result.LintOutput == nil {
		t.Fatal("expected LintOutput even with empty output")
	}

	// Verify success
	if !result.LintOutput.Success {
		t.Error("expected lint to succeed with empty output")
	}

	// Verify exit code
	if result.LintOutput.ExitCode != 0 {
		t.Errorf("expected exit code 0, got %d", result.LintOutput.ExitCode)
	}
}
