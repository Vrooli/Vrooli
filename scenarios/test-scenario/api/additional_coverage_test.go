package main

import (
	"net/http/httptest"
	"os"
	"testing"
	"time"
)

// TestSetupTestDirectoryEdgeCases tests all branches of setupTestDirectory
func TestSetupTestDirectoryEdgeCases(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("SuccessfulSetup", func(t *testing.T) {
		env := setupTestDirectory(t)
		if env == nil {
			t.Fatal("Expected environment to be created")
		}
		if env.TempDir == "" {
			t.Error("Expected temp directory to be set")
		}
		if env.OriginalWD == "" {
			t.Error("Expected original working directory to be set")
		}
		if env.Cleanup == nil {
			t.Error("Expected cleanup function to be set")
		}

		// Verify temp dir exists
		if _, err := os.Stat(env.TempDir); os.IsNotExist(err) {
			t.Error("Temp directory should exist")
		}

		env.Cleanup()

		// Verify cleanup removed temp dir
		if _, err := os.Stat(env.TempDir); !os.IsNotExist(err) {
			t.Error("Temp directory should be removed after cleanup")
		}
	})
}

// TestMakeHTTPRequestEdgeCases tests all branches of makeHTTPRequest
func TestMakeHTTPRequestEdgeCases(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("RequestWithNilBody", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/test",
			Body:   nil,
		}
		w, httpReq, err := makeHTTPRequest(req)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}
		if w == nil || httpReq == nil {
			t.Error("Expected valid request and recorder")
		}
	})

	t.Run("RequestWithMapBody", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/test",
			Body: map[string]interface{}{
				"key": "value",
			},
		}
		w, httpReq, err := makeHTTPRequest(req)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}
		if w == nil || httpReq == nil {
			t.Error("Expected valid request and recorder")
		}
		if httpReq.Header.Get("Content-Type") != "application/json" {
			t.Error("Expected Content-Type to be set")
		}
	})

	t.Run("RequestWithStringBody", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/test",
			Body:   "string body",
		}
		w, httpReq, err := makeHTTPRequest(req)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}
		if w == nil || httpReq == nil {
			t.Error("Expected valid request and recorder")
		}
	})

	t.Run("RequestWithByteBody", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/test",
			Body:   []byte("byte body"),
		}
		w, httpReq, err := makeHTTPRequest(req)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}
		if w == nil || httpReq == nil {
			t.Error("Expected valid request and recorder")
		}
	})

	t.Run("RequestWithHeaders", func(t *testing.T) {
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
		if httpReq.Header.Get("X-Custom-Header") != "custom-value" {
			t.Error("Expected X-Custom-Header to be set")
		}
		if httpReq.Header.Get("Authorization") != "Bearer token123" {
			t.Error("Expected Authorization to be set")
		}
		_ = w
	})

	t.Run("RequestWithQueryParams", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/test",
			QueryParams: map[string]string{
				"param1": "value1",
				"param2": "value2",
			},
		}
		w, httpReq, err := makeHTTPRequest(req)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}
		if httpReq.URL.Query().Get("param1") != "value1" {
			t.Error("Expected param1 to be set")
		}
		if httpReq.URL.Query().Get("param2") != "value2" {
			t.Error("Expected param2 to be set")
		}
		_ = w
	})

	t.Run("RequestWithPUTAndBody", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "PUT",
			Path:   "/test",
			Body: map[string]string{
				"update": "value",
			},
		}
		w, httpReq, err := makeHTTPRequest(req)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}
		if httpReq.Method != "PUT" {
			t.Errorf("Expected PUT method, got %s", httpReq.Method)
		}
		if httpReq.Header.Get("Content-Type") != "application/json" {
			t.Error("Expected Content-Type for PUT with body")
		}
		_ = w
	})

	t.Run("RequestWithCustomContentType", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/test",
			Body:   "plain text",
			Headers: map[string]string{
				"Content-Type": "text/plain",
			},
		}
		w, httpReq, err := makeHTTPRequest(req)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}
		if httpReq.Header.Get("Content-Type") != "text/plain" {
			t.Error("Custom Content-Type should be preserved")
		}
		_ = w
	})
}

