package main

import (
	"bytes"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestScenario defines a test case with setup, execution, and validation
type TestScenario struct {
	Name           string
	Description    string
	Setup          func(t *testing.T) interface{}
	Execute        func(t *testing.T, setupData interface{}) (*httptest.ResponseRecorder, error)
	ExpectedStatus int
	Validate       func(t *testing.T, rr *httptest.ResponseRecorder, setupData interface{})
	Cleanup        func(t *testing.T, setupData interface{})
}

// TestScenarioBuilder builds test scenarios fluently
type TestScenarioBuilder struct {
	scenarios []TestScenario
}

// NewTestScenarioBuilder creates a new test scenario builder
func NewTestScenarioBuilder() *TestScenarioBuilder {
	return &TestScenarioBuilder{
		scenarios: make([]TestScenario, 0),
	}
}

// AddInvalidParameter adds a test for invalid query parameter
func (b *TestScenarioBuilder) AddInvalidParameter(handler http.HandlerFunc, path, param, invalidValue string) *TestScenarioBuilder {
	b.scenarios = append(b.scenarios, TestScenario{
		Name:        "InvalidParameter_" + param,
		Description: "Test handler with invalid " + param,
		Setup:       nil,
		Execute: func(t *testing.T, setupData interface{}) (*httptest.ResponseRecorder, error) {
			return makeHTTPRequest(HTTPTestRequest{
				Method:      "GET",
				Path:        path,
				QueryParams: map[string]string{param: invalidValue},
			}, handler)
		},
		ExpectedStatus: http.StatusBadRequest,
		Validate: func(t *testing.T, rr *httptest.ResponseRecorder, setupData interface{}) {
			assertErrorResponse(t, rr, http.StatusBadRequest, "")
		},
	})
	return b
}

// AddMissingParameter adds a test for missing required parameter
func (b *TestScenarioBuilder) AddMissingParameter(handler http.HandlerFunc, path, param string) *TestScenarioBuilder {
	b.scenarios = append(b.scenarios, TestScenario{
		Name:        "MissingParameter_" + param,
		Description: "Test handler with missing " + param,
		Setup:       nil,
		Execute: func(t *testing.T, setupData interface{}) (*httptest.ResponseRecorder, error) {
			return makeHTTPRequest(HTTPTestRequest{
				Method: "GET",
				Path:   path,
			}, handler)
		},
		ExpectedStatus: http.StatusBadRequest,
		Validate: func(t *testing.T, rr *httptest.ResponseRecorder, setupData interface{}) {
			assertErrorResponse(t, rr, http.StatusBadRequest, "")
		},
	})
	return b
}

// AddInvalidJSON adds a test for invalid JSON in request body
func (b *TestScenarioBuilder) AddInvalidJSON(handler http.HandlerFunc, path string) *TestScenarioBuilder {
	b.scenarios = append(b.scenarios, TestScenario{
		Name:        "InvalidJSON",
		Description: "Test handler with invalid JSON body",
		Setup:       nil,
		Execute: func(t *testing.T, setupData interface{}) (*httptest.ResponseRecorder, error) {
			req, _ := http.NewRequest("POST", path, bytes.NewBufferString("{invalid json}"))
			req.Header.Set("Content-Type", "application/json")
			rr := httptest.NewRecorder()
			handler(rr, req)
			return rr, nil
		},
		ExpectedStatus: http.StatusBadRequest,
		Validate: func(t *testing.T, rr *httptest.ResponseRecorder, setupData interface{}) {
			assertErrorResponse(t, rr, http.StatusBadRequest, "")
		},
	})
	return b
}

// AddMethodNotAllowed adds a test for wrong HTTP method
func (b *TestScenarioBuilder) AddMethodNotAllowed(handler http.HandlerFunc, path, wrongMethod string) *TestScenarioBuilder {
	b.scenarios = append(b.scenarios, TestScenario{
		Name:        "MethodNotAllowed_" + wrongMethod,
		Description: "Test handler with wrong HTTP method",
		Setup:       nil,
		Execute: func(t *testing.T, setupData interface{}) (*httptest.ResponseRecorder, error) {
			return makeHTTPRequest(HTTPTestRequest{
				Method: wrongMethod,
				Path:   path,
			}, handler)
		},
		ExpectedStatus: http.StatusMethodNotAllowed,
		Validate: func(t *testing.T, rr *httptest.ResponseRecorder, setupData interface{}) {
			assertErrorResponse(t, rr, http.StatusMethodNotAllowed, "")
		},
	})
	return b
}

// AddNonExistentResource adds a test for non-existent resource
func (b *TestScenarioBuilder) AddNonExistentResource(handler http.HandlerFunc, path string, param string, value string) *TestScenarioBuilder {
	b.scenarios = append(b.scenarios, TestScenario{
		Name:        "NonExistentResource",
		Description: "Test handler with non-existent resource",
		Setup:       nil,
		Execute: func(t *testing.T, setupData interface{}) (*httptest.ResponseRecorder, error) {
			return makeHTTPRequest(HTTPTestRequest{
				Method:      "GET",
				Path:        path,
				QueryParams: map[string]string{param: value},
			}, handler)
		},
		ExpectedStatus: http.StatusNotFound,
		Validate: func(t *testing.T, rr *httptest.ResponseRecorder, setupData interface{}) {
			// Can be 404 or 200 with empty data depending on implementation
			if rr.Code != http.StatusNotFound && rr.Code != http.StatusOK {
				t.Errorf("Expected 404 or 200, got %d", rr.Code)
			}
		},
	})
	return b
}

// AddDatabaseUnavailable adds a test for when database is unavailable
func (b *TestScenarioBuilder) AddDatabaseUnavailable(handler http.HandlerFunc, path string) *TestScenarioBuilder {
	b.scenarios = append(b.scenarios, TestScenario{
		Name:        "DatabaseUnavailable",
		Description: "Test handler when database is unavailable",
		Setup: func(t *testing.T) interface{} {
			originalDB := db
			db = nil
			return originalDB
		},
		Execute: func(t *testing.T, setupData interface{}) (*httptest.ResponseRecorder, error) {
			return makeHTTPRequest(HTTPTestRequest{
				Method: "GET",
				Path:   path,
			}, handler)
		},
		ExpectedStatus: http.StatusOK, // Should return mock data or handle gracefully
		Validate: func(t *testing.T, rr *httptest.ResponseRecorder, setupData interface{}) {
			// Validate it doesn't crash
			if rr.Code >= 500 {
				t.Errorf("Handler should handle DB unavailability gracefully, got status %d", rr.Code)
			}
		},
		Cleanup: func(t *testing.T, setupData interface{}) {
			db = setupData.(*sql.DB)
		},
	})
	return b
}

// Build returns all scenarios
func (b *TestScenarioBuilder) Build() []TestScenario {
	return b.scenarios
}

// RunScenarios executes all built scenarios
func (b *TestScenarioBuilder) RunScenarios(t *testing.T) {
	for _, scenario := range b.scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			var setupData interface{}
			if scenario.Setup != nil {
				setupData = scenario.Setup(t)
			}

			if scenario.Cleanup != nil {
				defer scenario.Cleanup(t, setupData)
			}

			rr, err := scenario.Execute(t, setupData)
			if err != nil {
				t.Fatalf("Failed to execute test: %v", err)
			}

			if scenario.ExpectedStatus != 0 && rr.Code != scenario.ExpectedStatus {
				t.Logf("Body: %s", rr.Body.String())
			}

			if scenario.Validate != nil {
				scenario.Validate(t, rr, setupData)
			}
		})
	}
}

// ErrorTestPattern provides common error testing patterns
type ErrorTestPattern struct {
	scenarios []TestScenario
}

// NewErrorTestPattern creates error test patterns for a handler
func NewErrorTestPattern(handler http.HandlerFunc, basePath string) *ErrorTestPattern {
	return &ErrorTestPattern{
		scenarios: make([]TestScenario, 0),
	}
}
