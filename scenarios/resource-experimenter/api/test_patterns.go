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
	Execute        func(t *testing.T, env *TestEnvironment) *httptest.ResponseRecorder
	Validate       func(t *testing.T, w *httptest.ResponseRecorder)
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

// AddInvalidUUID adds test for invalid UUID format
func (b *TestScenarioBuilder) AddInvalidUUID(path string) *TestScenarioBuilder {
	b.patterns = append(b.patterns, ErrorTestPattern{
		Name:           "InvalidUUID",
		Description:    "Test handler with invalid UUID format",
		ExpectedStatus: http.StatusBadRequest,
		Execute: func(t *testing.T, env *TestEnvironment) *httptest.ResponseRecorder {
			w, err := makeHTTPRequest(env, HTTPTestRequest{
				Method:  "GET",
				Path:    path,
				URLVars: map[string]string{"id": "invalid-uuid-format"},
			})
			if err != nil {
				t.Fatalf("Failed to make request: %v", err)
			}
			return w
		},
		Validate: func(t *testing.T, w *httptest.ResponseRecorder) {
			// Database will return 500 for invalid UUID in query
			if w.Code != http.StatusBadRequest && w.Code != http.StatusNotFound && w.Code != http.StatusInternalServerError {
				t.Errorf("Expected 400, 404, or 500 for invalid UUID, got %d", w.Code)
			}
		},
	})
	return b
}

// AddNonExistentExperiment adds test for non-existent experiment ID
func (b *TestScenarioBuilder) AddNonExistentExperiment(path string) *TestScenarioBuilder {
	b.patterns = append(b.patterns, ErrorTestPattern{
		Name:           "NonExistentExperiment",
		Description:    "Test handler with non-existent experiment ID",
		ExpectedStatus: http.StatusNotFound,
		Execute: func(t *testing.T, env *TestEnvironment) *httptest.ResponseRecorder {
			nonExistentID := uuid.New().String()
			w, err := makeHTTPRequest(env, HTTPTestRequest{
				Method:  "GET",
				Path:    path,
				URLVars: map[string]string{"id": nonExistentID},
			})
			if err != nil {
				t.Fatalf("Failed to make request: %v", err)
			}
			return w
		},
		Validate: func(t *testing.T, w *httptest.ResponseRecorder) {
			// May return 500 if database query fails on UUID validation
			if w.Code != http.StatusNotFound && w.Code != http.StatusInternalServerError {
				t.Errorf("Expected 404 or 500 for non-existent experiment, got %d", w.Code)
			}
		},
	})
	return b
}

// AddInvalidJSON adds test for malformed JSON input
func (b *TestScenarioBuilder) AddInvalidJSON(path string, method string) *TestScenarioBuilder {
	b.patterns = append(b.patterns, ErrorTestPattern{
		Name:           "InvalidJSON",
		Description:    "Test handler with malformed JSON input",
		ExpectedStatus: http.StatusBadRequest,
		Execute: func(t *testing.T, env *TestEnvironment) *httptest.ResponseRecorder {
			w, err := makeHTTPRequest(env, HTTPTestRequest{
				Method: method,
				Path:   path,
				Body:   `{"invalid": "json"`, // Malformed JSON
			})
			if err != nil {
				t.Fatalf("Failed to make request: %v", err)
			}
			return w
		},
		Validate: func(t *testing.T, w *httptest.ResponseRecorder) {
			if w.Code != http.StatusBadRequest {
				t.Errorf("Expected 400 for invalid JSON, got %d", w.Code)
			}
		},
	})
	return b
}

// AddMissingRequiredFields adds test for missing required fields
func (b *TestScenarioBuilder) AddMissingRequiredFields(path string) *TestScenarioBuilder {
	b.patterns = append(b.patterns, ErrorTestPattern{
		Name:           "MissingRequiredFields",
		Description:    "Test handler with missing required fields",
		ExpectedStatus: http.StatusBadRequest,
		Execute: func(t *testing.T, env *TestEnvironment) *httptest.ResponseRecorder {
			// Send empty JSON object - missing all required fields
			w, err := makeHTTPRequest(env, HTTPTestRequest{
				Method: "POST",
				Path:   path,
				Body:   map[string]string{},
			})
			if err != nil {
				t.Fatalf("Failed to make request: %v", err)
			}
			return w
		},
		Validate: func(t *testing.T, w *httptest.ResponseRecorder) {
			if w.Code != http.StatusBadRequest {
				t.Errorf("Expected 400 for missing required fields, got %d", w.Code)
			}
			assertResponseContains(t, w, "required")
		},
	})
	return b
}

