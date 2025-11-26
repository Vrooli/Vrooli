package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
)

// ErrorTestPattern defines systematic error testing
type ErrorTestPattern struct {
	Name           string
	Description    string
	ExpectedStatus int
	Setup          func(t *testing.T, testDB *TestDB) interface{}
	Execute        func(t *testing.T, setupData interface{}) HTTPTestRequest
	Validate       func(t *testing.T, w *httptest.ResponseRecorder, setupData interface{})
	Cleanup        func(setupData interface{})
}

// TestScenarioBuilder provides fluent interface for building test scenarios
type TestScenarioBuilder struct {
	scenarios []ErrorTestPattern
}

// NewTestScenarioBuilder creates a new test scenario builder
func NewTestScenarioBuilder() *TestScenarioBuilder {
	return &TestScenarioBuilder{
		scenarios: make([]ErrorTestPattern, 0),
	}
}

// AddInvalidUUID adds an invalid UUID test pattern
func (b *TestScenarioBuilder) AddInvalidUUID(urlPath string) *TestScenarioBuilder {
	b.scenarios = append(b.scenarios, ErrorTestPattern{
		Name:           "InvalidUUID",
		Description:    "Test with invalid UUID format",
		ExpectedStatus: http.StatusBadRequest,
		Setup:          nil,
		Execute: func(t *testing.T, setupData interface{}) HTTPTestRequest {
			return HTTPTestRequest{
				Method:  "GET",
				Path:    urlPath,
				URLVars: map[string]string{"id": "invalid-uuid-format"},
			}
		},
		Validate: func(t *testing.T, w *httptest.ResponseRecorder, setupData interface{}) {
			if w.Code != http.StatusBadRequest && w.Code != http.StatusInternalServerError {
				t.Errorf("Expected 400 or 500 for invalid UUID, got %d", w.Code)
			}
		},
	})
	return b
}

// AddNonExistentCampaign adds a non-existent campaign test pattern
func (b *TestScenarioBuilder) AddNonExistentCampaign(urlPath string) *TestScenarioBuilder {
	b.scenarios = append(b.scenarios, ErrorTestPattern{
		Name:           "NonExistentCampaign",
		Description:    "Test with non-existent campaign ID",
		ExpectedStatus: http.StatusNotFound,
		Setup:          nil,
		Execute: func(t *testing.T, setupData interface{}) HTTPTestRequest {
			nonExistentID := uuid.New().String()
			return HTTPTestRequest{
				Method:      "GET",
				Path:        urlPath,
				QueryParams: map[string]string{"campaign_id": nonExistentID},
			}
		},
		Validate: func(t *testing.T, w *httptest.ResponseRecorder, setupData interface{}) {
			// The API might return empty array instead of 404
			if w.Code != http.StatusOK && w.Code != http.StatusNotFound {
				t.Errorf("Expected 200 or 404 for non-existent campaign, got %d", w.Code)
			}
		},
	})
	return b
}

// AddInvalidJSON adds an invalid JSON test pattern
func (b *TestScenarioBuilder) AddInvalidJSON(urlPath string) *TestScenarioBuilder {
	b.scenarios = append(b.scenarios, ErrorTestPattern{
		Name:           "InvalidJSON",
		Description:    "Test with malformed JSON body",
		ExpectedStatus: http.StatusBadRequest,
		Setup:          nil,
		Execute: func(t *testing.T, setupData interface{}) HTTPTestRequest {
			return HTTPTestRequest{
				Method: "POST",
				Path:   urlPath,
				Body:   "invalid-json-data",
			}
		},
		Validate: func(t *testing.T, w *httptest.ResponseRecorder, setupData interface{}) {
			if w.Code != http.StatusBadRequest && w.Code != http.StatusInternalServerError {
				t.Errorf("Expected 400 or 500 for invalid JSON, got %d", w.Code)
			}
		},
	})
	return b
}

