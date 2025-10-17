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
	App         *App
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
func (b *TestScenarioBuilder) AddInvalidUUID(urlPath string, method string) *TestScenarioBuilder {
	b.patterns = append(b.patterns, ErrorTestPattern{
		Name:           "InvalidUUID",
		Description:    "Test handler with invalid UUID format",
		ExpectedStatus: http.StatusBadRequest,
		Execute: func(t *testing.T, setupData interface{}) *HTTPTestRequest {
			return &HTTPTestRequest{
				Method:  method,
				Path:    urlPath,
				URLVars: map[string]string{"id": "invalid-uuid-format"},
			}
		},
	})
	return b
}

// AddNonExistentList adds non-existent list test pattern
func (b *TestScenarioBuilder) AddNonExistentList(urlPath string, method string) *TestScenarioBuilder {
	b.patterns = append(b.patterns, ErrorTestPattern{
		Name:           "NonExistentList",
		Description:    "Test handler with non-existent list ID",
		ExpectedStatus: http.StatusNotFound,
		Execute: func(t *testing.T, setupData interface{}) *HTTPTestRequest {
			nonExistentID := uuid.New().String()
			return &HTTPTestRequest{
				Method:  method,
				Path:    urlPath,
				URLVars: map[string]string{"id": nonExistentID},
			}
		},
	})
	return b
}

// AddInvalidJSON adds invalid JSON test pattern
func (b *TestScenarioBuilder) AddInvalidJSON(urlPath string, method string) *TestScenarioBuilder {
	b.patterns = append(b.patterns, ErrorTestPattern{
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
	})
	return b
}

// AddMissingRequiredFields adds missing fields test pattern
func (b *TestScenarioBuilder) AddMissingRequiredFields(urlPath string, method string, body interface{}) *TestScenarioBuilder {
	b.patterns = append(b.patterns, ErrorTestPattern{
		Name:           "MissingRequiredFields",
		Description:    "Test handler with missing required fields",
		ExpectedStatus: http.StatusBadRequest,
		Execute: func(t *testing.T, setupData interface{}) *HTTPTestRequest {
			return &HTTPTestRequest{
				Method: method,
				Path:   urlPath,
				Body:   body,
			}
		},
	})
	return b
}

// AddEmptyBody adds empty body test pattern
func (b *TestScenarioBuilder) AddEmptyBody(urlPath string, method string) *TestScenarioBuilder {
	b.patterns = append(b.patterns, ErrorTestPattern{
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

// RunConcurrencyTest runs a concurrency test pattern
func RunConcurrencyTest(t *testing.T, pattern ConcurrencyTestPattern) {
	// Setup
	var setupData interface{}
	if pattern.Setup != nil {
		setupData = pattern.Setup(t)
	}

	// Cleanup
	if pattern.Cleanup != nil {
		defer pattern.Cleanup(setupData)
	}

	// Run concurrent executions
	results := make([]error, pattern.Iterations)
	done := make(chan int, pattern.Concurrency)

	for i := 0; i < pattern.Iterations; i++ {
		// Wait if we've hit concurrency limit
		if i >= pattern.Concurrency {
			<-done
		}

		go func(iteration int) {
			defer func() { done <- iteration }()
			results[iteration] = pattern.Execute(t, setupData, iteration)
		}(i)
	}

	// Wait for all to complete
	for i := 0; i < pattern.Concurrency && i < pattern.Iterations; i++ {
		<-done
	}

	// Validate
	if pattern.Validate != nil {
		pattern.Validate(t, setupData, results)
	}
}

// Common test scenario helpers

// CreateListScenarios returns common test scenarios for list creation
func CreateListScenarios() *TestScenarioBuilder {
	return NewTestScenarioBuilder().
		AddInvalidJSON("/api/v1/lists", "POST").
		AddEmptyBody("/api/v1/lists", "POST").
		AddMissingRequiredFields("/api/v1/lists", "POST", map[string]interface{}{
			"name": "", // Empty name
		})
}

// GetListScenarios returns common test scenarios for getting a list
func GetListScenarios() *TestScenarioBuilder {
	return NewTestScenarioBuilder().
		AddInvalidUUID("/api/v1/lists/{id}", "GET").
		AddNonExistentList("/api/v1/lists/{id}", "GET")
}

// ComparisonScenarios returns common test scenarios for comparisons
func ComparisonScenarios() *TestScenarioBuilder {
	return NewTestScenarioBuilder().
		AddInvalidJSON("/api/v1/comparisons", "POST").
		AddEmptyBody("/api/v1/comparisons", "POST").
		AddMissingRequiredFields("/api/v1/comparisons", "POST", map[string]interface{}{
			"list_id": "",
		})
}

// RankingsScenarios returns common test scenarios for rankings
func RankingsScenarios() *TestScenarioBuilder {
	return NewTestScenarioBuilder().
		AddInvalidUUID("/api/v1/lists/{id}/rankings", "GET").
		AddNonExistentList("/api/v1/lists/{id}/rankings", "GET")
}

// EdgeCaseBuilder provides helpers for building edge case tests
type EdgeCaseBuilder struct {
	tests []func(*testing.T, *App)
}

// NewEdgeCaseBuilder creates a new edge case builder
func NewEdgeCaseBuilder() *EdgeCaseBuilder {
	return &EdgeCaseBuilder{
		tests: []func(*testing.T, *App){},
	}
}

// AddZeroItems adds a test for lists with zero items
func (b *EdgeCaseBuilder) AddZeroItems() *EdgeCaseBuilder {
	b.tests = append(b.tests, func(t *testing.T, app *App) {
		req := CreateListRequest{
			Name:        "Empty List",
			Description: "List with no items",
			Items:       []ItemInput{},
		}

		w, httpReq, _ := makeHTTPRequest(HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/lists",
			Body:   req,
		})

		app.CreateList(w, httpReq)

		if w.Code != http.StatusCreated {
			t.Errorf("Expected status %d for empty list, got %d", http.StatusCreated, w.Code)
		}
	})
	return b
}

// AddOneItem adds a test for lists with only one item
func (b *EdgeCaseBuilder) AddOneItem() *EdgeCaseBuilder {
	b.tests = append(b.tests, func(t *testing.T, app *App) {
		req := TestData.CreateListRequest("Single Item List", 1)

		w, httpReq, _ := makeHTTPRequest(HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/lists",
			Body:   req,
		})

		app.CreateList(w, httpReq)

		if w.Code != http.StatusCreated {
			t.Errorf("Expected status %d for single item list, got %d", http.StatusCreated, w.Code)
		}
	})
	return b
}

// AddLargeItemCount adds a test for lists with many items
func (b *EdgeCaseBuilder) AddLargeItemCount(count int) *EdgeCaseBuilder {
	b.tests = append(b.tests, func(t *testing.T, app *App) {
		req := TestData.CreateListRequest("Large List", count)

		w, httpReq, _ := makeHTTPRequest(HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/lists",
			Body:   req,
		})

		app.CreateList(w, httpReq)

		if w.Code != http.StatusCreated {
			t.Errorf("Expected status %d for large list, got %d", http.StatusCreated, w.Code)
		}
	})
	return b
}

// Build returns all edge case tests
func (b *EdgeCaseBuilder) Build() []func(*testing.T, *App) {
	return b.tests
}
