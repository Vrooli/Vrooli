
package main

import (
	"net/http/httptest"
	"fmt"
	"net/http"
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
			w, httpReq := makeHTTPRequest(*req)

			suite.Handler(w, httpReq)

			// Validate status
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

// Common error patterns that can be reused across handlers

// jobNotFoundPattern tests handlers with non-existent job IDs
func jobNotFoundPattern(urlPath string) ErrorTestPattern {
	return ErrorTestPattern{
		Name:           "JobNotFound",
		Description:    "Test handler with non-existent job ID",
		ExpectedStatus: http.StatusNotFound,
		Setup:          nil,
		Execute: func(t *testing.T, setupData interface{}) *HTTPTestRequest {
			nonExistentID := "NON-EXISTENT-JOB"
			return &HTTPTestRequest{
				Method:  "GET",
				Path:    urlPath,
				URLVars: map[string]string{"id": nonExistentID},
			}
		},
	}
}

// invalidJSONPattern tests handlers with malformed JSON
func invalidJSONPattern(method, urlPath string) ErrorTestPattern {
	return ErrorTestPattern{
		Name:           "InvalidJSON",
		Description:    "Test handler with malformed JSON input",
		ExpectedStatus: http.StatusBadRequest,
		Setup: func(t *testing.T) interface{} {
			env := setupTestDirectory(t)
			return env
		},
		Execute: func(t *testing.T, setupData interface{}) *HTTPTestRequest {
			return &HTTPTestRequest{
				Method: method,
				Path:   urlPath,
				Body:   `{"invalid": "json"`, // Malformed JSON
			}
		},
		Cleanup: func(setupData interface{}) {
			if env, ok := setupData.(*TestEnvironment); ok {
				env.Cleanup()
			}
		},
	}
}

// invalidSourcePattern tests import handler with invalid source
func invalidSourcePattern() ErrorTestPattern {
	return ErrorTestPattern{
		Name:           "InvalidSource",
		Description:    "Test import with invalid source type",
		ExpectedStatus: http.StatusBadRequest,
		Setup: func(t *testing.T) interface{} {
			return setupTestDirectory(t)
		},
		Execute: func(t *testing.T, setupData interface{}) *HTTPTestRequest {
			return &HTTPTestRequest{
				Method: "POST",
				Path:   "/api/v1/jobs/import",
				Body: ImportRequest{
					Source: "invalid-source",
					Data:   "test data",
				},
			}
		},
		Cleanup: func(setupData interface{}) {
			if env, ok := setupData.(*TestEnvironment); ok {
				env.Cleanup()
			}
		},
	}
}

// invalidStateTransitionPattern tests state transition with wrong current state
func invalidStateTransitionPattern(fromState, toState string) ErrorTestPattern {
	return ErrorTestPattern{
		Name:           fmt.Sprintf("InvalidStateTransition_%s_to_%s", fromState, toState),
		Description:    "Test invalid state transition",
		ExpectedStatus: http.StatusBadRequest,
		Setup: func(t *testing.T) interface{} {
			env := setupTestDirectory(t)
			job := setupTestJob(t, fromState)
			return map[string]interface{}{
				"env": env,
				"job": job,
			}
		},
		Execute: func(t *testing.T, setupData interface{}) *HTTPTestRequest {
			data := setupData.(map[string]interface{})
			job := data["job"].(*TestJob)

			var path string
			if toState == "approved" {
				path = fmt.Sprintf("/api/v1/jobs/%s/approve", job.Job.ID)
			} else if toState == "building" {
				path = fmt.Sprintf("/api/v1/jobs/%s/proposal", job.Job.ID)
			}

			return &HTTPTestRequest{
				Method:  "POST",
				Path:    path,
				URLVars: map[string]string{"id": job.Job.ID},
			}
		},
		Cleanup: func(setupData interface{}) {
			if setupData != nil {
				data := setupData.(map[string]interface{})
				if env, ok := data["env"].(*TestEnvironment); ok {
					env.Cleanup()
				}
				if job, ok := data["job"].(*TestJob); ok {
					job.Cleanup()
				}
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
	Name           string
	Description    string
	Concurrency    int
	Iterations     int
	Setup          func(t *testing.T) interface{}
	Execute        func(t *testing.T, setupData interface{}, iteration int) error
	Validate       func(t *testing.T, setupData interface{}, results []error)
	Cleanup        func(setupData interface{})
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

// AddJobNotFound adds job not found test pattern
func (b *TestScenarioBuilder) AddJobNotFound(urlPath string) *TestScenarioBuilder {
	b.patterns = append(b.patterns, jobNotFoundPattern(urlPath))
	return b
}

// AddInvalidJSON adds invalid JSON test pattern
func (b *TestScenarioBuilder) AddInvalidJSON(method, urlPath string) *TestScenarioBuilder {
	b.patterns = append(b.patterns, invalidJSONPattern(method, urlPath))
	return b
}

// AddInvalidSource adds invalid source test pattern
func (b *TestScenarioBuilder) AddInvalidSource() *TestScenarioBuilder {
	b.patterns = append(b.patterns, invalidSourcePattern())
	return b
}

// AddInvalidStateTransition adds invalid state transition test pattern
func (b *TestScenarioBuilder) AddInvalidStateTransition(fromState, toState string) *TestScenarioBuilder {
	b.patterns = append(b.patterns, invalidStateTransitionPattern(fromState, toState))
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