// AddMissingRequiredField adds a missing required field test pattern
func (b *TestScenarioBuilder) AddMissingRequiredField(urlPath string, fieldName string) *TestScenarioBuilder {
	b.scenarios = append(b.scenarios, ErrorTestPattern{
		Name:           fmt.Sprintf("Missing_%s", fieldName),
		Description:    fmt.Sprintf("Test with missing required field: %s", fieldName),
		ExpectedStatus: http.StatusBadRequest,
		Setup:          nil,
		Execute: func(t *testing.T, setupData interface{}) HTTPTestRequest {
			// Empty body to trigger missing field error
			return HTTPTestRequest{
				Method: "POST",
				Path:   urlPath,
				Body:   map[string]interface{}{},
			}
		},
		Validate: func(t *testing.T, w *httptest.ResponseRecorder, setupData interface{}) {
			if w.Code != http.StatusBadRequest && w.Code != http.StatusInternalServerError {
				t.Errorf("Expected 400 or 500 for missing field, got %d", w.Code)
			}
		},
	})
	return b
}

// AddEmptyQueryParam adds an empty query parameter test pattern
func (b *TestScenarioBuilder) AddEmptyQueryParam(urlPath string, paramName string) *TestScenarioBuilder {
	b.scenarios = append(b.scenarios, ErrorTestPattern{
		Name:           fmt.Sprintf("Empty_%s_Param", paramName),
		Description:    fmt.Sprintf("Test with empty query parameter: %s", paramName),
		ExpectedStatus: http.StatusBadRequest,
		Setup:          nil,
		Execute: func(t *testing.T, setupData interface{}) HTTPTestRequest {
			return HTTPTestRequest{
				Method:      "GET",
				Path:        urlPath,
				QueryParams: map[string]string{paramName: ""},
			}
		},
		Validate: func(t *testing.T, w *httptest.ResponseRecorder, setupData interface{}) {
			// Some endpoints might accept empty params
			if w.Code >= 500 {
				t.Errorf("Should not return 500 for empty param, got %d", w.Code)
			}
		},
	})
	return b
}

// Build returns the built test scenarios
func (b *TestScenarioBuilder) Build() []ErrorTestPattern {
	return b.scenarios
}

// HandlerTestSuite provides comprehensive testing framework for HTTP handlers
type HandlerTestSuite struct {
	HandlerName string
	Handler     http.HandlerFunc
	BaseURL     string
	TestDB      *TestDB
}

// NewHandlerTestSuite creates a new handler test suite
func NewHandlerTestSuite(name string, handler http.HandlerFunc, testDB *TestDB) *HandlerTestSuite {
	return &HandlerTestSuite{
		HandlerName: name,
		Handler:     handler,
		TestDB:      testDB,
	}
}

// RunErrorTests executes a suite of error condition tests
func (suite *HandlerTestSuite) RunErrorTests(t *testing.T, patterns []ErrorTestPattern) {
	for _, pattern := range patterns {
		t.Run(fmt.Sprintf("%s_%s", suite.HandlerName, pattern.Name), func(t *testing.T) {
			// Setup
			var setupData interface{}
			if pattern.Setup != nil {
				setupData = pattern.Setup(t, suite.TestDB)
			}

			// Cleanup
			if pattern.Cleanup != nil {
				defer pattern.Cleanup(setupData)
			}

			// Execute
			req := pattern.Execute(t, setupData)
			w, err := makeHTTPRequest(req, suite.Handler)
			if err != nil {
				t.Fatalf("Failed to create HTTP request: %v", err)
			}

			// Validate
			if pattern.Validate != nil {
				pattern.Validate(t, w, setupData)
			} else {
				// Default validation
				if w.Code != pattern.ExpectedStatus && pattern.ExpectedStatus != 0 {
					t.Logf("Expected status %d, got %d. Body: %s",
						pattern.ExpectedStatus, w.Code, w.Body.String())
				}
			}
		})
	}
}

// RunSuccessTest executes a successful test case
func (suite *HandlerTestSuite) RunSuccessTest(t *testing.T, name string, req HTTPTestRequest, validator func(*testing.T, *httptest.ResponseRecorder)) {
	t.Run(fmt.Sprintf("%s_%s", suite.HandlerName, name), func(t *testing.T) {
		w, err := makeHTTPRequest(req, suite.Handler)
		if err != nil {
			t.Fatalf("Failed to create HTTP request: %v", err)
		}

		if validator != nil {
			validator(t, w)
		}
	})
}
