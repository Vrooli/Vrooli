package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
)

// ErrorTestPattern defines a systematic approach to testing error conditions
type ErrorTestPattern struct {
	Name           string
	Description    string
	ExpectedStatus int
	Setup          func(t *testing.T) interface{}
	Execute        func(t *testing.T, setupData interface{}) *HTTPTestRequest
	Validate       func(t *testing.T, w *httptest.ResponseRecorder, setupData interface{})
	Cleanup        func(setupData interface{})
}

// HandlerTestSuite provides a comprehensive test framework for HTTP handlers
type HandlerTestSuite struct {
	HandlerName string
	BaseURL     string
}

// PerformanceTestPattern defines performance testing scenarios
type PerformanceTestPattern struct {
	Name        string
	Description string
	MaxDuration time.Duration
	Setup       func(t *testing.T) interface{}
	Execute     func(t *testing.T, setupData interface{}) time.Duration
	Cleanup     func(setupData interface{})
}

// TestScenarioBuilder provides a fluent interface for building test scenarios
type TestScenarioBuilder struct {
	patterns []ErrorTestPattern
}

// NewTestScenarioBuilder creates a new test scenario builder
func NewTestScenarioBuilder() *TestScenarioBuilder {
	return &TestScenarioBuilder{
		patterns: []ErrorTestPattern{},
	}
}

// AddInvalidUUID adds invalid UUID test pattern
func (b *TestScenarioBuilder) AddInvalidUUID(path string, method string) *TestScenarioBuilder {
	b.patterns = append(b.patterns, ErrorTestPattern{
		Name:           "InvalidUUID",
		Description:    "Test handler with invalid UUID format",
		ExpectedStatus: http.StatusBadRequest,
		Execute: func(t *testing.T, setupData interface{}) *HTTPTestRequest {
			return &HTTPTestRequest{
				Method: method,
				Path:   path,
			}
		},
	})
	return b
}

// AddNonExistentPersona adds non-existent persona test pattern
func (b *TestScenarioBuilder) AddNonExistentPersona(path string, method string) *TestScenarioBuilder {
	b.patterns = append(b.patterns, ErrorTestPattern{
		Name:           "NonExistentPersona",
		Description:    "Test handler with non-existent persona ID",
		ExpectedStatus: http.StatusNotFound,
		Execute: func(t *testing.T, setupData interface{}) *HTTPTestRequest {
			nonExistentID := uuid.New().String()
			return &HTTPTestRequest{
				Method: method,
				Path:   fmt.Sprintf(path, nonExistentID),
			}
		},
	})
	return b
}

// AddMissingRequiredField adds missing required field test pattern
func (b *TestScenarioBuilder) AddMissingRequiredField(path string, body interface{}) *TestScenarioBuilder {
	b.patterns = append(b.patterns, ErrorTestPattern{
		Name:           "MissingRequiredField",
		Description:    "Test handler with missing required fields",
		ExpectedStatus: http.StatusBadRequest,
		Execute: func(t *testing.T, setupData interface{}) *HTTPTestRequest {
			return &HTTPTestRequest{
				Method: "POST",
				Path:   path,
				Body:   body,
			}
		},
	})
	return b
}

// AddInvalidJSON adds invalid JSON test pattern
func (b *TestScenarioBuilder) AddInvalidJSON(path string) *TestScenarioBuilder {
	b.patterns = append(b.patterns, ErrorTestPattern{
		Name:           "InvalidJSON",
		Description:    "Test handler with malformed JSON input",
		ExpectedStatus: http.StatusBadRequest,
		Execute: func(t *testing.T, setupData interface{}) *HTTPTestRequest {
			return &HTTPTestRequest{
				Method: "POST",
				Path:   path,
				Body:   `{"invalid": "json"`, // Malformed JSON
			}
		},
	})
	return b
}

// AddEmptyBody adds empty body test pattern
func (b *TestScenarioBuilder) AddEmptyBody(path string) *TestScenarioBuilder {
	b.patterns = append(b.patterns, ErrorTestPattern{
		Name:           "EmptyBody",
		Description:    "Test handler with empty request body",
		ExpectedStatus: http.StatusBadRequest,
		Execute: func(t *testing.T, setupData interface{}) *HTTPTestRequest {
			return &HTTPTestRequest{
				Method: "POST",
				Path:   path,
				Body:   "",
			}
		},
	})
	return b
}

