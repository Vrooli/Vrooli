//go:build testing
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
	Setup          func(t *testing.T, ts *TestServer) interface{}
	Execute        func(t *testing.T, ts *TestServer, setupData interface{}) HTTPTestRequest
	Validate       func(t *testing.T, w *httptest.ResponseRecorder, setupData interface{})
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
func (b *TestScenarioBuilder) AddInvalidUUID(path string, method string) *TestScenarioBuilder {
	b.patterns = append(b.patterns, ErrorTestPattern{
		Name:           "InvalidUUID",
		Description:    "Test handler with invalid UUID format",
		ExpectedStatus: http.StatusBadRequest,
		Setup:          nil,
		Execute: func(t *testing.T, ts *TestServer, setupData interface{}) HTTPTestRequest {
			return HTTPTestRequest{
				Method:  method,
				Path:    path,
				URLVars: map[string]string{"id": "invalid-uuid"},
			}
		},
		Validate: func(t *testing.T, w *httptest.ResponseRecorder, setupData interface{}) {
			assertErrorResponse(t, w, http.StatusBadRequest, "")
		},
	})
	return b
}

// AddNonExistent adds non-existent resource test pattern
func (b *TestScenarioBuilder) AddNonExistent(path string, method string) *TestScenarioBuilder {
	b.patterns = append(b.patterns, ErrorTestPattern{
		Name:           "NonExistent",
		Description:    "Test handler with non-existent resource",
		ExpectedStatus: http.StatusNotFound,
		Setup:          nil,
		Execute: func(t *testing.T, ts *TestServer, setupData interface{}) HTTPTestRequest {
			nonExistentID := uuid.New().String()
			return HTTPTestRequest{
				Method:  method,
				Path:    path,
				URLVars: map[string]string{"id": nonExistentID},
			}
		},
		Validate: func(t *testing.T, w *httptest.ResponseRecorder, setupData interface{}) {
			assertErrorResponse(t, w, http.StatusNotFound, "not found")
		},
	})
	return b
}

// AddInvalidJSON adds invalid JSON test pattern
func (b *TestScenarioBuilder) AddInvalidJSON(path string, method string) *TestScenarioBuilder {
	b.patterns = append(b.patterns, ErrorTestPattern{
		Name:           "InvalidJSON",
		Description:    "Test handler with malformed JSON input",
		ExpectedStatus: http.StatusBadRequest,
		Setup: func(t *testing.T, ts *TestServer) interface{} {
			scenario := createTestScenario(t, ts.DB, "test-invalid-json")
			return scenario
		},
		Execute: func(t *testing.T, ts *TestServer, setupData interface{}) HTTPTestRequest {
			return HTTPTestRequest{
				Method: method,
				Path:   path,
				Body:   `{"invalid": "json"`, // Malformed JSON
			}
		},
		Validate: func(t *testing.T, w *httptest.ResponseRecorder, setupData interface{}) {
			assertErrorResponse(t, w, http.StatusBadRequest, "")
		},
	})
	return b
}

// AddMissingFields adds missing required fields test pattern
func (b *TestScenarioBuilder) AddMissingFields(path string, method string, payload interface{}) *TestScenarioBuilder {
	b.patterns = append(b.patterns, ErrorTestPattern{
		Name:           "MissingRequiredFields",
		Description:    "Test handler with missing required fields",
		ExpectedStatus: http.StatusBadRequest,
		Setup:          nil,
		Execute: func(t *testing.T, ts *TestServer, setupData interface{}) HTTPTestRequest {
			return HTTPTestRequest{
				Method: method,
				Path:   path,
				Body:   payload,
			}
		},
		Validate: func(t *testing.T, w *httptest.ResponseRecorder, setupData interface{}) {
			assertErrorResponse(t, w, http.StatusBadRequest, "required")
		},
	})
	return b
}

// AddInvalidValue adds invalid value test pattern
func (b *TestScenarioBuilder) AddInvalidValue(path string, method string, payload interface{}, errorMsg string) *TestScenarioBuilder {
	b.patterns = append(b.patterns, ErrorTestPattern{
		Name:           "InvalidValue",
		Description:    "Test handler with invalid field value",
		ExpectedStatus: http.StatusBadRequest,
		Setup: func(t *testing.T, ts *TestServer) interface{} {
			scenario := createTestScenario(t, ts.DB, "test-invalid-value")
			fr := createTestFeatureRequest(t, ts.DB, scenario.ID, "Test Request")
			return map[string]interface{}{
				"scenario": scenario,
				"request":  fr,
			}
		},
		Execute: func(t *testing.T, ts *TestServer, setupData interface{}) HTTPTestRequest {
			data := setupData.(map[string]interface{})
			fr := data["request"].(*TestFeatureRequest)

			return HTTPTestRequest{
				Method:  method,
				Path:    fmt.Sprintf(path, fr.ID),
				URLVars: map[string]string{"id": fr.ID},
				Body:    payload,
			}
		},
		Validate: func(t *testing.T, w *httptest.ResponseRecorder, setupData interface{}) {
			assertErrorResponse(t, w, http.StatusBadRequest, errorMsg)
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

// RunPatterns executes all test patterns
func (b *TestScenarioBuilder) RunPatterns(t *testing.T, ts *TestServer) {
	for _, pattern := range b.patterns {
		t.Run(pattern.Name, func(t *testing.T) {
			// Setup
			var setupData interface{}
			if pattern.Setup != nil {
				setupData = pattern.Setup(t, ts)
			}

			// Cleanup
			if pattern.Cleanup != nil {
				defer pattern.Cleanup(setupData)
			}

			// Execute
			req := pattern.Execute(t, ts, setupData)
			w := makeHTTPRequest(ts.Server, req)

			// Validate
			if pattern.Validate != nil {
				pattern.Validate(t, w, setupData)
			} else if w.Code != pattern.ExpectedStatus {
				t.Errorf("Expected status %d, got %d", pattern.ExpectedStatus, w.Code)
			}
		})
	}
}
