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

			// Validate
			if w.Code != pattern.ExpectedStatus {
				t.Errorf("Expected status %d, got %d. Response: %s", pattern.ExpectedStatus, w.Code, w.Body.String())
			}

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
	Cleanup     func(setupData interface{})
}

// RunPerformanceTests executes a suite of performance tests
func RunPerformanceTests(t *testing.T, patterns []PerformanceTestPattern) {
	for _, pattern := range patterns {
		t.Run(fmt.Sprintf("Performance_%s", pattern.Name), func(t *testing.T) {
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

			// Validate performance
			if duration > pattern.MaxDuration {
				t.Errorf("%s took %v, exceeds maximum %v", pattern.Name, duration, pattern.MaxDuration)
			} else {
				t.Logf("%s completed in %v (max: %v)", pattern.Name, duration, pattern.MaxDuration)
			}
		})
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

// AddInvalidUUID adds invalid UUID test pattern
func (b *TestScenarioBuilder) AddInvalidUUID(urlPath string, method string) *TestScenarioBuilder {
	b.patterns = append(b.patterns, ErrorTestPattern{
		Name:           "InvalidUUID",
		Description:    "Test handler with invalid UUID format",
		ExpectedStatus: http.StatusBadRequest,
		Setup:          nil,
		Execute: func(t *testing.T, setupData interface{}) *HTTPTestRequest {
			return &HTTPTestRequest{
				Method:  method,
				Path:    urlPath,
				URLVars: map[string]string{"id": "invalid-uuid"},
			}
		},
	})
	return b
}

// AddNonExistentTask adds non-existent task test pattern
func (b *TestScenarioBuilder) AddNonExistentTask(urlPath string, method string) *TestScenarioBuilder {
	b.patterns = append(b.patterns, ErrorTestPattern{
		Name:           "NonExistentTask",
		Description:    "Test handler with non-existent task ID",
		ExpectedStatus: http.StatusNotFound,
		Setup: func(t *testing.T) interface{} {
			return setupTestDirectory(t)
		},
		Execute: func(t *testing.T, setupData interface{}) *HTTPTestRequest {
			nonExistentID := uuid.New()
			req := &HTTPTestRequest{
				Method:  method,
				Path:    urlPath,
				URLVars: map[string]string{"id": nonExistentID.String()},
			}
			// For PUT/POST methods, include valid JSON body
			if method == "PUT" || method == "POST" {
				req.Body = map[string]interface{}{"title": "Test"}
			}
			return req
		},
		Cleanup: func(setupData interface{}) {
			if env, ok := setupData.(*TestEnvironment); ok {
				env.Cleanup()
			}
		},
	})
	return b
}

// AddInvalidJSON adds invalid JSON test pattern
func (b *TestScenarioBuilder) AddInvalidJSON(urlPath string, method string) *TestScenarioBuilder {
	b.patterns = append(b.patterns, ErrorTestPattern{
		Name:           "InvalidJSON",
		Description:    "Test handler with malformed JSON input",
		ExpectedStatus: http.StatusBadRequest,
		Setup: func(t *testing.T) interface{} {
			env := setupTestDirectory(t)
			return env
		},
		Execute: func(t *testing.T, setupData interface{}) *HTTPTestRequest {
			return &HTTPTestRequest{
				Method: method,
				Path:   urlPath,
				Body:   `{"invalid": "json"`, // Malformed JSON
			}
		},
		Cleanup: func(setupData interface{}) {
			if env, ok := setupData.(*TestEnvironment); ok {
				env.Cleanup()
			}
		},
	})
	return b
}

// AddMissingRequiredField adds missing required field test pattern
func (b *TestScenarioBuilder) AddMissingRequiredField(urlPath string, method string) *TestScenarioBuilder {
	b.patterns = append(b.patterns, ErrorTestPattern{
		Name:           "MissingRequiredField",
		Description:    "Test handler with missing required field",
		ExpectedStatus: http.StatusBadRequest,
		Setup: func(t *testing.T) interface{} {
			env := setupTestDirectory(t)
			return env
		},
		Execute: func(t *testing.T, setupData interface{}) *HTTPTestRequest {
			return &HTTPTestRequest{
				Method: method,
				Path:   urlPath,
				Body:   map[string]interface{}{"description": "Missing title"},
			}
		},
		Cleanup: func(setupData interface{}) {
			if env, ok := setupData.(*TestEnvironment); ok {
				env.Cleanup()
			}
		},
	})
	return b
}

// AddEmptyBody adds empty body test pattern
func (b *TestScenarioBuilder) AddEmptyBody(urlPath string, method string) *TestScenarioBuilder {
	b.patterns = append(b.patterns, ErrorTestPattern{
		Name:           "EmptyBody",
		Description:    "Test handler with empty request body",
		ExpectedStatus: http.StatusBadRequest,
		Setup: func(t *testing.T) interface{} {
			env := setupTestDirectory(t)
			return env
		},
		Execute: func(t *testing.T, setupData interface{}) *HTTPTestRequest {
			return &HTTPTestRequest{
				Method: method,
				Path:   urlPath,
				Body:   "",
			}
		},
		Cleanup: func(setupData interface{}) {
			if env, ok := setupData.(*TestEnvironment); ok {
				env.Cleanup()
			}
		},
	})
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

// Common performance test patterns

// CreateTaskPerformancePattern creates a performance test for task creation
func CreateTaskPerformancePattern(count int, maxDuration time.Duration) PerformanceTestPattern {
	return PerformanceTestPattern{
		Name:        fmt.Sprintf("CreateTask_%d", count),
		Description: fmt.Sprintf("Test creating %d tasks", count),
		MaxDuration: maxDuration,
		Setup: func(t *testing.T) interface{} {
			return setupTestDirectory(t)
		},
		Execute: func(t *testing.T, setupData interface{}) time.Duration {
			start := time.Now()
			for i := 0; i < count; i++ {
				task := &Task{
					ID:          uuid.New(),
					Title:       fmt.Sprintf("Performance Test Task %d", i),
					Description: "Performance testing",
					Status:      "pending",
					CreatedAt:   time.Now(),
					UpdatedAt:   time.Now(),
				}
				store.Create(task)
			}
			return time.Since(start)
		},
		Cleanup: func(setupData interface{}) {
			if env, ok := setupData.(*TestEnvironment); ok {
				env.Cleanup()
			}
		},
	}
}

// ListTasksPerformancePattern creates a performance test for listing tasks
func ListTasksPerformancePattern(taskCount int, maxDuration time.Duration) PerformanceTestPattern {
	return PerformanceTestPattern{
		Name:        fmt.Sprintf("ListTasks_With_%d_Tasks", taskCount),
		Description: fmt.Sprintf("Test listing with %d tasks", taskCount),
		MaxDuration: maxDuration,
		Setup: func(t *testing.T) interface{} {
			env := setupTestDirectory(t)
			// Create tasks
			for i := 0; i < taskCount; i++ {
				task := &Task{
					ID:          uuid.New(),
					Title:       fmt.Sprintf("Task %d", i),
					Description: "Test task",
					Status:      "pending",
					CreatedAt:   time.Now(),
					UpdatedAt:   time.Now(),
				}
				store.Create(task)
			}
			return env
		},
		Execute: func(t *testing.T, setupData interface{}) time.Duration {
			start := time.Now()
			store.List()
			return time.Since(start)
		},
		Cleanup: func(setupData interface{}) {
			if env, ok := setupData.(*TestEnvironment); ok {
				env.Cleanup()
			}
		},
	}
}
