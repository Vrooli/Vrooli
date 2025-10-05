package main

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
)

// TestScenario represents a comprehensive test scenario
type TestScenario struct {
	Name           string
	Description    string
	Setup          func(t *testing.T, env *TestEnvironment) interface{}
	Execute        func(t *testing.T, env *TestEnvironment, setupData interface{}) *httptest.ResponseRecorder
	Validate       func(t *testing.T, env *TestEnvironment, w *httptest.ResponseRecorder, setupData interface{})
	Cleanup        func(t *testing.T, env *TestEnvironment, setupData interface{})
	ExpectedStatus int
}

// TestScenarioBuilder provides fluent interface for building test scenarios
type TestScenarioBuilder struct {
	scenarios []TestScenario
	basePath  string
}

// NewTestScenarioBuilder creates a new test scenario builder
func NewTestScenarioBuilder(basePath string) *TestScenarioBuilder {
	return &TestScenarioBuilder{
		scenarios: []TestScenario{},
		basePath:  basePath,
	}
}

// AddInvalidUUID adds a test for invalid UUID format
func (b *TestScenarioBuilder) AddInvalidUUID(path string, method string) *TestScenarioBuilder {
	b.scenarios = append(b.scenarios, TestScenario{
		Name:           "InvalidUUID",
		Description:    "Test handler with invalid UUID format",
		ExpectedStatus: http.StatusNotFound, // Gin returns 404 for invalid UUID in path
		Setup:          nil,
		Execute: func(t *testing.T, env *TestEnvironment, setupData interface{}) *httptest.ResponseRecorder {
			w, err := makeHTTPRequest(env, HTTPTestRequest{
				Method: method,
				Path:   path + "/invalid-uuid-format",
			})
			if err != nil {
				t.Fatalf("Failed to make request: %v", err)
			}
			return w
		},
		Validate: func(t *testing.T, env *TestEnvironment, w *httptest.ResponseRecorder, setupData interface{}) {
			// Response should indicate error
			if w.Code != http.StatusNotFound && w.Code != http.StatusBadRequest {
				t.Errorf("Expected status 404 or 400 for invalid UUID, got %d", w.Code)
			}
		},
	})
	return b
}

// AddNonExistentResource adds a test for non-existent resource
func (b *TestScenarioBuilder) AddNonExistentResource(path string, method string, resourceType string) *TestScenarioBuilder {
	b.scenarios = append(b.scenarios, TestScenario{
		Name:           fmt.Sprintf("NonExistent%s", resourceType),
		Description:    fmt.Sprintf("Test handler with non-existent %s ID", resourceType),
		ExpectedStatus: http.StatusNotFound,
		Setup:          nil,
		Execute: func(t *testing.T, env *TestEnvironment, setupData interface{}) *httptest.ResponseRecorder {
			nonExistentID := uuid.New().String()
			w, err := makeHTTPRequest(env, HTTPTestRequest{
				Method: method,
				Path:   path + "/" + nonExistentID,
			})
			if err != nil {
				t.Fatalf("Failed to make request: %v", err)
			}
			return w
		},
		Validate: func(t *testing.T, env *TestEnvironment, w *httptest.ResponseRecorder, setupData interface{}) {
			assertErrorResponse(t, w, http.StatusNotFound, "error")
		},
	})
	return b
}

// AddInvalidJSON adds a test for malformed JSON request
func (b *TestScenarioBuilder) AddInvalidJSON(path string, method string) *TestScenarioBuilder {
	b.scenarios = append(b.scenarios, TestScenario{
		Name:           "InvalidJSON",
		Description:    "Test handler with malformed JSON body",
		ExpectedStatus: http.StatusBadRequest,
		Setup:          nil,
		Execute: func(t *testing.T, env *TestEnvironment, setupData interface{}) *httptest.ResponseRecorder {
			req, err := http.NewRequest(method, path, bytes.NewBufferString("{invalid json"))
			if err != nil {
				t.Fatalf("Failed to create request: %v", err)
			}
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			env.Router.ServeHTTP(w, req)
			return w
		},
		Validate: func(t *testing.T, env *TestEnvironment, w *httptest.ResponseRecorder, setupData interface{}) {
			if w.Code != http.StatusBadRequest {
				t.Errorf("Expected status 400 for invalid JSON, got %d", w.Code)
			}
		},
	})
	return b
}

// AddMissingRequiredField adds a test for missing required field
func (b *TestScenarioBuilder) AddMissingRequiredField(path string, method string, fieldName string, bodyWithoutField interface{}) *TestScenarioBuilder {
	b.scenarios = append(b.scenarios, TestScenario{
		Name:           fmt.Sprintf("Missing%s", fieldName),
		Description:    fmt.Sprintf("Test handler with missing required field: %s", fieldName),
		ExpectedStatus: http.StatusBadRequest,
		Setup:          nil,
		Execute: func(t *testing.T, env *TestEnvironment, setupData interface{}) *httptest.ResponseRecorder {
			w, err := makeHTTPRequest(env, HTTPTestRequest{
				Method: method,
				Path:   path,
				Body:   bodyWithoutField,
			})
			if err != nil {
				t.Fatalf("Failed to make request: %v", err)
			}
			return w
		},
		Validate: func(t *testing.T, env *TestEnvironment, w *httptest.ResponseRecorder, setupData interface{}) {
			assertErrorResponse(t, w, http.StatusBadRequest, "error")
		},
	})
	return b
}

