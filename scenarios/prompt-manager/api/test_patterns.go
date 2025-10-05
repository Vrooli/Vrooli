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
	Execute        func(t *testing.T, setupData interface{}) HTTPTestRequest
	Validate       func(t *testing.T, w *httptest.ResponseRecorder, setupData interface{})
	Cleanup        func(setupData interface{})
}

// TestScenarioBuilder provides fluent interface for building test scenarios
type TestScenarioBuilder struct {
	scenarios []ErrorTestPattern
}

// NewTestScenarioBuilder creates a new test scenario builder
func NewTestScenarioBuilder() *TestScenarioBuilder {
	return &TestScenarioBuilder{
		scenarios: []ErrorTestPattern{},
	}
}

// AddInvalidUUID adds an invalid UUID test scenario
func (b *TestScenarioBuilder) AddInvalidUUID(endpoint string) *TestScenarioBuilder {
	b.scenarios = append(b.scenarios, ErrorTestPattern{
		Name:           "InvalidUUID",
		Description:    "Test with invalid UUID format",
		ExpectedStatus: http.StatusBadRequest,
		Execute: func(t *testing.T, setupData interface{}) HTTPTestRequest {
			return HTTPTestRequest{
				Method:  "GET",
				Path:    endpoint,
				URLVars: map[string]string{"id": "invalid-uuid-format"},
			}
		},
	})
	return b
}

// AddNonExistentCampaign adds a non-existent campaign test
func (b *TestScenarioBuilder) AddNonExistentCampaign(endpoint string) *TestScenarioBuilder {
	b.scenarios = append(b.scenarios, ErrorTestPattern{
		Name:           "NonExistentCampaign",
		Description:    "Test with non-existent campaign ID",
		ExpectedStatus: http.StatusNotFound,
		Execute: func(t *testing.T, setupData interface{}) HTTPTestRequest {
			return HTTPTestRequest{
				Method:  "GET",
				Path:    endpoint,
				URLVars: map[string]string{"id": uuid.New().String()},
			}
		},
	})
	return b
}

// AddNonExistentPrompt adds a non-existent prompt test
func (b *TestScenarioBuilder) AddNonExistentPrompt(endpoint string) *TestScenarioBuilder {
	b.scenarios = append(b.scenarios, ErrorTestPattern{
		Name:           "NonExistentPrompt",
		Description:    "Test with non-existent prompt ID",
		ExpectedStatus: http.StatusNotFound,
		Execute: func(t *testing.T, setupData interface{}) HTTPTestRequest {
			return HTTPTestRequest{
				Method:  "GET",
				Path:    endpoint,
				URLVars: map[string]string{"id": uuid.New().String()},
			}
		},
	})
	return b
}

// AddInvalidJSON adds an invalid JSON test scenario
func (b *TestScenarioBuilder) AddInvalidJSON(endpoint string, method string) *TestScenarioBuilder {
	b.scenarios = append(b.scenarios, ErrorTestPattern{
		Name:           "InvalidJSON",
		Description:    "Test with malformed JSON input",
		ExpectedStatus: http.StatusBadRequest,
		Execute: func(t *testing.T, setupData interface{}) HTTPTestRequest {
			return HTTPTestRequest{
				Method: method,
				Path:   endpoint,
				Body:   `{"invalid": "json"`, // Malformed JSON
			}
		},
	})
	return b
}

// AddMissingRequiredField adds a missing required field test
func (b *TestScenarioBuilder) AddMissingRequiredField(endpoint string, method string) *TestScenarioBuilder {
	b.scenarios = append(b.scenarios, ErrorTestPattern{
		Name:           "MissingRequiredField",
		Description:    "Test with missing required field",
		ExpectedStatus: http.StatusBadRequest,
		Execute: func(t *testing.T, setupData interface{}) HTTPTestRequest {
			return HTTPTestRequest{
				Method: method,
				Path:   endpoint,
				Body:   map[string]interface{}{}, // Empty body
			}
		},
	})
	return b
}

