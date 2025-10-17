package main

import (
	"fmt"
	"net/http"
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
	Validate       func(t *testing.T, w interface{}, setupData interface{})
	Cleanup        func(setupData interface{})
}

// HandlerTestSuite provides a comprehensive test framework for HTTP handlers
type HandlerTestSuite struct {
	HandlerName string
	Service     *BookTalkService
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
			w, err := makeHTTPRequest(*req)
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

// invalidUUIDPattern tests handlers with invalid UUID formats
func invalidUUIDPattern(urlPath, varName string) ErrorTestPattern {
	return ErrorTestPattern{
		Name:           "InvalidUUID",
		Description:    "Test handler with invalid UUID format",
		ExpectedStatus: http.StatusBadRequest,
		Setup:          nil,
		Execute: func(t *testing.T, setupData interface{}) *HTTPTestRequest {
			return &HTTPTestRequest{
				Method:  "GET",
				Path:    urlPath,
				URLVars: map[string]string{varName: "invalid-uuid-format"},
			}
		},
	}
}

// nonExistentBookPattern tests handlers with non-existent book IDs
func nonExistentBookPattern(method, urlPath string) ErrorTestPattern {
	return ErrorTestPattern{
		Name:           "NonExistentBook",
		Description:    "Test handler with non-existent book ID",
		ExpectedStatus: http.StatusNotFound,
		Setup:          nil,
		Execute: func(t *testing.T, setupData interface{}) *HTTPTestRequest {
			nonExistentID := uuid.New()
			return &HTTPTestRequest{
				Method:  method,
				Path:    urlPath,
				URLVars: map[string]string{"book_id": nonExistentID.String()},
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
				Headers: map[string]string{
					"Content-Type": "application/json",
				},
			}
		},
	}
}

// missingRequiredFieldsPattern tests handlers with missing required fields
func missingRequiredFieldsPattern(urlPath, missingField string) ErrorTestPattern {
	return ErrorTestPattern{
		Name:           fmt.Sprintf("Missing_%s", missingField),
		Description:    fmt.Sprintf("Test handler with missing required field: %s", missingField),
		ExpectedStatus: http.StatusBadRequest,
		Setup:          nil,
		Execute: func(t *testing.T, setupData interface{}) *HTTPTestRequest {
			body := make(map[string]interface{})
			// Intentionally leave out the required field
			return &HTTPTestRequest{
				Method: "POST",
				Path:   urlPath,
				Body:   body,
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
		Setup:          nil,
		Execute: func(t *testing.T, setupData interface{}) *HTTPTestRequest {
			return &HTTPTestRequest{
				Method: method,
				Path:   urlPath,
				Body:   "",
			}
		},
	}
}

// unsupportedFileTypePattern tests file upload with unsupported file type
func unsupportedFileTypePattern() ErrorTestPattern {
	return ErrorTestPattern{
		Name:           "UnsupportedFileType",
		Description:    "Test file upload with unsupported file type",
		ExpectedStatus: http.StatusBadRequest,
		Setup:          nil,
		Execute: func(t *testing.T, setupData interface{}) *HTTPTestRequest {
			return &HTTPTestRequest{
				Method: "POST",
				Path:   "/api/v1/books/upload",
				// This would need special handling for file upload
			}
		},
	}
}

// bookNotProcessedPattern tests chat with unprocessed book
func bookNotProcessedPattern(bookID uuid.UUID) ErrorTestPattern {
	return ErrorTestPattern{
		Name:           "BookNotProcessed",
		Description:    "Test chat with book that hasn't finished processing",
		ExpectedStatus: http.StatusConflict,
		Setup:          nil,
		Execute: func(t *testing.T, setupData interface{}) *HTTPTestRequest {
			return &HTTPTestRequest{
				Method:  "POST",
				Path:    "/api/v1/books/{book_id}/chat",
				URLVars: map[string]string{"book_id": bookID.String()},
				Body: map[string]interface{}{
					"message":          "Test message",
					"user_id":          "test-user",
					"current_position": 10,
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
func (b *TestScenarioBuilder) AddInvalidUUID(urlPath, varName string) *TestScenarioBuilder {
	b.patterns = append(b.patterns, invalidUUIDPattern(urlPath, varName))
	return b
}

// AddNonExistentBook adds non-existent book test pattern
func (b *TestScenarioBuilder) AddNonExistentBook(method, urlPath string) *TestScenarioBuilder {
	b.patterns = append(b.patterns, nonExistentBookPattern(method, urlPath))
	return b
}

// AddInvalidJSON adds invalid JSON test pattern
func (b *TestScenarioBuilder) AddInvalidJSON(urlPath string) *TestScenarioBuilder {
	b.patterns = append(b.patterns, invalidJSONPattern(urlPath))
	return b
}

// AddMissingRequiredField adds missing required field test pattern
func (b *TestScenarioBuilder) AddMissingRequiredField(urlPath, fieldName string) *TestScenarioBuilder {
	b.patterns = append(b.patterns, missingRequiredFieldsPattern(urlPath, fieldName))
	return b
}

// AddEmptyBody adds empty body test pattern
func (b *TestScenarioBuilder) AddEmptyBody(method, urlPath string) *TestScenarioBuilder {
	b.patterns = append(b.patterns, emptyBodyPattern(method, urlPath))
	return b
}

// AddCustom adds a custom test pattern
func (b *TestScenarioBuilder) AddCustom(pattern ErrorTestPattern) *TestScenarioBuilder {
	b.patterns = append(b.patterns, pattern)
	return b
}

// Build returns the built patterns
func (b *TestScenarioBuilder) Build() []ErrorTestPattern {
	return b.patterns
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
			t.Errorf("Performance test '%s' exceeded max duration: %v > %v",
				pattern.Name, duration, pattern.MaxDuration)
		} else {
			t.Logf("Performance test '%s' completed in %v (max: %v)",
				pattern.Name, duration, pattern.MaxDuration)
		}
	})
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
		done := make(chan struct{}, pattern.Concurrency)

		for i := 0; i < pattern.Iterations; i++ {
			go func(iteration int) {
				results[iteration] = pattern.Execute(t, setupData, iteration)
				done <- struct{}{}
			}(i)

			// Limit concurrency
			if (i+1)%pattern.Concurrency == 0 {
				for j := 0; j < pattern.Concurrency; j++ {
					<-done
				}
			}
		}

		// Wait for remaining goroutines
		remaining := pattern.Iterations % pattern.Concurrency
		for i := 0; i < remaining; i++ {
			<-done
		}

		if pattern.Validate != nil {
			pattern.Validate(t, setupData, results)
		}
	})
}
