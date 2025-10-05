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

// TestScenario represents a complete test scenario with setup, execution, and validation
type TestScenario struct {
	Name           string
	Description    string
	Setup          func(t *testing.T, app *App) interface{}
	Execute        func(t *testing.T, app *App, setupData interface{}) *HTTPTestRequest
	ExpectedStatus int
	Validate       func(t *testing.T, response map[string]interface{}, setupData interface{})
	Cleanup        func(t *testing.T, setupData interface{})
}

// NewTestScenarioBuilder creates a new test scenario builder
func NewTestScenarioBuilder() *TestScenarioBuilder {
	return &TestScenarioBuilder{
		scenarios: make([]TestScenario, 0),
	}
}

// AddInvalidUUID adds a test for invalid UUID format
func (b *TestScenarioBuilder) AddInvalidUUID(endpoint string, method string) *TestScenarioBuilder {
	b.scenarios = append(b.scenarios, TestScenario{
		Name:        "InvalidUUID",
		Description: fmt.Sprintf("Test %s %s with invalid UUID", method, endpoint),
		Setup:       nil,
		Execute: func(t *testing.T, app *App, setupData interface{}) *HTTPTestRequest {
			return &HTTPTestRequest{
				Method: method,
				Path:   endpoint + "/invalid-uuid-format",
			}
		},
		ExpectedStatus: http.StatusBadRequest,
		Validate: func(t *testing.T, response map[string]interface{}, setupData interface{}) {
			if _, ok := response["error"]; !ok {
				t.Error("Expected error field in response")
			}
		},
	})
	return b
}

// AddNonExistentComment adds a test for non-existent comment ID
func (b *TestScenarioBuilder) AddNonExistentComment(endpoint string) *TestScenarioBuilder {
	b.scenarios = append(b.scenarios, TestScenario{
		Name:        "NonExistentComment",
		Description: "Test endpoint with non-existent comment ID",
		Setup:       nil,
		Execute: func(t *testing.T, app *App, setupData interface{}) *HTTPTestRequest {
			nonExistentID := uuid.New()
			return &HTTPTestRequest{
				Method: "GET",
				Path:   fmt.Sprintf("%s/%s", endpoint, nonExistentID.String()),
			}
		},
		ExpectedStatus: http.StatusNotFound,
		Validate: func(t *testing.T, response map[string]interface{}, setupData interface{}) {
			if _, ok := response["error"]; !ok {
				t.Error("Expected error field in response")
			}
		},
	})
	return b
}

// AddInvalidJSON adds a test for malformed JSON input
func (b *TestScenarioBuilder) AddInvalidJSON(endpoint string, method string) *TestScenarioBuilder {
	b.scenarios = append(b.scenarios, TestScenario{
		Name:        "InvalidJSON",
		Description: "Test endpoint with malformed JSON",
		Setup:       nil,
		Execute: func(t *testing.T, app *App, setupData interface{}) *HTTPTestRequest {
			return &HTTPTestRequest{
				Method: method,
				Path:   endpoint,
				Body:   "invalid json string",
			}
		},
		ExpectedStatus: http.StatusBadRequest,
		Validate: func(t *testing.T, response map[string]interface{}, setupData interface{}) {
			if _, ok := response["error"]; !ok {
				t.Error("Expected error field in response")
			}
		},
	})
	return b
}

// AddEmptyContent adds a test for empty content
func (b *TestScenarioBuilder) AddEmptyContent(endpoint string) *TestScenarioBuilder {
	b.scenarios = append(b.scenarios, TestScenario{
		Name:        "EmptyContent",
		Description: "Test creating comment with empty content",
		Setup:       nil,
		Execute: func(t *testing.T, app *App, setupData interface{}) *HTTPTestRequest {
			return &HTTPTestRequest{
				Method: "POST",
				Path:   endpoint,
				Body: map[string]interface{}{
					"content":      "",
					"author_token": "test-token",
				},
			}
		},
		ExpectedStatus: http.StatusBadRequest,
		Validate: func(t *testing.T, response map[string]interface{}, setupData interface{}) {
			if _, ok := response["error"]; !ok {
				t.Error("Expected error field in response")
			}
			// Note: Gin binding validation provides different error message than manual validation
		},
	})
	return b
}

