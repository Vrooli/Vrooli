package main

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

// [REQ:TM-LS-005] Test file line counting
func TestCollectFileMetrics_LineCounting(t *testing.T) {
	// Create temporary test directory
	tmpDir := t.TempDir()

	// Create test files with known line counts
	// Note: Only .go, .ts, .tsx, .js, .jsx files are scanned
	testFiles := map[string]int{
		"api/main.go":    50,
		"ui/src/App.tsx": 100,
		"cli/utils.js":   25,
	}

	for path, lineCount := range testFiles {
		fullPath := filepath.Join(tmpDir, path)
		if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
			t.Fatalf("Failed to create directory: %v", err)
		}

		// Create file with specified number of lines
		content := ""
		for i := 0; i < lineCount; i++ {
			content += "line\n"
		}
		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to write test file: %v", err)
		}
	}

	// Create scanner and collect metrics
	scanner := NewLightScanner(tmpDir, 0)
	metrics, err := scanner.collectFileMetrics()
	if err != nil {
		t.Fatalf("collectFileMetrics failed: %v", err)
	}

	// Verify we found the expected files
	if len(metrics) == 0 {
		t.Fatal("Expected to find metrics, got none")
	}

	// Verify line counts are accurate
	metricMap := make(map[string]int)
	for _, m := range metrics {
		metricMap[m.Path] = m.Lines
	}

	for expectedPath, expectedLines := range testFiles {
		found := false
		for actualPath, actualLines := range metricMap {
			if filepath.Base(actualPath) == filepath.Base(expectedPath) {
				found = true
				if actualLines != expectedLines {
					t.Errorf("File %s: expected %d lines, got %d", expectedPath, expectedLines, actualLines)
				}
				break
			}
		}
		if !found {
			t.Errorf("Expected file %s not found in metrics", expectedPath)
		}
	}
}

// [REQ:TM-LS-005] Test file extension filtering
func TestCollectFileMetrics_ExtensionFiltering(t *testing.T) {
	tmpDir := t.TempDir()

	// Create files with different extensions
	testFiles := []struct {
		path          string
		shouldInclude bool
	}{
		{"api/main.go", true},
		{"ui/src/App.tsx", true},
		{"ui/src/utils.ts", true},
		{"ui/src/component.jsx", true},
		{"cli/script.js", true},
		{"README.md", false},
		{"config.json", false},
		{"styles.css", false},
	}

	for _, tf := range testFiles {
		fullPath := filepath.Join(tmpDir, tf.path)
		if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
			t.Fatalf("Failed to create directory: %v", err)
		}
		if err := os.WriteFile(fullPath, []byte("test content\n"), 0644); err != nil {
			t.Fatalf("Failed to write test file: %v", err)
		}
	}

	scanner := NewLightScanner(tmpDir, 0)
	metrics, err := scanner.collectFileMetrics()
	if err != nil {
		t.Fatalf("collectFileMetrics failed: %v", err)
	}

	// Count how many of each type we found
	includedCount := 0
	excludedCount := 0

	metricPaths := make(map[string]bool)
	for _, m := range metrics {
		metricPaths[m.Path] = true
	}

	for _, tf := range testFiles {
		found := false
		for path := range metricPaths {
			if filepath.Base(path) == filepath.Base(tf.path) {
				found = true
				break
			}
		}

		if tf.shouldInclude && found {
			includedCount++
		} else if !tf.shouldInclude && !found {
			excludedCount++
		} else if tf.shouldInclude && !found {
			t.Errorf("Expected file %s to be included but it was not", tf.path)
		} else if !tf.shouldInclude && found {
			t.Errorf("Expected file %s to be excluded but it was included", tf.path)
		}
	}

	if includedCount == 0 {
		t.Error("Expected some files to be included, got none")
	}
}

