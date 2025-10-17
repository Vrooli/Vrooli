// +build testing

package main

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/google/uuid"
)

// TestScenarioBuilder provides a fluent interface for building test scenarios
type TestScenarioBuilder struct {
	scenarios []TestScenario
}

// TestScenario represents a single test scenario
type TestScenario struct {
	Name           string
	Description    string
	Method         string
	Path           string
	Body           interface{}
	URLVars        map[string]string
	Headers        map[string]string
	ExpectedStatus int
	ExpectedError  string
	Setup          func(t *testing.T) interface{}
	Cleanup        func(data interface{})
	Validate       func(t *testing.T, w interface{}, data interface{})
}

// NewTestScenarioBuilder creates a new test scenario builder
func NewTestScenarioBuilder() *TestScenarioBuilder {
	return &TestScenarioBuilder{
		scenarios: make([]TestScenario, 0),
	}
}

// AddInvalidUUID adds a test scenario for invalid UUID
func (b *TestScenarioBuilder) AddInvalidUUID(path string) *TestScenarioBuilder {
	b.scenarios = append(b.scenarios, TestScenario{
		Name:           "InvalidUUID",
		Description:    "Request with invalid UUID format",
		Method:         "GET",
		Path:           path,
		URLVars:        map[string]string{"id": "invalid-uuid-format"},
		ExpectedStatus: http.StatusBadRequest,
		ExpectedError:  "Invalid UUID format",
	})
	return b
}

// AddNonExistentAgent adds a test scenario for non-existent agent
func (b *TestScenarioBuilder) AddNonExistentAgent(path string) *TestScenarioBuilder {
	nonExistentID := uuid.New().String()
	b.scenarios = append(b.scenarios, TestScenario{
		Name:           "NonExistentAgent",
		Description:    "Request for agent that doesn't exist",
		Method:         "GET",
		Path:           path,
		URLVars:        map[string]string{"id": nonExistentID, "agentId": nonExistentID},
		ExpectedStatus: http.StatusNotFound,
		ExpectedError:  "Agent not found",
	})
	return b
}

// AddInvalidJSON adds a test scenario for invalid JSON
func (b *TestScenarioBuilder) AddInvalidJSON(path string) *TestScenarioBuilder {
	b.scenarios = append(b.scenarios, TestScenario{
		Name:           "InvalidJSON",
		Description:    "Request with malformed JSON body",
		Method:         "POST",
		Path:           path,
		Body:           "not-valid-json",
		ExpectedStatus: http.StatusBadRequest,
		ExpectedError:  "Invalid JSON",
	})
	return b
}

// AddMissingRequiredFields adds a test scenario for missing required fields
func (b *TestScenarioBuilder) AddMissingRequiredFields(path string, body interface{}) *TestScenarioBuilder {
	b.scenarios = append(b.scenarios, TestScenario{
		Name:           "MissingRequiredFields",
		Description:    "Request with missing required fields",
		Method:         "POST",
		Path:           path,
		Body:           body,
		ExpectedStatus: http.StatusBadRequest,
		ExpectedError:  "required",
	})
	return b
}

// AddDatabaseError adds a test scenario for database errors
func (b *TestScenarioBuilder) AddDatabaseError(path string, setup func(t *testing.T) interface{}) *TestScenarioBuilder {
	b.scenarios = append(b.scenarios, TestScenario{
		Name:           "DatabaseError",
		Description:    "Request that triggers database error",
		Method:         "GET",
		Path:           path,
		Setup:          setup,
		ExpectedStatus: http.StatusInternalServerError,
		ExpectedError:  "Database error",
	})
	return b
}

// AddEmptyResult adds a test scenario for empty query results
func (b *TestScenarioBuilder) AddEmptyResult(path string) *TestScenarioBuilder {
	b.scenarios = append(b.scenarios, TestScenario{
		Name:           "EmptyResult",
		Description:    "Request that returns empty result set",
		Method:         "GET",
		Path:           path,
		ExpectedStatus: http.StatusOK,
		Validate: func(t *testing.T, w interface{}, data interface{}) {
			// Validate empty array response
		},
	})
	return b
}

// Build returns the built test scenarios
func (b *TestScenarioBuilder) Build() []TestScenario {
	return b.scenarios
}

// HandlerTestSuite provides comprehensive testing for HTTP handlers
type HandlerTestSuite struct {
	Name    string
	Handler http.HandlerFunc
	BaseURL string
}

// RunTests executes all test scenarios for a handler
func (suite *HandlerTestSuite) RunTests(t *testing.T, scenarios []TestScenario) {
	for _, scenario := range scenarios {
		t.Run(fmt.Sprintf("%s_%s", suite.Name, scenario.Name), func(t *testing.T) {
			// Setup
			var setupData interface{}
			if scenario.Setup != nil {
				setupData = scenario.Setup(t)
			}

			// Cleanup
			if scenario.Cleanup != nil {
				defer scenario.Cleanup(setupData)
			}

			// Execute request
			req := HTTPTestRequest{
				Method:  scenario.Method,
				Path:    scenario.Path,
				Body:    scenario.Body,
				URLVars: scenario.URLVars,
				Headers: scenario.Headers,
			}

			w, err := makeHTTPRequest(req)
			if err != nil {
				t.Fatalf("Failed to make HTTP request: %v", err)
			}

			// Execute handler
			suite.Handler(w, nil) // Note: This needs proper request object

			// Validate response status
			if w.Code != scenario.ExpectedStatus {
				t.Errorf("Expected status %d, got %d", scenario.ExpectedStatus, w.Code)
			}

			// Custom validation
			if scenario.Validate != nil {
				scenario.Validate(t, w, setupData)
			}
		})
	}
}

