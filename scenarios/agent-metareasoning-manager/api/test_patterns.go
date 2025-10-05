// +build testing

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
			w, httpReq, err := makeHTTPRequest(*req)
			if err != nil {
				t.Fatalf("Failed to create HTTP request: %v", err)
			}

			// Execute handler
			suite.Handler(w, httpReq)

			// Validate expected status
			if w.Code != pattern.ExpectedStatus {
				t.Errorf("Expected status %d, got %d. Response: %s", pattern.ExpectedStatus, w.Code, w.Body.String())
			}

			// Custom validation
			if pattern.Validate != nil {
				pattern.Validate(t, w, setupData)
			}
		})
	}
}

// Common error patterns that can be reused across handlers

// invalidJSONPattern tests handlers with malformed JSON
func invalidJSONPattern(method, urlPath string) ErrorTestPattern {
	return ErrorTestPattern{
		Name:           "InvalidJSON",
		Description:    "Test handler with malformed JSON input",
		ExpectedStatus: http.StatusBadRequest,
		Execute: func(t *testing.T, setupData interface{}) *HTTPTestRequest {
			return &HTTPTestRequest{
				Method: method,
				Path:   urlPath,
				Body:   `{"invalid": "json"`, // Malformed JSON
			}
		},
	}
}

// emptyBodyPattern tests handlers with empty request body
func emptyBodyPattern(method, urlPath string) ErrorTestPattern {
	return ErrorTestPattern{
		Name:           "EmptyBody",
		Description:    "Test handler with empty request body",
		ExpectedStatus: http.StatusBadRequest,
		Execute: func(t *testing.T, setupData interface{}) *HTTPTestRequest {
			return &HTTPTestRequest{
				Method: method,
				Path:   urlPath,
				Body:   "",
			}
		},
	}
}

// missingRequiredFieldsPattern tests handlers with missing required fields
func missingRequiredFieldsPattern(method, urlPath string, missingFields map[string]interface{}) ErrorTestPattern {
	return ErrorTestPattern{
		Name:           "MissingRequiredFields",
		Description:    "Test handler with missing required fields",
		ExpectedStatus: http.StatusBadRequest,
		Execute: func(t *testing.T, setupData interface{}) *HTTPTestRequest {
			return &HTTPTestRequest{
				Method: method,
				Path:   urlPath,
				Body:   missingFields,
			}
		},
	}
}

// invalidPlatformPattern tests handlers with invalid platform parameter
func invalidPlatformPattern(urlPath string) ErrorTestPattern {
	return ErrorTestPattern{
		Name:           "InvalidPlatform",
		Description:    "Test handler with invalid platform",
		ExpectedStatus: http.StatusBadRequest,
		Execute: func(t *testing.T, setupData interface{}) *HTTPTestRequest {
			return &HTTPTestRequest{
				Method:  "POST",
				Path:    urlPath,
				URLVars: map[string]string{"platform": "invalid-platform", "workflowId": "test-id"},
			}
		},
	}
}

// invalidWorkflowIDPattern tests handlers with invalid workflow ID
func invalidWorkflowIDPattern(urlPath string) ErrorTestPattern {
	return ErrorTestPattern{
		Name:           "InvalidWorkflowID",
		Description:    "Test handler with invalid workflow ID",
		ExpectedStatus: http.StatusBadRequest,
		Execute: func(t *testing.T, setupData interface{}) *HTTPTestRequest {
			return &HTTPTestRequest{
				Method:  "POST",
				Path:    urlPath,
				URLVars: map[string]string{"platform": "n8n", "workflowId": ""},
			}
		},
	}
}

// nonExistentWorkflowPattern tests handlers with non-existent workflow
func nonExistentWorkflowPattern(urlPath string) ErrorTestPattern {
	return ErrorTestPattern{
		Name:           "NonExistentWorkflow",
		Description:    "Test handler with non-existent workflow",
		ExpectedStatus: http.StatusInternalServerError,
		Execute: func(t *testing.T, setupData interface{}) *HTTPTestRequest {
			nonExistentID := uuid.New().String()
			return &HTTPTestRequest{
				Method:  "POST",
				Path:    urlPath,
				URLVars: map[string]string{"platform": "n8n", "workflowId": nonExistentID},
			}
		},
	}
}

// invalidReasoningTypePattern tests analyze handler with invalid reasoning type
func invalidReasoningTypePattern() ErrorTestPattern {
	return ErrorTestPattern{
		Name:           "InvalidReasoningType",
		Description:    "Test analyze handler with invalid reasoning type",
		ExpectedStatus: http.StatusBadRequest,
		Execute: func(t *testing.T, setupData interface{}) *HTTPTestRequest {
			return &HTTPTestRequest{
				Method: "POST",
				Path:   "/analyze",
				Body: AnalyzeRequest{
					Type:  "invalid-type",
					Input: "test input",
				},
			}
		},
	}
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
func (b *TestScenarioBuilder) AddInvalidJSON(method, urlPath string) *TestScenarioBuilder {
	b.patterns = append(b.patterns, invalidJSONPattern(method, urlPath))
	return b
}

// AddEmptyBody adds empty body test pattern
func (b *TestScenarioBuilder) AddEmptyBody(method, urlPath string) *TestScenarioBuilder {
	b.patterns = append(b.patterns, emptyBodyPattern(method, urlPath))
	return b
}

// AddMissingRequiredFields adds missing fields test pattern
func (b *TestScenarioBuilder) AddMissingRequiredFields(method, urlPath string, fields map[string]interface{}) *TestScenarioBuilder {
	b.patterns = append(b.patterns, missingRequiredFieldsPattern(method, urlPath, fields))
	return b
}

// AddInvalidPlatform adds invalid platform test pattern
func (b *TestScenarioBuilder) AddInvalidPlatform(urlPath string) *TestScenarioBuilder {
	b.patterns = append(b.patterns, invalidPlatformPattern(urlPath))
	return b
}

// AddNonExistentWorkflow adds non-existent workflow test pattern
func (b *TestScenarioBuilder) AddNonExistentWorkflow(urlPath string) *TestScenarioBuilder {
	b.patterns = append(b.patterns, nonExistentWorkflowPattern(urlPath))
	return b
}

// AddInvalidReasoningType adds invalid reasoning type test pattern
func (b *TestScenarioBuilder) AddInvalidReasoningType() *TestScenarioBuilder {
	b.patterns = append(b.patterns, invalidReasoningTypePattern())
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
