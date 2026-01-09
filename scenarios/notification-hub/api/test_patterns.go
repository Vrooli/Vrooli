
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
	Setup          func(t *testing.T, env *TestEnvironment) interface{}
	Execute        func(t *testing.T, env *TestEnvironment, setupData interface{}) HTTPTestRequest
	Validate       func(t *testing.T, w *httptest.ResponseRecorder, setupData interface{})
	Cleanup        func(setupData interface{})
}

// TestScenarioBuilder provides a fluent interface for building test scenarios
type TestScenarioBuilder struct {
	scenarios []ErrorTestPattern
}

// NewTestScenarioBuilder creates a new scenario builder
func NewTestScenarioBuilder() *TestScenarioBuilder {
	return &TestScenarioBuilder{
		scenarios: []ErrorTestPattern{},
	}
}

// AddInvalidUUID adds a test for invalid UUID format
func (b *TestScenarioBuilder) AddInvalidUUID(path string, method string) *TestScenarioBuilder {
	b.scenarios = append(b.scenarios, ErrorTestPattern{
		Name:           "InvalidUUID",
		Description:    "Test handler with invalid UUID format",
		ExpectedStatus: http.StatusBadRequest,
		Execute: func(t *testing.T, env *TestEnvironment, setupData interface{}) HTTPTestRequest {
			return HTTPTestRequest{
				Method:  method,
				Path:    path,
				Headers: map[string]string{"X-API-Key": env.TestAPIKey},
			}
		},
	})
	return b
}

// AddNonExistentProfile adds a test for non-existent profile ID
func (b *TestScenarioBuilder) AddNonExistentProfile(path string, method string) *TestScenarioBuilder {
	b.scenarios = append(b.scenarios, ErrorTestPattern{
		Name:           "NonExistentProfile",
		Description:    "Test handler with non-existent profile ID",
		ExpectedStatus: http.StatusNotFound,
		Execute: func(t *testing.T, env *TestEnvironment, setupData interface{}) HTTPTestRequest {
			nonExistentID := uuid.New()
			actualPath := fmt.Sprintf(path, nonExistentID.String())
			return HTTPTestRequest{
				Method:  method,
				Path:    actualPath,
				Headers: map[string]string{"X-API-Key": env.TestAPIKey},
			}
		},
	})
	return b
}

// AddNonExistentContact adds a test for non-existent contact ID
func (b *TestScenarioBuilder) AddNonExistentContact(path string, method string) *TestScenarioBuilder {
	b.scenarios = append(b.scenarios, ErrorTestPattern{
		Name:           "NonExistentContact",
		Description:    "Test handler with non-existent contact ID",
		ExpectedStatus: http.StatusNotFound,
		Execute: func(t *testing.T, env *TestEnvironment, setupData interface{}) HTTPTestRequest {
			nonExistentID := uuid.New()
			actualPath := fmt.Sprintf(path, env.TestProfile.ID.String(), nonExistentID.String())
			return HTTPTestRequest{
				Method:  method,
				Path:    actualPath,
				Headers: map[string]string{"X-API-Key": env.TestAPIKey},
			}
		},
	})
	return b
}

// AddInvalidJSON adds a test for malformed JSON input
func (b *TestScenarioBuilder) AddInvalidJSON(path string, method string) *TestScenarioBuilder {
	b.scenarios = append(b.scenarios, ErrorTestPattern{
		Name:           "InvalidJSON",
		Description:    "Test handler with malformed JSON input",
		ExpectedStatus: http.StatusBadRequest,
		Execute: func(t *testing.T, env *TestEnvironment, setupData interface{}) HTTPTestRequest {
			actualPath := fmt.Sprintf(path, env.TestProfile.ID.String())
			return HTTPTestRequest{
				Method:  method,
				Path:    actualPath,
				Body:    `{"invalid": "json"`, // Malformed JSON
				Headers: map[string]string{"X-API-Key": env.TestAPIKey},
			}
		},
	})
	return b
}

// AddMissingAPIKey adds a test for missing API key authentication
func (b *TestScenarioBuilder) AddMissingAPIKey(path string, method string) *TestScenarioBuilder {
	b.scenarios = append(b.scenarios, ErrorTestPattern{
		Name:           "MissingAPIKey",
		Description:    "Test handler without API key",
		ExpectedStatus: http.StatusUnauthorized,
		Execute: func(t *testing.T, env *TestEnvironment, setupData interface{}) HTTPTestRequest {
			actualPath := fmt.Sprintf(path, env.TestProfile.ID.String())
			return HTTPTestRequest{
				Method:  method,
				Path:    actualPath,
				Headers: map[string]string{}, // No API key
			}
		},
	})
	return b
}

