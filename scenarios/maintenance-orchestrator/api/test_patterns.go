package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestScenarioBuilder provides a fluent interface for building test scenarios
type TestScenarioBuilder struct {
	scenarios []TestScenario
}

// TestScenario represents a test case scenario
type TestScenario struct {
	Name           string
	Description    string
	Method         string
	Path           string
	Body           interface{}
	ExpectedStatus int
	ExpectedError  string
	ValidateFunc   func(t *testing.T, resp map[string]interface{})
}

// NewTestScenarioBuilder creates a new test scenario builder
func NewTestScenarioBuilder() *TestScenarioBuilder {
	return &TestScenarioBuilder{
		scenarios: make([]TestScenario, 0),
	}
}

// AddInvalidID adds a test for invalid ID format
func (b *TestScenarioBuilder) AddInvalidID(path string) *TestScenarioBuilder {
	b.scenarios = append(b.scenarios, TestScenario{
		Name:           "InvalidID",
		Description:    "Test with invalid ID format",
		Method:         "GET",
		Path:           path,
		ExpectedStatus: http.StatusNotFound,
		ExpectedError:  "Scenario not found",
	})
	return b
}

// AddNonExistentScenario adds a test for non-existent scenario
func (b *TestScenarioBuilder) AddNonExistentScenario(method, path string) *TestScenarioBuilder {
	b.scenarios = append(b.scenarios, TestScenario{
		Name:           "NonExistentScenario",
		Description:    "Test with non-existent scenario ID",
		Method:         method,
		Path:           path,
		ExpectedStatus: http.StatusNotFound,
		ExpectedError:  "Scenario not found",
	})
	return b
}

// AddNonExistentPreset adds a test for non-existent preset
func (b *TestScenarioBuilder) AddNonExistentPreset(path string) *TestScenarioBuilder {
	b.scenarios = append(b.scenarios, TestScenario{
		Name:           "NonExistentPreset",
		Description:    "Test with non-existent preset ID",
		Method:         "POST",
		Path:           path,
		ExpectedStatus: http.StatusNotFound,
		ExpectedError:  "Preset not found",
	})
	return b
}

// AddInvalidJSON adds a test for malformed JSON
func (b *TestScenarioBuilder) AddInvalidJSON(path string) *TestScenarioBuilder {
	b.scenarios = append(b.scenarios, TestScenario{
		Name:           "InvalidJSON",
		Description:    "Test with malformed JSON body",
		Method:         "POST",
		Path:           path,
		Body:           "invalid json",
		ExpectedStatus: http.StatusBadRequest,
	})
	return b
}

// AddCustom adds a custom test scenario
func (b *TestScenarioBuilder) AddCustom(scenario TestScenario) *TestScenarioBuilder {
	b.scenarios = append(b.scenarios, scenario)
	return b
}

// Build returns the built test scenarios
func (b *TestScenarioBuilder) Build() []TestScenario {
	return b.scenarios
}

// ErrorTestPattern defines a systematic approach to testing error conditions
type ErrorTestPattern struct {
	Name           string
	Description    string
	ExpectedStatus int
	Setup          func(t *testing.T) interface{}
	Execute        func(t *testing.T, env *TestEnvironment, setupData interface{}) *HTTPTestRequest
	Validate       func(t *testing.T, w *httptest.ResponseRecorder, setupData interface{})
	Cleanup        func(setupData interface{})
}

// HandlerTestSuite provides a comprehensive test framework for HTTP handlers
type HandlerTestSuite struct {
	HandlerName     string
	BaseURL         string
	RequiredURLVars []string
}

// RunErrorTests executes a suite of error condition tests
func (suite *HandlerTestSuite) RunErrorTests(t *testing.T, env *TestEnvironment, patterns []ErrorTestPattern) {
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
			req := pattern.Execute(t, env, setupData)
			w := makeHTTPRequest(env, *req)

			// Check status code
			if w.Code != pattern.ExpectedStatus {
				t.Errorf("Expected status %d, got %d. Body: %s", pattern.ExpectedStatus, w.Code, w.Body.String())
			}

			// Validate
			if pattern.Validate != nil {
				pattern.Validate(t, w, setupData)
			}
		})
	}
}

// Common error patterns that can be reused across handlers

// invalidIDPattern tests handlers with invalid ID formats
func invalidIDPattern(method, urlPath string) ErrorTestPattern {
	return ErrorTestPattern{
		Name:           "InvalidID",
		Description:    "Test handler with invalid ID format",
		ExpectedStatus: http.StatusNotFound,
		Setup:          nil,
		Execute: func(t *testing.T, env *TestEnvironment, setupData interface{}) *HTTPTestRequest {
			return &HTTPTestRequest{
				Method: method,
				Path:   urlPath,
			}
		},
	}
}

// nonExistentScenarioPattern tests handlers with non-existent scenario IDs
func nonExistentScenarioPattern(method, urlPath string) ErrorTestPattern {
	return ErrorTestPattern{
		Name:           "NonExistentScenario",
		Description:    "Test handler with non-existent scenario ID",
		ExpectedStatus: http.StatusNotFound,
		Setup:          nil,
		Execute: func(t *testing.T, env *TestEnvironment, setupData interface{}) *HTTPTestRequest {
			return &HTTPTestRequest{
				Method: method,
				Path:   urlPath,
			}
		},
	}
}

// nonExistentPresetPattern tests handlers with non-existent preset IDs
func nonExistentPresetPattern(urlPath string) ErrorTestPattern {
	return ErrorTestPattern{
		Name:           "NonExistentPreset",
		Description:    "Test handler with non-existent preset ID",
		ExpectedStatus: http.StatusNotFound,
		Setup:          nil,
		Execute: func(t *testing.T, env *TestEnvironment, setupData interface{}) *HTTPTestRequest {
			return &HTTPTestRequest{
				Method: "POST",
				Path:   urlPath,
			}
		},
	}
}

// TableTestCase represents a table-driven test case
type TableTestCase struct {
	Name           string
	Setup          func(*TestEnvironment)
	Request        HTTPTestRequest
	ExpectedStatus int
	Validate       func(*testing.T, *httptest.ResponseRecorder)
}

// RunTableTests executes table-driven tests
func RunTableTests(t *testing.T, env *TestEnvironment, tests []TableTestCase) {
	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			if tt.Setup != nil {
				tt.Setup(env)
			}

			w := makeHTTPRequest(env, tt.Request)

			if w.Code != tt.ExpectedStatus {
				t.Errorf("Expected status %d, got %d. Body: %s", tt.ExpectedStatus, w.Code, w.Body.String())
			}

			if tt.Validate != nil {
				tt.Validate(t, w)
			}
		})
	}
}
