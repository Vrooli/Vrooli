package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

// [REQ:TM-SS-001] [REQ:TM-SS-002] [REQ:TM-SS-007] Test end-to-end batch processing with mock AI resource
func TestSmartScanner_ScanScenarioEndToEnd(t *testing.T) {
	// Create test scenario files
	vrooliRoot := t.TempDir()
	scenarioName := "test-scenario"
	scenarioDir := filepath.Join(vrooliRoot, "scenarios", scenarioName)

	if err := os.MkdirAll(scenarioDir, 0755); err != nil {
		t.Fatalf("Failed to create scenario dir: %v", err)
	}

	// Create test files with different content
	testFiles := map[string]string{
		"main.go":      "package main\n\nfunc main() {\n\tprintln(\"hello\")\n}",
		"util.go":      "package main\n\nfunc unused() {}\nfunc helper() {}",
		"handlers.go":  "package main\n\nfunc HandleRequest() error {\n\treturn nil\n}",
		"types.go":     "package main\n\ntype Config struct {\n\tPort int\n}",
		"constants.go": "package main\n\nconst DefaultPort = 8080",
	}

	for name, content := range testFiles {
		path := filepath.Join(scenarioDir, name)
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to write test file %s: %v", name, err)
		}
	}

	oldRoot := os.Getenv("VROOLI_ROOT")
	os.Setenv("VROOLI_ROOT", vrooliRoot)
	defer os.Setenv("VROOLI_ROOT", oldRoot)

	// Create mock AI resource server
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/analyze" {
			http.NotFound(w, r)
			return
		}

		// Parse request
		var reqBody map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
			http.Error(w, "invalid request", http.StatusBadRequest)
			return
		}

		// Return mock AI issues
		line := 3
		col := 1
		response := map[string]interface{}{
			"issues": []AIIssue{
				{
					FilePath:         "util.go",
					Category:         "dead_code",
					Severity:         "medium",
					Title:            "Unused function",
					Description:      "Function 'unused' is never called",
					LineNumber:       &line,
					ColumnNumber:     &col,
					RemediationSteps: "Remove the unused function or add usage",
				},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer mockServer.Close()

	// Configure scanner with mock server
	oldEnv := os.Getenv("CLAUDE_CODE_URL")
	os.Setenv("CLAUDE_CODE_URL", mockServer.URL)
	defer os.Setenv("CLAUDE_CODE_URL", oldEnv)

	config := SmartScanConfig{
		MaxFilesPerBatch: 2, // Small batches to test batching logic
		MaxConcurrent:    2,
		MaxTokensPerReq:  100000,
	}

	scanner, err := NewSmartScanner(config)
	if err != nil {
		t.Fatalf("Failed to create scanner: %v", err)
	}

	// Prepare files to scan
	filesToScan := []string{}
	for name := range testFiles {
		filesToScan = append(filesToScan, name)
	}

	req := SmartScanRequest{
		Scenario:    scenarioName,
		Files:       filesToScan,
		ForceRescan: false,
	}

	// Execute scan
	ctx := context.Background()
	result, err := scanner.ScanScenario(ctx, req)
	if err != nil {
		t.Fatalf("ScanScenario failed: %v", err)
	}

	// Verify results
	if result.SessionID != scanner.sessionID {
		t.Errorf("SessionID mismatch: got %s, want %s", result.SessionID, scanner.sessionID)
	}

	if result.FilesAnalyzed != len(filesToScan) {
		t.Errorf("FilesAnalyzed = %d, want %d", result.FilesAnalyzed, len(filesToScan))
	}

	// Should have multiple batches (5 files, batch size 2 = 3 batches)
	expectedBatches := 3
	if len(result.BatchResults) != expectedBatches {
		t.Errorf("BatchResults count = %d, want %d", len(result.BatchResults), expectedBatches)
	}

	// Each batch should have processed files
	for _, batch := range result.BatchResults {
		if len(batch.Files) == 0 {
			t.Error("Batch should have processed files")
		}
		if len(batch.Files) > config.MaxFilesPerBatch {
			t.Errorf("Batch has %d files, exceeds max %d", len(batch.Files), config.MaxFilesPerBatch)
		}
	}

	// Verify all files are marked as analyzed
	for _, file := range filesToScan {
		if !scanner.isFileAnalyzed(file) {
			t.Errorf("File %s should be marked as analyzed", file)
		}
	}

	// Test that second scan skips already-analyzed files
	result2, err := scanner.ScanScenario(ctx, req)
	if err != nil {
		t.Fatalf("Second ScanScenario failed: %v", err)
	}

	if result2.FilesAnalyzed != 0 {
		t.Errorf("Second scan should have analyzed 0 files (already analyzed), got %d", result2.FilesAnalyzed)
	}

	// Test force rescan
	req.ForceRescan = true
	result3, err := scanner.ScanScenario(ctx, req)
	if err != nil {
		t.Fatalf("Force rescan failed: %v", err)
	}

	if result3.FilesAnalyzed != len(filesToScan) {
		t.Errorf("Force rescan should analyze all files, got %d, want %d", result3.FilesAnalyzed, len(filesToScan))
	}
}

// [REQ:TM-SS-001] Test batch processing with AI resource errors
func TestSmartScanner_BatchProcessingWithErrors(t *testing.T) {
	vrooliRoot := t.TempDir()
	scenarioName := "test-scenario"
	scenarioDir := filepath.Join(vrooliRoot, "scenarios", scenarioName)
	os.MkdirAll(scenarioDir, 0755)

	// Create a test file
	testFile := "test.go"
	testContent := "package main\n\nfunc main() {}"
	os.WriteFile(filepath.Join(scenarioDir, testFile), []byte(testContent), 0644)

	oldRoot := os.Getenv("VROOLI_ROOT")
	os.Setenv("VROOLI_ROOT", vrooliRoot)
	defer os.Setenv("VROOLI_ROOT", oldRoot)

	tests := []struct {
		name               string
		serverBehavior     func(w http.ResponseWriter, r *http.Request)
		expectError        bool
		expectIssues       bool
		expectGracefulFail bool
	}{
		{
			name: "AI resource returns 500 error",
			serverBehavior: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte("Internal server error"))
			},
			expectError:  false, // Should handle gracefully
			expectIssues: false,
		},
		{
			name: "AI resource returns invalid JSON",
			serverBehavior: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.Write([]byte("{invalid json"))
			},
			expectError:  false,
			expectIssues: false,
		},
		{
			name: "AI resource returns valid issues",
			serverBehavior: func(w http.ResponseWriter, r *http.Request) {
				line := 1
				response := map[string]interface{}{
					"issues": []AIIssue{
						{
							FilePath:    testFile,
							Category:    "style",
							Severity:    "low",
							Title:       "Style issue",
							Description: "Inconsistent formatting",
							LineNumber:  &line,
						},
					},
				}
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(response)
			},
			expectError:  false,
			expectIssues: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockServer := httptest.NewServer(http.HandlerFunc(tt.serverBehavior))
			defer mockServer.Close()

			oldEnv := os.Getenv("CLAUDE_CODE_URL")
			os.Setenv("CLAUDE_CODE_URL", mockServer.URL)
			defer os.Setenv("CLAUDE_CODE_URL", oldEnv)

			config := GetDefaultSmartScanConfig()
			scanner, err := NewSmartScanner(config)
			if err != nil {
				t.Fatalf("Failed to create scanner: %v", err)
			}

			req := SmartScanRequest{
				Scenario:    scenarioName,
				Files:       []string{testFile},
				ForceRescan: true,
			}

			result, err := scanner.ScanScenario(context.Background(), req)

			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}

			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			if tt.expectIssues && result.IssuesFound == 0 {
				t.Error("Expected issues but found none")
			}

			if !tt.expectIssues && result.IssuesFound > 0 {
				t.Errorf("Expected no issues but found %d", result.IssuesFound)
			}
		})
	}
}

