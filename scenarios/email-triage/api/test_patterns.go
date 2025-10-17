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
	Execute        func(t *testing.T, setupData interface{}) HTTPTestRequest
	Validate       func(t *testing.T, w *httptest.ResponseRecorder, setupData interface{})
	Cleanup        func(setupData interface{})
}

// HandlerTestSuite provides a comprehensive test framework for HTTP handlers
type HandlerTestSuite struct {
	HandlerName    string
	Handler        http.HandlerFunc
	BaseURL        string
	RequiredURLVars []string
}

// RunErrorTests executes a suite of error condition tests
func (suite *HandlerTestSuite) RunErrorTests(t *testing.T, patterns []ErrorTestPattern) {
	for _, pattern := range patterns {
		t.Run(fmt.Sprintf("%s_%s", suite.HandlerName, pattern.Name), func(t *testing.T) {
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
			w, httpReq, err := makeHTTPRequest(req)
			if err != nil {
				t.Fatalf("Failed to create HTTP request: %v", err)
			}

			// Execute handler
			suite.Handler(w, httpReq)

			// Check expected status
			if w.Code != pattern.ExpectedStatus {
				t.Errorf("Expected status %d, got %d. Response: %s",
					pattern.ExpectedStatus, w.Code, w.Body.String())
			}

			// Validate
			if pattern.Validate != nil {
				pattern.Validate(t, w, setupData)
			}
		})
	}
}

// Common error patterns that can be reused across handlers

// invalidUUIDPattern tests handlers with invalid UUID formats
func invalidUUIDPattern(method, urlPath string) ErrorTestPattern {
	return ErrorTestPattern{
		Name:           "InvalidUUID",
		Description:    "Test handler with invalid UUID format",
		ExpectedStatus: http.StatusBadRequest,
		Setup:          nil,
		Execute: func(t *testing.T, setupData interface{}) HTTPTestRequest {
			return HTTPTestRequest{
				Method:  method,
				Path:    urlPath,
				URLVars: map[string]string{"id": "invalid-uuid"},
				Context: map[string]interface{}{"user_id": uuid.New().String()},
			}
		},
		Validate: func(t *testing.T, w *httptest.ResponseRecorder, setupData interface{}) {
			assertErrorResponse(t, w, http.StatusBadRequest, "")
		},
	}
}

// nonExistentResourcePattern tests handlers with non-existent resource IDs
func nonExistentResourcePattern(method, urlPath string) ErrorTestPattern {
	return ErrorTestPattern{
		Name:           "NonExistentResource",
		Description:    "Test handler with non-existent resource ID",
		ExpectedStatus: http.StatusNotFound,
		Setup:          nil,
		Execute: func(t *testing.T, setupData interface{}) HTTPTestRequest {
			nonExistentID := uuid.New()
			return HTTPTestRequest{
				Method:  method,
				Path:    urlPath,
				URLVars: map[string]string{"id": nonExistentID.String()},
				Context: map[string]interface{}{"user_id": uuid.New().String()},
			}
		},
		Validate: func(t *testing.T, w *httptest.ResponseRecorder, setupData interface{}) {
			assertErrorResponse(t, w, http.StatusNotFound, "not found")
		},
	}
}

// invalidJSONPattern tests handlers with malformed JSON
func invalidJSONPattern(method, urlPath string) ErrorTestPattern {
	return ErrorTestPattern{
		Name:           "InvalidJSON",
		Description:    "Test handler with malformed JSON input",
		ExpectedStatus: http.StatusBadRequest,
		Setup:          nil,
		Execute: func(t *testing.T, setupData interface{}) HTTPTestRequest {
			return HTTPTestRequest{
				Method:  method,
				Path:    urlPath,
				Body:    `{"invalid": "json"`, // Malformed JSON
				Context: map[string]interface{}{"user_id": uuid.New().String()},
			}
		},
		Validate: func(t *testing.T, w *httptest.ResponseRecorder, setupData interface{}) {
			assertErrorResponse(t, w, http.StatusBadRequest, "")
		},
	}
}

// missingAuthPattern tests handlers without authentication
func missingAuthPattern(method, urlPath string) ErrorTestPattern {
	return ErrorTestPattern{
		Name:           "MissingAuth",
		Description:    "Test handler without authentication",
		ExpectedStatus: http.StatusUnauthorized,
		Setup:          nil,
		Execute: func(t *testing.T, setupData interface{}) HTTPTestRequest {
			return HTTPTestRequest{
				Method: method,
				Path:   urlPath,
				// No user_id in context
			}
		},
		Validate: func(t *testing.T, w *httptest.ResponseRecorder, setupData interface{}) {
			assertErrorResponse(t, w, http.StatusUnauthorized, "authentication required")
		},
	}
}

// emptyBodyPattern tests POST/PUT handlers with empty body
func emptyBodyPattern(method, urlPath string) ErrorTestPattern {
	return ErrorTestPattern{
		Name:           "EmptyBody",
		Description:    "Test handler with empty request body",
		ExpectedStatus: http.StatusBadRequest,
		Setup:          nil,
		Execute: func(t *testing.T, setupData interface{}) HTTPTestRequest {
			return HTTPTestRequest{
				Method:  method,
				Path:    urlPath,
				Body:    "",
				Context: map[string]interface{}{"user_id": uuid.New().String()},
			}
		},
		Validate: func(t *testing.T, w *httptest.ResponseRecorder, setupData interface{}) {
			assertErrorResponse(t, w, http.StatusBadRequest, "")
		},
	}
}

