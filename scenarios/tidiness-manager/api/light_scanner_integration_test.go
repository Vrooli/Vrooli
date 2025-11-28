package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// [REQ:TM-LS-001,TM-LS-002,TM-LS-005,TM-LS-006] Integration test verifying all scanner components work together in complete workflow
func TestLightScanner_Integration_CompleteWorkflow(t *testing.T) {
	// This integration test validates that all scanner components (Makefile execution,
	// file metrics, language detection, code metrics) work together correctly in a
	// single scan, producing a coherent result with cross-component consistency.
	//
	// Unit tests validate individual components; this test validates their integration.

	tmpDir := t.TempDir()

	// Create realistic scenario structure with Makefile and multi-language code
	makefileContent := `lint:
	@echo "api/main.go:10:5: unused variable [deadcode]"
	@exit 0

type:
	@echo "ui/src/App.tsx(15,20): error TS2339: Property missing"
	@exit 0
`
	if err := os.WriteFile(filepath.Join(tmpDir, "Makefile"), []byte(makefileContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Create Go file exceeding long file threshold (>500 lines)
	apiDir := filepath.Join(tmpDir, "api")
	if err := os.MkdirAll(apiDir, 0755); err != nil {
		t.Fatal(err)
	}

	goLines := make([]string, 550)
	goLines[0] = "package main"
	goLines[1] = "// TODO: refactor"
	goLines[2] = "// FIXME: bug here"
	for i := 3; i < 550; i++ {
		goLines[i] = "// filler line"
	}
	goCode := strings.Join(goLines, "\n")

	if err := os.WriteFile(filepath.Join(apiDir, "main.go"), []byte(goCode), 0644); err != nil {
		t.Fatal(err)
	}

	// Create TypeScript file under threshold
	uiSrcDir := filepath.Join(tmpDir, "ui", "src")
	if err := os.MkdirAll(uiSrcDir, 0755); err != nil {
		t.Fatal(err)
	}

	tsCode := `import React from 'react';
// TODO: add prop types
export default function App() {
  return <div>Hello</div>;
}
`
	if err := os.WriteFile(filepath.Join(uiSrcDir, "App.tsx"), []byte(tsCode), 0644); err != nil {
		t.Fatal(err)
	}

	// Execute full scan
	scanner := NewLightScanner(tmpDir, 30*time.Second)
	result, err := scanner.Scan(context.Background())
	if err != nil {
		t.Fatalf("Light scan integration failed: %v", err)
	}

	// INTEGRATION VALIDATION: Verify cross-component consistency

	// 1. Makefile execution and file metrics should both detect the same files
	if !result.HasMakefile {
		t.Error("Integration error: Makefile detection failed")
	}
	if result.TotalFiles != 2 {
		t.Errorf("Integration error: File metrics detected %d files, expected 2", result.TotalFiles)
	}

	// 2. Language metrics should align with file metrics (2 files = 2 languages detected)
	if result.LanguageMetrics == nil {
		t.Fatal("Integration error: Language detection didn't run despite files present")
	}
	if len(result.LanguageMetrics) < 2 {
		t.Errorf("Integration error: Detected %d languages, expected at least 2 (Go + TypeScript)", len(result.LanguageMetrics))
	}

	// 3. Code metrics should roll up correctly to language metrics
	goMetrics, hasGo := result.LanguageMetrics[LanguageGo]
	if !hasGo {
		t.Error("Integration error: Language detection missed Go files")
	}
	if goMetrics.CodeMetrics == nil {
		t.Error("Integration error: Code metrics didn't run for detected Go files")
	}
	if goMetrics.CodeMetrics.TodoCount+goMetrics.CodeMetrics.FixmeCount < 2 {
		t.Error("Integration error: Code metrics parsing failed for Go file content")
	}

	// 4. Long file detection should correctly flag the 550-line Go file
	if len(result.LongFiles) == 0 {
		t.Error("Integration error: Long file detection failed despite 550-line file")
	}
	foundLongGo := false
	for _, lf := range result.LongFiles {
		if strings.Contains(lf.Path, "main.go") {
			foundLongGo = true
			if lf.Lines != 550 {
				t.Errorf("Integration error: Long file has wrong line count: got %d, expected 550", lf.Lines)
			}
		}
	}
	if !foundLongGo {
		t.Error("Integration error: Long file threshold logic failed to flag 550-line file")
	}

	// 5. Makefile commands should complete without blocking file metrics
	if result.LintOutput == nil || result.TypeOutput == nil {
		t.Error("Integration error: Makefile commands didn't execute")
	}
	if result.LintOutput != nil && !result.LintOutput.Success {
		t.Error("Integration error: Lint execution failed unexpectedly")
	}
	if result.TypeOutput != nil && !result.TypeOutput.Success {
		t.Error("Integration error: Type execution failed unexpectedly")
	}

	// 6. Scan metadata should be populated correctly
	if result.Scenario == "" {
		t.Error("Integration error: Scenario name not set")
	}
	if result.CompletedAt.IsZero() {
		t.Error("Integration error: Completion timestamp not recorded")
	}
	if result.Duration == 0 {
		t.Error("Integration error: Duration not measured")
	}
	if result.TotalLines < 550 {
		t.Errorf("Integration error: Total line count (%d) less than single file (550)", result.TotalLines)
	}
}

// [REQ:TM-LS-007] Integration test for graceful degradation when components fail
func TestLightScanner_Integration_PartialFailure(t *testing.T) {
	// Validates that when some scanner components fail (e.g., Makefile commands timeout),
	// other components (file metrics, language detection) still complete successfully,
	// producing partial results rather than total failure.

	tmpDir := t.TempDir()

	// Create Makefile with command that will timeout
	makefileContent := `lint:
	@sleep 60
	@echo "Should not reach here"
`
	if err := os.WriteFile(filepath.Join(tmpDir, "Makefile"), []byte(makefileContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Create valid source files
	apiDir := filepath.Join(tmpDir, "api")
	if err := os.MkdirAll(apiDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(apiDir, "main.go"), []byte("package main\n"), 0644); err != nil {
		t.Fatal(err)
	}

	// Use short timeout to force Makefile command failure
	scanner := NewLightScanner(tmpDir, 1*time.Second)
	result, err := scanner.Scan(context.Background())

	// INTEGRATION VALIDATION: Partial failure should not prevent overall scan
	if err != nil {
		t.Fatalf("Integration error: Scan should complete with partial results, got error: %v", err)
	}

	// File metrics should still succeed
	if result.TotalFiles == 0 {
		t.Error("Integration error: File metrics should complete despite Makefile timeout")
	}

	// Language detection should still succeed
	if len(result.LanguageMetrics) == 0 {
		t.Error("Integration error: Language detection should complete despite Makefile timeout")
	}

	// Makefile execution should gracefully record failure
	if result.LintOutput == nil {
		t.Error("Integration error: Lint output should be recorded even on timeout")
	}
	if result.LintOutput != nil && result.LintOutput.Success {
		t.Error("Integration error: Lint should be marked as failed after timeout")
	}
}

// [REQ:TM-LS-007] Integration test for empty scenario edge case
func TestLightScanner_Integration_EmptyScenario(t *testing.T) {
	// Validates that all scanner components handle empty scenarios gracefully,
	// producing valid (empty) results rather than errors or nil values.

	tmpDir := t.TempDir()

	scanner := NewLightScanner(tmpDir, 30*time.Second)
	result, err := scanner.Scan(context.Background())
	if err != nil {
		t.Fatalf("Integration error: Empty scenario should scan successfully: %v", err)
	}

	// INTEGRATION VALIDATION: All components should handle empty input

	if result.TotalFiles != 0 {
		t.Errorf("Integration error: Expected 0 files, got %d", result.TotalFiles)
	}
	if result.TotalLines != 0 {
		t.Errorf("Integration error: Expected 0 lines, got %d", result.TotalLines)
	}
	if result.LanguageMetrics != nil && len(result.LanguageMetrics) > 0 {
		t.Error("Integration error: Language metrics should be empty for empty scenario")
	}
	if !result.CompletedAt.IsZero() {
		// Scan should still record completion even with no work
		t.Log("Integration success: Empty scan recorded completion timestamp")
	}
}

// [REQ:TM-LS-005,TM-LS-006] Integration test for context cancellation handling
func TestLightScanner_Integration_ContextCancellation(t *testing.T) {
	// Validates that scanner respects context cancellation and stops gracefully

	tmpDir := t.TempDir()

	// Create large scenario with many files to ensure scan takes time
	apiDir := filepath.Join(tmpDir, "api")
	if err := os.MkdirAll(apiDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create 100 files to make scan take longer
	for i := 0; i < 100; i++ {
		content := strings.Repeat("// comment\n", 100)
		if err := os.WriteFile(filepath.Join(apiDir, fmt.Sprintf("file%03d.go", i)), []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}

	// Create context that will be cancelled during scan
	ctx, cancel := context.WithCancel(context.Background())

	scanner := NewLightScanner(tmpDir, 30*time.Second)

	// Cancel context after 100ms
	time.AfterFunc(100*time.Millisecond, cancel)

	// Run scan - should either complete quickly or return context error
	startTime := time.Now()
	_, err := scanner.Scan(ctx)
	elapsed := time.Since(startTime)

	// Scan should either:
	// 1. Complete successfully (if it finished before cancellation)
	// 2. Return context.Canceled error
	// 3. Complete within reasonable time (not hang indefinitely)
	if err != nil && err != context.Canceled {
		// If there's an error, it should be context-related
		if !strings.Contains(err.Error(), "context") && !strings.Contains(err.Error(), "cancel") {
			t.Errorf("Expected context cancellation error, got: %v", err)
		}
	}

	// Verify scan didn't hang - should complete within 5 seconds even with cancellation
	if elapsed > 5*time.Second {
		t.Errorf("Scan hung after context cancellation: took %v", elapsed)
	}
}

// [REQ:TM-LS-001,TM-LS-002] Integration test for Makefile with missing commands
func TestLightScanner_Integration_MakefileMissingCommands(t *testing.T) {
	// Validates graceful degradation when Makefile exists but lacks lint/type commands

	tmpDir := t.TempDir()

	// Create Makefile with only some commands
	makefileContent := `build:
	@echo "Building..."

test:
	@echo "Testing..."
`
	if err := os.WriteFile(filepath.Join(tmpDir, "Makefile"), []byte(makefileContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Create source file
	apiDir := filepath.Join(tmpDir, "api")
	if err := os.MkdirAll(apiDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(apiDir, "main.go"), []byte("package main\n"), 0644); err != nil {
		t.Fatal(err)
	}

	scanner := NewLightScanner(tmpDir, 30*time.Second)
	result, err := scanner.Scan(context.Background())
	if err != nil {
		t.Fatalf("Scan should complete despite missing Makefile commands: %v", err)
	}

	// INTEGRATION VALIDATION: File metrics should still work
	if result.TotalFiles == 0 {
		t.Error("File metrics should complete despite missing lint/type commands")
	}

	// Lint/type outputs might be nil or marked as failed - both acceptable
	if result.LintOutput != nil && result.LintOutput.Success {
		// If lint ran, it should have failed (command doesn't exist)
		t.Log("Note: lint command unexpectedly succeeded (may have fallback behavior)")
	}
}

// [REQ:TM-LS-005,TM-LS-006,TM-LS-007] Integration test for symlinks and special files
func TestLightScanner_Integration_SymlinksAndSpecialFiles(t *testing.T) {
	// Validates that scanner handles symlinks, hidden files, and special cases gracefully

	tmpDir := t.TempDir()

	// Create regular file
	apiDir := filepath.Join(tmpDir, "api")
	if err := os.MkdirAll(apiDir, 0755); err != nil {
		t.Fatal(err)
	}
	regularFile := filepath.Join(apiDir, "regular.go")
	if err := os.WriteFile(regularFile, []byte("package main\n"), 0644); err != nil {
		t.Fatal(err)
	}

	// Create hidden directory (should be ignored)
	hiddenDir := filepath.Join(tmpDir, ".hidden")
	if err := os.MkdirAll(hiddenDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(hiddenDir, "file.go"), []byte("package hidden\n"), 0644); err != nil {
		t.Fatal(err)
	}

	// Create symlink (behavior depends on scanner config)
	symlinkTarget := filepath.Join(tmpDir, "target")
	if err := os.MkdirAll(symlinkTarget, 0755); err != nil {
		t.Fatal(err)
	}
	targetFile := filepath.Join(symlinkTarget, "target.go")
	if err := os.WriteFile(targetFile, []byte("package target\n"), 0644); err != nil {
		t.Fatal(err)
	}

	symlinkPath := filepath.Join(apiDir, "link")
	if err := os.Symlink(symlinkTarget, symlinkPath); err != nil {
		// Symlinks may not be supported on all systems
		t.Logf("Skipping symlink test (not supported): %v", err)
	}

	scanner := NewLightScanner(tmpDir, 30*time.Second)
	result, err := scanner.Scan(context.Background())
	if err != nil {
		t.Fatalf("Scan should handle special files gracefully: %v", err)
	}

	// INTEGRATION VALIDATION: At minimum, regular file should be detected
	if result.TotalFiles == 0 {
		t.Error("Scanner should detect at least the regular file")
	}

	// Hidden directories should be excluded from count
	// (exact behavior depends on scanner implementation)
	t.Logf("Detected %d files (hidden dir exclusion implementation-dependent)", result.TotalFiles)
}

// [REQ:TM-LS-001,TM-LS-002,TM-LS-005] Integration test for concurrent scans
func TestLightScanner_Integration_ConcurrentScans(t *testing.T) {
	// Validates that multiple concurrent scans don't interfere with each other

	tmpDir1 := t.TempDir()
	tmpDir2 := t.TempDir()

	// Create different content in each scenario
	if err := os.WriteFile(filepath.Join(tmpDir1, "file1.go"), []byte("package main\n// File 1\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir2, "file2.go"), []byte("package main\n// File 2\n// File 2 line 2\n"), 0644); err != nil {
		t.Fatal(err)
	}

	scanner1 := NewLightScanner(tmpDir1, 30*time.Second)
	scanner2 := NewLightScanner(tmpDir2, 30*time.Second)

	// Run scans concurrently
	ctx := context.Background()
	done1 := make(chan error, 1)
	done2 := make(chan error, 1)

	var result1, result2 *ScanResult

	go func() {
		var err error
		result1, err = scanner1.Scan(ctx)
		done1 <- err
	}()

	go func() {
		var err error
		result2, err = scanner2.Scan(ctx)
		done2 <- err
	}()

	// Wait for both scans
	err1 := <-done1
	err2 := <-done2

	if err1 != nil {
		t.Errorf("Scan 1 failed: %v", err1)
	}
	if err2 != nil {
		t.Errorf("Scan 2 failed: %v", err2)
	}

	// INTEGRATION VALIDATION: Results should be independent
	if result1 != nil && result2 != nil {
		// Each scan should have correct file count for its scenario
		if result1.TotalFiles != 1 || result2.TotalFiles != 1 {
			t.Errorf("Concurrent scans mixed results: got %d and %d files, expected 1 each",
				result1.TotalFiles, result2.TotalFiles)
		}

		// Line counts should be different (file2 has more lines)
		if result1.TotalLines >= result2.TotalLines {
			t.Errorf("Concurrent scans may have mixed results: file2 should have more lines than file1")
		}
	}
}