// AddCustom adds a custom test pattern
func (b *TestScenarioBuilder) AddCustom(pattern ErrorTestPattern) *TestScenarioBuilder {
	b.patterns = append(b.patterns, pattern)
	return b
}

// Build returns the configured test patterns
func (b *TestScenarioBuilder) Build() []ErrorTestPattern {
	return b.patterns
}

// RunErrorPatterns executes error test patterns with proper setup/cleanup
func RunErrorPatterns(t *testing.T, patterns []ErrorTestPattern, handler func(*testing.T, *HTTPTestRequest) *httptest.ResponseRecorder) {
	for _, pattern := range patterns {
		t.Run(pattern.Name, func(t *testing.T) {
			// Setup
			var setupData interface{}
			if pattern.Setup != nil {
				setupData = pattern.Setup(t)
			}

			// Cleanup
			if pattern.Cleanup != nil {
				defer pattern.Cleanup(setupData)
			}

			// Execute
			req := pattern.Execute(t, setupData)
			w := handler(t, req)

			// Validate status code
			if w.Code != pattern.ExpectedStatus {
				t.Errorf("Expected status %d, got %d. Response: %s",
					pattern.ExpectedStatus, w.Code, w.Body.String())
			}

			// Additional validation
			if pattern.Validate != nil {
				pattern.Validate(t, w, setupData)
			}
		})
	}
}

// Common test patterns

// personaNotFoundPattern creates a pattern for testing non-existent personas
func personaNotFoundPattern(personaID string) ErrorTestPattern {
	return ErrorTestPattern{
		Name:           "PersonaNotFound",
		Description:    "Test with non-existent persona ID",
		ExpectedStatus: http.StatusNotFound,
		Execute: func(t *testing.T, setupData interface{}) *HTTPTestRequest {
			return &HTTPTestRequest{
				Method: "GET",
				Path:   fmt.Sprintf("/api/persona/%s", personaID),
			}
		},
	}
}

// invalidRequestBodyPattern creates a pattern for testing invalid request bodies
func invalidRequestBodyPattern(path string, invalidBody interface{}) ErrorTestPattern {
	return ErrorTestPattern{
		Name:           "InvalidRequestBody",
		Description:    "Test with invalid request body",
		ExpectedStatus: http.StatusBadRequest,
		Execute: func(t *testing.T, setupData interface{}) *HTTPTestRequest {
			return &HTTPTestRequest{
				Method: "POST",
				Path:   path,
				Body:   invalidBody,
			}
		},
	}
}

// Performance test helpers

// measureExecutionTime measures the execution time of a function
func measureExecutionTime(fn func()) time.Duration {
	start := time.Now()
	fn()
	return time.Since(start)
}

// runPerformanceTest runs a performance test pattern
func runPerformanceTest(t *testing.T, pattern PerformanceTestPattern) {
	t.Run(pattern.Name, func(t *testing.T) {
		// Setup
		var setupData interface{}
		if pattern.Setup != nil {
			setupData = pattern.Setup(t)
		}

		// Cleanup
		if pattern.Cleanup != nil {
			defer pattern.Cleanup(setupData)
		}

		// Execute and measure
		duration := pattern.Execute(t, setupData)

		// Validate performance
		if duration > pattern.MaxDuration {
			t.Errorf("Performance test failed: execution took %v, expected max %v",
				duration, pattern.MaxDuration)
		} else {
			t.Logf("Performance test passed: execution took %v (max: %v)",
				duration, pattern.MaxDuration)
		}
	})
}

// Concurrency test helpers

// runConcurrentRequests executes multiple requests concurrently
func runConcurrentRequests(t *testing.T, count int, fn func(i int) error) []error {
	errors := make([]error, count)
	done := make(chan bool, count)

	for i := 0; i < count; i++ {
		go func(index int) {
			errors[index] = fn(index)
			done <- true
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < count; i++ {
		<-done
	}

	return errors
}

// validateNoConcurrencyErrors checks that no errors occurred during concurrent execution
func validateNoConcurrencyErrors(t *testing.T, errors []error) {
	for i, err := range errors {
		if err != nil {
			t.Errorf("Concurrent request %d failed: %v", i, err)
		}
	}
}
