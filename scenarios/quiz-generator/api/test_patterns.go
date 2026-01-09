
package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
)

// TestScenario represents a test case with setup, execution, and validation
type TestScenario struct {
	Name           string
	Description    string
	Method         string
	Path           string
	Body           interface{}
	ExpectedStatus int
	ValidateBody   func(t *testing.T, body map[string]interface{})
	Setup          func(t *testing.T, env *TestEnvironment) interface{}
	Cleanup        func(t *testing.T, env *TestEnvironment, data interface{})
}

// TestScenarioBuilder provides a fluent interface for building test scenarios
type TestScenarioBuilder struct {
	scenarios []TestScenario
}

// NewTestScenarioBuilder creates a new test scenario builder
func NewTestScenarioBuilder() *TestScenarioBuilder {
	return &TestScenarioBuilder{
		scenarios: []TestScenario{},
	}
}

// AddInvalidUUID adds a test for invalid UUID format
func (b *TestScenarioBuilder) AddInvalidUUID(path, method string) *TestScenarioBuilder {
	invalidPath := path
	if path == "/api/v1/quiz/:id" {
		invalidPath = "/api/v1/quiz/invalid-uuid-format"
	}

	b.scenarios = append(b.scenarios, TestScenario{
		Name:           "InvalidUUID",
		Description:    "Test with invalid UUID format",
		Method:         method,
		Path:           invalidPath,
		ExpectedStatus: http.StatusNotFound, // Gin returns 404 for invalid UUID
		ValidateBody: func(t *testing.T, body map[string]interface{}) {
			if _, hasError := body["error"]; !hasError {
				t.Error("Expected error field in response")
			}
		},
	})

	return b
}

// AddNonExistentQuiz adds a test for non-existent quiz ID
func (b *TestScenarioBuilder) AddNonExistentQuiz(path, method string) *TestScenarioBuilder {
	nonExistentID := uuid.New().String()
	nonExistentPath := path
	if path == "/api/v1/quiz/:id" {
		nonExistentPath = fmt.Sprintf("/api/v1/quiz/%s", nonExistentID)
	}

	b.scenarios = append(b.scenarios, TestScenario{
		Name:           "NonExistentQuiz",
		Description:    "Test with non-existent quiz ID",
		Method:         method,
		Path:           nonExistentPath,
		ExpectedStatus: http.StatusNotFound,
		ValidateBody: func(t *testing.T, body map[string]interface{}) {
			if _, hasError := body["error"]; !hasError {
				t.Error("Expected error field in response")
			}
		},
	})

	return b
}

// AddInvalidJSON adds a test for malformed JSON in request body
func (b *TestScenarioBuilder) AddInvalidJSON(path, method string) *TestScenarioBuilder {
	b.scenarios = append(b.scenarios, TestScenario{
		Name:           "InvalidJSON",
		Description:    "Test with malformed JSON body",
		Method:         method,
		Path:           path,
		Body:           "invalid-json{{{",
		ExpectedStatus: http.StatusBadRequest,
		ValidateBody: func(t *testing.T, body map[string]interface{}) {
			if _, hasError := body["error"]; !hasError {
				t.Error("Expected error field in response")
			}
		},
	})

	return b
}

// AddMissingRequiredFields adds tests for missing required fields
func (b *TestScenarioBuilder) AddMissingRequiredFields(path, method string, fieldName string) *TestScenarioBuilder {
	b.scenarios = append(b.scenarios, TestScenario{
		Name:           fmt.Sprintf("Missing%s", fieldName),
		Description:    fmt.Sprintf("Test with missing %s field", fieldName),
		Method:         method,
		Path:           path,
		Body:           map[string]interface{}{}, // Empty body
		ExpectedStatus: http.StatusBadRequest,
		ValidateBody: func(t *testing.T, body map[string]interface{}) {
			if _, hasError := body["error"]; !hasError {
				t.Error("Expected error field in response")
			}
		},
	})

	return b
}

// AddEmptyRequestBody adds a test for empty request body
func (b *TestScenarioBuilder) AddEmptyRequestBody(path, method string) *TestScenarioBuilder {
	b.scenarios = append(b.scenarios, TestScenario{
		Name:           "EmptyRequestBody",
		Description:    "Test with empty request body",
		Method:         method,
		Path:           path,
		Body:           nil,
		ExpectedStatus: http.StatusBadRequest,
		ValidateBody: func(t *testing.T, body map[string]interface{}) {
			if _, hasError := body["error"]; !hasError {
				t.Error("Expected error field in response")
			}
		},
	})

	return b
}

