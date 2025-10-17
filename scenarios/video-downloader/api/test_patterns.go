// +build testing

package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
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

// invalidIDPattern tests handlers with invalid ID formats
func invalidIDPattern(urlPath string) ErrorTestPattern {
	return ErrorTestPattern{
		Name:           "InvalidID",
		Description:    "Test handler with invalid ID format",
		ExpectedStatus: http.StatusBadRequest,
		Setup:          nil,
		Execute: func(t *testing.T, setupData interface{}) *HTTPTestRequest {
			return &HTTPTestRequest{
				Method:  "GET",
				Path:    urlPath,
				URLVars: map[string]string{"id": "invalid-id", "download_id": "invalid-id"},
			}
		},
	}
}

// nonExistentResourcePattern tests handlers with non-existent resource IDs
func nonExistentResourcePattern(urlPath string, idVar string) ErrorTestPattern {
	return ErrorTestPattern{
		Name:           "NonExistentResource",
		Description:    "Test handler with non-existent resource ID",
		ExpectedStatus: http.StatusNotFound,
		Setup:          nil,
		Execute: func(t *testing.T, setupData interface{}) *HTTPTestRequest {
			urlVars := map[string]string{idVar: "999999"}
			return &HTTPTestRequest{
				Method:  "GET",
				Path:    urlPath,
				URLVars: urlVars,
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
func missingRequiredFieldPattern(method, urlPath string, missingField string) ErrorTestPattern {
	return ErrorTestPattern{
		Name:           fmt.Sprintf("Missing_%s", missingField),
		Description:    fmt.Sprintf("Test handler with missing %s field", missingField),
		ExpectedStatus: http.StatusBadRequest,
		Setup:          nil,
		Execute: func(t *testing.T, setupData interface{}) *HTTPTestRequest {
			// Create request with missing field
			body := map[string]interface{}{}
			return &HTTPTestRequest{
				Method: method,
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

// AddInvalidID adds invalid ID test pattern
func (b *TestScenarioBuilder) AddInvalidID(urlPath string) *TestScenarioBuilder {
	b.patterns = append(b.patterns, invalidIDPattern(urlPath))
	return b
}

// AddNonExistentResource adds non-existent resource test pattern
func (b *TestScenarioBuilder) AddNonExistentResource(urlPath string, idVar string) *TestScenarioBuilder {
	b.patterns = append(b.patterns, nonExistentResourcePattern(urlPath, idVar))
	return b
}

// AddInvalidJSON adds invalid JSON test pattern
func (b *TestScenarioBuilder) AddInvalidJSON(method, urlPath string) *TestScenarioBuilder {
	b.patterns = append(b.patterns, invalidJSONPattern(method, urlPath))
	return b
}

// AddMissingRequiredField adds missing field test pattern
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

// Performance test utilities

// BenchmarkHandler provides performance testing for handlers
func BenchmarkHandler(b *testing.B, handler http.HandlerFunc, req HTTPTestRequest) {
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		w, httpReq, err := makeHTTPRequest(req)
		if err != nil {
			b.Fatalf("Failed to create request: %v", err)
		}
		handler(w, httpReq)
	}
}

// Concurrency test utilities

// ConcurrentHandlerTest tests handler under concurrent load
func ConcurrentHandlerTest(t *testing.T, handler http.HandlerFunc, req HTTPTestRequest, concurrency int) {
	errors := make(chan error, concurrency)

	for i := 0; i < concurrency; i++ {
		go func(iteration int) {
			w, httpReq, err := makeHTTPRequest(req)
			if err != nil {
				errors <- fmt.Errorf("iteration %d: failed to create request: %v", iteration, err)
				return
			}

			handler(w, httpReq)

			if w.Code >= 500 {
				errors <- fmt.Errorf("iteration %d: got server error: %d", iteration, w.Code)
				return
			}

			errors <- nil
		}(i)
	}

	// Collect results
	for i := 0; i < concurrency; i++ {
		if err := <-errors; err != nil {
			t.Error(err)
		}
	}
}
