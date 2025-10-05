package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

// TestServer_NewServer tests server initialization
func TestServer_NewServer(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("Default_Port", func(t *testing.T) {
		// Clear environment
		originalPort := os.Getenv("CODE_SMELL_API_PORT")
		os.Unsetenv("CODE_SMELL_API_PORT")
		defer func() {
			if originalPort != "" {
				os.Setenv("CODE_SMELL_API_PORT", originalPort)
			}
		}()

		server := NewServer()
		if server == nil {
			t.Fatal("Expected server to be initialized")
		}
		if server.port != "8090" {
			t.Errorf("Expected default port 8090, got %s", server.port)
		}
		if server.router == nil {
			t.Error("Expected router to be initialized")
		}
	})

	t.Run("Custom_Port", func(t *testing.T) {
		os.Setenv("CODE_SMELL_API_PORT", "9999")
		defer os.Unsetenv("CODE_SMELL_API_PORT")

		server := NewServer()
		if server.port != "9999" {
			t.Errorf("Expected port 9999, got %s", server.port)
		}
	})
}

// TestHandleHealth tests the health endpoint
func TestHandleHealth(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	server := createTestServer()

	t.Run("Success", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/health",
		}
		w := executeHandler(server.handleHealth, req)

		response := assertJSONResponse(t, w, http.StatusOK, []string{"status", "timestamp", "service", "version"})
		assertResponseField(t, response, "status", "healthy")
		assertResponseField(t, response, "service", "code-smell")
		assertResponseField(t, response, "version", "1.0.0")
	})
}

// TestHandleHealthLive tests the liveness endpoint
func TestHandleHealthLive(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	server := createTestServer()

	t.Run("Success", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/health/live",
		}
		w := executeHandler(server.handleHealthLive, req)

		response := assertJSONResponse(t, w, http.StatusOK, []string{"status"})
		assertResponseField(t, response, "status", "alive")
	})
}

// TestHandleHealthReady tests the readiness endpoint
func TestHandleHealthReady(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	server := createTestServer()

	t.Run("Success", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/health/ready",
		}
		w := executeHandler(server.handleHealthReady, req)

		// Should be ready since checkEngineReady returns true
		response := assertJSONResponse(t, w, http.StatusOK, []string{"status"})
		assertResponseField(t, response, "status", "ready")
	})
}

// TestHandleAnalyze tests the analyze endpoint
func TestHandleAnalyze(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	server := createTestServer()

	t.Run("Success", func(t *testing.T) {
		body := makeJSONBody(buildValidAnalyzeRequest())
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/code-smell/analyze",
			Body:   body,
		}
		w := executeHandler(server.handleAnalyze, req)

		response := assertJSONResponse(t, w, http.StatusOK,
			[]string{"violations", "auto_fixed", "needs_review", "total_files", "duration_ms"})

		// Validate response fields
		if totalFiles, ok := response["total_files"].(float64); ok {
			if int(totalFiles) != 1 {
				t.Errorf("Expected total_files to be 1, got %v", totalFiles)
			}
		}
	})

	t.Run("Error_InvalidJSON", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/code-smell/analyze",
			Body:   `{"invalid": "json"`,
		}
		w := executeHandler(server.handleAnalyze, req)
		assertErrorResponse(t, w, http.StatusBadRequest, "Invalid request body")
	})

	t.Run("Error_NoPaths", func(t *testing.T) {
		body := makeJSONBody(map[string]interface{}{
			"paths":    []string{},
			"auto_fix": false,
		})
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/code-smell/analyze",
			Body:   body,
		}
		w := executeHandler(server.handleAnalyze, req)
		assertErrorResponse(t, w, http.StatusBadRequest, "No paths provided")
	})

	t.Run("Success_WithAutoFix", func(t *testing.T) {
		body := makeJSONBody(map[string]interface{}{
			"paths":          []string{"/test/file.go"},
			"auto_fix":       true,
			"risk_threshold": "low",
		})
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/code-smell/analyze",
			Body:   body,
		}
		w := executeHandler(server.handleAnalyze, req)
		assertJSONResponse(t, w, http.StatusOK, []string{"violations", "auto_fixed"})
	})

	t.Run("Success_WithRules", func(t *testing.T) {
		body := makeJSONBody(map[string]interface{}{
			"paths": []string{"/test/file.go"},
			"rules": []string{"hardcoded-port"},
		})
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/code-smell/analyze",
			Body:   body,
		}
		w := executeHandler(server.handleAnalyze, req)
		assertJSONResponse(t, w, http.StatusOK, []string{"violations"})
	})
}

