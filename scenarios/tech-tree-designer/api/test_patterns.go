//go:build testing
// +build testing

package main

import (
	"database/sql"
	"fmt"
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
)

// TestScenarioBuilder builds comprehensive test scenarios using fluent interface
type TestScenarioBuilder struct {
	scenarios []TestScenario
}

// TestScenario represents a single test case
type TestScenario struct {
	Name           string
	Method         string
	Path           string
	Body           interface{}
	ExpectedStatus int
	ErrorContains  string
}

// NewTestScenarioBuilder creates a new test scenario builder
func NewTestScenarioBuilder() *TestScenarioBuilder {
	return &TestScenarioBuilder{
		scenarios: make([]TestScenario, 0),
	}
}

// AddInvalidUUID adds test cases for invalid UUID parameters
func (b *TestScenarioBuilder) AddInvalidUUID(pathTemplate string) *TestScenarioBuilder {
	b.scenarios = append(b.scenarios, TestScenario{
		Name:           "Invalid UUID format",
		Method:         "GET",
		Path:           fmt.Sprintf(pathTemplate, "invalid-uuid"),
		ExpectedStatus: http.StatusNotFound,
		ErrorContains:  "",
	})
	return b
}

// AddNonExistentResource adds test cases for non-existent resources
func (b *TestScenarioBuilder) AddNonExistentResource(pathTemplate string, resourceType string) *TestScenarioBuilder {
	nonExistentID := "00000000-0000-0000-0000-999999999999"
	b.scenarios = append(b.scenarios, TestScenario{
		Name:           fmt.Sprintf("Non-existent %s", resourceType),
		Method:         "GET",
		Path:           fmt.Sprintf(pathTemplate, nonExistentID),
		ExpectedStatus: http.StatusNotFound,
		ErrorContains:  "not found",
	})
	return b
}

// AddInvalidJSON adds test cases for malformed JSON
func (b *TestScenarioBuilder) AddInvalidJSON(path string) *TestScenarioBuilder {
	b.scenarios = append(b.scenarios, TestScenario{
		Name:           "Malformed JSON request",
		Method:         "POST",
		Path:           path,
		Body:           "invalid-json",
		ExpectedStatus: http.StatusBadRequest,
		ErrorContains:  "",
	})
	return b
}

// AddMissingRequiredFields adds test cases for missing required fields
func (b *TestScenarioBuilder) AddMissingRequiredFields(path string, body interface{}) *TestScenarioBuilder {
	b.scenarios = append(b.scenarios, TestScenario{
		Name:           "Missing required fields",
		Method:         "POST",
		Path:           path,
		Body:           body,
		ExpectedStatus: http.StatusBadRequest,
		ErrorContains:  "",
	})
	return b
}

// Build returns all configured test scenarios
func (b *TestScenarioBuilder) Build() []TestScenario {
	return b.scenarios
}

// ErrorTestPattern provides systematic error testing
type ErrorTestPattern struct {
	t      *testing.T
	router *gin.Engine
}

// NewErrorTestPattern creates a new error test pattern
func NewErrorTestPattern(t *testing.T, router *gin.Engine) *ErrorTestPattern {
	return &ErrorTestPattern{
		t:      t,
		router: router,
	}
}

// TestInvalidMethods tests all invalid HTTP methods for an endpoint
func (p *ErrorTestPattern) TestInvalidMethods(path string, validMethods ...string) {
	allMethods := []string{"GET", "POST", "PUT", "DELETE", "PATCH"}
	validMethodMap := make(map[string]bool)
	for _, m := range validMethods {
		validMethodMap[m] = true
	}

	for _, method := range allMethods {
		if validMethodMap[method] {
			continue
		}

		p.t.Run(fmt.Sprintf("Invalid method %s", method), func(t *testing.T) {
			w := makeHTTPRequest(t, p.router, method, path, nil)
			if w.Code != http.StatusMethodNotAllowed && w.Code != http.StatusNotFound {
				t.Errorf("Expected status 404 or 405 for invalid method %s, got %d", method, w.Code)
			}
		})
	}
}

