// +build testing

package main

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
)

// TestScenarioBuilder provides a fluent interface for building test scenarios
type TestScenarioBuilder struct {
	scenarios []HTTPTestRequest
}

// NewTestScenarioBuilder creates a new test scenario builder
func NewTestScenarioBuilder() *TestScenarioBuilder {
	return &TestScenarioBuilder{
		scenarios: make([]HTTPTestRequest, 0),
	}
}

// AddInvalidJSON adds a test for invalid JSON body
func (b *TestScenarioBuilder) AddInvalidJSON(path, method string) *TestScenarioBuilder {
	// This will be handled by creating a raw request with invalid JSON
	b.scenarios = append(b.scenarios, HTTPTestRequest{
		Method: method,
		Path:   path,
		Body:   nil, // Will be replaced with invalid JSON in test
	})
	return b
}

// AddMissingRequiredField adds a test for missing required fields
func (b *TestScenarioBuilder) AddMissingRequiredField(path, method string, body interface{}) *TestScenarioBuilder {
	b.scenarios = append(b.scenarios, HTTPTestRequest{
		Method: method,
		Path:   path,
		Body:   body,
	})
	return b
}

// AddNonExistentItem adds a test for non-existent item
func (b *TestScenarioBuilder) AddNonExistentItem(path, method string) *TestScenarioBuilder {
	nonExistentID := uuid.New().String()
	b.scenarios = append(b.scenarios, HTTPTestRequest{
		Method: method,
		Path:   path,
		Body: map[string]interface{}{
			"item_external_id": nonExistentID,
			"scenario_id":      "test-scenario",
		},
	})
	return b
}

// AddNonExistentUser adds a test for non-existent user
func (b *TestScenarioBuilder) AddNonExistentUser(path, method string) *TestScenarioBuilder {
	nonExistentID := uuid.New().String()
	b.scenarios = append(b.scenarios, HTTPTestRequest{
		Method: method,
		Path:   path,
		Body: map[string]interface{}{
			"user_id":     nonExistentID,
			"scenario_id": "test-scenario",
		},
	})
	return b
}

// Build returns the built scenarios
func (b *TestScenarioBuilder) Build() []HTTPTestRequest {
	return b.scenarios
}

// ErrorTestPattern defines a systematic approach to testing error conditions
type ErrorTestPattern struct {
	Name           string
	Description    string
	ExpectedStatus int
	Setup          func(t *testing.T, env *TestEnvironment) interface{}
	Execute        func(t *testing.T, env *TestEnvironment, setupData interface{}) *httptest.ResponseRecorder
	Validate       func(t *testing.T, w *httptest.ResponseRecorder, setupData interface{})
	Cleanup        func(env *TestEnvironment, setupData interface{})
}

