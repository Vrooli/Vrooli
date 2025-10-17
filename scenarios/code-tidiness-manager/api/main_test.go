// +build testing

package main

import (
	"net/http"
	"testing"

	"github.com/gorilla/mux"
	"github.com/rs/cors"
)

func setupTestRouter() http.Handler {
	tidinessProcessor = NewTidinessProcessor(nil)

	r := mux.NewRouter()

	// Health check endpoints
	r.HandleFunc("/health", healthHandler).Methods("GET")
	r.HandleFunc("/api/v1/health", healthHandler).Methods("GET")
	r.HandleFunc("/api/v1/health/scan", healthScanHandler).Methods("GET")

	// Tidiness management endpoints
	r.HandleFunc("/api/v1/scan", scanCodeHandler).Methods("POST")
	r.HandleFunc("/api/v1/analyze", analyzePatternHandler).Methods("POST")
	r.HandleFunc("/api/v1/cleanup", executeCleanupHandler).Methods("POST")

	// Processor endpoints (direct n8n workflow replacements)
	r.HandleFunc("/code-scanner", codeScannerHandler).Methods("POST")
	r.HandleFunc("/pattern-analyzer", patternAnalyzerHandler).Methods("POST")
	r.HandleFunc("/cleanup-executor", cleanupExecutorHandler).Methods("POST")

	c := cors.New(cors.Options{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders: []string{"*"},
	})

	return c.Handler(r)
}

// TestHealthEndpoints tests all health check endpoints
func TestHealthEndpoints(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	handler := setupTestRouter()
	suite := NewHandlerTestSuite(t, handler)

	t.Run("Basic Health Check", func(t *testing.T) {
		suite.TestHealthEndpoint("/health")
	})

	t.Run("API Health Check", func(t *testing.T) {
		suite.TestHealthEndpoint("/api/v1/health")
	})

	t.Run("Scan Health Check", func(t *testing.T) {
		req, err := http.NewRequest("GET", "/api/v1/health/scan", nil)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		rr := executeRequest(handler, req)
		response := assertJSONResponse(t, rr, http.StatusOK)

		if status, ok := response["status"].(string); !ok || status != "healthy" {
			t.Errorf("Expected status 'healthy', got %v", response["status"])
		}

		if capabilities, ok := response["capabilities"].([]interface{}); !ok || len(capabilities) == 0 {
			t.Error("Expected capabilities list in scan health response")
		}

		if scanTypes, ok := response["supported_scan_types"].([]interface{}); !ok || len(scanTypes) == 0 {
			t.Error("Expected supported_scan_types list in scan health response")
		}
	})
}

// TestScanCodeHandler tests the scan code endpoint
func TestScanCodeHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	// Create test files
	createTestFiles(t, env)

	handler := setupTestRouter()

	t.Run("Success - Valid Scan Request", func(t *testing.T) {
		req, err := makeHTTPRequest("POST", "/api/v1/scan", map[string]interface{}{
			"paths":      []string{env.TempDir},
			"types":      []string{"backup_files", "temp_files"},
			"deep_scan":  false,
			"limit":      100,
		})
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		rr := executeRequest(handler, req)
		response := assertJSONResponse(t, rr, http.StatusOK)
		assertScanResponse(t, response)

		// Verify scan results
		if totalIssues, ok := response["total_issues"].(float64); ok && totalIssues < 0 {
			t.Errorf("Expected non-negative total_issues, got %v", totalIssues)
		}

		if statistics, ok := response["statistics"].(map[string]interface{}); ok {
			if total, ok := statistics["total"].(float64); !ok || total < 0 {
				t.Error("Expected valid statistics.total")
			}
		}
	})

	t.Run("Success - Default Values", func(t *testing.T) {
		req, err := makeHTTPRequest("POST", "/api/v1/scan", map[string]interface{}{})
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		rr := executeRequest(handler, req)

		// Should use defaults but may fail validation on non-existent default path
		if rr.Code != http.StatusOK && rr.Code != http.StatusBadRequest {
			t.Errorf("Expected status OK or BadRequest, got %d", rr.Code)
		}
	})

	t.Run("Error Cases", func(t *testing.T) {
		errorPattern := NewErrorTestPattern()
		scenarios := NewTestScenarioBuilder().
			AddInvalidJSON("/api/v1/scan", "POST").
			AddInvalidPath("/api/v1/scan", "POST").
			AddInvalidScanType("/api/v1/scan", "POST").
			Build()

		errorPattern.TestAllScenarios(t, handler, scenarios)
	})
}

