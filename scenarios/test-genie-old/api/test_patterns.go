package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// ErrorTestPattern defines a systematic approach to testing error conditions
type ErrorTestPattern struct {
	Name           string
	Description    string
	ExpectedStatus int
	RequestConfig  HTTPTestRequest
	Validate       func(t *testing.T, recorder *httptest.ResponseRecorder)
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

// AddInvalidUUID adds invalid UUID test pattern
func (b *TestScenarioBuilder) AddInvalidUUID(path string, paramName string) *TestScenarioBuilder {
	pattern := ErrorTestPattern{
		Name:           fmt.Sprintf("InvalidUUID_%s", paramName),
		Description:    fmt.Sprintf("Test with invalid UUID for parameter %s", paramName),
		ExpectedStatus: http.StatusBadRequest,
		RequestConfig: HTTPTestRequest{
			Method:    "GET",
			Path:      path,
			URLParams: map[string]string{paramName: "invalid-uuid-format"},
		},
		Validate: func(t *testing.T, recorder *httptest.ResponseRecorder) {
			assertErrorResponse(t, recorder, http.StatusBadRequest, "invalid")
		},
	}
	b.patterns = append(b.patterns, pattern)
	return b
}

// AddNonExistentResource adds non-existent resource test pattern
func (b *TestScenarioBuilder) AddNonExistentResource(path string, paramName string, resourceType string) *TestScenarioBuilder {
	pattern := ErrorTestPattern{
		Name:           fmt.Sprintf("NonExistent_%s", resourceType),
		Description:    fmt.Sprintf("Test with non-existent %s", resourceType),
		ExpectedStatus: http.StatusNotFound,
		RequestConfig: HTTPTestRequest{
			Method:    "GET",
			Path:      path,
			URLParams: map[string]string{paramName: uuid.New().String()},
		},
		Validate: func(t *testing.T, recorder *httptest.ResponseRecorder) {
			assertErrorResponse(t, recorder, http.StatusNotFound, "not found")
		},
	}
	b.patterns = append(b.patterns, pattern)
	return b
}

// AddInvalidJSON adds invalid JSON test pattern
func (b *TestScenarioBuilder) AddInvalidJSON(path string, method string) *TestScenarioBuilder {
	pattern := ErrorTestPattern{
		Name:           "InvalidJSON",
		Description:    "Test with malformed JSON input",
		ExpectedStatus: http.StatusBadRequest,
		RequestConfig: HTTPTestRequest{
			Method: method,
			Path:   path,
			Body:   `{"invalid": "json"`, // Malformed JSON
			Headers: map[string]string{
				"Content-Type": "application/json",
			},
		},
		Validate: func(t *testing.T, recorder *httptest.ResponseRecorder) {
			if recorder.Code != http.StatusBadRequest {
				t.Errorf("Expected status %d for invalid JSON, got %d", http.StatusBadRequest, recorder.Code)
			}
		},
	}
	b.patterns = append(b.patterns, pattern)
	return b
}

// AddMissingRequiredField adds missing required field test pattern
func (b *TestScenarioBuilder) AddMissingRequiredField(path string, method string, fieldName string) *TestScenarioBuilder {
	pattern := ErrorTestPattern{
		Name:           fmt.Sprintf("MissingField_%s", fieldName),
		Description:    fmt.Sprintf("Test with missing required field: %s", fieldName),
		ExpectedStatus: http.StatusBadRequest,
		RequestConfig: HTTPTestRequest{
			Method: method,
			Path:   path,
			Body:   map[string]interface{}{}, // Empty body
		},
		Validate: func(t *testing.T, recorder *httptest.ResponseRecorder) {
			assertErrorResponse(t, recorder, http.StatusBadRequest, "")
		},
	}
	b.patterns = append(b.patterns, pattern)
	return b
}

// AddCustomPattern adds a custom error test pattern
func (b *TestScenarioBuilder) AddCustomPattern(pattern ErrorTestPattern) *TestScenarioBuilder {
	b.patterns = append(b.patterns, pattern)
	return b
}

// Build returns the configured test patterns
func (b *TestScenarioBuilder) Build() []ErrorTestPattern {
	return b.patterns
}

// HandlerTestSuite provides a comprehensive test framework for HTTP handlers
type HandlerTestSuite struct {
	HandlerName string
	Handler     gin.HandlerFunc
	BaseURL     string
}

// RunErrorTests executes a suite of error condition tests
func (suite *HandlerTestSuite) RunErrorTests(t *testing.T, patterns []ErrorTestPattern) {
	for _, pattern := range patterns {
		t.Run(fmt.Sprintf("%s_%s", suite.HandlerName, pattern.Name), func(t *testing.T) {
			recorder := executeGinRequest(pattern.RequestConfig, suite.Handler)

			// Check expected status
			if recorder.Code != pattern.ExpectedStatus {
				t.Errorf("Expected status %d, got %d. Response: %s",
					pattern.ExpectedStatus, recorder.Code, recorder.Body.String())
			}

			// Run custom validation if provided
			if pattern.Validate != nil {
				pattern.Validate(t, recorder)
			}
		})
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

// RunPerformanceTest executes a performance test pattern
func RunPerformanceTest(t *testing.T, pattern PerformanceTestPattern) {
	t.Run(pattern.Name, func(t *testing.T) {
		var setupData interface{}
		if pattern.Setup != nil {
			setupData = pattern.Setup(t)
		}

		if pattern.Cleanup != nil {
			defer pattern.Cleanup(setupData)
		}

		duration := pattern.Execute(t, setupData)

		if duration > pattern.MaxDuration {
			t.Errorf("Performance test exceeded max duration: %v > %v",
				duration, pattern.MaxDuration)
		} else {
			t.Logf("Performance test completed in %v (max: %v)", duration, pattern.MaxDuration)
		}
	})
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
		var setupData interface{}
		if pattern.Setup != nil {
			setupData = pattern.Setup(t)
		}

		if pattern.Cleanup != nil {
			defer pattern.Cleanup(setupData)
		}

		results := make([]error, pattern.Iterations)
		done := make(chan struct{})
		errors := make(chan error, pattern.Iterations)

		// Execute concurrent operations
		for i := 0; i < pattern.Iterations; i++ {
			go func(iteration int) {
				err := pattern.Execute(t, setupData, iteration)
				errors <- err
				if iteration == pattern.Iterations-1 {
					close(done)
				}
			}(i)
		}

		// Collect results
		for i := 0; i < pattern.Iterations; i++ {
			results[i] = <-errors
		}

		<-done

		// Validate results
		if pattern.Validate != nil {
			pattern.Validate(t, setupData, results)
		}
	})
}

// Common test scenarios

// CreateGetHandlerTests creates standard tests for GET handlers
func CreateGetHandlerTests(handlerName string, basePath string, paramName string) []ErrorTestPattern {
	return NewTestScenarioBuilder().
		AddInvalidUUID(basePath, paramName).
		AddNonExistentResource(basePath, paramName, handlerName).
		Build()
}

// CreatePostHandlerTests creates standard tests for POST handlers
func CreatePostHandlerTests(handlerName string, basePath string, requiredFields []string) []ErrorTestPattern {
	builder := NewTestScenarioBuilder().
		AddInvalidJSON(basePath, "POST")

	for _, field := range requiredFields {
		builder.AddMissingRequiredField(basePath, "POST", field)
	}

	return builder.Build()
}

// CreatePutHandlerTests creates standard tests for PUT handlers
func CreatePutHandlerTests(handlerName string, basePath string, paramName string, requiredFields []string) []ErrorTestPattern {
	builder := NewTestScenarioBuilder().
		AddInvalidUUID(basePath, paramName).
		AddNonExistentResource(basePath, paramName, handlerName).
		AddInvalidJSON(basePath, "PUT")

	for _, field := range requiredFields {
		builder.AddMissingRequiredField(basePath, "PUT", field)
	}

	return builder.Build()
}

// CreateDeleteHandlerTests creates standard tests for DELETE handlers
func CreateDeleteHandlerTests(handlerName string, basePath string, paramName string) []ErrorTestPattern {
	return NewTestScenarioBuilder().
		AddInvalidUUID(basePath, paramName).
		AddNonExistentResource(basePath, paramName, handlerName).
		Build()
}
