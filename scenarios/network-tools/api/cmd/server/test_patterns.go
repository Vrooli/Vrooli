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
			w, err := makeHTTPRequest(suite.Handler, *req)
			if err != nil {
				t.Fatalf("Failed to create HTTP request: %v", err)
			}

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

// Common error patterns that can be reused across handlers

// emptyBodyPattern tests handlers with empty request body
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
				Body:   "",
			}
		},
	}
}

// invalidJSONPattern tests handlers with malformed JSON
func invalidJSONPattern(urlPath string) ErrorTestPattern {
	return ErrorTestPattern{
		Name:           "InvalidJSON",
		Description:    "Test handler with malformed JSON input",
		ExpectedStatus: http.StatusBadRequest,
		Setup:          nil,
		Execute: func(t *testing.T, setupData interface{}) *HTTPTestRequest {
			return &HTTPTestRequest{
				Method: "POST",
				Path:   urlPath,
				Body:   `{"invalid": "json"`, // Malformed JSON
			}
		},
	}
}

// missingRequiredFieldPattern tests handlers with missing required fields
func missingRequiredFieldPattern(urlPath, fieldName string) ErrorTestPattern {
	return ErrorTestPattern{
		Name:           fmt.Sprintf("Missing_%s", fieldName),
		Description:    fmt.Sprintf("Test handler with missing %s field", fieldName),
		ExpectedStatus: http.StatusBadRequest,
		Setup:          nil,
		Execute: func(t *testing.T, setupData interface{}) *HTTPTestRequest {
			return &HTTPTestRequest{
				Method: "POST",
				Path:   urlPath,
				Body:   map[string]interface{}{}, // Empty object
			}
		},
		Validate: func(t *testing.T, w *httptest.ResponseRecorder, setupData interface{}) {
			// Expect error message to contain "is required" or the field name
			assertErrorResponse(t, w, http.StatusBadRequest, "required")
		},
	}
}

// invalidURLPattern tests HTTP request handler with invalid URL
func invalidURLPattern(urlPath string) ErrorTestPattern {
	return ErrorTestPattern{
		Name:           "InvalidURL",
		Description:    "Test HTTP handler with invalid URL",
		ExpectedStatus: http.StatusBadRequest,
		Setup:          nil,
		Execute: func(t *testing.T, setupData interface{}) *HTTPTestRequest {
			return &HTTPTestRequest{
				Method: "POST",
				Path:   urlPath,
				Body: map[string]interface{}{
					"url":    "not-a-valid-url",
					"method": "GET",
				},
			}
		},
	}
}

// unsupportedRecordTypePattern tests DNS handler with unsupported record type
func unsupportedRecordTypePattern(urlPath string) ErrorTestPattern {
	return ErrorTestPattern{
		Name:           "UnsupportedRecordType",
		Description:    "Test DNS handler with unsupported record type",
		ExpectedStatus: http.StatusBadRequest,
		Setup:          nil,
		Execute: func(t *testing.T, setupData interface{}) *HTTPTestRequest {
			return &HTTPTestRequest{
				Method: "POST",
				Path:   urlPath,
				Body: map[string]interface{}{
					"query":       "example.com",
					"record_type": "UNSUPPORTED",
				},
			}
		},
	}
}

// invalidPortRangePattern tests port scan with invalid port range
func invalidPortRangePattern(urlPath string) ErrorTestPattern {
	return ErrorTestPattern{
		Name:           "InvalidPortRange",
		Description:    "Test port scan with invalid port range",
		ExpectedStatus: http.StatusBadRequest,
		Setup:          nil,
		Execute: func(t *testing.T, setupData interface{}) *HTTPTestRequest {
			return &HTTPTestRequest{
				Method: "POST",
				Path:   urlPath,
				Body: map[string]interface{}{
					"target": "localhost",
					"ports":  []int{99999}, // Invalid port
				},
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

// RunConcurrencyTest executes a concurrency test pattern
func RunConcurrencyTest(t *testing.T, pattern ConcurrencyTestPattern) {
	t.Run(pattern.Name, func(t *testing.T) {
		// Setup
		var setupData interface{}
		if pattern.Setup != nil {
			setupData = pattern.Setup(t)
		}

		// Cleanup
		if pattern.Cleanup != nil {
			defer pattern.Cleanup(setupData)
		}

		// Execute concurrently
		results := make([]error, pattern.Concurrency*pattern.Iterations)
		done := make(chan struct{})

		for i := 0; i < pattern.Concurrency; i++ {
			go func(workerID int) {
				for j := 0; j < pattern.Iterations; j++ {
					iteration := workerID*pattern.Iterations + j
					results[iteration] = pattern.Execute(t, setupData, iteration)
				}
				done <- struct{}{}
			}(i)
		}

		// Wait for all workers
		for i := 0; i < pattern.Concurrency; i++ {
			<-done
		}

		// Validate
		if pattern.Validate != nil {
			pattern.Validate(t, setupData, results)
		}
	})
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

// AddEmptyBody adds empty body test pattern
func (b *TestScenarioBuilder) AddEmptyBody(urlPath string) *TestScenarioBuilder {
	b.patterns = append(b.patterns, emptyBodyPattern(urlPath))
	return b
}

// AddInvalidJSON adds invalid JSON test pattern
func (b *TestScenarioBuilder) AddInvalidJSON(urlPath string) *TestScenarioBuilder {
	b.patterns = append(b.patterns, invalidJSONPattern(urlPath))
	return b
}

// AddMissingField adds missing required field test pattern
func (b *TestScenarioBuilder) AddMissingField(urlPath, fieldName string) *TestScenarioBuilder {
	b.patterns = append(b.patterns, missingRequiredFieldPattern(urlPath, fieldName))
	return b
}

// AddInvalidURL adds invalid URL test pattern
func (b *TestScenarioBuilder) AddInvalidURL(urlPath string) *TestScenarioBuilder {
	b.patterns = append(b.patterns, invalidURLPattern(urlPath))
	return b
}

// AddUnsupportedRecordType adds unsupported DNS record type pattern
func (b *TestScenarioBuilder) AddUnsupportedRecordType(urlPath string) *TestScenarioBuilder {
	b.patterns = append(b.patterns, unsupportedRecordTypePattern(urlPath))
	return b
}

// AddInvalidPortRange adds invalid port range pattern
func (b *TestScenarioBuilder) AddInvalidPortRange(urlPath string) *TestScenarioBuilder {
	b.patterns = append(b.patterns, invalidPortRangePattern(urlPath))
	return b
}

// AddCustom adds a custom test pattern
func (b *TestScenarioBuilder) AddCustom(pattern ErrorTestPattern) *TestScenarioBuilder {
	b.patterns = append(b.patterns, pattern)
	return b
}

// Build returns the built test patterns
func (b *TestScenarioBuilder) Build() []ErrorTestPattern {
	return b.patterns
}