// TestAssertJSONResponseEdgeCases tests all branches of assertJSONResponse
func TestAssertJSONResponseEdgeCases(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("ResponseWithNilExpectedFields", func(t *testing.T) {
		w := httptest.NewRecorder()
		w.WriteHeader(200)
		w.Write([]byte(`{"status": "ok", "data": "test"}`))

		response := assertJSONResponse(t, w, 200, nil)
		if response == nil {
			t.Error("Expected response to be returned")
		}
		if response["status"] != "ok" {
			t.Error("Expected response to contain status field")
		}
	})

	t.Run("ResponseWithExpectedFieldsNilValue", func(t *testing.T) {
		w := httptest.NewRecorder()
		w.WriteHeader(200)
		w.Write([]byte(`{"status": "ok", "optional": null}`))

		response := assertJSONResponse(t, w, 200, map[string]interface{}{
			"status": nil, // Only check existence
		})
		if response == nil {
			t.Error("Expected response to be returned")
		}
	})

	t.Run("ResponseWithMatchingFields", func(t *testing.T) {
		w := httptest.NewRecorder()
		w.WriteHeader(201)
		w.Write([]byte(`{"id": "123", "created": true}`))

		response := assertJSONResponse(t, w, 201, map[string]interface{}{
			"id":      nil,
			"created": true,
		})
		if response == nil {
			t.Error("Expected response to be returned")
		}
	})
}

// TestAssertErrorResponseEdgeCases tests all branches of assertErrorResponse
func TestAssertErrorResponseEdgeCases(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("JSONErrorResponse", func(t *testing.T) {
		w := httptest.NewRecorder()
		w.WriteHeader(400)
		w.Write([]byte(`{"error": "validation failed"}`))

		assertErrorResponse(t, w, 400, "validation")
	})

	t.Run("PlainTextErrorResponse", func(t *testing.T) {
		w := httptest.NewRecorder()
		w.WriteHeader(500)
		w.Write([]byte("Internal Server Error: database connection failed"))

		assertErrorResponse(t, w, 500, "database")
	})

	t.Run("ErrorResponseWithEmptyMessage", func(t *testing.T) {
		w := httptest.NewRecorder()
		w.WriteHeader(404)
		w.Write([]byte(`{"error": "not found"}`))

		assertErrorResponse(t, w, 404, "")
	})

	t.Run("MalformedJSONError", func(t *testing.T) {
		w := httptest.NewRecorder()
		w.WriteHeader(500)
		w.Write([]byte(`{"error": "broken`))

		// Should handle malformed JSON gracefully
		assertErrorResponse(t, w, 500, "")
	})
}

// TestAssertTextResponseEdgeCases tests all branches of assertTextResponse
func TestAssertTextResponseEdgeCases(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("TextResponseWithContent", func(t *testing.T) {
		w := httptest.NewRecorder()
		w.WriteHeader(200)
		w.Write([]byte("Success: operation completed"))

		assertTextResponse(t, w, 200, "Success")
	})

	t.Run("TextResponseWithEmptyExpectedContent", func(t *testing.T) {
		w := httptest.NewRecorder()
		w.WriteHeader(204)

		assertTextResponse(t, w, 204, "")
	})

	t.Run("TextResponseWithMultilineContent", func(t *testing.T) {
		w := httptest.NewRecorder()
		w.WriteHeader(200)
		w.Write([]byte("Line 1\nLine 2\nLine 3"))

		assertTextResponse(t, w, 200, "Line 2")
	})
}

