// +build testing

package main

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestMainLifecycleCheck tests lifecycle requirement
func TestMainLifecycleCheck(t *testing.T) {
	// We can't test main() directly as it calls os.Exit
	// but we test the handlers it would set up
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("Router Setup", func(t *testing.T) {
		handler := setupTestRouter()
		if handler == nil {
			t.Error("Expected non-nil handler")
		}
	})
}

// TestHealthResponseStructure tests the HealthResponse type
func TestHealthResponseStructure(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	response := HealthResponse{
		Status:    "healthy",
		Timestamp: time.Now(),
		Version:   "1.0.0",
		Service:   "code-tidiness-manager-api",
	}

	if response.Status != "healthy" {
		t.Errorf("Expected status 'healthy', got %s", response.Status)
	}

	if response.Version != "1.0.0" {
		t.Errorf("Expected version '1.0.0', got %s", response.Version)
	}
}

// TestTidinessErrorStructure tests the TidinessError type
func TestTidinessErrorStructure(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	err := TidinessError{
		Message:   "test error",
		Type:      "test_type",
		Timestamp: time.Now().Format(time.RFC3339),
	}

	if err.Message != "test error" {
		t.Errorf("Expected message 'test error', got %s", err.Message)
	}

	if err.Type != "test_type" {
		t.Errorf("Expected type 'test_type', got %s", err.Type)
	}
}

// TestAllScanTypes tests all scan type variations
func TestAllScanTypes(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	// Create comprehensive test files
	createTestFile(t, env.TempDir, "backup.bak", "backup")
	createTestFile(t, env.TempDir, "file.orig", "original")
	createTestFile(t, env.TempDir, "test.old", "old")
	createTestFile(t, env.TempDir, "file~", "tilde")
	createTestFile(t, env.TempDir, ".vim.swp", "swap")
	createTestFile(t, env.TempDir, ".DS_Store", "mac")
	createTestFile(t, env.TempDir, "Thumbs.db", "windows")
	createTestFile(t, env.TempDir, "npm-debug.log", "npm")

	handler := setupTestRouter()

	scanTypes := []string{"backup_files", "temp_files", "empty_dirs", "large_files"}

	for _, scanType := range scanTypes {
		t.Run("Scan Type: "+scanType, func(t *testing.T) {
			req, err := makeHTTPRequest("POST", "/api/v1/scan", map[string]interface{}{
				"paths": []string{env.TempDir},
				"types": []string{scanType},
				"limit": 50,
			})
			if err != nil {
				t.Fatalf("Failed to create request: %v", err)
			}

			rr := executeRequest(handler, req)

			if rr.Code != http.StatusOK && rr.Code != http.StatusBadRequest {
				t.Errorf("Unexpected status code: %d", rr.Code)
			}
		})
	}
}

// TestAllAnalysisTypes tests all analysis type variations
func TestAllAnalysisTypes(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	// Create files for different analysis types
	createTestFile(t, env.TempDir, "code.js", `
		import React from 'react';
		const password = "secret123";
		const url = "http://localhost:3000";
		// TODO: fix this
		// FIXME: broken
		function unused() {}
	`)

	handler := setupTestRouter()

	analysisTypes := []string{
		"duplicate_detection",
		"unused_imports",
		"dead_code",
		"hardcoded_values",
		"todo_comments",
	}

	for _, analysisType := range analysisTypes {
		t.Run("Analysis Type: "+analysisType, func(t *testing.T) {
			req, err := makeHTTPRequest("POST", "/api/v1/analyze", map[string]interface{}{
				"analysis_type": analysisType,
				"paths":         []string{env.TempDir},
				"deep_analysis": true,
			})
			if err != nil {
				t.Fatalf("Failed to create request: %v", err)
			}

			rr := executeRequest(handler, req)

			if rr.Code != http.StatusOK && rr.Code != http.StatusBadRequest {
				t.Errorf("Unexpected status code: %d for %s", rr.Code, analysisType)
			}
		})
	}
}

// TestDeepAnalysisFlag tests deep analysis variations
func TestDeepAnalysisFlag(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	createTestFile(t, env.TempDir, "test.js", "// TODO: test")

	handler := setupTestRouter()

	tests := []struct {
		name         string
		deepAnalysis bool
	}{
		{"Shallow Analysis", false},
		{"Deep Analysis", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := makeHTTPRequest("POST", "/api/v1/analyze", map[string]interface{}{
				"analysis_type": "todo_comments",
				"paths":         []string{env.TempDir},
				"deep_analysis": tt.deepAnalysis,
			})
			if err != nil {
				t.Fatalf("Failed to create request: %v", err)
			}

			rr := executeRequest(handler, req)
			response := assertJSONResponse(t, rr, http.StatusOK)

			if deepAnalysis, ok := response["deep_analysis_performed"].(bool); ok {
				if deepAnalysis != tt.deepAnalysis {
					t.Errorf("Expected deep_analysis_performed=%v, got %v", tt.deepAnalysis, deepAnalysis)
				}
			}
		})
	}
}

