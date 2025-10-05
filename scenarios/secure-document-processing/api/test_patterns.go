package main

import (
	"fmt"
	"net/http"
	"testing"
)

// TestScenario represents a test scenario with expected behavior
type TestScenario struct {
	Name           string
	Description    string
	Request        HTTPTestRequest
	ExpectedStatus int
	Validate       func(t *testing.T, w interface{})
}

// TestScenarioBuilder provides a fluent interface for building test scenarios
type TestScenarioBuilder struct {
	scenarios []TestScenario
}

// NewTestScenarioBuilder creates a new test scenario builder
func NewTestScenarioBuilder() *TestScenarioBuilder {
	return &TestScenarioBuilder{
		scenarios: make([]TestScenario, 0),
	}
}

// AddScenario adds a custom scenario
func (b *TestScenarioBuilder) AddScenario(scenario TestScenario) *TestScenarioBuilder {
	b.scenarios = append(b.scenarios, scenario)
	return b
}

// AddInvalidMethod adds a test for invalid HTTP method
func (b *TestScenarioBuilder) AddInvalidMethod(path string, validMethod string) *TestScenarioBuilder {
	invalidMethods := []string{"POST", "PUT", "DELETE", "PATCH"}
	for _, method := range invalidMethods {
		if method != validMethod {
			b.scenarios = append(b.scenarios, TestScenario{
				Name:        fmt.Sprintf("InvalidMethod_%s", method),
				Description: fmt.Sprintf("Test %s with invalid method %s", path, method),
				Request: HTTPTestRequest{
					Method: method,
					Path:   path,
				},
				ExpectedStatus: http.StatusMethodNotAllowed,
			})
			break // Just add one invalid method test
		}
	}
	return b
}

// AddMissingContentType adds a test for missing Content-Type header
func (b *TestScenarioBuilder) AddMissingContentType(path string, method string) *TestScenarioBuilder {
	b.scenarios = append(b.scenarios, TestScenario{
		Name:        "MissingContentType",
		Description: "Test request with missing Content-Type header",
		Request: HTTPTestRequest{
			Method: method,
			Path:   path,
			Body:   map[string]string{"test": "data"},
		},
		ExpectedStatus: http.StatusOK, // API should still work even without Content-Type
	})
	return b
}

// AddEmptyResponse adds a test for handlers that should return empty arrays
func (b *TestScenarioBuilder) AddEmptyResponse(path string, method string) *TestScenarioBuilder {
	b.scenarios = append(b.scenarios, TestScenario{
		Name:        "EmptyResponse",
		Description: "Test that handler returns valid empty response",
		Request: HTTPTestRequest{
			Method: method,
			Path:   path,
		},
		ExpectedStatus: http.StatusOK,
	})
	return b
}

// Build returns the constructed test scenarios
func (b *TestScenarioBuilder) Build() []TestScenario {
	return b.scenarios
}

// ErrorTestPattern defines a systematic approach to testing error conditions
type ErrorTestPattern struct {
	Name           string
	Description    string
	ExpectedStatus int
	Setup          func(t *testing.T) interface{}
	Execute        func(t *testing.T, setupData interface{}) HTTPTestRequest
	Validate       func(t *testing.T, w interface{}, setupData interface{})
	Cleanup        func(setupData interface{})
}

// HandlerTestSuite provides a comprehensive test framework for HTTP handlers
type HandlerTestSuite struct {
	HandlerName string
	Handler     http.HandlerFunc
	BaseURL     string
}

// RunTests executes a suite of test scenarios
func (suite *HandlerTestSuite) RunTests(t *testing.T, scenarios []TestScenario) {
	for _, scenario := range scenarios {
		t.Run(fmt.Sprintf("%s_%s", suite.HandlerName, scenario.Name), func(t *testing.T) {
			cleanup := setupTestLogger()
			defer cleanup()

			w, err := makeHTTPRequest(scenario.Request, suite.Handler)
			if err != nil {
				t.Fatalf("Failed to create HTTP request: %v", err)
			}

			if w.Code != scenario.ExpectedStatus {
				t.Errorf("Expected status %d, got %d. Body: %s",
					scenario.ExpectedStatus, w.Code, w.Body.String())
			}

			if scenario.Validate != nil {
				scenario.Validate(t, w)
			}
		})
	}
}

// RunErrorTests executes a suite of error condition tests
func (suite *HandlerTestSuite) RunErrorTests(t *testing.T, patterns []ErrorTestPattern) {
	for _, pattern := range patterns {
		t.Run(fmt.Sprintf("%s_%s", suite.HandlerName, pattern.Name), func(t *testing.T) {
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
			req := pattern.Execute(t, setupData)
			w, err := makeHTTPRequest(req, suite.Handler)
			if err != nil {
				t.Fatalf("Failed to create HTTP request: %v", err)
			}

			if w.Code != pattern.ExpectedStatus {
				t.Errorf("Expected status %d, got %d. Body: %s",
					pattern.ExpectedStatus, w.Code, w.Body.String())
			}

			// Validate
			if pattern.Validate != nil {
				pattern.Validate(t, w, setupData)
			}
		})
	}
}
