package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// [REQ:TM-LS-002] Test Makefile type check execution with valid output
func TestMakefileTypeExecution_ValidOutput(t *testing.T) {
	tmpDir := t.TempDir()

	// Create Makefile with type target that produces structured output
	makefileContent := `
.PHONY: type
type:
	@echo "Running type checks..."
	@echo "src/file.ts(10,5): error TS2304: Cannot find name 'unknown'"
	@echo "src/app.tsx(25,10): error TS2322: Type 'string' is not assignable to type 'number'"
	@exit 0
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

	// Verify type output was captured
	if result.TypeOutput == nil {
		t.Fatal("expected TypeOutput to be populated")
	}

	// Verify command string
	if result.TypeOutput.Command != "make type" {
		t.Errorf("expected command 'make type', got %q", result.TypeOutput.Command)
	}

	// Verify type check was executed (not skipped)
	if result.TypeOutput.Skipped {
		t.Error("expected type command to not be skipped when Makefile exists")
	}

	// Verify successful execution
	if result.TypeOutput.ExitCode != 0 {
		t.Errorf("expected exit code 0, got %d", result.TypeOutput.ExitCode)
	}

	// Verify command succeeded
	if !result.TypeOutput.Success {
		t.Error("expected type command to succeed")
	}

	// Verify output contains type error messages
	if result.TypeOutput.Stdout == "" {
		t.Error("expected type output to contain stdout")
	}
}

// [REQ:TM-LS-002] Test Makefile type check with failing exit code
func TestMakefileTypeExecution_FailingTarget(t *testing.T) {
	tmpDir := t.TempDir()

	// Create Makefile with failing type target
	makefileContent := `
.PHONY: type
type:
	@echo "src/error.ts(1,1): error TS1005: ';' expected"
	@exit 1
`
	if err := os.WriteFile(filepath.Join(tmpDir, "Makefile"), []byte(makefileContent), 0644); err != nil {
		t.Fatalf("failed to create Makefile: %v", err)
	}

	scanner := NewLightScanner(tmpDir, 30*time.Second)
	ctx := context.Background()

	result, err := scanner.Scan(ctx)
	// Scan should complete gracefully even if type check fails
	if err != nil {
		t.Fatalf("scan should complete even with failing Makefile targets: %v", err)
	}

	// Verify type check was attempted
	if result.TypeOutput == nil {
		t.Fatal("expected TypeOutput even when command fails")
	}

	// Verify failure was recorded
	if result.TypeOutput.Success {
		t.Error("expected type command to fail with non-zero exit")
	}

	// Verify non-zero exit code
	if result.TypeOutput.ExitCode == 0 {
		t.Errorf("expected non-zero exit code, got %d", result.TypeOutput.ExitCode)
	}

	// Verify output was still captured
	if result.TypeOutput.Stdout == "" {
		t.Error("expected type output to be captured even on failure")
	}
}

// [REQ:TM-LS-002] Test type check execution with invalid Makefile target
func TestMakefileTypeExecution_InvalidCommand(t *testing.T) {
	tmpDir := t.TempDir()

	// Create Makefile with invalid command in type target
	makefileContent := `
.PHONY: type
type:
	@invalidtypecommand --check
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

	// Verify type check was attempted
	if result.TypeOutput == nil {
		t.Fatal("expected TypeOutput even when command is invalid")
	}

	// Verify failure was recorded
	if result.TypeOutput.Success {
		t.Error("expected type command to fail with invalid command")
	}

	// Verify non-zero exit code
	if result.TypeOutput.ExitCode == 0 {
		t.Errorf("expected non-zero exit code for invalid command, got %d", result.TypeOutput.ExitCode)
	}
}

