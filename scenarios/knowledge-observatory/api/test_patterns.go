package main

import (
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
	Execute        func(t *testing.T, server *Server, setupData interface{}) HTTPTestRequest
	Validate       func(t *testing.T, w interface{}, setupData interface{})
	Cleanup        func(setupData interface{})
}

// TestScenarioBuilder provides a fluent interface for building test scenarios
type TestScenarioBuilder struct {
	scenarios []ErrorTestPattern
}

// NewTestScenarioBuilder creates a new test scenario builder
func NewTestScenarioBuilder() *TestScenarioBuilder {
	return &TestScenarioBuilder{
		scenarios: []ErrorTestPattern{},
	}
}

// AddInvalidJSON adds a test for invalid JSON input
func (b *TestScenarioBuilder) AddInvalidJSON(path string) *TestScenarioBuilder {
	b.scenarios = append(b.scenarios, ErrorTestPattern{
		Name:           "InvalidJSON",
		Description:    "Test endpoint with malformed JSON input",
		ExpectedStatus: http.StatusBadRequest,
		Execute: func(t *testing.T, server *Server, setupData interface{}) HTTPTestRequest {
			return HTTPTestRequest{
				Method: "POST",
				Path:   path,
				Body:   `{"invalid": json"}`, // Malformed JSON
			}
		},
	})
	return b
}

// AddMissingQueryParam adds a test for missing required query parameters
func (b *TestScenarioBuilder) AddMissingQueryParam(path string, param string) *TestScenarioBuilder {
	b.scenarios = append(b.scenarios, ErrorTestPattern{
		Name:           fmt.Sprintf("Missing%s", param),
		Description:    fmt.Sprintf("Test endpoint with missing %s parameter", param),
		ExpectedStatus: http.StatusBadRequest,
		Execute: func(t *testing.T, server *Server, setupData interface{}) HTTPTestRequest {
			return HTTPTestRequest{
				Method: "POST",
				Path:   path,
				Body:   map[string]interface{}{}, // Empty request
			}
		},
	})
	return b
}

// AddEmptyRequest adds a test for empty request body
func (b *TestScenarioBuilder) AddEmptyRequest(path string) *TestScenarioBuilder {
	b.scenarios = append(b.scenarios, ErrorTestPattern{
		Name:           "EmptyRequest",
		Description:    "Test endpoint with empty request body",
		ExpectedStatus: http.StatusBadRequest,
		Execute: func(t *testing.T, server *Server, setupData interface{}) HTTPTestRequest {
			return HTTPTestRequest{
				Method: "POST",
				Path:   path,
				Body:   "",
			}
		},
	})
	return b
}

// AddInvalidMethod adds a test for invalid HTTP method
func (b *TestScenarioBuilder) AddInvalidMethod(path string, invalidMethod string) *TestScenarioBuilder {
	b.scenarios = append(b.scenarios, ErrorTestPattern{
		Name:           fmt.Sprintf("Invalid%sMethod", invalidMethod),
		Description:    fmt.Sprintf("Test endpoint with invalid %s method", invalidMethod),
		ExpectedStatus: http.StatusMethodNotAllowed,
		Execute: func(t *testing.T, server *Server, setupData interface{}) HTTPTestRequest {
			return HTTPTestRequest{
				Method: invalidMethod,
				Path:   path,
			}
		},
	})
	return b
}

// AddNegativeLimit adds a test for negative limit values
func (b *TestScenarioBuilder) AddNegativeLimit(path string) *TestScenarioBuilder {
	b.scenarios = append(b.scenarios, ErrorTestPattern{
		Name:           "NegativeLimit",
		Description:    "Test endpoint with negative limit value",
		ExpectedStatus: http.StatusBadRequest,
		Execute: func(t *testing.T, server *Server, setupData interface{}) HTTPTestRequest {
			return HTTPTestRequest{
				Method: "POST",
				Path:   path,
				Body: map[string]interface{}{
					"query": "test",
					"limit": -1,
				},
			}
		},
	})
	return b
}

// AddExcessiveLimit adds a test for excessively large limit values
func (b *TestScenarioBuilder) AddExcessiveLimit(path string, maxAllowed int) *TestScenarioBuilder {
	b.scenarios = append(b.scenarios, ErrorTestPattern{
		Name:           "ExcessiveLimit",
		Description:    "Test endpoint with excessive limit value",
		ExpectedStatus: http.StatusOK, // Should cap at max, not error
		Execute: func(t *testing.T, server *Server, setupData interface{}) HTTPTestRequest {
			return HTTPTestRequest{
				Method: "POST",
				Path:   path,
				Body: map[string]interface{}{
					"query": "test",
					"limit": maxAllowed * 10,
				},
			}
		},
	})
	return b
}

// Build returns all configured test scenarios
func (b *TestScenarioBuilder) Build() []ErrorTestPattern {
	return b.scenarios
}

// PerformanceTestPattern defines performance testing scenarios
type PerformanceTestPattern struct {
	Name           string
	Description    string
	MaxDuration    time.Duration
	MinThroughput  int // Requests per second
	Setup          func(t *testing.T) interface{}
	Execute        func(t *testing.T, server *Server, setupData interface{}) time.Duration
	Validate       func(t *testing.T, duration time.Duration, setupData interface{})
	Cleanup        func(setupData interface{})
}

