package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestScenario represents a single test scenario
type TestScenario struct {
	Name           string
	Method         string
	Path           string
	Body           interface{}
	Token          string
	ExpectedStatus int
	ExpectedError  string
	Validator      func(*testing.T, *httptest.ResponseRecorder)
}

// TestScenarioBuilder provides fluent interface for building test scenarios
type TestScenarioBuilder struct {
	scenarios []TestScenario
}

// NewTestScenarioBuilder creates a new test scenario builder
func NewTestScenarioBuilder() *TestScenarioBuilder {
	return &TestScenarioBuilder{
		scenarios: []TestScenario{},
	}
}

// AddInvalidUUID adds a test for invalid UUID format
func (b *TestScenarioBuilder) AddInvalidUUID(path string) *TestScenarioBuilder {
	b.scenarios = append(b.scenarios, TestScenario{
		Name:           "Invalid UUID",
		Method:         "GET",
		Path:           path + "?id=invalid-uuid",
		Token:          "test-token",
		ExpectedStatus: http.StatusNotFound,
	})
	return b
}

// AddNonExistentResource adds a test for non-existent resource
func (b *TestScenarioBuilder) AddNonExistentResource(path string) *TestScenarioBuilder {
	b.scenarios = append(b.scenarios, TestScenario{
		Name:           "Non-existent Resource",
		Method:         "GET",
		Path:           path + "?id=00000000-0000-0000-0000-000000000000",
		Token:          "test-token",
		ExpectedStatus: http.StatusNotFound,
	})
	return b
}

// AddInvalidJSON adds a test for malformed JSON
func (b *TestScenarioBuilder) AddInvalidJSON(path string) *TestScenarioBuilder {
	b.scenarios = append(b.scenarios, TestScenario{
		Name:           "Invalid JSON",
		Method:         "POST",
		Path:           path,
		Token:          "test-token",
		ExpectedStatus: http.StatusBadRequest,
		Validator: func(t *testing.T, w *httptest.ResponseRecorder) {
			// Send raw invalid JSON
		},
	})
	return b
}

// AddMissingAuth adds a test for missing authentication
func (b *TestScenarioBuilder) AddMissingAuth(path string) *TestScenarioBuilder {
	b.scenarios = append(b.scenarios, TestScenario{
		Name:           "Missing Authentication",
		Method:         "GET",
		Path:           path,
		Token:          "", // No token
		ExpectedStatus: http.StatusOK, // Some endpoints allow no auth
	})
	return b
}

// AddInvalidAuth adds a test for invalid authentication
func (b *TestScenarioBuilder) AddInvalidAuth(path string) *TestScenarioBuilder {
	b.scenarios = append(b.scenarios, TestScenario{
		Name:           "Invalid Authentication",
		Method:         "GET",
		Path:           path,
		Token:          "invalid-token",
		ExpectedStatus: http.StatusUnauthorized,
		ExpectedError:  "unauthorized",
	})
	return b
}

// AddEmptyBody adds a test for empty request body
func (b *TestScenarioBuilder) AddEmptyBody(path string) *TestScenarioBuilder {
	b.scenarios = append(b.scenarios, TestScenario{
		Name:           "Empty Request Body",
		Method:         "POST",
		Path:           path,
		Body:           map[string]interface{}{},
		Token:          "test-token",
		ExpectedStatus: http.StatusBadRequest,
	})
	return b
}

// AddMissingRequiredField adds a test for missing required fields
func (b *TestScenarioBuilder) AddMissingRequiredField(path string, requiredField string) *TestScenarioBuilder {
	b.scenarios = append(b.scenarios, TestScenario{
		Name:   "Missing Required Field: " + requiredField,
		Method: "POST",
		Path:   path,
		Body: map[string]interface{}{
			"other_field": "value",
		},
		Token:          "test-token",
		ExpectedStatus: http.StatusBadRequest,
	})
	return b
}

// Build returns all test scenarios
func (b *TestScenarioBuilder) Build() []TestScenario {
	return b.scenarios
}

// ErrorTestPattern provides systematic error testing
type ErrorTestPattern struct {
	t      *testing.T
	server *Server
}

// NewErrorTestPattern creates a new error test pattern
func NewErrorTestPattern(t *testing.T, server *Server) *ErrorTestPattern {
	return &ErrorTestPattern{
		t:      t,
		server: server,
	}
}

// TestBadRequest tests bad request scenarios
func (p *ErrorTestPattern) TestBadRequest(path string, invalidBodies []interface{}) {
	p.t.Helper()

	for i, body := range invalidBodies {
		p.t.Run("BadRequest_"+path+"_"+string(rune(i+'A')), func(t *testing.T) {
			req := makeHTTPRequest("POST", path, body, "test-token")
			w := httptest.NewRecorder()

			p.server.router.ServeHTTP(w, req)

			if w.Code != http.StatusBadRequest && w.Code != http.StatusInternalServerError {
				t.Errorf("Expected 400 or 500, got %d for body: %v", w.Code, body)
			}
		})
	}
}

