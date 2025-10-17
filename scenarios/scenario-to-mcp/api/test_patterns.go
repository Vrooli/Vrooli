// +build testing

package main

import (
	"fmt"
	"net/http"
	"testing"
	"time"
)

// TestScenarioBuilder provides fluent interface for building test scenarios
type TestScenarioBuilder struct {
	scenarios []HTTPTestRequest
}

// NewTestScenarioBuilder creates a new scenario builder
func NewTestScenarioBuilder() *TestScenarioBuilder {
	return &TestScenarioBuilder{
		scenarios: []HTTPTestRequest{},
	}
}

// AddInvalidJSON adds test for invalid JSON input
func (b *TestScenarioBuilder) AddInvalidJSON(path string) *TestScenarioBuilder {
	b.scenarios = append(b.scenarios, HTTPTestRequest{
		Method: "POST",
		Path:   path,
		Body:   `{"invalid": json`,
	})
	return b
}

// AddMissingField adds test for missing required field
func (b *TestScenarioBuilder) AddMissingField(path string, body interface{}) *TestScenarioBuilder {
	b.scenarios = append(b.scenarios, HTTPTestRequest{
		Method: "POST",
		Path:   path,
		Body:   body,
	})
	return b
}

// AddNonExistentResource adds test for non-existent resource
func (b *TestScenarioBuilder) AddNonExistentResource(path string, resourceID string) *TestScenarioBuilder {
	b.scenarios = append(b.scenarios, HTTPTestRequest{
		Method:  "GET",
		Path:    path,
		URLVars: map[string]string{"id": resourceID, "name": resourceID},
	})
	return b
}

// Build returns all built scenarios
func (b *TestScenarioBuilder) Build() []HTTPTestRequest {
	return b.scenarios
}

// ErrorTestPattern defines systematic error testing
type ErrorTestPattern struct {
	Name           string
	Description    string
	Request        HTTPTestRequest
	ExpectedStatus int
	ValidateError  func(t *testing.T, response map[string]interface{})
}

// HandlerTestSuite provides comprehensive handler testing
type HandlerTestSuite struct {
	Name           string
	TestServer     *TestServer
	SuccessTests   []SuccessTestCase
	ErrorTests     []ErrorTestPattern
	EdgeCaseTests  []EdgeCaseTest
}

// SuccessTestCase defines a successful operation test
type SuccessTestCase struct {
	Name           string
	Request        HTTPTestRequest
	ExpectedStatus int
	Validate       func(t *testing.T, response map[string]interface{})
}

// EdgeCaseTest defines edge case testing
type EdgeCaseTest struct {
	Name           string
	Setup          func(t *testing.T) interface{}
	Request        HTTPTestRequest
	ExpectedStatus int
	Validate       func(t *testing.T, response map[string]interface{}, setupData interface{})
	Cleanup        func(setupData interface{})
}

// RunAllTests executes complete test suite
func (suite *HandlerTestSuite) RunAllTests(t *testing.T) {
	t.Run(suite.Name+"_Success", func(t *testing.T) {
		suite.runSuccessTests(t)
	})

	t.Run(suite.Name+"_Errors", func(t *testing.T) {
		suite.runErrorTests(t)
	})

	t.Run(suite.Name+"_EdgeCases", func(t *testing.T) {
		suite.runEdgeCaseTests(t)
	})
}

func (suite *HandlerTestSuite) runSuccessTests(t *testing.T) {
	for _, test := range suite.SuccessTests {
		t.Run(test.Name, func(t *testing.T) {
			w := makeHTTPRequest(suite.TestServer, test.Request)

			if w.Code != test.ExpectedStatus {
				t.Errorf("Expected status %d, got %d. Body: %s",
					test.ExpectedStatus, w.Code, w.Body.String())
				return
			}

			if test.Validate != nil {
				assertJSONResponse(t, w, test.ExpectedStatus, func(resp map[string]interface{}) error {
					test.Validate(t, resp)
					return nil
				})
			}
		})
	}
}

func (suite *HandlerTestSuite) runErrorTests(t *testing.T) {
	for _, test := range suite.ErrorTests {
		t.Run(test.Name, func(t *testing.T) {
			w := makeHTTPRequest(suite.TestServer, test.Request)

			if w.Code != test.ExpectedStatus {
				t.Errorf("Expected status %d, got %d", test.ExpectedStatus, w.Code)
				return
			}

			if test.ValidateError != nil {
				assertJSONResponse(t, w, test.ExpectedStatus, func(resp map[string]interface{}) error {
					test.ValidateError(t, resp)
					return nil
				})
			}
		})
	}
}

