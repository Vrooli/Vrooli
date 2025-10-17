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
	HandlerName    string
	Handler        http.HandlerFunc
	BaseURL        string
	RequiredURLVars []string
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
			w := testHandlerWithRequest(t, suite.Handler, *req)

			// Validate
			if w.Code != pattern.ExpectedStatus {
				t.Errorf("Expected status %d, got %d. Response: %s",
					pattern.ExpectedStatus, w.Code, w.Body.String())
			}

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

// nonExistentResourcePattern tests handlers with non-existent resource IDs
func nonExistentResourcePattern(method, urlPath, resourceType string) ErrorTestPattern {
	return ErrorTestPattern{
		Name:           fmt.Sprintf("NonExistent%s", resourceType),
		Description:    fmt.Sprintf("Test handler with non-existent %s ID", resourceType),
		ExpectedStatus: http.StatusNotFound,
		Setup:          nil,
		Execute: func(t *testing.T, setupData interface{}) *HTTPTestRequest {
			// Use a valid UUID that doesn't exist
			nonExistentID := "00000000-0000-0000-0000-000000000001"
			return &HTTPTestRequest{
				Method:  method,
				Path:    urlPath,
				URLVars: map[string]string{"id": nonExistentID},
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

// missingRequiredFieldPattern tests handlers with missing required fields
func missingRequiredFieldPattern(method, urlPath, fieldName string) ErrorTestPattern {
	return ErrorTestPattern{
		Name:           fmt.Sprintf("Missing%s", fieldName),
		Description:    fmt.Sprintf("Test handler with missing required field: %s", fieldName),
		ExpectedStatus: http.StatusBadRequest,
		Setup:          nil,
		Execute: func(t *testing.T, setupData interface{}) *HTTPTestRequest {
			return &HTTPTestRequest{
				Method: method,
				Path:   urlPath,
				Body:   map[string]interface{}{}, // Empty body
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

// PerformanceTestPattern defines performance testing scenarios
type PerformanceTestPattern struct {
	Name           string
	Description    string
	MaxDuration    time.Duration
	Setup          func(t *testing.T) interface{}
	Execute        func(t *testing.T, setupData interface{}) time.Duration
	Cleanup        func(setupData interface{})
}

// ConcurrencyTestPattern defines concurrency testing scenarios
type ConcurrencyTestPattern struct {
	Name           string
	Description    string
	Concurrency    int
	Iterations     int
	Setup          func(t *testing.T) interface{}
	Execute        func(t *testing.T, setupData interface{}, iteration int) error
	Validate       func(t *testing.T, setupData interface{}, results []error)
	Cleanup        func(setupData interface{})
}

// RunConcurrencyTest executes a concurrency test pattern
func (p *ConcurrencyTestPattern) Run(t *testing.T) {
	// Setup
	var setupData interface{}
	if p.Setup != nil {
		setupData = p.Setup(t)
	}

	// Cleanup
	if p.Cleanup != nil {
		defer p.Cleanup(setupData)
	}

	// Execute concurrent operations
	results := make([]error, p.Iterations)
	done := make(chan struct{})

	for i := 0; i < p.Iterations; i++ {
		go func(iteration int) {
			results[iteration] = p.Execute(t, setupData, iteration)
			done <- struct{}{}
		}(i)
	}

	// Wait for all to complete
	for i := 0; i < p.Iterations; i++ {
		<-done
	}

	// Validate
	if p.Validate != nil {
		p.Validate(t, setupData, results)
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

// AddInvalidUUID adds invalid UUID test pattern
func (b *TestScenarioBuilder) AddInvalidUUID(method, urlPath string) *TestScenarioBuilder {
	b.patterns = append(b.patterns, invalidUUIDPattern(method, urlPath))
	return b
}

// AddNonExistentResource adds non-existent resource test pattern
func (b *TestScenarioBuilder) AddNonExistentResource(method, urlPath, resourceType string) *TestScenarioBuilder {
	b.patterns = append(b.patterns, nonExistentResourcePattern(method, urlPath, resourceType))
	return b
}

// AddInvalidJSON adds invalid JSON test pattern
func (b *TestScenarioBuilder) AddInvalidJSON(method, urlPath string) *TestScenarioBuilder {
	b.patterns = append(b.patterns, invalidJSONPattern(method, urlPath))
	return b
}

// AddMissingRequiredField adds missing required field test pattern
func (b *TestScenarioBuilder) AddMissingRequiredField(method, urlPath, fieldName string) *TestScenarioBuilder {
	b.patterns = append(b.patterns, missingRequiredFieldPattern(method, urlPath, fieldName))
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

// Build returns the configured test patterns
func (b *TestScenarioBuilder) Build() []ErrorTestPattern {
	return b.patterns
}

// DatabaseTestHelper provides utilities for database testing
type DatabaseTestHelper struct {
	DB *sql.DB
}

// AssertRowCount validates the number of rows in a table
func (h *DatabaseTestHelper) AssertRowCount(t *testing.T, table string, expected int) {
	var count int
	err := h.DB.QueryRow(fmt.Sprintf("SELECT COUNT(*) FROM %s", table)).Scan(&count)
	if err != nil {
		t.Fatalf("Failed to count rows in %s: %v", table, err)
	}

	if count != expected {
		t.Errorf("Expected %d rows in %s, got %d", expected, table, count)
	}
}

// AssertRecordExists validates that a record exists
func (h *DatabaseTestHelper) AssertRecordExists(t *testing.T, table, idField, id string) {
	var exists bool
	query := fmt.Sprintf("SELECT EXISTS(SELECT 1 FROM %s WHERE %s = $1)", table, idField)
	err := h.DB.QueryRow(query, id).Scan(&exists)
	if err != nil {
		t.Fatalf("Failed to check if record exists in %s: %v", table, err)
	}

	if !exists {
		t.Errorf("Expected record with %s=%s to exist in %s", idField, id, table)
	}
}

// AssertRecordNotExists validates that a record does not exist
func (h *DatabaseTestHelper) AssertRecordNotExists(t *testing.T, table, idField, id string) {
	var exists bool
	query := fmt.Sprintf("SELECT EXISTS(SELECT 1 FROM %s WHERE %s = $1)", table, idField)
	err := h.DB.QueryRow(query, id).Scan(&exists)
	if err != nil {
		t.Fatalf("Failed to check if record exists in %s: %v", table, err)
	}

	if exists {
		t.Errorf("Expected record with %s=%s to not exist in %s", idField, id, table)
	}
}