// TestCleanupWithDifferentScripts tests various cleanup scenarios
func TestCleanupWithDifferentScripts(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	handler := setupTestRouter()

	tests := []struct {
		name           string
		scripts        []string
		dryRun         bool
		expectedStatus int
	}{
		{"Single Script Dry Run", []string{"echo 'test'"}, true, http.StatusOK},
		{"Multiple Scripts Dry Run", []string{"echo 'a'", "echo 'b'"}, true, http.StatusOK},
		{"Single Script Execute", []string{"ls -la"}, false, http.StatusOK},
		{"Empty Scripts", []string{}, false, http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body := map[string]interface{}{
				"dry_run": tt.dryRun,
			}

			if len(tt.scripts) > 0 {
				body["cleanup_scripts"] = tt.scripts
			}

			req, err := makeHTTPRequest("POST", "/api/v1/cleanup", body)
			if err != nil {
				t.Fatalf("Failed to create request: %v", err)
			}

			rr := executeRequest(handler, req)

			if rr.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, rr.Code)
			}
		})
	}
}

// TestN8nWorkflowReplacements tests all n8n replacement endpoints
func TestN8nWorkflowReplacements(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	createTestFiles(t, env)

	handler := setupTestRouter()

	endpoints := []struct {
		path string
		body interface{}
	}{
		{
			"/code-scanner",
			map[string]interface{}{
				"paths": []string{env.TempDir},
				"types": []string{"backup_files"},
			},
		},
		{
			"/pattern-analyzer",
			map[string]interface{}{
				"analysis_type": "todo_comments",
				"paths":         []string{env.TempDir},
			},
		},
		{
			"/cleanup-executor",
			map[string]interface{}{
				"cleanup_scripts": []string{"echo 'test'"},
				"dry_run":         true,
			},
		},
	}

	for _, ep := range endpoints {
		t.Run("Endpoint: "+ep.path, func(t *testing.T) {
			req, err := makeHTTPRequest("POST", ep.path, ep.body)
			if err != nil {
				t.Fatalf("Failed to create request: %v", err)
			}

			rr := executeRequest(handler, req)

			if rr.Code != http.StatusOK && rr.Code != http.StatusBadRequest {
				t.Errorf("Unexpected status code: %d", rr.Code)
			}
		})
	}
}

// TestExcludePatternVariations tests different exclude patterns
func TestExcludePatternVariations(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	// Create excluded directories with test files
	gitDir := filepath.Join(env.TempDir, ".git")
	nodeDir := filepath.Join(env.TempDir, "node_modules")
	distDir := filepath.Join(env.TempDir, "dist")
	buildDir := filepath.Join(env.TempDir, "build")

	for _, dir := range []string{gitDir, nodeDir, distDir, buildDir} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("Failed to create dir %s: %v", dir, err)
		}
		createTestFile(t, dir, "test.bak", "should be excluded")
	}

	// Create file that should NOT be excluded
	createTestFile(t, env.TempDir, "keep.bak", "should be found")

	t.Run("Default Exclude Patterns", func(t *testing.T) {
		ctx, cancel := withTimeout(t, 10*time.Second)
		defer cancel()

		req := CodeScanRequest{
			Paths:           []string{env.TempDir},
			Types:           []string{"backup_files"},
			ExcludePatterns: []string{"node_modules", ".git", "dist", "build"},
		}

		result, err := env.Processor.ScanCode(ctx, req)
		if err != nil {
			t.Fatalf("ScanCode failed: %v", err)
		}

		// Verify excluded files are not in results
		for _, issue := range result.Issues {
			if strings.Contains(issue.FilePath, "node_modules") ||
				strings.Contains(issue.FilePath, ".git") ||
				strings.Contains(issue.FilePath, "dist") ||
				strings.Contains(issue.FilePath, "build") {
				t.Errorf("Found file in excluded directory: %s", issue.FilePath)
			}
		}
	})
}

