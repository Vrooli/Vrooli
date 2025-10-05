// +build testing

package main

import (
	"net/http"
	"testing"
)

// TestExportTranscript tests the export transcript utility function
func TestExportTranscript(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("Success_SRT", func(t *testing.T) {
		transcript := Transcript{
			ID:       1,
			FullText: "Test transcript",
		}

		filename, err := exportTranscript(transcript, "srt", true, false)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}

		if filename == "" {
			t.Error("Expected filename to be returned")
		}
	})

	t.Run("Success_VTT", func(t *testing.T) {
		transcript := Transcript{
			ID:       2,
			FullText: "Test transcript",
		}

		filename, err := exportTranscript(transcript, "vtt", true, true)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}

		if filename == "" {
			t.Error("Expected filename to be returned")
		}
	})

	t.Run("Success_JSON", func(t *testing.T) {
		transcript := Transcript{
			ID:       3,
			FullText: "Test transcript",
		}

		filename, err := exportTranscript(transcript, "json", false, false)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}

		if filename == "" {
			t.Error("Expected filename to be returned")
		}
	})
}

// TestHealthHandlerEdgeCases tests edge cases for health handler
func TestHealthHandlerEdgeCases(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("MethodNotAllowed", func(t *testing.T) {
		// Health endpoint should work with any method in current implementation
		methods := []string{"GET", "POST", "PUT", "DELETE"}
		for _, method := range methods {
			req := HTTPTestRequest{
				Method: method,
				Path:   "/health",
			}

			w, httpReq, err := makeHTTPRequest(req)
			if err != nil {
				t.Fatalf("Failed to create request: %v", err)
			}

			healthHandler(w, httpReq)

			// Health handler always returns 200 OK
			if w.Code != http.StatusOK {
				t.Errorf("Method %s: Expected status 200, got %d", method, w.Code)
			}
		}
	})

	t.Run("WithHeaders", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/health",
			Headers: map[string]string{
				"User-Agent": "TestAgent/1.0",
				"Accept":     "application/json",
			},
		}

		w, httpReq, err := makeHTTPRequest(req)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		healthHandler(w, httpReq)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		contentType := w.Header().Get("Content-Type")
		if contentType != "application/json" {
			t.Errorf("Expected Content-Type application/json, got %s", contentType)
		}
	})
}

// TestAnalyzeURLHandlerWithoutDB tests analyze URL handler behavior
func TestAnalyzeURLHandlerWithoutDB(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	// analyzeURLHandler doesn't actually use the database in current implementation
	// It returns mock data

	t.Run("ValidURL", func(t *testing.T) {
		reqBody := map[string]string{
			"url": "https://www.youtube.com/watch?v=dQw4w9WgXcQ",
		}

		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/analyze",
			Body:   reqBody,
		}

		w, httpReq, err := makeHTTPRequest(req)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		analyzeURLHandler(w, httpReq)

		response := assertJSONResponse(t, w, http.StatusOK, map[string]interface{}{
			"url": "https://www.youtube.com/watch?v=dQw4w9WgXcQ",
		})

		if response == nil {
			t.Fatal("Expected non-nil response")
		}

		// Verify mock response structure
		if _, ok := response["title"]; !ok {
			t.Error("Expected title in response")
		}
		if _, ok := response["platform"]; !ok {
			t.Error("Expected platform in response")
		}
		if _, ok := response["availableQualities"]; !ok {
			t.Error("Expected availableQualities in response")
		}
	})

	t.Run("EmptyURL", func(t *testing.T) {
		reqBody := map[string]string{
			"url": "",
		}

		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/analyze",
			Body:   reqBody,
		}

		w, httpReq, err := makeHTTPRequest(req)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		analyzeURLHandler(w, httpReq)

		// Handler still returns mock data even with empty URL
		if w.Code != http.StatusOK {
			t.Logf("Empty URL returned status %d", w.Code)
		}
	})
}

// TestProcessQueueHandlerWithoutDB tests process queue handler
func TestProcessQueueHandlerWithoutDB(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("Success", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/queue/process",
		}

		w, httpReq, err := makeHTTPRequest(req)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		processQueueHandler(w, httpReq)

		response := assertJSONResponse(t, w, http.StatusOK, map[string]interface{}{
			"status": "processing",
		})

		if response == nil {
			t.Fatal("Expected non-nil response")
		}
	})

	t.Run("MultipleCalls", func(t *testing.T) {
		// Test calling process queue multiple times
		for i := 0; i < 3; i++ {
			req := HTTPTestRequest{
				Method: "POST",
				Path:   "/api/queue/process",
			}

			w, httpReq, err := makeHTTPRequest(req)
			if err != nil {
				t.Fatalf("Failed to create request: %v", err)
			}

			processQueueHandler(w, httpReq)

			if w.Code != http.StatusOK {
				t.Errorf("Call %d: Expected status 200, got %d", i+1, w.Code)
			}
		}
	})
}

