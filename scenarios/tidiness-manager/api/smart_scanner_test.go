package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

// Test helpers
func createTestScanner(t *testing.T) *SmartScanner {
	t.Helper()
	config := GetDefaultSmartScanConfig()
	scanner, err := NewSmartScanner(config)
	if err != nil {
		t.Fatalf("Failed to create scanner: %v", err)
	}
	return scanner
}

func mustCreateScanner() *SmartScanner {
	config := GetDefaultSmartScanConfig()
	scanner, err := NewSmartScanner(config)
	if err != nil {
		panic(fmt.Sprintf("Failed to create scanner: %v", err))
	}
	return scanner
}

func setupTempVrooliRoot(t *testing.T, scenarioName string) (vrooliRoot, scenarioDir string, cleanup func()) {
	t.Helper()
	vrooliRoot = t.TempDir()
	scenarioDir = filepath.Join(vrooliRoot, "scenarios", scenarioName)

	if err := os.MkdirAll(scenarioDir, 0755); err != nil {
		t.Fatalf("Failed to create scenario dir: %v", err)
	}

	oldRoot := os.Getenv("VROOLI_ROOT")
	os.Setenv("VROOLI_ROOT", vrooliRoot)

	cleanup = func() {
		os.Setenv("VROOLI_ROOT", oldRoot)
	}

	return vrooliRoot, scenarioDir, cleanup
}

// [REQ:TM-SS-001] Test configurable batch size limits
func TestSmartScannerBatchSizeLimits(t *testing.T) {
	tests := []struct {
		name            string
		fileCount       int
		batchSize       int
		expectedBatches int
		expectedSizes   []int
	}{
		{
			name:            "25 files with batch size 10",
			fileCount:       25,
			batchSize:       10,
			expectedBatches: 3,
			expectedSizes:   []int{10, 10, 5},
		},
		{
			name:            "exact multiple - 20 files with batch size 10",
			fileCount:       20,
			batchSize:       10,
			expectedBatches: 2,
			expectedSizes:   []int{10, 10},
		},
		{
			name:            "single batch - 5 files with batch size 10",
			fileCount:       5,
			batchSize:       10,
			expectedBatches: 1,
			expectedSizes:   []int{5},
		},
		{
			name:            "empty input",
			fileCount:       0,
			batchSize:       10,
			expectedBatches: 0,
			expectedSizes:   []int{},
		},
		{
			name:            "batch size 1",
			fileCount:       3,
			batchSize:       1,
			expectedBatches: 3,
			expectedSizes:   []int{1, 1, 1},
		},
		{
			name:            "large batch size",
			fileCount:       5,
			batchSize:       100,
			expectedBatches: 1,
			expectedSizes:   []int{5},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			files := make([]string, tt.fileCount)
			for i := range files {
				files[i] = "file" + string(rune('0'+i%10))
			}

			batches := createBatches(files, tt.batchSize)

			if len(batches) != tt.expectedBatches {
				t.Errorf("createBatches() created %d batches, want %d", len(batches), tt.expectedBatches)
			}

			for i, expectedSize := range tt.expectedSizes {
				if i >= len(batches) {
					t.Errorf("Missing batch %d", i)
					continue
				}
				if len(batches[i]) != expectedSize {
					t.Errorf("Batch %d size = %d, want %d", i, len(batches[i]), expectedSize)
				}
			}
		})
	}
}

// [REQ:TM-SS-001] Test generateSessionID uniqueness
func TestGenerateSessionID(t *testing.T) {
	id1 := generateSessionID()
	time.Sleep(1100 * time.Millisecond) // Sleep > 1 second to ensure different Unix timestamps
	id2 := generateSessionID()

	if id1 == "" {
		t.Error("Session ID should not be empty")
	}

	if id2 == "" {
		t.Error("Session ID should not be empty")
	}

	// IDs should be different when generated at different times
	if id1 == id2 {
		t.Error("Session IDs generated at different times should be unique")
	}

	// Session ID should have expected format
	expectedPrefix := "smart-scan-"
	if len(id1) < len(expectedPrefix) || id1[:len(expectedPrefix)] != expectedPrefix {
		t.Errorf("Session ID should start with %s, got %s", expectedPrefix, id1)
	}
}

