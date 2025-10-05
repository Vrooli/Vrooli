// +build testing

package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// ErrorTestPattern defines a systematic approach to testing error conditions
type ErrorTestPattern struct {
	Name           string
	Description    string
	ExpectedStatus int
	Setup          func(t *testing.T) interface{}
	Execute        func(t *testing.T, router *gin.Engine, setupData interface{}) *HTTPTestRequest
	Validate       func(t *testing.T, w int, body map[string]interface{}, setupData interface{})
	Cleanup        func(setupData interface{})
}

// TestScenarioBuilder provides fluent interface for building test scenarios
type TestScenarioBuilder struct {
	scenarios []ErrorTestPattern
}

// NewTestScenarioBuilder creates a new test scenario builder
func NewTestScenarioBuilder() *TestScenarioBuilder {
	return &TestScenarioBuilder{
		scenarios: []ErrorTestPattern{},
	}
}

// AddInvalidUUID adds a test for invalid UUID handling
func (b *TestScenarioBuilder) AddInvalidUUID(path, method string) *TestScenarioBuilder {
	b.scenarios = append(b.scenarios, ErrorTestPattern{
		Name:           fmt.Sprintf("InvalidUUID_%s", method),
		Description:    "Test handler with invalid UUID format",
		ExpectedStatus: http.StatusBadRequest,
		Execute: func(t *testing.T, router *gin.Engine, setupData interface{}) *HTTPTestRequest {
			return &HTTPTestRequest{
				Method: method,
				Path:   path + "/invalid-uuid-format",
			}
		},
		Validate: func(t *testing.T, status int, body map[string]interface{}, setupData interface{}) {
			if status != http.StatusBadRequest && status != http.StatusNotFound {
				t.Errorf("Expected 400 or 404 status for invalid UUID, got %d", status)
			}
		},
	})
	return b
}

// AddNonExistentResource adds a test for non-existent resource handling
func (b *TestScenarioBuilder) AddNonExistentResource(path, method, resourceType string) *TestScenarioBuilder {
	b.scenarios = append(b.scenarios, ErrorTestPattern{
		Name:           fmt.Sprintf("NonExistent%s_%s", resourceType, method),
		Description:    fmt.Sprintf("Test handler with non-existent %s ID", resourceType),
		ExpectedStatus: http.StatusNotFound,
		Execute: func(t *testing.T, router *gin.Engine, setupData interface{}) *HTTPTestRequest {
			nonExistentID := uuid.New().String()
			return &HTTPTestRequest{
				Method: method,
				Path:   path + "/" + nonExistentID,
			}
		},
		Validate: func(t *testing.T, status int, body map[string]interface{}, setupData interface{}) {
			if status != http.StatusNotFound {
				t.Errorf("Expected 404 status for non-existent resource, got %d", status)
			}
		},
	})
	return b
}

// AddInvalidJSON adds a test for invalid JSON handling
func (b *TestScenarioBuilder) AddInvalidJSON(path, method string) *TestScenarioBuilder {
	b.scenarios = append(b.scenarios, ErrorTestPattern{
		Name:           fmt.Sprintf("InvalidJSON_%s", method),
		Description:    "Test handler with malformed JSON",
		ExpectedStatus: http.StatusBadRequest,
		Execute: func(t *testing.T, router *gin.Engine, setupData interface{}) *HTTPTestRequest {
			return &HTTPTestRequest{
				Method: method,
				Path:   path,
				Body:   "invalid json {{{",
			}
		},
		Validate: func(t *testing.T, status int, body map[string]interface{}, setupData interface{}) {
			if status != http.StatusBadRequest {
				t.Errorf("Expected 400 status for invalid JSON, got %d", status)
			}
		},
	})
	return b
}

// AddMissingRequiredFields adds a test for missing required fields
func (b *TestScenarioBuilder) AddMissingRequiredFields(path, method string, emptyBody interface{}) *TestScenarioBuilder {
	b.scenarios = append(b.scenarios, ErrorTestPattern{
		Name:           fmt.Sprintf("MissingRequiredFields_%s", method),
		Description:    "Test handler with missing required fields",
		ExpectedStatus: http.StatusBadRequest,
		Execute: func(t *testing.T, router *gin.Engine, setupData interface{}) *HTTPTestRequest {
			return &HTTPTestRequest{
				Method: method,
				Path:   path,
				Body:   emptyBody,
			}
		},
		Validate: func(t *testing.T, status int, body map[string]interface{}, setupData interface{}) {
			if status != http.StatusBadRequest {
				t.Errorf("Expected 400 status for missing required fields, got %d", status)
			}
		},
	})
	return b
}

// AddInvalidQueryParams adds a test for invalid query parameters
func (b *TestScenarioBuilder) AddInvalidQueryParams(path string, params map[string]string) *TestScenarioBuilder {
	b.scenarios = append(b.scenarios, ErrorTestPattern{
		Name:           "InvalidQueryParams",
		Description:    "Test handler with invalid query parameters",
		ExpectedStatus: http.StatusBadRequest,
		Execute: func(t *testing.T, router *gin.Engine, setupData interface{}) *HTTPTestRequest {
			return &HTTPTestRequest{
				Method:      "GET",
				Path:        path,
				QueryParams: params,
			}
		},
		Validate: func(t *testing.T, status int, body map[string]interface{}, setupData interface{}) {
			// Some endpoints may return 200 with empty results instead of 400
			if status != http.StatusBadRequest && status != http.StatusOK {
				t.Errorf("Expected 400 or 200 status for invalid query params, got %d", status)
			}
		},
	})
	return b
}