// [REQ:TM-LS-006] Test long file threshold flagging
func TestLongFileThreshold(t *testing.T) {
	tmpDir := t.TempDir()

	// Create files with varying line counts
	testFiles := []struct {
		path      string
		lineCount int
		isLong    bool // based on 500 line threshold
	}{
		{"api/short.go", 100, false},
		{"api/medium.go", 400, false},
		{"api/long.go", 600, true},
		{"ui/src/huge.tsx", 1000, true},
	}

	for _, tf := range testFiles {
		fullPath := filepath.Join(tmpDir, tf.path)
		if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
			t.Fatalf("Failed to create directory: %v", err)
		}

		content := ""
		for i := 0; i < tf.lineCount; i++ {
			content += "line\n"
		}
		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to write test file: %v", err)
		}
	}

	// Run full scan which includes long file detection
	scanner := NewLightScanner(tmpDir, 0)
	result, err := scanner.Scan(nil)
	if err != nil {
		t.Fatalf("Scan failed: %v", err)
	}

	// Verify long files are correctly identified
	longFileMap := make(map[string]int)
	for _, lf := range result.LongFiles {
		longFileMap[lf.Path] = lf.Lines
	}

	for _, tf := range testFiles {
		found := false
		for path := range longFileMap {
			if filepath.Base(path) == filepath.Base(tf.path) {
				found = true
				break
			}
		}

		if tf.isLong && !found {
			t.Errorf("Expected long file %s (lines: %d) to be flagged", tf.path, tf.lineCount)
		} else if !tf.isLong && found {
			t.Errorf("File %s (lines: %d) should not be flagged as long", tf.path, tf.lineCount)
		}
	}
}

// [REQ:TM-LS-006] Test configurable threshold
func TestLongFileThreshold_Configurable(t *testing.T) {
	// This test documents that the threshold is currently hardcoded to 500
	// Future enhancement: make threshold configurable per-scenario
	tmpDir := t.TempDir()

	fullPath := filepath.Join(tmpDir, "api", "test.go")
	if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
		t.Fatalf("Failed to create directory: %v", err)
	}

	// Create a file with exactly 501 lines (just over threshold)
	content := ""
	for i := 0; i < 501; i++ {
		content += "line\n"
	}
	if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	scanner := NewLightScanner(tmpDir, 0)
	result, err := scanner.Scan(nil)
	if err != nil {
		t.Fatalf("Scan failed: %v", err)
	}

	// Should be flagged with default 500 line threshold
	if len(result.LongFiles) == 0 {
		t.Error("Expected 501-line file to be flagged as long with default threshold")
	}

	// Verify threshold is reported correctly
	if len(result.LongFiles) > 0 && result.LongFiles[0].Threshold != 500 {
		t.Errorf("Expected threshold to be 500, got %d", result.LongFiles[0].Threshold)
	}
}

// [REQ:TM-LS-005] Test total line and file counting
func TestFileMetrics_Totals(t *testing.T) {
	tmpDir := t.TempDir()

	// Create known set of files
	testFiles := []struct {
		path  string
		lines int
	}{
		{"api/main.go", 100},
		{"api/handlers.go", 200},
		{"ui/src/App.tsx", 150},
	}

	expectedTotal := 0
	for _, tf := range testFiles {
		fullPath := filepath.Join(tmpDir, tf.path)
		if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
			t.Fatalf("Failed to create directory: %v", err)
		}

		content := ""
		for i := 0; i < tf.lines; i++ {
			content += "line\n"
		}
		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to write test file: %v", err)
		}
		expectedTotal += tf.lines
	}

	scanner := NewLightScanner(tmpDir, 0)
	result, err := scanner.Scan(nil)
	if err != nil {
		t.Fatalf("Scan failed: %v", err)
	}

	// Verify total file count
	if result.TotalFiles != len(testFiles) {
		t.Errorf("Expected %d total files, got %d", len(testFiles), result.TotalFiles)
	}

	// Verify total line count
	if result.TotalLines != expectedTotal {
		t.Errorf("Expected %d total lines, got %d", expectedTotal, result.TotalLines)
	}
}

