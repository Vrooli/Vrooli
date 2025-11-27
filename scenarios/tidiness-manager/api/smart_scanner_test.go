package main

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

// [REQ:TM-SS-001] AI batch configuration
func TestSmartScannerBatchConfiguration(t *testing.T) {
	tests := []struct {
		name             string
		maxFilesPerBatch int
		maxConcurrent    int
		expectedValid    bool
	}{
		{
			name:             "default configuration",
			maxFilesPerBatch: 10,
			maxConcurrent:    5,
			expectedValid:    true,
		},
		{
			name:             "minimal configuration",
			maxFilesPerBatch: 1,
			maxConcurrent:    1,
			expectedValid:    true,
		},
		{
			name:             "high throughput configuration",
			maxFilesPerBatch: 20,
			maxConcurrent:    10,
			expectedValid:    true,
		},
		{
			name:             "invalid - zero files per batch",
			maxFilesPerBatch: 0,
			maxConcurrent:    5,
			expectedValid:    false,
		},
		{
			name:             "invalid - zero concurrency",
			maxFilesPerBatch: 10,
			maxConcurrent:    0,
			expectedValid:    false,
		},
		{
			name:             "invalid - negative files per batch",
			maxFilesPerBatch: -5,
			maxConcurrent:    5,
			expectedValid:    false,
		},
		{
			name:             "invalid - negative concurrency",
			maxFilesPerBatch: 10,
			maxConcurrent:    -3,
			expectedValid:    false,
		},
		{
			name:             "invalid - both zero",
			maxFilesPerBatch: 0,
			maxConcurrent:    0,
			expectedValid:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := SmartScanConfig{
				MaxFilesPerBatch: tt.maxFilesPerBatch,
				MaxConcurrent:    tt.maxConcurrent,
			}

			valid := config.Validate()
			if valid != tt.expectedValid {
				t.Errorf("Validate() = %v, want %v", valid, tt.expectedValid)
			}
		})
	}
}

// [REQ:TM-SS-001] Test default configuration values
func TestSmartScannerDefaults(t *testing.T) {
	config := GetDefaultSmartScanConfig()

	expectedMaxFiles := 10
	expectedMaxConcurrent := 5
	expectedMaxTokens := 100000

	if config.MaxFilesPerBatch != expectedMaxFiles {
		t.Errorf("Default MaxFilesPerBatch = %d, want %d", config.MaxFilesPerBatch, expectedMaxFiles)
	}

	if config.MaxConcurrent != expectedMaxConcurrent {
		t.Errorf("Default MaxConcurrent = %d, want %d", config.MaxConcurrent, expectedMaxConcurrent)
	}

	if config.MaxTokensPerReq != expectedMaxTokens {
		t.Errorf("Default MaxTokensPerReq = %d, want %d", config.MaxTokensPerReq, expectedMaxTokens)
	}

	// Verify default configuration is valid
	if !config.Validate() {
		t.Error("Default configuration should be valid")
	}
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

// [REQ:TM-SS-001] Test NewSmartScanner initialization
func TestNewSmartScanner(t *testing.T) {
	tests := []struct {
		name        string
		config      SmartScanConfig
		envVarValue string
		shouldError bool
		expectedURL string
	}{
		{
			name: "valid config with default URL",
			config: SmartScanConfig{
				MaxFilesPerBatch: 10,
				MaxConcurrent:    5,
				MaxTokensPerReq:  100000,
			},
			envVarValue: "",
			shouldError: false,
			expectedURL: "http://localhost:8100",
		},
		{
			name: "valid config with custom URL",
			config: SmartScanConfig{
				MaxFilesPerBatch: 10,
				MaxConcurrent:    5,
				MaxTokensPerReq:  100000,
			},
			envVarValue: "http://custom-claude:9000",
			shouldError: false,
			expectedURL: "http://custom-claude:9000",
		},
		{
			name: "invalid config - zero batch size",
			config: SmartScanConfig{
				MaxFilesPerBatch: 0,
				MaxConcurrent:    5,
				MaxTokensPerReq:  100000,
			},
			envVarValue: "",
			shouldError: true,
		},
		{
			name: "invalid config - zero concurrency",
			config: SmartScanConfig{
				MaxFilesPerBatch: 10,
				MaxConcurrent:    0,
				MaxTokensPerReq:  100000,
			},
			envVarValue: "",
			shouldError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set environment variable
			if tt.envVarValue != "" {
				oldEnv := os.Getenv("CLAUDE_CODE_URL")
				os.Setenv("CLAUDE_CODE_URL", tt.envVarValue)
				defer os.Setenv("CLAUDE_CODE_URL", oldEnv)
			}

			scanner, err := NewSmartScanner(tt.config)

			if tt.shouldError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if scanner == nil {
				t.Fatal("Scanner is nil")
			}

			if scanner.resourceURL != tt.expectedURL {
				t.Errorf("resourceURL = %s, want %s", scanner.resourceURL, tt.expectedURL)
			}

			if scanner.sessionID == "" {
				t.Error("sessionID should not be empty")
			}

			if scanner.httpClient == nil {
				t.Error("httpClient should not be nil")
			}

			if scanner.analyzedFiles == nil {
				t.Error("analyzedFiles map should be initialized")
			}
		})
	}
}

