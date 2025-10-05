// +build testing

package main

import (
	"database/sql"
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
			w, httpReq, err := makeHTTPRequest(*req)
			if err != nil {
				t.Fatalf("Failed to create HTTP request: %v", err)
			}

			// Execute handler
			suite.Handler(w, httpReq)

			// Validate status code
			if w.Code != pattern.ExpectedStatus {
				t.Errorf("Expected status %d, got %d. Response: %s",
					pattern.ExpectedStatus, w.Code, w.Body.String())
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

// missingRequiredFieldPattern tests handlers with missing required fields
func missingRequiredFieldPattern(method, urlPath string, body map[string]interface{}) ErrorTestPattern {
	return ErrorTestPattern{
		Name:           "MissingRequiredField",
		Description:    "Test handler with missing required fields",
		ExpectedStatus: http.StatusBadRequest,
		Execute: func(t *testing.T, setupData interface{}) *HTTPTestRequest {
			return &HTTPTestRequest{
				Method: method,
				Path:   urlPath,
				Body:   body,
			}
		},
	}
}

// invalidMethodPattern tests handlers with wrong HTTP methods
func invalidMethodPattern(urlPath string, invalidMethod string) ErrorTestPattern {
	return ErrorTestPattern{
		Name:           "InvalidMethod",
		Description:    "Test handler with invalid HTTP method",
		ExpectedStatus: http.StatusMethodNotAllowed,
		Execute: func(t *testing.T, setupData interface{}) *HTTPTestRequest {
			return &HTTPTestRequest{
				Method: invalidMethod,
				Path:   urlPath,
			}
		},
	}
}

// databaseErrorPattern tests handlers when database is unavailable
func databaseErrorPattern(method, urlPath string, body interface{}) ErrorTestPattern {
	return ErrorTestPattern{
		Name:           "DatabaseError",
		Description:    "Test handler behavior when database is unavailable",
		ExpectedStatus: http.StatusInternalServerError,
		Setup: func(t *testing.T) interface{} {
			// Store original db and set to nil to simulate database error
			originalDB := db
			db = nil
			return originalDB
		},
		Execute: func(t *testing.T, setupData interface{}) *HTTPTestRequest {
			return &HTTPTestRequest{
				Method: method,
				Path:   urlPath,
				Body:   body,
			}
		},
		Cleanup: func(setupData interface{}) {
			// Restore original db
			if setupData != nil {
				db = setupData.(*sql.DB)
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

// AddMissingRequiredField adds missing required field test pattern
func (b *TestScenarioBuilder) AddMissingRequiredField(method, urlPath string, body map[string]interface{}) *TestScenarioBuilder {
	b.patterns = append(b.patterns, missingRequiredFieldPattern(method, urlPath, body))
	return b
}

// AddInvalidMethod adds invalid method test pattern
func (b *TestScenarioBuilder) AddInvalidMethod(urlPath string, invalidMethod string) *TestScenarioBuilder {
	b.patterns = append(b.patterns, invalidMethodPattern(urlPath, invalidMethod))
	return b
}

// AddDatabaseError adds database error test pattern
func (b *TestScenarioBuilder) AddDatabaseError(method, urlPath string, body interface{}) *TestScenarioBuilder {
	b.patterns = append(b.patterns, databaseErrorPattern(method, urlPath, body))
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

// RunConcurrencyTest executes a concurrency test pattern
func RunConcurrencyTest(t *testing.T, pattern ConcurrencyTestPattern) {
	t.Run(fmt.Sprintf("Concurrency_%s", pattern.Name), func(t *testing.T) {
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
		results := make([]error, pattern.Iterations)
		done := make(chan struct{})

		for i := 0; i < pattern.Iterations; i++ {
			go func(iteration int) {
				results[iteration] = pattern.Execute(t, setupData, iteration)
				done <- struct{}{}
			}(i)
		}

		// Wait for all goroutines
		for i := 0; i < pattern.Iterations; i++ {
			<-done
		}

		// Validate results
		if pattern.Validate != nil {
			pattern.Validate(t, setupData, results)
		}
	})
}

// RunPerformanceTest executes a performance test pattern
func RunPerformanceTest(t *testing.T, pattern PerformanceTestPattern) {
	t.Run(fmt.Sprintf("Performance_%s", pattern.Name), func(t *testing.T) {
		// Setup
		var setupData interface{}
		if pattern.Setup != nil {
			setupData = pattern.Setup(t)
		}

		// Cleanup
		if pattern.Cleanup != nil {
			defer pattern.Cleanup(setupData)
		}

		// Execute and measure time
		duration := pattern.Execute(t, setupData)

		// Validate performance
		if duration > pattern.MaxDuration {
			t.Errorf("Performance test %s took %v, expected max %v",
				pattern.Name, duration, pattern.MaxDuration)
		} else {
			t.Logf("Performance test %s completed in %v (max: %v)",
				pattern.Name, duration, pattern.MaxDuration)
		}
	})
}
