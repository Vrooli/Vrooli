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
			w := testHandlerWithRequest(t, suite.Handler, *req)

			// Validate status code
			if w.Code != pattern.ExpectedStatus {
				t.Errorf("Expected status %d, got %d. Response: %s",
					pattern.ExpectedStatus, w.Code, w.Body.String())
			}

			// Additional validation
			if pattern.Validate != nil {
				pattern.Validate(t, w, setupData)
			}
		})
	}
}

// Common error patterns for chore-tracking

// invalidChoreIDPattern tests handlers with invalid chore ID
func invalidChoreIDPattern(urlPath string) ErrorTestPattern {
	return ErrorTestPattern{
		Name:           "InvalidChoreID",
		Description:    "Test handler with invalid chore ID format",
		ExpectedStatus: http.StatusBadRequest,
		Setup:          nil,
		Execute: func(t *testing.T, setupData interface{}) *HTTPTestRequest {
			return &HTTPTestRequest{
				Method:  "POST",
				Path:    urlPath,
				URLVars: map[string]string{"id": "invalid-id"},
			}
		},
	}
}

// nonExistentChorePattern tests handlers with non-existent chore ID
func nonExistentChorePattern(urlPath string) ErrorTestPattern {
	return ErrorTestPattern{
		Name:           "NonExistentChore",
		Description:    "Test handler with non-existent chore ID",
		ExpectedStatus: http.StatusInternalServerError,
		Setup: func(t *testing.T) interface{} {
			return setupTestDB(t)
		},
		Execute: func(t *testing.T, setupData interface{}) *HTTPTestRequest {
			return &HTTPTestRequest{
				Method:  "POST",
				Path:    urlPath,
				URLVars: map[string]string{"id": "999999"},
				Body:    map[string]int{"user_id": 999999},
			}
		},
		Cleanup: func(setupData interface{}) {
			if env, ok := setupData.(*TestEnvironment); ok {
				env.Cleanup()
			}
		},
	}
}

// invalidJSONPattern tests handlers with malformed JSON
func invalidJSONPattern(urlPath string) ErrorTestPattern {
	return ErrorTestPattern{
		Name:           "InvalidJSON",
		Description:    "Test handler with malformed JSON input",
		ExpectedStatus: http.StatusBadRequest,
		Setup:          nil,
		Execute: func(t *testing.T, setupData interface{}) *HTTPTestRequest {
			return &HTTPTestRequest{
				Method: "POST",
				Path:   urlPath,
				Body:   `{"invalid": "json"`,
			}
		},
	}
}

// emptyBodyPattern tests handlers with empty request body
func emptyBodyPattern(urlPath string) ErrorTestPattern {
	return ErrorTestPattern{
		Name:           "EmptyBody",
		Description:    "Test handler with empty request body",
		ExpectedStatus: http.StatusBadRequest,
		Setup:          nil,
		Execute: func(t *testing.T, setupData interface{}) *HTTPTestRequest {
			return &HTTPTestRequest{
				Method: "POST",
				Path:   urlPath,
				Body:   "",
			}
		},
	}
}

// insufficientPointsPattern tests reward redemption with insufficient points
func insufficientPointsPattern() ErrorTestPattern {
	return ErrorTestPattern{
		Name:           "InsufficientPoints",
		Description:    "Test reward redemption with insufficient points",
		ExpectedStatus: http.StatusBadRequest,
		Setup: func(t *testing.T) interface{} {
			env := setupTestDB(t)
			user := setupTestUser(t, env, "TestUser")
			reward := setupTestReward(t, env, "ExpensiveReward", 1000)
			return map[string]interface{}{
				"env":    env,
				"user":   user,
				"reward": reward,
			}
		},
		Execute: func(t *testing.T, setupData interface{}) *HTTPTestRequest {
			data := setupData.(map[string]interface{})
			user := data["user"].(*TestUser)
			reward := data["reward"].(*TestReward)

			return &HTTPTestRequest{
				Method:  "POST",
				Path:    fmt.Sprintf("/api/rewards/%d/redeem", reward.Reward.ID),
				URLVars: map[string]string{"id": fmt.Sprintf("%d", reward.Reward.ID)},
				Body:    map[string]int{"user_id": user.User.ID},
			}
		},
		Cleanup: func(setupData interface{}) {
			data := setupData.(map[string]interface{})
			if user, ok := data["user"].(*TestUser); ok {
				user.Cleanup()
			}
			if reward, ok := data["reward"].(*TestReward); ok {
				reward.Cleanup()
			}
			if env, ok := data["env"].(*TestEnvironment); ok {
				env.Cleanup()
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

// AddInvalidChoreID adds invalid chore ID test pattern
func (b *TestScenarioBuilder) AddInvalidChoreID(urlPath string) *TestScenarioBuilder {
	b.patterns = append(b.patterns, invalidChoreIDPattern(urlPath))
	return b
}

// AddNonExistentChore adds non-existent chore test pattern
func (b *TestScenarioBuilder) AddNonExistentChore(urlPath string) *TestScenarioBuilder {
	b.patterns = append(b.patterns, nonExistentChorePattern(urlPath))
	return b
}

// AddInvalidJSON adds invalid JSON test pattern
func (b *TestScenarioBuilder) AddInvalidJSON(urlPath string) *TestScenarioBuilder {
	b.patterns = append(b.patterns, invalidJSONPattern(urlPath))
	return b
}

// AddEmptyBody adds empty body test pattern
func (b *TestScenarioBuilder) AddEmptyBody(urlPath string) *TestScenarioBuilder {
	b.patterns = append(b.patterns, emptyBodyPattern(urlPath))
	return b
}

// AddInsufficientPoints adds insufficient points test pattern
func (b *TestScenarioBuilder) AddInsufficientPoints() *TestScenarioBuilder {
	b.patterns = append(b.patterns, insufficientPointsPattern())
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

// RunPerformanceTest executes a performance test and validates timing
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
			t.Errorf("Performance test exceeded max duration: %v > %v",
				duration, pattern.MaxDuration)
		} else {
			t.Logf("Performance test completed in %v (max: %v)",
				duration, pattern.MaxDuration)
		}
	})
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

// RunConcurrencyTest executes concurrent operations and validates results
func RunConcurrencyTest(t *testing.T, pattern ConcurrencyTestPattern) {
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

		// Execute concurrent operations
		results := make([]error, pattern.Iterations)
		done := make(chan struct{}, pattern.Concurrency)

		for i := 0; i < pattern.Iterations; i++ {
			done <- struct{}{}
			go func(iteration int) {
				defer func() { <-done }()
				results[iteration] = pattern.Execute(t, setupData, iteration)
			}(i)
		}

		// Wait for all to complete
		for i := 0; i < pattern.Concurrency; i++ {
			done <- struct{}{}
		}

		// Validate
		if pattern.Validate != nil {
			pattern.Validate(t, setupData, results)
		}
	})
}
