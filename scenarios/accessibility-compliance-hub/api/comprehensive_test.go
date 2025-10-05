package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// TestTestHelpers tests the test helper functions
func TestTestHelpers(t *testing.T) {
	t.Run("SetupTestDirectory", func(t *testing.T) {
		env := setupTestDirectory(t)
		if env == nil {
			t.Fatal("Expected environment to be created")
		}
		defer env.Cleanup()

		if env.TempDir == "" {
			t.Error("Expected temp directory to be set")
		}
		if env.OriginalWD == "" {
			t.Error("Expected original working directory to be set")
		}
	})

	t.Run("MakeHTTPRequest_WithBody", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/test",
			Body:   map[string]string{"key": "value"},
		}

		w, err := makeHTTPRequest(req)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if w == nil {
			t.Error("Expected response recorder to be created")
		}
	})

	t.Run("MakeHTTPRequest_WithStringBody", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/test",
			Body:   "test string",
		}

		w, err := makeHTTPRequest(req)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if w == nil {
			t.Error("Expected response recorder to be created")
		}
	})

	t.Run("MakeHTTPRequest_WithByteBody", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/test",
			Body:   []byte("test bytes"),
		}

		w, err := makeHTTPRequest(req)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if w == nil {
			t.Error("Expected response recorder to be created")
		}
	})

	t.Run("MakeHTTPRequest_WithHeaders", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/test",
			Headers: map[string]string{
				"X-Custom-Header": "test-value",
			},
		}

		w, err := makeHTTPRequest(req)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if w == nil {
			t.Error("Expected response recorder to be created")
		}
	})

	t.Run("MakeHTTPRequest_WithQueryParams", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/test",
			QueryParams: map[string]string{
				"param1": "value1",
				"param2": "value2",
			},
		}

		w, err := makeHTTPRequest(req)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if w == nil {
			t.Error("Expected response recorder to be created")
		}
	})

	t.Run("AssertJSONResponse_Success", func(t *testing.T) {
		w := httptest.NewRecorder()
		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "ok",
			"value":  123,
		})

		response := assertJSONResponse(t, w, http.StatusOK, map[string]interface{}{
			"status": "ok",
		})

		if response == nil {
			t.Error("Expected response to be parsed")
		}
	})

	t.Run("AssertJSONResponse_WrongStatus", func(t *testing.T) {
		w := httptest.NewRecorder()
		w.WriteHeader(http.StatusBadRequest)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "test"})

		// This will log an error but shouldn't crash
		assertJSONResponse(&testing.T{}, w, http.StatusOK, nil)
	})

	t.Run("AssertJSONArray_Success", func(t *testing.T) {
		w := httptest.NewRecorder()
		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]interface{}{"item1", "item2", "item3"})

		array := assertJSONArray(t, w, http.StatusOK, 2)

		if array == nil {
			t.Error("Expected array to be parsed")
		}
		if len(array) < 2 {
			t.Errorf("Expected at least 2 items, got %d", len(array))
		}
	})

	t.Run("AssertErrorResponse", func(t *testing.T) {
		w := httptest.NewRecorder()
		w.WriteHeader(http.StatusBadRequest)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": "test error message",
		})

		assertErrorResponse(t, w, http.StatusBadRequest, "test error")
	})

	t.Run("TestDataGenerator", func(t *testing.T) {
		scan := TestData.Scan("id-1", "https://example.com", "completed")
		if scan.ID != "id-1" {
			t.Errorf("Expected ID 'id-1', got '%s'", scan.ID)
		}

		violation := TestData.Violation("v-1", "contrast", "low contrast", "high", "button")
		if violation.Type != "contrast" {
			t.Errorf("Expected type 'contrast', got '%s'", violation.Type)
		}

		report := TestData.Report("r-1", "s-1", "Test Report", 85.5)
		if report.Score != 85.5 {
			t.Errorf("Expected score 85.5, got %f", report.Score)
		}
	})
}

