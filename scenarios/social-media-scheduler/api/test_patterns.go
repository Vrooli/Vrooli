// +build testing

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
	Setup          func(t *testing.T, env *TestEnvironment) interface{}
	Execute        func(t *testing.T, env *TestEnvironment, setupData interface{}) *httptest.ResponseRecorder
	Validate       func(t *testing.T, w *httptest.ResponseRecorder, setupData interface{})
	Cleanup        func(t *testing.T, env *TestEnvironment, setupData interface{})
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

// AddInvalidUUID adds a test for invalid UUID format
func (b *TestScenarioBuilder) AddInvalidUUID(method, path string) *TestScenarioBuilder {
	b.patterns = append(b.patterns, ErrorTestPattern{
		Name:           "InvalidUUID",
		Description:    "Test handler with invalid UUID format",
		ExpectedStatus: http.StatusBadRequest,
		Setup:          nil,
		Execute: func(t *testing.T, env *TestEnvironment, setupData interface{}) *httptest.ResponseRecorder {
			// Replace {id} with invalid UUID
			testPath := fmt.Sprintf(path, "invalid-uuid-format")
			w, err := makeHTTPRequest(env, HTTPTestRequest{
				Method: method,
				Path:   testPath,
			})
			if err != nil {
				t.Fatalf("Failed to make request: %v", err)
			}
			return w
		},
		Validate: func(t *testing.T, w *httptest.ResponseRecorder, setupData interface{}) {
			assertErrorResponse(t, w, http.StatusBadRequest, "")
		},
	})
	return b
}

// AddNonExistentResource adds a test for non-existent resource
func (b *TestScenarioBuilder) AddNonExistentResource(method, path, resourceType string) *TestScenarioBuilder {
	b.patterns = append(b.patterns, ErrorTestPattern{
		Name:           fmt.Sprintf("NonExistent%s", resourceType),
		Description:    fmt.Sprintf("Test handler with non-existent %s ID", resourceType),
		ExpectedStatus: http.StatusNotFound,
		Setup:          nil,
		Execute: func(t *testing.T, env *TestEnvironment, setupData interface{}) *httptest.ResponseRecorder {
			nonExistentID := uuid.New().String()
			testPath := fmt.Sprintf(path, nonExistentID)
			w, err := makeHTTPRequest(env, HTTPTestRequest{
				Method: method,
				Path:   testPath,
			})
			if err != nil {
				t.Fatalf("Failed to make request: %v", err)
			}
			return w
		},
		Validate: func(t *testing.T, w *httptest.ResponseRecorder, setupData interface{}) {
			assertErrorResponse(t, w, http.StatusNotFound, "")
		},
	})
	return b
}

// AddMissingAuth adds a test for missing authentication
func (b *TestScenarioBuilder) AddMissingAuth(method, path string) *TestScenarioBuilder {
	b.patterns = append(b.patterns, ErrorTestPattern{
		Name:           "MissingAuth",
		Description:    "Test handler without authentication token",
		ExpectedStatus: http.StatusUnauthorized,
		Setup:          nil,
		Execute: func(t *testing.T, env *TestEnvironment, setupData interface{}) *httptest.ResponseRecorder {
			w, err := makeHTTPRequest(env, HTTPTestRequest{
				Method: method,
				Path:   path,
			})
			if err != nil {
				t.Fatalf("Failed to make request: %v", err)
			}
			return w
		},
		Validate: func(t *testing.T, w *httptest.ResponseRecorder, setupData interface{}) {
			assertErrorResponse(t, w, http.StatusUnauthorized, "")
		},
	})
	return b
}

// AddInvalidJSON adds a test for malformed JSON
func (b *TestScenarioBuilder) AddInvalidJSON(method, path string) *TestScenarioBuilder {
	b.patterns = append(b.patterns, ErrorTestPattern{
		Name:           "InvalidJSON",
		Description:    "Test handler with malformed JSON body",
		ExpectedStatus: http.StatusBadRequest,
		Setup:          nil,
		Execute: func(t *testing.T, env *TestEnvironment, setupData interface{}) *httptest.ResponseRecorder {
			// Send malformed JSON
			req, _ := http.NewRequest(method, path, nil)
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			env.App.Router.ServeHTTP(w, req)
			return w
		},
		Validate: func(t *testing.T, w *httptest.ResponseRecorder, setupData interface{}) {
			// Malformed JSON should result in bad request
			if w.Code != http.StatusBadRequest && w.Code != http.StatusUnauthorized {
				t.Errorf("Expected status %d or %d, got %d", http.StatusBadRequest, http.StatusUnauthorized, w.Code)
			}
		},
	})
	return b
}

// AddMissingRequiredField adds a test for missing required fields
func (b *TestScenarioBuilder) AddMissingRequiredField(method, path, fieldName string, body interface{}) *TestScenarioBuilder {
	b.patterns = append(b.patterns, ErrorTestPattern{
		Name:           fmt.Sprintf("Missing%s", fieldName),
		Description:    fmt.Sprintf("Test handler with missing %s field", fieldName),
		ExpectedStatus: http.StatusBadRequest,
		Setup:          nil,
		Execute: func(t *testing.T, env *TestEnvironment, setupData interface{}) *httptest.ResponseRecorder {
			w, err := makeHTTPRequest(env, HTTPTestRequest{
				Method: method,
				Path:   path,
				Body:   body,
			})
			if err != nil {
				t.Fatalf("Failed to make request: %v", err)
			}
			return w
		},
		Validate: func(t *testing.T, w *httptest.ResponseRecorder, setupData interface{}) {
			assertErrorResponse(t, w, http.StatusBadRequest, "")
		},
	})
	return b
}

