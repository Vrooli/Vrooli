package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// TestScenarioBuilder provides fluent interface for building test scenarios
type TestScenarioBuilder struct {
	scenarios []ErrorTestScenario
}

// ErrorTestScenario defines a single error test case
type ErrorTestScenario struct {
	Name           string
	Method         string
	Path           string
	Body           interface{}
	ExpectedStatus int
	ExpectedError  string
	Setup          func(t *testing.T, env *TestEnvironment) interface{}
	Cleanup        func(setupData interface{})
}

// NewTestScenarioBuilder creates a new test scenario builder
func NewTestScenarioBuilder() *TestScenarioBuilder {
	return &TestScenarioBuilder{
		scenarios: make([]ErrorTestScenario, 0),
	}
}

// AddInvalidUUID adds a test for invalid UUID format
func (b *TestScenarioBuilder) AddInvalidUUID(path, method string) *TestScenarioBuilder {
	b.scenarios = append(b.scenarios, ErrorTestScenario{
		Name:           "InvalidUUID",
		Method:         method,
		Path:           path,
		ExpectedStatus: http.StatusBadRequest,
		ExpectedError:  "invalid UUID",
	})
	return b
}

// AddNonExistentSchema adds a test for non-existent schema
func (b *TestScenarioBuilder) AddNonExistentSchema(path, method string) *TestScenarioBuilder {
	b.scenarios = append(b.scenarios, ErrorTestScenario{
		Name:           "NonExistentSchema",
		Method:         method,
		Path:           path,
		ExpectedStatus: http.StatusNotFound,
		ExpectedError:  "not found",
		Setup: func(t *testing.T, env *TestEnvironment) interface{} {
			return uuid.New() // Use a valid but non-existent UUID
		},
	})
	return b
}

// AddInvalidJSON adds a test for malformed JSON
func (b *TestScenarioBuilder) AddInvalidJSON(path, method string) *TestScenarioBuilder {
	b.scenarios = append(b.scenarios, ErrorTestScenario{
		Name:           "InvalidJSON",
		Method:         method,
		Path:           path,
		Body:           `{"invalid": "json"`, // Malformed JSON
		ExpectedStatus: http.StatusBadRequest,
		ExpectedError:  "invalid",
	})
	return b
}

// AddMissingRequiredField adds a test for missing required fields
func (b *TestScenarioBuilder) AddMissingRequiredField(path, method string, body interface{}) *TestScenarioBuilder {
	b.scenarios = append(b.scenarios, ErrorTestScenario{
		Name:           "MissingRequiredField",
		Method:         method,
		Path:           path,
		Body:           body,
		ExpectedStatus: http.StatusBadRequest,
		ExpectedError:  "required",
	})
	return b
}

// AddEmptyBody adds a test for empty request body
func (b *TestScenarioBuilder) AddEmptyBody(path, method string) *TestScenarioBuilder {
	b.scenarios = append(b.scenarios, ErrorTestScenario{
		Name:           "EmptyBody",
		Method:         method,
		Path:           path,
		Body:           "",
		ExpectedStatus: http.StatusBadRequest,
		ExpectedError:  "",
	})
	return b
}

// AddCustomScenario adds a custom test scenario
func (b *TestScenarioBuilder) AddCustomScenario(scenario ErrorTestScenario) *TestScenarioBuilder {
	b.scenarios = append(b.scenarios, scenario)
	return b
}

// Build returns the built scenarios
func (b *TestScenarioBuilder) Build() []ErrorTestScenario {
	return b.scenarios
}

// RunScenarios executes all built test scenarios
func (b *TestScenarioBuilder) RunScenarios(t *testing.T, router *gin.Engine, env *TestEnvironment) {
	for _, scenario := range b.scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			var setupData interface{}
			if scenario.Setup != nil {
				setupData = scenario.Setup(t, env)
			}

			if scenario.Cleanup != nil {
				defer scenario.Cleanup(setupData)
			}

			// Build the path if setup data is UUID
			path := scenario.Path
			if setupData != nil {
				if id, ok := setupData.(uuid.UUID); ok {
					path = fmt.Sprintf(scenario.Path, id)
				}
			}

			req := HTTPTestRequest{
				Method: scenario.Method,
				Path:   path,
				Body:   scenario.Body,
			}

			w := makeHTTPRequest(router, req)

			if w.Code != scenario.ExpectedStatus {
				t.Errorf("Expected status %d, got %d. Body: %s",
					scenario.ExpectedStatus, w.Code, w.Body.String())
			}

			if scenario.ExpectedError != "" {
				body := w.Body.String()
				if body == "" {
					t.Errorf("Expected error message containing '%s', but got empty body",
						scenario.ExpectedError)
				}
			}
		})
	}
}

// HandlerTestSuite provides comprehensive handler testing
type HandlerTestSuite struct {
	Name   string
	Router *gin.Engine
	Env    *TestEnvironment
}

// NewHandlerTestSuite creates a new handler test suite
func NewHandlerTestSuite(name string, router *gin.Engine, env *TestEnvironment) *HandlerTestSuite {
	return &HandlerTestSuite{
		Name:   name,
		Router: router,
		Env:    env,
	}
}

// TestSuccessCase tests a successful handler execution
func (s *HandlerTestSuite) TestSuccessCase(t *testing.T, testCase SuccessTestCase) {
	t.Run(testCase.Name, func(t *testing.T) {
		var setupData interface{}
		if testCase.Setup != nil {
			setupData = testCase.Setup(t, s.Env)
		}

		if testCase.Cleanup != nil {
			defer testCase.Cleanup(setupData)
		}

		req := testCase.BuildRequest(setupData)
		w := makeHTTPRequest(s.Router, req)

		if w.Code != testCase.ExpectedStatus {
			t.Errorf("Expected status %d, got %d. Body: %s",
				testCase.ExpectedStatus, w.Code, w.Body.String())
		}

		if testCase.Validate != nil {
			testCase.Validate(t, w, setupData)
		}
	})
}

