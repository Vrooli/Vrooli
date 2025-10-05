package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// TestScenarioBuilder provides a fluent interface for building test scenarios
type TestScenarioBuilder struct {
	scenarios []TestScenario
}

// TestScenario represents a single test case
type TestScenario struct {
	Name           string
	Description    string
	Method         string
	Path           string
	Body           string
	ExpectedStatus int
	ExpectedKeys   []string
	ExpectedError  string
	Setup          func(t *testing.T) interface{}
	Cleanup        func(data interface{})
}

// NewTestScenarioBuilder creates a new test scenario builder
func NewTestScenarioBuilder() *TestScenarioBuilder {
	return &TestScenarioBuilder{
		scenarios: make([]TestScenario, 0),
	}
}

// AddScenario adds a custom test scenario
func (b *TestScenarioBuilder) AddScenario(scenario TestScenario) *TestScenarioBuilder {
	b.scenarios = append(b.scenarios, scenario)
	return b
}

// AddInvalidJSON adds a test for invalid JSON input
func (b *TestScenarioBuilder) AddInvalidJSON(path, method string) *TestScenarioBuilder {
	b.scenarios = append(b.scenarios, TestScenario{
		Name:           "InvalidJSON",
		Description:    "Test handler with malformed JSON",
		Method:         method,
		Path:           path,
		Body:           `{"invalid": "json"`,
		ExpectedStatus: http.StatusBadRequest,
		ExpectedError:  "Invalid request body",
	})
	return b
}

// AddMissingField adds a test for missing required field
func (b *TestScenarioBuilder) AddMissingField(path, method, fieldName string) *TestScenarioBuilder {
	b.scenarios = append(b.scenarios, TestScenario{
		Name:           fmt.Sprintf("Missing_%s", fieldName),
		Description:    fmt.Sprintf("Test handler with missing %s", fieldName),
		Method:         method,
		Path:           path,
		Body:           `{}`,
		ExpectedStatus: http.StatusBadRequest,
		ExpectedError:  fieldName,
	})
	return b
}

// AddEmptyBody adds a test for empty request body
func (b *TestScenarioBuilder) AddEmptyBody(path, method string) *TestScenarioBuilder {
	b.scenarios = append(b.scenarios, TestScenario{
		Name:           "EmptyBody",
		Description:    "Test handler with empty request body",
		Method:         method,
		Path:           path,
		Body:           "",
		ExpectedStatus: http.StatusBadRequest,
	})
	return b
}

// Build returns the built scenarios
func (b *TestScenarioBuilder) Build() []TestScenario {
	return b.scenarios
}

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
	Handler     http.HandlerFunc
	BaseURL     string
}

// RunSuccessTests executes success path tests
func (suite *HandlerTestSuite) RunSuccessTests(t *testing.T, scenarios []TestScenario) {
	for _, scenario := range scenarios {
		t.Run(fmt.Sprintf("%s_Success_%s", suite.HandlerName, scenario.Name), func(t *testing.T) {
			cleanup := setupTestLogger()
			defer cleanup()

			// Setup
			var setupData interface{}
			if scenario.Setup != nil {
				setupData = scenario.Setup(t)
			}

			// Cleanup
			if scenario.Cleanup != nil {
				defer scenario.Cleanup(setupData)
			}

			// Execute
			req := HTTPTestRequest{
				Method: scenario.Method,
				Path:   scenario.Path,
				Body:   scenario.Body,
			}
			w := executeHandler(suite.Handler, req)

			// Validate
			if scenario.ExpectedKeys != nil {
				assertJSONResponse(t, w, scenario.ExpectedStatus, scenario.ExpectedKeys)
			} else {
				if w.Code != scenario.ExpectedStatus {
					t.Errorf("Expected status %d, got %d", scenario.ExpectedStatus, w.Code)
				}
			}
		})
	}
}

// RunErrorTests executes error condition tests
func (suite *HandlerTestSuite) RunErrorTests(t *testing.T, scenarios []TestScenario) {
	for _, scenario := range scenarios {
		t.Run(fmt.Sprintf("%s_Error_%s", suite.HandlerName, scenario.Name), func(t *testing.T) {
			cleanup := setupTestLogger()
			defer cleanup()

			// Setup
			var setupData interface{}
			if scenario.Setup != nil {
				setupData = scenario.Setup(t)
			}

			// Cleanup
			if scenario.Cleanup != nil {
				defer scenario.Cleanup(setupData)
			}

			// Execute
			req := HTTPTestRequest{
				Method: scenario.Method,
				Path:   scenario.Path,
				Body:   scenario.Body,
			}
			w := executeHandler(suite.Handler, req)

			// Validate
			if scenario.ExpectedError != "" {
				assertErrorResponse(t, w, scenario.ExpectedStatus, scenario.ExpectedError)
			} else {
				if w.Code != scenario.ExpectedStatus {
					t.Errorf("Expected status %d, got %d", scenario.ExpectedStatus, w.Code)
				}
			}
		})
	}
}

// PerformanceTestPattern defines performance testing scenarios
type PerformanceTestPattern struct {
	Name           string
	Description    string
	MaxDuration    time.Duration
	IterationCount int
	Setup          func(t *testing.T) interface{}
	Execute        func(t *testing.T, setupData interface{})
	Cleanup        func(setupData interface{})
}

// RunPerformanceTest executes a performance test pattern
func RunPerformanceTest(t *testing.T, pattern PerformanceTestPattern) {
	cleanup := setupTestLogger()
	defer cleanup()

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
	startTime := time.Now()
	for i := 0; i < pattern.IterationCount; i++ {
		pattern.Execute(t, setupData)
	}
	duration := time.Since(startTime)

	// Validate
	if duration > pattern.MaxDuration {
		t.Errorf("Performance test '%s' exceeded max duration: %v > %v",
			pattern.Name, duration, pattern.MaxDuration)
	}

	avgDuration := duration / time.Duration(pattern.IterationCount)
	t.Logf("Performance test '%s': %d iterations in %v (avg: %v)",
		pattern.Name, pattern.IterationCount, duration, avgDuration)
}

// Common test data builders

// buildValidAnalyzeRequest creates a valid analyze request
func buildValidAnalyzeRequest() map[string]interface{} {
	return map[string]interface{}{
		"paths":    []string{"/test/file.go"},
		"auto_fix": false,
	}
}

// buildValidFixRequest creates a valid fix request
func buildValidFixRequest() map[string]interface{} {
	return map[string]interface{}{
		"violation_id": "test-violation-1",
		"action":       "approve",
	}
}

// buildValidLearnRequest creates a valid learn request
func buildValidLearnRequest() map[string]interface{} {
	return map[string]interface{}{
		"pattern":     "test-pattern",
		"is_positive": true,
		"context": map[string]interface{}{
			"file": "test.go",
		},
	}
}
