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

			// Validate status
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

// invalidUUIDPattern tests handlers with invalid UUID formats
func invalidUUIDPattern(method, urlPath string) ErrorTestPattern {
	return ErrorTestPattern{
		Name:           "InvalidUUID",
		Description:    "Test handler with invalid UUID format",
		ExpectedStatus: http.StatusBadRequest,
		Setup:          nil,
		Execute: func(t *testing.T, setupData interface{}) *HTTPTestRequest {
			return &HTTPTestRequest{
				Method:  method,
				Path:    urlPath,
				URLVars: map[string]string{"id": "invalid-uuid"},
			}
		},
	}
}

// nonExistentVisitorPattern tests handlers with non-existent visitor IDs
func nonExistentVisitorPattern(method, urlPath string) ErrorTestPattern {
	return ErrorTestPattern{
		Name:           "NonExistentVisitor",
		Description:    "Test handler with non-existent visitor ID",
		ExpectedStatus: http.StatusNotFound,
		Setup:          nil,
		Execute: func(t *testing.T, setupData interface{}) *HTTPTestRequest {
			nonExistentID := uuid.New()
			return &HTTPTestRequest{
				Method:  method,
				Path:    urlPath,
				URLVars: map[string]string{"id": nonExistentID.String()},
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
		Setup:          nil,
		Execute: func(t *testing.T, setupData interface{}) *HTTPTestRequest {
			return &HTTPTestRequest{
				Method: method,
				Path:   urlPath,
				Body:   `{"invalid": "json"`, // Malformed JSON
			}
		},
	}
}

// missingRequiredFieldsPattern tests handlers with missing required fields
func missingRequiredFieldsPattern(urlPath string) ErrorTestPattern {
	return ErrorTestPattern{
		Name:           "MissingRequiredFields",
		Description:    "Test handler with missing required fields",
		ExpectedStatus: http.StatusBadRequest,
		Setup:          nil,
		Execute: func(t *testing.T, setupData interface{}) *HTTPTestRequest {
			return &HTTPTestRequest{
				Method: "POST",
				Path:   urlPath,
				Body: map[string]interface{}{
					"incomplete": "data",
				},
			}
		},
	}
}

// invalidMethodPattern tests handlers with wrong HTTP method
func invalidMethodPattern(invalidMethod, urlPath string) ErrorTestPattern {
	return ErrorTestPattern{
		Name:           "InvalidMethod",
		Description:    fmt.Sprintf("Test handler with invalid method %s", invalidMethod),
		ExpectedStatus: http.StatusMethodNotAllowed,
		Setup:          nil,
		Execute: func(t *testing.T, setupData interface{}) *HTTPTestRequest {
			return &HTTPTestRequest{
				Method: invalidMethod,
				Path:   urlPath,
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

// RunPerformanceTest executes a performance test and validates duration
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

		// Execute and measure
		duration := pattern.Execute(t, setupData)

		// Validate duration
		if duration > pattern.MaxDuration {
			t.Errorf("Performance test exceeded max duration. Expected < %v, got %v",
				pattern.MaxDuration, duration)
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
func (b *TestScenarioBuilder) AddInvalidUUID(method, urlPath string) *TestScenarioBuilder {
	b.patterns = append(b.patterns, invalidUUIDPattern(method, urlPath))
	return b
}

// AddNonExistentVisitor adds non-existent visitor test pattern
func (b *TestScenarioBuilder) AddNonExistentVisitor(method, urlPath string) *TestScenarioBuilder {
	b.patterns = append(b.patterns, nonExistentVisitorPattern(method, urlPath))
	return b
}

// AddInvalidJSON adds invalid JSON test pattern
func (b *TestScenarioBuilder) AddInvalidJSON(method, urlPath string) *TestScenarioBuilder {
	b.patterns = append(b.patterns, invalidJSONPattern(method, urlPath))
	return b
}

// AddMissingRequiredFields adds missing required fields test pattern
func (b *TestScenarioBuilder) AddMissingRequiredFields(urlPath string) *TestScenarioBuilder {
	b.patterns = append(b.patterns, missingRequiredFieldsPattern(urlPath))
	return b
}

// AddInvalidMethod adds invalid method test pattern
func (b *TestScenarioBuilder) AddInvalidMethod(invalidMethod, urlPath string) *TestScenarioBuilder {
	b.patterns = append(b.patterns, invalidMethodPattern(invalidMethod, urlPath))
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

// Example usage patterns

// ExampleHandlerTest demonstrates how to use the testing framework
func ExampleHandlerTest(t *testing.T) {
	// Setup test environment
	loggerCleanup := setupTestLogger()
	defer loggerCleanup()

	env := setupTestDatabase(t)
	if env == nil {
		t.Skip("Test database not available")
		return
	}
	defer env.Cleanup()

	// Create test visitor
	visitor := setupTestVisitor(t, "test-fingerprint-123")
	defer visitor.Cleanup()

	// Test successful case
	t.Run("Success", func(t *testing.T) {
		w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
			Method:  "GET",
			Path:    fmt.Sprintf("/api/v1/visitor/%s", visitor.Visitor.ID),
			URLVars: map[string]string{"id": visitor.Visitor.ID},
		})
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		getVisitorHandler(w, httpReq)

		response := assertJSONResponse(t, w, http.StatusOK, map[string]interface{}{
			"id":          visitor.Visitor.ID,
			"fingerprint": visitor.Visitor.Fingerprint,
		})

		if response != nil {
			if _, ok := response["first_seen"]; !ok {
				t.Error("Expected first_seen in response")
			}
		}
	})

	// Test error conditions using patterns
	suite := &HandlerTestSuite{
		HandlerName: "getVisitorHandler",
		Handler:     getVisitorHandler,
		BaseURL:     "/api/v1/visitor/{id}",
	}

	patterns := NewTestScenarioBuilder().
		AddInvalidUUID("GET", "/api/v1/visitor/invalid-uuid").
		AddNonExistentVisitor("GET", "/api/v1/visitor/{id}").
		Build()

	suite.RunErrorTests(t, patterns)
}