// TestTestPatterns tests the test pattern framework
func TestTestPatterns(t *testing.T) {
	t.Run("NewTestScenarioBuilder", func(t *testing.T) {
		builder := NewTestScenarioBuilder()
		if builder == nil {
			t.Fatal("Expected builder to be created")
		}

		patterns := builder.Build()
		if patterns == nil {
			t.Error("Expected empty patterns array")
		}
		if len(patterns) != 0 {
			t.Errorf("Expected 0 patterns, got %d", len(patterns))
		}
	})

	t.Run("AddInvalidMethod", func(t *testing.T) {
		builder := NewTestScenarioBuilder().
			AddInvalidMethod("/test")

		patterns := builder.Build()
		if len(patterns) != 1 {
			t.Errorf("Expected 1 pattern, got %d", len(patterns))
		}

		if patterns[0].ExpectedStatus != http.StatusMethodNotAllowed {
			t.Errorf("Expected status %d, got %d",
				http.StatusMethodNotAllowed, patterns[0].ExpectedStatus)
		}
	})

	t.Run("AddMalformedJSON", func(t *testing.T) {
		builder := NewTestScenarioBuilder().
			AddMalformedJSON("/test")

		patterns := builder.Build()
		if len(patterns) != 1 {
			t.Errorf("Expected 1 pattern, got %d", len(patterns))
		}

		if patterns[0].ExpectedStatus != http.StatusBadRequest {
			t.Errorf("Expected status %d, got %d",
				http.StatusBadRequest, patterns[0].ExpectedStatus)
		}
	})

	t.Run("AddMissingContentType", func(t *testing.T) {
		builder := NewTestScenarioBuilder().
			AddMissingContentType("/test")

		patterns := builder.Build()
		if len(patterns) != 1 {
			t.Errorf("Expected 1 pattern, got %d", len(patterns))
		}
	})

	t.Run("ChainMultiplePatterns", func(t *testing.T) {
		builder := NewTestScenarioBuilder().
			AddInvalidMethod("/test1").
			AddMalformedJSON("/test2").
			AddMissingContentType("/test3")

		patterns := builder.Build()
		if len(patterns) != 3 {
			t.Errorf("Expected 3 patterns, got %d", len(patterns))
		}
	})

	t.Run("AddCustomPattern", func(t *testing.T) {
		customPattern := ErrorTestPattern{
			Name:           "CustomTest",
			Description:    "Custom test pattern",
			ExpectedStatus: http.StatusTeapot,
		}

		builder := NewTestScenarioBuilder().
			AddCustom(customPattern)

		patterns := builder.Build()
		if len(patterns) != 1 {
			t.Errorf("Expected 1 pattern, got %d", len(patterns))
		}

		if patterns[0].ExpectedStatus != http.StatusTeapot {
			t.Errorf("Expected status %d, got %d",
				http.StatusTeapot, patterns[0].ExpectedStatus)
		}
	})
}

