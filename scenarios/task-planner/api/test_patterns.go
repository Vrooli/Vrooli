package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
)

// ErrorTestPattern defines a systematic approach to testing error conditions
type ErrorTestPattern struct {
	Name           string
	Description    string
	ExpectedStatus int
	Request        HTTPTestRequest
}

// TestScenarioBuilder provides a fluent interface for building test scenarios
type TestScenarioBuilder struct {
	patterns []ErrorTestPattern
}

// NewTestScenarioBuilder creates a new test scenario builder
func NewTestScenarioBuilder() *TestScenarioBuilder {
	return &TestScenarioBuilder{
		patterns: make([]ErrorTestPattern, 0),
	}
}

// AddInvalidUUID adds a test pattern for invalid UUID format
func (b *TestScenarioBuilder) AddInvalidUUID(path, method string) *TestScenarioBuilder {
	b.patterns = append(b.patterns, ErrorTestPattern{
		Name:           "InvalidUUID",
		Description:    "Test with invalid UUID format",
		ExpectedStatus: http.StatusBadRequest,
		Request: HTTPTestRequest{
			Method:  method,
			Path:    path,
			URLVars: map[string]string{"taskId": "invalid-uuid-format"},
		},
	})
	return b
}

// AddNonExistentTask adds a test pattern for non-existent task
func (b *TestScenarioBuilder) AddNonExistentTask(path, method string) *TestScenarioBuilder {
	nonExistentID := uuid.New().String()
	b.patterns = append(b.patterns, ErrorTestPattern{
		Name:           "NonExistentTask",
		Description:    "Test with non-existent task ID",
		ExpectedStatus: http.StatusNotFound,
		Request: HTTPTestRequest{
			Method:  method,
			Path:    path,
			URLVars: map[string]string{"taskId": nonExistentID},
		},
	})
	return b
}

// AddInvalidJSON adds a test pattern for malformed JSON payload
func (b *TestScenarioBuilder) AddInvalidJSON(path, method string) *TestScenarioBuilder {
	b.patterns = append(b.patterns, ErrorTestPattern{
		Name:           "InvalidJSON",
		Description:    "Test with malformed JSON payload",
		ExpectedStatus: http.StatusBadRequest,
		Request: HTTPTestRequest{
			Method: method,
			Path:   path,
			Body:   "invalid-json-{malformed",
		},
	})
	return b
}

// AddMissingRequiredField adds a test pattern for missing required fields
func (b *TestScenarioBuilder) AddMissingRequiredField(path, method string, body interface{}) *TestScenarioBuilder {
	b.patterns = append(b.patterns, ErrorTestPattern{
		Name:           "MissingRequiredField",
		Description:    "Test with missing required fields",
		ExpectedStatus: http.StatusBadRequest,
		Request: HTTPTestRequest{
			Method: method,
			Path:   path,
			Body:   body,
		},
	})
	return b
}

// AddInvalidStatusTransition adds a test pattern for invalid status transitions
func (b *TestScenarioBuilder) AddInvalidStatusTransition(body interface{}) *TestScenarioBuilder {
	b.patterns = append(b.patterns, ErrorTestPattern{
		Name:           "InvalidStatusTransition",
		Description:    "Test with invalid status transition",
		ExpectedStatus: http.StatusBadRequest,
		Request: HTTPTestRequest{
			Method: "PUT",
			Path:   "/api/tasks/status",
			Body:   body,
		},
	})
	return b
}

// AddUnauthorizedAccess adds a test pattern for unauthorized access
func (b *TestScenarioBuilder) AddUnauthorizedAccess(path, method string) *TestScenarioBuilder {
	b.patterns = append(b.patterns, ErrorTestPattern{
		Name:           "UnauthorizedAccess",
		Description:    "Test with invalid or missing API token",
		ExpectedStatus: http.StatusUnauthorized,
		Request: HTTPTestRequest{
			Method: method,
			Path:   path,
			Body: map[string]interface{}{
				"app_id":    uuid.New().String(),
				"raw_text":  "Sample text",
				"api_token": "invalid-token",
			},
		},
	})
	return b
}

// AddEmptyPayload adds a test pattern for empty request body
func (b *TestScenarioBuilder) AddEmptyPayload(path, method string) *TestScenarioBuilder {
	b.patterns = append(b.patterns, ErrorTestPattern{
		Name:           "EmptyPayload",
		Description:    "Test with empty request body",
		ExpectedStatus: http.StatusBadRequest,
		Request: HTTPTestRequest{
			Method: method,
			Path:   path,
			Body:   map[string]interface{}{},
		},
	})
	return b
}

