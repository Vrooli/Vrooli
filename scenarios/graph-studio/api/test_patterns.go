//go:build testing
// +build testing

package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
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

// TestScenarioBuilder provides a fluent interface for building test scenarios
type TestScenarioBuilder struct {
	scenarios []ErrorTestPattern
}

// NewTestScenarioBuilder creates a new test scenario builder
func NewTestScenarioBuilder() *TestScenarioBuilder {
	return &TestScenarioBuilder{
		scenarios: make([]ErrorTestPattern, 0),
	}
}

// AddInvalidUUID adds an invalid UUID test pattern
func (b *TestScenarioBuilder) AddInvalidUUID(urlPath string) *TestScenarioBuilder {
	b.scenarios = append(b.scenarios, ErrorTestPattern{
		Name:           "InvalidUUID",
		Description:    "Test handler with invalid UUID format",
		ExpectedStatus: http.StatusBadRequest,
		Execute: func(t *testing.T, setupData interface{}) *HTTPTestRequest {
			return &HTTPTestRequest{
				Method:  "GET",
				Path:    urlPath,
				URLVars: map[string]string{"id": "invalid-uuid"},
			}
		},
	})
	return b
}

// AddNonExistentGraph adds a non-existent graph test pattern
func (b *TestScenarioBuilder) AddNonExistentGraph(urlPath string, method string) *TestScenarioBuilder {
	b.scenarios = append(b.scenarios, ErrorTestPattern{
		Name:           "NonExistentGraph",
		Description:    "Test handler with non-existent graph ID",
		ExpectedStatus: http.StatusNotFound,
		Execute: func(t *testing.T, setupData interface{}) *HTTPTestRequest {
			nonExistentID := uuid.New()
			return &HTTPTestRequest{
				Method:  method,
				Path:    urlPath,
				URLVars: map[string]string{"id": nonExistentID.String()},
				Headers: map[string]string{"X-User-ID": uuid.New().String()},
			}
		},
	})
	return b
}

// AddInvalidJSON adds a malformed JSON test pattern
func (b *TestScenarioBuilder) AddInvalidJSON(urlPath string, method string) *TestScenarioBuilder {
	b.scenarios = append(b.scenarios, ErrorTestPattern{
		Name:           "InvalidJSON",
		Description:    "Test handler with malformed JSON input",
		ExpectedStatus: http.StatusBadRequest,
		Execute: func(t *testing.T, setupData interface{}) *HTTPTestRequest {
			return &HTTPTestRequest{
				Method:  method,
				Path:    urlPath,
				Body:    `{"invalid": "json"`, // Malformed JSON
				Headers: map[string]string{"X-User-ID": uuid.New().String()},
			}
		},
	})
	return b
}

// AddMissingRequiredField adds a test pattern for missing required fields
func (b *TestScenarioBuilder) AddMissingRequiredField(urlPath string, method string, fieldName string) *TestScenarioBuilder {
	b.scenarios = append(b.scenarios, ErrorTestPattern{
		Name:           fmt.Sprintf("Missing_%s", fieldName),
		Description:    fmt.Sprintf("Test handler with missing required field: %s", fieldName),
		ExpectedStatus: http.StatusBadRequest,
		Execute: func(t *testing.T, setupData interface{}) *HTTPTestRequest {
			body := make(map[string]interface{})
			// Intentionally leave out the required field
			return &HTTPTestRequest{
				Method:  method,
				Path:    urlPath,
				Body:    body,
				Headers: map[string]string{"X-User-ID": uuid.New().String()},
			}
		},
	})
	return b
}

// AddUnauthorizedAccess adds an unauthorized access test pattern
func (b *TestScenarioBuilder) AddUnauthorizedAccess(urlPath string, method string, graphID string) *TestScenarioBuilder {
	b.scenarios = append(b.scenarios, ErrorTestPattern{
		Name:           "UnauthorizedAccess",
		Description:    "Test handler with unauthorized user access",
		ExpectedStatus: http.StatusForbidden,
		Execute: func(t *testing.T, setupData interface{}) *HTTPTestRequest {
			return &HTTPTestRequest{
				Method:  method,
				Path:    urlPath,
				URLVars: map[string]string{"id": graphID},
				Headers: map[string]string{"X-User-ID": uuid.New().String()}, // Different user
			}
		},
	})
	return b
}

// AddInvalidGraphType adds a test pattern for invalid graph types
func (b *TestScenarioBuilder) AddInvalidGraphType(urlPath string) *TestScenarioBuilder {
	b.scenarios = append(b.scenarios, ErrorTestPattern{
		Name:           "InvalidGraphType",
		Description:    "Test handler with invalid graph type",
		ExpectedStatus: http.StatusBadRequest,
		Execute: func(t *testing.T, setupData interface{}) *HTTPTestRequest {
			body := map[string]interface{}{
				"name": "Test Graph",
				"type": "invalid-type-xyz",
			}
			return &HTTPTestRequest{
				Method:  "POST",
				Path:    urlPath,
				Body:    body,
				Headers: map[string]string{"X-User-ID": uuid.New().String()},
			}
		},
	})
	return b
}