// AddEmptyBody adds a test for empty request body
func (b *TestScenarioBuilder) AddEmptyBody(path string, method string) *TestScenarioBuilder {
	b.scenarios = append(b.scenarios, TestScenario{
		Name:           "EmptyBody",
		Description:    "Test handler with empty request body",
		ExpectedStatus: http.StatusBadRequest,
		Setup:          nil,
		Execute: func(t *testing.T, env *TestEnvironment, setupData interface{}) *httptest.ResponseRecorder {
			req, err := http.NewRequest(method, path, nil)
			if err != nil {
				t.Fatalf("Failed to create request: %v", err)
			}
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			env.Router.ServeHTTP(w, req)
			return w
		},
		Validate: func(t *testing.T, env *TestEnvironment, w *httptest.ResponseRecorder, setupData interface{}) {
			if w.Code != http.StatusBadRequest && w.Code != http.StatusOK {
				t.Errorf("Expected status 400 or 200 for empty body, got %d", w.Code)
			}
		},
	})
	return b
}

// AddBoundaryValue adds a test for boundary value testing
func (b *TestScenarioBuilder) AddBoundaryValue(name string, path string, method string, body interface{}, expectedStatus int) *TestScenarioBuilder {
	b.scenarios = append(b.scenarios, TestScenario{
		Name:           fmt.Sprintf("Boundary_%s", name),
		Description:    fmt.Sprintf("Test boundary value: %s", name),
		ExpectedStatus: expectedStatus,
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
		Validate: func(t *testing.T, env *TestEnvironment, w *httptest.ResponseRecorder, setupData interface{}) {
			if w.Code != expectedStatus {
				t.Errorf("Expected status %d for boundary value test, got %d", expectedStatus, w.Code)
			}
		},
	})
	return b
}

// Build returns all scenarios
func (b *TestScenarioBuilder) Build() []TestScenario {
	return b.scenarios
}

// RunScenarios executes all scenarios
func (b *TestScenarioBuilder) RunScenarios(t *testing.T, env *TestEnvironment) {
	for _, scenario := range b.scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			var setupData interface{}
			if scenario.Setup != nil {
				setupData = scenario.Setup(t, env)
			}

			if scenario.Cleanup != nil {
				defer scenario.Cleanup(t, env, setupData)
			}

			w := scenario.Execute(t, env, setupData)

			if scenario.ExpectedStatus > 0 && w.Code != scenario.ExpectedStatus {
				t.Errorf("Expected status %d, got %d. Body: %s", scenario.ExpectedStatus, w.Code, w.Body.String())
			}

			if scenario.Validate != nil {
				scenario.Validate(t, env, w, setupData)
			}
		})
	}
}

// Common test patterns for the contact book API

// PersonEndpointPatterns returns common test patterns for person endpoints
func PersonEndpointPatterns() *TestScenarioBuilder {
	builder := NewTestScenarioBuilder("/api/v1/contacts")

	// Error patterns
	builder.AddInvalidUUID("/api/v1/contacts", "GET")
	builder.AddNonExistentResource("/api/v1/contacts", "GET", "Person")
	builder.AddInvalidJSON("/api/v1/contacts", "POST")
	builder.AddMissingRequiredField("/api/v1/contacts", "POST", "FullName", map[string]interface{}{
		"emails": []string{"test@example.com"},
	})

	// Boundary value testing
	builder.AddBoundaryValue("EmptyFullName", "/api/v1/contacts", "POST", CreatePersonRequest{
		FullName: "",
		Emails:   []string{"test@example.com"},
	}, http.StatusBadRequest)

	builder.AddBoundaryValue("VeryLongFullName", "/api/v1/contacts", "POST", CreatePersonRequest{
		FullName: string(make([]byte, 1000)),
		Emails:   []string{"test@example.com"},
	}, http.StatusCreated)

	return builder
}

// RelationshipEndpointPatterns returns common test patterns for relationship endpoints
func RelationshipEndpointPatterns() *TestScenarioBuilder {
	builder := NewTestScenarioBuilder("/api/v1/relationships")

	builder.AddInvalidJSON("/api/v1/relationships", "POST")
	builder.AddMissingRequiredField("/api/v1/relationships", "POST", "FromPersonID", map[string]interface{}{
		"to_person_id":      uuid.New().String(),
		"relationship_type": "friend",
	})

	return builder
}

// SearchEndpointPatterns returns common test patterns for search endpoints
func SearchEndpointPatterns() *TestScenarioBuilder {
	builder := NewTestScenarioBuilder("/api/v1/search")

	builder.AddInvalidJSON("/api/v1/search", "POST")
	builder.AddEmptyBody("/api/v1/search", "POST")

	return builder
}