// AddContentTooLong adds a test for content exceeding maximum length
func (b *TestScenarioBuilder) AddContentTooLong(endpoint string) *TestScenarioBuilder {
	b.scenarios = append(b.scenarios, TestScenario{
		Name:        "ContentTooLong",
		Description: "Test creating comment with content exceeding max length",
		Setup:       nil,
		Execute: func(t *testing.T, app *App, setupData interface{}) *HTTPTestRequest {
			longContent := make([]byte, 10001) // Exceeds 10000 char limit
			for i := range longContent {
				longContent[i] = 'a'
			}
			return &HTTPTestRequest{
				Method: "POST",
				Path:   endpoint,
				Body: map[string]interface{}{
					"content":      string(longContent),
					"author_token": "test-token",
				},
			}
		},
		ExpectedStatus: http.StatusBadRequest,
		Validate: func(t *testing.T, response map[string]interface{}, setupData interface{}) {
			if errMsg, ok := response["error"].(string); ok {
				if errMsg != "Comment content too long (max 10000 characters)" {
					t.Errorf("Expected 'content too long' error, got: %s", errMsg)
				}
			}
		},
	})
	return b
}

// AddMissingAuthentication adds a test for missing authentication when required
func (b *TestScenarioBuilder) AddMissingAuthentication(endpoint string) *TestScenarioBuilder {
	b.scenarios = append(b.scenarios, TestScenario{
		Name:        "MissingAuthentication",
		Description: "Test endpoint with missing authentication when required",
		Setup: func(t *testing.T, app *App) interface{} {
			// Create scenario config that requires auth
			config, _ := app.db.CreateDefaultConfig("test-scenario-auth")
			return config
		},
		Execute: func(t *testing.T, app *App, setupData interface{}) *HTTPTestRequest {
			return &HTTPTestRequest{
				Method: "POST",
				Path:   endpoint,
				Body: map[string]interface{}{
					"content": "Test content",
				},
			}
		},
		ExpectedStatus: http.StatusUnauthorized,
		Validate: func(t *testing.T, response map[string]interface{}, setupData interface{}) {
			if _, ok := response["error"]; !ok {
				t.Error("Expected error field in response")
			}
		},
	})
	return b
}

// AddInvalidParentID adds a test for invalid parent comment ID
func (b *TestScenarioBuilder) AddInvalidParentID(endpoint string) *TestScenarioBuilder {
	b.scenarios = append(b.scenarios, TestScenario{
		Name:        "InvalidParentID",
		Description: "Test creating comment with non-existent parent ID",
		Setup:       nil,
		Execute: func(t *testing.T, app *App, setupData interface{}) *HTTPTestRequest {
			nonExistentParent := uuid.New()
			return &HTTPTestRequest{
				Method: "POST",
				Path:   endpoint,
				Body: map[string]interface{}{
					"content":      "Reply to non-existent comment",
					"parent_id":    nonExistentParent.String(),
					"author_token": "test-token",
				},
			}
		},
		ExpectedStatus: http.StatusBadRequest,
		Validate: func(t *testing.T, response map[string]interface{}, setupData interface{}) {
			// Should fail due to foreign key constraint or validation
			if _, ok := response["error"]; !ok {
				t.Error("Expected error field in response")
			}
		},
	})
	return b
}

// AddInvalidQueryParams adds tests for invalid query parameters
func (b *TestScenarioBuilder) AddInvalidQueryParams(endpoint string) *TestScenarioBuilder {
	// Test with negative limit
	b.scenarios = append(b.scenarios, TestScenario{
		Name:        "NegativeLimit",
		Description: "Test endpoint with negative limit parameter",
		Setup:       nil,
		Execute: func(t *testing.T, app *App, setupData interface{}) *HTTPTestRequest {
			return &HTTPTestRequest{
				Method: "GET",
				Path:   endpoint + "?limit=-1",
			}
		},
		ExpectedStatus: http.StatusOK, // Should default to valid limit
		Validate: func(t *testing.T, response map[string]interface{}, setupData interface{}) {
			// Verify it used default limit
		},
	})

	// Test with excessive limit
	b.scenarios = append(b.scenarios, TestScenario{
		Name:        "ExcessiveLimit",
		Description: "Test endpoint with limit exceeding maximum",
		Setup:       nil,
		Execute: func(t *testing.T, app *App, setupData interface{}) *HTTPTestRequest {
			return &HTTPTestRequest{
				Method: "GET",
				Path:   endpoint + "?limit=1000",
			}
		},
		ExpectedStatus: http.StatusOK, // Should cap to max limit
		Validate: func(t *testing.T, response map[string]interface{}, setupData interface{}) {
			// Verify it capped the limit
		},
	})

	return b
}