// [REQ:TM-SS-001] Test buildAnalysisPrompt
func TestBuildAnalysisPrompt(t *testing.T) {
	scanner := createTestScanner(t)

	tests := []struct {
		name         string
		fileContents map[string]string
		expectedIn   []string
	}{
		{
			name: "single file",
			fileContents: map[string]string{
				"main.go": "package main\n\nfunc main() {}",
			},
			expectedIn: []string{
				"code tidiness analyzer",
				"main.go",
				"package main",
			},
		},
		{
			name: "multiple files",
			fileContents: map[string]string{
				"main.go": "package main",
				"util.go": "func unused() {}",
			},
			expectedIn: []string{
				"Dead code",
				"Code duplication",
				"complexity",
				"main.go",
				"util.go",
			},
		},
		{
			name:         "empty files",
			fileContents: map[string]string{},
			expectedIn: []string{
				"code tidiness analyzer",
				"Dead code",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prompt := scanner.buildAnalysisPrompt(tt.fileContents)

			for _, expected := range tt.expectedIn {
				if !strings.Contains(prompt, expected) {
					t.Errorf("Prompt should contain %q", expected)
				}
			}

			// Verify JSON format instruction
			if !strings.Contains(prompt, "JSON") {
				t.Error("Prompt should mention JSON format")
			}
		})
	}
}

// [REQ:TM-SS-001] Test readFileContent with valid files
func TestReadFileContent(t *testing.T) {
	scenarioName := "test-scenario"
	_, scenarioDir, cleanup := setupTempVrooliRoot(t, scenarioName)
	defer cleanup()

	tests := []struct {
		name        string
		filePath    string
		content     string
		wantContent string
		wantError   bool
	}{
		{
			name:        "simple go file",
			filePath:    "api/main.go",
			content:     "package main\n\nfunc main() {\n\tprintln(\"hello\")\n}",
			wantContent: "package main\n\nfunc main() {\n\tprintln(\"hello\")\n}",
			wantError:   false,
		},
		{
			name:        "empty file",
			filePath:    "empty.go",
			content:     "",
			wantContent: "",
			wantError:   false,
		},
		{
			name:        "file with unicode",
			filePath:    "unicode.go",
			content:     "// Comment with emoji 🚀\npackage main",
			wantContent: "// Comment with emoji 🚀\npackage main",
			wantError:   false,
		},
	}

	scanner := createTestScanner(t)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test file
			fullPath := filepath.Join(scenarioDir, tt.filePath)
			if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
				t.Fatalf("Failed to create dir: %v", err)
			}
			if err := os.WriteFile(fullPath, []byte(tt.content), 0644); err != nil {
				t.Fatalf("Failed to create test file: %v", err)
			}

			// Read file content
			content, err := scanner.readFileContent(scenarioName, tt.filePath)

			if tt.wantError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if content != tt.wantContent {
				t.Errorf("Content mismatch: got %q, want %q", content, tt.wantContent)
			}
		})
	}
}

// [REQ:TM-SS-001] Test readFileContent error handling
func TestReadFileContent_Errors(t *testing.T) {
	scanner := createTestScanner(t)
	_, _, cleanup := setupTempVrooliRoot(t, "test-scenario")
	defer cleanup()

	tests := []struct {
		name     string
		scenario string
		filePath string
	}{
		{
			name:     "non-existent file",
			scenario: "test-scenario",
			filePath: "nonexistent/file.go",
		},
		{
			name:     "non-existent scenario",
			scenario: "nonexistent-scenario",
			filePath: "api/main.go",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := scanner.readFileContent(tt.scenario, tt.filePath)
			if err == nil {
				t.Error("Expected error for invalid file path")
			}
		})
	}
}