// [REQ:TM-SS-001] Test concurrent batch processing
func TestSmartScanner_ConcurrentBatchProcessing(t *testing.T) {
	vrooliRoot := t.TempDir()
	scenarioName := "test-scenario"
	scenarioDir := filepath.Join(vrooliRoot, "scenarios", scenarioName)
	os.MkdirAll(scenarioDir, 0755)

	// Create multiple test files
	numFiles := 20
	for i := 0; i < numFiles; i++ {
		filename := fmt.Sprintf("file-%d.go", i)
		content := fmt.Sprintf("package main\n\nfunc func%d() {}", i)
		os.WriteFile(filepath.Join(scenarioDir, filename), []byte(content), 0644)
	}

	oldRoot := os.Getenv("VROOLI_ROOT")
	os.Setenv("VROOLI_ROOT", vrooliRoot)
	defer os.Setenv("VROOLI_ROOT", oldRoot)

	// Track concurrent requests
	var requestCount int32
	var maxConcurrent int32
	var mu sync.Mutex

	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		requestCount++
		if requestCount > maxConcurrent {
			maxConcurrent = requestCount
		}
		mu.Unlock()

		// Simulate processing time
		time.Sleep(50 * time.Millisecond)

		response := map[string]interface{}{
			"issues": []AIIssue{},
		}
		json.NewEncoder(w).Encode(response)

		mu.Lock()
		requestCount--
		mu.Unlock()
	}))
	defer mockServer.Close()

	oldEnv := os.Getenv("CLAUDE_CODE_URL")
	os.Setenv("CLAUDE_CODE_URL", mockServer.URL)
	defer os.Setenv("CLAUDE_CODE_URL", oldEnv)

	config := SmartScanConfig{
		MaxFilesPerBatch: 2,
		MaxConcurrent:    3, // Limit to 3 concurrent batches
		MaxTokensPerReq:  100000,
	}

	scanner, err := NewSmartScanner(config)
	if err != nil {
		t.Fatalf("Failed to create scanner: %v", err)
	}

	files := []string{}
	for i := 0; i < numFiles; i++ {
		files = append(files, fmt.Sprintf("file-%d.go", i))
	}

	req := SmartScanRequest{
		Scenario:    scenarioName,
		Files:       files,
		ForceRescan: true,
	}

	result, err := scanner.ScanScenario(context.Background(), req)
	if err != nil {
		t.Fatalf("ScanScenario failed: %v", err)
	}

	// Verify concurrency was respected
	if maxConcurrent > int32(config.MaxConcurrent) {
		t.Errorf("Max concurrent requests = %d, exceeds limit %d", maxConcurrent, config.MaxConcurrent)
	}

	// Verify all files were processed
	if result.FilesAnalyzed != numFiles {
		t.Errorf("FilesAnalyzed = %d, want %d", result.FilesAnalyzed, numFiles)
	}
}

