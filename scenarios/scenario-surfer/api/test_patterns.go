
package main

import (
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
	Request        HTTPTestRequest
	ExpectedStatus int
	ExpectedError  string
	Setup          func(t *testing.T) interface{}
	Cleanup        func(setupData interface{})
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

// AddInvalidJSON adds a test for invalid JSON request body
func (b *TestScenarioBuilder) AddInvalidJSON(path string) *TestScenarioBuilder {
	b.patterns = append(b.patterns, ErrorTestPattern{
		Name:        "InvalidJSON",
		Description: "Test handler with malformed JSON body",
		Request: HTTPTestRequest{
			Method: "POST",
			Path:   path,
			Body:   nil, // Will be overridden with raw invalid JSON
		},
		ExpectedStatus: http.StatusBadRequest,
		ExpectedError:  "Invalid request body",
	})
	return b
}

// AddMissingRequiredFields adds a test for missing required fields
func (b *TestScenarioBuilder) AddMissingRequiredFields(path string, missingField string) *TestScenarioBuilder {
	b.patterns = append(b.patterns, ErrorTestPattern{
		Name:        fmt.Sprintf("Missing_%s", missingField),
		Description: fmt.Sprintf("Test handler with missing required field: %s", missingField),
		Request: HTTPTestRequest{
			Method: "POST",
			Path:   path,
			Body:   map[string]interface{}{}, // Empty body
		},
		ExpectedStatus: http.StatusBadRequest,
		ExpectedError:  "Missing required fields",
	})
	return b
}

// AddNonExistentResource adds a test for non-existent resource
func (b *TestScenarioBuilder) AddNonExistentResource(path string, resourceType string) *TestScenarioBuilder {
	b.patterns = append(b.patterns, ErrorTestPattern{
		Name:        fmt.Sprintf("NonExistent_%s", resourceType),
		Description: fmt.Sprintf("Test handler with non-existent %s", resourceType),
		Request: HTTPTestRequest{
			Method: "GET",
			Path:   path,
		},
		ExpectedStatus: http.StatusNotFound,
		ExpectedError:  "",
	})
	return b
}

// AddMethodNotAllowed adds a test for wrong HTTP method
func (b *TestScenarioBuilder) AddMethodNotAllowed(path string, wrongMethod string) *TestScenarioBuilder {
	b.patterns = append(b.patterns, ErrorTestPattern{
		Name:        fmt.Sprintf("MethodNotAllowed_%s", wrongMethod),
		Description: fmt.Sprintf("Test handler with wrong HTTP method: %s", wrongMethod),
		Request: HTTPTestRequest{
			Method: wrongMethod,
			Path:   path,
		},
		ExpectedStatus: http.StatusMethodNotAllowed,
		ExpectedError:  "",
	})
	return b
}

// AddEmptyRequestBody adds a test for empty request body
func (b *TestScenarioBuilder) AddEmptyRequestBody(path string) *TestScenarioBuilder {
	b.patterns = append(b.patterns, ErrorTestPattern{
		Name:        "EmptyRequestBody",
		Description: "Test handler with empty request body",
		Request: HTTPTestRequest{
			Method: "POST",
			Path:   path,
			Body:   nil,
		},
		ExpectedStatus: http.StatusBadRequest,
		ExpectedError:  "",
	})
	return b
}

// Build returns the built error test patterns
func (b *TestScenarioBuilder) Build() []ErrorTestPattern {
	return b.patterns
}

// HandlerTestSuite provides a comprehensive test framework for HTTP handlers
type HandlerTestSuite struct {
	Name        string
	Environment *TestEnvironment
	Patterns    []ErrorTestPattern
}

// NewHandlerTestSuite creates a new handler test suite
func NewHandlerTestSuite(name string, env *TestEnvironment) *HandlerTestSuite {
	return &HandlerTestSuite{
		Name:        name,
		Environment: env,
		Patterns:    []ErrorTestPattern{},
	}
}

// WithPatterns adds error test patterns to the suite
func (suite *HandlerTestSuite) WithPatterns(patterns []ErrorTestPattern) *HandlerTestSuite {
	suite.Patterns = patterns
	return suite
}

// RunTests executes all error test patterns
func (suite *HandlerTestSuite) RunTests(t *testing.T) {
	for _, pattern := range suite.Patterns {
		t.Run(fmt.Sprintf("%s_%s", suite.Name, pattern.Name), func(t *testing.T) {
			// Setup
			var setupData interface{}
			if pattern.Setup != nil {
				setupData = pattern.Setup(t)
			}

			// Cleanup
			if pattern.Cleanup != nil {
				defer pattern.Cleanup(setupData)
			}

			// Execute request
			w, err := makeHTTPRequest(suite.Environment, pattern.Request)
			if err != nil {
				t.Fatalf("Failed to make HTTP request: %v", err)
			}

			// Validate status code
			if w.Code != pattern.ExpectedStatus {
				t.Errorf("Expected status %d, got %d. Body: %s",
					pattern.ExpectedStatus, w.Code, w.Body.String())
			}

			// Validate error message if specified
			if pattern.ExpectedError != "" {
				assertErrorResponse(t, w, pattern.ExpectedStatus, pattern.ExpectedError)
			}
		})
	}
}

// PerformanceTestSuite provides performance testing capabilities
type PerformanceTestSuite struct {
	Name        string
	Environment *TestEnvironment
	MaxDuration map[string]int // endpoint -> max milliseconds
}

// NewPerformanceTestSuite creates a new performance test suite
func NewPerformanceTestSuite(name string, env *TestEnvironment) *PerformanceTestSuite {
	return &PerformanceTestSuite{
		Name:        name,
		Environment: env,
		MaxDuration: make(map[string]int),
	}
}

// AddEndpoint adds an endpoint with maximum allowed duration
func (suite *PerformanceTestSuite) AddEndpoint(path string, maxMilliseconds int) *PerformanceTestSuite {
	suite.MaxDuration[path] = maxMilliseconds
	return suite
}

// RunLoadTest executes a load test on an endpoint
func (suite *PerformanceTestSuite) RunLoadTest(t *testing.T, endpoint string, requests int, concurrency int) {
	t.Run(fmt.Sprintf("LoadTest_%s_%dreqs_%dconcurrent", suite.Name, requests, concurrency), func(t *testing.T) {
		// This is a simplified load test
		// In production, you'd use more sophisticated tools

		done := make(chan bool, concurrency)
		errors := make(chan error, requests)

		startTime := measureRequestDuration(func() {
			for i := 0; i < requests; i++ {
				go func() {
					w, err := makeHTTPRequest(suite.Environment, HTTPTestRequest{
						Method: "GET",
						Path:   endpoint,
					})

					if err != nil {
						errors <- err
					} else if w.Code != http.StatusOK {
						errors <- fmt.Errorf("request failed with status: %d", w.Code)
					}

					done <- true
				}()
			}

			// Wait for all requests to complete
			for i := 0; i < requests; i++ {
				<-done
			}
		})

		// Check for errors
		close(errors)
		errorCount := 0
		for err := range errors {
			if err != nil {
				t.Logf("Request error: %v", err)
				errorCount++
			}
		}

		if errorCount > 0 {
			t.Errorf("Load test had %d errors out of %d requests", errorCount, requests)
		}

		avgDuration := startTime / time.Duration(requests)
		t.Logf("Load test completed: %d requests in %v (avg: %v per request)",
			requests, startTime, avgDuration)
	})
}

// IntegrationTestPattern defines patterns for integration testing
type IntegrationTestPattern struct {
	Name        string
	Description string
	Steps       []func(t *testing.T, env *TestEnvironment) error
	Cleanup     func(t *testing.T)
}

// NewIntegrationTestPattern creates a new integration test pattern
func NewIntegrationTestPattern(name, description string) *IntegrationTestPattern {
	return &IntegrationTestPattern{
		Name:        name,
		Description: description,
		Steps:       []func(t *testing.T, env *TestEnvironment) error{},
	}
}

// AddStep adds a test step to the integration pattern
func (p *IntegrationTestPattern) AddStep(step func(t *testing.T, env *TestEnvironment) error) *IntegrationTestPattern {
	p.Steps = append(p.Steps, step)
	return p
}

// WithCleanup adds a cleanup function
func (p *IntegrationTestPattern) WithCleanup(cleanup func(t *testing.T)) *IntegrationTestPattern {
	p.Cleanup = cleanup
	return p
}

// Run executes the integration test pattern
func (p *IntegrationTestPattern) Run(t *testing.T, env *TestEnvironment) {
	if p.Cleanup != nil {
		defer p.Cleanup(t)
	}

	for i, step := range p.Steps {
		if err := step(t, env); err != nil {
			t.Fatalf("Step %d failed: %v", i+1, err)
		}
	}
}

// Common validation helpers

// validateHealthResponse validates standard health check response
func validateHealthResponse(t *testing.T, w *httptest.ResponseRecorder) {
	t.Helper()

	response := assertJSONResponse(t, w, http.StatusOK)
	assertResponseHasField(t, response, "status")
	assertResponseHasField(t, response, "timestamp")
	assertResponseHasField(t, response, "service")

	if status, ok := response["status"].(string); ok {
		if status != "healthy" {
			t.Errorf("Expected status 'healthy', got '%s'", status)
		}
	}
}

// validateScenariosResponse validates scenarios list response
func validateScenariosResponse(t *testing.T, w *httptest.ResponseRecorder) {
	t.Helper()

	response := assertJSONResponse(t, w, http.StatusOK)
	assertResponseHasField(t, response, "scenarios")
	assertResponseHasField(t, response, "timestamp")
}

// validateHealthyScenariosResponse validates healthy scenarios response
func validateHealthyScenariosResponse(t *testing.T, w *httptest.ResponseRecorder) {
	t.Helper()

	response := assertJSONResponse(t, w, http.StatusOK)
	assertResponseHasField(t, response, "scenarios")
	assertResponseHasField(t, response, "categories")

	// Validate scenarios is an array
	if scenarios, ok := response["scenarios"].([]interface{}); ok {
		t.Logf("Found %d healthy scenarios", len(scenarios))
	} else {
		t.Error("Expected 'scenarios' to be an array")
	}
}