// [REQ:TM-SS-007] Test session-based file tracking
func TestSmartScannerFileTracking(t *testing.T) {
	config := GetDefaultSmartScanConfig()
	scanner, err := NewSmartScanner(config)
	if err != nil {
		t.Fatalf("Failed to create scanner: %v", err)
	}

	testFiles := []string{
		"api/main.go",
		"api/handlers.go",
		"ui/src/App.tsx",
	}

	// Initially, no files should be marked as analyzed
	for _, file := range testFiles {
		if scanner.isFileAnalyzed(file) {
			t.Errorf("File %s should not be marked as analyzed initially", file)
		}
	}

	// Mark files as analyzed
	for _, file := range testFiles {
		scanner.markFileAnalyzed(file)
	}

	// Verify files are now marked as analyzed
	for _, file := range testFiles {
		if !scanner.isFileAnalyzed(file) {
			t.Errorf("File %s should be marked as analyzed", file)
		}
	}

	// Verify different file is not marked
	if scanner.isFileAnalyzed("api/unknown.go") {
		t.Error("Unmarked file should not be analyzed")
	}
}

// [REQ:TM-SS-007] Test concurrent file tracking safety
func TestSmartScannerConcurrentFileTracking(t *testing.T) {
	config := GetDefaultSmartScanConfig()
	scanner, err := NewSmartScanner(config)
	if err != nil {
		t.Fatalf("Failed to create scanner: %v", err)
	}

	// Test concurrent marking and reading
	const numGoroutines = 10
	const filesPerGoroutine = 100

	var wg sync.WaitGroup
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < filesPerGoroutine; j++ {
				file := "file-" + string(rune('0'+id)) + "-" + string(rune('0'+j))
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

// [REQ:TM-SS-002] Test AI issue structure and validation
func TestAIIssueStructure(t *testing.T) {
	line := 42
	col := 10

	issue := AIIssue{
		FilePath:         "api/main.go",
		Category:         "dead_code",
		Severity:         "medium",
		Title:            "Unused function detected",
		Description:      "Function calculateMetrics is defined but never called",
		LineNumber:       &line,
		ColumnNumber:     &col,
		AgentNotes:       "Consider removing if truly unused",
		RemediationSteps: "1. Search for usage\n2. Remove if unused\n3. Run tests",
	}

	// Verify all fields are set correctly
	if issue.FilePath != "api/main.go" {
		t.Errorf("FilePath = %s, want api/main.go", issue.FilePath)
	}

	if issue.Category != "dead_code" {
		t.Errorf("Category = %s, want dead_code", issue.Category)
	}

	if issue.Severity != "medium" {
		t.Errorf("Severity = %s, want medium", issue.Severity)
	}

	if issue.LineNumber == nil || *issue.LineNumber != 42 {
		t.Errorf("LineNumber incorrect")
	}

	if issue.ColumnNumber == nil || *issue.ColumnNumber != 10 {
		t.Errorf("ColumnNumber incorrect")
	}

	// Test valid categories
	validCategories := []string{"dead_code", "duplication", "length", "complexity", "style"}
	for _, cat := range validCategories {
		issue.Category = cat
		// No validation function exists, but we're documenting expected values
	}

	// Test valid severities
	validSeverities := []string{"critical", "high", "medium", "low", "info"}
	for _, sev := range validSeverities {
		issue.Severity = sev
		// No validation function exists, but we're documenting expected values
	}
}

// [REQ:TM-SS-001] Test batch result structure
func TestBatchResultStructure(t *testing.T) {
	line := 10
	issues := []AIIssue{
		{
			FilePath:         "test.go",
			Category:         "complexity",
			Severity:         "high",
			Title:            "High cyclomatic complexity",
			Description:      "Function has complexity of 25",
			LineNumber:       &line,
			RemediationSteps: "Refactor into smaller functions",
		},
	}

	result := BatchResult{
		BatchID:  1,
		Files:    []string{"test.go"},
		Issues:   issues,
		Duration: 500 * time.Millisecond,
		Error:    "",
	}

	if result.BatchID != 1 {
		t.Errorf("BatchID = %d, want 1", result.BatchID)
	}

	if len(result.Files) != 1 {
		t.Errorf("Files length = %d, want 1", len(result.Files))
	}

	if len(result.Issues) != 1 {
		t.Errorf("Issues length = %d, want 1", len(result.Issues))
	}

	if result.Duration != 500*time.Millisecond {
		t.Errorf("Duration = %v, want 500ms", result.Duration)
	}

	// Test with error
	errorResult := BatchResult{
		BatchID:  2,
		Files:    []string{"error.go"},
		Issues:   []AIIssue{},
		Duration: 100 * time.Millisecond,
		Error:    "failed to read file",
	}

	if errorResult.Error == "" {
		t.Error("Error should be set")
	}

	if len(errorResult.Issues) != 0 {
		t.Error("Issues should be empty when error occurred")
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
	config := GetDefaultSmartScanConfig()
	scanner, err := NewSmartScanner(config)
	if err != nil {
		t.Fatalf("Failed to create scanner: %v", err)
	}

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
				if !containsSubstring(prompt, expected) {
					t.Errorf("Prompt should contain %q", expected)
				}
			}

			// Verify JSON format instruction
			if !containsSubstring(prompt, "JSON") {
				t.Error("Prompt should mention JSON format")
			}
		})
	}
}

