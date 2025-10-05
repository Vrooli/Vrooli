// +build testing

package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
)

// TestScenarioBuilder provides a fluent interface for building test scenarios
type TestScenarioBuilder struct {
	scenarios []ErrorTestScenario
}

// ErrorTestScenario represents a single error test case
type ErrorTestScenario struct {
	Name           string
	Method         string
	Path           string
	Body           interface{}
	URLVars        map[string]string
	QueryParams    map[string]string
	Headers        map[string]string
	ExpectedStatus int
	Description    string
}

// NewTestScenarioBuilder creates a new test scenario builder
func NewTestScenarioBuilder() *TestScenarioBuilder {
	return &TestScenarioBuilder{
		scenarios: []ErrorTestScenario{},
	}
}

// AddInvalidJSON adds a test for invalid JSON input
func (b *TestScenarioBuilder) AddInvalidJSON(path string) *TestScenarioBuilder {
	b.scenarios = append(b.scenarios, ErrorTestScenario{
		Name:           "InvalidJSON",
		Method:         "POST",
		Path:           path,
		Body:           `{"invalid": "json"`, // Malformed JSON
		ExpectedStatus: http.StatusBadRequest,
		Description:    "Handler should reject malformed JSON",
	})
	return b
}

// AddMissingRequiredField adds tests for missing required fields
func (b *TestScenarioBuilder) AddMissingRequiredField(path string, field string) *TestScenarioBuilder {
	var body interface{}

	// Create body with missing field based on endpoint
	switch {
	case path == "/api/v1/shopping/research":
		body = ShoppingResearchRequest{
			ProfileID: "test-user-1",
			Query:     "", // Missing query
		}
	case path == "/api/v1/shopping/pattern-analysis":
		// Pattern analysis currently doesn't validate profile_id
		body = map[string]interface{}{}
	default:
		body = map[string]interface{}{}
	}

	b.scenarios = append(b.scenarios, ErrorTestScenario{
		Name:           fmt.Sprintf("Missing_%s", field),
		Method:         "POST",
		Path:           path,
		Body:           body,
		ExpectedStatus: http.StatusBadRequest,
		Description:    fmt.Sprintf("Handler should reject request missing %s", field),
	})
	return b
}

// AddEmptyBody adds a test for empty request body
func (b *TestScenarioBuilder) AddEmptyBody(path string) *TestScenarioBuilder {
	b.scenarios = append(b.scenarios, ErrorTestScenario{
		Name:           "EmptyBody",
		Method:         "POST",
		Path:           path,
		Body:           "",
		ExpectedStatus: http.StatusBadRequest,
		Description:    "Handler should reject empty request body",
	})
	return b
}

// AddInvalidPathParameter adds a test for invalid path parameters
func (b *TestScenarioBuilder) AddInvalidPathParameter(path string, paramName string, invalidValue string) *TestScenarioBuilder {
	b.scenarios = append(b.scenarios, ErrorTestScenario{
		Name:           fmt.Sprintf("Invalid_%s", paramName),
		Method:         "GET",
		Path:           path,
		URLVars:        map[string]string{paramName: invalidValue},
		ExpectedStatus: http.StatusBadRequest,
		Description:    fmt.Sprintf("Handler should reject invalid %s", paramName),
	})
	return b
}

// AddUnauthorizedAccess adds a test for unauthorized access
func (b *TestScenarioBuilder) AddUnauthorizedAccess(path string) *TestScenarioBuilder {
	b.scenarios = append(b.scenarios, ErrorTestScenario{
		Name:           "UnauthorizedAccess",
		Method:         "POST",
		Path:           path,
		Body:           map[string]interface{}{"query": "test"},
		Headers:        map[string]string{}, // No auth header
		ExpectedStatus: http.StatusOK,       // Our middleware allows anonymous
		Description:    "Handler allows anonymous access with fallback",
	})
	return b
}

// AddInvalidBudget adds a test for invalid budget values
func (b *TestScenarioBuilder) AddInvalidBudget(path string) *TestScenarioBuilder {
	b.scenarios = append(b.scenarios, ErrorTestScenario{
		Name:   "InvalidBudget",
		Method: "POST",
		Path:   path,
		Body: ShoppingResearchRequest{
			ProfileID: "test-user-1",
			Query:     "laptop",
			BudgetMax: -100.00, // Negative budget
		},
		ExpectedStatus: http.StatusOK, // Currently accepted, could be validated
		Description:    "Handler behavior with negative budget",
	})
	return b
}