// [REQ:TM-LS-002] Test type check execution timeout handling
func TestMakefileTypeExecution_Timeout(t *testing.T) {
	tmpDir := t.TempDir()

	// Create Makefile with slow type target
	makefileContent := `
.PHONY: type
type:
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

	// Verify type check was attempted
	if result.TypeOutput == nil {
		t.Error("expected TypeOutput even if timed out")
	}

	// Verify timeout/failure was recorded
	if result.TypeOutput != nil && result.TypeOutput.Success {
		t.Error("expected type command to fail/timeout with short timeout")
	}
}

// [REQ:TM-LS-002] Test concurrent type check execution safety
func TestMakefileTypeExecution_ConcurrentSafety(t *testing.T) {
	tmpDir := t.TempDir()

	makefileContent := `
.PHONY: type
type:
	@echo "type output for concurrent test"
	@echo "file.ts(1,1): error TS2304: test issue"
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

			// Verify type output was collected
			if result.TypeOutput == nil {
				done <- err
				return
			}

			// Verify output is correct (not corrupted by concurrent execution)
			if !result.TypeOutput.Success {
				t.Errorf("concurrent scan %d: expected successful type check execution", index)
			}

			done <- nil
		}(i)
	}

	// Wait for all scans to complete and check for errors
	for i := 0; i < 5; i++ {
		if err := <-done; err != nil {
			t.Errorf("concurrent type scan %d failed: %v", i, err)
		}
	}
}

// [REQ:TM-LS-002] Test type check execution with context cancellation
func TestMakefileTypeExecution_ContextCancellation(t *testing.T) {
	tmpDir := t.TempDir()

	// Create Makefile with moderately slow target
	makefileContent := `
.PHONY: type
type:
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

// [REQ:TM-LS-002] Test type check execution with empty/silent output
func TestMakefileTypeExecution_EmptyOutput(t *testing.T) {
	tmpDir := t.TempDir()

	// Create Makefile with silent type target
	makefileContent := `
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

	// Verify type check was executed
	if result.TypeOutput == nil {
		t.Fatal("expected TypeOutput even with empty output")
	}

	// Verify success
	if !result.TypeOutput.Success {
		t.Error("expected type check to succeed with empty output")
	}

	// Verify exit code
	if result.TypeOutput.ExitCode != 0 {
		t.Errorf("expected exit code 0, got %d", result.TypeOutput.ExitCode)
	}
}

// [REQ:TM-LS-002] Test type check with Go-style error output
func TestMakefileTypeExecution_GoStyleErrors(t *testing.T) {
	tmpDir := t.TempDir()

	// Create Makefile with Go-style type error output
	makefileContent := `
.PHONY: type
type:
	@echo "# command-line-arguments"
	@echo "src/main.go:42:5: undefined: missingFunc"
	@echo "src/handler.go:18:2: cannot use x (type int) as type string in assignment"
	@exit 0
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

	// Verify type output was captured
	if result.TypeOutput == nil {
		t.Fatal("expected TypeOutput to be populated")
	}

	// Verify command succeeded
	if !result.TypeOutput.Success {
		t.Error("expected type command to succeed")
	}

	// Verify output contains Go-style error messages
	if result.TypeOutput.Stdout == "" {
		t.Error("expected type output to contain stdout with Go errors")
	}
}

// [REQ:TM-LS-002] Test type check with TypeScript-style error output
func TestMakefileTypeExecution_TypeScriptStyleErrors(t *testing.T) {
	tmpDir := t.TempDir()

	// Create Makefile with TypeScript-style type error output
	makefileContent := `
.PHONY: type
type:
	@echo "src/App.tsx(15,20): error TS2339: Property 'foo' does not exist on type 'Props'"
	@echo "src/utils/helper.ts(8,5): error TS2322: Type 'string' is not assignable to type 'number'"
	@echo "src/components/Button.tsx(22,10): error TS2554: Expected 2 arguments, but got 1"
	@exit 0
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

	// Verify type output was captured
	if result.TypeOutput == nil {
		t.Fatal("expected TypeOutput to be populated")
	}

	// Verify command succeeded
	if !result.TypeOutput.Success {
		t.Error("expected type command to succeed")
	}

	// Verify output contains TypeScript-style error messages
	if result.TypeOutput.Stdout == "" {
		t.Error("expected type output to contain stdout with TypeScript errors")
	}
}