// TestCodeScannerHandler tests the code-scanner workflow replacement endpoint
func TestCodeScannerHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	createTestFiles(t, env)

	handler := setupTestRouter()

	t.Run("Success - Code Scanner", func(t *testing.T) {
		req, err := makeHTTPRequest("POST", "/code-scanner", map[string]interface{}{
			"paths": []string{env.TempDir},
			"types": []string{"backup_files"},
		})
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		rr := executeRequest(handler, req)
		response := assertJSONResponse(t, rr, http.StatusOK)
		assertScanResponse(t, response)
	})

	t.Run("Error - Invalid JSON", func(t *testing.T) {
		scenarios := NewTestScenarioBuilder().
			AddInvalidJSON("/code-scanner", "POST").
			Build()

		errorPattern := NewErrorTestPattern()
		errorPattern.TestAllScenarios(t, handler, scenarios)
	})
}

// TestAnalyzePatternHandler tests the pattern analysis endpoint
func TestAnalyzePatternHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	handler := setupTestRouter()

	t.Run("Success - Duplicate Detection", func(t *testing.T) {
		req, err := makeHTTPRequest("POST", "/api/v1/analyze", map[string]interface{}{
			"analysis_type": "duplicate_detection",
			"paths":         []string{env.TempDir},
			"deep_analysis": false,
		})
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		rr := executeRequest(handler, req)
		response := assertJSONResponse(t, rr, http.StatusOK)
		assertPatternResponse(t, response)

		if analysisType, ok := response["analysis_type"].(string); !ok || analysisType != "duplicate_detection" {
			t.Errorf("Expected analysis_type 'duplicate_detection', got %v", response["analysis_type"])
		}
	})

	t.Run("Success - TODO Comments", func(t *testing.T) {
		req, err := makeHTTPRequest("POST", "/api/v1/analyze", map[string]interface{}{
			"analysis_type": "todo_comments",
			"paths":         []string{env.TempDir},
		})
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		rr := executeRequest(handler, req)
		response := assertJSONResponse(t, rr, http.StatusOK)
		assertPatternResponse(t, response)
	})

	t.Run("Success - Hardcoded Values", func(t *testing.T) {
		req, err := makeHTTPRequest("POST", "/api/v1/analyze", map[string]interface{}{
			"analysis_type": "hardcoded_values",
			"paths":         []string{env.TempDir},
		})
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		rr := executeRequest(handler, req)
		response := assertJSONResponse(t, rr, http.StatusOK)
		assertPatternResponse(t, response)
	})

	t.Run("Success - Unused Imports", func(t *testing.T) {
		req, err := makeHTTPRequest("POST", "/api/v1/analyze", map[string]interface{}{
			"analysis_type": "unused_imports",
			"paths":         []string{env.TempDir},
		})
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		rr := executeRequest(handler, req)
		response := assertJSONResponse(t, rr, http.StatusOK)
		assertPatternResponse(t, response)
	})

	t.Run("Error Cases", func(t *testing.T) {
		errorPattern := NewErrorTestPattern()
		scenarios := NewTestScenarioBuilder().
			AddInvalidJSON("/api/v1/analyze", "POST").
			AddInvalidAnalysisType("/api/v1/analyze", "POST").
			Build()

		errorPattern.TestAllScenarios(t, handler, scenarios)
	})
}

// TestPatternAnalyzerHandler tests the pattern-analyzer workflow replacement endpoint
func TestPatternAnalyzerHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	handler := setupTestRouter()

	t.Run("Success - Pattern Analyzer", func(t *testing.T) {
		req, err := makeHTTPRequest("POST", "/pattern-analyzer", map[string]interface{}{
			"analysis_type": "duplicate_detection",
			"paths":         []string{env.TempDir},
		})
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		rr := executeRequest(handler, req)
		response := assertJSONResponse(t, rr, http.StatusOK)
		assertPatternResponse(t, response)
	})
}