// PerformanceTestPattern defines performance testing scenarios
type PerformanceTestPattern struct {
	Name           string
	Description    string
	MaxDuration    time.Duration
	Iterations     int
	Setup          func(t *testing.T) interface{}
	Execute        func(t *testing.T, setupData interface{}, iteration int) error
	Validate       func(t *testing.T, setupData interface{}, durations []time.Duration)
	Cleanup        func(setupData interface{})
}

// RunPerformanceTest executes a performance test pattern
func RunPerformanceTest(t *testing.T, pattern PerformanceTestPattern) {
	t.Run(fmt.Sprintf("Performance_%s", pattern.Name), func(t *testing.T) {
		// Setup
		var setupData interface{}
		if pattern.Setup != nil {
			setupData = pattern.Setup(t)
		}

		// Cleanup
		if pattern.Cleanup != nil {
			defer pattern.Cleanup(setupData)
		}

		// Execute iterations
		durations := make([]time.Duration, pattern.Iterations)
		for i := 0; i < pattern.Iterations; i++ {
			start := time.Now()
			err := pattern.Execute(t, setupData, i)
			durations[i] = time.Since(start)

			if err != nil {
				t.Errorf("Iteration %d failed: %v", i, err)
			}

			if durations[i] > pattern.MaxDuration {
				t.Errorf("Iteration %d exceeded max duration: %v > %v",
					i, durations[i], pattern.MaxDuration)
			}
		}

		// Validate
		if pattern.Validate != nil {
			pattern.Validate(t, setupData, durations)
		}

		// Report statistics
		avg := averageDuration(durations)
		max := maxDuration(durations)
		min := minDuration(durations)

		t.Logf("Performance stats - Avg: %v, Min: %v, Max: %v", avg, min, max)
	})
}

// ConcurrencyTestPattern defines concurrency testing scenarios
type ConcurrencyTestPattern struct {
	Name        string
	Description string
	Concurrency int
	Iterations  int
	Setup       func(t *testing.T) interface{}
	Execute     func(t *testing.T, setupData interface{}, iteration int) error
	Validate    func(t *testing.T, setupData interface{}, results []error)
	Cleanup     func(setupData interface{})
}

// RunConcurrencyTest executes a concurrency test pattern
func RunConcurrencyTest(t *testing.T, pattern ConcurrencyTestPattern) {
	t.Run(fmt.Sprintf("Concurrency_%s", pattern.Name), func(t *testing.T) {
		// Setup
		var setupData interface{}
		if pattern.Setup != nil {
			setupData = pattern.Setup(t)
		}

		// Cleanup
		if pattern.Cleanup != nil {
			defer pattern.Cleanup(setupData)
		}

		// Execute concurrent iterations
		results := make([]error, pattern.Iterations)
		done := make(chan struct{}, pattern.Concurrency)

		for i := 0; i < pattern.Iterations; i++ {
			// Limit concurrency
			done <- struct{}{}

			go func(iteration int) {
				defer func() { <-done }()
				results[iteration] = pattern.Execute(t, setupData, iteration)
			}(i)
		}

		// Wait for all goroutines to complete
		for i := 0; i < pattern.Concurrency; i++ {
			done <- struct{}{}
		}

		// Count errors
		errorCount := 0
		for i, err := range results {
			if err != nil {
				t.Errorf("Iteration %d failed: %v", i, err)
				errorCount++
			}
		}

		t.Logf("Concurrency test: %d/%d succeeded", pattern.Iterations-errorCount, pattern.Iterations)

		// Validate
		if pattern.Validate != nil {
			pattern.Validate(t, setupData, results)
		}
	})
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
func (b *TestScenarioBuilder) AddInvalidUUID(method, urlPath string) *TestScenarioBuilder {
	b.patterns = append(b.patterns, invalidUUIDPattern(method, urlPath))
	return b
}

// AddNonExistentResource adds non-existent resource test pattern
func (b *TestScenarioBuilder) AddNonExistentResource(method, urlPath string) *TestScenarioBuilder {
	b.patterns = append(b.patterns, nonExistentResourcePattern(method, urlPath))
	return b
}

// AddInvalidJSON adds invalid JSON test pattern
func (b *TestScenarioBuilder) AddInvalidJSON(method, urlPath string) *TestScenarioBuilder {
	b.patterns = append(b.patterns, invalidJSONPattern(method, urlPath))
	return b
}

// AddMissingAuth adds missing authentication test pattern
func (b *TestScenarioBuilder) AddMissingAuth(method, urlPath string) *TestScenarioBuilder {
	b.patterns = append(b.patterns, missingAuthPattern(method, urlPath))
	return b
}

// AddEmptyBody adds empty body test pattern
func (b *TestScenarioBuilder) AddEmptyBody(method, urlPath string) *TestScenarioBuilder {
	b.patterns = append(b.patterns, emptyBodyPattern(method, urlPath))
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

// Utility functions for performance testing

func averageDuration(durations []time.Duration) time.Duration {
	if len(durations) == 0 {
		return 0
	}
	var total time.Duration
	for _, d := range durations {
		total += d
	}
	return total / time.Duration(len(durations))
}

func maxDuration(durations []time.Duration) time.Duration {
	if len(durations) == 0 {
		return 0
	}
	max := durations[0]
	for _, d := range durations {
		if d > max {
			max = d
		}
	}
	return max
}

func minDuration(durations []time.Duration) time.Duration {
	if len(durations) == 0 {
		return 0
	}
	min := durations[0]
	for _, d := range durations {
		if d < min {
			min = d
		}
	}
	return min
}
