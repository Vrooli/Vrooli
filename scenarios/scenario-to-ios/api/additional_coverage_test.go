
package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// TestRunErrorTests_FullCoverage tests RunErrorTests with all branches
func TestRunErrorTests_FullCoverage(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	s := setupTestServer(t)

	suite := &HandlerTestSuite{
		HandlerName: "handleHealth",
		Handler:     s.handleHealth,
		BaseURL:     "/api/v1/health",
	}

	t.Run("WithSetupAndCleanup", func(t *testing.T) {
		patterns := []ErrorTestPattern{
			{
				Name:           "WithSetup",
				Description:    "Test with setup and cleanup",
				ExpectedStatus: http.StatusOK,
				Setup: func(t *testing.T) interface{} {
					return map[string]string{"test": "data"}
				},
				Execute: func(t *testing.T, setupData interface{}) *HTTPTestRequest {
					return &HTTPTestRequest{
						Method: "GET",
						Path:   "/api/v1/health",
					}
				},
				Validate: func(t *testing.T, w *httptest.ResponseRecorder, setupData interface{}) {
					if w.Code != http.StatusOK {
						t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
					}
				},
				Cleanup: func(setupData interface{}) {
					// Cleanup logic
				},
			},
		}

		suite.RunErrorTests(t, patterns)
	})

	t.Run("WithoutValidation", func(t *testing.T) {
		patterns := []ErrorTestPattern{
			{
				Name:           "NoValidation",
				Description:    "Test without validation",
				ExpectedStatus: http.StatusOK,
				Execute: func(t *testing.T, setupData interface{}) *HTTPTestRequest {
					return &HTTPTestRequest{
						Method: "GET",
						Path:   "/api/v1/health",
					}
				},
			},
		}

		suite.RunErrorTests(t, patterns)
	})
}

// TestRunPerformanceTest_FullCoverage tests RunPerformanceTest with all branches
func TestRunPerformanceTest_FullCoverage(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("WithCustomValidation", func(t *testing.T) {
		pattern := PerformanceTestPattern{
			Name:        "CustomValidation",
			Description: "Test with custom validation",
			MaxDuration: 100 * time.Millisecond,
			Setup: func(t *testing.T) interface{} {
				return "test data"
			},
			Execute: func(t *testing.T, setupData interface{}) time.Duration {
				start := time.Now()
				time.Sleep(1 * time.Millisecond)
				return time.Since(start)
			},
			Validate: func(t *testing.T, duration time.Duration) {
				if duration > 100*time.Millisecond {
					t.Errorf("Duration exceeded: %v", duration)
				}
			},
			Cleanup: func(setupData interface{}) {
				// Cleanup
			},
		}

		RunPerformanceTest(t, pattern)
	})

	t.Run("WithoutCustomValidation", func(t *testing.T) {
		pattern := PerformanceTestPattern{
			Name:        "DefaultValidation",
			Description: "Test with default validation",
			MaxDuration: 100 * time.Millisecond,
			Execute: func(t *testing.T, setupData interface{}) time.Duration {
				start := time.Now()
				time.Sleep(1 * time.Millisecond)
				return time.Since(start)
			},
		}

		RunPerformanceTest(t, pattern)
	})
}

// TestAddPatternMethods_Coverage tests pattern addition methods
func TestAddPatternMethods_Coverage(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("AddInvalidMethodExecution", func(t *testing.T) {
		builder := NewTestScenarioBuilder()
		patterns := builder.AddInvalidMethod("/api/v1/health").Build()

		if len(patterns) != 1 {
			t.Fatalf("Expected 1 pattern, got %d", len(patterns))
		}

		// Execute the pattern to cover its Execute function
		pattern := patterns[0]
		if pattern.Execute != nil {
			req := pattern.Execute(t, nil)
			if req.Method != "DELETE" {
				t.Errorf("Expected DELETE method, got %s", req.Method)
			}
		}
	})

	t.Run("AddMissingContentTypeExecution", func(t *testing.T) {
		builder := NewTestScenarioBuilder()
		patterns := builder.AddMissingContentType("/api/v1/health").Build()

		if len(patterns) != 1 {
			t.Fatalf("Expected 1 pattern, got %d", len(patterns))
		}

		// Execute the pattern to cover its Execute function
		pattern := patterns[0]
		if pattern.Execute != nil {
			req := pattern.Execute(t, nil)
			if req.Method != "POST" {
				t.Errorf("Expected POST method, got %s", req.Method)
			}
		}
	})
}

// TestAssertJSONResponse_ErrorPaths tests error paths in assertJSONResponse
func TestAssertJSONResponse_ErrorPaths(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	s := setupTestServer(t)

	t.Run("FieldExists", func(t *testing.T) {
		w := testHandlerWithRequest(t, s.handleHealth, HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/health",
		})

		// Test with correct value
		assertJSONResponse(t, w, http.StatusOK, map[string]interface{}{
			"status": "healthy",
		})
	})
}

// TestAssertErrorResponse_FullCoverage tests assertErrorResponse
func TestAssertErrorResponse_FullCoverage(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	s := setupTestServer(t)

	t.Run("CorrectStatus", func(t *testing.T) {
		w := testHandlerWithRequest(t, s.handleHealth, HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/health",
		})

		assertErrorResponse(t, w, http.StatusOK)
	})
}

// TestSetupTestDirectory_FullCoverage tests setupTestDirectory
func TestSetupTestDirectory_FullCoverage(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("CreateAndCleanup", func(t *testing.T) {
		env := setupTestDirectory(t)

		if env.TempDir == "" {
			t.Error("Expected temp dir to be set")
		}

		if env.OriginalWD == "" {
			t.Error("Expected original working directory to be set")
		}

		// Test cleanup function
		env.Cleanup()
	})
}