func (suite *HandlerTestSuite) runEdgeCaseTests(t *testing.T) {
	for _, test := range suite.EdgeCaseTests {
		t.Run(test.Name, func(t *testing.T) {
			var setupData interface{}
			if test.Setup != nil {
				setupData = test.Setup(t)
			}

			if test.Cleanup != nil {
				defer test.Cleanup(setupData)
			}

			w := makeHTTPRequest(suite.TestServer, test.Request)

			if w.Code != test.ExpectedStatus {
				t.Errorf("Expected status %d, got %d", test.ExpectedStatus, w.Code)
				return
			}

			if test.Validate != nil {
				assertJSONResponse(t, w, test.ExpectedStatus, func(resp map[string]interface{}) error {
					test.Validate(t, resp, setupData)
					return nil
				})
			}
		})
	}
}

// PerformanceTestPattern defines performance testing
type PerformanceTestPattern struct {
	Name         string
	Description  string
	Request      HTTPTestRequest
	MaxDuration  time.Duration
	MinThroughput int // requests per second
}

// RunPerformanceTest executes performance test
func RunPerformanceTest(t *testing.T, ts *TestServer, pattern PerformanceTestPattern) {
	t.Run(pattern.Name, func(t *testing.T) {
		start := time.Now()
		w := makeHTTPRequest(ts, pattern.Request)
		duration := time.Since(start)

		if w.Code != http.StatusOK {
			t.Errorf("Request failed with status %d", w.Code)
			return
		}

		if duration > pattern.MaxDuration {
			t.Errorf("Request took %v, expected max %v", duration, pattern.MaxDuration)
		}

		t.Logf("✓ %s completed in %v", pattern.Name, duration)
	})
}

// LoadTestPattern defines load testing scenarios
type LoadTestPattern struct {
	Name           string
	Description    string
	Request        HTTPTestRequest
	Concurrency    int
	TotalRequests  int
	MaxAvgDuration time.Duration
}

// RunLoadTest executes load test
func RunLoadTest(t *testing.T, ts *TestServer, pattern LoadTestPattern) {
	t.Run(pattern.Name, func(t *testing.T) {
		if testing.Short() {
			t.Skip("Skipping load test in short mode")
		}

		results := make(chan time.Duration, pattern.TotalRequests)
		errors := make(chan error, pattern.TotalRequests)

		start := time.Now()

		// Simulate concurrent requests
		for i := 0; i < pattern.Concurrency; i++ {
			go func() {
				for j := 0; j < pattern.TotalRequests/pattern.Concurrency; j++ {
					reqStart := time.Now()
					w := makeHTTPRequest(ts, pattern.Request)
					reqDuration := time.Since(reqStart)

					if w.Code != http.StatusOK {
						errors <- fmt.Errorf("request failed with status %d", w.Code)
					}
					results <- reqDuration
				}
			}()
		}

		// Collect results
		totalDuration := time.Duration(0)
		errorCount := 0

		for i := 0; i < pattern.TotalRequests; i++ {
			select {
			case d := <-results:
				totalDuration += d
			case <-errors:
				errorCount++
			}
		}

		avgDuration := totalDuration / time.Duration(pattern.TotalRequests)
		totalTime := time.Since(start)

		if errorCount > 0 {
			t.Errorf("%d requests failed out of %d", errorCount, pattern.TotalRequests)
		}

		if avgDuration > pattern.MaxAvgDuration {
			t.Errorf("Average duration %v exceeds max %v", avgDuration, pattern.MaxAvgDuration)
		}

		t.Logf("✓ %s: %d requests in %v (avg: %v, errors: %d)",
			pattern.Name, pattern.TotalRequests, totalTime, avgDuration, errorCount)
	})
}

// IntegrationTestPattern defines integration testing scenarios
type IntegrationTestPattern struct {
	Name        string
	Description string
	Steps       []IntegrationStep
}

// IntegrationStep represents a step in integration test
type IntegrationStep struct {
	Name     string
	Request  HTTPTestRequest
	Validate func(t *testing.T, response map[string]interface{}) interface{}
}

// RunIntegrationTest executes multi-step integration test
func RunIntegrationTest(t *testing.T, ts *TestServer, pattern IntegrationTestPattern) {
	t.Run(pattern.Name, func(t *testing.T) {
		for _, step := range pattern.Steps {
			t.Run(step.Name, func(t *testing.T) {
				w := makeHTTPRequest(ts, step.Request)

				if w.Code >= 400 {
					t.Errorf("Step failed with status %d: %s", w.Code, w.Body.String())
					t.FailNow()
				}

				if step.Validate != nil {
					assertJSONResponse(t, w, http.StatusOK, func(resp map[string]interface{}) error {
						_ = step.Validate(t, resp)
						return nil
					})
				}
			})

			if t.Failed() {
				t.Logf("Integration test stopped at step: %s", step.Name)
				break
			}
		}
	})
}