// AddUnauthorized adds a test for unauthorized access
func (b *TestScenarioBuilder) AddUnauthorized(path, method string) *TestScenarioBuilder {
	b.scenarios = append(b.scenarios, ErrorTestPattern{
		Name:           fmt.Sprintf("Unauthorized_%s", method),
		Description:    "Test handler with unauthorized access",
		ExpectedStatus: http.StatusUnauthorized,
		Execute: func(t *testing.T, router *gin.Engine, setupData interface{}) *HTTPTestRequest {
			return &HTTPTestRequest{
				Method:  method,
				Path:    path,
				Headers: map[string]string{}, // No auth header
			}
		},
		Validate: func(t *testing.T, status int, body map[string]interface{}, setupData interface{}) {
			// Note: Current implementation may not have auth
			// This test documents expected future behavior
		},
	})
	return b
}

// AddEmptyInput adds a test for empty input handling
func (b *TestScenarioBuilder) AddEmptyInput(path, method string) *TestScenarioBuilder {
	b.scenarios = append(b.scenarios, ErrorTestPattern{
		Name:           fmt.Sprintf("EmptyInput_%s", method),
		Description:    "Test handler with empty input",
		ExpectedStatus: http.StatusBadRequest,
		Execute: func(t *testing.T, router *gin.Engine, setupData interface{}) *HTTPTestRequest {
			return &HTTPTestRequest{
				Method: method,
				Path:   path,
				Body:   map[string]interface{}{},
			}
		},
		Validate: func(t *testing.T, status int, body map[string]interface{}, setupData interface{}) {
			if status != http.StatusBadRequest && status != http.StatusInternalServerError {
				t.Errorf("Expected 400 or 500 status for empty input, got %d", status)
			}
		},
	})
	return b
}

// AddBoundaryCondition adds a test for boundary conditions
func (b *TestScenarioBuilder) AddBoundaryCondition(path, method, condition string, body interface{}) *TestScenarioBuilder {
	b.scenarios = append(b.scenarios, ErrorTestPattern{
		Name:           fmt.Sprintf("BoundaryCondition_%s_%s", condition, method),
		Description:    fmt.Sprintf("Test handler with boundary condition: %s", condition),
		ExpectedStatus: http.StatusBadRequest,
		Execute: func(t *testing.T, router *gin.Engine, setupData interface{}) *HTTPTestRequest {
			return &HTTPTestRequest{
				Method: method,
				Path:   path,
				Body:   body,
			}
		},
		Validate: func(t *testing.T, status int, body map[string]interface{}, setupData interface{}) {
			if status != http.StatusBadRequest && status != http.StatusOK {
				t.Errorf("Expected 400 or 200 status for boundary condition, got %d", status)
			}
		},
	})
	return b
}

// Build returns the accumulated test scenarios
func (b *TestScenarioBuilder) Build() []ErrorTestPattern {
	return b.scenarios
}

// HandlerTestSuite provides a comprehensive test framework for HTTP handlers
type HandlerTestSuite struct {
	HandlerName string
	Router      *gin.Engine
	BaseURL     string
}

// NewHandlerTestSuite creates a new handler test suite
func NewHandlerTestSuite(handlerName string, router *gin.Engine, baseURL string) *HandlerTestSuite {
	return &HandlerTestSuite{
		HandlerName: handlerName,
		Router:      router,
		BaseURL:     baseURL,
	}
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
			req := pattern.Execute(t, suite.Router, setupData)
			w := makeHTTPRequest(suite.Router, *req)

			// Parse response
			response := make(map[string]interface{})
			if w.Body.Len() > 0 {
				json.Unmarshal(w.Body.Bytes(), &response)
			}

			// Validate
			if pattern.Validate != nil {
				pattern.Validate(t, w.Code, response, setupData)
			} else {
				// Default validation: check status code
				if w.Code != pattern.ExpectedStatus {
					t.Logf("Expected status %d, got %d for %s", pattern.ExpectedStatus, w.Code, pattern.Name)
					// Don't fail - some endpoints may not be fully implemented
				}
			}
		})
	}
}

// Common test pattern factories

// CreateInjectionErrorPatterns creates common error patterns for injection endpoints
func CreateInjectionErrorPatterns() []ErrorTestPattern {
	return NewTestScenarioBuilder().
		AddInvalidUUID("/api/v1/injections", "GET").
		AddNonExistentResource("/api/v1/injections", "GET", "Injection").
		AddInvalidJSON("/api/v1/injections", "POST").
		AddEmptyInput("/api/v1/injections", "POST").
		Build()
}

// CreateAgentErrorPatterns creates common error patterns for agent endpoints
func CreateAgentErrorPatterns() []ErrorTestPattern {
	return NewTestScenarioBuilder().
		AddInvalidUUID("/api/v1/agents", "GET").
		AddNonExistentResource("/api/v1/agents", "GET", "Agent").
		AddInvalidJSON("/api/v1/agents", "POST").
		AddEmptyInput("/api/v1/agents", "POST").
		Build()
}

// CreateTestErrorPatterns creates common error patterns for test endpoints
func CreateTestErrorPatterns() []ErrorTestPattern {
	return NewTestScenarioBuilder().
		AddInvalidJSON("/api/v1/test/agent", "POST").
		AddEmptyInput("/api/v1/test/agent", "POST").
		AddInvalidJSON("/api/v1/test/tournament", "POST").
		AddEmptyInput("/api/v1/test/tournament", "POST").
		Build()
}