// AddInvalidAPIKey adds a test for invalid API key
func (b *TestScenarioBuilder) AddInvalidAPIKey(path string, method string) *TestScenarioBuilder {
	b.scenarios = append(b.scenarios, ErrorTestPattern{
		Name:           "InvalidAPIKey",
		Description:    "Test handler with invalid API key",
		ExpectedStatus: http.StatusUnauthorized,
		Execute: func(t *testing.T, env *TestEnvironment, setupData interface{}) HTTPTestRequest {
			actualPath := fmt.Sprintf(path, env.TestProfile.ID.String())
			return HTTPTestRequest{
				Method:  method,
				Path:    actualPath,
				Headers: map[string]string{"X-API-Key": "invalid-key-12345"},
			}
		},
	})
	return b
}

// AddMissingRequiredField adds a test for missing required fields in request body
func (b *TestScenarioBuilder) AddMissingRequiredField(path string, method string, fieldName string) *TestScenarioBuilder {
	b.scenarios = append(b.scenarios, ErrorTestPattern{
		Name:           fmt.Sprintf("MissingField_%s", fieldName),
		Description:    fmt.Sprintf("Test handler with missing required field: %s", fieldName),
		ExpectedStatus: http.StatusBadRequest,
		Execute: func(t *testing.T, env *TestEnvironment, setupData interface{}) HTTPTestRequest {
			actualPath := fmt.Sprintf(path, env.TestProfile.ID.String())
			// Empty JSON object to trigger missing field validation
			return HTTPTestRequest{
				Method:  method,
				Path:    actualPath,
				Body:    map[string]interface{}{},
				Headers: map[string]string{"X-API-Key": env.TestAPIKey},
			}
		},
	})
	return b
}

// AddEmptyArray adds a test for empty array inputs
func (b *TestScenarioBuilder) AddEmptyArray(path string, method string, arrayField string) *TestScenarioBuilder {
	b.scenarios = append(b.scenarios, ErrorTestPattern{
		Name:           fmt.Sprintf("EmptyArray_%s", arrayField),
		Description:    fmt.Sprintf("Test handler with empty array for: %s", arrayField),
		ExpectedStatus: http.StatusBadRequest,
		Execute: func(t *testing.T, env *TestEnvironment, setupData interface{}) HTTPTestRequest {
			actualPath := fmt.Sprintf(path, env.TestProfile.ID.String())
			return HTTPTestRequest{
				Method:  method,
				Path:    actualPath,
				Body:    map[string]interface{}{arrayField: []interface{}{}},
				Headers: map[string]string{"X-API-Key": env.TestAPIKey},
			}
		},
	})
	return b
}

// Build returns the completed set of test scenarios
func (b *TestScenarioBuilder) Build() []ErrorTestPattern {
	return b.scenarios
}

// PerformanceTestPattern defines performance testing scenarios
type PerformanceTestPattern struct {
	Name         string
	Description  string
	MaxDuration  time.Duration
	Setup        func(t *testing.T, env *TestEnvironment) interface{}
	Execute      func(t *testing.T, env *TestEnvironment, setupData interface{}) time.Duration
	Validate     func(t *testing.T, duration time.Duration, setupData interface{})
	Cleanup      func(setupData interface{})
}

// PerformanceTestBuilder provides a fluent interface for building performance tests
type PerformanceTestBuilder struct {
	tests []PerformanceTestPattern
}

// NewPerformanceTestBuilder creates a new performance test builder
func NewPerformanceTestBuilder() *PerformanceTestBuilder {
	return &PerformanceTestBuilder{
		tests: []PerformanceTestPattern{},
	}
}

// AddEndpointLatencyTest adds a test for endpoint response time
func (b *PerformanceTestBuilder) AddEndpointLatencyTest(name string, path string, maxLatency time.Duration) *PerformanceTestBuilder {
	b.tests = append(b.tests, PerformanceTestPattern{
		Name:        fmt.Sprintf("Latency_%s", name),
		Description: fmt.Sprintf("Test %s endpoint latency", name),
		MaxDuration: maxLatency,
		Execute: func(t *testing.T, env *TestEnvironment, setupData interface{}) time.Duration {
			start := time.Now()
			w, err := makeHTTPRequest(env, HTTPTestRequest{
				Method:  "GET",
				Path:    path,
				Headers: map[string]string{"X-API-Key": env.TestAPIKey},
			})
			duration := time.Since(start)

			if err != nil {
				t.Errorf("Request failed: %v", err)
			}
			if w.Code != http.StatusOK && w.Code != http.StatusNotFound {
				t.Errorf("Unexpected status: %d", w.Code)
			}

			return duration
		},
		Validate: func(t *testing.T, duration time.Duration, setupData interface{}) {
			// Validation happens in the test runner
		},
	})
	return b
}