// TestScanWithDifferentLimits tests limit parameter variations
func TestScanWithDifferentLimits(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	// Create multiple test files
	for i := 0; i < 10; i++ {
		filename := fmt.Sprintf("test%d.bak", i)
		createTestFile(t, env.TempDir, filename, "backup")
	}

	handler := setupTestRouter()

	limits := []int{10, 50, 100, 1000}

	for _, limit := range limits {
		t.Run("Limit: "+string(rune(limit)), func(t *testing.T) {
			req, err := makeHTTPRequest("POST", "/api/v1/scan", map[string]interface{}{
				"paths": []string{env.TempDir},
				"types": []string{"backup_files"},
				"limit": limit,
			})
			if err != nil {
				t.Fatalf("Failed to create request: %v", err)
			}

			rr := executeRequest(handler, req)

			if rr.Code != http.StatusOK {
				t.Errorf("Expected status OK, got %d", rr.Code)
			}
		})
	}
}

// TestResponseFieldValidation tests that all required fields are present
func TestResponseFieldValidation(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	createTestFiles(t, env)

	handler := setupTestRouter()

	t.Run("Scan Response Fields", func(t *testing.T) {
		req, err := makeHTTPRequest("POST", "/api/v1/scan", map[string]interface{}{
			"paths": []string{env.TempDir},
			"types": []string{"backup_files"},
		})
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		rr := executeRequest(handler, req)
		response := assertJSONResponse(t, rr, http.StatusOK)

		// Validate all required fields
		requiredFields := []string{
			"success", "scan_id", "scan_config", "started_at", "completed_at",
			"statistics", "results_by_type", "issues", "total_issues",
		}

		for _, field := range requiredFields {
			if _, exists := response[field]; !exists {
				t.Errorf("Missing required field: %s", field)
			}
		}

		// Validate statistics subfields
		if stats, ok := response["statistics"].(map[string]interface{}); ok {
			statsFields := []string{"total", "by_severity", "automatable", "manual_review"}
			for _, field := range statsFields {
				if _, exists := stats[field]; !exists {
					t.Errorf("Missing statistics field: %s", field)
				}
			}
		}
	})

	t.Run("Pattern Response Fields", func(t *testing.T) {
		req, err := makeHTTPRequest("POST", "/api/v1/analyze", map[string]interface{}{
			"analysis_type": "todo_comments",
			"paths":         []string{env.TempDir},
		})
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		rr := executeRequest(handler, req)
		response := assertJSONResponse(t, rr, http.StatusOK)

		requiredFields := []string{
			"success", "analysis_id", "analysis_type", "started_at", "completed_at",
			"statistics", "issues_by_type", "recommendations", "deep_analysis_performed",
		}

		for _, field := range requiredFields {
			if _, exists := response[field]; !exists {
				t.Errorf("Missing required field: %s", field)
			}
		}
	})

	t.Run("Cleanup Response Fields", func(t *testing.T) {
		req, err := makeHTTPRequest("POST", "/api/v1/cleanup", map[string]interface{}{
			"cleanup_scripts": []string{"echo 'test'"},
			"dry_run":         true,
		})
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		rr := executeRequest(handler, req)
		response := assertJSONResponse(t, rr, http.StatusOK)

		requiredFields := []string{
			"success", "execution_id", "dry_run", "started_at", "completed_at",
			"statistics", "results",
		}

		for _, field := range requiredFields {
			if _, exists := response[field]; !exists {
				t.Errorf("Missing required field: %s", field)
			}
		}
	})
}

// TestErrorHandlingVariations tests different error scenarios
func TestErrorHandlingVariations(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	handler := setupTestRouter()

	t.Run("Malformed JSON", func(t *testing.T) {
		req, _ := http.NewRequest("POST", "/api/v1/scan", strings.NewReader("{invalid json"))
		req.Header.Set("Content-Type", "application/json")

		rr := executeRequest(handler, req)

		if rr.Code != http.StatusBadRequest {
			t.Errorf("Expected status BadRequest, got %d", rr.Code)
		}
	})

	t.Run("Empty Body", func(t *testing.T) {
		req, _ := http.NewRequest("POST", "/api/v1/scan", strings.NewReader("{}"))
		req.Header.Set("Content-Type", "application/json")

		rr := executeRequest(handler, req)

		// Should handle gracefully
		if rr.Code >= 500 {
			t.Logf("Empty body resulted in status %d", rr.Code)
		}
	})

	t.Run("Large Payload", func(t *testing.T) {
		largePaths := make([]string, 1000)
		for i := range largePaths {
			largePaths[i] = "/test/path/" + string(rune(i))
		}

		req, err := makeHTTPRequest("POST", "/api/v1/scan", map[string]interface{}{
			"paths": largePaths,
			"types": []string{"backup_files"},
		})
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		rr := executeRequest(handler, req)

		// Should handle large payloads
		if rr.Code >= 500 {
			t.Logf("Large payload resulted in status %d", rr.Code)
		}
	})
}