// TestEdgeCases tests common edge cases
func (p *ErrorTestPattern) TestEdgeCases(scenarios []TestScenario) {
	for _, scenario := range scenarios {
		p.t.Run(scenario.Name, func(t *testing.T) {
			w := makeHTTPRequest(t, p.router, scenario.Method, scenario.Path, scenario.Body)

			if w.Code != scenario.ExpectedStatus {
				t.Errorf("Expected status %d, got %d for %s %s",
					scenario.ExpectedStatus, w.Code, scenario.Method, scenario.Path)
			}

			if scenario.ErrorContains != "" {
				assertErrorResponse(t, w, scenario.ExpectedStatus, scenario.ErrorContains)
			}
		})
	}
}

// HandlerTestSuite provides comprehensive HTTP handler testing
type HandlerTestSuite struct {
	t      *testing.T
	router *gin.Engine
	db     *sql.DB
}

// NewHandlerTestSuite creates a new handler test suite
func NewHandlerTestSuite(t *testing.T, router *gin.Engine, db *sql.DB) *HandlerTestSuite {
	return &HandlerTestSuite{
		t:      t,
		router: router,
		db:     db,
	}
}

// TestGETEndpoint tests a GET endpoint with success and error cases
func (s *HandlerTestSuite) TestGETEndpoint(path string, setupFunc func() string, expectedFields ...string) {
	s.t.Run("Success case", func(t *testing.T) {
		var resourceID string
		if setupFunc != nil {
			resourceID = setupFunc()
			if resourceID != "" {
				path = fmt.Sprintf(path, resourceID)
			}
		}

		w := makeHTTPRequest(t, s.router, "GET", path, nil)
		response := assertJSONResponse(t, w, http.StatusOK)

		// Validate expected fields
		for _, field := range expectedFields {
			if _, ok := response[field]; !ok {
				t.Errorf("Expected field '%s' not found in response", field)
			}
		}
	})

	s.t.Run("Error cases", func(t *testing.T) {
		errorPattern := NewErrorTestPattern(t, s.router)

		// Test invalid methods
		errorPattern.TestInvalidMethods(path, "GET")

		// Test edge cases if path has parameters
		if contains(path, "/:") {
			scenarios := NewTestScenarioBuilder().
				AddInvalidUUID(path).
				AddNonExistentResource(path, "resource").
				Build()
			errorPattern.TestEdgeCases(scenarios)
		}
	})
}

// TestPOSTEndpoint tests a POST endpoint with success and error cases
func (s *HandlerTestSuite) TestPOSTEndpoint(path string, validBody interface{}, expectedStatus int, expectedFields ...string) {
	s.t.Run("Success case", func(t *testing.T) {
		w := makeHTTPRequest(t, s.router, "POST", path, validBody)
		response := assertJSONResponse(t, w, expectedStatus)

		// Validate expected fields
		for _, field := range expectedFields {
			if _, ok := response[field]; !ok {
				t.Errorf("Expected field '%s' not found in response", field)
			}
		}
	})

	s.t.Run("Error cases", func(t *testing.T) {
		errorPattern := NewErrorTestPattern(t, s.router)

		// Test invalid methods
		errorPattern.TestInvalidMethods(path, "POST")

		// Test malformed JSON
		scenarios := NewTestScenarioBuilder().
			AddInvalidJSON(path).
			Build()
		errorPattern.TestEdgeCases(scenarios)
	})
}

// TestPUTEndpoint tests a PUT endpoint with success and error cases
func (s *HandlerTestSuite) TestPUTEndpoint(path string, validBody interface{}, setupFunc func() string, expectedStatus int) {
	s.t.Run("Success case", func(t *testing.T) {
		if setupFunc != nil {
			resourceID := setupFunc()
			path = fmt.Sprintf(path, resourceID)
		}

		w := makeHTTPRequest(t, s.router, "PUT", path, validBody)
		assertJSONResponse(t, w, expectedStatus)
	})

	s.t.Run("Error cases", func(t *testing.T) {
		errorPattern := NewErrorTestPattern(t, s.router)

		// Test invalid methods
		errorPattern.TestInvalidMethods(path, "PUT")

		// Test edge cases
		scenarios := NewTestScenarioBuilder().
			AddInvalidJSON(path).
			AddNonExistentResource(path, "resource").
			Build()
		errorPattern.TestEdgeCases(scenarios)
	})
}
