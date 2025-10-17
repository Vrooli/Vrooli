package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
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
	Handler     http.HandlerFunc
	BaseURL     string
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
			w := makeHTTPRequest(*req, suite.Handler)

			// Validate
			if pattern.Validate != nil {
				pattern.Validate(t, w, setupData)
			} else {
				// Default validation: check status code
				assertStatus(t, w, pattern.ExpectedStatus)
			}
		})
	}
}

// Common error patterns that can be reused across handlers

// invalidJSONPattern tests handlers with malformed JSON
func invalidJSONPattern(method, path string) ErrorTestPattern {
	return ErrorTestPattern{
		Name:           "InvalidJSON",
		Description:    "Test handler with malformed JSON input",
		ExpectedStatus: http.StatusBadRequest,
		Setup:          nil,
		Execute: func(t *testing.T, setupData interface{}) *HTTPTestRequest {
			return &HTTPTestRequest{
				Method: method,
				Path:   path,
				Body:   `{"invalid": "json"`, // Malformed JSON
			}
		},
		Validate: func(t *testing.T, w *httptest.ResponseRecorder, setupData interface{}) {
			assertStatus(t, w, http.StatusBadRequest)
			assertResponseContains(t, w, "Invalid request body")
		},
	}
}

// emptyBodyPattern tests handlers with empty request body
func emptyBodyPattern(method, path string) ErrorTestPattern {
	return ErrorTestPattern{
		Name:           "EmptyBody",
		Description:    "Test handler with empty request body",
		ExpectedStatus: http.StatusBadRequest,
		Setup:          nil,
		Execute: func(t *testing.T, setupData interface{}) *HTTPTestRequest {
			return &HTTPTestRequest{
				Method: method,
				Path:   path,
				Body:   "",
			}
		},
	}
}

// missingRequiredFieldPattern tests handlers with missing required fields
func missingRequiredFieldPattern(method, path, fieldName string) ErrorTestPattern {
	return ErrorTestPattern{
		Name:           fmt.Sprintf("Missing_%s", fieldName),
		Description:    fmt.Sprintf("Test handler with missing %s field", fieldName),
		ExpectedStatus: http.StatusBadRequest,
		Setup:          nil,
		Execute: func(t *testing.T, setupData interface{}) *HTTPTestRequest {
			return &HTTPTestRequest{
				Method: method,
				Path:   path,
				Body:   map[string]interface{}{},
			}
		},
		Validate: func(t *testing.T, w *httptest.ResponseRecorder, setupData interface{}) {
			assertStatus(t, w, http.StatusBadRequest)
			assertResponseContains(t, w, "required")
		},
	}
}

// methodNotAllowedPattern tests handlers with wrong HTTP method
func methodNotAllowedPattern(wrongMethod, path string) ErrorTestPattern {
	return ErrorTestPattern{
		Name:           fmt.Sprintf("MethodNotAllowed_%s", wrongMethod),
		Description:    fmt.Sprintf("Test handler with %s method (should fail)", wrongMethod),
		ExpectedStatus: http.StatusMethodNotAllowed,
		Setup:          nil,
		Execute: func(t *testing.T, setupData interface{}) *HTTPTestRequest {
			return &HTTPTestRequest{
				Method: wrongMethod,
				Path:   path,
			}
		},
	}
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

// AddInvalidJSON adds invalid JSON test pattern
func (b *TestScenarioBuilder) AddInvalidJSON(method, path string) *TestScenarioBuilder {
	b.patterns = append(b.patterns, invalidJSONPattern(method, path))
	return b
}

// AddEmptyBody adds empty body test pattern
func (b *TestScenarioBuilder) AddEmptyBody(method, path string) *TestScenarioBuilder {
	b.patterns = append(b.patterns, emptyBodyPattern(method, path))
	return b
}

// AddMissingField adds missing required field test pattern
func (b *TestScenarioBuilder) AddMissingField(method, path, fieldName string) *TestScenarioBuilder {
	b.patterns = append(b.patterns, missingRequiredFieldPattern(method, path, fieldName))
	return b
}

// AddMethodNotAllowed adds method not allowed test pattern
func (b *TestScenarioBuilder) AddMethodNotAllowed(wrongMethod, path string) *TestScenarioBuilder {
	b.patterns = append(b.patterns, methodNotAllowedPattern(wrongMethod, path))
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

// Template for comprehensive handler testing
func TemplateComprehensiveHandlerTest(t *testing.T, handlerName string, handler http.HandlerFunc) {
	t.Run(fmt.Sprintf("%s_Comprehensive", handlerName), func(t *testing.T) {
		// 1. Setup
		loggerCleanup := setupTestLogger()
		defer loggerCleanup()

		// 2. Test successful scenarios
		t.Run("Success_Cases", func(t *testing.T) {
			// Add success test cases here
		})

		// 3. Test error conditions
		t.Run("Error_Cases", func(t *testing.T) {
			// Add error test cases here
		})

		// 4. Test edge cases
		t.Run("Edge_Cases", func(t *testing.T) {
			// Add edge case tests here
		})

		// 5. Test performance (if needed)
		t.Run("Performance", func(t *testing.T) {
			// Add performance tests here
		})
	})
}
