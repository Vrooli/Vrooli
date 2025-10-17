// +build testing

package main

import (
	"database/sql"
	"fmt"
	"net/http"
	"testing"
)

// TestScenario represents a reusable test scenario
type TestScenario struct {
	Name           string
	Description    string
	Request        HTTPTestRequest
	ExpectedStatus int
	ValidateBody   func(t *testing.T, body string)
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

// AddInvalidUUID adds a test scenario for invalid UUID handling
func (b *TestScenarioBuilder) AddInvalidUUID(path string) *TestScenarioBuilder {
	b.scenarios = append(b.scenarios, TestScenario{
		Name:        "InvalidUUID",
		Description: "Test handler with invalid UUID format",
		Request: HTTPTestRequest{
			Method:  "GET",
			Path:    path,
			URLVars: map[string]string{"id": "invalid-uuid-format"},
		},
		ExpectedStatus: http.StatusNotFound, // Most handlers return 404 for invalid IDs
	})
	return b
}

// AddNonExistentPattern adds a test scenario for non-existent pattern ID
func (b *TestScenarioBuilder) AddNonExistentPattern(path string) *TestScenarioBuilder {
	b.scenarios = append(b.scenarios, TestScenario{
		Name:        "NonExistentPattern",
		Description: "Test handler with non-existent pattern ID",
		Request: HTTPTestRequest{
			Method:  "GET",
			Path:    path,
			URLVars: map[string]string{"id": "non-existent-pattern-id"},
		},
		ExpectedStatus: http.StatusNotFound,
	})
	return b
}

// AddNonExistentRecipe adds a test scenario for non-existent recipe ID
func (b *TestScenarioBuilder) AddNonExistentRecipe(path string) *TestScenarioBuilder {
	b.scenarios = append(b.scenarios, TestScenario{
		Name:        "NonExistentRecipe",
		Description: "Test handler with non-existent recipe ID",
		Request: HTTPTestRequest{
			Method:  "GET",
			Path:    path,
			URLVars: map[string]string{"id": "non-existent-recipe-id"},
		},
		ExpectedStatus: http.StatusNotFound,
	})
	return b
}

// AddInvalidJSON adds a test scenario for invalid JSON body
func (b *TestScenarioBuilder) AddInvalidJSON(path string, method string) *TestScenarioBuilder {
	b.scenarios = append(b.scenarios, TestScenario{
		Name:        "InvalidJSON",
		Description: "Test handler with malformed JSON",
		Request: HTTPTestRequest{
			Method: method,
			Path:   path,
			Body:   "invalid-json-{{{",
		},
		ExpectedStatus: http.StatusBadRequest,
	})
	return b
}

// AddMissingRequiredParam adds a test scenario for missing required query parameter
func (b *TestScenarioBuilder) AddMissingRequiredParam(path string, paramName string) *TestScenarioBuilder {
	b.scenarios = append(b.scenarios, TestScenario{
		Name:        fmt.Sprintf("Missing_%s", paramName),
		Description: fmt.Sprintf("Test handler with missing required parameter: %s", paramName),
		Request: HTTPTestRequest{
			Method: "GET",
			Path:   path,
			Query:  map[string]string{},
		},
		ExpectedStatus: http.StatusBadRequest,
	})
	return b
}

// AddEmptyQueryParam adds a test scenario for empty query parameter
func (b *TestScenarioBuilder) AddEmptyQueryParam(path string, paramName string) *TestScenarioBuilder {
	b.scenarios = append(b.scenarios, TestScenario{
		Name:        fmt.Sprintf("Empty_%s", paramName),
		Description: fmt.Sprintf("Test handler with empty parameter: %s", paramName),
		Request: HTTPTestRequest{
			Method: "GET",
			Path:   path,
			Query:  map[string]string{paramName: ""},
		},
		ExpectedStatus: http.StatusOK, // Empty params often default to showing all
	})
	return b
}

// Build returns the constructed test scenarios
func (b *TestScenarioBuilder) Build() []TestScenario {
	return b.scenarios
}

// ErrorTestPattern defines a systematic approach to testing error conditions
type ErrorTestPattern struct {
	Name           string
	Description    string
	ExpectedStatus int
	Setup          func(t *testing.T) interface{}
	Execute        func(t *testing.T, setupData interface{}) HTTPTestRequest
	Validate       func(t *testing.T, w interface{}, setupData interface{})
	Cleanup        func(setupData interface{})
}

// HandlerTestSuite provides comprehensive test framework for HTTP handlers
type HandlerTestSuite struct {
	HandlerName string
	Handler     http.HandlerFunc
	BaseURL     string
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
			w := makeHTTPRequest(suite.Handler, req)

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

// Common error patterns

// DatabaseErrorPattern tests database connection errors
func DatabaseErrorPattern(urlPath string) ErrorTestPattern {
	return ErrorTestPattern{
		Name:           "DatabaseError",
		Description:    "Test handler when database is unavailable",
		ExpectedStatus: http.StatusInternalServerError,
		Setup: func(t *testing.T) interface{} {
			// Close database connection to simulate error
			originalDB := db
			if db != nil {
				db.Close()
				db = nil
			}
			return originalDB
		},
		Execute: func(t *testing.T, setupData interface{}) HTTPTestRequest {
			return HTTPTestRequest{
				Method: "GET",
				Path:   urlPath,
			}
		},
		Cleanup: func(setupData interface{}) {
			// Restore database connection
			if setupData != nil {
				db = setupData.(*sql.DB)
			}
		},
	}
}

// InvalidParameterPattern tests invalid parameter handling
func InvalidParameterPattern(urlPath string, paramName string, invalidValue string) ErrorTestPattern {
	return ErrorTestPattern{
		Name:           fmt.Sprintf("Invalid_%s", paramName),
		Description:    fmt.Sprintf("Test handler with invalid %s parameter", paramName),
		ExpectedStatus: http.StatusBadRequest,
		Execute: func(t *testing.T, setupData interface{}) HTTPTestRequest {
			return HTTPTestRequest{
				Method: "GET",
				Path:   urlPath,
				Query:  map[string]string{paramName: invalidValue},
			}
		},
	}
}

// PaginationTestScenarios creates test scenarios for pagination
func PaginationTestScenarios(basePath string) []TestScenario {
	return []TestScenario{
		{
			Name:        "Pagination_ValidLimitOffset",
			Description: "Test valid limit and offset parameters",
			Request: HTTPTestRequest{
				Method: "GET",
				Path:   basePath,
				Query:  map[string]string{"limit": "10", "offset": "0"},
			},
			ExpectedStatus: http.StatusOK,
		},
		{
			Name:        "Pagination_LargeLimit",
			Description: "Test limit exceeding maximum",
			Request: HTTPTestRequest{
				Method: "GET",
				Path:   basePath,
				Query:  map[string]string{"limit": "1000"},
			},
			ExpectedStatus: http.StatusOK, // Should clamp to max
		},
		{
			Name:        "Pagination_NegativeOffset",
			Description: "Test negative offset",
			Request: HTTPTestRequest{
				Method: "GET",
				Path:   basePath,
				Query:  map[string]string{"offset": "-1"},
			},
			ExpectedStatus: http.StatusOK, // Should default to 0
		},
		{
			Name:        "Pagination_InvalidLimit",
			Description: "Test non-numeric limit",
			Request: HTTPTestRequest{
				Method: "GET",
				Path:   basePath,
				Query:  map[string]string{"limit": "abc"},
			},
			ExpectedStatus: http.StatusOK, // Should use default
		},
	}
}