// TestHandleGetRules tests the rules endpoint
func TestHandleGetRules(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	server := createTestServer()

	t.Run("Success", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/code-smell/rules",
		}
		w := executeHandler(server.handleGetRules, req)

		response := assertJSONResponse(t, w, http.StatusOK,
			[]string{"rules", "categories", "vrooli_specific_count"})

		// Validate rules array
		if rules, ok := response["rules"].([]interface{}); ok {
			if len(rules) == 0 {
				t.Error("Expected at least one rule")
			}
		} else {
			t.Error("Expected 'rules' to be an array")
		}

		// Validate vrooli_specific_count
		if count, ok := response["vrooli_specific_count"].(float64); ok {
			if int(count) < 0 {
				t.Error("Expected vrooli_specific_count to be non-negative")
			}
		}
	})
}

// TestHandleFix tests the fix endpoint
func TestHandleFix(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	server := createTestServer()

	t.Run("Success_Approve", func(t *testing.T) {
		body := makeJSONBody(map[string]interface{}{
			"violation_id": "test-violation-1",
			"action":       "approve",
		})
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/code-smell/fix",
			Body:   body,
		}
		w := executeHandler(server.handleFix, req)

		response := assertJSONResponse(t, w, http.StatusOK,
			[]string{"success", "violation_id", "action"})
		assertResponseField(t, response, "success", true)
		assertResponseField(t, response, "action", "approve")
	})

	t.Run("Success_Reject", func(t *testing.T) {
		body := makeJSONBody(map[string]interface{}{
			"violation_id": "test-violation-2",
			"action":       "reject",
		})
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/code-smell/fix",
			Body:   body,
		}
		w := executeHandler(server.handleFix, req)

		response := assertJSONResponse(t, w, http.StatusOK, []string{"success"})
		assertResponseField(t, response, "success", true)
	})

	t.Run("Success_Ignore", func(t *testing.T) {
		body := makeJSONBody(map[string]interface{}{
			"violation_id": "test-violation-3",
			"action":       "ignore",
		})
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/code-smell/fix",
			Body:   body,
		}
		w := executeHandler(server.handleFix, req)
		assertJSONResponse(t, w, http.StatusOK, []string{"success"})
	})

	t.Run("Success_WithModifiedFix", func(t *testing.T) {
		body := makeJSONBody(map[string]interface{}{
			"violation_id": "test-violation-4",
			"action":       "approve",
			"modified_fix": "custom fix code",
		})
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/code-smell/fix",
			Body:   body,
		}
		w := executeHandler(server.handleFix, req)
		assertJSONResponse(t, w, http.StatusOK, []string{"success"})
	})

	t.Run("Error_InvalidJSON", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/code-smell/fix",
			Body:   `{"invalid": "json"`,
		}
		w := executeHandler(server.handleFix, req)
		assertErrorResponse(t, w, http.StatusBadRequest, "Invalid request body")
	})

	t.Run("Error_MissingViolationID", func(t *testing.T) {
		body := makeJSONBody(map[string]interface{}{
			"action": "approve",
		})
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/code-smell/fix",
			Body:   body,
		}
		w := executeHandler(server.handleFix, req)
		assertErrorResponse(t, w, http.StatusBadRequest, "Violation ID required")
	})

	t.Run("Error_InvalidAction", func(t *testing.T) {
		body := makeJSONBody(map[string]interface{}{
			"violation_id": "test-violation-5",
			"action":       "invalid_action",
		})
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/code-smell/fix",
			Body:   body,
		}
		w := executeHandler(server.handleFix, req)
		assertErrorResponse(t, w, http.StatusBadRequest, "Invalid action")
	})
}

