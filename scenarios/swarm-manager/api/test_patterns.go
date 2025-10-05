// +build testing

package main

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// ErrorTestPattern defines a systematic approach to testing error conditions
type ErrorTestPattern struct {
	Name           string
	Description    string
	ExpectedStatus int
	Setup          func(t *testing.T) interface{}
	Execute        func(t *testing.T, setupData interface{}) *FiberTestRequest
	Validate       func(t *testing.T, req *FiberTestRequest, setupData interface{})
	Cleanup        func(setupData interface{})
}

// HandlerTestSuite provides a comprehensive test framework for HTTP handlers
type HandlerTestSuite struct {
	HandlerName    string
	BaseURL        string
	RequiredParams []string
}

// RunErrorTests executes a suite of error condition tests
func (suite *HandlerTestSuite) RunErrorTests(t *testing.T, app *fiber.App, patterns []ErrorTestPattern) {
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
			w, err := makeFiberRequest(app, *req)
			if err != nil {
				t.Fatalf("Failed to execute request: %v", err)
			}

			// Validate status code
			if w.Code != pattern.ExpectedStatus {
				t.Errorf("Expected status %d, got %d. Response: %s",
					pattern.ExpectedStatus, w.Code, w.Body.String())
			}

			// Additional validation
			if pattern.Validate != nil {
				pattern.Validate(t, req, setupData)
			}
		})
	}
}

// Common error patterns that can be reused across handlers

// invalidUUIDPattern tests handlers with invalid UUID formats
func invalidUUIDPattern(urlPath string) ErrorTestPattern {
	return ErrorTestPattern{
		Name:           "InvalidUUID",
		Description:    "Test handler with invalid UUID format",
		ExpectedStatus: http.StatusBadRequest,
		Setup:          nil,
		Execute: func(t *testing.T, setupData interface{}) *FiberTestRequest {
			return &FiberTestRequest{
				Method: "GET",
				Path:   urlPath,
			}
		},
		Validate: func(t *testing.T, req *FiberTestRequest, setupData interface{}) {
			// Additional validation can be added here
		},
	}
}