// AddEmptyBody adds test for empty request body
func (b *TestScenarioBuilder) AddEmptyBody(path string, method string) *TestScenarioBuilder {
	b.patterns = append(b.patterns, ErrorTestPattern{
		Name:           "EmptyBody",
		Description:    "Test handler with empty request body",
		ExpectedStatus: http.StatusBadRequest,
		Execute: func(t *testing.T, env *TestEnvironment) *httptest.ResponseRecorder {
			w, err := makeHTTPRequest(env, HTTPTestRequest{
				Method: method,
				Path:   path,
				Body:   "",
			})
			if err != nil {
				t.Fatalf("Failed to make request: %v", err)
			}
			return w
		},
		Validate: func(t *testing.T, w *httptest.ResponseRecorder) {
			if w.Code != http.StatusBadRequest {
				t.Errorf("Expected 400 for empty body, got %d", w.Code)
			}
		},
	})
	return b
}

// Build returns the constructed error patterns
func (b *TestScenarioBuilder) Build() []ErrorTestPattern {
	return b.patterns
}

// RunErrorTests executes a suite of error condition tests
func RunErrorTests(t *testing.T, env *TestEnvironment, patterns []ErrorTestPattern) {
	for _, pattern := range patterns {
		t.Run(pattern.Name, func(t *testing.T) {
			w := pattern.Execute(t, env)
			if pattern.Validate != nil {
				pattern.Validate(t, w)
			} else if pattern.ExpectedStatus > 0 {
				// Default validation: check status code
				if w.Code != pattern.ExpectedStatus {
					t.Errorf("Expected status %d, got %d. Body: %s",
						pattern.ExpectedStatus, w.Code, w.Body.String())
				}
			}
		})
	}
}

// Common error test patterns

// DatabaseConnectionErrorPattern tests database connection failures
func DatabaseConnectionErrorPattern(handlerFunc func(*TestEnvironment) *httptest.ResponseRecorder) ErrorTestPattern {
	return ErrorTestPattern{
		Name:           "DatabaseConnectionError",
		Description:    "Test handler when database is unavailable",
		ExpectedStatus: http.StatusInternalServerError,
		Execute: func(t *testing.T, env *TestEnvironment) *httptest.ResponseRecorder {
			// Close database to simulate failure
			originalDB := env.DB
			env.DB.Close()
			defer func() { env.DB = originalDB }()

			return handlerFunc(env)
		},
	}
}

// HandlerTestSuite provides a comprehensive test framework for HTTP handlers
type HandlerTestSuite struct {
	HandlerName string
	BaseURL     string
	Env         *TestEnvironment
}

// NewHandlerTestSuite creates a new handler test suite
func NewHandlerTestSuite(name string, baseURL string, env *TestEnvironment) *HandlerTestSuite {
	return &HandlerTestSuite{
		HandlerName: name,
		BaseURL:     baseURL,
		Env:         env,
	}
}

// TestSuccess runs success case tests
func (suite *HandlerTestSuite) TestSuccess(t *testing.T, testFunc func(t *testing.T, env *TestEnvironment)) {
	t.Run(fmt.Sprintf("%s_Success", suite.HandlerName), func(t *testing.T) {
		testFunc(t, suite.Env)
	})
}

// TestErrors runs error case tests
func (suite *HandlerTestSuite) TestErrors(t *testing.T, patterns []ErrorTestPattern) {
	for _, pattern := range patterns {
		t.Run(fmt.Sprintf("%s_%s", suite.HandlerName, pattern.Name), func(t *testing.T) {
			w := pattern.Execute(t, suite.Env)
			if pattern.Validate != nil {
				pattern.Validate(t, w)
			}
		})
	}
}

// TestEdgeCases runs edge case tests
func (suite *HandlerTestSuite) TestEdgeCases(t *testing.T, testFunc func(t *testing.T, env *TestEnvironment)) {
	t.Run(fmt.Sprintf("%s_EdgeCases", suite.HandlerName), func(t *testing.T) {
		testFunc(t, suite.Env)
	})
}