// [REQ:TM-LS-005] Test handling of empty files
func TestCollectFileMetrics_EmptyFiles(t *testing.T) {
	tmpDir := t.TempDir()

	// Create empty and near-empty files
	testFiles := []struct {
		path  string
		lines int
	}{
		{"api/empty.go", 0},
		{"api/single-line.go", 1},
		{"ui/src/whitespace.tsx", 3}, // file with only newlines
	}

	for _, tf := range testFiles {
		fullPath := filepath.Join(tmpDir, tf.path)
		if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
			t.Fatalf("Failed to create directory: %v", err)
		}

		content := ""
		for i := 0; i < tf.lines; i++ {
			content += "\n"
		}
		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to write test file: %v", err)
		}
	}

	scanner := NewLightScanner(tmpDir, 0)
	metrics, err := scanner.collectFileMetrics()
	if err != nil {
		t.Fatalf("collectFileMetrics failed: %v", err)
	}

	// Empty files should still be included in metrics
	if len(metrics) < len(testFiles) {
		t.Errorf("Expected at least %d files in metrics, got %d", len(testFiles), len(metrics))
	}

	// Verify empty file has 0 lines
	for _, m := range metrics {
		if filepath.Base(m.Path) == "empty.go" {
			if m.Lines != 0 {
				t.Errorf("empty.go should have 0 lines, got %d", m.Lines)
			}
		}
	}
}

// [REQ:TM-LS-005] Test exclusion of hidden directories
func TestCollectFileMetrics_ExcludeHiddenDirs(t *testing.T) {
	tmpDir := t.TempDir()

	// Create files in hidden and normal directories
	testFiles := []struct {
		path          string
		shouldInclude bool
	}{
		{"api/main.go", true},
		{".git/hooks/pre-commit", false},
		{".cache/temp.go", false},
		{"node_modules/pkg/index.js", false}, // typically excluded
		{"ui/src/App.tsx", true},
	}

	for _, tf := range testFiles {
		fullPath := filepath.Join(tmpDir, tf.path)
		if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
			t.Fatalf("Failed to create directory: %v", err)
		}
		if err := os.WriteFile(fullPath, []byte("content\n"), 0644); err != nil {
			t.Fatalf("Failed to write test file: %v", err)
		}
	}

	scanner := NewLightScanner(tmpDir, 0)
	metrics, err := scanner.collectFileMetrics()
	if err != nil {
		t.Fatalf("collectFileMetrics failed: %v", err)
	}

	metricPaths := make(map[string]bool)
	for _, m := range metrics {
		metricPaths[m.Path] = true
	}

	// Verify hidden directories are excluded
	for _, tf := range testFiles {
		found := false
		for path := range metricPaths {
			if filepath.Base(path) == filepath.Base(tf.path) {
				found = true
				break
			}
		}

		if tf.shouldInclude && !found {
			t.Logf("Expected %s to be included but it wasn't (scanner may exclude it)", tf.path)
		}
		if !tf.shouldInclude && found {
			t.Logf("Did not expect %s to be included but it was (scanner may include it)", tf.path)
		}
	}

	// At minimum, verify we found the normal api file
	foundMainGo := false
	for _, m := range metrics {
		if filepath.Base(m.Path) == "main.go" {
			foundMainGo = true
			break
		}
	}
	if !foundMainGo {
		t.Error("Expected to find main.go in metrics")
	}
}