// TestRunErrorTestsEdgeCases tests all branches of RunErrorTests
func TestRunErrorTestsEdgeCases(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("PatternWithoutSetup", func(t *testing.T) {
		suite := &HandlerTestSuite{
			HandlerName: "TestHandler",
			Handler:     handleCommentedError,
			BaseURL:     "/test",
		}

		patterns := []ErrorTestPattern{
			{
				Name:           "NoSetup",
				Description:    "Test without setup",
				ExpectedStatus: 200,
				Setup:          nil,
				Execute: func(t *testing.T, setupData interface{}) *HTTPTestRequest {
					return &HTTPTestRequest{Method: "GET", Path: "/test"}
				},
				Validate: nil,
				Cleanup:  nil,
			},
		}

		suite.RunErrorTests(t, patterns)
	})

	t.Run("PatternWithValidate", func(t *testing.T) {
		suite := &HandlerTestSuite{
			HandlerName: "TestHandler",
			Handler:     handleCommentedError,
			BaseURL:     "/test",
		}

		validateCalled := false
		patterns := []ErrorTestPattern{
			{
				Name:           "WithValidate",
				Description:    "Test with validate",
				ExpectedStatus: 200,
				Setup:          nil,
				Execute: func(t *testing.T, setupData interface{}) *HTTPTestRequest {
					return &HTTPTestRequest{Method: "GET", Path: "/test"}
				},
				Validate: func(t *testing.T, w *httptest.ResponseRecorder, setupData interface{}) {
					validateCalled = true
					if w.Code != 200 {
						t.Error("Validate: Expected 200 status")
					}
				},
				Cleanup: nil,
			},
		}

		suite.RunErrorTests(t, patterns)

		if !validateCalled {
			t.Error("Expected validate function to be called")
		}
	})
}

// TestRunPerformanceTestsEdgeCases tests all branches of RunPerformanceTests
func TestRunPerformanceTestsEdgeCases(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("PatternWithoutSetup", func(t *testing.T) {
		patterns := []PerformanceTestPattern{
			{
				Name:        "NoSetup",
				Description: "Performance test without setup",
				MaxDuration: 1000000, // 1ms
				Setup:       nil,
				Execute: func(t *testing.T, setupData interface{}) time.Duration {
					if setupData != nil {
						t.Error("Expected nil setup data")
					}
					return 100 * time.Nanosecond
				},
				Cleanup: nil,
			},
		}

		RunPerformanceTests(t, patterns)
	})

	t.Run("PatternWithCleanup", func(t *testing.T) {
		cleanupCalled := false
		patterns := []PerformanceTestPattern{
			{
				Name:        "WithCleanup",
				Description: "Performance test with cleanup",
				MaxDuration: 1000000,
				Setup: func(t *testing.T) interface{} {
					return "test-data"
				},
				Execute: func(t *testing.T, setupData interface{}) time.Duration {
					if setupData.(string) != "test-data" {
						t.Error("Expected setup data to be passed")
					}
					return 100 * time.Nanosecond
				},
				Cleanup: func(setupData interface{}) {
					cleanupCalled = true
					if setupData.(string) != "test-data" {
						panic("Expected setup data in cleanup")
					}
				},
			},
		}

		RunPerformanceTests(t, patterns)

		if !cleanupCalled {
			t.Error("Expected cleanup to be called")
		}
	})
}

// TestVerboseLogging tests the verbose logging path
func TestVerboseLogging(t *testing.T) {
	// Test with verbose mode enabled
	originalEnv := os.Getenv("TEST_VERBOSE")
	os.Setenv("TEST_VERBOSE", "true")
	defer os.Setenv("TEST_VERBOSE", originalEnv)

	cleanup := setupTestLogger()
	defer cleanup()

	t.Log("Verbose logging is enabled")
}

// TestVerboseLoggingDisabled tests the non-verbose path
func TestVerboseLoggingDisabled(t *testing.T) {
	// Test with verbose mode disabled (default)
	originalEnv := os.Getenv("TEST_VERBOSE")
	os.Setenv("TEST_VERBOSE", "")
	defer os.Setenv("TEST_VERBOSE", originalEnv)

	cleanup := setupTestLogger()
	defer cleanup()

	t.Log("Verbose logging is disabled")
}