// AddInvalidQueryParam adds a test pattern for invalid query parameters
func (b *TestScenarioBuilder) AddInvalidQueryParam(path string, paramName, paramValue string) *TestScenarioBuilder {
	b.patterns = append(b.patterns, ErrorTestPattern{
		Name:           fmt.Sprintf("InvalidQueryParam_%s", paramName),
		Description:    fmt.Sprintf("Test with invalid %s query parameter", paramName),
		ExpectedStatus: http.StatusBadRequest,
		Request: HTTPTestRequest{
			Method:      "GET",
			Path:        path,
			QueryParams: map[string]string{paramName: paramValue},
		},
	})
	return b
}

// Build returns the constructed test patterns
func (b *TestScenarioBuilder) Build() []ErrorTestPattern {
	return b.patterns
}

// HandlerTestSuite provides a comprehensive test framework for HTTP handlers
type HandlerTestSuite struct {
	HandlerName string
	TestEnv     *TestEnvironment
}

// NewHandlerTestSuite creates a new handler test suite
func NewHandlerTestSuite(name string, env *TestEnvironment) *HandlerTestSuite {
	return &HandlerTestSuite{
		HandlerName: name,
		TestEnv:     env,
	}
}

// RunErrorTests executes a suite of error condition tests
func (suite *HandlerTestSuite) RunErrorTests(t *testing.T, patterns []ErrorTestPattern) {
	for _, pattern := range patterns {
		t.Run(fmt.Sprintf("%s_%s", suite.HandlerName, pattern.Name), func(t *testing.T) {
			w, err := makeHTTPRequest(suite.TestEnv, pattern.Request)
			if err != nil {
				t.Fatalf("Failed to make HTTP request: %v", err)
			}

			if w.Code != pattern.ExpectedStatus {
				t.Errorf("Expected status %d, got %d. Body: %s",
					pattern.ExpectedStatus, w.Code, w.Body.String())
			}

			// Validate error response structure
			if pattern.ExpectedStatus >= 400 {
				response := assertJSONResponse(t, w, pattern.ExpectedStatus)
				if _, ok := response["error"]; !ok {
					t.Errorf("Expected 'error' field in error response, got: %v", response)
				}
			}
		})
	}
}

// RunSuccessTest executes a success path test
func (suite *HandlerTestSuite) RunSuccessTest(t *testing.T, name string, req HTTPTestRequest, validate func(*testing.T, *httptest.ResponseRecorder)) {
	t.Run(fmt.Sprintf("%s_%s", suite.HandlerName, name), func(t *testing.T) {
		w, err := makeHTTPRequest(suite.TestEnv, req)
		if err != nil {
			t.Fatalf("Failed to make HTTP request: %v", err)
		}

		validate(t, w)
	})
}

// Common validation helpers

// ValidateTaskResponse validates a task response structure
func ValidateTaskResponse(t *testing.T, task map[string]interface{}) {
	t.Helper()

	requiredFields := []string{"id", "title", "description", "status", "priority", "app_id", "created_at", "updated_at"}
	for _, field := range requiredFields {
		if _, ok := task[field]; !ok {
			t.Errorf("Expected field '%s' in task response, got: %v", field, task)
		}
	}

	// Validate status is a valid value
	validStatuses := map[string]bool{
		"backlog": true, "in_progress": true, "blocked": true,
		"review": true, "completed": true, "cancelled": true,
	}
	if status, ok := task["status"].(string); ok {
		if !validStatuses[status] {
			t.Errorf("Invalid task status: %s", status)
		}
	}

	// Validate priority is a valid value
	validPriorities := map[string]bool{
		"critical": true, "high": true, "medium": true, "low": true,
	}
	if priority, ok := task["priority"].(string); ok {
		if !validPriorities[priority] {
			t.Errorf("Invalid task priority: %s", priority)
		}
	}
}

// ValidateAppResponse validates an app response structure
func ValidateAppResponse(t *testing.T, app map[string]interface{}) {
	t.Helper()

	requiredFields := []string{"id", "name", "display_name", "type", "total_tasks", "completed_tasks", "created_at", "updated_at"}
	for _, field := range requiredFields {
		if _, ok := app[field]; !ok {
			t.Errorf("Expected field '%s' in app response, got: %v", field, app)
		}
	}
}

// ValidateStatusHistoryResponse validates a status history response
func ValidateStatusHistoryResponse(t *testing.T, history []interface{}) {
	t.Helper()

	if len(history) == 0 {
		return // Empty history is valid
	}

	for i, item := range history {
		change, ok := item.(map[string]interface{})
		if !ok {
			t.Errorf("Status history item %d is not a map: %v", i, item)
			continue
		}

		requiredFields := []string{"task_id", "from_status", "to_status", "changed_at"}
		for _, field := range requiredFields {
			if _, ok := change[field]; !ok {
				t.Errorf("Expected field '%s' in status history item %d, got: %v", field, i, change)
			}
		}
	}
}