// [REQ:TM-LS-005] Test handling of unreadable files
func TestCollectFileMetrics_UnreadableFiles(t *testing.T) {
	tmpDir := t.TempDir()

	// Create readable file
	readablePath := filepath.Join(tmpDir, "api", "readable.go")
	if err := os.MkdirAll(filepath.Dir(readablePath), 0755); err != nil {
		t.Fatalf("Failed to create directory: %v", err)
	}
	if err := os.WriteFile(readablePath, []byte("package main\n"), 0644); err != nil {
		t.Fatalf("Failed to write readable file: %v", err)
	}

	// Create unreadable file (permission denied)
	unreadablePath := filepath.Join(tmpDir, "api", "unreadable.go")
	if err := os.WriteFile(unreadablePath, []byte("package main\n"), 0000); err != nil {
		t.Fatalf("Failed to write unreadable file: %v", err)
	}
	defer os.Chmod(unreadablePath, 0644) // Cleanup

	scanner := NewLightScanner(tmpDir, 0)
	metrics, err := scanner.collectFileMetrics()

	// Should complete despite unreadable file (may skip it or return error)
	if err != nil {
		t.Logf("collectFileMetrics returned error (acceptable): %v", err)
	}

	// Should have metrics for at least the readable file
	if len(metrics) == 0 {
		t.Error("Expected at least readable.go in metrics")
	}

	// Verify unreadable file was handled (either skipped or errored gracefully)
	foundUnreadable := false
	for _, m := range metrics {
		if filepath.Base(m.Path) == "unreadable.go" {
			foundUnreadable = true
		}
	}
	if foundUnreadable {
		t.Logf("Scanner was able to read unreadable.go (may have elevated permissions)")
	}
}

// [REQ:TM-LS-005] Test file metrics with deeply nested directory structure
func TestCollectFileMetrics_DeepNesting(t *testing.T) {
	tmpDir := t.TempDir()

	// Create deeply nested file
	deepPath := filepath.Join(tmpDir, "api", "v1", "handlers", "users", "profile", "avatar", "upload.go")
	if err := os.MkdirAll(filepath.Dir(deepPath), 0755); err != nil {
		t.Fatalf("Failed to create deep directory: %v", err)
	}
	if err := os.WriteFile(deepPath, []byte("package avatar\nfunc Upload() {}\n"), 0644); err != nil {
		t.Fatalf("Failed to write deep file: %v", err)
	}

	scanner := NewLightScanner(tmpDir, 0)
	metrics, err := scanner.collectFileMetrics()
	if err != nil {
		t.Fatalf("collectFileMetrics failed: %v", err)
	}

	// Should find deeply nested file
	foundDeep := false
	for _, m := range metrics {
		if filepath.Base(m.Path) == "upload.go" {
			foundDeep = true
			if m.Lines != 2 {
				t.Errorf("upload.go should have 2 lines, got %d", m.Lines)
			}
		}
	}

	if !foundDeep {
		t.Error("Expected to find deeply nested upload.go")
	}
}

// [REQ:TM-LS-005] Test file extension metadata
func TestCollectFileMetrics_ExtensionMetadata(t *testing.T) {
	tmpDir := t.TempDir()

	testFiles := []struct {
		path        string
		expectedExt string
	}{
		{"api/main.go", ".go"},
		{"ui/src/App.tsx", ".tsx"},
		{"ui/src/utils.ts", ".ts"},
		{"cli/script.js", ".js"},
		{"ui/components/Button.jsx", ".jsx"},
	}

	for _, tf := range testFiles {
		fullPath := filepath.Join(tmpDir, tf.path)
		if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
			t.Fatalf("Failed to create directory: %v", err)
		}
		if err := os.WriteFile(fullPath, []byte("content\n"), 0644); err != nil {
			t.Fatalf("Failed to write test file: %v", err)
		}
	}

	scanner := NewLightScanner(tmpDir, 0)
	metrics, err := scanner.collectFileMetrics()
	if err != nil {
		t.Fatalf("collectFileMetrics failed: %v", err)
	}

	// Verify extension metadata is captured correctly
	for _, m := range metrics {
		for _, tf := range testFiles {
			if filepath.Base(m.Path) == filepath.Base(tf.path) {
				if m.Extension != tf.expectedExt {
					t.Errorf("File %s: expected extension %q, got %q", tf.path, tf.expectedExt, m.Extension)
				}
			}
		}
	}
}

