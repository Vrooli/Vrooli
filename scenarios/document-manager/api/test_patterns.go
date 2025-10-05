package main

import (
	"database/sql"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gorilla/mux"
)

// ErrorTestPattern defines a systematic approach to testing error conditions
type ErrorTestPattern struct {
	Name           string
	Description    string
	Method         string
	Path           string
	Body           interface{}
	URLVars        map[string]string
	ExpectedStatus int
	ErrorSubstring string
}

// HandlerTestSuite provides a comprehensive test framework for HTTP handlers
type HandlerTestSuite struct {
	Router  *mux.Router
	BaseURL string
}

// RunErrorTests executes a suite of error condition tests
func (suite *HandlerTestSuite) RunErrorTests(t *testing.T, patterns []ErrorTestPattern) {
	for _, pattern := range patterns {
		t.Run(pattern.Name, func(t *testing.T) {
			req := HTTPTestRequest{
				Method:  pattern.Method,
				Path:    pattern.Path,
				Body:    pattern.Body,
				URLVars: pattern.URLVars,
			}

			w := makeHTTPRequest(suite.Router, req)

			if w.Code != pattern.ExpectedStatus {
				t.Errorf("Expected status %d, got %d. Response: %s",
					pattern.ExpectedStatus, w.Code, w.Body.String())
			}

			if pattern.ErrorSubstring != "" {
				assertErrorResponse(t, w, pattern.ExpectedStatus, pattern.ErrorSubstring)
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
func (b *TestScenarioBuilder) AddInvalidJSON(method, path string) *TestScenarioBuilder {
	b.patterns = append(b.patterns, ErrorTestPattern{
		Name:           "InvalidJSON",
		Description:    "Test handler with malformed JSON input",
		Method:         method,
		Path:           path,
		Body:           `{"invalid": "json"`,
		ExpectedStatus: http.StatusBadRequest,
		ErrorSubstring: "Invalid JSON",
	})
	return b
}

// AddEmptyBody adds empty body test pattern for POST/PUT requests
func (b *TestScenarioBuilder) AddEmptyBody(method, path string) *TestScenarioBuilder {
	b.patterns = append(b.patterns, ErrorTestPattern{
		Name:           "EmptyBody",
		Description:    "Test handler with empty request body",
		Method:         method,
		Path:           path,
		Body:           "",
		ExpectedStatus: http.StatusBadRequest,
		ErrorSubstring: "Invalid JSON",
	})
	return b
}

// AddMissingRequiredFields adds test pattern for missing required fields
func (b *TestScenarioBuilder) AddMissingRequiredFields(method, path string, body interface{}) *TestScenarioBuilder {
	b.patterns = append(b.patterns, ErrorTestPattern{
		Name:           "MissingRequiredFields",
		Description:    "Test handler with missing required fields",
		Method:         method,
		Path:           path,
		Body:           body,
		ExpectedStatus: http.StatusInternalServerError,
		ErrorSubstring: "Failed",
	})
	return b
}

// AddMethodNotAllowed adds method not allowed test pattern
func (b *TestScenarioBuilder) AddMethodNotAllowed(method, path string) *TestScenarioBuilder {
	b.patterns = append(b.patterns, ErrorTestPattern{
		Name:           "MethodNotAllowed",
		Description:    "Test handler with unsupported HTTP method",
		Method:         method,
		Path:           path,
		ExpectedStatus: http.StatusMethodNotAllowed,
		ErrorSubstring: "Method not allowed",
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
	Name           string
	Description    string
	MaxDuration    time.Duration
	Setup          func(t *testing.T) interface{}
	Execute        func(t *testing.T, setupData interface{}) time.Duration
	Validate       func(t *testing.T, duration time.Duration, setupData interface{})
	Cleanup        func(setupData interface{})
}

// RunPerformanceTest executes a performance test pattern
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
			t.Errorf("Performance test failed: execution took %v, expected max %v",
				duration, pattern.MaxDuration)
		}

		// Additional validation
		if pattern.Validate != nil {
			pattern.Validate(t, duration, setupData)
		}
	})
}

// DatabaseTestPattern provides patterns for database testing
type DatabaseTestPattern struct {
	Name        string
	Description string
	Setup       func(t *testing.T, db *sql.DB) interface{}
	Execute     func(t *testing.T, db *sql.DB, setupData interface{}) error
	Validate    func(t *testing.T, db *sql.DB, setupData interface{})
	Cleanup     func(t *testing.T, db *sql.DB, setupData interface{})
}

// RunDatabaseTest executes a database test pattern
func RunDatabaseTest(t *testing.T, db *sql.DB, pattern DatabaseTestPattern) {
	t.Run(pattern.Name, func(t *testing.T) {
		// Setup
		var setupData interface{}
		if pattern.Setup != nil {
			setupData = pattern.Setup(t, db)
		}

		// Cleanup
		if pattern.Cleanup != nil {
			defer pattern.Cleanup(t, db, setupData)
		}

		// Execute
		err := pattern.Execute(t, db, setupData)
		if err != nil {
			t.Errorf("Database test execution failed: %v", err)
		}

		// Validate
		if pattern.Validate != nil {
			pattern.Validate(t, db, setupData)
		}
	})
}

// IntegrationTestPattern defines integration testing scenarios
type IntegrationTestPattern struct {
	Name        string
	Description string
	Setup       func(t *testing.T) interface{}
	Steps       []IntegrationStep
	Validate    func(t *testing.T, results []interface{})
	Cleanup     func(setupData interface{})
}

// IntegrationStep represents a single step in an integration test
type IntegrationStep struct {
	Name        string
	Execute     func(t *testing.T, setupData interface{}) (interface{}, error)
	ExpectError bool
}

// RunIntegrationTest executes an integration test pattern
func RunIntegrationTest(t *testing.T, pattern IntegrationTestPattern) {
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

		// Execute steps
		results := make([]interface{}, 0, len(pattern.Steps))
		for _, step := range pattern.Steps {
			result, err := step.Execute(t, setupData)

			if step.ExpectError && err == nil {
				t.Errorf("Step '%s': expected error but got none", step.Name)
			} else if !step.ExpectError && err != nil {
				t.Errorf("Step '%s': unexpected error: %v", step.Name, err)
			}

			results = append(results, result)
		}

		// Validate overall results
		if pattern.Validate != nil {
			pattern.Validate(t, results)
		}
	})
}

// MiddlewareTestPattern defines middleware testing scenarios
type MiddlewareTestPattern struct {
	Name           string
	Description    string
	Middleware     func(http.Handler) http.Handler
	Handler        http.HandlerFunc
	Request        HTTPTestRequest
	ValidateBefore func(t *testing.T, r *http.Request)
	ValidateAfter  func(t *testing.T, w *httptest.ResponseRecorder)
}

// RunMiddlewareTest executes a middleware test pattern
func RunMiddlewareTest(t *testing.T, pattern MiddlewareTestPattern) {
	t.Run(pattern.Name, func(t *testing.T) {
		// Create router with middleware
		router := mux.NewRouter()
		router.Use(pattern.Middleware)
		router.HandleFunc(pattern.Request.Path, pattern.Handler)

		// Execute request
		w := makeHTTPRequest(router, pattern.Request)

		// Validate
		if pattern.ValidateAfter != nil {
			pattern.ValidateAfter(t, w)
		}
	})
}

// Example usage template
func ExampleTestScenarioBuilderUsage() *TestScenarioBuilder {
	return NewTestScenarioBuilder().
		AddInvalidJSON("POST", "/api/applications").
		AddEmptyBody("POST", "/api/applications").
		AddMethodNotAllowed("DELETE", "/api/applications").
		AddMissingRequiredFields("POST", "/api/applications", map[string]string{}).
		AddCustom(ErrorTestPattern{
			Name:           "CustomError",
			Description:    "Custom error test",
			Method:         "GET",
			Path:           "/api/custom",
			ExpectedStatus: http.StatusBadRequest,
		})
}

// Helper function to create common test scenarios for CRUD operations
func createCRUDTestPatterns(basePath string) []ErrorTestPattern {
	return NewTestScenarioBuilder().
		AddInvalidJSON("POST", basePath).
		AddEmptyBody("POST", basePath).
		AddMethodNotAllowed("DELETE", basePath).
		Build()
}
