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
			w, err := makeHTTPRequest(*req)
			if err != nil {
				t.Fatalf("Failed to create HTTP request: %v", err)
			}

			// Execute the handler
			httpReq, _ := http.NewRequest(req.Method, req.Path, nil)
			suite.Handler(w, httpReq)

			// Validate status code
			if w.Code != pattern.ExpectedStatus {
				t.Errorf("Expected status %d, got %d", pattern.ExpectedStatus, w.Code)
			}

			// Custom validation
			if pattern.Validate != nil {
				pattern.Validate(t, w, setupData)
			}
		})
	}
}

// TestScenarioBuilder provides a fluent interface for building test scenarios
type TestScenarioBuilder struct {
	scenarios []ErrorTestPattern
}

// NewTestScenarioBuilder creates a new test scenario builder
func NewTestScenarioBuilder() *TestScenarioBuilder {
	return &TestScenarioBuilder{
		scenarios: make([]ErrorTestPattern, 0),
	}
}

// AddInvalidMethod adds a test for invalid HTTP method
func (b *TestScenarioBuilder) AddInvalidMethod(urlPath string, invalidMethod string, validMethod string) *TestScenarioBuilder {
	b.scenarios = append(b.scenarios, ErrorTestPattern{
		Name:           "InvalidMethod_" + invalidMethod,
		Description:    fmt.Sprintf("Test handler with invalid method %s (expects %s)", invalidMethod, validMethod),
		ExpectedStatus: http.StatusMethodNotAllowed,
		Setup:          nil,
		Execute: func(t *testing.T, setupData interface{}) *HTTPTestRequest {
			return &HTTPTestRequest{
				Method: invalidMethod,
				Path:   urlPath,
			}
		},
	})
	return b
}

// AddMalformedJSON adds a test for malformed JSON input
func (b *TestScenarioBuilder) AddMalformedJSON(urlPath string) *TestScenarioBuilder {
	b.scenarios = append(b.scenarios, ErrorTestPattern{
		Name:           "MalformedJSON",
		Description:    "Test handler with malformed JSON",
		ExpectedStatus: http.StatusBadRequest,
		Setup:          nil,
		Execute: func(t *testing.T, setupData interface{}) *HTTPTestRequest {
			return &HTTPTestRequest{
				Method:  "POST",
				Path:    urlPath,
				Headers: map[string]string{"Content-Type": "application/json"},
				Body:    "not valid json",
			}
		},
	})
	return b
}

// AddEmptyBody adds a test for empty request body
func (b *TestScenarioBuilder) AddEmptyBody(urlPath string) *TestScenarioBuilder {
	b.scenarios = append(b.scenarios, ErrorTestPattern{
		Name:           "EmptyBody",
		Description:    "Test handler with empty request body",
		ExpectedStatus: http.StatusBadRequest,
		Setup:          nil,
		Execute: func(t *testing.T, setupData interface{}) *HTTPTestRequest {
			return &HTTPTestRequest{
				Method: "POST",
				Path:   urlPath,
			}
		},
	})
	return b
}

// AddMissingField adds a test for missing required field
func (b *TestScenarioBuilder) AddMissingField(urlPath string, fieldName string) *TestScenarioBuilder {
	b.scenarios = append(b.scenarios, ErrorTestPattern{
		Name:           "MissingField_" + fieldName,
		Description:    fmt.Sprintf("Test handler with missing field: %s", fieldName),
		ExpectedStatus: http.StatusBadRequest,
		Setup:          nil,
		Execute: func(t *testing.T, setupData interface{}) *HTTPTestRequest {
			// Create a body with the field missing
			body := make(map[string]interface{})
			return &HTTPTestRequest{
				Method: "POST",
				Path:   urlPath,
				Body:   body,
			}
		},
	})
	return b
}

// AddInvalidFieldValue adds a test for invalid field value
func (b *TestScenarioBuilder) AddInvalidFieldValue(urlPath string, fieldName string, invalidValue interface{}) *TestScenarioBuilder {
	b.scenarios = append(b.scenarios, ErrorTestPattern{
		Name:           "InvalidField_" + fieldName,
		Description:    fmt.Sprintf("Test handler with invalid field %s: %v", fieldName, invalidValue),
		ExpectedStatus: http.StatusBadRequest,
		Setup:          nil,
		Execute: func(t *testing.T, setupData interface{}) *HTTPTestRequest {
			body := map[string]interface{}{
				fieldName: invalidValue,
			}
			return &HTTPTestRequest{
				Method: "POST",
				Path:   urlPath,
				Body:   body,
			}
		},
	})
	return b
}

// Build returns all configured error test patterns
func (b *TestScenarioBuilder) Build() []ErrorTestPattern {
	return b.scenarios
}

// Common reusable error patterns

// invalidMethodPattern tests handlers with wrong HTTP methods
func invalidMethodPattern(urlPath string, invalidMethod string) ErrorTestPattern {
	return ErrorTestPattern{
		Name:           "InvalidMethod",
		Description:    "Test handler with invalid HTTP method",
		ExpectedStatus: http.StatusMethodNotAllowed,
		Setup:          nil,
		Execute: func(t *testing.T, setupData interface{}) *HTTPTestRequest {
			return &HTTPTestRequest{
				Method: invalidMethod,
				Path:   urlPath,
			}
		},
	}
}

// malformedJSONPattern tests handlers with malformed JSON
func malformedJSONPattern(urlPath string) ErrorTestPattern {
	return ErrorTestPattern{
		Name:           "MalformedJSON",
		Description:    "Test handler with malformed JSON",
		ExpectedStatus: http.StatusBadRequest,
		Setup:          nil,
		Execute: func(t *testing.T, setupData interface{}) *HTTPTestRequest {
			return &HTTPTestRequest{
				Method:  "POST",
				Path:    urlPath,
				Headers: map[string]string{"Content-Type": "application/json"},
				Body:    "invalid json",
			}
		},
	}
}

// emptyBodyPattern tests handlers with empty body
func emptyBodyPattern(urlPath string) ErrorTestPattern {
	return ErrorTestPattern{
		Name:           "EmptyBody",
		Description:    "Test handler with empty request body",
		ExpectedStatus: http.StatusBadRequest,
		Setup:          nil,
		Execute: func(t *testing.T, setupData interface{}) *HTTPTestRequest {
			return &HTTPTestRequest{
				Method: "POST",
				Path:   urlPath,
			}
		},
	}
}