// AddEmptyName adds a test pattern for empty graph name
func (b *TestScenarioBuilder) AddEmptyName(urlPath string) *TestScenarioBuilder {
	b.scenarios = append(b.scenarios, ErrorTestPattern{
		Name:           "EmptyName",
		Description:    "Test handler with empty graph name",
		ExpectedStatus: http.StatusBadRequest,
		Execute: func(t *testing.T, setupData interface{}) *HTTPTestRequest {
			body := map[string]interface{}{
				"name": "",
				"type": "mind-maps",
			}
			return &HTTPTestRequest{
				Method:  "POST",
				Path:    urlPath,
				Body:    body,
				Headers: map[string]string{"X-User-ID": uuid.New().String()},
			}
		},
	})
	return b
}

// AddNameTooLong adds a test pattern for graph name that exceeds max length
func (b *TestScenarioBuilder) AddNameTooLong(urlPath string, maxLength int) *TestScenarioBuilder {
	b.scenarios = append(b.scenarios, ErrorTestPattern{
		Name:           "NameTooLong",
		Description:    "Test handler with graph name exceeding max length",
		ExpectedStatus: http.StatusBadRequest,
		Execute: func(t *testing.T, setupData interface{}) *HTTPTestRequest {
			longName := ""
			for i := 0; i < maxLength+10; i++ {
				longName += "a"
			}
			body := map[string]interface{}{
				"name": longName,
				"type": "mind-maps",
			}
			return &HTTPTestRequest{
				Method:  "POST",
				Path:    urlPath,
				Body:    body,
				Headers: map[string]string{"X-User-ID": uuid.New().String()},
			}
		},
	})
	return b
}

// Build returns all scenarios
func (b *TestScenarioBuilder) Build() []ErrorTestPattern {
	return b.scenarios
}

// HandlerTestSuite provides a comprehensive test framework for HTTP handlers
type HandlerTestSuite struct {
	HandlerName string
	Handler     http.HandlerFunc
	BaseURL     string
	Router      *mux.Router
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

			// Execute handler through router
			if suite.Router != nil {
				suite.Router.ServeHTTP(w, w.Result().Request)
			} else if suite.Handler != nil {
				httpReq := w.Result().Request
				suite.Handler.ServeHTTP(w, httpReq)
			}

			// Validate status code
			if w.Code != pattern.ExpectedStatus {
				t.Errorf("Expected status %d, got %d. Body: %s",
					pattern.ExpectedStatus, w.Code, w.Body.String())
			}

			// Custom validation
			if pattern.Validate != nil {
				pattern.Validate(t, w, setupData)
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
	Validate    func(t *testing.T, duration time.Duration, setupData interface{})
	Cleanup     func(setupData interface{})
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

		// Validate duration
		if duration > pattern.MaxDuration {
			t.Errorf("Operation took %v, expected < %v", duration, pattern.MaxDuration)
		}

		// Custom validation
		if pattern.Validate != nil {
			pattern.Validate(t, duration, setupData)
		}
	})
}

// Common performance patterns

// BulkOperationPattern tests performance of bulk operations
func BulkOperationPattern(name string, count int, maxDuration time.Duration,
	operation func() error) PerformanceTestPattern {
	return PerformanceTestPattern{
		Name:        name,
		Description: fmt.Sprintf("Test performance of %d bulk operations", count),
		MaxDuration: maxDuration,
		Execute: func(t *testing.T, setupData interface{}) time.Duration {
			start := time.Now()
			for i := 0; i < count; i++ {
				if err := operation(); err != nil {
					t.Fatalf("Operation failed at iteration %d: %v", i, err)
				}
			}
			return time.Since(start)
		},
	}
}

// ConcurrentOperationPattern tests performance under concurrent load
func ConcurrentOperationPattern(name string, concurrency int, operations int,
	maxDuration time.Duration, operation func() error) PerformanceTestPattern {
	return PerformanceTestPattern{
		Name:        name,
		Description: fmt.Sprintf("Test %d concurrent operations with %d goroutines", operations, concurrency),
		MaxDuration: maxDuration,
		Execute: func(t *testing.T, setupData interface{}) time.Duration {
			start := time.Now()

			done := make(chan bool, concurrency)
			errors := make(chan error, operations)

			for i := 0; i < concurrency; i++ {
				go func() {
					for j := 0; j < operations/concurrency; j++ {
						if err := operation(); err != nil {
							errors <- err
							return
						}
					}
					done <- true
				}()
			}

			// Wait for completion
			for i := 0; i < concurrency; i++ {
				select {
				case <-done:
					// Success
				case err := <-errors:
					t.Fatalf("Concurrent operation failed: %v", err)
				case <-time.After(maxDuration * 2):
					t.Fatal("Concurrent operations timed out")
				}
			}

			return time.Since(start)
		},
	}
}
