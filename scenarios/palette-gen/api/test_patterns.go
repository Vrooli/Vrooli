package main

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// ErrorTestCase defines a single error test scenario
type ErrorTestCase struct {
	Name           string
	Method         string
	Path           string
	Body           interface{}
	Handler        http.HandlerFunc
	ExpectedStatus int
	Description    string
}

// TestScenarioBuilder provides fluent interface for building test scenarios
type TestScenarioBuilder struct {
	scenarios []ErrorTestCase
}

// NewTestScenarioBuilder creates a new test scenario builder
func NewTestScenarioBuilder() *TestScenarioBuilder {
	return &TestScenarioBuilder{
		scenarios: make([]ErrorTestCase, 0),
	}
}

// AddMethodNotAllowed adds a method not allowed test case
func (b *TestScenarioBuilder) AddMethodNotAllowed(path string, invalidMethod string, handler http.HandlerFunc) *TestScenarioBuilder {
	b.scenarios = append(b.scenarios, ErrorTestCase{
		Name:           "MethodNotAllowed_" + invalidMethod,
		Method:         invalidMethod,
		Path:           path,
		Body:           nil,
		Handler:        handler,
		ExpectedStatus: http.StatusMethodNotAllowed,
		Description:    "Test " + invalidMethod + " method on endpoint that doesn't support it",
	})
	return b
}

// AddInvalidJSON adds an invalid JSON test case
func (b *TestScenarioBuilder) AddInvalidJSON(path string, handler http.HandlerFunc) *TestScenarioBuilder {
	b.scenarios = append(b.scenarios, ErrorTestCase{
		Name:           "InvalidJSON",
		Method:         "POST",
		Path:           path,
		Body:           `{"invalid": json}`, // Invalid JSON as string
		Handler:        handler,
		ExpectedStatus: http.StatusBadRequest,
		Description:    "Test malformed JSON input",
	})
	return b
}

// AddMissingRequiredField adds a missing required field test case
func (b *TestScenarioBuilder) AddMissingRequiredField(path string, body interface{}, handler http.HandlerFunc) *TestScenarioBuilder {
	b.scenarios = append(b.scenarios, ErrorTestCase{
		Name:           "MissingRequiredField",
		Method:         "POST",
		Path:           path,
		Body:           body,
		Handler:        handler,
		ExpectedStatus: http.StatusBadRequest,
		Description:    "Test request with missing required fields",
	})
	return b
}

// AddEmptyRequest adds an empty request body test case
func (b *TestScenarioBuilder) AddEmptyRequest(path string, handler http.HandlerFunc) *TestScenarioBuilder {
	b.scenarios = append(b.scenarios, ErrorTestCase{
		Name:           "EmptyRequestBody",
		Method:         "POST",
		Path:           path,
		Body:           map[string]interface{}{},
		Handler:        handler,
		ExpectedStatus: http.StatusBadRequest,
		Description:    "Test request with empty body",
	})
	return b
}

// Build returns the built test scenarios
func (b *TestScenarioBuilder) Build() []ErrorTestCase {
	return b.scenarios
}

// HandlerTestSuite provides comprehensive testing for HTTP handlers
type HandlerTestSuite struct {
	Name    string
	Handler http.HandlerFunc
}

// RunSuccessTests executes success path tests
func (s *HandlerTestSuite) RunSuccessTests(t *testing.T, testCases []SuccessTestCase) {
	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			cleanup := setupTestEnvironment(t)
			defer cleanup()

			w, err := makeHTTPRequest(tc.Method, tc.Path, tc.Body, tc.Handler)
			if err != nil {
				t.Fatalf("Failed to make HTTP request: %v", err)
			}

			if w.Code != tc.ExpectedStatus {
				t.Errorf("Expected status %d, got %d. Body: %s", tc.ExpectedStatus, w.Code, w.Body.String())
			}

			if tc.Validate != nil {
				tc.Validate(t, w)
			}
		})
	}
}

// RunErrorTests executes error path tests
func (s *HandlerTestSuite) RunErrorTests(t *testing.T, testCases []ErrorTestCase) {
	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			cleanup := setupTestEnvironment(t)
			defer cleanup()

			var w *httptest.ResponseRecorder
			var err error

			// Handle string body (for invalid JSON tests)
			if bodyStr, ok := tc.Body.(string); ok {
				req := httptest.NewRequest(tc.Method, tc.Path, bytes.NewBufferString(bodyStr))
				req.Header.Set("Content-Type", "application/json")
				w = httptest.NewRecorder()
				tc.Handler(w, req)
			} else {
				w, err = makeHTTPRequest(tc.Method, tc.Path, tc.Body, tc.Handler)
				if err != nil {
					t.Fatalf("Failed to make HTTP request: %v", err)
				}
			}

			assertErrorResponse(t, w, tc.ExpectedStatus)
		})
	}
}

// SuccessTestCase defines a single success test scenario
type SuccessTestCase struct {
	Name           string
	Method         string
	Path           string
	Body           interface{}
	Handler        http.HandlerFunc
	ExpectedStatus int
	Validate       func(t *testing.T, w *httptest.ResponseRecorder)
	Description    string
}

// PerformanceTestCase defines a performance test scenario
type PerformanceTestCase struct {
	Name          string
	Setup         func(t *testing.T)
	Execute       func(t *testing.T)
	Cleanup       func(t *testing.T)
	MaxDuration   time.Duration
	Description   string
	IterationCount int
}

// RunPerformanceTest executes a performance test
func RunPerformanceTest(t *testing.T, tc PerformanceTestCase) {
	t.Run(tc.Name, func(t *testing.T) {
		if tc.Setup != nil {
			tc.Setup(t)
		}

		if tc.Cleanup != nil {
			defer tc.Cleanup(t)
		}

		iterations := tc.IterationCount
		if iterations == 0 {
			iterations = 1
		}

		totalDuration := time.Duration(0)
		for i := 0; i < iterations; i++ {
			duration := measureExecutionTime(t, tc.Name, func() {
				tc.Execute(t)
			})
			totalDuration += duration
		}

		avgDuration := totalDuration / time.Duration(iterations)

		if tc.MaxDuration > 0 {
			assertExecutionTime(t, avgDuration, tc.MaxDuration, tc.Name)
		}

		t.Logf("Performance: %s - Avg: %v over %d iterations", tc.Name, avgDuration, iterations)
	})
}