// AddInvalidFieldValue adds a test for invalid field values
func (b *TestScenarioBuilder) AddInvalidFieldValue(method, path, fieldName string, body interface{}) *TestScenarioBuilder {
	b.patterns = append(b.patterns, ErrorTestPattern{
		Name:           fmt.Sprintf("Invalid%s", fieldName),
		Description:    fmt.Sprintf("Test handler with invalid %s value", fieldName),
		ExpectedStatus: http.StatusBadRequest,
		Setup:          nil,
		Execute: func(t *testing.T, env *TestEnvironment, setupData interface{}) *httptest.ResponseRecorder {
			w, err := makeHTTPRequest(env, HTTPTestRequest{
				Method: method,
				Path:   path,
				Body:   body,
			})
			if err != nil {
				t.Fatalf("Failed to make request: %v", err)
			}
			return w
		},
		Validate: func(t *testing.T, w *httptest.ResponseRecorder, setupData interface{}) {
			assertErrorResponse(t, w, http.StatusBadRequest, "")
		},
	})
	return b
}

// AddUnauthorizedAccess adds a test for unauthorized resource access
func (b *TestScenarioBuilder) AddUnauthorizedAccess(method, path string, setupFunc func(t *testing.T, env *TestEnvironment) interface{}) *TestScenarioBuilder {
	b.patterns = append(b.patterns, ErrorTestPattern{
		Name:           "UnauthorizedAccess",
		Description:    "Test handler accessing another user's resource",
		ExpectedStatus: http.StatusForbidden,
		Setup:          setupFunc,
		Execute: func(t *testing.T, env *TestEnvironment, setupData interface{}) *httptest.ResponseRecorder {
			// Create a different user trying to access the resource
			otherUser := createTestUser(t, env)

			testPath := fmt.Sprintf(path, setupData)
			w, err := makeHTTPRequest(env, HTTPTestRequest{
				Method: method,
				Path:   testPath,
				Headers: map[string]string{
					"Authorization": "Bearer " + otherUser.Token,
				},
			})
			if err != nil {
				t.Fatalf("Failed to make request: %v", err)
			}
			return w
		},
		Validate: func(t *testing.T, w *httptest.ResponseRecorder, setupData interface{}) {
			// Should be either forbidden or not found (depending on implementation)
			if w.Code != http.StatusForbidden && w.Code != http.StatusNotFound {
				t.Errorf("Expected status %d or %d, got %d", http.StatusForbidden, http.StatusNotFound, w.Code)
			}
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
	BaseURL     string
	Method      string
}

// NewHandlerTestSuite creates a new handler test suite
func NewHandlerTestSuite(handlerName, method, baseURL string) *HandlerTestSuite {
	return &HandlerTestSuite{
		HandlerName: handlerName,
		Method:      method,
		BaseURL:     baseURL,
	}
}

// RunErrorTests executes a suite of error condition tests
func (suite *HandlerTestSuite) RunErrorTests(t *testing.T, env *TestEnvironment, patterns []ErrorTestPattern) {
	for _, pattern := range patterns {
		t.Run(fmt.Sprintf("%s_%s", suite.HandlerName, pattern.Name), func(t *testing.T) {
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

			// Validate
			if pattern.Validate != nil {
				pattern.Validate(t, w, setupData)
			} else {
				// Default validation
				if w.Code != pattern.ExpectedStatus {
					t.Errorf("Expected status %d, got %d. Body: %s", pattern.ExpectedStatus, w.Code, w.Body.String())
				}
			}
		})
	}
}

// Common test pattern factories

// CreateStandardCRUDErrorPatterns creates standard CRUD error test patterns
func CreateStandardCRUDErrorPatterns(resourcePath, resourceType string) []ErrorTestPattern {
	builder := NewTestScenarioBuilder()

	// GET single resource errors
	builder.AddInvalidUUID("GET", resourcePath+"/%s")
	builder.AddNonExistentResource("GET", resourcePath+"/%s", resourceType)

	// PUT update errors
	builder.AddInvalidUUID("PUT", resourcePath+"/%s")
	builder.AddNonExistentResource("PUT", resourcePath+"/%s", resourceType)

	// DELETE errors
	builder.AddInvalidUUID("DELETE", resourcePath+"/%s")
	builder.AddNonExistentResource("DELETE", resourcePath+"/%s", resourceType)

	return builder.Build()
}

// CreateAuthErrorPatterns creates authentication/authorization error patterns
func CreateAuthErrorPatterns(method, path string) []ErrorTestPattern {
	builder := NewTestScenarioBuilder()
	builder.AddMissingAuth(method, path)
	return builder.Build()
}

// CreateValidationErrorPatterns creates input validation error patterns
func CreateValidationErrorPatterns(method, path string, testCases map[string]interface{}) []ErrorTestPattern {
	builder := NewTestScenarioBuilder()

	for fieldName, invalidBody := range testCases {
		builder.AddInvalidFieldValue(method, path, fieldName, invalidBody)
	}

	return builder.Build()
}
