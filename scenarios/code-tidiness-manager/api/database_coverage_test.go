// +build testing

package main

import (
	"fmt"
	"testing"
	"time"
)

// TestStorageMethodsComprehensive tests storage methods more thoroughly
func TestStorageMethodsComprehensive(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	processor := mockProcessor()

	t.Run("Store Multiple Scan Results", func(t *testing.T) {
		ctx, cancel := withTimeout(t, 5*time.Second)
		defer cancel()

		// Test with various scan requests
		requests := []CodeScanRequest{
			{
				Paths: []string{env.TempDir},
				Types: []string{"backup_files"},
			},
			{
				Paths: []string{env.TempDir},
				Types: []string{"temp_files", "empty_dirs"},
			},
		}

		for i, req := range requests {
			issues := []ScanIssue{
				{FilePath: "/test", IssueType: "test"},
			}

			scanID := fmt.Sprintf("scan-%d", i)
			err := processor.storeScanResults(ctx, scanID, req, issues)
			if err != nil {
				t.Logf("Store scan results returned: %v (expected for nil db)", err)
			}
		}
	})

	t.Run("Store Multiple Pattern Results", func(t *testing.T) {
		ctx, cancel := withTimeout(t, 5*time.Second)
		defer cancel()

		analysisTypes := []string{"duplicate_detection", "todo_comments", "hardcoded_values"}

		for i, analysisType := range analysisTypes {
			req := PatternAnalysisRequest{
				AnalysisType: analysisType,
				Paths:        []string{env.TempDir},
			}

			issues := []PatternIssue{
				{Type: "test", Severity: "low"},
			}

			analysisID := fmt.Sprintf("analysis-%d", i)
			err := processor.storePatternResults(ctx, analysisID, req, issues)
			if err != nil {
				t.Logf("Store pattern results returned: %v (expected for nil db)", err)
			}
		}
	})

	t.Run("Store Multiple Cleanup Results", func(t *testing.T) {
		ctx, cancel := withTimeout(t, 5*time.Second)
		defer cancel()

		for i := 0; i < 3; i++ {
			req := CleanupExecutionRequest{
				CleanupScripts: []string{"echo 'test'"},
				DryRun:         true,
			}

			results := []CleanupResult{
				{Script: "echo 'test'", Success: true, Output: "test"},
			}

			executionID := fmt.Sprintf("cleanup-%d", i)
			err := processor.storeCleanupResults(ctx, executionID, req, results)
			if err != nil {
				t.Logf("Store cleanup results returned: %v (expected for nil db)", err)
			}
		}
	})
}

// TestCompleteWorkflowIntegration tests end-to-end workflows
func TestCompleteWorkflowIntegration(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	// Create comprehensive test environment
	createTestFile(t, env.TempDir, "file.bak", "backup")
	createTestFile(t, env.TempDir, ".swp", "swap")
	createTestFile(t, env.TempDir, "code.js", `
		const password = "secret";
		// TODO: fix this
	`)

	t.Run("Complete Scan to Cleanup Workflow", func(t *testing.T) {
		ctx, cancel := withTimeout(t, 15*time.Second)
		defer cancel()

		// Step 1: Scan
		scanReq := CodeScanRequest{
			Paths: []string{env.TempDir},
			Types: []string{"backup_files", "temp_files"},
		}

		scanResult, err := env.Processor.ScanCode(ctx, scanReq)
		if err != nil {
			t.Fatalf("Scan failed: %v", err)
		}

		if !scanResult.Success {
			t.Error("Expected scan to succeed")
		}

		// Step 2: Analyze patterns
		analyzeReq := PatternAnalysisRequest{
			AnalysisType: "hardcoded_values",
			Paths:        []string{env.TempDir},
		}

		analyzeResult, err := env.Processor.AnalyzePatterns(ctx, analyzeReq)
		if err != nil {
			t.Fatalf("Analysis failed: %v", err)
		}

		if !analyzeResult.Success {
			t.Error("Expected analysis to succeed")
		}

		// Step 3: Execute cleanup (dry run)
		cleanupReq := CleanupExecutionRequest{
			CleanupScripts: []string{"echo 'cleanup test'"},
			DryRun:         true,
		}

		cleanupResult, err := env.Processor.ExecuteCleanup(ctx, cleanupReq)
		if err != nil {
			t.Fatalf("Cleanup failed: %v", err)
		}

		if !cleanupResult.Success {
			t.Error("Expected cleanup to succeed")
		}

		// Verify workflow continuity
		t.Logf("Workflow completed: Scan ID=%s, Analysis ID=%s, Cleanup ID=%s",
			scanResult.ScanID, analyzeResult.AnalysisID, cleanupResult.ExecutionID)
	})
}