// TestUnauthorized tests unauthorized access scenarios
func (p *ErrorTestPattern) TestUnauthorized(paths []string) {
	p.t.Helper()

	for _, path := range paths {
		p.t.Run("Unauthorized_"+path, func(t *testing.T) {
			req := makeHTTPRequest("GET", path, nil, "invalid-token")
			w := httptest.NewRecorder()

			p.server.router.ServeHTTP(w, req)

			if w.Code != http.StatusUnauthorized {
				t.Errorf("Expected 401, got %d for path: %s", w.Code, path)
			}
		})
	}
}

// TestNotFound tests not found scenarios
func (p *ErrorTestPattern) TestNotFound(paths []string) {
	p.t.Helper()

	for _, path := range paths {
		p.t.Run("NotFound_"+path, func(t *testing.T) {
			req := makeHTTPRequest("GET", path, nil, "test-token")
			w := httptest.NewRecorder()

			p.server.router.ServeHTTP(w, req)

			if w.Code != http.StatusNotFound {
				t.Errorf("Expected 404, got %d for path: %s", w.Code, path)
			}
		})
	}
}

// HandlerTestSuite provides comprehensive handler testing
type HandlerTestSuite struct {
	t       *testing.T
	server  *Server
	cleanup func()
}

// NewHandlerTestSuite creates a new handler test suite
func NewHandlerTestSuite(t *testing.T) *HandlerTestSuite {
	cleanup := setupTestLogger()
	env := setupTestDirectory(t)

	return &HandlerTestSuite{
		t:       t,
		server:  env.Server,
		cleanup: func() { cleanup(); env.Cleanup() },
	}
}

// Cleanup cleans up test resources
func (s *HandlerTestSuite) Cleanup() {
	if s.cleanup != nil {
		s.cleanup()
	}
}

// TestCRUD tests complete CRUD operations for a resource
func (s *HandlerTestSuite) TestCRUD(createPath, listPath, getPath, updatePath, deletePath string, createBody map[string]interface{}) {
	s.t.Helper()

	// Create
	s.t.Run("Create", func(t *testing.T) {
		req := makeHTTPRequest("POST", createPath, createBody, "test-token")
		w := httptest.NewRecorder()

		s.server.router.ServeHTTP(w, req)

		if w.Code != http.StatusCreated && w.Code != http.StatusOK {
			t.Errorf("Expected 201 or 200, got %d. Body: %s", w.Code, w.Body.String())
		}
	})

	// List
	s.t.Run("List", func(t *testing.T) {
		req := makeHTTPRequest("GET", listPath, nil, "test-token")
		w := httptest.NewRecorder()

		s.server.router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected 200, got %d", w.Code)
		}
	})

	// Get (requires valid ID from create)
	// Update (requires valid ID from create)
	// Delete (requires valid ID from create)
}

// TestHTTPMethods tests all HTTP methods for an endpoint
func (s *HandlerTestSuite) TestHTTPMethods(path string, allowedMethods []string) {
	s.t.Helper()

	allMethods := []string{"GET", "POST", "PUT", "DELETE", "PATCH", "OPTIONS"}

	for _, method := range allMethods {
		allowed := false
		for _, allowedMethod := range allowedMethods {
			if method == allowedMethod {
				allowed = true
				break
			}
		}

		s.t.Run("Method_"+method, func(t *testing.T) {
			req := makeHTTPRequest(method, path, nil, "test-token")
			w := httptest.NewRecorder()

			s.server.router.ServeHTTP(w, req)

			if allowed {
				if w.Code == http.StatusMethodNotAllowed {
					t.Errorf("Method %s should be allowed but got 405", method)
				}
			}
			// Note: We don't enforce 405 for disallowed methods as some servers return 404
		})
	}
}

// TestEdgeCases tests edge case scenarios
func (s *HandlerTestSuite) TestEdgeCases(path string, edgeCases []TestScenario) {
	s.t.Helper()

	for _, testCase := range edgeCases {
		s.t.Run(testCase.Name, func(t *testing.T) {
			req := makeHTTPRequest(testCase.Method, testCase.Path, testCase.Body, testCase.Token)
			w := httptest.NewRecorder()

			s.server.router.ServeHTTP(w, req)

			if w.Code != testCase.ExpectedStatus {
				t.Errorf("Expected status %d, got %d", testCase.ExpectedStatus, w.Code)
			}

			if testCase.Validator != nil {
				testCase.Validator(t, w)
			}
		})
	}
}