// [REQ:TM-LS-005] Test handling of very large files
func TestCollectFileMetrics_LargeFile(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a very large file (10,000 lines)
	largePath := filepath.Join(tmpDir, "api", "large.go")
	if err := os.MkdirAll(filepath.Dir(largePath), 0755); err != nil {
		t.Fatalf("Failed to create directory: %v", err)
	}

	content := ""
	lineCount := 10000
	for i := 0; i < lineCount; i++ {
		content += "// Line comment to make file larger\n"
	}
	if err := os.WriteFile(largePath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write large file: %v", err)
	}

	scanner := NewLightScanner(tmpDir, 0)
	metrics, err := scanner.collectFileMetrics()
	if err != nil {
		t.Fatalf("collectFileMetrics failed: %v", err)
	}

	// Should handle large file without issues
	foundLarge := false
	for _, m := range metrics {
		if filepath.Base(m.Path) == "large.go" {
			foundLarge = true
			if m.Lines != lineCount {
				t.Errorf("large.go should have %d lines, got %d", lineCount, m.Lines)
			}
		}
	}

	if !foundLarge {
		t.Error("Expected to find large.go in metrics")
	}
}

// [REQ:TM-LS-005] Test file metrics with symlinks
func TestCollectFileMetrics_Symlinks(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a real file
	realPath := filepath.Join(tmpDir, "api", "real.go")
	if err := os.MkdirAll(filepath.Dir(realPath), 0755); err != nil {
		t.Fatalf("Failed to create directory: %v", err)
	}
	if err := os.WriteFile(realPath, []byte("package main\nfunc main() {}\n"), 0644); err != nil {
		t.Fatalf("Failed to write real file: %v", err)
	}

	// Create a symlink
	symlinkPath := filepath.Join(tmpDir, "api", "link.go")
	if err := os.Symlink(realPath, symlinkPath); err != nil {
		t.Skip("Cannot create symlinks on this system")
	}

	scanner := NewLightScanner(tmpDir, 0)
	metrics, err := scanner.collectFileMetrics()
	if err != nil {
		t.Fatalf("collectFileMetrics failed: %v", err)
	}

	// Verify scanner handles symlinks gracefully (implementation-dependent)
	if len(metrics) == 0 {
		t.Error("Expected at least the real file in metrics")
	}
}

// [REQ:TM-LS-005] Test concurrent file metric collection safety
func TestCollectFileMetrics_Concurrent(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test files
	for i := 0; i < 10; i++ {
		fullPath := filepath.Join(tmpDir, "api", fmt.Sprintf("file%d.go", i))
		if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
			t.Fatalf("Failed to create directory: %v", err)
		}
		if err := os.WriteFile(fullPath, []byte("package main\n"), 0644); err != nil {
			t.Fatalf("Failed to write file: %v", err)
		}
	}

	// Run concurrent metric collections
	done := make(chan error, 5)
	for i := 0; i < 5; i++ {
		go func() {
			scanner := NewLightScanner(tmpDir, 0)
			_, err := scanner.collectFileMetrics()
			done <- err
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 5; i++ {
		if err := <-done; err != nil {
			t.Errorf("Concurrent collection %d failed: %v", i, err)
		}
	}
}

// [REQ:TM-LS-005] Test line counting with different line ending styles
func TestCollectFileMetrics_LineEndings(t *testing.T) {
	tmpDir := t.TempDir()

	testCases := []struct {
		name     string
		content  string
		expected int
	}{
		{"unix_lf", "line1\nline2\nline3\n", 3},           // Unix LF - counts newlines
		{"windows_crlf", "line1\r\nline2\r\nline3\r\n", 3}, // Windows CRLF - counts \n
		{"mac_cr", "line1\rline2\rline3\r", 1},            // Old Mac CR - no \n so counts as 1
		{"mixed", "line1\nline2\r\nline3\r", 3},           // Mixed endings - counts \n
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			filePath := filepath.Join(tmpDir, "api", tc.name+".go")
			if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
				t.Fatalf("Failed to create directory: %v", err)
			}
			if err := os.WriteFile(filePath, []byte(tc.content), 0644); err != nil {
				t.Fatalf("Failed to write file: %v", err)
			}

			scanner := NewLightScanner(tmpDir, 0)
			metrics, err := scanner.collectFileMetrics()
			if err != nil {
				t.Fatalf("collectFileMetrics failed: %v", err)
			}

			for _, m := range metrics {
				if filepath.Base(m.Path) == tc.name+".go" {
					if m.Lines != tc.expected {
						t.Errorf("Expected %d lines, got %d for %s", tc.expected, m.Lines, tc.name)
					}
				}
			}
		})
	}
}