// TestHandleGetQueue tests the queue endpoint
func TestHandleGetQueue(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	server := createTestServer()

	t.Run("Success_All", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/code-smell/queue",
		}
		w := executeHandler(server.handleGetQueue, req)

		response := assertJSONResponse(t, w, http.StatusOK,
			[]string{"violations", "total", "by_severity"})

		// Validate by_severity map
		if bySeverity, ok := response["by_severity"].(map[string]interface{}); ok {
			expectedKeys := []string{"error", "warning", "info"}
			for _, key := range expectedKeys {
				if _, exists := bySeverity[key]; !exists {
					t.Errorf("Expected key '%s' in by_severity", key)
				}
			}
		}
	})

	t.Run("Success_FilterBySeverity", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/code-smell/queue?severity=error",
		}
		w := executeHandler(server.handleGetQueue, req)
		assertJSONResponse(t, w, http.StatusOK, []string{"violations", "total"})
	})

	t.Run("Success_FilterByFile", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/code-smell/queue?file=test.go",
		}
		w := executeHandler(server.handleGetQueue, req)
		assertJSONResponse(t, w, http.StatusOK, []string{"violations"})
	})

	t.Run("Success_MultipleFilters", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/code-smell/queue?severity=warning&file=main.go",
		}
		w := executeHandler(server.handleGetQueue, req)
		assertJSONResponse(t, w, http.StatusOK, []string{"violations"})
	})
}

// TestHandleLearn tests the learn endpoint
func TestHandleLearn(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	server := createTestServer()

	t.Run("Success_PositivePattern", func(t *testing.T) {
		body := makeJSONBody(map[string]interface{}{
			"pattern":     "test-pattern",
			"is_positive": true,
		})
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/code-smell/learn",
			Body:   body,
		}
		w := executeHandler(server.handleLearn, req)

		response := assertJSONResponse(t, w, http.StatusOK,
			[]string{"success", "pattern", "confidence"})
		assertResponseField(t, response, "success", true)
	})

	t.Run("Success_NegativePattern", func(t *testing.T) {
		body := makeJSONBody(map[string]interface{}{
			"pattern":     "bad-pattern",
			"is_positive": false,
		})
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/code-smell/learn",
			Body:   body,
		}
		w := executeHandler(server.handleLearn, req)
		assertJSONResponse(t, w, http.StatusOK, []string{"success"})
	})

	t.Run("Success_WithContext", func(t *testing.T) {
		body := makeJSONBody(map[string]interface{}{
			"pattern":     "context-pattern",
			"is_positive": true,
			"context": map[string]interface{}{
				"file": "test.go",
				"line": 42,
			},
		})
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/code-smell/learn",
			Body:   body,
		}
		w := executeHandler(server.handleLearn, req)
		assertJSONResponse(t, w, http.StatusOK, []string{"success"})
	})

	t.Run("Error_InvalidJSON", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/code-smell/learn",
			Body:   `{"invalid": "json"`,
		}
		w := executeHandler(server.handleLearn, req)
		assertErrorResponse(t, w, http.StatusBadRequest, "Invalid request body")
	})

	t.Run("Error_MissingPattern", func(t *testing.T) {
		body := makeJSONBody(map[string]interface{}{
			"is_positive": true,
		})
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/code-smell/learn",
			Body:   body,
		}
		w := executeHandler(server.handleLearn, req)
		assertErrorResponse(t, w, http.StatusBadRequest, "Pattern required")
	})

	t.Run("Error_EmptyPattern", func(t *testing.T) {
		body := makeJSONBody(map[string]interface{}{
			"pattern":     "",
			"is_positive": true,
		})
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/code-smell/learn",
			Body:   body,
		}
		w := executeHandler(server.handleLearn, req)
		assertErrorResponse(t, w, http.StatusBadRequest, "Pattern required")
	})
}