// AddBulkOperationTest adds a test for bulk operation performance
func (b *PerformanceTestBuilder) AddBulkOperationTest(name string, count int, maxDuration time.Duration, operation func(*testing.T, *TestEnvironment, int)) *PerformanceTestBuilder {
	b.tests = append(b.tests, PerformanceTestPattern{
		Name:        fmt.Sprintf("Bulk_%s_%d", name, count),
		Description: fmt.Sprintf("Test %s with %d items", name, count),
		MaxDuration: maxDuration,
		Execute: func(t *testing.T, env *TestEnvironment, setupData interface{}) time.Duration {
			start := time.Now()
			operation(t, env, count)
			return time.Since(start)
		},
	})
	return b
}

// Build returns the completed set of performance tests
func (b *PerformanceTestBuilder) Build() []PerformanceTestPattern {
	return b.tests
}

// RunErrorTests executes a suite of error condition tests
func RunErrorTests(t *testing.T, env *TestEnvironment, patterns []ErrorTestPattern) {
	for _, pattern := range patterns {
		t.Run(pattern.Name, func(t *testing.T) {
			// Setup
			var setupData interface{}
			if pattern.Setup != nil {
				setupData = pattern.Setup(t, env)
			}

			// Cleanup
			if pattern.Cleanup != nil {
				defer pattern.Cleanup(setupData)
			}

			// Execute
			req := pattern.Execute(t, env, setupData)
			w, err := makeHTTPRequest(env, req)
			if err != nil {
				t.Fatalf("Failed to make HTTP request: %v", err)
			}

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

// RunPerformanceTests executes a suite of performance tests
func RunPerformanceTests(t *testing.T, env *TestEnvironment, patterns []PerformanceTestPattern) {
	for _, pattern := range patterns {
		t.Run(pattern.Name, func(t *testing.T) {
			// Setup
			var setupData interface{}
			if pattern.Setup != nil {
				setupData = pattern.Setup(t, env)
			}

			// Cleanup
			if pattern.Cleanup != nil {
				defer pattern.Cleanup(setupData)
			}

			// Execute and measure
			duration := pattern.Execute(t, env, setupData)

			// Validate duration
			if duration > pattern.MaxDuration {
				t.Errorf("%s took %v, expected less than %v",
					pattern.Name, duration, pattern.MaxDuration)
			} else {
				t.Logf("%s completed in %v (limit: %v)", pattern.Name, duration, pattern.MaxDuration)
			}

			// Custom validation
			if pattern.Validate != nil {
				pattern.Validate(t, duration, setupData)
			}
		})
	}
}

// ConcurrencyTestPattern defines concurrency testing scenarios
type ConcurrencyTestPattern struct {
	Name        string
	Description string
	Concurrency int
	Iterations  int
	Setup       func(t *testing.T, env *TestEnvironment) interface{}
	Execute     func(t *testing.T, env *TestEnvironment, setupData interface{}, iteration int) error
	Validate    func(t *testing.T, env *TestEnvironment, setupData interface{}, results []error)
	Cleanup     func(setupData interface{})
}

// ConcurrencyTestBuilder provides a fluent interface for building concurrency tests
type ConcurrencyTestBuilder struct {
	tests []ConcurrencyTestPattern
}

// NewConcurrencyTestBuilder creates a new concurrency test builder
func NewConcurrencyTestBuilder() *ConcurrencyTestBuilder {
	return &ConcurrencyTestBuilder{
		tests: []ConcurrencyTestPattern{},
	}
}

// AddConcurrentRequests adds a test for concurrent HTTP requests
func (b *ConcurrencyTestBuilder) AddConcurrentRequests(name, path, method string, concurrency, iterations int) *ConcurrencyTestBuilder {
	b.tests = append(b.tests, ConcurrencyTestPattern{
		Name:        fmt.Sprintf("Concurrent_%s_%dx%d", name, concurrency, iterations),
		Description: fmt.Sprintf("Test %d concurrent %s requests with %d iterations each", concurrency, name, iterations),
		Concurrency: concurrency,
		Iterations:  iterations,
		Execute: func(t *testing.T, env *TestEnvironment, setupData interface{}, iteration int) error {
			_, err := makeHTTPRequest(env, HTTPTestRequest{
				Method:  method,
				Path:    path,
				Headers: map[string]string{"X-API-Key": env.TestAPIKey},
			})
			return err
		},
	})
	return b
}

// Build returns the completed set of concurrency tests
func (b *ConcurrencyTestBuilder) Build() []ConcurrencyTestPattern {
	return b.tests
}

// RunConcurrencyTests executes a suite of concurrency tests
func RunConcurrencyTests(t *testing.T, env *TestEnvironment, patterns []ConcurrencyTestPattern) {
	for _, pattern := range patterns {
		t.Run(pattern.Name, func(t *testing.T) {
			// Setup
			var setupData interface{}
			if pattern.Setup != nil {
				setupData = pattern.Setup(t, env)
			}

			// Cleanup
			if pattern.Cleanup != nil {
				defer pattern.Cleanup(setupData)
			}

			// Execute concurrently
			results := make([]error, pattern.Iterations)
			done := make(chan bool, pattern.Concurrency)

			for i := 0; i < pattern.Iterations; i++ {
				go func(iteration int) {
					results[iteration] = pattern.Execute(t, env, setupData, iteration)
					done <- true
				}(i)

				// Limit concurrency
				if (i+1)%pattern.Concurrency == 0 {
					for j := 0; j < pattern.Concurrency; j++ {
						<-done
					}
				}
			}

			// Wait for remaining
			remaining := pattern.Iterations % pattern.Concurrency
			for j := 0; j < remaining; j++ {
				<-done
			}

			// Count errors
			errorCount := 0
			for _, err := range results {
				if err != nil {
					errorCount++
				}
			}

			if errorCount > 0 {
				t.Errorf("%d out of %d concurrent operations failed", errorCount, pattern.Iterations)
			}

			// Custom validation
			if pattern.Validate != nil {
				pattern.Validate(t, env, setupData, results)
			}
		})
	}
}

// EdgeCaseTestPattern defines edge case testing scenarios
type EdgeCaseTestPattern struct {
	Name        string
	Description string
	Setup       func(t *testing.T, env *TestEnvironment) interface{}
	Execute     func(t *testing.T, env *TestEnvironment, setupData interface{}) error
	Validate    func(t *testing.T, env *TestEnvironment, setupData interface{}, err error)
	Cleanup     func(setupData interface{})
}

// EdgeCaseTestBuilder provides a fluent interface for building edge case tests
type EdgeCaseTestBuilder struct {
	tests []EdgeCaseTestPattern
}

// NewEdgeCaseTestBuilder creates a new edge case test builder
func NewEdgeCaseTestBuilder() *EdgeCaseTestBuilder {
	return &EdgeCaseTestBuilder{
		tests: []EdgeCaseTestPattern{},
	}
}

// AddNullValueTest adds a test for null/nil value handling
func (b *EdgeCaseTestBuilder) AddNullValueTest(name, path, method string, bodyBuilder func() map[string]interface{}) *EdgeCaseTestBuilder {
	b.tests = append(b.tests, EdgeCaseTestPattern{
		Name:        fmt.Sprintf("NullValue_%s", name),
		Description: fmt.Sprintf("Test %s with null values", name),
		Execute: func(t *testing.T, env *TestEnvironment, setupData interface{}) error {
			_, err := makeHTTPRequest(env, HTTPTestRequest{
				Method:  method,
				Path:    path,
				Body:    bodyBuilder(),
				Headers: map[string]string{"X-API-Key": env.TestAPIKey},
			})
			return err
		},
	})
	return b
}

// AddBoundaryValueTest adds a test for boundary conditions
func (b *EdgeCaseTestBuilder) AddBoundaryValueTest(name string, testFunc func(*testing.T, *TestEnvironment) error) *EdgeCaseTestBuilder {
	b.tests = append(b.tests, EdgeCaseTestPattern{
		Name:        fmt.Sprintf("Boundary_%s", name),
		Description: fmt.Sprintf("Test %s boundary conditions", name),
		Execute: func(t *testing.T, env *TestEnvironment, setupData interface{}) error {
			return testFunc(t, env)
		},
	})
	return b
}

// Build returns the completed set of edge case tests
func (b *EdgeCaseTestBuilder) Build() []EdgeCaseTestPattern {
	return b.tests
}

// RunEdgeCaseTests executes a suite of edge case tests
func RunEdgeCaseTests(t *testing.T, env *TestEnvironment, patterns []EdgeCaseTestPattern) {
	for _, pattern := range patterns {
		t.Run(pattern.Name, func(t *testing.T) {
			// Setup
			var setupData interface{}
			if pattern.Setup != nil {
				setupData = pattern.Setup(t, env)
			}

			// Cleanup
			if pattern.Cleanup != nil {
				defer pattern.Cleanup(setupData)
			}

			// Execute
			err := pattern.Execute(t, env, setupData)

			// Validate
			if pattern.Validate != nil {
				pattern.Validate(t, env, setupData, err)
			}
		})
	}
}