// nonExistentTaskPattern tests handlers with non-existent task IDs
func nonExistentTaskPattern(method, urlPathTemplate string) ErrorTestPattern {
	return ErrorTestPattern{
		Name:           "NonExistentTask",
		Description:    "Test handler with non-existent task ID",
		ExpectedStatus: http.StatusNotFound,
		Setup:          nil,
		Execute: func(t *testing.T, setupData interface{}) *FiberTestRequest {
			nonExistentID := uuid.New().String()
			path := fmt.Sprintf(urlPathTemplate, nonExistentID)
			return &FiberTestRequest{
				Method: method,
				Path:   path,
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
		Execute: func(t *testing.T, setupData interface{}) *FiberTestRequest {
			return &FiberTestRequest{
				Method: method,
				Path:   urlPath,
				Body:   `{"invalid": "json"`, // Malformed JSON
			}
		},
		Cleanup: nil,
	}
}

// missingRequiredFieldPattern tests handlers with missing required fields
func missingRequiredFieldPattern(method, urlPath string, fieldName string) ErrorTestPattern {
	return ErrorTestPattern{
		Name:           fmt.Sprintf("MissingField_%s", fieldName),
		Description:    fmt.Sprintf("Test handler with missing required field: %s", fieldName),
		ExpectedStatus: http.StatusBadRequest,
		Setup:          nil,
		Execute: func(t *testing.T, setupData interface{}) *FiberTestRequest {
			body := map[string]interface{}{
				// Intentionally missing the required field
				"other_field": "value",
			}
			return &FiberTestRequest{
				Method: method,
				Path:   urlPath,
				Body:   body,
			}
		},
	}
}

// emptyBodyPattern tests POST/PUT handlers with empty body
func emptyBodyPattern(method, urlPath string) ErrorTestPattern {
	return ErrorTestPattern{
		Name:           "EmptyBody",
		Description:    "Test handler with empty request body",
		ExpectedStatus: http.StatusBadRequest,
		Setup:          nil,
		Execute: func(t *testing.T, setupData interface{}) *FiberTestRequest {
			return &FiberTestRequest{
				Method: method,
				Path:   urlPath,
				Body:   nil,
			}
		},
	}
}

// PerformanceTestPattern defines performance testing scenarios
type PerformanceTestPattern struct {
	Name           string
	Description    string
	MaxDuration    time.Duration
	Setup          func(t *testing.T) interface{}
	Execute        func(t *testing.T, setupData interface{}) time.Duration
	Cleanup        func(setupData interface{})
}

// ConcurrencyTestPattern defines concurrency testing scenarios
type ConcurrencyTestPattern struct {
	Name           string
	Description    string
	Concurrency    int
	Iterations     int
	Setup          func(t *testing.T) interface{}
	Execute        func(t *testing.T, setupData interface{}, iteration int) error
	Validate       func(t *testing.T, setupData interface{}, results []error)
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

// AddInvalidUUID adds invalid UUID test pattern
func (b *TestScenarioBuilder) AddInvalidUUID(urlPath string) *TestScenarioBuilder {
	b.patterns = append(b.patterns, invalidUUIDPattern(urlPath))
	return b
}

// AddNonExistentTask adds non-existent task test pattern
func (b *TestScenarioBuilder) AddNonExistentTask(method, urlPathTemplate string) *TestScenarioBuilder {
	b.patterns = append(b.patterns, nonExistentTaskPattern(method, urlPathTemplate))
	return b
}

// AddInvalidJSON adds invalid JSON test pattern
func (b *TestScenarioBuilder) AddInvalidJSON(method, urlPath string) *TestScenarioBuilder {
	b.patterns = append(b.patterns, invalidJSONPattern(method, urlPath))
	return b
}

// AddMissingField adds missing required field test pattern
func (b *TestScenarioBuilder) AddMissingField(method, urlPath, fieldName string) *TestScenarioBuilder {
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

// Template for comprehensive handler testing
func TemplateComprehensiveHandlerTest(t *testing.T, handlerName string, app *fiber.App) {
	// This template can be copied and customized for each handler
	t.Run(fmt.Sprintf("%s_Comprehensive", handlerName), func(t *testing.T) {
		// 1. Setup
		loggerCleanup := setupTestLogger()
		defer loggerCleanup()

		env := setupTestDirectory(t)
		defer env.Cleanup()

		// 2. Test successful scenarios
		t.Run("Success_Cases", func(t *testing.T) {
			// Add success test cases here
		})

		// 3. Test error conditions
		t.Run("Error_Cases", func(t *testing.T) {
			// Add error test cases here
		})

		// 4. Test edge cases
		t.Run("Edge_Cases", func(t *testing.T) {
			// Add edge case tests here
		})

		// 5. Test performance (if needed)
		t.Run("Performance", func(t *testing.T) {
			// Add performance tests here
		})
	})
}

// Specific pattern builders for swarm-manager

// buildTaskErrorPatterns builds error patterns for task endpoints
func buildTaskErrorPatterns() []ErrorTestPattern {
	return NewTestScenarioBuilder().
		AddNonExistentTask("GET", "/api/tasks/%s").
		AddNonExistentTask("PUT", "/api/tasks/%s").
		AddNonExistentTask("DELETE", "/api/tasks/%s").
		AddInvalidJSON("POST", "/api/tasks").
		AddInvalidJSON("PUT", "/api/tasks/valid-id").
		AddEmptyBody("POST", "/api/tasks").
		Build()
}

// buildProblemErrorPatterns builds error patterns for problem endpoints
func buildProblemErrorPatterns() []ErrorTestPattern {
	return NewTestScenarioBuilder().
		AddNonExistentTask("GET", "/api/problems/%s").
		AddNonExistentTask("PUT", "/api/problems/%s/resolve").
		AddInvalidJSON("POST", "/api/problems/scan").
		AddEmptyBody("POST", "/api/problems/scan").
		Build()
}

// buildAgentErrorPatterns builds error patterns for agent endpoints
func buildAgentErrorPatterns() []ErrorTestPattern {
	return NewTestScenarioBuilder().
		AddInvalidJSON("POST", "/api/agents/test-agent/heartbeat").
		AddEmptyBody("POST", "/api/agents/test-agent/heartbeat").
		Build()
}

// buildConfigErrorPatterns builds error patterns for config endpoints
func buildConfigErrorPatterns() []ErrorTestPattern {
	return NewTestScenarioBuilder().
		AddInvalidJSON("PUT", "/api/config").
		AddEmptyBody("PUT", "/api/config").
		Build()
}
