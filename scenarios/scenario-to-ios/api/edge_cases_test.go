
package main

import (
	"net/http"
	"testing"
)

// TestEdgeCases_HelperFunctions tests edge cases for helper functions
func TestEdgeCases_HelperFunctions(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("SetupTestDirectoryCleanup", func(t *testing.T) {
		env := setupTestDirectory(t)

		// Verify environment is set up
		if env.TempDir == "" {
			t.Error("Expected temp dir to be set")
		}

		// Call cleanup
		env.Cleanup()

		// Verify cleanup doesn't panic on second call
		env.Cleanup()
	})

	t.Run("MakeHTTPRequestWithNilBody", func(t *testing.T) {
		w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/health",
			Body:   nil,
		})

		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		if w == nil {
			t.Error("Expected non-nil response recorder")
		}

		if httpReq == nil {
			t.Error("Expected non-nil HTTP request")
		}
	})

	t.Run("AssertJSONResponseWithCorrectStatus", func(t *testing.T) {
		s := setupTestServer(t)

		w := testHandlerWithRequest(t, s.handleHealth, HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/health",
		})

		// Test with correct status
		response := assertJSONResponse(t, w, http.StatusOK, nil)

		if response == nil {
			t.Error("Response should not be nil with correct status")
		}
	})

	t.Run("AssertContentTypeCorrect", func(t *testing.T) {
		s := setupTestServer(t)

		w := testHandlerWithRequest(t, s.handleHealth, HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/health",
		})

		// Test with correct content type
		assertContentType(t, w, "application/json")
	})

	t.Run("AssertErrorResponseWithSuccessStatus", func(t *testing.T) {
		s := setupTestServer(t)

		w := testHandlerWithRequest(t, s.handleHealth, HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/health",
		})

		// Call assertErrorResponse with the actual status
		assertErrorResponse(t, w, http.StatusOK)
	})

	t.Run("AssertErrorResponseCorrectStatus", func(t *testing.T) {
		s := setupTestServer(t)

		w := testHandlerWithRequest(t, s.handleHealth, HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/health",
		})

		// Test with correct status
		assertErrorResponse(t, w, http.StatusOK)
	})
}

// TestPatternBuilder_AllMethods tests all pattern builder methods
func TestPatternBuilder_AllMethods(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("BuilderWithAllPatterns", func(t *testing.T) {
		builder := NewTestScenarioBuilder()

		customPattern := ErrorTestPattern{
			Name:           "CustomPattern",
			Description:    "A custom test pattern",
			ExpectedStatus: http.StatusOK,
			Execute: func(t *testing.T, setupData interface{}) *HTTPTestRequest {
				return &HTTPTestRequest{
					Method: "GET",
					Path:   "/api/v1/health",
				}
			},
		}

		patterns := builder.
			AddInvalidMethod("/api/v1/health").
			AddMissingContentType("/api/v1/health").
			AddCustom(customPattern).
			Build()

		if len(patterns) != 3 {
			t.Errorf("Expected 3 patterns, got %d", len(patterns))
		}

		// Verify pattern names
		expectedNames := []string{"InvalidMethod", "MissingContentType", "CustomPattern"}
		for i, pattern := range patterns {
			if pattern.Name != expectedNames[i] {
				t.Errorf("Expected pattern %d to be %s, got %s", i, expectedNames[i], pattern.Name)
			}
		}
	})

	t.Run("EmptyBuilder", func(t *testing.T) {
		builder := NewTestScenarioBuilder()
		patterns := builder.Build()

		if len(patterns) != 0 {
			t.Errorf("Expected 0 patterns from empty builder, got %d", len(patterns))
		}
	})
}

// TestTemplateFunction_Coverage tests template function
func TestTemplateFunction_Coverage(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	s := setupTestServer(t)

	// Call template function to increase coverage
	TemplateComprehensiveHandlerTest(t, "handleHealth", s.handleHealth)
}

// TestHTTPRequestCreation_AllCombinations tests all combinations of request creation
func TestHTTPRequestCreation_AllCombinations(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("RequestWithAllOptions", func(t *testing.T) {
		w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/test",
			Body:   map[string]string{"key": "value"},
			QueryParams: map[string]string{
				"param1": "value1",
				"param2": "value2",
			},
			Headers: map[string]string{
				"X-Custom-Header":  "custom-value",
				"X-Another-Header": "another-value",
			},
		})

		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		if w == nil {
			t.Error("Expected non-nil response recorder")
		}

		if httpReq == nil {
			t.Error("Expected non-nil HTTP request")
		}

		// Verify query params
		if httpReq.URL.Query().Get("param1") != "value1" {
			t.Error("Query param param1 not set")
		}

		// Verify headers
		if httpReq.Header.Get("X-Custom-Header") != "custom-value" {
			t.Error("Custom header not set")
		}

		// Verify content type is automatically set
		if httpReq.Header.Get("Content-Type") != "application/json" {
			t.Error("Content-Type not automatically set")
		}
	})

	t.Run("RequestWithExplicitContentType", func(t *testing.T) {
		w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/test",
			Body:   "plain text body",
			Headers: map[string]string{
				"Content-Type": "text/plain",
			},
		})

		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		if w == nil {
			t.Error("Expected non-nil response recorder")
		}

		if httpReq == nil {
			t.Error("Expected non-nil HTTP request")
		}

		// Verify explicit content type is preserved
		if httpReq.Header.Get("Content-Type") != "text/plain" {
			t.Error("Explicit Content-Type was overwritten")
		}
	})
}

// TestAssertJSONResponse_FieldValidation tests field validation in assertJSONResponse
func TestAssertJSONResponse_FieldValidation(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	s := setupTestServer(t)

	t.Run("ValidateFieldExists", func(t *testing.T) {
		w := testHandlerWithRequest(t, s.handleHealth, HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/health",
		})

		response := assertJSONResponse(t, w, http.StatusOK, map[string]interface{}{
			"status": "healthy",
		})

		if response == nil {
			t.Fatal("Response is nil")
		}
	})

	t.Run("ValidateFieldWithNilExpectedValue", func(t *testing.T) {
		w := testHandlerWithRequest(t, s.handleHealth, HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/health",
		})

		// When expectedValue is nil, just check existence
		response := assertJSONResponse(t, w, http.StatusOK, map[string]interface{}{
			"status":    nil,
			"service":   nil,
			"timestamp": nil,
		})

		if response == nil {
			t.Fatal("Response is nil")
		}
	})

	t.Run("ValidateExistingFields", func(t *testing.T) {
		w := testHandlerWithRequest(t, s.handleHealth, HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/health",
		})

		// Test with existing fields
		response := assertJSONResponse(t, w, http.StatusOK, map[string]interface{}{
			"status": "healthy",
		})

		if response == nil {
			t.Error("Response should not be nil")
		}
	})
}