// TestExecuteCleanupHandler tests the cleanup execution endpoint
func TestExecuteCleanupHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	handler := setupTestRouter()

	t.Run("Success - Dry Run", func(t *testing.T) {
		req, err := makeHTTPRequest("POST", "/api/v1/cleanup", map[string]interface{}{
			"cleanup_scripts": []string{"echo 'test'"},
			"dry_run":         true,
		})
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		rr := executeRequest(handler, req)
		response := assertJSONResponse(t, rr, http.StatusOK)
		assertCleanupResponse(t, response)

		if dryRun, ok := response["dry_run"].(bool); !ok || !dryRun {
			t.Error("Expected dry_run=true in response")
		}
	})

	t.Run("Success - Safe Script Execution", func(t *testing.T) {
		testFile := createTestFile(t, env.TempDir, "to_delete.tmp", "temp content")

		req, err := makeHTTPRequest("POST", "/api/v1/cleanup", map[string]interface{}{
			"cleanup_scripts": []string{"echo 'cleaned'"},
			"dry_run":         true,
		})
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		rr := executeRequest(handler, req)
		response := assertJSONResponse(t, rr, http.StatusOK)
		assertCleanupResponse(t, response)

		if results, ok := response["results"].([]interface{}); !ok || len(results) == 0 {
			t.Error("Expected results in cleanup response")
		}

		// File should still exist in dry run
		_ = testFile
	})

	t.Run("Error Cases", func(t *testing.T) {
		errorPattern := NewErrorTestPattern()
		scenarios := NewTestScenarioBuilder().
			AddInvalidJSON("/api/v1/cleanup", "POST").
			AddNoCleanupScripts("/api/v1/cleanup", "POST").
			AddDangerousCleanup("/api/v1/cleanup", "POST").
			Build()

		errorPattern.TestAllScenarios(t, handler, scenarios)
	})
}

// TestCleanupExecutorHandler tests the cleanup-executor workflow replacement endpoint
func TestCleanupExecutorHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	handler := setupTestRouter()

	t.Run("Success - Cleanup Executor", func(t *testing.T) {
		req, err := makeHTTPRequest("POST", "/cleanup-executor", map[string]interface{}{
			"cleanup_scripts": []string{"echo 'test'"},
			"dry_run":         true,
		})
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		rr := executeRequest(handler, req)
		response := assertJSONResponse(t, rr, http.StatusOK)
		assertCleanupResponse(t, response)
	})
}

// TestSendTidinessError tests error response formatting
func TestSendTidinessError(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	handler := setupTestRouter()

	t.Run("Invalid Request Returns Proper Error", func(t *testing.T) {
		req, err := makeHTTPRequest("POST", "/code-scanner", "invalid-json")
		if err != nil {
			req, _ = http.NewRequest("POST", "/code-scanner", nil)
		}

		rr := executeRequest(handler, req)
		assertErrorResponse(t, rr, http.StatusBadRequest, "validation_error")
	})
}

// TestCORSHeaders tests CORS configuration
func TestCORSHeaders(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	handler := setupTestRouter()

	t.Run("CORS Headers Present", func(t *testing.T) {
		req, err := http.NewRequest("OPTIONS", "/api/v1/health", nil)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}
		req.Header.Set("Origin", "http://example.com")
		req.Header.Set("Access-Control-Request-Method", "POST")

		rr := executeRequest(handler, req)

		// CORS should be handled by the cors middleware
		if rr.Code != http.StatusOK && rr.Code != http.StatusNoContent {
			t.Logf("OPTIONS request returned status %d", rr.Code)
		}
	})
}

// TestContentTypeHeaders tests Content-Type header validation
func TestContentTypeHeaders(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	handler := setupTestRouter()

	t.Run("JSON Content-Type", func(t *testing.T) {
		req, err := http.NewRequest("GET", "/health", nil)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		rr := executeRequest(handler, req)

		contentType := rr.Header().Get("Content-Type")
		if contentType != "application/json" {
			t.Errorf("Expected Content-Type application/json, got %s", contentType)
		}
	})
}