// Build returns all configured test scenarios
func (b *TestScenarioBuilder) Build() []ErrorTestScenario {
	return b.scenarios
}

// HandlerTestSuite provides a comprehensive test framework for HTTP handlers
type HandlerTestSuite struct {
	Name           string
	Server         *Server
	SuccessTests   []SuccessTestCase
	ErrorTests     []ErrorTestScenario
	PerformanceMax int // milliseconds
}

// SuccessTestCase represents a successful operation test
type SuccessTestCase struct {
	Name           string
	Request        HTTPTestRequest
	ExpectedStatus int
	Validator      func(t *testing.T, response map[string]interface{})
}

// RunSuccessTests executes all success test cases
func (suite *HandlerTestSuite) RunSuccessTests(t *testing.T) {
	for _, test := range suite.SuccessTests {
		t.Run(fmt.Sprintf("%s_Success_%s", suite.Name, test.Name), func(t *testing.T) {
			w := makeHTTPRequest(suite.Server, test.Request)

			if w.Code != test.ExpectedStatus {
				t.Errorf("Expected status %d, got %d. Response: %s",
					test.ExpectedStatus, w.Code, w.Body.String())
				return
			}

			if test.Validator != nil {
				var response map[string]interface{}
				if err := json.Unmarshal(w.Body.Bytes(), &response); err == nil {
					test.Validator(t, response)
				}
			}
		})
	}
}

// RunErrorTests executes all error test cases
func (suite *HandlerTestSuite) RunErrorTests(t *testing.T) {
	for _, test := range suite.ErrorTests {
		t.Run(fmt.Sprintf("%s_Error_%s", suite.Name, test.Name), func(t *testing.T) {
			req := HTTPTestRequest{
				Method:      test.Method,
				Path:        test.Path,
				Body:        test.Body,
				URLVars:     test.URLVars,
				QueryParams: test.QueryParams,
				Headers:     test.Headers,
			}

			w := makeHTTPRequest(suite.Server, req)

			if w.Code != test.ExpectedStatus {
				t.Logf("Test: %s - %s", test.Name, test.Description)
				t.Errorf("Expected status %d, got %d. Response: %s",
					test.ExpectedStatus, w.Code, w.Body.String())
			}
		})
	}
}

// RunAllTests executes both success and error tests
func (suite *HandlerTestSuite) RunAllTests(t *testing.T) {
	t.Run(suite.Name+"_Success", func(t *testing.T) {
		suite.RunSuccessTests(t)
	})

	t.Run(suite.Name+"_Errors", func(t *testing.T) {
		suite.RunErrorTests(t)
	})
}

// ErrorTestPattern defines systematic error testing patterns
type ErrorTestPattern struct {
	testScenarios []ErrorTestScenario
}

// NewErrorTestPattern creates a new error test pattern
func NewErrorTestPattern() *ErrorTestPattern {
	return &ErrorTestPattern{
		testScenarios: []ErrorTestScenario{},
	}
}

// WithInvalidJSON adds invalid JSON test
func (p *ErrorTestPattern) WithInvalidJSON(path string) *ErrorTestPattern {
	p.testScenarios = append(p.testScenarios, ErrorTestScenario{
		Name:           "InvalidJSON",
		Method:         "POST",
		Path:           path,
		Body:           `{"broken": json}`,
		ExpectedStatus: http.StatusBadRequest,
	})
	return p
}

// WithMissingFields adds missing required fields test
func (p *ErrorTestPattern) WithMissingFields(path string, body interface{}) *ErrorTestPattern {
	p.testScenarios = append(p.testScenarios, ErrorTestScenario{
		Name:           "MissingFields",
		Method:         "POST",
		Path:           path,
		Body:           body,
		ExpectedStatus: http.StatusBadRequest,
	})
	return p
}

// GetScenarios returns all configured test scenarios
func (p *ErrorTestPattern) GetScenarios() []ErrorTestScenario {
	return p.testScenarios
}
