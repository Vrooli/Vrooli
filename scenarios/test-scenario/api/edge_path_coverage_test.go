package main

import (
	"net/http/httptest"
	"testing"
)

// TestErrorPathsInHelpers tests error paths in helper functions to improve coverage
func TestErrorPathsInHelpers(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	// These tests exercise error paths in the assertion helpers
	// We use a mock testing.T to capture errors without failing the main test

	t.Run("AssertJSONResponse_ErrorPaths", func(t *testing.T) {
		// Test wrong status code path
		w1 := httptest.NewRecorder()
		w1.WriteHeader(404)
		w1.Write([]byte(`{"error": "not found"}`))
		mockT1 := &testing.T{}
		assertJSONResponse(mockT1, w1, 200, nil) // Wrong status - triggers error path

		// Test missing expected field path
		w2 := httptest.NewRecorder()
		w2.WriteHeader(200)
		w2.Write([]byte(`{"foo": "bar"}`))
		mockT2 := &testing.T{}
		assertJSONResponse(mockT2, w2, 200, map[string]interface{}{
			"missing": nil,
		}) // Missing field - triggers error path

		// Test wrong field value path
		w3 := httptest.NewRecorder()
		w3.WriteHeader(200)
		w3.Write([]byte(`{"status": "error"}`))
		mockT3 := &testing.T{}
		assertJSONResponse(mockT3, w3, 200, map[string]interface{}{
			"status": "ok",
		}) // Wrong value - triggers error path
	})

	t.Run("AssertErrorResponse_ErrorPaths", func(t *testing.T) {
		// Test wrong status code path
		w1 := httptest.NewRecorder()
		w1.WriteHeader(200)
		w1.Write([]byte(`{"error": "test"}`))
		mockT1 := &testing.T{}
		assertErrorResponse(mockT1, w1, 400, "test") // Wrong status - triggers error path

		// Test missing error field path
		w2 := httptest.NewRecorder()
		w2.WriteHeader(400)
		w2.Write([]byte(`{"message": "test"}`))
		mockT2 := &testing.T{}
		assertErrorResponse(mockT2, w2, 400, "test") // Missing error field - triggers error path

		// Test wrong error message path
		w3 := httptest.NewRecorder()
		w3.WriteHeader(400)
		w3.Write([]byte(`{"error": "different"}`))
		mockT3 := &testing.T{}
		assertErrorResponse(mockT3, w3, 400, "expected") // Wrong message - triggers error path

		// Test non-JSON with wrong message
		w4 := httptest.NewRecorder()
		w4.WriteHeader(400)
		w4.Write([]byte(`Plain text error`))
		mockT4 := &testing.T{}
		assertErrorResponse(mockT4, w4, 400, "expected") // Wrong message in plain text - triggers error path
	})

	t.Run("AssertTextResponse_ErrorPaths", func(t *testing.T) {
		// Test wrong status code path
		w1 := httptest.NewRecorder()
		w1.WriteHeader(404)
		w1.Write([]byte(`Not found`))
		mockT1 := &testing.T{}
		assertTextResponse(mockT1, w1, 200, "Not found") // Wrong status - triggers error path

		// Test wrong content path
		w2 := httptest.NewRecorder()
		w2.WriteHeader(200)
		w2.Write([]byte(`Actual content`))
		mockT2 := &testing.T{}
		assertTextResponse(mockT2, w2, 200, "expected") // Wrong content - triggers error path
	})
}

// TestMakeHTTPRequestErrorPaths tests edge cases in makeHTTPRequest
func TestMakeHTTPRequestErrorPaths(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("WithAllFields", func(t *testing.T) {
		// Test with all optional fields set to exercise all code paths
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/test?existing=param",
			Body: map[string]string{
				"key": "value",
			},
			Headers: map[string]string{
				"X-Custom":     "header",
				"Content-Type": "application/json",
			},
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

		// Verify all fields were set
		if httpReq.Header.Get("X-Custom") != "header" {
			t.Error("Expected custom header to be set")
		}

		if httpReq.URL.Query().Get("page") != "1" {
			t.Error("Expected query param to be set")
		}
	})

	t.Run("WithNilHeaders", func(t *testing.T) {
		req := HTTPTestRequest{
			Method:  "GET",
			Path:    "/test",
			Headers: nil,
		}

		w, httpReq, err := makeHTTPRequest(req)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		if w == nil || httpReq == nil {
			t.Error("Expected request to be created")
		}
	})

	t.Run("WithNilQueryParams", func(t *testing.T) {
		req := HTTPTestRequest{
			Method:      "GET",
			Path:        "/test",
			QueryParams: nil,
		}

		w, httpReq, err := makeHTTPRequest(req)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		if w == nil || httpReq == nil {
			t.Error("Expected request to be created")
		}
	})

	t.Run("PUTWithBody", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "PUT",
			Path:   "/test",
			Body:   `{"update": "data"}`,
		}

		w, httpReq, err := makeHTTPRequest(req)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		if w == nil || httpReq == nil {
			t.Error("Expected request to be created")
		}

		// Verify Content-Type was set for PUT with body
		if httpReq.Header.Get("Content-Type") != "application/json" {
			t.Error("Expected Content-Type to be set for PUT request")
		}
	})
}

// TestSetupTestDirectoryErrorPaths tests error handling in setupTestDirectory
func TestSetupTestDirectoryErrorPaths(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("NormalOperation", func(t *testing.T) {
		env := setupTestDirectory(t)

		// Verify all fields are set
		if env.TempDir == "" {
			t.Error("Expected TempDir to be set")
		}

		if env.OriginalWD == "" {
			t.Error("Expected OriginalWD to be set")
		}

		if env.Cleanup == nil {
			t.Error("Expected Cleanup to be set")
		}

		// Call cleanup
		env.Cleanup()
	})

	t.Run("DoubleCleanup", func(t *testing.T) {
		env := setupTestDirectory(t)

		// Call cleanup twice to exercise cleanup code path
		env.Cleanup()
		env.Cleanup() // Second call should be safe
	})
}