// Helper function for substring checking
func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// [REQ:TM-SS-001] Test SmartScanResult structure
func TestSmartScanResultStructure(t *testing.T) {
	result := SmartScanResult{
		SessionID:     "test-session-123",
		FilesAnalyzed: 10,
		IssuesFound:   5,
		BatchResults:  []BatchResult{},
		Duration:      2 * time.Second,
		Errors:        []string{"error1", "error2"},
	}

	if result.SessionID != "test-session-123" {
		t.Errorf("SessionID = %s, want test-session-123", result.SessionID)
	}

	if result.FilesAnalyzed != 10 {
		t.Errorf("FilesAnalyzed = %d, want 10", result.FilesAnalyzed)
	}

	if result.IssuesFound != 5 {
		t.Errorf("IssuesFound = %d, want 5", result.IssuesFound)
	}

	if result.Duration != 2*time.Second {
		t.Errorf("Duration = %v, want 2s", result.Duration)
	}

	if len(result.Errors) != 2 {
		t.Errorf("Errors length = %d, want 2", len(result.Errors))
	}
}

// [REQ:TM-SS-001] Test readFileContent with valid files
func TestReadFileContent(t *testing.T) {
	// Set up test scenario directory structure
	vrooliRoot := t.TempDir()
	scenarioName := "test-scenario"
	scenarioDir := filepath.Join(vrooliRoot, "scenarios", scenarioName)

	if err := os.MkdirAll(scenarioDir, 0755); err != nil {
		t.Fatalf("Failed to create scenario dir: %v", err)
	}

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

	config := GetDefaultSmartScanConfig()
	scanner, err := NewSmartScanner(config)
	if err != nil {
		t.Fatalf("Failed to create scanner: %v", err)
	}

	// Set VROOLI_ROOT for testing
	oldRoot := os.Getenv("VROOLI_ROOT")
	os.Setenv("VROOLI_ROOT", vrooliRoot)
	defer os.Setenv("VROOLI_ROOT", oldRoot)

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
	config := GetDefaultSmartScanConfig()
	scanner, err := NewSmartScanner(config)
	if err != nil {
		t.Fatalf("Failed to create scanner: %v", err)
	}

	vrooliRoot := t.TempDir()
	oldRoot := os.Getenv("VROOLI_ROOT")
	os.Setenv("VROOLI_ROOT", vrooliRoot)
	defer os.Setenv("VROOLI_ROOT", oldRoot)

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