// [REQ:TM-SS-007] Test context cancellation during batch processing
func TestSmartScanner_ContextCancellation(t *testing.T) {
	vrooliRoot := t.TempDir()
	scenarioName := "test-scenario"
	scenarioDir := filepath.Join(vrooliRoot, "scenarios", scenarioName)
	os.MkdirAll(scenarioDir, 0755)

	// Create test files
	for i := 0; i < 10; i++ {
		filename := fmt.Sprintf("file-%d.go", i)
		os.WriteFile(filepath.Join(scenarioDir, filename), []byte("package main"), 0644)
	}

	oldRoot := os.Getenv("VROOLI_ROOT")
	os.Setenv("VROOLI_ROOT", vrooliRoot)
	defer os.Setenv("VROOLI_ROOT", oldRoot)

	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simulate slow processing
		time.Sleep(500 * time.Millisecond)
		json.NewEncoder(w).Encode(map[string]interface{}{"issues": []AIIssue{}})
	}))
	defer mockServer.Close()

	oldEnv := os.Getenv("CLAUDE_CODE_URL")
	os.Setenv("CLAUDE_CODE_URL", mockServer.URL)
	defer os.Setenv("CLAUDE_CODE_URL", oldEnv)

	config := GetDefaultSmartScanConfig()
	scanner, err := NewSmartScanner(config)
	if err != nil {
		t.Fatalf("Failed to create scanner: %v", err)
	}

	files := []string{}
	for i := 0; i < 10; i++ {
		files = append(files, fmt.Sprintf("file-%d.go", i))
	}

	req := SmartScanRequest{
		Scenario:    scenarioName,
		Files:       files,
		ForceRescan: true,
	}

	// Create context with short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	result, err := scanner.ScanScenario(ctx, req)

	// Should complete even if some batches time out
	// (graceful degradation)
	if err != nil {
		t.Logf("Scan completed with error (expected): %v", err)
	}

	// Should have partial results
	t.Logf("Partial results: %d files analyzed", result.FilesAnalyzed)
}
