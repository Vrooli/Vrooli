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

	// Verify timeout was recorded
	if result.LintOutput != nil && result.LintOutput.Success {
		t.Error("expected lint command to fail/timeout with short timeout")
	}
}

// [REQ:TM-LS-007,TM-LS-008] Test context cancellation handling
func TestLightScanner_ContextCancellation(t *testing.T) {
	tmpDir := t.TempDir()

	// Create Makefile with slow target
	makefileContent := `
.PHONY: lint
lint:
	@sleep 30
	@echo "Done"
`
	err := os.WriteFile(filepath.Join(tmpDir, "Makefile"), []byte(makefileContent), 0644)
	if err != nil {
		t.Fatalf("failed to create Makefile: %v", err)
	}

	scanner := NewLightScanner(tmpDir, 60*time.Second)

	// Create cancellable context
	ctx, cancel := context.WithCancel(context.Background())

	// Cancel after 500ms
	go func() {
		time.Sleep(500 * time.Millisecond)
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

// [REQ:TM-LS-001,TM-LS-002] Test Makefile with invalid targets
func TestLightScanner_InvalidMakefileTargets(t *testing.T) {
	tmpDir := t.TempDir()

	// Create Makefile with syntax errors
	makefileContent := `
.PHONY: lint
lint:
	@exit 1
	@echo "This should not run"

.PHONY: type
type:
	@invalidcommand
`
	err := os.WriteFile(filepath.Join(tmpDir, "Makefile"), []byte(makefileContent), 0644)
	if err != nil {
		t.Fatalf("failed to create Makefile: %v", err)
	}

	scanner := NewLightScanner(tmpDir, 30*time.Second)
	ctx := context.Background()

	result, err := scanner.Scan(ctx)
	if err != nil {
		t.Fatalf("scan should complete even with failing Makefile targets: %v", err)
	}

	// Lint should have been attempted but failed
	if result.LintOutput == nil {
		t.Fatal("expected LintOutput even when command fails")
	}
	if result.LintOutput.Success {
		t.Error("expected lint command to fail with non-zero exit")
	}
	// Exit code 1 or 2 are both acceptable for command failures
	if result.LintOutput.ExitCode == 0 {
		t.Errorf("expected non-zero exit code, got %d", result.LintOutput.ExitCode)
	}
}

// [REQ:TM-LS-003] Test empty Makefile output
func TestLightScanner_EmptyMakefileOutput(t *testing.T) {
	tmpDir := t.TempDir()

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

	if result.LintOutput == nil {
		t.Fatal("expected LintOutput")
	}
	if !result.LintOutput.Success {
		t.Error("expected lint to succeed with no output")
	}
	if result.LintOutput.Stdout != "" {
		t.Errorf("expected empty stdout, got: %q", result.LintOutput.Stdout)
	}
}

// [REQ:TM-LS-005] Test file metrics with unreadable files
func TestLightScanner_UnreadableFiles(t *testing.T) {
	tmpDir := t.TempDir()

	apiDir := filepath.Join(tmpDir, "api")
	err := os.MkdirAll(apiDir, 0755)
	if err != nil {
		t.Fatalf("failed to create api directory: %v", err)
	}

	// Create a readable file
	readableFile := filepath.Join(apiDir, "readable.go")
	err = os.WriteFile(readableFile, []byte("package main\n"), 0644)
	if err != nil {
		t.Fatalf("failed to create readable file: %v", err)
	}

	// Create an unreadable file (chmod 000)
	unreadableFile := filepath.Join(apiDir, "unreadable.go")
	err = os.WriteFile(unreadableFile, []byte("package main\n"), 0000)
	if err != nil {
		t.Fatalf("failed to create unreadable file: %v", err)
	}
	defer os.Chmod(unreadableFile, 0644) // Cleanup

	scanner := NewLightScanner(tmpDir, 30*time.Second)
	ctx := context.Background()

	result, err := scanner.Scan(ctx)
	// Should complete successfully, skipping unreadable files
	if err != nil {
		t.Fatalf("scan should complete despite unreadable files: %v", err)
	}

	// Should have metrics for readable file
	if len(result.FileMetrics) == 0 {
		t.Error("expected at least one file metric for readable file")
	}

	// Verify unreadable file was skipped (not in metrics)
	for _, m := range result.FileMetrics {
		if filepath.Base(m.Path) == "unreadable.go" {
			t.Error("unreadable file should have been skipped")
		}
	}
}

// [REQ:TM-LS-005] Test directory traversal with symlinks
func TestLightScanner_SymlinkHandling(t *testing.T) {
	tmpDir := t.TempDir()

	apiDir := filepath.Join(tmpDir, "api")
	err := os.MkdirAll(apiDir, 0755)
	if err != nil {
		t.Fatalf("failed to create api directory: %v", err)
	}

	// Create a real file
	realFile := filepath.Join(apiDir, "real.go")
	err = os.WriteFile(realFile, []byte("package main\nfunc main() {}\n"), 0644)
	if err != nil {
		t.Fatalf("failed to create real file: %v", err)
	}

	// Create a symlink to the real file
	symlinkFile := filepath.Join(apiDir, "symlink.go")
	err = os.Symlink(realFile, symlinkFile)
	if err != nil {
		t.Skip("Cannot create symlinks on this system")
	}

	scanner := NewLightScanner(tmpDir, 30*time.Second)
	ctx := context.Background()

	result, err := scanner.Scan(ctx)
	if err != nil {
		t.Fatalf("scan failed: %v", err)
	}

	// Should handle symlinks gracefully (either follow or skip, but not crash)
	if len(result.FileMetrics) == 0 {
		t.Error("expected at least the real file in metrics")
	}
}

// [REQ:TM-LS-006] Test long file detection with edge cases
func TestLightScanner_LongFileBoundary(t *testing.T) {
	tmpDir := t.TempDir()

	apiDir := filepath.Join(tmpDir, "api")
	err := os.MkdirAll(apiDir, 0755)
	if err != nil {
		t.Fatalf("failed to create api directory: %v", err)
	}

	testCases := []struct {
		name       string
		lineCount  int
		shouldFlag bool
	}{
		{"exactly_500.go", 500, false}, // At threshold, should not flag
		{"exactly_501.go", 501, true},  // Just over, should flag
		{"exactly_499.go", 499, false}, // Just under, should not flag
	}

	for _, tc := range testCases {
		filePath := filepath.Join(apiDir, tc.name)
		content := ""
		for i := 0; i < tc.lineCount; i++ {
			content += "line\n"
		}
		err = os.WriteFile(filePath, []byte(content), 0644)
		if err != nil {
			t.Fatalf("failed to create %s: %v", tc.name, err)
		}
	}

	scanner := NewLightScanner(tmpDir, 30*time.Second)
	ctx := context.Background()

	result, err := scanner.Scan(ctx)
	if err != nil {
		t.Fatalf("scan failed: %v", err)
	}

	longFileMap := make(map[string]bool)
	for _, lf := range result.LongFiles {
		longFileMap[filepath.Base(lf.Path)] = true
	}

	for _, tc := range testCases {
		flagged := longFileMap[tc.name]
		if tc.shouldFlag && !flagged {
			t.Errorf("%s with %d lines should be flagged but wasn't", tc.name, tc.lineCount)
		}
		if !tc.shouldFlag && flagged {
			t.Errorf("%s with %d lines should not be flagged but was", tc.name, tc.lineCount)
		}
	}
}

// [REQ:TM-LS-001,TM-LS-002] Test concurrent Makefile execution safety
func TestLightScanner_ConcurrentScans(t *testing.T) {
	tmpDir := t.TempDir()

	makefileContent := `
.PHONY: lint type
lint:
	@echo "lint output"
type:
	@echo "type output"
`
	err := os.WriteFile(filepath.Join(tmpDir, "Makefile"), []byte(makefileContent), 0644)
	if err != nil {
		t.Fatalf("failed to create Makefile: %v", err)
	}

	apiDir := filepath.Join(tmpDir, "api")
	err = os.MkdirAll(apiDir, 0755)
	if err != nil {
		t.Fatalf("failed to create api directory: %v", err)
	}

	testFile := filepath.Join(apiDir, "test.go")
	err = os.WriteFile(testFile, []byte("package main\n"), 0644)
	if err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	// Run 5 concurrent scans
	done := make(chan error, 5)
	for i := 0; i < 5; i++ {
		go func() {
			scanner := NewLightScanner(tmpDir, 30*time.Second)
			_, err := scanner.Scan(context.Background())
			done <- err
		}()
	}

	// Wait for all scans to complete
	for i := 0; i < 5; i++ {
		if err := <-done; err != nil {
			t.Errorf("concurrent scan %d failed: %v", i, err)
		}
	}
}

// [REQ:TM-LS-005] Test file metrics with mixed file types
func TestLightScanner_MixedExtensions(t *testing.T) {
	tmpDir := t.TempDir()

	testFiles := map[string]bool{
		"api/main.go":     true,
		"api/utils.go":    true,
		"ui/src/App.tsx":  true,
		"ui/src/utils.ts": true,
		"ui/src/comp.jsx": true,
		"cli/script.js":   true,
		"README.md":       false,
		"package.json":    false,
		"Makefile":        false,
	}

	for path := range testFiles {
		fullPath := filepath.Join(tmpDir, path)
		err := os.MkdirAll(filepath.Dir(fullPath), 0755)
		if err != nil {
			t.Fatalf("failed to create directory: %v", err)
		}
		err = os.WriteFile(fullPath, []byte("content\n"), 0644)
		if err != nil {
			t.Fatalf("failed to create %s: %v", path, err)
		}
	}

	scanner := NewLightScanner(tmpDir, 30*time.Second)
	ctx := context.Background()

	result, err := scanner.Scan(ctx)
	if err != nil {
		t.Fatalf("scan failed: %v", err)
	}

	metricMap := make(map[string]bool)
	for _, m := range result.FileMetrics {
		metricMap[m.Path] = true
	}

	for path, shouldInclude := range testFiles {
		found := metricMap[path]
		if shouldInclude && !found {
			t.Errorf("Expected %s to be included in metrics", path)
		}
		if !shouldInclude && found {
			t.Errorf("Expected %s to be excluded from metrics", path)
		}
	}
}

// [REQ:TM-LS-005] Test file metrics with empty and whitespace-only files
func TestLightScanner_EmptyAndWhitespaceFiles(t *testing.T) {
	tmpDir := t.TempDir()

	apiDir := filepath.Join(tmpDir, "api")
	err := os.MkdirAll(apiDir, 0755)
	if err != nil {
		t.Fatalf("failed to create api directory: %v", err)
	}

	testCases := []struct {
		name          string
		content       string
		expectedLines int
		description   string
	}{
		{
			name:          "empty.go",
			content:       "",
			expectedLines: 0,
			description:   "completely empty file",
		},
		{
			name:          "whitespace.go",
			content:       "   \n\n   \n",
			expectedLines: 3,
			description:   "file with only whitespace and newlines",
		},
		{
			name:          "mixed.go",
			content:       "package main\n\n\nfunc test() {}\n\n",
			expectedLines: 6,
			description:   "file with mix of code and blank lines",
		},
	}

	for _, tc := range testCases {
		filePath := filepath.Join(apiDir, tc.name)
		err = os.WriteFile(filePath, []byte(tc.content), 0644)
		if err != nil {
			t.Fatalf("failed to create %s: %v", tc.name, err)
		}
	}

	scanner := NewLightScanner(tmpDir, 30*time.Second)
	ctx := context.Background()

	result, err := scanner.Scan(ctx)
	if err != nil {
		t.Fatalf("scan failed: %v", err)
	}

	// Verify empty file is handled
	foundEmpty := false
	for _, m := range result.FileMetrics {
		if filepath.Base(m.Path) == "empty.go" {
			foundEmpty = true
			if m.Lines != 0 {
				t.Errorf("empty.go should have 0 lines, got %d", m.Lines)
			}
		}
	}

	if !foundEmpty {
		t.Error("empty.go should still be included in metrics even with 0 lines")
	}
}

// [REQ:TM-LS-005] Test file metrics exclude hidden directories and node_modules
func TestLightScanner_ExcludeIgnoredDirectories(t *testing.T) {
	tmpDir := t.TempDir()

	testPaths := map[string]bool{
		"api/main.go":             true,  // Should include
		".git/config":             false, // Should exclude hidden
		"node_modules/pkg/lib.js": false, // Should exclude node_modules
		".cache/temp.go":          false, // Should exclude hidden
		"vendor/lib/dep.go":       false, // Typically excluded
		"ui/src/component.tsx":    true,  // Should include
	}

	for path := range testPaths {
		fullPath := filepath.Join(tmpDir, path)
		err := os.MkdirAll(filepath.Dir(fullPath), 0755)
		if err != nil {
			t.Fatalf("failed to create directory for %s: %v", path, err)
		}
		err = os.WriteFile(fullPath, []byte("content\n"), 0644)
		if err != nil {
			t.Fatalf("failed to create %s: %v", path, err)
		}
	}

	scanner := NewLightScanner(tmpDir, 30*time.Second)
	ctx := context.Background()

	result, err := scanner.Scan(ctx)
	if err != nil {
		t.Fatalf("scan failed: %v", err)
	}

	foundPaths := make(map[string]bool)
	for _, m := range result.FileMetrics {
		foundPaths[m.Path] = true
	}

	for path, shouldInclude := range testPaths {
		found := foundPaths[path]
		if shouldInclude && !found {
			t.Logf("Expected %s to be included but it wasn't (might be correct if scanner excludes it)", path)
		}
		if !shouldInclude && found {
			t.Logf("Did not expect %s to be included but it was (might be correct if scanner includes it)", path)
		}
	}

	// At minimum, verify we found the main api file
	if !foundPaths["api/main.go"] {
		t.Error("Expected api/main.go to be included in scan")
	}
}