// [REQ:TM-SS-001] Test createBatches with extreme sizes
func TestCreateBatches_ExtremeSizes(t *testing.T) {
	tests := []struct {
		name      string
		fileCount int
		batchSize int
	}{
		{"single file", 1, 1},
		{"single file large batch", 1, 1000},
		{"many small batches", 100, 1},
		{"large files small batches", 1000, 5},
		{"equal distribution", 100, 10},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			files := make([]string, tt.fileCount)
			for i := range files {
				files[i] = fmt.Sprintf("file-%d.go", i)
			}

			batches := createBatches(files, tt.batchSize)

			// Verify no files lost
			totalFiles := 0
			for _, batch := range batches {
				totalFiles += len(batch)
			}
			if totalFiles != tt.fileCount {
				t.Errorf("Lost files: got %d, want %d", totalFiles, tt.fileCount)
			}

			// Verify no batch exceeds size
			for i, batch := range batches {
				if len(batch) > tt.batchSize {
					t.Errorf("Batch %d size %d exceeds limit %d", i, len(batch), tt.batchSize)
				}
			}

			// Verify no duplicate files
			seen := make(map[string]bool)
			for _, batch := range batches {
				for _, file := range batch {
					if seen[file] {
						t.Errorf("Duplicate file in batches: %s", file)
					}
					seen[file] = true
				}
			}
		})
	}
}

// [REQ:TM-SS-001] Test generateSessionID concurrent generation (may have collisions)
func TestGenerateSessionID_Concurrent(t *testing.T) {
	const numGoroutines = 100
	ids := make(chan string, numGoroutines)

	var wg sync.WaitGroup
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ids <- generateSessionID()
		}()
	}

	wg.Wait()
	close(ids)

	// Collect all IDs
	seen := make(map[string]bool)
	for id := range ids {
		seen[id] = true
	}

	// Note: Current implementation uses Unix timestamp, so concurrent calls may generate duplicates
	// This test verifies it doesn't panic and generates valid IDs
	if len(seen) == 0 {
		t.Error("Should generate at least one ID")
	}

	// All IDs should have correct format
	for id := range seen {
		if !strings.HasPrefix(id, "smart-scan-") {
			t.Errorf("Invalid ID format: %s", id)
		}
	}
}

// [REQ:TM-SS-001] Test buildAnalysisPrompt with large files
func TestBuildAnalysisPrompt_LargeFiles(t *testing.T) {
	scanner := createTestScanner(t)

	// Create large file content (10K lines)
	var largeContent strings.Builder
	for i := 0; i < 10000; i++ {
		largeContent.WriteString(fmt.Sprintf("func function%d() { return nil }\n", i))
	}

	fileContents := map[string]string{
		"large.go": largeContent.String(),
	}

	prompt := scanner.buildAnalysisPrompt(fileContents)

	// Prompt should be generated without error
	if len(prompt) == 0 {
		t.Error("Prompt should not be empty for large files")
	}

	// Should contain file name
	if !strings.Contains(prompt, "large.go") {
		t.Error("Prompt should contain file name")
	}
}

// [REQ:TM-SS-001] Test buildAnalysisPrompt with special characters
func TestBuildAnalysisPrompt_SpecialChars(t *testing.T) {
	scanner := createTestScanner(t)

	fileContents := map[string]string{
		"unicode.go":  "// 中文注释\n// Emoji 🚀\npackage main",
		"quotes.go":   "const s = \"string with \\\"quotes\\\"\"",
		"newlines.go": "package main\n\n\n\nfunc test() {}",
	}

	prompt := scanner.buildAnalysisPrompt(fileContents)

	// Should handle special characters
	for filename := range fileContents {
		if !strings.Contains(prompt, filename) {
			t.Errorf("Prompt should contain filename %s", filename)
		}
	}

	if len(prompt) == 0 {
		t.Error("Prompt should not be empty")
	}
}

// [REQ:TM-SS-001] Benchmark createBatches with various sizes
func BenchmarkCreateBatches_Small(b *testing.B) {
	files := make([]string, 10)
	for i := range files {
		files[i] = fmt.Sprintf("file-%d.go", i)
	}
	batchSize := 5

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = createBatches(files, batchSize)
	}
}