// TestErrorCases tests multiple error scenarios
func (s *HandlerTestSuite) TestErrorCases(t *testing.T, scenarios []ErrorTestScenario) {
	for _, scenario := range scenarios {
		s.testErrorScenario(t, scenario)
	}
}

func (s *HandlerTestSuite) testErrorScenario(t *testing.T, scenario ErrorTestScenario) {
	t.Run(scenario.Name, func(t *testing.T) {
		var setupData interface{}
		if scenario.Setup != nil {
			setupData = scenario.Setup(t, s.Env)
		}

		if scenario.Cleanup != nil {
			defer scenario.Cleanup(setupData)
		}

		path := scenario.Path
		if setupData != nil {
			if id, ok := setupData.(uuid.UUID); ok {
				path = fmt.Sprintf(scenario.Path, id)
			}
		}

		req := HTTPTestRequest{
			Method: scenario.Method,
			Path:   path,
			Body:   scenario.Body,
		}

		w := makeHTTPRequest(s.Router, req)

		if w.Code != scenario.ExpectedStatus {
			t.Errorf("Expected status %d, got %d. Body: %s",
				scenario.ExpectedStatus, w.Code, w.Body.String())
		}
	})
}

// SuccessTestCase defines a successful test case
type SuccessTestCase struct {
	Name           string
	ExpectedStatus int
	Setup          func(t *testing.T, env *TestEnvironment) interface{}
	BuildRequest   func(setupData interface{}) HTTPTestRequest
	Validate       func(t *testing.T, w *httptest.ResponseRecorder, setupData interface{})
	Cleanup        func(setupData interface{})
}

// PerformanceTestPattern defines performance testing
type PerformanceTestPattern struct {
	Name        string
	Description string
	MaxDuration time.Duration
	Iterations  int
	Setup       func(t *testing.T) interface{}
	Execute     func(t *testing.T, setupData interface{})
	Validate    func(t *testing.T, duration time.Duration, setupData interface{})
	Cleanup     func(setupData interface{})
}

// RunPerformanceTest executes a performance test pattern
func RunPerformanceTest(t *testing.T, pattern PerformanceTestPattern) {
	t.Run(pattern.Name, func(t *testing.T) {
		var setupData interface{}
		if pattern.Setup != nil {
			setupData = pattern.Setup(t)
		}

		if pattern.Cleanup != nil {
			defer pattern.Cleanup(setupData)
		}

		start := time.Now()

		for i := 0; i < pattern.Iterations; i++ {
			pattern.Execute(t, setupData)
		}

		duration := time.Since(start)

		if duration > pattern.MaxDuration {
			t.Errorf("Performance test exceeded max duration. Expected: %v, Got: %v",
				pattern.MaxDuration, duration)
		}

		if pattern.Validate != nil {
			pattern.Validate(t, duration, setupData)
		}

		avgDuration := duration / time.Duration(pattern.Iterations)
		t.Logf("%s: %d iterations in %v (avg: %v per iteration)",
			pattern.Name, pattern.Iterations, duration, avgDuration)
	})
}

// IntegrationTestPattern defines integration testing
type IntegrationTestPattern struct {
	Name        string
	Description string
	Timeout     time.Duration
	Setup       func(t *testing.T) interface{}
	Execute     func(t *testing.T, setupData interface{}) error
	Validate    func(t *testing.T, setupData interface{}, result error)
	Cleanup     func(setupData interface{})
}

// RunIntegrationTest executes an integration test pattern
func RunIntegrationTest(t *testing.T, pattern IntegrationTestPattern) {
	t.Run(pattern.Name, func(t *testing.T) {
		if pattern.Timeout > 0 {
			// Set a timeout for the test
			done := make(chan bool)
			go func() {
				time.Sleep(pattern.Timeout)
				select {
				case <-done:
				default:
					t.Errorf("Integration test timeout after %v", pattern.Timeout)
				}
			}()
			defer func() { done <- true }()
		}

		var setupData interface{}
		if pattern.Setup != nil {
			setupData = pattern.Setup(t)
		}

		if pattern.Cleanup != nil {
			defer pattern.Cleanup(setupData)
		}

		result := pattern.Execute(t, setupData)

		if pattern.Validate != nil {
			pattern.Validate(t, setupData, result)
		}
	})
}

// DatabaseTestPattern provides database-specific testing patterns
type DatabaseTestPattern struct {
	Name           string
	RequireCleanDB bool
	Setup          func(t *testing.T, db *TestDatabase) interface{}
	Execute        func(t *testing.T, db *TestDatabase, setupData interface{}) error
	Validate       func(t *testing.T, db *TestDatabase, setupData interface{}, result error)
	Cleanup        func(db *TestDatabase, setupData interface{})
}

// RunDatabaseTest executes a database test pattern
func RunDatabaseTest(t *testing.T, testDB *TestDatabase, pattern DatabaseTestPattern) {
	skipIfNoDatabase(t, testDB)

	t.Run(pattern.Name, func(t *testing.T) {
		var setupData interface{}
		if pattern.Setup != nil {
			setupData = pattern.Setup(t, testDB)
		}

		if pattern.Cleanup != nil {
			defer pattern.Cleanup(testDB, setupData)
		}

		result := pattern.Execute(t, testDB, setupData)

		if pattern.Validate != nil {
			pattern.Validate(t, testDB, setupData, result)
		}
	})
}
