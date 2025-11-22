package main

import (
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
