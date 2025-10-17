package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// TestPatternBuilderCoverage improves coverage for the pattern builder
func TestPatternBuilderCoverage(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("AddInvalidJSON_ExecutePattern", func(t *testing.T) {
		builder := NewTestScenarioBuilder()
		builder.AddInvalidJSON("/test-endpoint", "POST")
		patterns := builder.Build()

		if len(patterns) != 1 {
			t.Errorf("Expected 1 pattern, got %d", len(patterns))
		}

		pattern := patterns[0]
		if pattern.Name != "InvalidJSON" {
			t.Errorf("Expected pattern name 'InvalidJSON', got '%s'", pattern.Name)
		}

		// Execute the pattern with a test handler
		testHandler := func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`{"error": "invalid JSON"}`))
		}

		// Call the pattern's Execute function to get the request
		req := pattern.Execute(t, nil)
		w, httpReq, err := makeHTTPRequest(*req)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		testHandler(w, httpReq)

		if w.Code != pattern.ExpectedStatus {
			t.Logf("Handler returned status %d (expected %d)", w.Code, pattern.ExpectedStatus)
		}
	})

	t.Run("AddMissingContentType_ExecutePattern", func(t *testing.T) {
		builder := NewTestScenarioBuilder()
		builder.AddMissingContentType("/test-endpoint", "POST")
		patterns := builder.Build()

		if len(patterns) != 1 {
			t.Errorf("Expected 1 pattern, got %d", len(patterns))
		}

		pattern := patterns[0]
		if pattern.Name != "MissingContentType" {
			t.Errorf("Expected pattern name 'MissingContentType', got '%s'", pattern.Name)
		}

		// Execute the pattern
		testHandler := func(w http.ResponseWriter, r *http.Request) {
			if r.Header.Get("Content-Type") == "" {
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte(`{"error": "missing content type"}`))
				return
			}
			w.WriteHeader(http.StatusOK)
		}

		req := pattern.Execute(t, nil)
		w, httpReq, err := makeHTTPRequest(*req)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		testHandler(w, httpReq)

		if w.Code != pattern.ExpectedStatus {
			t.Logf("Handler returned status %d (expected %d)", w.Code, pattern.ExpectedStatus)
		}
	})

	t.Run("AddCustom_ExecutePattern", func(t *testing.T) {
		builder := NewTestScenarioBuilder()
		customPattern := ErrorTestPattern{
			Name:           "CustomTest",
			Description:    "Custom test pattern",
			ExpectedStatus: 200,
			Execute: func(t *testing.T, setupData interface{}) *HTTPTestRequest {
				return &HTTPTestRequest{
					Method:  "POST",
					Path:    "/custom",
					Body:    `{"test": "data"}`,
					Headers: map[string]string{"X-Custom": "value"},
				}
			},
		}
		builder.AddCustom(customPattern)
		patterns := builder.Build()

		if len(patterns) != 1 {
			t.Errorf("Expected 1 pattern, got %d", len(patterns))
		}

		pattern := patterns[0]
		if pattern.Name != "CustomTest" {
			t.Errorf("Expected pattern name 'CustomTest', got '%s'", pattern.Name)
		}

		// Execute the pattern
		testHandler := func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"status": "ok"}`))
		}

		req := pattern.Execute(t, nil)
		w, httpReq, err := makeHTTPRequest(*req)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		testHandler(w, httpReq)

		if w.Code != pattern.ExpectedStatus {
			t.Errorf("Expected status %d, got %d", pattern.ExpectedStatus, w.Code)
		}
	})

	t.Run("ChainedPatternBuild", func(t *testing.T) {
		builder := NewTestScenarioBuilder()
		patterns := builder.
			AddInvalidJSON("/endpoint1", "POST").
			AddMissingContentType("/endpoint2", "PUT").
			AddCustom(ErrorTestPattern{
				Name:           "ChainedCustom",
				Description:    "Chained custom pattern",
				ExpectedStatus: 404,
				Execute: func(t *testing.T, setupData interface{}) *HTTPTestRequest {
					return &HTTPTestRequest{
						Method: "GET",
						Path:   "/endpoint3",
					}
				},
			}).
			Build()

		if len(patterns) != 3 {
			t.Errorf("Expected 3 patterns, got %d", len(patterns))
		}
	})
}

// TestHandlerTestSuiteCoverage improves coverage for HandlerTestSuite
func TestHandlerTestSuiteCoverage(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	t.Run("ExecuteMultiplePatterns", func(t *testing.T) {
		testHandler := func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`{"error": "bad request"}`))
		}

		suite := HandlerTestSuite{
			HandlerName: "TestHandler",
			Handler:     testHandler,
			BaseURL:     "/test",
		}

		patterns := []ErrorTestPattern{
			{
				Name:           "Pattern1",
				Description:    "First pattern",
				ExpectedStatus: 400,
				Execute: func(t *testing.T, setupData interface{}) *HTTPTestRequest {
					return &HTTPTestRequest{
						Method: "GET",
						Path:   "/test1",
					}
				},
			},
			{
				Name:           "Pattern2",
				Description:    "Second pattern",
				ExpectedStatus: 400,
				Execute: func(t *testing.T, setupData interface{}) *HTTPTestRequest {
					return &HTTPTestRequest{
						Method: "POST",
						Path:   "/test2",
						Body:   `{"data": "test"}`,
					}
				},
			},
		}

		suite.RunErrorTests(t, patterns)
	})

	t.Run("ExecuteWithSetupAndCleanup", func(t *testing.T) {
		testHandler := func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"status": "ok"}`))
		}

		suite := HandlerTestSuite{
			HandlerName: "TestHandler",
			Handler:     testHandler,
			BaseURL:     "/test",
		}

		setupCalled := false
		cleanupCalled := false

		patterns := []ErrorTestPattern{
			{
				Name:           "WithSetup",
				Description:    "Pattern with setup and cleanup",
				ExpectedStatus: 200,
				Setup: func(t *testing.T) interface{} {
					setupCalled = true
					return "test data"
				},
				Execute: func(t *testing.T, setupData interface{}) *HTTPTestRequest {
					if setupData != "test data" {
						t.Error("Setup data not passed correctly")
					}
					return &HTTPTestRequest{
						Method: "GET",
						Path:   "/test",
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

	t.Run("ExecuteWithValidation", func(t *testing.T) {
		testHandler := func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"result": "success"}`))
		}

		suite := HandlerTestSuite{
			HandlerName: "TestHandler",
			Handler:     testHandler,
			BaseURL:     "/test",
		}

		validationCalled := false

		patterns := []ErrorTestPattern{
			{
				Name:           "WithValidation",
				Description:    "Pattern with custom validation",
				ExpectedStatus: 200,
				Execute: func(t *testing.T, setupData interface{}) *HTTPTestRequest {
					return &HTTPTestRequest{
						Method: "GET",
						Path:   "/test",
					}
				},
				Validate: func(t *testing.T, w *httptest.ResponseRecorder, setupData interface{}) {
					validationCalled = true
					response := assertJSONResponse(t, w, 200, map[string]interface{}{
						"result": "success",
					})
					if response == nil {
						t.Error("Expected response to be parsed")
					}
				},
			},
		}

		suite.RunErrorTests(t, patterns)

		if !validationCalled {
			t.Error("Validation was not called")
		}
	})
}

// TestRunPerformanceTestsCoverage improves coverage for RunPerformanceTests
func TestRunPerformanceTestsCoverage(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	t.Run("ExecuteFastOperation", func(t *testing.T) {
		patterns := []PerformanceTestPattern{
			{
				Name:        "FastOperation",
				Description: "Test fast operation",
				MaxDuration: 100 * time.Millisecond,
				Setup:       nil,
				Execute: func(t *testing.T, setupData interface{}) time.Duration {
					start := time.Now()
					// Fast operation
					_ = 1 + 1
					return time.Since(start)
				},
			},
		}

		RunPerformanceTests(t, patterns)
	})

	t.Run("ExecuteModerateOperation", func(t *testing.T) {
		patterns := []PerformanceTestPattern{
			{
				Name:        "ModerateOperation",
				Description: "Test operation within reasonable threshold",
				MaxDuration: 10 * time.Millisecond, // Reasonable threshold
				Setup:       nil,
				Execute: func(t *testing.T, setupData interface{}) time.Duration {
					start := time.Now()
					// Simulate some work
					sum := 0
					for i := 0; i < 1000; i++ {
						sum += i
					}
					_ = sum
					return time.Since(start)
				},
			},
		}

		RunPerformanceTests(t, patterns)
	})

	t.Run("ExecuteWithSetup", func(t *testing.T) {
		setupData := "test data"
		patterns := []PerformanceTestPattern{
			{
				Name:        "WithSetup",
				Description: "Test with setup data",
				MaxDuration: 100 * time.Millisecond,
				Setup: func(t *testing.T) interface{} {
					return setupData
				},
				Execute: func(t *testing.T, data interface{}) time.Duration {
					start := time.Now()
					if data != setupData {
						t.Error("Setup data not passed correctly")
					}
					return time.Since(start)
				},
			},
		}

		RunPerformanceTests(t, patterns)
	})

	t.Run("ExecuteWithCleanup", func(t *testing.T) {
		cleanupCalled := false
		patterns := []PerformanceTestPattern{
			{
				Name:        "WithCleanup",
				Description: "Test with cleanup",
				MaxDuration: 100 * time.Millisecond,
				Setup: func(t *testing.T) interface{} {
					return "data"
				},
				Execute: func(t *testing.T, data interface{}) time.Duration {
					start := time.Now()
					return time.Since(start)
				},
				Cleanup: func(data interface{}) {
					cleanupCalled = true
				},
			},
		}

		RunPerformanceTests(t, patterns)

		if !cleanupCalled {
			t.Error("Cleanup was not called")
		}
	})

	t.Run("ExecuteMultiplePatterns", func(t *testing.T) {
		patterns := []PerformanceTestPattern{
			{
				Name:        "Test1",
				Description: "First test",
				MaxDuration: 50 * time.Millisecond,
				Execute: func(t *testing.T, data interface{}) time.Duration {
					start := time.Now()
					return time.Since(start)
				},
			},
			{
				Name:        "Test2",
				Description: "Second test",
				MaxDuration: 50 * time.Millisecond,
				Execute: func(t *testing.T, data interface{}) time.Duration {
					start := time.Now()
					return time.Since(start)
				},
			},
		}

		RunPerformanceTests(t, patterns)
	})
}