// RunErrorPattern executes a single error test pattern
func RunErrorPattern(t *testing.T, env *TestEnvironment, pattern ErrorTestPattern) {
	t.Run(pattern.Name, func(t *testing.T) {
		// Setup
		var setupData interface{}
		if pattern.Setup != nil {
			setupData = pattern.Setup(t, env)
		}

		// Cleanup
		if pattern.Cleanup != nil {
			defer pattern.Cleanup(env, setupData)
		}

		// Execute
		w := pattern.Execute(t, env, setupData)

		// Validate status
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

// Common error patterns

// InvalidJSONPattern tests handlers with malformed JSON
func InvalidJSONPattern(path, method string) ErrorTestPattern {
	return ErrorTestPattern{
		Name:           "InvalidJSON",
		Description:    "Test handler with malformed JSON body",
		ExpectedStatus: http.StatusBadRequest,
		Execute: func(t *testing.T, env *TestEnvironment, setupData interface{}) *httptest.ResponseRecorder {
			req, _ := http.NewRequest(method, path, bytes.NewBufferString("{invalid json"))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			env.Router.ServeHTTP(w, req)
			return w
		},
	}
}

// MissingRequiredFieldPattern tests handlers with missing required fields
func MissingRequiredFieldPattern(path, method string, body interface{}) ErrorTestPattern {
	return ErrorTestPattern{
		Name:           "MissingRequiredField",
		Description:    "Test handler with missing required fields",
		ExpectedStatus: http.StatusBadRequest,
		Execute: func(t *testing.T, env *TestEnvironment, setupData interface{}) *httptest.ResponseRecorder {
			return makeHTTPRequest(env.Router, HTTPTestRequest{
				Method: method,
				Path:   path,
				Body:   body,
			})
		},
	}
}

// EmptyBodyPattern tests handlers with empty request body
func EmptyBodyPattern(path, method string) ErrorTestPattern {
	return ErrorTestPattern{
		Name:           "EmptyBody",
		Description:    "Test handler with empty request body",
		ExpectedStatus: http.StatusBadRequest,
		Execute: func(t *testing.T, env *TestEnvironment, setupData interface{}) *httptest.ResponseRecorder {
			req, _ := http.NewRequest(method, path, bytes.NewBufferString(""))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			env.Router.ServeHTTP(w, req)
			return w
		},
	}
}

// NonExistentResourcePattern tests handlers with non-existent resources
func NonExistentResourcePattern(path, method string, body interface{}) ErrorTestPattern {
	return ErrorTestPattern{
		Name:           "NonExistentResource",
		Description:    "Test handler with non-existent resource",
		ExpectedStatus: http.StatusNotFound,
		Execute: func(t *testing.T, env *TestEnvironment, setupData interface{}) *httptest.ResponseRecorder {
			return makeHTTPRequest(env.Router, HTTPTestRequest{
				Method: method,
				Path:   path,
				Body:   body,
			})
		},
	}
}

// PerformanceTestPattern tests handler performance
type PerformanceTestPattern struct {
	Name              string
	Description       string
	MaxDuration       time.Duration
	RequestCount      int
	ConcurrentWorkers int
	Setup             func(t *testing.T, env *TestEnvironment) interface{}
	Execute           func(t *testing.T, env *TestEnvironment, setupData interface{}) *httptest.ResponseRecorder
	Cleanup           func(env *TestEnvironment, setupData interface{})
}

// RunPerformanceTest executes a performance test pattern
func RunPerformanceTest(t *testing.T, env *TestEnvironment, pattern PerformanceTestPattern) {
	t.Run(pattern.Name, func(t *testing.T) {
		// Setup
		var setupData interface{}
		if pattern.Setup != nil {
			setupData = pattern.Setup(t, env)
		}

		// Cleanup
		if pattern.Cleanup != nil {
			defer pattern.Cleanup(env, setupData)
		}

		// Execute performance test
		start := time.Now()
		var wg sync.WaitGroup

		results := make(chan time.Duration, pattern.RequestCount)

		workers := pattern.ConcurrentWorkers
		if workers == 0 {
			workers = 1
		}

		requestsPerWorker := pattern.RequestCount / workers

		for i := 0; i < workers; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for j := 0; j < requestsPerWorker; j++ {
					reqStart := time.Now()
					pattern.Execute(t, env, setupData)
					results <- time.Since(reqStart)
				}
			}()
		}

		wg.Wait()
		close(results)
		totalDuration := time.Since(start)

		// Calculate statistics
		var totalReqDuration time.Duration
		var maxReqDuration time.Duration
		count := 0

		for duration := range results {
			totalReqDuration += duration
			if duration > maxReqDuration {
				maxReqDuration = duration
			}
			count++
		}

		avgDuration := totalReqDuration / time.Duration(count)

		t.Logf("Performance results for %s:", pattern.Name)
		t.Logf("  Total time: %v", totalDuration)
		t.Logf("  Requests: %d", count)
		t.Logf("  Avg request time: %v", avgDuration)
		t.Logf("  Max request time: %v", maxReqDuration)
		t.Logf("  Requests/sec: %.2f", float64(count)/totalDuration.Seconds())

		if totalDuration > pattern.MaxDuration {
			t.Errorf("Performance test exceeded max duration. Expected: %v, Got: %v",
				pattern.MaxDuration, totalDuration)
		}
	})
}
