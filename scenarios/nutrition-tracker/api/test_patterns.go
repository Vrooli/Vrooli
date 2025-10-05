// +build testing

package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

// ErrorTestPattern defines a systematic approach to testing error conditions
type ErrorTestPattern struct {
	Name           string
	Description    string
	ExpectedStatus int
	Setup          func(t *testing.T) interface{}
	Execute        func(t *testing.T, setupData interface{}) HTTPTestRequest
	Validate       func(t *testing.T, recorder *httptest.ResponseRecorder, setupData interface{})
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
			recorder, err := makeHTTPRequest(req, suite.Handler)
			if err != nil {
				t.Fatalf("Failed to make HTTP request: %v", err)
			}

			// Validate status code
			if recorder.Code != pattern.ExpectedStatus {
				t.Errorf("Expected status %d, got %d. Body: %s",
					pattern.ExpectedStatus, recorder.Code, recorder.Body.String())
			}

			// Custom validation
			if pattern.Validate != nil {
				pattern.Validate(t, recorder, setupData)
			}
		})
	}
}

// TestScenarioBuilder provides a fluent interface for building test scenarios
type TestScenarioBuilder struct {
	patterns []ErrorTestPattern
}

// NewTestScenarioBuilder creates a new test scenario builder
func NewTestScenarioBuilder() *TestScenarioBuilder {
	return &TestScenarioBuilder{
		patterns: make([]ErrorTestPattern, 0),
	}
}

// AddPattern adds a custom error test pattern
func (b *TestScenarioBuilder) AddPattern(pattern ErrorTestPattern) *TestScenarioBuilder {
	b.patterns = append(b.patterns, pattern)
	return b
}

// AddInvalidJSON adds a test for invalid JSON input
func (b *TestScenarioBuilder) AddInvalidJSON(method, path string) *TestScenarioBuilder {
	pattern := ErrorTestPattern{
		Name:           "InvalidJSON",
		Description:    "Test handler with invalid JSON in request body",
		ExpectedStatus: http.StatusBadRequest,
		Execute: func(t *testing.T, setupData interface{}) HTTPTestRequest {
			return HTTPTestRequest{
				Method: method,
				Path:   path,
				Body:   "invalid json",
			}
		},
	}
	b.patterns = append(b.patterns, pattern)
	return b
}

// AddMissingQueryParam adds a test for missing required query parameters
func (b *TestScenarioBuilder) AddMissingQueryParam(path, paramName string) *TestScenarioBuilder {
	pattern := ErrorTestPattern{
		Name:           fmt.Sprintf("Missing%sParam", paramName),
		Description:    fmt.Sprintf("Test handler without required %s parameter", paramName),
		ExpectedStatus: http.StatusBadRequest,
		Execute: func(t *testing.T, setupData interface{}) HTTPTestRequest {
			return HTTPTestRequest{
				Method: "GET",
				Path:   path,
				Query:  map[string]string{},
			}
		},
	}
	b.patterns = append(b.patterns, pattern)
	return b
}

// AddNonExistentResource adds a test for non-existent resource
func (b *TestScenarioBuilder) AddNonExistentResource(method, path string, idVar string) *TestScenarioBuilder {
	pattern := ErrorTestPattern{
		Name:           "NonExistentResource",
		Description:    "Test handler with non-existent resource ID",
		ExpectedStatus: http.StatusNotFound,
		Execute: func(t *testing.T, setupData interface{}) HTTPTestRequest {
			return HTTPTestRequest{
				Method:  method,
				Path:    path,
				URLVars: map[string]string{idVar: "00000000-0000-0000-0000-000000000000"},
			}
		},
	}
	b.patterns = append(b.patterns, pattern)
	return b
}

