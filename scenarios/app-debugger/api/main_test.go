package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHealthHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("Success", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/health", nil)
		w := httptest.NewRecorder()

		healthHandler(w, req)

		data := assertJSONResponse(t, w, http.StatusOK)

		// Validate response structure
		assertMapKeyExists(t, data, "status")
		assertMapKeyExists(t, data, "apps")
		assertMapKeyExists(t, data, "message")

		// Validate values
		if data["status"] != "healthy" {
			t.Errorf("Expected status 'healthy', got %v", data["status"])
		}
	})

	t.Run("MethodNotAllowed", func(t *testing.T) {
		// Health endpoint should accept all methods
		methods := []string{"POST", "PUT", "DELETE"}
		for _, method := range methods {
			req := httptest.NewRequest(method, "/health", nil)
			w := httptest.NewRecorder()

			healthHandler(w, req)

			// Health handler doesn't validate methods, should still return 200
			if w.Code != http.StatusOK {
				t.Errorf("Health handler should accept %s method, got status %d", method, w.Code)
			}
		}
	})
}

func TestDebugHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("Success_Performance", func(t *testing.T) {
		debugReq := DebugRequest{
			AppName:   "test-app",
			DebugType: "performance",
		}

		body, _ := json.Marshal(debugReq)
		req := httptest.NewRequest("POST", "/api/debug", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		debugHandler(w, req)

		data := assertJSONResponse(t, w, http.StatusOK)

		// Validate debug session response
		assertMapKeyExists(t, data, "id")
		assertMapKeyExists(t, data, "app_name")
		assertMapKeyExists(t, data, "debug_type")
		assertMapKeyExists(t, data, "status")

		if data["app_name"] != "test-app" {
			t.Errorf("Expected app_name 'test-app', got %v", data["app_name"])
		}
		if data["debug_type"] != "performance" {
			t.Errorf("Expected debug_type 'performance', got %v", data["debug_type"])
		}
	})

	t.Run("Success_Error", func(t *testing.T) {
		debugReq := DebugRequest{
			AppName:   "test-app",
			DebugType: "error",
		}

		body, _ := json.Marshal(debugReq)
		req := httptest.NewRequest("POST", "/api/debug", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		debugHandler(w, req)

		data := assertJSONResponse(t, w, http.StatusOK)
		assertMapKeyExists(t, data, "id")
		if data["debug_type"] != "error" {
			t.Errorf("Expected debug_type 'error', got %v", data["debug_type"])
		}
	})

	t.Run("Success_Logs", func(t *testing.T) {
		debugReq := DebugRequest{
			AppName:   "test-app",
			DebugType: "logs",
		}

		body, _ := json.Marshal(debugReq)
		req := httptest.NewRequest("POST", "/api/debug", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		debugHandler(w, req)

		data := assertJSONResponse(t, w, http.StatusOK)
		assertMapKeyExists(t, data, "id")
		if data["debug_type"] != "logs" {
			t.Errorf("Expected debug_type 'logs', got %v", data["debug_type"])
		}
	})

	t.Run("Success_Health", func(t *testing.T) {
		debugReq := DebugRequest{
			AppName:   "test-app",
			DebugType: "health",
		}

		body, _ := json.Marshal(debugReq)
		req := httptest.NewRequest("POST", "/api/debug", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		debugHandler(w, req)

		data := assertJSONResponse(t, w, http.StatusOK)
		assertMapKeyExists(t, data, "id")
		if data["debug_type"] != "health" {
			t.Errorf("Expected debug_type 'health', got %v", data["debug_type"])
		}
	})

	t.Run("Success_General", func(t *testing.T) {
		debugReq := DebugRequest{
			AppName:   "test-app",
			DebugType: "general",
		}

		body, _ := json.Marshal(debugReq)
		req := httptest.NewRequest("POST", "/api/debug", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		debugHandler(w, req)

		data := assertJSONResponse(t, w, http.StatusOK)
		assertMapKeyExists(t, data, "id")
	})

	t.Run("MethodNotAllowed_GET", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/debug", nil)
		w := httptest.NewRecorder()

		debugHandler(w, req)

		assertErrorResponse(t, w, http.StatusMethodNotAllowed)
	})

	t.Run("MethodNotAllowed_PUT", func(t *testing.T) {
		req := httptest.NewRequest("PUT", "/api/debug", nil)
		w := httptest.NewRecorder()

		debugHandler(w, req)

		assertErrorResponse(t, w, http.StatusMethodNotAllowed)
	})

	t.Run("MalformedJSON", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/debug", bytes.NewReader([]byte("invalid json")))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		debugHandler(w, req)

		assertErrorResponse(t, w, http.StatusBadRequest)
	})

	t.Run("EmptyBody", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/debug", nil)
		w := httptest.NewRecorder()

		debugHandler(w, req)

		assertErrorResponse(t, w, http.StatusBadRequest)
	})

	t.Run("EmptyAppName", func(t *testing.T) {
		debugReq := DebugRequest{
			AppName:   "",
			DebugType: "performance",
		}

		body, _ := json.Marshal(debugReq)
		req := httptest.NewRequest("POST", "/api/debug", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		debugHandler(w, req)

		// Should still succeed but might have issues in debug manager
		// This tests that the handler doesn't crash
		if w.Code != http.StatusOK && w.Code != http.StatusInternalServerError {
			t.Errorf("Unexpected status code: %d", w.Code)
		}
	})
}

func TestReportErrorHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("Success", func(t *testing.T) {
		errorReport := ErrorReport{
			AppName:      "test-app",
			ErrorMessage: "Test error message",
			StackTrace:   "line 1\nline 2\nline 3",
			Context: map[string]interface{}{
				"version": "1.0.0",
				"env":     "test",
			},
		}

		body, _ := json.Marshal(errorReport)
		req := httptest.NewRequest("POST", "/api/report-error", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		reportErrorHandler(w, req)

		data := assertJSONResponse(t, w, http.StatusOK)

		assertMapKeyExists(t, data, "status")
		assertMapKeyExists(t, data, "app_name")
		assertMapKeyExists(t, data, "error_type")

		if data["status"] != "received" {
			t.Errorf("Expected status 'received', got %v", data["status"])
		}
		if data["app_name"] != "test-app" {
			t.Errorf("Expected app_name 'test-app', got %v", data["app_name"])
		}
	})

	t.Run("MethodNotAllowed_GET", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/report-error", nil)
		w := httptest.NewRecorder()

		reportErrorHandler(w, req)

		assertErrorResponse(t, w, http.StatusMethodNotAllowed)
	})

	t.Run("MalformedJSON", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/report-error", bytes.NewReader([]byte("bad json")))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		reportErrorHandler(w, req)

		assertErrorResponse(t, w, http.StatusBadRequest)
	})

	t.Run("EmptyBody", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/report-error", nil)
		w := httptest.NewRecorder()

		reportErrorHandler(w, req)

		assertErrorResponse(t, w, http.StatusBadRequest)
	})

	t.Run("MinimalReport", func(t *testing.T) {
		errorReport := ErrorReport{
			AppName: "minimal-app",
		}

		body, _ := json.Marshal(errorReport)
		req := httptest.NewRequest("POST", "/api/report-error", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		reportErrorHandler(w, req)

		data := assertJSONResponse(t, w, http.StatusOK)
		if data["app_name"] != "minimal-app" {
			t.Errorf("Expected app_name 'minimal-app', got %v", data["app_name"])
		}
	})
}

func TestListAppsHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("Success", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/apps", nil)
		w := httptest.NewRecorder()

		listAppsHandler(w, req)

		// Response should be valid JSON
		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		var apps []map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &apps); err != nil {
			t.Errorf("Failed to parse JSON response: %v", err)
		}

		// Should return at least some mock apps
		if len(apps) == 0 {
			t.Error("Expected at least one app in the list")
		}

		// Validate app structure
		for _, app := range apps {
			assertMapKeyExists(t, app, "name")
			assertMapKeyExists(t, app, "status")
			assertMapKeyExists(t, app, "health")
		}
	})

	t.Run("AcceptsAllMethods", func(t *testing.T) {
		// List apps should work with any HTTP method in current implementation
		methods := []string{"GET", "POST", "PUT", "DELETE"}
		for _, method := range methods {
			req := httptest.NewRequest(method, "/api/apps", nil)
			w := httptest.NewRecorder()

			listAppsHandler(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("Expected status 200 for %s method, got %d", method, w.Code)
			}
		}
	})
}

func TestHTTPHandlersIntegration(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("FullDebugWorkflow", func(t *testing.T) {
		// 1. Check health
		healthReq := httptest.NewRequest("GET", "/health", nil)
		healthW := httptest.NewRecorder()
		healthHandler(healthW, healthReq)

		if healthW.Code != http.StatusOK {
			t.Fatalf("Health check failed: %d", healthW.Code)
		}

		// 2. List apps
		appsReq := httptest.NewRequest("GET", "/api/apps", nil)
		appsW := httptest.NewRecorder()
		listAppsHandler(appsW, appsReq)

		if appsW.Code != http.StatusOK {
			t.Fatalf("List apps failed: %d", appsW.Code)
		}

		// 3. Start debug session
		debugReq := DebugRequest{
			AppName:   "test-app",
			DebugType: "performance",
		}
		body, _ := json.Marshal(debugReq)
		debugHTTPReq := httptest.NewRequest("POST", "/api/debug", bytes.NewReader(body))
		debugHTTPReq.Header.Set("Content-Type", "application/json")
		debugW := httptest.NewRecorder()
		debugHandler(debugW, debugHTTPReq)

		if debugW.Code != http.StatusOK {
			t.Fatalf("Debug session failed: %d", debugW.Code)
		}

		// 4. Report error
		errorReport := ErrorReport{
			AppName:      "test-app",
			ErrorMessage: "Integration test error",
		}
		errorBody, _ := json.Marshal(errorReport)
		errorReq := httptest.NewRequest("POST", "/api/report-error", bytes.NewReader(errorBody))
		errorReq.Header.Set("Content-Type", "application/json")
		errorW := httptest.NewRecorder()
		reportErrorHandler(errorW, errorReq)

		if errorW.Code != http.StatusOK {
			t.Fatalf("Error report failed: %d", errorW.Code)
		}
	})
}