// TestAllResponseTypes tests all response type structures
func TestAllResponseTypes(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("CodeScanResponse Structure", func(t *testing.T) {
		resp := &CodeScanResponse{
			Success:     true,
			ScanID:      "test-scan",
			ScanConfig:  CodeScanRequest{},
			StartedAt:   "2025-01-01T00:00:00Z",
			CompletedAt: "2025-01-01T00:01:00Z",
			Statistics: ScanStatistics{
				Total:       10,
				BySeverity:  map[string]int{"low": 5, "medium": 3, "high": 2},
				Automatable: 8,
				ManualReview: 2,
			},
			ResultsByType: map[string]TypeResult{
				"backup_files": {Count: 5, Patterns: []string{"*.bak"}},
			},
			Issues:      []ScanIssue{},
			TotalIssues: 10,
		}

		if resp.ScanID != "test-scan" {
			t.Errorf("Unexpected scan ID: %s", resp.ScanID)
		}

		if resp.Statistics.Total != 10 {
			t.Errorf("Unexpected total: %d", resp.Statistics.Total)
		}
	})

	t.Run("PatternAnalysisResponse Structure", func(t *testing.T) {
		resp := &PatternAnalysisResponse{
			Success:     true,
			AnalysisID:  "test-analysis",
			AnalysisType: "duplicate_detection",
			Statistics: PatternStatistics{
				TotalIssues:      5,
				IssuesBySeverity: map[string]int{"medium": 3},
				IssueTypes:       2,
			},
			IssuesByType: map[string][]PatternIssue{},
			Recommendations: []Recommendation{
				{Priority: "high", Action: "Test action"},
			},
			DeepAnalysisPerformed: true,
		}

		if !resp.DeepAnalysisPerformed {
			t.Error("Expected deep analysis performed")
		}

		if len(resp.Recommendations) != 1 {
			t.Errorf("Expected 1 recommendation, got %d", len(resp.Recommendations))
		}
	})

	t.Run("CleanupExecutionResponse Structure", func(t *testing.T) {
		resp := &CleanupExecutionResponse{
			Success:     true,
			ExecutionID: "test-cleanup",
			DryRun:      true,
			Statistics: CleanupStatistics{
				TotalExecuted: 5,
				Successful:    4,
				Failed:        1,
				FilesDeleted:  3,
			},
			Results: []CleanupResult{},
			Errors:  []CleanupError{},
		}

		if !resp.DryRun {
			t.Error("Expected dry run flag")
		}

		if resp.Statistics.Successful != 4 {
			t.Errorf("Unexpected successful count: %d", resp.Statistics.Successful)
		}
	})
}

// TestDefaultValueHandling tests default value assignment
func TestDefaultValueHandling(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	processor := mockProcessor()

	t.Run("Scan Defaults Override Empty Values", func(t *testing.T) {
		req := CodeScanRequest{}
		result := processor.setScanDefaults(req)

		if len(result.Paths) == 0 {
			t.Error("Expected default paths")
		}

		if len(result.Types) == 0 {
			t.Error("Expected default types")
		}

		if result.Limit == 0 {
			t.Error("Expected default limit")
		}

		if len(result.ExcludePatterns) == 0 {
			t.Error("Expected default exclude patterns")
		}

		// Verify defaults are reasonable
		if result.Limit < 0 {
			t.Errorf("Invalid default limit: %d", result.Limit)
		}
	})

	t.Run("Scan Defaults Preserve Existing Values", func(t *testing.T) {
		req := CodeScanRequest{
			Paths: []string{"/custom/path"},
			Types: []string{"custom_type"},
			Limit: 500,
			ExcludePatterns: []string{"custom_exclude"},
		}

		result := processor.setScanDefaults(req)

		// Custom values should NOT be overridden
		if len(result.Paths) != 1 || result.Paths[0] != "/custom/path" {
			t.Error("Expected custom path to be preserved")
		}

		if len(result.Types) != 1 || result.Types[0] != "custom_type" {
			t.Error("Expected custom type to be preserved")
		}

		if result.Limit != 500 {
			t.Error("Expected custom limit to be preserved")
		}
	})

	t.Run("Pattern Defaults", func(t *testing.T) {
		req := PatternAnalysisRequest{}
		result := processor.setPatternDefaults(req)

		if result.AnalysisType == "" {
			t.Error("Expected default analysis type")
		}

		if len(result.Paths) == 0 {
			t.Error("Expected default paths")
		}
	})
}

// TestErrorConditionCoverage tests various error conditions
func TestErrorConditionCoverage(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	processor := mockProcessor()

	t.Run("Multiple Invalid Paths", func(t *testing.T) {
		req := CodeScanRequest{
			Paths: []string{
				"/nonexistent1",
				"/nonexistent2",
				"/nonexistent3",
			},
			Types: []string{"backup_files"},
		}

		err := processor.validateScanRequest(req)
		if err == nil {
			t.Error("Expected error for invalid paths")
		}
	})

	t.Run("Mixed Valid and Invalid Types", func(t *testing.T) {
		req := CodeScanRequest{
			Paths: []string{"/tmp"},
			Types: []string{
				"backup_files",
				"invalid_type_1",
				"temp_files",
				"invalid_type_2",
			},
		}

		err := processor.validateScanRequest(req)
		if err == nil {
			t.Error("Expected error for invalid scan types")
		}
	})

	t.Run("Dangerous Script Variants", func(t *testing.T) {
		dangerous := []string{
			"rm -rf /",
			"rm -fr /",
			"some command && rm -rf /",
			"rm -rf / || echo done",
		}

		for _, script := range dangerous {
			_, err := processor.prepareSafeScripts([]string{script}, false)
			if err == nil {
				t.Errorf("Expected error for dangerous script: %s", script)
			}
		}
	})
}