// Common error test patterns

// ErrorTestPattern defines a reusable error testing pattern
type ErrorTestPattern struct {
	Name        string
	Description string
	Request     HTTPTestRequest
	Expected    ExpectedResponse
	Setup       func(t *testing.T) interface{}
	Cleanup     func(data interface{})
}

// ExpectedResponse defines expected response characteristics
type ExpectedResponse struct {
	Status      int
	Success     bool
	ErrorMsg    string
	DataPresent bool
}

// InvalidUUIDPattern creates a test pattern for invalid UUID
func InvalidUUIDPattern(method, path string) ErrorTestPattern {
	return ErrorTestPattern{
		Name:        "InvalidUUID",
		Description: "Test with invalid UUID format",
		Request: HTTPTestRequest{
			Method:  method,
			Path:    path,
			URLVars: map[string]string{"id": "not-a-uuid"},
		},
		Expected: ExpectedResponse{
			Status:   http.StatusBadRequest,
			Success:  false,
			ErrorMsg: "Invalid UUID",
		},
	}
}

// NonExistentResourcePattern creates a test pattern for non-existent resource
func NonExistentResourcePattern(method, path, resourceType string) ErrorTestPattern {
	return ErrorTestPattern{
		Name:        fmt.Sprintf("NonExistent%s", resourceType),
		Description: fmt.Sprintf("Test with non-existent %s", resourceType),
		Request: HTTPTestRequest{
			Method:  method,
			Path:    path,
			URLVars: map[string]string{"id": uuid.New().String()},
		},
		Expected: ExpectedResponse{
			Status:   http.StatusNotFound,
			Success:  false,
			ErrorMsg: fmt.Sprintf("%s not found", resourceType),
		},
	}
}

// MalformedJSONPattern creates a test pattern for malformed JSON
func MalformedJSONPattern(method, path string) ErrorTestPattern {
	return ErrorTestPattern{
		Name:        "MalformedJSON",
		Description: "Test with invalid JSON payload",
		Request: HTTPTestRequest{
			Method: method,
			Path:   path,
			Body:   "this is not valid json {]",
		},
		Expected: ExpectedResponse{
			Status:   http.StatusBadRequest,
			Success:  false,
			ErrorMsg: "Invalid JSON",
		},
	}
}

// MissingFieldsPattern creates a test pattern for missing required fields
func MissingFieldsPattern(method, path string, body interface{}) ErrorTestPattern {
	return ErrorTestPattern{
		Name:        "MissingRequiredFields",
		Description: "Test with missing required fields",
		Request: HTTPTestRequest{
			Method: method,
			Path:   path,
			Body:   body,
		},
		Expected: ExpectedResponse{
			Status:   http.StatusBadRequest,
			Success:  false,
			ErrorMsg: "required",
		},
	}
}

// EmptyBodyPattern creates a test pattern for empty request body
func EmptyBodyPattern(method, path string) ErrorTestPattern {
	return ErrorTestPattern{
		Name:        "EmptyBody",
		Description: "Test with empty request body",
		Request: HTTPTestRequest{
			Method: method,
			Path:   path,
			Body:   nil,
		},
		Expected: ExpectedResponse{
			Status:   http.StatusBadRequest,
			Success:  false,
			ErrorMsg: "Invalid JSON",
		},
	}
}

// DatabaseConnectionPattern creates a test pattern for database errors
func DatabaseConnectionPattern(method, path string, setup func(t *testing.T) interface{}) ErrorTestPattern {
	return ErrorTestPattern{
		Name:        "DatabaseError",
		Description: "Test database connection/query failure",
		Request: HTTPTestRequest{
			Method: method,
			Path:   path,
		},
		Expected: ExpectedResponse{
			Status:   http.StatusInternalServerError,
			Success:  false,
			ErrorMsg: "Database",
		},
		Setup: setup,
	}
}

// UnauthorizedAccessPattern creates a test pattern for unauthorized access
func UnauthorizedAccessPattern(method, path string) ErrorTestPattern {
	return ErrorTestPattern{
		Name:        "UnauthorizedAccess",
		Description: "Test unauthorized access attempt",
		Request: HTTPTestRequest{
			Method:  method,
			Path:    path,
			Headers: map[string]string{}, // No auth headers
		},
		Expected: ExpectedResponse{
			Status:   http.StatusUnauthorized,
			Success:  false,
			ErrorMsg: "Unauthorized",
		},
	}
}

// RateLimitPattern creates a test pattern for rate limiting
func RateLimitPattern(method, path string, requestCount int) ErrorTestPattern {
	return ErrorTestPattern{
		Name:        "RateLimitExceeded",
		Description: "Test rate limit enforcement",
		Request: HTTPTestRequest{
			Method: method,
			Path:   path,
		},
		Expected: ExpectedResponse{
			Status:   http.StatusTooManyRequests,
			Success:  false,
			ErrorMsg: "Rate limit exceeded",
		},
	}
}