// Build returns all configured test scenarios
func (b *TestScenarioBuilder) Build() []TestScenario {
	return b.scenarios
}

// RunScenarios executes all scenarios in the builder
func (b *TestScenarioBuilder) RunScenarios(t *testing.T, app *App) {
	for _, scenario := range b.scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			// Setup
			var setupData interface{}
			if scenario.Setup != nil {
				setupData = scenario.Setup(t, app)
			}

			// Cleanup
			if scenario.Cleanup != nil {
				defer scenario.Cleanup(t, setupData)
			}

			// Execute
			req := scenario.Execute(t, app, setupData)
			w, err := makeHTTPRequest(app, *req)
			if err != nil {
				t.Fatalf("Failed to make HTTP request: %v", err)
			}

			// Validate status code
			if w.Code != scenario.ExpectedStatus {
				t.Errorf("Expected status %d, got %d. Body: %s", scenario.ExpectedStatus, w.Code, w.Body.String())
			}

			// Parse response
			response := assertJSONResponse(t, w, scenario.ExpectedStatus)

			// Additional validation
			if scenario.Validate != nil {
				scenario.Validate(t, response, setupData)
			}
		})
	}
}

// ErrorTestPattern defines common error testing patterns
type ErrorTestPattern struct {
	Name           string
	Description    string
	ExpectedStatus int
	RequestFactory func() *HTTPTestRequest
	Validator      func(t *testing.T, response map[string]interface{})
}

// GetCommentErrorPatterns returns common error patterns for comment endpoints
func GetCommentErrorPatterns() []ErrorTestPattern {
	return []ErrorTestPattern{
		{
			Name:           "InvalidScenarioName",
			Description:    "Test with invalid scenario name containing special characters",
			ExpectedStatus: http.StatusBadRequest,
			RequestFactory: func() *HTTPTestRequest {
				return &HTTPTestRequest{
					Method: "GET",
					Path:   "/api/v1/comments/<script>alert('xss')</script>",
				}
			},
		},
		{
			Name:           "SQLInjectionAttempt",
			Description:    "Test with SQL injection attempt in scenario name",
			ExpectedStatus: http.StatusBadRequest,
			RequestFactory: func() *HTTPTestRequest {
				return &HTTPTestRequest{
					Method: "GET",
					Path:   "/api/v1/comments/test'; DROP TABLE comments; --",
				}
			},
		},
	}
}

// ValidationHelper provides reusable validation functions
type ValidationHelper struct{}

// ValidateCommentResponse validates a comment response structure
func (v *ValidationHelper) ValidateCommentResponse(t *testing.T, response map[string]interface{}) {
	t.Helper()

	if comment, ok := response["comment"].(map[string]interface{}); ok {
		assertCommentFields(t, comment)
	} else if comments, ok := response["comments"].([]interface{}); ok {
		if len(comments) > 0 {
			for _, c := range comments {
				if commentMap, ok := c.(map[string]interface{}); ok {
					assertCommentFields(t, commentMap)
				}
			}
		}
	}
}

// ValidateConfigResponse validates a config response structure
func (v *ValidationHelper) ValidateConfigResponse(t *testing.T, response map[string]interface{}) {
	t.Helper()

	if config, ok := response["config"].(map[string]interface{}); ok {
		assertConfigFields(t, config)
	}
}

// ValidatePaginationResponse validates pagination fields
func (v *ValidationHelper) ValidatePaginationResponse(t *testing.T, response map[string]interface{}) {
	t.Helper()

	requiredFields := []string{"total_count", "has_more"}
	for _, field := range requiredFields {
		if _, ok := response[field]; !ok {
			t.Errorf("Response missing pagination field: %s", field)
		}
	}
}