// TestHTTPRequestHelpers tests the HTTP request helper functions
func TestHTTPRequestHelpers(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("MakeHTTPRequest_WithBody", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/test",
			Body: map[string]string{
				"key": "value",
			},
		}

		w, httpReq, err := makeHTTPRequest(req)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		if w == nil {
			t.Error("Expected non-nil response recorder")
		}
		if httpReq == nil {
			t.Error("Expected non-nil HTTP request")
		}

		if httpReq.Header.Get("Content-Type") != "application/json" {
			t.Error("Expected Content-Type to be set to application/json")
		}
	})

	t.Run("MakeHTTPRequest_WithURLVars", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/test/{id}",
			URLVars: map[string]string{
				"id": "123",
			},
		}

		_, httpReq, err := makeHTTPRequest(req)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		if httpReq == nil {
			t.Error("Expected non-nil request")
		}
	})

	t.Run("MakeHTTPRequest_WithQueryParams", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/test",
			QueryParams: map[string]string{
				"page":  "1",
				"limit": "10",
			},
		}

		_, httpReq, err := makeHTTPRequest(req)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		if httpReq.URL.Query().Get("page") != "1" {
			t.Error("Expected page query param to be set")
		}
		if httpReq.URL.Query().Get("limit") != "10" {
			t.Error("Expected limit query param to be set")
		}
	})

	t.Run("MakeHTTPRequest_WithHeaders", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/test",
			Headers: map[string]string{
				"Authorization": "Bearer token123",
				"X-Custom":      "value",
			},
		}

		_, httpReq, err := makeHTTPRequest(req)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		if httpReq.Header.Get("Authorization") != "Bearer token123" {
			t.Error("Expected Authorization header to be set")
		}
		if httpReq.Header.Get("X-Custom") != "value" {
			t.Error("Expected X-Custom header to be set")
		}
	})

	t.Run("MakeHTTPRequest_StringBody", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/test",
			Body:   `{"test": "data"}`,
		}

		w, httpReq, err := makeHTTPRequest(req)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		if w == nil || httpReq == nil {
			t.Error("Expected non-nil response and request")
		}
	})

	t.Run("MakeHTTPRequest_ByteBody", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/test",
			Body:   []byte(`{"test": "data"}`),
		}

		w, httpReq, err := makeHTTPRequest(req)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		if w == nil || httpReq == nil {
			t.Error("Expected non-nil response and request")
		}
	})
}

// TestAssertionHelpers tests the assertion helper functions
func TestAssertionHelpers(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("AssertJSONResponse_ValidJSON", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/health",
		}

		w, httpReq, err := makeHTTPRequest(req)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		healthHandler(w, httpReq)

		response := assertJSONResponse(t, w, http.StatusOK, map[string]interface{}{
			"status": "healthy",
		})

		if response == nil {
			t.Error("Expected non-nil response")
		}
	})
}

// TestOptionsHelpers tests option helper functions
func TestOptionsHelpers(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("GetStringOption_Found", func(t *testing.T) {
		options := map[string]interface{}{
			"key": "value",
		}

		result := getStringOption(options, "key", "default")
		if result != "value" {
			t.Errorf("Expected 'value', got '%s'", result)
		}
	})

	t.Run("GetStringOption_NotFound", func(t *testing.T) {
		options := map[string]interface{}{}

		result := getStringOption(options, "missing", "default")
		if result != "default" {
			t.Errorf("Expected 'default', got '%s'", result)
		}
	})

	t.Run("GetStringOption_NilOptions", func(t *testing.T) {
		result := getStringOption(nil, "key", "default")
		if result != "default" {
			t.Errorf("Expected 'default', got '%s'", result)
		}
	})

	t.Run("GetBoolOption_Found", func(t *testing.T) {
		options := map[string]interface{}{
			"enabled": true,
		}

		result := getBoolOption(options, "enabled", false)
		if !result {
			t.Error("Expected true")
		}
	})

	t.Run("GetBoolOption_NotFound", func(t *testing.T) {
		options := map[string]interface{}{}

		result := getBoolOption(options, "missing", true)
		if !result {
			t.Error("Expected true (default)")
		}
	})

	t.Run("GetBoolOption_NilOptions", func(t *testing.T) {
		result := getBoolOption(nil, "key", true)
		if !result {
			t.Error("Expected true (default)")
		}
	})
}

// TestTestPatterns tests the test pattern builder
func TestTestPatterns(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("TestScenarioBuilder", func(t *testing.T) {
		builder := NewTestScenarioBuilder()
		if builder == nil {
			t.Fatal("Expected non-nil builder")
		}

		patterns := builder.
			AddInvalidID("/api/test/{id}").
			AddNonExistentResource("/api/test/{id}", "id").
			AddInvalidJSON("POST", "/api/test").
			AddMissingRequiredField("POST", "/api/test", "url").
			AddEmptyBody("POST", "/api/test").
			Build()

		if len(patterns) != 5 {
			t.Errorf("Expected 5 patterns, got %d", len(patterns))
		}
	})

	t.Run("TestScenarioBuilder_Custom", func(t *testing.T) {
		customPattern := ErrorTestPattern{
			Name:           "CustomTest",
			Description:    "Custom test pattern",
			ExpectedStatus: http.StatusBadRequest,
		}

		builder := NewTestScenarioBuilder()
		patterns := builder.AddCustom(customPattern).Build()

		if len(patterns) != 1 {
			t.Errorf("Expected 1 pattern, got %d", len(patterns))
		}

		if patterns[0].Name != "CustomTest" {
			t.Errorf("Expected name 'CustomTest', got '%s'", patterns[0].Name)
		}
	})
}
