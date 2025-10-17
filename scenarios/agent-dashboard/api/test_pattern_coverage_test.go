package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestTestScenarioBuilder tests the test pattern builder
func TestTestScenarioBuilder(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("BuildEmptyScenario", func(t *testing.T) {
		builder := NewTestScenarioBuilder()
		patterns := builder.Build()

		if len(patterns) != 0 {
			t.Errorf("Expected 0 patterns, got %d", len(patterns))
		}
	})

	t.Run("AddInvalidAgentID", func(t *testing.T) {
		builder := NewTestScenarioBuilder()
		builder.AddInvalidAgentID("/api/v1/test")
		patterns := builder.Build()

		if len(patterns) != 1 {
			t.Errorf("Expected 1 pattern, got %d", len(patterns))
		}

		if patterns[0].Name != "InvalidAgentID" {
			t.Errorf("Expected pattern name InvalidAgentID, got %s", patterns[0].Name)
		}
	})

	t.Run("AddInvalidResource", func(t *testing.T) {
		builder := NewTestScenarioBuilder()
		builder.AddInvalidResource("/api/v1/test")
		patterns := builder.Build()

		if len(patterns) != 1 {
			t.Errorf("Expected 1 pattern, got %d", len(patterns))
		}

		if patterns[0].Name != "InvalidResource" {
			t.Errorf("Expected pattern name InvalidResource, got %s", patterns[0].Name)
		}
	})

	t.Run("AddInvalidJSON", func(t *testing.T) {
		builder := NewTestScenarioBuilder()
		builder.AddInvalidJSON("/api/v1/test", "POST")
		patterns := builder.Build()

		if len(patterns) != 1 {
			t.Errorf("Expected 1 pattern, got %d", len(patterns))
		}

		if patterns[0].Name != "InvalidJSON" {
			t.Errorf("Expected pattern name InvalidJSON, got %s", patterns[0].Name)
		}
	})

	t.Run("AddMissingQueryParam", func(t *testing.T) {
		builder := NewTestScenarioBuilder()
		builder.AddMissingQueryParam("/api/v1/test", "capability")
		patterns := builder.Build()

		if len(patterns) != 1 {
			t.Errorf("Expected 1 pattern, got %d", len(patterns))
		}

		if patterns[0].Name != "MissingcapabilityParam" {
			t.Errorf("Expected pattern name MissingcapabilityParam, got %s", patterns[0].Name)
		}
	})

	t.Run("AddInvalidLineCount", func(t *testing.T) {
		builder := NewTestScenarioBuilder()
		builder.AddInvalidLineCount("/api/v1/test")
		patterns := builder.Build()

		if len(patterns) != 1 {
			t.Errorf("Expected 1 pattern, got %d", len(patterns))
		}

		if patterns[0].Name != "InvalidLineCount" {
			t.Errorf("Expected pattern name InvalidLineCount, got %s", patterns[0].Name)
		}
	})

	t.Run("AddCustomPattern", func(t *testing.T) {
		builder := NewTestScenarioBuilder()
		customPattern := ErrorTestPattern{
			Name:           "CustomTest",
			Description:    "Custom test pattern",
			ExpectedStatus: http.StatusTeapot,
		}
		builder.AddCustom(customPattern)
		patterns := builder.Build()

		if len(patterns) != 1 {
			t.Errorf("Expected 1 pattern, got %d", len(patterns))
		}

		if patterns[0].Name != "CustomTest" {
			t.Errorf("Expected pattern name CustomTest, got %s", patterns[0].Name)
		}

		if patterns[0].ExpectedStatus != http.StatusTeapot {
			t.Errorf("Expected status 418, got %d", patterns[0].ExpectedStatus)
		}
	})

	t.Run("ChainedBuilding", func(t *testing.T) {
		builder := NewTestScenarioBuilder().
			AddInvalidAgentID("/api/v1/test").
			AddInvalidJSON("/api/v1/test", "POST").
			AddMissingQueryParam("/api/v1/test", "param1")

		patterns := builder.Build()

		if len(patterns) != 3 {
			t.Errorf("Expected 3 patterns, got %d", len(patterns))
		}
	})
}

