package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

// ErrorTestPattern defines a systematic approach to testing error conditions
type ErrorTestPattern struct {
	Name           string
	Description    string
	ExpectedStatus int
	Setup          func(t *testing.T) interface{}
	Execute        func(t *testing.T, setupData interface{}) HTTPTestRequest
	Validate       func(t *testing.T, w *httptest.ResponseRecorder, setupData interface{})
	Cleanup        func(setupData interface{})
}

// TestScenarioBuilder provides fluent interface for building test scenarios
type TestScenarioBuilder struct {
	patterns []ErrorTestPattern
}

// NewTestScenarioBuilder creates a new test scenario builder
func NewTestScenarioBuilder() *TestScenarioBuilder {
	return &TestScenarioBuilder{
		patterns: []ErrorTestPattern{},
	}
}

// AddInvalidUUID adds an invalid UUID test pattern
func (b *TestScenarioBuilder) AddInvalidUUID(urlPath string, method string) *TestScenarioBuilder {
	b.patterns = append(b.patterns, ErrorTestPattern{
		Name:           "InvalidUUID",
		Description:    "Test handler with invalid UUID format",
		ExpectedStatus: http.StatusBadRequest,
		Execute: func(t *testing.T, setupData interface{}) HTTPTestRequest {
			return HTTPTestRequest{
				Method:  method,
				Path:    urlPath,
				URLVars: map[string]string{"id": "invalid-uuid-format"},
			}
		},
	})
	return b
}

// AddNonExistentIssue adds a non-existent issue test pattern
func (b *TestScenarioBuilder) AddNonExistentIssue(urlPath string, method string) *TestScenarioBuilder {
	b.patterns = append(b.patterns, ErrorTestPattern{
		Name:           "NonExistentIssue",
		Description:    "Test handler with non-existent issue ID",
		ExpectedStatus: http.StatusNotFound,
		Execute: func(t *testing.T, setupData interface{}) HTTPTestRequest {
			nonExistentID := uuid.New().String()
			return HTTPTestRequest{
				Method:  method,
				Path:    urlPath,
				URLVars: map[string]string{"id": nonExistentID},
			}
		},
	})
	return b
}

// AddInvalidJSON adds an invalid JSON test pattern
func (b *TestScenarioBuilder) AddInvalidJSON(urlPath string, method string) *TestScenarioBuilder {
	b.patterns = append(b.patterns, ErrorTestPattern{
		Name:           "InvalidJSON",
		Description:    "Test handler with malformed JSON",
		ExpectedStatus: http.StatusBadRequest,
		Execute: func(t *testing.T, setupData interface{}) HTTPTestRequest {
			// Use raw string body instead of struct to bypass JSON encoding
			return HTTPTestRequest{
				Method:  method,
				Path:    urlPath,
				Headers: map[string]string{"Content-Type": "application/json"},
			}
		},
	})
	return b
}

// AddMissingRequiredField adds a missing field test pattern
func (b *TestScenarioBuilder) AddMissingRequiredField(urlPath string, method string, fieldName string) *TestScenarioBuilder {
	b.patterns = append(b.patterns, ErrorTestPattern{
		Name:           fmt.Sprintf("Missing%sField", fieldName),
		Description:    fmt.Sprintf("Test handler with missing %s field", fieldName),
		ExpectedStatus: http.StatusBadRequest,
		Execute: func(t *testing.T, setupData interface{}) HTTPTestRequest {
			// Create incomplete body based on the field
			body := map[string]interface{}{}
			// Don't include the required field
			return HTTPTestRequest{
				Method: method,
				Path:   urlPath,
				Body:   body,
			}
		},
	})
	return b
}

// AddEmptyBody adds an empty body test pattern
func (b *TestScenarioBuilder) AddEmptyBody(urlPath string, method string) *TestScenarioBuilder {
	b.patterns = append(b.patterns, ErrorTestPattern{
		Name:           "EmptyBody",
		Description:    "Test handler with empty request body",
		ExpectedStatus: http.StatusBadRequest,
		Execute: func(t *testing.T, setupData interface{}) HTTPTestRequest {
			return HTTPTestRequest{
				Method: method,
				Path:   urlPath,
				Body:   nil,
			}
		},
	})
	return b
}

// Build returns the constructed patterns
func (b *TestScenarioBuilder) Build() []ErrorTestPattern {
	return b.patterns
}

// HandlerTestSuite provides comprehensive test framework for HTTP handlers
type HandlerTestSuite struct {
	HandlerName string
	Router      *mux.Router
	BaseURL     string
}

// NewHandlerTestSuite creates a new handler test suite
func NewHandlerTestSuite(name string, router *mux.Router, baseURL string) *HandlerTestSuite {
	return &HandlerTestSuite{
		HandlerName: name,
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
			req := pattern.Execute(t, setupData)
			w, err := makeHTTPRequest(req, suite.Router)
			if err != nil {
				t.Fatalf("Failed to make HTTP request: %v", err)
			}

			// Validate status code
			if w.Code != pattern.ExpectedStatus {
				t.Errorf("Expected status %d, got %d. Body: %s",
					pattern.ExpectedStatus, w.Code, w.Body.String())
			}

			// Custom validation
			if pattern.Validate != nil {
				pattern.Validate(t, w, setupData)
			}
		})
	}
}

// RunSuccessTest executes a single success test case
func (suite *HandlerTestSuite) RunSuccessTest(t *testing.T, name string, req HTTPTestRequest, validate func(*testing.T, *httptest.ResponseRecorder)) {
	t.Run(fmt.Sprintf("%s_%s", suite.HandlerName, name), func(t *testing.T) {
		w, err := makeHTTPRequest(req, suite.Router)
		if err != nil {
			t.Fatalf("Failed to make HTTP request: %v", err)
		}
		validate(t, w)
	})
}
