// +build testing

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
	Setup          func(t *testing.T, env *TestEnvironment) interface{}
	Execute        func(t *testing.T, env *TestEnvironment, setupData interface{}) HTTPTestRequest
	Validate       func(t *testing.T, w *httptest.ResponseRecorder, setupData interface{})
	Cleanup        func(setupData interface{})
}

// HandlerTestSuite provides a comprehensive test framework for HTTP handlers
type HandlerTestSuite struct {
	HandlerName string
	Handler     http.Handler
	BaseURL     string
}

// RunErrorTests executes a suite of error condition tests
func (suite *HandlerTestSuite) RunErrorTests(t *testing.T, env *TestEnvironment, patterns []ErrorTestPattern) {
	for _, pattern := range patterns {
		t.Run(fmt.Sprintf("%s_%s", suite.HandlerName, pattern.Name), func(t *testing.T) {
			// Setup
			var setupData interface{}
			if pattern.Setup != nil {
				setupData = pattern.Setup(t, env)
			}

			// Cleanup
			if pattern.Cleanup != nil {
				defer pattern.Cleanup(setupData)
			}

			// Execute
			req := pattern.Execute(t, env, setupData)
			w := makeHTTPRequest(req, suite.Handler)

			// Validate status
			if w.Code != pattern.ExpectedStatus {
				t.Errorf("Expected status %d, got %d. Body: %s", pattern.ExpectedStatus, w.Code, w.Body.String())
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
	patterns []ErrorTestPattern
}

// NewTestScenarioBuilder creates a new scenario builder
func NewTestScenarioBuilder() *TestScenarioBuilder {
	return &TestScenarioBuilder{
		patterns: make([]ErrorTestPattern, 0),
	}
}

// AddInvalidIssueID adds pattern for testing invalid issue ID
func (b *TestScenarioBuilder) AddInvalidIssueID(path string) *TestScenarioBuilder {
	b.patterns = append(b.patterns, ErrorTestPattern{
		Name:           "InvalidIssueID",
		Description:    "Test handler with invalid issue ID",
		ExpectedStatus: http.StatusNotFound,
		Execute: func(t *testing.T, env *TestEnvironment, setupData interface{}) HTTPTestRequest {
			return HTTPTestRequest{
				Method: "GET",
				Path:   path,
			}
		},
	})
	return b
}

// AddNonExistentIssue adds pattern for testing non-existent issues
func (b *TestScenarioBuilder) AddNonExistentIssue(component string) *TestScenarioBuilder {
	b.patterns = append(b.patterns, ErrorTestPattern{
		Name:           "NonExistentIssue",
		Description:    "Test handler with non-existent issue ID",
		ExpectedStatus: http.StatusNotFound,
		Execute: func(t *testing.T, env *TestEnvironment, setupData interface{}) HTTPTestRequest {
			nonExistentID := fmt.Sprintf("non-existent-%d", time.Now().Unix())
			path := fmt.Sprintf("/api/v1/issues/%s/workarounds", nonExistentID)
			return HTTPTestRequest{
				Method: "GET",
				Path:   path,
			}
		},
	})
	return b
}

// AddInvalidJSON adds pattern for testing malformed JSON
func (b *TestScenarioBuilder) AddInvalidJSON(path string) *TestScenarioBuilder {
	b.patterns = append(b.patterns, ErrorTestPattern{
		Name:           "InvalidJSON",
		Description:    "Test handler with malformed JSON input",
		ExpectedStatus: http.StatusBadRequest,
		Execute: func(t *testing.T, env *TestEnvironment, setupData interface{}) HTTPTestRequest {
			return HTTPTestRequest{
				Method: "POST",
				Path:   path,
				Body:   `{"invalid": "json"`, // Malformed JSON
			}
		},
	})
	return b
}

// AddMissingRequiredFields adds pattern for testing missing required fields
func (b *TestScenarioBuilder) AddMissingRequiredFields(path string) *TestScenarioBuilder {
	b.patterns = append(b.patterns, ErrorTestPattern{
		Name:           "MissingRequiredFields",
		Description:    "Test handler with missing required fields",
		ExpectedStatus: http.StatusBadRequest,
		Execute: func(t *testing.T, env *TestEnvironment, setupData interface{}) HTTPTestRequest {
			// Empty JSON object with no required fields
			return HTTPTestRequest{
				Method: "POST",
				Path:   path,
				Body:   `{}`,
			}
		},
	})
	return b
}

// AddEmptyRequestBody adds pattern for testing empty request body
func (b *TestScenarioBuilder) AddEmptyRequestBody(path string) *TestScenarioBuilder {
	b.patterns = append(b.patterns, ErrorTestPattern{
		Name:           "EmptyRequestBody",
		Description:    "Test handler with empty request body",
		ExpectedStatus: http.StatusBadRequest,
		Execute: func(t *testing.T, env *TestEnvironment, setupData interface{}) HTTPTestRequest {
			return HTTPTestRequest{
				Method: "POST",
				Path:   path,
				Body:   "",
			}
		},
	})
	return b
}

// AddInvalidQueryParams adds pattern for testing invalid query parameters
func (b *TestScenarioBuilder) AddInvalidQueryParams(path string, params map[string]string) *TestScenarioBuilder {
	b.patterns = append(b.patterns, ErrorTestPattern{
		Name:           "InvalidQueryParams",
		Description:    "Test handler with invalid query parameters",
		ExpectedStatus: http.StatusOK, // Usually returns empty results rather than error
		Execute: func(t *testing.T, env *TestEnvironment, setupData interface{}) HTTPTestRequest {
			queryString := ""
			for key, value := range params {
				if queryString != "" {
					queryString += "&"
				}
				queryString += fmt.Sprintf("%s=%s", key, value)
			}
			fullPath := path
			if queryString != "" {
				fullPath += "?" + queryString
			}
			return HTTPTestRequest{
				Method: "GET",
				Path:   fullPath,
			}
		},
	})
	return b
}

// Build returns the built patterns
func (b *TestScenarioBuilder) Build() []ErrorTestPattern {
	return b.patterns
}

// PerformanceTestPattern defines performance testing scenarios
type PerformanceTestPattern struct {
	Name           string
	Description    string
	MaxDuration    time.Duration
	Setup          func(t *testing.T, env *TestEnvironment) interface{}
	Execute        func(t *testing.T, env *TestEnvironment, setupData interface{}) time.Duration
	Validate       func(t *testing.T, duration time.Duration, setupData interface{})
	Cleanup        func(setupData interface{})
}

// PerformanceTestSuite manages performance testing
type PerformanceTestSuite struct {
	Name     string
	Patterns []PerformanceTestPattern
}

// RunPerformanceTests executes performance tests
func (suite *PerformanceTestSuite) RunPerformanceTests(t *testing.T, env *TestEnvironment) {
	for _, pattern := range suite.Patterns {
		t.Run(fmt.Sprintf("Performance_%s", pattern.Name), func(t *testing.T) {
			// Setup
			var setupData interface{}
			if pattern.Setup != nil {
				setupData = pattern.Setup(t, env)
			}

			// Cleanup
			if pattern.Cleanup != nil {
				defer pattern.Cleanup(setupData)
			}

			// Execute and measure
			duration := pattern.Execute(t, env, setupData)

			// Validate performance
			if duration > pattern.MaxDuration {
				t.Errorf("Performance test '%s' exceeded max duration. Expected: %v, Got: %v",
					pattern.Name, pattern.MaxDuration, duration)
			}

			// Custom validation
			if pattern.Validate != nil {
				pattern.Validate(t, duration, setupData)
			}
		})
	}
}

// Common validation helpers

// validateIssuesResponse validates the structure of an issues list response
func validateIssuesResponse(data map[string]interface{}) bool {
	issues, ok := data["issues"]
	if !ok {
		return false
	}

	// Issues can be nil (empty) or an array
	if issues == nil {
		return true
	}

	// Check if issues is an array
	_, ok = issues.([]interface{})
	return ok
}

// validateWorkaroundsResponse validates the structure of a workarounds response
func validateWorkaroundsResponse(data map[string]interface{}) bool {
	workarounds, ok := data["workarounds"]
	if !ok {
		return false
	}

	// Workarounds can be nil (empty) or an array
	if workarounds == nil {
		return true
	}

	// Check if workarounds is an array
	_, ok = workarounds.([]interface{})
	return ok
}

// validateAnalysisResponse validates the structure of an analysis response
func validateAnalysisResponse(data map[string]interface{}) bool {
	required := []string{"analysis", "suggested_fix", "confidence"}
	for _, field := range required {
		if _, ok := data[field]; !ok {
			return false
		}
	}

	// Validate confidence is a number
	if confidence, ok := data["confidence"].(float64); ok {
		if confidence < 0 || confidence > 1 {
			return false
		}
	}

	return true
}

// validateComponentHealth validates component health structure
func validateComponentHealth(t *testing.T, component interface{}) bool {
	t.Helper()

	comp, ok := component.(map[string]interface{})
	if !ok {
		t.Error("Component is not a valid object")
		return false
	}

	required := []string{"component", "status", "last_check", "response_time_ms"}
	for _, field := range required {
		if _, ok := comp[field]; !ok {
			t.Errorf("Component missing required field: '%s'", field)
			return false
		}
	}

	// Validate status is one of the expected values
	if status, ok := comp["status"].(string); ok {
		validStatuses := map[string]bool{
			"healthy":   true,
			"unhealthy": true,
			"degraded":  true,
		}
		if !validStatuses[status] {
			t.Errorf("Invalid status value: '%s'", status)
			return false
		}
	}

	return true
}
