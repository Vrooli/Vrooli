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

			// Custom validation
			if pattern.Validate != nil {
				pattern.Validate(t, w, setupData)
			}
		})
	}
}

// Common error patterns that can be reused across handlers

// invalidUUIDPattern tests handlers with invalid UUID formats
func invalidUUIDPattern(urlPath string, method string) ErrorTestPattern {
	return ErrorTestPattern{
		Name:           "InvalidUUID",
		Description:    "Test handler with invalid UUID format",
		ExpectedStatus: http.StatusBadRequest,
		Setup:          nil,
		Execute: func(t *testing.T, setupData interface{}) HTTPTestRequest {
			return HTTPTestRequest{
				Method:  method,
				Path:    urlPath,
				URLVars: map[string]string{"id": "invalid-uuid"},
			}
		},
		Validate: func(t *testing.T, w *httptest.ResponseRecorder, setupData interface{}) {
			assertErrorResponse(t, w, http.StatusBadRequest, "invalid")
		},
	}
}

// nonExistentChatbotPattern tests handlers with non-existent chatbot IDs
func nonExistentChatbotPattern(urlPath string, method string, db *Database) ErrorTestPattern {
	return ErrorTestPattern{
		Name:           "NonExistentChatbot",
		Description:    "Test handler with non-existent chatbot ID",
		ExpectedStatus: http.StatusNotFound,
		Setup:          nil,
		Execute: func(t *testing.T, setupData interface{}) HTTPTestRequest {
			nonExistentID := uuid.New().String()
			return HTTPTestRequest{
				Method:  method,
				Path:    urlPath,
				URLVars: map[string]string{"id": nonExistentID},
			}
		},
		Validate: func(t *testing.T, w *httptest.ResponseRecorder, setupData interface{}) {
			assertErrorResponse(t, w, http.StatusNotFound, "not found")
		},
	}
}

// invalidJSONPattern tests handlers with malformed JSON
func invalidJSONPattern(urlPath string, method string) ErrorTestPattern {
	return ErrorTestPattern{
		Name:           "InvalidJSON",
		Description:    "Test handler with malformed JSON input",
		ExpectedStatus: http.StatusBadRequest,
		Setup:          nil,
		Execute: func(t *testing.T, setupData interface{}) HTTPTestRequest {
			return HTTPTestRequest{
				Method: method,
				Path:   urlPath,
				Body:   `{"invalid": "json"`, // Malformed JSON
			}
		},
		Validate: func(t *testing.T, w *httptest.ResponseRecorder, setupData interface{}) {
			assertErrorResponse(t, w, http.StatusBadRequest, "")
		},
	}
}

// missingRequiredFieldPattern tests handlers with missing required fields
func missingRequiredFieldPattern(urlPath string, method string, fieldName string) ErrorTestPattern {
	return ErrorTestPattern{
		Name:           fmt.Sprintf("Missing_%s", fieldName),
		Description:    fmt.Sprintf("Test handler with missing required field: %s", fieldName),
		ExpectedStatus: http.StatusBadRequest,
		Setup:          nil,
		Execute: func(t *testing.T, setupData interface{}) HTTPTestRequest {
			body := map[string]interface{}{
				// Intentionally missing the required field
			}
			return HTTPTestRequest{
				Method: method,
				Path:   urlPath,
				Body:   body,
			}
		},
		Validate: func(t *testing.T, w *httptest.ResponseRecorder, setupData interface{}) {
			assertErrorResponse(t, w, http.StatusBadRequest, "")
		},
	}
}

// emptyBodyPattern tests handlers with empty request body
func emptyBodyPattern(urlPath string, method string) ErrorTestPattern {
	return ErrorTestPattern{
		Name:           "EmptyBody",
		Description:    "Test handler with empty request body",
		ExpectedStatus: http.StatusBadRequest,
		Setup:          nil,
		Execute: func(t *testing.T, setupData interface{}) HTTPTestRequest {
			return HTTPTestRequest{
				Method: method,
				Path:   urlPath,
				Body:   "",
			}
		},
	}
}