// AddEmptyBody adds a test for empty request body when body is required
func (b *TestScenarioBuilder) AddEmptyBody(method, path string) *TestScenarioBuilder {
	pattern := ErrorTestPattern{
		Name:           "EmptyBody",
		Description:    "Test handler with empty request body",
		ExpectedStatus: http.StatusBadRequest,
		Execute: func(t *testing.T, setupData interface{}) HTTPTestRequest {
			return HTTPTestRequest{
				Method: method,
				Path:   path,
				Body:   nil,
			}
		},
	}
	b.patterns = append(b.patterns, pattern)
	return b
}

// AddInvalidDateFormat adds a test for invalid date format
func (b *TestScenarioBuilder) AddInvalidDateFormat(path, paramName string) *TestScenarioBuilder {
	pattern := ErrorTestPattern{
		Name:           "InvalidDateFormat",
		Description:    "Test handler with invalid date format",
		ExpectedStatus: http.StatusBadRequest,
		Execute: func(t *testing.T, setupData interface{}) HTTPTestRequest {
			return HTTPTestRequest{
				Method: "GET",
				Path:   path,
				Query:  map[string]string{paramName: "invalid-date"},
			}
		},
	}
	b.patterns = append(b.patterns, pattern)
	return b
}

// Build returns the list of error test patterns
func (b *TestScenarioBuilder) Build() []ErrorTestPattern {
	return b.patterns
}

// Common error pattern generators

// InvalidJSONPattern creates a pattern for testing invalid JSON input
func InvalidJSONPattern(method, path string) ErrorTestPattern {
	return ErrorTestPattern{
		Name:           "InvalidJSON",
		Description:    "Test with invalid JSON body",
		ExpectedStatus: http.StatusBadRequest,
		Execute: func(t *testing.T, setupData interface{}) HTTPTestRequest {
			return HTTPTestRequest{
				Method: method,
				Path:   path,
				Body:   "not valid json",
			}
		},
	}
}

// MissingRequiredFieldPattern creates a pattern for testing missing required fields
func MissingRequiredFieldPattern(method, path string, body interface{}) ErrorTestPattern {
	return ErrorTestPattern{
		Name:           "MissingRequiredField",
		Description:    "Test with missing required field in body",
		ExpectedStatus: http.StatusBadRequest,
		Execute: func(t *testing.T, setupData interface{}) HTTPTestRequest {
			return HTTPTestRequest{
				Method: method,
				Path:   path,
				Body:   body,
			}
		},
	}
}

// DatabaseErrorPattern creates a pattern for simulating database errors
func DatabaseErrorPattern(method, path string) ErrorTestPattern {
	return ErrorTestPattern{
		Name:           "DatabaseError",
		Description:    "Test handler behavior with database errors",
		ExpectedStatus: http.StatusInternalServerError,
		Setup: func(t *testing.T) interface{} {
			// This would typically involve mocking or temporarily breaking DB connection
			return nil
		},
		Execute: func(t *testing.T, setupData interface{}) HTTPTestRequest {
			return HTTPTestRequest{
				Method: method,
				Path:   path,
			}
		},
	}
}

// UnauthorizedAccessPattern creates a pattern for testing unauthorized access
func UnauthorizedAccessPattern(method, path string) ErrorTestPattern {
	return ErrorTestPattern{
		Name:           "UnauthorizedAccess",
		Description:    "Test handler with unauthorized access",
		ExpectedStatus: http.StatusUnauthorized,
		Execute: func(t *testing.T, setupData interface{}) HTTPTestRequest {
			return HTTPTestRequest{
				Method:  method,
				Path:    path,
				Headers: map[string]string{},
			}
		},
	}
}

// EdgeCasePattern creates patterns for boundary value testing
func EdgeCasePattern(name, method, path string, body interface{}, expectedStatus int) ErrorTestPattern {
	return ErrorTestPattern{
		Name:           name,
		Description:    fmt.Sprintf("Test edge case: %s", name),
		ExpectedStatus: expectedStatus,
		Execute: func(t *testing.T, setupData interface{}) HTTPTestRequest {
			return HTTPTestRequest{
				Method: method,
				Path:   path,
				Body:   body,
			}
		},
	}
}