// TestHandlerTestSuite tests the handler test suite functionality
func TestHandlerTestSuite(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("RunErrorTests", func(t *testing.T) {
		// Create a simple test handler
		testHandler := func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Query().Get("error") == "true" {
				errorResponse(w, "Test error", http.StatusBadRequest)
				return
			}
			jsonResponse(w, APIResponse{Success: true}, http.StatusOK)
		}

		suite := &HandlerTestSuite{
			HandlerName: "TestHandler",
			Handler:     testHandler,
			BaseURL:     "/api/v1/test",
		}

		// Create a simple error pattern
		patterns := []ErrorTestPattern{
			{
				Name:           "TriggerError",
				Description:    "Test error condition",
				ExpectedStatus: http.StatusBadRequest,
				Execute: func(t *testing.T, setupData interface{}) *HTTPTestRequest {
					return &HTTPTestRequest{
						Method: "GET",
						Path:   "/api/v1/test?error=true",
						QueryParams: map[string]string{
							"error": "true",
						},
					}
				},
			},
		}

		suite.RunErrorTests(t, patterns)
	})

	t.Run("RunErrorTestsWithSetupAndCleanup", func(t *testing.T) {
		setupCalled := false
		cleanupCalled := false

		testHandler := func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}

		suite := &HandlerTestSuite{
			HandlerName: "TestHandler",
			Handler:     testHandler,
			BaseURL:     "/api/v1/test",
		}

		patterns := []ErrorTestPattern{
			{
				Name:           "WithSetupCleanup",
				ExpectedStatus: http.StatusOK,
				Setup: func(t *testing.T) interface{} {
					setupCalled = true
					return "setup_data"
				},
				Execute: func(t *testing.T, setupData interface{}) *HTTPTestRequest {
					if setupData != "setup_data" {
						t.Error("Setup data not passed correctly")
					}
					return &HTTPTestRequest{
						Method: "GET",
						Path:   "/api/v1/test",
					}
				},
				Cleanup: func(setupData interface{}) {
					cleanupCalled = true
				},
			},
		}

		suite.RunErrorTests(t, patterns)

		if !setupCalled {
			t.Error("Setup was not called")
		}

		if !cleanupCalled {
			t.Error("Cleanup was not called")
		}
	})

	t.Run("RunErrorTestsWithValidation", func(t *testing.T) {
		validationCalled := false

		testHandler := func(w http.ResponseWriter, r *http.Request) {
			errorResponse(w, "validation test", http.StatusBadRequest)
		}

		suite := &HandlerTestSuite{
			HandlerName: "TestHandler",
			Handler:     testHandler,
			BaseURL:     "/api/v1/test",
		}

		patterns := []ErrorTestPattern{
			{
				Name:           "WithValidation",
				ExpectedStatus: http.StatusBadRequest,
				Execute: func(t *testing.T, setupData interface{}) *HTTPTestRequest {
					return &HTTPTestRequest{
						Method: "GET",
						Path:   "/api/v1/test",
					}
				},
				Validate: func(t *testing.T, w *httptest.ResponseRecorder, setupData interface{}) {
					validationCalled = true
					assertErrorResponse(t, w, http.StatusBadRequest, "validation test")
				},
			},
		}

		suite.RunErrorTests(t, patterns)

		if !validationCalled {
			t.Error("Validation was not called")
		}
	})
}

// TestErrorPatterns tests individual error patterns
func TestErrorPatterns(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("InvalidAgentIDPattern", func(t *testing.T) {
		pattern := invalidAgentIDPattern("/api/v1/test")

		if pattern.Name != "InvalidAgentID" {
			t.Errorf("Expected name InvalidAgentID, got %s", pattern.Name)
		}

		if pattern.ExpectedStatus != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", pattern.ExpectedStatus)
		}

		req := pattern.Execute(t, nil)
		if req == nil {
			t.Fatal("Execute returned nil request")
		}

		if req.Method != "GET" {
			t.Errorf("Expected GET method, got %s", req.Method)
		}
	})

	t.Run("InvalidResourcePattern", func(t *testing.T) {
		pattern := invalidResourcePattern("/api/v1/test")

		if pattern.Name != "InvalidResource" {
			t.Errorf("Expected name InvalidResource, got %s", pattern.Name)
		}

		req := pattern.Execute(t, nil)
		if req == nil {
			t.Fatal("Execute returned nil request")
		}

		if req.Method != "POST" {
			t.Errorf("Expected POST method, got %s", req.Method)
		}
	})

	t.Run("InvalidJSONPattern", func(t *testing.T) {
		pattern := invalidJSONPattern("/api/v1/test", "PUT")

		if pattern.Name != "InvalidJSON" {
			t.Errorf("Expected name InvalidJSON, got %s", pattern.Name)
		}

		req := pattern.Execute(t, nil)
		if req == nil {
			t.Fatal("Execute returned nil request")
		}

		if req.Method != "PUT" {
			t.Errorf("Expected PUT method, got %s", req.Method)
		}

		// Check that body is malformed JSON
		if bodyStr, ok := req.Body.(string); ok {
			if bodyStr != `{"invalid": "json"` {
				t.Errorf("Expected malformed JSON, got %s", bodyStr)
			}
		} else {
			t.Error("Body is not a string")
		}
	})

	t.Run("MissingQueryParamPattern", func(t *testing.T) {
		pattern := missingQueryParamPattern("/api/v1/test", "capability")

		if pattern.Name != "MissingcapabilityParam" {
			t.Errorf("Expected name MissingcapabilityParam, got %s", pattern.Name)
		}

		req := pattern.Execute(t, nil)
		if req == nil {
			t.Fatal("Execute returned nil request")
		}

		if req.QueryParams != nil && len(req.QueryParams) > 0 {
			t.Error("Expected no query params for missing param pattern")
		}
	})

	t.Run("InvalidLineCountPattern", func(t *testing.T) {
		pattern := invalidLineCountPattern("/api/v1/test")

		if pattern.Name != "InvalidLineCount" {
			t.Errorf("Expected name InvalidLineCount, got %s", pattern.Name)
		}

		req := pattern.Execute(t, nil)
		if req == nil {
			t.Fatal("Execute returned nil request")
		}

		if req.QueryParams == nil {
			t.Fatal("Expected query params for line count pattern")
		}

		lines, exists := req.QueryParams["lines"]
		if !exists {
			t.Error("Expected lines query param")
		}

		if lines != "99999" {
			t.Errorf("Expected lines to be 99999, got %s", lines)
		}
	})
}