func TestEdgeCases(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("VeryLongAppName", func(t *testing.T) {
		longName := ""
		for i := 0; i < 1000; i++ {
			longName += "a"
		}

		debugReq := DebugRequest{
			AppName:   longName,
			DebugType: "performance",
		}

		body, _ := json.Marshal(debugReq)
		req := httptest.NewRequest("POST", "/api/debug", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		debugHandler(w, req)

		// Should handle gracefully without crashing
		if w.Code != http.StatusOK && w.Code != http.StatusInternalServerError {
			t.Errorf("Unexpected status for long app name: %d", w.Code)
		}
	})

	t.Run("SpecialCharactersInAppName", func(t *testing.T) {
		debugReq := DebugRequest{
			AppName:   "../../../etc/passwd",
			DebugType: "performance",
		}

		body, _ := json.Marshal(debugReq)
		req := httptest.NewRequest("POST", "/api/debug", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		debugHandler(w, req)

		// Should handle path traversal attempts safely
		// The sanitizePathComponent function should prevent this
		if w.Code != http.StatusOK && w.Code != http.StatusInternalServerError {
			t.Errorf("Unexpected status for path traversal: %d", w.Code)
		}
	})

	t.Run("NullContextInErrorReport", func(t *testing.T) {
		errorReport := ErrorReport{
			AppName:      "test-app",
			ErrorMessage: "Test error",
			Context:      nil,
		}

		body, _ := json.Marshal(errorReport)
		req := httptest.NewRequest("POST", "/api/report-error", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		reportErrorHandler(w, req)

		// Should handle nil context gracefully
		assertJSONResponse(t, w, http.StatusOK)
	})
}