// AddEmptySearchQuery adds an empty search query test
func (b *TestScenarioBuilder) AddEmptySearchQuery(endpoint string) *TestScenarioBuilder {
	b.scenarios = append(b.scenarios, ErrorTestPattern{
		Name:           "EmptySearchQuery",
		Description:    "Test search with empty query",
		ExpectedStatus: http.StatusBadRequest,
		Execute: func(t *testing.T, setupData interface{}) HTTPTestRequest {
			return HTTPTestRequest{
				Method: "GET",
				Path:   endpoint + "?q=",
			}
		},
	})
	return b
}

// Build returns the built scenarios
func (b *TestScenarioBuilder) Build() []ErrorTestPattern {
	return b.scenarios
}

// HandlerTestSuite provides a comprehensive test framework for HTTP handlers
type HandlerTestSuite struct {
	Name        string
	Description string
	Setup       func(t *testing.T) interface{}
	Tests       []HandlerTest
	Cleanup     func(setupData interface{})
}

// HandlerTest defines a single handler test case
type HandlerTest struct {
	Name           string
	Request        HTTPTestRequest
	ExpectedStatus int
	ValidateBody   func(t *testing.T, body string)
}

// RunSuite executes all tests in the suite
func (suite *HandlerTestSuite) RunSuite(t *testing.T, server *APIServer, router *mux.Router) {
	t.Run(suite.Name, func(t *testing.T) {
		var setupData interface{}
		if suite.Setup != nil {
			setupData = suite.Setup(t)
		}

		if suite.Cleanup != nil {
			defer suite.Cleanup(setupData)
		}

		for _, test := range suite.Tests {
			t.Run(test.Name, func(t *testing.T) {
				w := makeHTTPRequest(server, router, test.Request)

				if w.Code != test.ExpectedStatus {
					t.Errorf("Expected status %d, got %d. Body: %s",
						test.ExpectedStatus, w.Code, w.Body.String())
				}

				if test.ValidateBody != nil {
					test.ValidateBody(t, w.Body.String())
				}
			})
		}
	})
}

// PerformanceTestPattern defines performance testing scenarios
type PerformanceTestPattern struct {
	Name         string
	Description  string
	MaxDuration  time.Duration
	Iterations   int
	Setup        func(t *testing.T) interface{}
	Execute      func(t *testing.T, iteration int, setupData interface{}) HTTPTestRequest
	Validate     func(t *testing.T, avgDuration time.Duration, setupData interface{})
	Cleanup      func(setupData interface{})
}

// RunPerformanceTest executes a performance test pattern
func RunPerformanceTest(t *testing.T, server *APIServer, router *mux.Router, pattern PerformanceTestPattern) {
	t.Run(fmt.Sprintf("Performance_%s", pattern.Name), func(t *testing.T) {
		var setupData interface{}
		if pattern.Setup != nil {
			setupData = pattern.Setup(t)
		}

		if pattern.Cleanup != nil {
			defer pattern.Cleanup(setupData)
		}

		var totalDuration time.Duration
		successCount := 0

		for i := 0; i < pattern.Iterations; i++ {
			req := pattern.Execute(t, i, setupData)

			start := time.Now()
			w := makeHTTPRequest(server, router, req)
			duration := time.Since(start)

			if w.Code >= 200 && w.Code < 300 {
				totalDuration += duration
				successCount++
			}
		}

		if successCount == 0 {
			t.Fatal("No successful requests in performance test")
		}

		avgDuration := totalDuration / time.Duration(successCount)

		if avgDuration > pattern.MaxDuration {
			t.Errorf("Average duration %v exceeds maximum %v", avgDuration, pattern.MaxDuration)
		}

		if pattern.Validate != nil {
			pattern.Validate(t, avgDuration, setupData)
		}

		t.Logf("%s: %d iterations, avg %v, max allowed %v",
			pattern.Name, successCount, avgDuration, pattern.MaxDuration)
	})
}

// EdgeCasePattern defines edge case test scenarios
type EdgeCasePattern struct {
	Name        string
	Description string
	Test        func(t *testing.T, server *APIServer, router *mux.Router)
}

// RunEdgeCaseTests executes a suite of edge case tests
func RunEdgeCaseTests(t *testing.T, server *APIServer, router *mux.Router, patterns []EdgeCasePattern) {
	for _, pattern := range patterns {
		t.Run(fmt.Sprintf("EdgeCase_%s", pattern.Name), func(t *testing.T) {
			pattern.Test(t, server, router)
		})
	}
}