// TestMakeHTTPRequest tests the HTTP request helper with various body types
func TestMakeHTTPRequest(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("WithStringBody", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/test",
			Body:   "test body",
		}

		w, httpReq, err := makeHTTPRequest(req)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		if w == nil {
			t.Error("Expected response recorder to be created")
		}

		if httpReq == nil {
			t.Error("Expected HTTP request to be created")
		}

		if httpReq.Header.Get("Content-Type") != "application/json" {
			t.Errorf("Expected Content-Type application/json, got %s", httpReq.Header.Get("Content-Type"))
		}
	})

	t.Run("WithByteSliceBody", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/test",
			Body:   []byte("test bytes"),
		}

		_, httpReq, err := makeHTTPRequest(req)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		if httpReq == nil {
			t.Error("Expected HTTP request to be created")
		}
	})

	t.Run("WithCustomHeaders", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/test",
			Headers: map[string]string{
				"X-Custom-Header": "custom-value",
			},
		}

		_, httpReq, err := makeHTTPRequest(req)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		if httpReq.Header.Get("X-Custom-Header") != "custom-value" {
			t.Errorf("Expected custom header, got %s", httpReq.Header.Get("X-Custom-Header"))
		}
	})

	t.Run("WithQueryParams", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/test",
			QueryParams: map[string]string{
				"param1": "value1",
				"param2": "value2",
			},
		}

		_, httpReq, err := makeHTTPRequest(req)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		if httpReq.URL.Query().Get("param1") != "value1" {
			t.Errorf("Expected param1=value1, got %s", httpReq.URL.Query().Get("param1"))
		}

		if httpReq.URL.Query().Get("param2") != "value2" {
			t.Errorf("Expected param2=value2, got %s", httpReq.URL.Query().Get("param2"))
		}
	})

	t.Run("WithoutBody", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/test",
		}

		_, httpReq, err := makeHTTPRequest(req)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		if httpReq == nil {
			t.Error("Expected HTTP request to be created")
		}
	})
}

// TestAssertJSONResponse tests the JSON response assertion helper
func TestAssertJSONResponse(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("ValidResponse", func(t *testing.T) {
		w := httptest.NewRecorder()
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"success": true, "message": "test"}`))

		response := assertJSONResponse(t, w, http.StatusOK, map[string]interface{}{
			"success": true,
		})

		if response == nil {
			t.Error("Expected response map to be returned")
		}
	})

	t.Run("CorrectStatus", func(t *testing.T) {
		w := httptest.NewRecorder()
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error": "bad request"}`))

		// This should parse successfully with matching status
		response := assertJSONResponse(t, w, http.StatusBadRequest, nil)

		if response == nil {
			t.Error("Expected response to be parsed")
		}
	})
}

// TestAssertErrorResponse tests the error response assertion helper
func TestAssertErrorResponse(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("ValidErrorResponse", func(t *testing.T) {
		w := httptest.NewRecorder()
		errorResponse(w, "test error message", http.StatusBadRequest)

		// This should not fail
		assertErrorResponse(t, w, http.StatusBadRequest, "test error")
	})

	t.Run("MatchingErrorMessage", func(t *testing.T) {
		w := httptest.NewRecorder()
		errorResponse(w, "expected error message", http.StatusBadRequest)

		// This should match successfully
		assertErrorResponse(t, w, http.StatusBadRequest, "expected error")
	})
}