// HandlerTestSuite provides a comprehensive test framework for HTTP handlers
type HandlerTestSuite struct {
	HandlerName    string
	Server         *Server
	BaseURL        string
	SuccessTests   []SuccessTestCase
	ErrorTests     []ErrorTestPattern
	PerformanceTests []PerformanceTestPattern
}

// SuccessTestCase defines a successful operation test case
type SuccessTestCase struct {
	Name        string
	Description string
	Setup       func(t *testing.T) interface{}
	Execute     func(t *testing.T, server *Server, setupData interface{}) HTTPTestRequest
	Validate    func(t *testing.T, w interface{}, setupData interface{})
	Cleanup     func(setupData interface{})
}

// RunSuccessTests executes all success test cases
func (suite *HandlerTestSuite) RunSuccessTests(t *testing.T) {
	for _, testCase := range suite.SuccessTests {
		t.Run(fmt.Sprintf("%s_Success_%s", suite.HandlerName, testCase.Name), func(t *testing.T) {
			var setupData interface{}
			if testCase.Setup != nil {
				setupData = testCase.Setup(t)
			}

			if testCase.Cleanup != nil {
				defer testCase.Cleanup(setupData)
			}

			req := testCase.Execute(t, suite.Server, setupData)
			w, err := makeHTTPRequest(suite.Server, req)
			if err != nil {
				t.Fatalf("Failed to execute request: %v", err)
			}

			if testCase.Validate != nil {
				testCase.Validate(t, w, setupData)
			}
		})
	}
}

// RunErrorTests executes all error test patterns
func (suite *HandlerTestSuite) RunErrorTests(t *testing.T) {
	for _, pattern := range suite.ErrorTests {
		t.Run(fmt.Sprintf("%s_Error_%s", suite.HandlerName, pattern.Name), func(t *testing.T) {
			var setupData interface{}
			if pattern.Setup != nil {
				setupData = pattern.Setup(t)
			}

			if pattern.Cleanup != nil {
				defer pattern.Cleanup(setupData)
			}

			req := pattern.Execute(t, suite.Server, setupData)
			w, err := makeHTTPRequest(suite.Server, req)
			if err != nil {
				t.Fatalf("Failed to execute request: %v", err)
			}

			assertStatusCode(t, w, pattern.ExpectedStatus)

			if pattern.Validate != nil {
				pattern.Validate(t, w, setupData)
			}
		})
	}
}

// RunPerformanceTests executes all performance test patterns
func (suite *HandlerTestSuite) RunPerformanceTests(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance tests in short mode")
	}

	for _, pattern := range suite.PerformanceTests {
		t.Run(fmt.Sprintf("%s_Performance_%s", suite.HandlerName, pattern.Name), func(t *testing.T) {
			var setupData interface{}
			if pattern.Setup != nil {
				setupData = pattern.Setup(t)
			}

			if pattern.Cleanup != nil {
				defer pattern.Cleanup(setupData)
			}

			duration := pattern.Execute(t, suite.Server, setupData)

			if duration > pattern.MaxDuration {
				t.Errorf("%s took %v, expected less than %v", pattern.Name, duration, pattern.MaxDuration)
			}

			if pattern.Validate != nil {
				pattern.Validate(t, duration, setupData)
			}
		})
	}
}

// RunAllTests executes all test suites (success, error, and performance)
func (suite *HandlerTestSuite) RunAllTests(t *testing.T) {
	t.Run("SuccessTests", func(t *testing.T) {
		suite.RunSuccessTests(t)
	})

	t.Run("ErrorTests", func(t *testing.T) {
		suite.RunErrorTests(t)
	})

	if !testing.Short() {
		t.Run("PerformanceTests", func(t *testing.T) {
			suite.RunPerformanceTests(t)
		})
	}
}

// Common test patterns for reuse

// CreateSearchErrorPatterns creates standard error patterns for search endpoint
func CreateSearchErrorPatterns() []ErrorTestPattern {
	builder := NewTestScenarioBuilder()
	return builder.
		AddInvalidJSON("/api/v1/knowledge/search").
		AddMissingQueryParam("/api/v1/knowledge/search", "query").
		AddEmptyRequest("/api/v1/knowledge/search").
		Build()
}

// CreateGraphErrorPatterns creates standard error patterns for graph endpoint
func CreateGraphErrorPatterns() []ErrorTestPattern {
	return []ErrorTestPattern{
		{
			Name:           "InvalidDepth",
			Description:    "Test graph endpoint with invalid depth parameter",
			ExpectedStatus: http.StatusOK, // Should use default
			Execute: func(t *testing.T, server *Server, setupData interface{}) HTTPTestRequest {
				return HTTPTestRequest{
					Method: "POST",
					Path:   "/api/v1/knowledge/graph",
					Body: map[string]interface{}{
						"depth": -1,
					},
				}
			},
		},
	}
}

// CreateMetricsErrorPatterns creates standard error patterns for metrics endpoint
func CreateMetricsErrorPatterns() []ErrorTestPattern {
	return []ErrorTestPattern{
		{
			Name:           "InvalidTimeRange",
			Description:    "Test metrics endpoint with invalid time range",
			ExpectedStatus: http.StatusOK, // Should use default
			Execute: func(t *testing.T, server *Server, setupData interface{}) HTTPTestRequest {
				return HTTPTestRequest{
					Method: "POST",
					Path:   "/api/v1/knowledge/metrics",
					Body: map[string]interface{}{
						"time_range": "invalid",
					},
				}
			},
		},
	}
}
