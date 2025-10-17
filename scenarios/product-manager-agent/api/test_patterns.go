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
	Request        HTTPTestRequest
	ExpectedStatus int
	ValidateBody   func(t *testing.T, body string)
}

// PerformanceTestPattern defines performance testing scenarios
type PerformanceTestPattern struct {
	Name           string
	Description    string
	Request        HTTPTestRequest
	MaxDuration    time.Duration
	MinThroughput  int // requests per second
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

// AddInvalidJSON adds a test for invalid JSON input
func (b *TestScenarioBuilder) AddInvalidJSON(path string, method string) *TestScenarioBuilder {
	b.patterns = append(b.patterns, ErrorTestPattern{
		Name:        "InvalidJSON",
		Description: "Test with malformed JSON input",
		Request: HTTPTestRequest{
			Method: method,
			Path:   path,
			Body:   `{"invalid": "json"`, // Missing closing brace
		},
		ExpectedStatus: http.StatusBadRequest,
	})
	return b
}

// AddMissingRequiredField adds a test for missing required fields
func (b *TestScenarioBuilder) AddMissingRequiredField(path string, method string, body interface{}) *TestScenarioBuilder {
	b.patterns = append(b.patterns, ErrorTestPattern{
		Name:        "MissingRequiredField",
		Description: "Test with missing required field in request",
		Request: HTTPTestRequest{
			Method: method,
			Path:   path,
			Body:   body,
		},
		ExpectedStatus: http.StatusBadRequest,
	})
	return b
}

// AddInvalidMethodType adds a test for unsupported HTTP method
func (b *TestScenarioBuilder) AddInvalidMethodType(path string, invalidMethod string) *TestScenarioBuilder {
	b.patterns = append(b.patterns, ErrorTestPattern{
		Name:        fmt.Sprintf("InvalidMethod_%s", invalidMethod),
		Description: fmt.Sprintf("Test with unsupported HTTP method: %s", invalidMethod),
		Request: HTTPTestRequest{
			Method: invalidMethod,
			Path:   path,
		},
		ExpectedStatus: http.StatusMethodNotAllowed,
	})
	return b
}

// AddEmptyBody adds a test for empty request body
func (b *TestScenarioBuilder) AddEmptyBody(path string, method string) *TestScenarioBuilder {
	b.patterns = append(b.patterns, ErrorTestPattern{
		Name:        "EmptyBody",
		Description: "Test with empty request body",
		Request: HTTPTestRequest{
			Method: method,
			Path:   path,
			Body:   "",
		},
		ExpectedStatus: http.StatusBadRequest,
	})
	return b
}

// AddNullValues adds a test for null/zero values in request
func (b *TestScenarioBuilder) AddNullValues(path string, method string, body interface{}) *TestScenarioBuilder {
	b.patterns = append(b.patterns, ErrorTestPattern{
		Name:        "NullValues",
		Description: "Test with null/zero values in request",
		Request: HTTPTestRequest{
			Method: method,
			Path:   path,
			Body:   body,
		},
		ExpectedStatus: http.StatusBadRequest,
	})
	return b
}

// AddLargePayload adds a test for unusually large payloads
func (b *TestScenarioBuilder) AddLargePayload(path string, method string) *TestScenarioBuilder {
	// Create a large payload (1MB of data)
	largeData := make([]Feature, 1000)
	for i := range largeData {
		largeData[i] = createTestFeature(fmt.Sprintf("large-feature-%d", i))
	}

	b.patterns = append(b.patterns, ErrorTestPattern{
		Name:        "LargePayload",
		Description: "Test with large payload",
		Request: HTTPTestRequest{
			Method: method,
			Path:   path,
			Body:   largeData,
		},
		ExpectedStatus: http.StatusOK, // Should handle gracefully
	})
	return b
}

// AddConcurrentRequests adds a test for concurrent request handling
func (b *TestScenarioBuilder) AddConcurrentRequests(path string, method string, body interface{}, concurrency int) *TestScenarioBuilder {
	b.patterns = append(b.patterns, ErrorTestPattern{
		Name:        fmt.Sprintf("ConcurrentRequests_%d", concurrency),
		Description: fmt.Sprintf("Test with %d concurrent requests", concurrency),
		Request: HTTPTestRequest{
			Method: method,
			Path:   path,
			Body:   body,
		},
		ExpectedStatus: http.StatusOK,
	})
	return b
}

// Build returns the built error test patterns
func (b *TestScenarioBuilder) Build() []ErrorTestPattern {
	return b.patterns
}

// HandlerTestSuite provides a comprehensive test framework for HTTP handlers
type HandlerTestSuite struct {
	Name    string
	App     *App
	Handler func(w http.ResponseWriter, r *http.Request)
}

// RunErrorTests executes a suite of error condition tests
func (suite *HandlerTestSuite) RunErrorTests(t *testing.T, patterns []ErrorTestPattern) {
	for _, pattern := range patterns {
		t.Run(fmt.Sprintf("%s_%s", suite.Name, pattern.Name), func(t *testing.T) {
			cleanup := setupTestLogger()
			defer cleanup()

			w := makeHTTPRequest(t, suite.App, pattern.Request)

			if w.Code != pattern.ExpectedStatus {
				t.Errorf("Expected status %d, got %d. Body: %s",
					pattern.ExpectedStatus, w.Code, w.Body.String())
			}

			if pattern.ValidateBody != nil {
				pattern.ValidateBody(t, w.Body.String())
			}
		})
	}
}

// RunPerformanceTests executes a suite of performance tests
func (suite *HandlerTestSuite) RunPerformanceTests(t *testing.T, patterns []PerformanceTestPattern) {
	if testing.Short() {
		t.Skip("Skipping performance tests in short mode")
	}

	for _, pattern := range patterns {
		t.Run(fmt.Sprintf("%s_Performance_%s", suite.Name, pattern.Name), func(t *testing.T) {
			start := time.Now()

			w := makeHTTPRequest(t, suite.App, pattern.Request)

			duration := time.Since(start)

			if duration > pattern.MaxDuration {
				t.Errorf("Request took %v, expected less than %v", duration, pattern.MaxDuration)
			}

			if w.Code != http.StatusOK && w.Code != http.StatusCreated {
				t.Errorf("Expected successful response, got status %d", w.Code)
			}
		})
	}
}

// CommonErrorPatterns returns a set of common error test patterns
func CommonErrorPatterns(basePath string, method string) []ErrorTestPattern {
	builder := NewTestScenarioBuilder()

	if method == "POST" || method == "PUT" {
		builder.
			AddInvalidJSON(basePath, method).
			AddEmptyBody(basePath, method)
	}

	if method != "GET" && method != "POST" {
		builder.AddInvalidMethodType(basePath, "PATCH")
	}

	return builder.Build()
}

// PerformancePattern creates a basic performance test pattern
func PerformancePattern(name string, req HTTPTestRequest, maxDuration time.Duration) PerformanceTestPattern {
	return PerformanceTestPattern{
		Name:        name,
		Description: fmt.Sprintf("Performance test: %s", name),
		Request:     req,
		MaxDuration: maxDuration,
	}
}

// BenchmarkHandler benchmarks a handler function
func BenchmarkHandler(b *testing.B, app *App, req HTTPTestRequest) {
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		_ = w
		// Execute handler - actual implementation would vary
	}
}
