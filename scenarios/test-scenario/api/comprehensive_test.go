package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestAssertHelpersCoverage improves coverage for assertion helpers
func TestAssertHelpersCoverage(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("AssertJSONResponse_SuccessCase", func(t *testing.T) {
		w := httptest.NewRecorder()
		w.WriteHeader(200)
		w.Write([]byte(`{"foo": "bar", "status": "ok"}`))

		// This should succeed
		response := assertJSONResponse(t, w, 200, map[string]interface{}{
			"foo":    "bar",
			"status": "ok",
		})

		if response == nil {
			t.Error("Expected response to be parsed")
		}
	})

	t.Run("AssertJSONResponse_WithNilExpectedFields", func(t *testing.T) {
		w := httptest.NewRecorder()
		w.WriteHeader(200)
		w.Write([]byte(`{"foo": "bar"}`))

		// Test with nil expected fields (should just parse without validation)
		response := assertJSONResponse(t, w, 200, nil)

		if response == nil {
			t.Error("Expected response to be parsed")
		}
	})

	t.Run("AssertErrorResponse_SuccessCase", func(t *testing.T) {
		w := httptest.NewRecorder()
		w.WriteHeader(400)
		w.Write([]byte(`{"error": "test error message"}`))

		// This should succeed
		assertErrorResponse(t, w, 400, "test error")
	})

	t.Run("AssertErrorResponse_WithEmptyMessage", func(t *testing.T) {
		w := httptest.NewRecorder()
		w.WriteHeader(400)
		w.Write([]byte(`{"error": "any error"}`))

		// Should work with empty expected message (no validation)
		assertErrorResponse(t, w, 400, "")
	})

	t.Run("AssertErrorResponse_NonJSONPlainText", func(t *testing.T) {
		w := httptest.NewRecorder()
		w.WriteHeader(400)
		w.Write([]byte(`Plain text error message`))

		// Should handle non-JSON error responses
		assertErrorResponse(t, w, 400, "error message")
	})

	t.Run("AssertTextResponse_SuccessCase", func(t *testing.T) {
		w := httptest.NewRecorder()
		w.WriteHeader(200)
		w.Write([]byte(`Expected content here`))

		// This should pass
		assertTextResponse(t, w, 200, "Expected content")
	})

	t.Run("AssertTextResponse_WithEmptyExpectedContent", func(t *testing.T) {
		w := httptest.NewRecorder()
		w.WriteHeader(200)
		w.Write([]byte(`Any content`))

		// Should work with empty expected content (no validation)
		assertTextResponse(t, w, 200, "")
	})
}

// TestMakeHTTPRequestCoverage improves coverage for HTTP request creation
func TestMakeHTTPRequestCoverage(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("MakeHTTPRequest_WithCustomHeaders", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/test",
			Headers: map[string]string{
				"X-Custom-Header": "custom-value",
				"Authorization":   "Bearer token123",
			},
		}
		w, httpReq, err := makeHTTPRequest(req)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		if w == nil || httpReq == nil {
			t.Error("Expected request to be created")
		}

		if httpReq.Header.Get("X-Custom-Header") != "custom-value" {
			t.Error("Expected custom header to be set")
		}

		if httpReq.Header.Get("Authorization") != "Bearer token123" {
			t.Error("Expected authorization header to be set")
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
		w, httpReq, err := makeHTTPRequest(req)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		if w == nil || httpReq == nil {
			t.Error("Expected request to be created")
		}

		if httpReq.URL.Query().Get("page") != "1" {
			t.Error("Expected page query param to be set")
		}

		if httpReq.URL.Query().Get("limit") != "10" {
			t.Error("Expected limit query param to be set")
		}
	})

	t.Run("MakeHTTPRequest_WithByteBody", func(t *testing.T) {
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
			t.Error("Expected request to be created")
		}

		if httpReq.Header.Get("Content-Type") != "application/json" {
			t.Error("Expected Content-Type header to be set for POST request")
		}
	})

	t.Run("MakeHTTPRequest_WithStringBody", func(t *testing.T) {
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
			t.Error("Expected request to be created")
		}
	})

	t.Run("MakeHTTPRequest_WithStructBody", func(t *testing.T) {
		type TestData struct {
			Name  string `json:"name"`
			Value int    `json:"value"`
		}

		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/test",
			Body: TestData{
				Name:  "test",
				Value: 123,
			},
		}
		w, httpReq, err := makeHTTPRequest(req)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		if w == nil || httpReq == nil {
			t.Error("Expected request to be created")
		}
	})

	t.Run("MakeHTTPRequest_NoBody", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/test",
			Body:   nil,
		}
		respRecorder, httpReq, err := makeHTTPRequest(req)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		if respRecorder == nil || httpReq == nil {
			t.Error("Expected request to be created")
		}
	})

	t.Run("MakeHTTPRequest_CustomContentType", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/test",
			Body:   `test data`,
			Headers: map[string]string{
				"Content-Type": "text/plain",
			},
		}
		_, httpReq, err := makeHTTPRequest(req)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		if httpReq.Header.Get("Content-Type") != "text/plain" {
			t.Error("Expected custom Content-Type to be preserved")
		}
	})
}

