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
			w := testHandlerWithRequest(t, suite.Handler, *req)

			// Check status code
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

// invalidAgentIDPattern tests handlers with invalid agent ID formats
func invalidAgentIDPattern(urlPath string) ErrorTestPattern {
	return ErrorTestPattern{
		Name:           "InvalidAgentID",
		Description:    "Test handler with invalid agent ID format",
		ExpectedStatus: http.StatusBadRequest,
		Setup:          nil,
		Execute: func(t *testing.T, setupData interface{}) *HTTPTestRequest {
			return &HTTPTestRequest{
				Method: "GET",
				Path:   urlPath + "?agent_id=invalid-agent-id",
				QueryParams: map[string]string{
					"agent_id": "invalid-agent-id",
				},
			}
		},
		Validate: func(t *testing.T, w *httptest.ResponseRecorder, setupData interface{}) {
			// Verify error message mentions invalid agent ID
			assertErrorResponse(t, w, http.StatusBadRequest, "")
		},
	}
}

// invalidResourcePattern tests handlers with invalid resource names
func invalidResourcePattern(urlPath string) ErrorTestPattern {
	return ErrorTestPattern{
		Name:           "InvalidResource",
		Description:    "Test handler with invalid resource name",
		ExpectedStatus: http.StatusBadRequest,
		Setup:          nil,
		Execute: func(t *testing.T, setupData interface{}) *HTTPTestRequest {
			return &HTTPTestRequest{
				Method: "POST",
				Path:   urlPath,
				Body: map[string]interface{}{
					"resource": "invalid-resource-name",
					"agent_id": "test-agent-123",
				},
			}
		},
	}
}

// invalidJSONPattern tests handlers with malformed JSON
func invalidJSONPattern(urlPath string, method string) ErrorTestPattern {
	return ErrorTestPattern{
		Name:           "InvalidJSON",
		Description:    "Test handler with malformed JSON input",
		ExpectedStatus: http.StatusBadRequest,
		Setup:          nil,
		Execute: func(t *testing.T, setupData interface{}) *HTTPTestRequest {
			return &HTTPTestRequest{
				Method: method,
				Path:   urlPath,
				Body:   `{"invalid": "json"`, // Malformed JSON
			}
		},
	}
}

// missingQueryParamPattern tests handlers with missing required query parameters
func missingQueryParamPattern(urlPath string, paramName string) ErrorTestPattern {
	return ErrorTestPattern{
		Name:           fmt.Sprintf("Missing%sParam", paramName),
		Description:    fmt.Sprintf("Test handler with missing %s query parameter", paramName),
		ExpectedStatus: http.StatusBadRequest,
		Setup:          nil,
		Execute: func(t *testing.T, setupData interface{}) *HTTPTestRequest {
			return &HTTPTestRequest{
				Method: "GET",
				Path:   urlPath,
			}
		},
	}
}

// invalidLineCountPattern tests log endpoint with invalid line counts
func invalidLineCountPattern(urlPath string) ErrorTestPattern {
	return ErrorTestPattern{
		Name:           "InvalidLineCount",
		Description:    "Test handler with invalid line count parameter",
		ExpectedStatus: http.StatusBadRequest,
		Setup:          nil,
		Execute: func(t *testing.T, setupData interface{}) *HTTPTestRequest {
			return &HTTPTestRequest{
				Method: "GET",
				Path:   urlPath,
				QueryParams: map[string]string{
					"agent_id": "claude-code:agent-123",
					"lines":    "99999", // Exceeds maximum
				},
			}
		},
	}
}

// PerformanceTestPattern defines performance testing scenarios
type PerformanceTestPattern struct {
	Name           string
	Description    string
	MaxDuration    time.Duration
	Setup          func(t *testing.T) interface{}
	Execute        func(t *testing.T, setupData interface{}) time.Duration
	Cleanup        func(setupData interface{})
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

// AddInvalidAgentID adds invalid agent ID test pattern
func (b *TestScenarioBuilder) AddInvalidAgentID(urlPath string) *TestScenarioBuilder {
	b.patterns = append(b.patterns, invalidAgentIDPattern(urlPath))
	return b
}

// AddInvalidResource adds invalid resource test pattern
func (b *TestScenarioBuilder) AddInvalidResource(urlPath string) *TestScenarioBuilder {
	b.patterns = append(b.patterns, invalidResourcePattern(urlPath))
	return b
}

// AddInvalidJSON adds invalid JSON test pattern
func (b *TestScenarioBuilder) AddInvalidJSON(urlPath string, method string) *TestScenarioBuilder {
	b.patterns = append(b.patterns, invalidJSONPattern(urlPath, method))
	return b
}

// AddMissingQueryParam adds missing query parameter test pattern
func (b *TestScenarioBuilder) AddMissingQueryParam(urlPath string, paramName string) *TestScenarioBuilder {
	b.patterns = append(b.patterns, missingQueryParamPattern(urlPath, paramName))
	return b
}

// AddInvalidLineCount adds invalid line count test pattern
func (b *TestScenarioBuilder) AddInvalidLineCount(urlPath string) *TestScenarioBuilder {
	b.patterns = append(b.patterns, invalidLineCountPattern(urlPath))
	return b
}

// AddCustom adds a custom test pattern
func (b *TestScenarioBuilder) AddCustom(pattern ErrorTestPattern) *TestScenarioBuilder {
	b.patterns = append(b.patterns, pattern)
	return b
}

// Build returns the complete set of test patterns
func (b *TestScenarioBuilder) Build() []ErrorTestPattern {
	return b.patterns
}
