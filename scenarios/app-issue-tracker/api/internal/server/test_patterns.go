//go:build testing
// +build testing

package server

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestScenario represents a single test scenario for error testing
type TestScenario struct {
	Name           string
	Method         string
	Path           string
	Body           interface{}
	URLVars        map[string]string
	QueryParams    map[string]string
	ExpectedStatus int
	ExpectedError  string
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

// AddInvalidIssueID adds a test for invalid issue ID format
func (b *TestScenarioBuilder) AddInvalidIssueID(method, path string) *TestScenarioBuilder {
	b.scenarios = append(b.scenarios, TestScenario{
		Name:           "Invalid Issue ID",
		Method:         method,
		Path:           path,
		URLVars:        map[string]string{"id": "not-a-valid-id"},
		ExpectedStatus: http.StatusNotFound,
		ExpectedError:  "",
	})
	return b
}

// AddNonExistentIssue adds a test for non-existent issue
func (b *TestScenarioBuilder) AddNonExistentIssue(method, path string) *TestScenarioBuilder {
	b.scenarios = append(b.scenarios, TestScenario{
		Name:           "Non-Existent Issue",
		Method:         method,
		Path:           path,
		URLVars:        map[string]string{"id": "issue-nonexistent"},
		ExpectedStatus: http.StatusNotFound,
		ExpectedError:  "",
	})
	return b
}

// AddInvalidJSON adds a test for malformed JSON
func (b *TestScenarioBuilder) AddInvalidJSON(method, path string) *TestScenarioBuilder {
	b.scenarios = append(b.scenarios, TestScenario{
		Name:           "Invalid JSON",
		Method:         method,
		Path:           path,
		Body:           "{invalid json",
		ExpectedStatus: http.StatusBadRequest,
		ExpectedError:  "",
	})
	return b
}

// AddMissingRequiredField adds a test for missing required fields
func (b *TestScenarioBuilder) AddMissingRequiredField(method, path, fieldName string) *TestScenarioBuilder {
	b.scenarios = append(b.scenarios, TestScenario{
		Name:           "Missing Required Field: " + fieldName,
		Method:         method,
		Path:           path,
		Body:           map[string]interface{}{},
		ExpectedStatus: http.StatusBadRequest,
		ExpectedError:  "",
	})
	return b
}

// AddInvalidStatus adds a test for invalid status value
func (b *TestScenarioBuilder) AddInvalidStatus(method, path string) *TestScenarioBuilder {
	b.scenarios = append(b.scenarios, TestScenario{
		Name:   "Invalid Status Value",
		Method: method,
		Path:   path,
		Body: map[string]interface{}{
			"status": "invalid-status",
		},
		ExpectedStatus: http.StatusBadRequest,
		ExpectedError:  "invalid status",
	})
	return b
}

// AddEmptyBody adds a test for empty request body
func (b *TestScenarioBuilder) AddEmptyBody(method, path string) *TestScenarioBuilder {
	b.scenarios = append(b.scenarios, TestScenario{
		Name:           "Empty Request Body",
		Method:         method,
		Path:           path,
		Body:           nil,
		ExpectedStatus: http.StatusBadRequest,
		ExpectedError:  "",
	})
	return b
}

// Build returns the built test scenarios
func (b *TestScenarioBuilder) Build() []TestScenario {
	return b.scenarios
}

// ErrorTestPattern provides systematic error testing for handlers
type ErrorTestPattern struct {
	Handler http.HandlerFunc
	Server  *Server
}

// NewErrorTestPattern creates a new error test pattern
func NewErrorTestPattern(handler http.HandlerFunc, server *Server) *ErrorTestPattern {
	return &ErrorTestPattern{
		Handler: handler,
		Server:  server,
	}
}

// RunScenarios executes all test scenarios
func (p *ErrorTestPattern) RunScenarios(t *testing.T, scenarios []TestScenario) {
	t.Helper()

	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			req := HTTPTestRequest{
				Method:      scenario.Method,
				Path:        scenario.Path,
				Body:        scenario.Body,
				URLVars:     scenario.URLVars,
				QueryParams: scenario.QueryParams,
			}

			w := makeHTTPRequest(p.Handler, req)

			if w.Code != scenario.ExpectedStatus {
				t.Errorf("Expected status %d, got %d. Response: %s",
					scenario.ExpectedStatus, w.Code, w.Body.String())
			}

			if scenario.ExpectedError != "" {
				assertErrorResponse(t, w, scenario.ExpectedStatus, scenario.ExpectedError)
			}
		})
	}
}

// HandlerTestSuite provides comprehensive testing for HTTP handlers
type HandlerTestSuite struct {
	Server      *Server
	Handler     http.HandlerFunc
	HandlerName string
}

// NewHandlerTestSuite creates a new handler test suite
func NewHandlerTestSuite(server *Server, handler http.HandlerFunc, name string) *HandlerTestSuite {
	return &HandlerTestSuite{
		Server:      server,
		Handler:     handler,
		HandlerName: name,
	}
}

// TestSuccess tests successful handler execution
func (s *HandlerTestSuite) TestSuccess(t *testing.T, req HTTPTestRequest, expectedStatus int) *httptest.ResponseRecorder {
	t.Helper()

	w := makeHTTPRequest(s.Handler, req)

	if w.Code != expectedStatus {
		t.Errorf("[%s] Expected status %d, got %d. Response: %s",
			s.HandlerName, expectedStatus, w.Code, w.Body.String())
	}

	return w
}

// TestError tests error cases
func (s *HandlerTestSuite) TestError(t *testing.T, scenarios []TestScenario) {
	t.Helper()

	pattern := NewErrorTestPattern(s.Handler, s.Server)
	pattern.RunScenarios(t, scenarios)
}

// TestEdgeCases tests edge cases like empty values, null values, etc.
func (s *HandlerTestSuite) TestEdgeCases(t *testing.T, scenarios []TestScenario) {
	t.Helper()

	pattern := NewErrorTestPattern(s.Handler, s.Server)
	pattern.RunScenarios(t, scenarios)
}

// Common test scenario patterns for reuse
var (
	// GetHandlerErrorScenarios common error scenarios for GET handlers with ID
	GetHandlerErrorScenarios = func(path string) []TestScenario {
		return NewTestScenarioBuilder().
			AddInvalidIssueID(http.MethodGet, path).
			AddNonExistentIssue(http.MethodGet, path).
			Build()
	}

	// UpdateHandlerErrorScenarios common error scenarios for UPDATE handlers
	UpdateHandlerErrorScenarios = func(path string) []TestScenario {
		return NewTestScenarioBuilder().
			AddInvalidIssueID(http.MethodPut, path).
			AddNonExistentIssue(http.MethodPut, path).
			AddInvalidJSON(http.MethodPut, path).
			Build()
	}

	// DeleteHandlerErrorScenarios common error scenarios for DELETE handlers
	DeleteHandlerErrorScenarios = func(path string) []TestScenario {
		return NewTestScenarioBuilder().
			AddInvalidIssueID(http.MethodDelete, path).
			AddNonExistentIssue(http.MethodDelete, path).
			Build()
	}

	// CreateHandlerErrorScenarios common error scenarios for CREATE handlers
	CreateHandlerErrorScenarios = func(path string) []TestScenario {
		return NewTestScenarioBuilder().
			AddInvalidJSON(http.MethodPost, path).
			AddEmptyBody(http.MethodPost, path).
			Build()
	}
)