func BenchmarkCreateBatches_Large(b *testing.B) {
	files := make([]string, 1000)
	for i := range files {
		files[i] = fmt.Sprintf("file-%d.go", i)
	}
	batchSize := 10

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = createBatches(files, batchSize)
	}
}

func BenchmarkCreateBatches_VeryLarge(b *testing.B) {
	files := make([]string, 10000)
	for i := range files {
		files[i] = fmt.Sprintf("file-%d.go", i)
	}
	batchSize := 50

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = createBatches(files, batchSize)
	}
}

// [REQ:TM-SS-007] Benchmark file tracking operations
func BenchmarkFileTracking_Mark(b *testing.B) {
	scanner := mustCreateScanner()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		scanner.markFileAnalyzed(fmt.Sprintf("file-%d.go", i))
	}
}

func BenchmarkFileTracking_Check(b *testing.B) {
	scanner := mustCreateScanner()

	// Pre-populate with 1000 files
	for i := 0; i < 1000; i++ {
		scanner.markFileAnalyzed(fmt.Sprintf("file-%d.go", i))
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = scanner.isFileAnalyzed(fmt.Sprintf("file-%d.go", i%1000))
	}
}

// [REQ:TM-SS-001] Benchmark generateSessionID
func BenchmarkGenerateSessionID(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = generateSessionID()
	}
}

// [REQ:TM-SS-001] Benchmark buildAnalysisPrompt
func BenchmarkBuildAnalysisPrompt(b *testing.B) {
	scanner := mustCreateScanner()

	fileContents := map[string]string{
		"main.go": "package main\n\nfunc main() {}",
		"util.go": "package main\n\nfunc util() {}",
		"test.go": "package main\n\nfunc test() {}",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = scanner.buildAnalysisPrompt(fileContents)
	}
}

// [REQ:TM-SS-001] Fuzz test SmartScanConfig validation
func FuzzSmartScanConfigValidation(f *testing.F) {
	// Seed corpus
	f.Add(10, 5, 100000)
	f.Add(0, 0, 0)
	f.Add(-1, -1, -1)
	f.Add(1, 1, 1)

	f.Fuzz(func(t *testing.T, maxFiles, maxConcurrent, maxTokens int) {
		config := SmartScanConfig{
			MaxFilesPerBatch: maxFiles,
			MaxConcurrent:    maxConcurrent,
			MaxTokensPerReq:  maxTokens,
		}

		// Should not panic
		valid := config.Validate()

		// Validation rules
		if maxFiles > 0 && maxConcurrent > 0 {
			if !valid {
				t.Errorf("Valid config marked as invalid: %+v", config)
			}
		}
		if maxFiles <= 0 || maxConcurrent <= 0 {
			if valid {
				t.Errorf("Invalid config marked as valid: %+v", config)
			}
		}
	})
}

// [REQ:TM-SS-001] Fuzz test createBatches
func FuzzCreateBatches(f *testing.F) {
	// Seed corpus
	f.Add(10, 5)
	f.Add(0, 1)
	f.Add(100, 10)

	f.Fuzz(func(t *testing.T, fileCount, batchSize int) {
		// Skip invalid inputs
		if fileCount < 0 || fileCount > 10000 {
			t.Skip()
		}
		if batchSize <= 0 || batchSize > 1000 {
			t.Skip()
		}

		files := make([]string, fileCount)
		for i := range files {
			files[i] = fmt.Sprintf("file-%d.go", i)
		}

		// Should not panic
		batches := createBatches(files, batchSize)

		// Verify total files preserved
		totalFiles := 0
		for _, batch := range batches {
			totalFiles += len(batch)
		}
		if totalFiles != fileCount {
			t.Errorf("Lost files: got %d, want %d", totalFiles, fileCount)
		}

		// Verify no batch exceeds size
		for _, batch := range batches {
			if len(batch) > batchSize {
				t.Errorf("Batch size %d exceeds limit %d", len(batch), batchSize)
			}
		}
	})
}