// TestSetupTestDirectoryCoverage improves coverage for test directory setup
func TestSetupTestDirectoryCoverage(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("SetupTestDirectory_VerifyCleanup", func(t *testing.T) {
		env := setupTestDirectory(t)

		// Verify the temp directory exists
		if env.TempDir == "" {
			t.Fatal("Expected temp directory to be created")
		}

		// Verify cleanup function is set
		if env.Cleanup == nil {
			t.Fatal("Expected cleanup function to be set")
		}

		// Call cleanup
		env.Cleanup()
	})
}

// TestHandlerTestSuiteLifecycle validates full handler lifecycle coverage using the pattern helpers
func TestHandlerTestSuiteLifecycle(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	t.Run("LifecyclePatterns", func(t *testing.T) {
		var (
			setupCalled      bool
			cleanupCalled    bool
			invalidValidated bool
			missingValidated bool
			successValidated bool
		)

		suite := &HandlerTestSuite{
			HandlerName: "LifecycleHandler",
			BaseURL:     "/api",
			Handler: func(w http.ResponseWriter, r *http.Request) {
				defer r.Body.Close()

				var payload map[string]interface{}
				if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
					w.WriteHeader(http.StatusBadRequest)
					json.NewEncoder(w).Encode(map[string]string{"error": "invalid json"})
					return
				}

				nameValue, _ := payload["name"].(string)
				nameValue = strings.TrimSpace(nameValue)
				if nameValue == "" {
					w.WriteHeader(http.StatusBadRequest)
					json.NewEncoder(w).Encode(map[string]string{"error": "missing name"})
					return
				}

				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(map[string]string{
					"status": "ok",
					"name":   nameValue,
				})
			},
		}

		builder := NewTestScenarioBuilder()
		builder.AddCustom(ErrorTestPattern{
			Name:           "InvalidJSONLifecycle",
			Description:    "Invalid JSON payload should return a bad request",
			ExpectedStatus: http.StatusBadRequest,
			Setup: func(t *testing.T) interface{} {
				setupCalled = true
				return map[string]string{"scenario": "invalid-json"}
			},
			Execute: func(t *testing.T, setupData interface{}) *HTTPTestRequest {
				if !setupCalled {
					t.Fatalf("setup was not called before execute: %#v", setupData)
				}
				return &HTTPTestRequest{
					Method: "POST",
					Path:   suite.BaseURL + "/resources",
					Body:   `{"name": "missing quote}`,
				}
			},
			Validate: func(t *testing.T, w *httptest.ResponseRecorder, setupData interface{}) {
				invalidValidated = true
				assertErrorResponse(t, w, http.StatusBadRequest, "invalid json")
				dataMap, ok := setupData.(map[string]string)
				if !ok || dataMap["scenario"] != "invalid-json" {
					t.Errorf("unexpected setup data in validation: %#v", setupData)
				}
			},
			Cleanup: func(data interface{}) {
				cleanupCalled = true
				if dataMap, ok := data.(map[string]string); !ok || dataMap["scenario"] != "invalid-json" {
					t.Errorf("unexpected cleanup data: %#v", data)
				}
			},
		})

		builder.AddCustom(ErrorTestPattern{
			Name:           "MissingNameEdgeCase",
			Description:    "Whitespace-only names should be rejected",
			ExpectedStatus: http.StatusBadRequest,
			Execute: func(t *testing.T, setupData interface{}) *HTTPTestRequest {
				return &HTTPTestRequest{
					Method: "POST",
					Path:   suite.BaseURL + "/resources",
					Body:   map[string]string{"name": "  "},
				}
			},
			Validate: func(t *testing.T, w *httptest.ResponseRecorder, setupData interface{}) {
				missingValidated = true
				assertErrorResponse(t, w, http.StatusBadRequest, "missing name")
			},
		})

		builder.AddCustom(ErrorTestPattern{
			Name:           "ValidPayloadSuccess",
			Description:    "Valid payload should succeed",
			ExpectedStatus: http.StatusOK,
			Execute: func(t *testing.T, setupData interface{}) *HTTPTestRequest {
				return &HTTPTestRequest{
					Method: "POST",
					Path:   suite.BaseURL + "/resources",
					Body:   map[string]string{"name": "LifecycleTester"},
				}
			},
			Validate: func(t *testing.T, w *httptest.ResponseRecorder, setupData interface{}) {
				successValidated = true
				response := assertJSONResponse(t, w, http.StatusOK, map[string]interface{}{"status": "ok"})
				if response != nil {
					if responseName, ok := response["name"].(string); !ok || responseName != "LifecycleTester" {
						t.Errorf("expected response name 'LifecycleTester', got %v", response["name"])
					}
				}
			},
		})

		suite.RunErrorTests(t, builder.Build())

		if !setupCalled {
			t.Error("expected setup to run before executing the pattern")
		}
		if !cleanupCalled {
			t.Error("expected cleanup to run after executing the pattern")
		}
		if !invalidValidated {
			t.Error("expected invalid JSON validation to run")
		}
		if !missingValidated {
			t.Error("expected missing name validation to run")
		}
		if !successValidated {
			t.Error("expected success validation to run")
		}
	})
}
