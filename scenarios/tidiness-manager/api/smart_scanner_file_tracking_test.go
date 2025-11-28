package main

import (
	"fmt"
	"sync"
	"testing"
)

// Test helper: creates a scanner with default config
func newTestScanner(t *testing.T) *SmartScanner {
	t.Helper()
	scanner, err := NewSmartScanner(GetDefaultSmartScanConfig())
	if err != nil {
		t.Fatalf("Failed to create scanner: %v", err)
	}
	return scanner
}

// Test helper: verifies files are marked as analyzed
func assertFilesAnalyzed(t *testing.T, scanner *SmartScanner, files []string, shouldBeAnalyzed bool) {
	t.Helper()
	for _, file := range files {
		if scanner.isFileAnalyzed(file) != shouldBeAnalyzed {
			if shouldBeAnalyzed {
				t.Errorf("File %q should be marked as analyzed", file)
			} else {
				t.Errorf("File %q should NOT be marked as analyzed", file)
			}
		}
	}
}

// [REQ:TM-SS-007] Test session-based file tracking
func TestSmartScannerFileTracking(t *testing.T) {
	scanner := newTestScanner(t)

	testFiles := []string{
		"api/main.go",
		"api/handlers.go",
		"ui/src/App.tsx",
	}

	// Initially, no files should be marked as analyzed
	assertFilesAnalyzed(t, scanner, testFiles, false)

	// Mark files as analyzed
	for _, file := range testFiles {
		scanner.markFileAnalyzed(file)
	}

	// Verify files are now marked as analyzed
	assertFilesAnalyzed(t, scanner, testFiles, true)

	// Verify different file is not marked
	if scanner.isFileAnalyzed("api/unknown.go") {
		t.Error("Unmarked file should not be analyzed")
	}
}

// [REQ:TM-SS-007] Test concurrent file tracking safety
func TestSmartScannerConcurrentFileTracking(t *testing.T) {
	scanner := newTestScanner(t)

	// Test concurrent marking and reading
	const numGoroutines = 10
	const filesPerGoroutine = 100

	var wg sync.WaitGroup
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < filesPerGoroutine; j++ {
				file := fmt.Sprintf("file-%d-%d.go", id, j)
				scanner.markFileAnalyzed(file)
				// Verify it's marked
				if !scanner.isFileAnalyzed(file) {
					t.Errorf("File %s should be marked as analyzed", file)
				}
			}
		}(i)
	}

	wg.Wait()
}

// [REQ:TM-SS-007] Test file tracking with special characters
func TestSmartScannerFileTracking_SpecialChars(t *testing.T) {
	scanner := newTestScanner(t)

	specialFiles := []string{
		"api/file-with-dashes.go",
		"api/file_with_underscores.go",
		"ui/Component.Name.tsx",
		"docs/file with spaces.md",
		"深圳/中文文件.go",
		"emoji/file🚀.ts",
	}

	// Mark all files
	for _, file := range specialFiles {
		scanner.markFileAnalyzed(file)
	}

	// Verify all marked correctly
	assertFilesAnalyzed(t, scanner, specialFiles, true)
}

// [REQ:TM-SS-007] Test file tracking clear/reset
func TestSmartScannerFileTracking_Reset(t *testing.T) {
	scanner := newTestScanner(t)

	// Mark some files
	files := []string{"a.go", "b.go", "c.go"}
	for _, file := range files {
		scanner.markFileAnalyzed(file)
	}

	// Verify marked
	assertFilesAnalyzed(t, scanner, files, true)

	// Create new session (simulates reset)
	scanner.sessionID = generateSessionID()
	scanner.analyzedFiles = make(map[string]bool)

	// Verify files no longer marked
	assertFilesAnalyzed(t, scanner, files, false)
}