// TestHandleGetStats tests the stats endpoint
func TestHandleGetStats(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	server := createTestServer()

	t.Run("Success_All", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/code-smell/stats",
		}
		w := executeHandler(server.handleGetStats, req)

		response := assertJSONResponse(t, w, http.StatusOK,
			[]string{"period", "files_analyzed", "violations_found", "auto_fixed", "patterns_learned"})
		assertResponseField(t, response, "period", "all")
	})

	t.Run("Success_Weekly", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/code-smell/stats?period=week",
		}
		w := executeHandler(server.handleGetStats, req)

		response := assertJSONResponse(t, w, http.StatusOK, []string{"period"})
		assertResponseField(t, response, "period", "week")
	})

	t.Run("Success_Monthly", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/code-smell/stats?period=month",
		}
		w := executeHandler(server.handleGetStats, req)

		response := assertJSONResponse(t, w, http.StatusOK, []string{"period"})
		assertResponseField(t, response, "period", "month")
	})
}

// TestHandleDocs tests the documentation endpoint
func TestHandleDocs(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	server := createTestServer()

	t.Run("Success", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/docs",
		}
		w := executeHandler(server.handleDocs, req)

		response := assertJSONResponse(t, w, http.StatusOK,
			[]string{"service", "version", "endpoints"})

		assertResponseField(t, response, "service", "code-smell")
		assertResponseField(t, response, "version", "1.0.0")

		// Validate endpoints array
		if endpoints, ok := response["endpoints"].([]interface{}); ok {
			if len(endpoints) != 6 {
				t.Errorf("Expected 6 endpoints, got %d", len(endpoints))
			}
		} else {
			t.Error("Expected 'endpoints' to be an array")
		}
	})
}

// TestHelperFunctions tests helper functions
func TestHelperFunctions(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("checkEngineReady", func(t *testing.T) {
		result := checkEngineReady()
		if !result {
			t.Error("Expected checkEngineReady to return true")
		}
	})

	t.Run("getCategories", func(t *testing.T) {
		rules := []Rule{
			{Category: "security"},
			{Category: "performance"},
			{Category: "security"},
		}
		categories := getCategories(rules)
		if len(categories) != 2 {
			t.Errorf("Expected 2 unique categories, got %d", len(categories))
		}
	})

	t.Run("sendJSON", func(t *testing.T) {
		w := httptest.NewRecorder()
		data := map[string]string{"test": "value"}
		sendJSON(w, http.StatusOK, data)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		var result map[string]string
		if err := json.NewDecoder(w.Body).Decode(&result); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if result["test"] != "value" {
			t.Errorf("Expected 'value', got '%s'", result["test"])
		}
	})

	t.Run("sendError", func(t *testing.T) {
		w := httptest.NewRecorder()
		sendError(w, http.StatusBadRequest, "test error")

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", w.Code)
		}

		var result map[string]string
		if err := json.NewDecoder(w.Body).Decode(&result); err != nil {
			t.Fatalf("Failed to decode error response: %v", err)
		}

		if result["error"] != "test error" {
			t.Errorf("Expected 'test error', got '%s'", result["error"])
		}
	})
}

// TestEdgeCases tests edge cases and boundary conditions
func TestEdgeCases(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	server := createTestServer()

	t.Run("EmptyRequestBody", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/code-smell/analyze",
			Body:   "",
		}
		w := executeHandler(server.handleAnalyze, req)
		// Should return bad request
		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400 for empty body, got %d", w.Code)
		}
	})

	t.Run("LargePaths", func(t *testing.T) {
		paths := make([]string, 1000)
		for i := range paths {
			paths[i] = "/test/file" + string(rune(i)) + ".go"
		}
		body := makeJSONBody(map[string]interface{}{
			"paths": paths,
		})
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/code-smell/analyze",
			Body:   body,
		}
		w := executeHandler(server.handleAnalyze, req)
		// Should still process successfully
		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200 for large paths, got %d", w.Code)
		}
	})
}