// AddInvalidFieldValues adds tests for invalid field values
func (b *TestScenarioBuilder) AddInvalidFieldValues(path, method string, body interface{}) *TestScenarioBuilder {
	b.scenarios = append(b.scenarios, TestScenario{
		Name:           "InvalidFieldValues",
		Description:    "Test with invalid field values",
		Method:         method,
		Path:           path,
		Body:           body,
		ExpectedStatus: http.StatusBadRequest,
		ValidateBody: func(t *testing.T, body map[string]interface{}) {
			if _, hasError := body["error"]; !hasError {
				t.Error("Expected error field in response")
			}
		},
	})

	return b
}

// Build returns all configured test scenarios
func (b *TestScenarioBuilder) Build() []TestScenario {
	return b.scenarios
}

// ErrorTestPattern defines systematic error testing patterns
type ErrorTestPattern struct {
	Name           string
	Description    string
	ExpectedStatus int
	Setup          func(t *testing.T, env *TestEnvironment) interface{}
	Execute        func(t *testing.T, env *TestEnvironment, setupData interface{}) *httptest.ResponseRecorder
	Validate       func(t *testing.T, w *httptest.ResponseRecorder, setupData interface{})
	Cleanup        func(t *testing.T, env *TestEnvironment, setupData interface{})
}

// RunErrorTests executes a suite of error pattern tests
func RunErrorTests(t *testing.T, env *TestEnvironment, patterns []ErrorTestPattern) {
	for _, pattern := range patterns {
		t.Run(pattern.Name, func(t *testing.T) {
			// Setup
			var setupData interface{}
			if pattern.Setup != nil {
				setupData = pattern.Setup(t, env)
			}

			// Cleanup
			if pattern.Cleanup != nil {
				defer pattern.Cleanup(t, env, setupData)
			}

			// Execute
			w := pattern.Execute(t, env, setupData)

			// Validate status
			if w.Code != pattern.ExpectedStatus {
				t.Errorf("Expected status %d, got %d. Body: %s",
					pattern.ExpectedStatus, w.Code, w.Body.String())
			}

			// Additional validation
			if pattern.Validate != nil {
				pattern.Validate(t, w, setupData)
			}
		})
	}
}

// Common error patterns

// DatabaseConnectionError tests behavior when database is unavailable
func DatabaseConnectionError(handlerName string) ErrorTestPattern {
	return ErrorTestPattern{
		Name:           "DatabaseConnectionError",
		Description:    "Test handler behavior when database connection fails",
		ExpectedStatus: http.StatusInternalServerError,
		Setup: func(t *testing.T, env *TestEnvironment) interface{} {
			// Close database connection temporarily
			env.DB.Close()
			return nil
		},
		Execute: func(t *testing.T, env *TestEnvironment, setupData interface{}) *httptest.ResponseRecorder {
			// This would need to be customized per handler
			return nil
		},
		Validate: func(t *testing.T, w *httptest.ResponseRecorder, setupData interface{}) {
			if w != nil {
				assertErrorResponse(t, w, http.StatusInternalServerError)
			}
		},
	}
}

// EdgeCasePatterns provides common edge case test patterns
type EdgeCasePatterns struct {
	scenarios []TestScenario
}

// NewEdgeCasePatterns creates edge case pattern builder
func NewEdgeCasePatterns() *EdgeCasePatterns {
	return &EdgeCasePatterns{
		scenarios: []TestScenario{},
	}
}

// AddEmptyStringFields adds tests for empty string fields
func (e *EdgeCasePatterns) AddEmptyStringFields(path, method string, body interface{}) *EdgeCasePatterns {
	e.scenarios = append(e.scenarios, TestScenario{
		Name:           "EmptyStringFields",
		Description:    "Test with empty string fields",
		Method:         method,
		Path:           path,
		Body:           body,
		ExpectedStatus: http.StatusBadRequest,
	})
	return e
}

// AddBoundaryValues adds tests for boundary value conditions
func (e *EdgeCasePatterns) AddBoundaryValues(path, method string, body interface{}, expectedStatus int) *EdgeCasePatterns {
	e.scenarios = append(e.scenarios, TestScenario{
		Name:           "BoundaryValues",
		Description:    "Test with boundary value conditions",
		Method:         method,
		Path:           path,
		Body:           body,
		ExpectedStatus: expectedStatus,
	})
	return e
}

// AddNullValues adds tests for null/nil values
func (e *EdgeCasePatterns) AddNullValues(path, method string) *EdgeCasePatterns {
	e.scenarios = append(e.scenarios, TestScenario{
		Name:           "NullValues",
		Description:    "Test with null values in request",
		Method:         method,
		Path:           path,
		Body:           map[string]interface{}{"content": nil},
		ExpectedStatus: http.StatusBadRequest,
	})
	return e
}

// Build returns all edge case scenarios
func (e *EdgeCasePatterns) Build() []TestScenario {
	return e.scenarios
}
