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
	Execute        func(t *testing.T, setupData interface{}) HTTPTestRequest
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
			w, httpReq, err := makeHTTPRequest(req)
			if err != nil {
				t.Fatalf("Failed to create HTTP request: %v", err)
			}

			// Execute handler
			suite.Handler(w, httpReq)

			// Validate status code
			if w.Code != pattern.ExpectedStatus {
				t.Errorf("Expected status %d, got %d. Response: %s", pattern.ExpectedStatus, w.Code, w.Body.String())
			}

			// Additional validation
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

// NewTestScenarioBuilder creates a new test scenario builder
func NewTestScenarioBuilder() *TestScenarioBuilder {
	return &TestScenarioBuilder{
		patterns: []ErrorTestPattern{},
	}
}

// AddInvalidJSON adds invalid JSON test pattern
func (b *TestScenarioBuilder) AddInvalidJSON(method, urlPath string) *TestScenarioBuilder {
	b.patterns = append(b.patterns, ErrorTestPattern{
		Name:           "InvalidJSON",
		Description:    "Test handler with malformed JSON input",
		ExpectedStatus: http.StatusBadRequest,
		Execute: func(t *testing.T, setupData interface{}) HTTPTestRequest {
			return HTTPTestRequest{
				Method: method,
				Path:   urlPath,
				Body:   `{"invalid": "json"`, // Malformed JSON
			}
		},
	})
	return b
}

// AddInvalidUUID adds invalid UUID test pattern
func (b *TestScenarioBuilder) AddInvalidUUID(method, urlPath string) *TestScenarioBuilder {
	b.patterns = append(b.patterns, ErrorTestPattern{
		Name:           "InvalidUUID",
		Description:    "Test handler with invalid UUID format",
		ExpectedStatus: http.StatusBadRequest,
		Execute: func(t *testing.T, setupData interface{}) HTTPTestRequest {
			return HTTPTestRequest{
				Method:  method,
				Path:    urlPath,
				URLVars: map[string]string{"id": "invalid-uuid"},
			}
		},
	})
	return b
}

// AddNonExistentProgram adds non-existent program test pattern
func (b *TestScenarioBuilder) AddNonExistentProgram(method, urlPath string) *TestScenarioBuilder {
	b.patterns = append(b.patterns, ErrorTestPattern{
		Name:           "NonExistentProgram",
		Description:    "Test handler with non-existent program ID",
		ExpectedStatus: http.StatusNotFound,
		Execute: func(t *testing.T, setupData interface{}) HTTPTestRequest {
			nonExistentID := uuid.New().String()
			return HTTPTestRequest{
				Method:  method,
				Path:    urlPath,
				URLVars: map[string]string{"id": nonExistentID},
			}
		},
	})
	return b
}

// AddMissingRequiredField adds missing required field test pattern
func (b *TestScenarioBuilder) AddMissingRequiredField(method, urlPath, fieldName string) *TestScenarioBuilder {
	b.patterns = append(b.patterns, ErrorTestPattern{
		Name:           fmt.Sprintf("Missing_%s", fieldName),
		Description:    fmt.Sprintf("Test handler with missing %s field", fieldName),
		ExpectedStatus: http.StatusBadRequest,
		Execute: func(t *testing.T, setupData interface{}) HTTPTestRequest {
			// Create incomplete request body
			body := map[string]interface{}{}
			return HTTPTestRequest{
				Method: method,
				Path:   urlPath,
				Body:   body,
			}
		},
	})
	return b
}

// AddInvalidMode adds invalid mode test pattern (for analyze endpoint)
func (b *TestScenarioBuilder) AddInvalidMode(urlPath string) *TestScenarioBuilder {
	b.patterns = append(b.patterns, ErrorTestPattern{
		Name:           "InvalidMode",
		Description:    "Test analyze handler with invalid mode",
		ExpectedStatus: http.StatusBadRequest,
		Execute: func(t *testing.T, setupData interface{}) HTTPTestRequest {
			return HTTPTestRequest{
				Method: "POST",
				Path:   urlPath,
				Body: AnalysisRequest{
					Mode:         "invalid_mode",
					ScenarioPath: "/test/path",
				},
			}
		},
	})
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
	Name        string
	Description string
	MaxDuration time.Duration
	Setup       func(t *testing.T) interface{}
	Execute     func(t *testing.T, setupData interface{}) time.Duration
	Cleanup     func(setupData interface{})
}

// RunPerformanceTest executes a performance test
func RunPerformanceTest(t *testing.T, pattern PerformanceTestPattern) {
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

		// Execute and measure
		duration := pattern.Execute(t, setupData)

		// Validate performance
		if duration > pattern.MaxDuration {
			t.Errorf("Performance test '%s' took %v, expected max %v", pattern.Name, duration, pattern.MaxDuration)
		} else {
			t.Logf("Performance test '%s' completed in %v (max: %v)", pattern.Name, duration, pattern.MaxDuration)
		}
	})
}

// Common test patterns

// invalidAnalysisRequestPattern creates pattern for invalid analysis requests
func invalidAnalysisRequestPattern() ErrorTestPattern {
	return ErrorTestPattern{
		Name:           "InvalidAnalysisRequest",
		Description:    "Test analyze handler with missing required fields",
		ExpectedStatus: http.StatusBadRequest,
		Execute: func(t *testing.T, setupData interface{}) HTTPTestRequest {
			return HTTPTestRequest{
				Method: "POST",
				Path:   "/api/v1/referral/analyze",
				Body: AnalysisRequest{
					Mode: "local",
					// Missing ScenarioPath for local mode
				},
			}
		},
	}
}

// invalidGenerateRequestPattern creates pattern for invalid generate requests
func invalidGenerateRequestPattern() ErrorTestPattern {
	return ErrorTestPattern{
		Name:           "InvalidGenerateRequest",
		Description:    "Test generate handler with invalid data",
		ExpectedStatus: http.StatusBadRequest,
		Execute: func(t *testing.T, setupData interface{}) HTTPTestRequest {
			return HTTPTestRequest{
				Method: "POST",
				Path:   "/api/v1/referral/generate",
				Body:   map[string]interface{}{}, // Empty body
			}
		},
	}
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

// RunConcurrencyTest executes a concurrency test
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
		results := make([]error, pattern.Iterations)
		done := make(chan bool, pattern.Concurrency)

		for i := 0; i < pattern.Iterations; i++ {
			iteration := i
			go func() {
				results[iteration] = pattern.Execute(t, setupData, iteration)
				done <- true
			}()

			// Limit concurrency
			if (i+1)%pattern.Concurrency == 0 {
				for j := 0; j < pattern.Concurrency; j++ {
					<-done
				}
			}
		}

		// Wait for remaining
		remaining := pattern.Iterations % pattern.Concurrency
		for i := 0; i < remaining; i++ {
			<-done
		}

		// Validate
		if pattern.Validate != nil {
			pattern.Validate(t, setupData, results)
		}
	})
}

// Example usage patterns

// ExampleErrorTestUsage demonstrates how to use error test patterns
func ExampleErrorTestUsage(t *testing.T) {
	patterns := NewTestScenarioBuilder().
		AddInvalidJSON("POST", "/api/v1/referral/analyze").
		AddInvalidMode("/api/v1/referral/analyze").
		AddCustom(invalidAnalysisRequestPattern()).
		Build()

	suite := &HandlerTestSuite{
		HandlerName: "analyzeHandler",
		Handler:     analyzeHandler,
		BaseURL:     "/api/v1/referral/analyze",
	}

	suite.RunErrorTests(t, patterns)
}