// TestHandlerTestSuite tests the handler test suite framework
func TestHandlerTestSuite(t *testing.T) {
	t.Run("RunErrorTests_WithPatterns", func(t *testing.T) {
		suite := &HandlerTestSuite{
			HandlerName: "testHandler",
			Handler:     healthHandler,
			BaseURL:     "/health",
		}

		// Health handler doesn't validate methods, so use a pattern that will pass
		patterns := []ErrorTestPattern{
			{
				Name:           "ValidRequest",
				Description:    "Test with valid request",
				ExpectedStatus: http.StatusOK,
				Execute: func(t *testing.T, setupData interface{}) *HTTPTestRequest {
					return &HTTPTestRequest{
						Method: "GET",
						Path:   "/health",
					}
				},
			},
		}

		// This will run the error tests
		suite.RunErrorTests(t, patterns)
	})

	t.Run("RunErrorTests_WithExecuteFunction", func(t *testing.T) {
		suite := &HandlerTestSuite{
			HandlerName: "testHandler",
			Handler:     healthHandler,
			BaseURL:     "/health",
		}

		patterns := []ErrorTestPattern{
			{
				Name:           "TestPattern",
				Description:    "Test pattern",
				ExpectedStatus: http.StatusOK,
				Execute: func(t *testing.T, setupData interface{}) *HTTPTestRequest {
					return &HTTPTestRequest{
						Method: "GET",
						Path:   "/health",
					}
				},
			},
		}

		suite.RunErrorTests(t, patterns)
	})

	t.Run("RunErrorTests_WithSetupAndCleanup", func(t *testing.T) {
		suite := &HandlerTestSuite{
			HandlerName: "testHandler",
			Handler:     healthHandler,
			BaseURL:     "/health",
		}

		setupCalled := false
		cleanupCalled := false

		patterns := []ErrorTestPattern{
			{
				Name:           "TestWithSetup",
				Description:    "Test with setup and cleanup",
				ExpectedStatus: http.StatusOK,
				Setup: func(t *testing.T) interface{} {
					setupCalled = true
					return "test data"
				},
				Execute: func(t *testing.T, setupData interface{}) *HTTPTestRequest {
					return &HTTPTestRequest{
						Method: "GET",
						Path:   "/health",
					}
				},
				Cleanup: func(setupData interface{}) {
					cleanupCalled = true
				},
			},
		}

		suite.RunErrorTests(t, patterns)

		if !setupCalled {
			t.Error("Expected setup to be called")
		}
		if !cleanupCalled {
			t.Error("Expected cleanup to be called")
		}
	})

	t.Run("RunErrorTests_WithValidation", func(t *testing.T) {
		suite := &HandlerTestSuite{
			HandlerName: "testHandler",
			Handler:     healthHandler,
			BaseURL:     "/health",
		}

		validateCalled := false

		patterns := []ErrorTestPattern{
			{
				Name:           "TestWithValidation",
				Description:    "Test with validation",
				ExpectedStatus: http.StatusOK,
				Execute: func(t *testing.T, setupData interface{}) *HTTPTestRequest {
					return &HTTPTestRequest{
						Method: "GET",
						Path:   "/health",
					}
				},
				Validate: func(t *testing.T, req *HTTPTestRequest, setupData interface{}) {
					validateCalled = true
				},
			},
		}

		suite.RunErrorTests(t, patterns)

		if !validateCalled {
			t.Error("Expected validate to be called")
		}
	})
}

// TestPerformanceTestPattern tests the performance test pattern
func TestPerformanceTestPattern(t *testing.T) {
	t.Run("Run_SuccessfulPerformanceTest", func(t *testing.T) {
		pattern := &PerformanceTestPattern{
			Name:        "TestPerformance",
			Description: "Test performance pattern",
			MaxDuration: 100 * time.Millisecond,
			Execute: func(t *testing.T, setupData interface{}) time.Duration {
				// Simulate some work
				return 50 * time.Millisecond
			},
		}

		pattern.Run(t)
	})

	t.Run("Run_WithSetupAndCleanup", func(t *testing.T) {
		setupCalled := false
		cleanupCalled := false

		pattern := &PerformanceTestPattern{
			Name:        "TestWithSetup",
			Description: "Test with setup and cleanup",
			MaxDuration: 100 * time.Millisecond,
			Setup: func(t *testing.T) interface{} {
				setupCalled = true
				return "test data"
			},
			Execute: func(t *testing.T, setupData interface{}) time.Duration {
				return 10 * time.Millisecond
			},
			Cleanup: func(setupData interface{}) {
				cleanupCalled = true
			},
		}

		pattern.Run(t)

		if !setupCalled {
			t.Error("Expected setup to be called")
		}
		if !cleanupCalled {
			t.Error("Expected cleanup to be called")
		}
	})

	t.Run("Run_WithValidation", func(t *testing.T) {
		validateCalled := false

		pattern := &PerformanceTestPattern{
			Name:        "TestWithValidation",
			Description: "Test with validation",
			MaxDuration: 100 * time.Millisecond,
			Execute: func(t *testing.T, setupData interface{}) time.Duration {
				return 25 * time.Millisecond
			},
			Validate: func(t *testing.T, duration time.Duration) {
				validateCalled = true
			},
		}

		pattern.Run(t)

		if !validateCalled {
			t.Error("Expected validate to be called")
		}
	})
}