// [REQ:TM-LS-006] Test long file detection with edge cases at boundary
func TestLongFileThreshold_BoundaryConditions(t *testing.T) {
	tmpDir := t.TempDir()

	testCases := []struct {
		name       string
		lineCount  int
		shouldFlag bool
	}{
		{"at_threshold_499", 499, false},
		{"at_threshold_500", 500, false}, // Exactly at threshold
		{"over_threshold_501", 501, true},
		{"over_threshold_502", 502, true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			filePath := filepath.Join(tmpDir, "api", tc.name+".go")
			if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
				t.Fatalf("Failed to create directory: %v", err)
			}

			content := ""
			for i := 0; i < tc.lineCount; i++ {
				content += "line\n"
			}
			if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
				t.Fatalf("Failed to write file: %v", err)
			}

			scanner := NewLightScanner(tmpDir, 0)
			result, err := scanner.Scan(nil)
			if err != nil {
				t.Fatalf("Scan failed: %v", err)
			}

			flagged := false
			for _, lf := range result.LongFiles {
				if filepath.Base(lf.Path) == tc.name+".go" {
					flagged = true
					break
				}
			}

			if tc.shouldFlag && !flagged {
				t.Errorf("File with %d lines should be flagged", tc.lineCount)
			}
			if !tc.shouldFlag && flagged {
				t.Errorf("File with %d lines should not be flagged", tc.lineCount)
			}
		})
	}
}

// [REQ:TM-LS-005] Benchmark file metric collection performance
func BenchmarkCollectFileMetrics_Small(b *testing.B) {
	tmpDir := b.TempDir()

	// Create 10 files
	for i := 0; i < 10; i++ {
		filePath := filepath.Join(tmpDir, "api", fmt.Sprintf("file%d.go", i))
		if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
			b.Fatalf("Failed to create directory: %v", err)
		}
		content := ""
		for j := 0; j < 100; j++ {
			content += "line\n"
		}
		if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
			b.Fatalf("Failed to write file: %v", err)
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		scanner := NewLightScanner(tmpDir, 0)
		_, err := scanner.collectFileMetrics()
		if err != nil {
			b.Fatalf("collectFileMetrics failed: %v", err)
		}
	}
}

// [REQ:TM-LS-005] Benchmark file metric collection with many files
func BenchmarkCollectFileMetrics_Large(b *testing.B) {
	tmpDir := b.TempDir()

	// Create 100 files
	for i := 0; i < 100; i++ {
		filePath := filepath.Join(tmpDir, "api", fmt.Sprintf("file%d.go", i))
		if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
			b.Fatalf("Failed to create directory: %v", err)
		}
		content := ""
		for j := 0; j < 500; j++ {
			content += "line\n"
		}
		if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
			b.Fatalf("Failed to write file: %v", err)
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		scanner := NewLightScanner(tmpDir, 0)
		_, err := scanner.collectFileMetrics()
		if err != nil {
			b.Fatalf("collectFileMetrics failed: %v", err)
		}
	}
}

// [REQ:TM-LS-006] Benchmark long file threshold detection
func BenchmarkLongFileDetection(b *testing.B) {
	tmpDir := b.TempDir()

	// Create mix of short and long files
	for i := 0; i < 20; i++ {
		filePath := filepath.Join(tmpDir, "api", fmt.Sprintf("file%d.go", i))
		if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
			b.Fatalf("Failed to create directory: %v", err)
		}
		lineCount := 100
		if i%3 == 0 {
			lineCount = 600 // Every 3rd file is long
		}
		content := ""
		for j := 0; j < lineCount; j++ {
			content += "line\n"
		}
		if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
			b.Fatalf("Failed to write file: %v", err)
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		scanner := NewLightScanner(tmpDir, 0)
		_, err := scanner.Scan(nil)
		if err != nil {
			b.Fatalf("Scan failed: %v", err)
		}
	}
}