// PerformanceTestPattern defines performance testing scenarios
type PerformanceTestPattern struct {
	Name           string
	Description    string
	MaxDuration    time.Duration
	Iterations     int
	Setup          func(t *testing.T) interface{}
	Execute        func(t *testing.T, setupData interface{}, iteration int) time.Duration
	Cleanup        func(setupData interface{})
}

// RunPerformanceTests executes a suite of performance tests
func (suite *HandlerTestSuite) RunPerformanceTests(t *testing.T, patterns []PerformanceTestPattern) {
	for _, pattern := range patterns {
		t.Run(fmt.Sprintf("%s_Performance_%s", suite.HandlerName, pattern.Name), func(t *testing.T) {
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
			var totalDuration time.Duration
			for i := 0; i < pattern.Iterations; i++ {
				duration := pattern.Execute(t, setupData, i)
				totalDuration += duration
			}

			avgDuration := totalDuration / time.Duration(pattern.Iterations)

			if avgDuration > pattern.MaxDuration {
				t.Errorf("Average duration %v exceeded max duration %v", avgDuration, pattern.MaxDuration)
			}

			t.Logf("Performance: %d iterations, avg: %v, total: %v", pattern.Iterations, avgDuration, totalDuration)
		})
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

// TestScenarioBuilder provides a fluent interface for building test scenarios
type TestScenarioBuilder struct {
	patterns []ErrorTestPattern
	db       *Database
}

// NewTestScenarioBuilder creates a new test scenario builder
func NewTestScenarioBuilder(db *Database) *TestScenarioBuilder {
	return &TestScenarioBuilder{
		patterns: []ErrorTestPattern{},
		db:       db,
	}
}

// AddInvalidUUID adds invalid UUID test pattern
func (b *TestScenarioBuilder) AddInvalidUUID(urlPath string, method string) *TestScenarioBuilder {
	b.patterns = append(b.patterns, invalidUUIDPattern(urlPath, method))
	return b
}

// AddNonExistentChatbot adds non-existent chatbot test pattern
func (b *TestScenarioBuilder) AddNonExistentChatbot(urlPath string, method string) *TestScenarioBuilder {
	b.patterns = append(b.patterns, nonExistentChatbotPattern(urlPath, method, b.db))
	return b
}

// AddInvalidJSON adds invalid JSON test pattern
func (b *TestScenarioBuilder) AddInvalidJSON(urlPath string, method string) *TestScenarioBuilder {
	b.patterns = append(b.patterns, invalidJSONPattern(urlPath, method))
	return b
}

// AddMissingRequiredField adds missing required field test pattern
func (b *TestScenarioBuilder) AddMissingRequiredField(urlPath string, method string, fieldName string) *TestScenarioBuilder {
	b.patterns = append(b.patterns, missingRequiredFieldPattern(urlPath, method, fieldName))
	return b
}

// AddEmptyBody adds empty body test pattern
func (b *TestScenarioBuilder) AddEmptyBody(urlPath string, method string) *TestScenarioBuilder {
	b.patterns = append(b.patterns, emptyBodyPattern(urlPath, method))
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

// Example usage patterns and templates

// ComprehensiveHandlerTest provides a template for comprehensive handler testing
func ComprehensiveHandlerTest(t *testing.T, handlerName string, handler http.HandlerFunc, db *Database) {
	t.Run(fmt.Sprintf("%s_Comprehensive", handlerName), func(t *testing.T) {
		// 1. Setup
		loggerCleanup := setupTestLogger()
		defer loggerCleanup()

		// 2. Test successful scenarios
		t.Run("Success_Cases", func(t *testing.T) {
			// Add success test cases here
		})

		// 3. Test error conditions
		t.Run("Error_Cases", func(t *testing.T) {
			// Add error test cases here
		})

		// 4. Test edge cases
		t.Run("Edge_Cases", func(t *testing.T) {
			// Add edge case tests here
		})
	})
}
