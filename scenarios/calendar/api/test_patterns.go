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
	Setup          func(t *testing.T) interface{}
	Execute        func(t *testing.T, setupData interface{}) HTTPTestRequest
	Validate       func(t *testing.T, rr *httptest.ResponseRecorder, setupData interface{})
	Cleanup        func(setupData interface{})
}

// HandlerTestSuite provides a comprehensive test framework for HTTP handlers
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
			reqData := pattern.Execute(t, setupData)

			// Create the HTTP request
			httpReq, err := createHTTPRequestFromData(reqData)
			if err != nil {
				t.Fatalf("Failed to create HTTP request: %v", err)
			}

			// Create response recorder
			rr := httptest.NewRecorder()

			// Execute handler
			suite.Handler.ServeHTTP(rr, httpReq)

			// Validate status
			if rr.Code != pattern.ExpectedStatus {
				t.Errorf("Expected status %d, got %d. Body: %s",
					pattern.ExpectedStatus, rr.Code, rr.Body.String())
			}

			// Additional validation
			if pattern.Validate != nil {
				pattern.Validate(t, rr, setupData)
			}
		})
	}
}

// TestScenarioBuilder provides a fluent interface for building test scenarios
type TestScenarioBuilder struct {
	patterns []ErrorTestPattern
}

// NewTestScenarioBuilder creates a new test scenario builder
func NewTestScenarioBuilder() *TestScenarioBuilder {
	return &TestScenarioBuilder{
		patterns: make([]ErrorTestPattern, 0),
	}
}

// AddInvalidUUID adds an invalid UUID test pattern
func (b *TestScenarioBuilder) AddInvalidUUID(urlPath string) *TestScenarioBuilder {
	b.patterns = append(b.patterns, invalidUUIDPattern(urlPath))
	return b
}

// AddNonExistentEvent adds a non-existent event test pattern
func (b *TestScenarioBuilder) AddNonExistentEvent(urlPath string) *TestScenarioBuilder {
	b.patterns = append(b.patterns, nonExistentEventPattern(urlPath))
	return b
}

// AddInvalidJSON adds an invalid JSON test pattern
func (b *TestScenarioBuilder) AddInvalidJSON(urlPath string) *TestScenarioBuilder {
	b.patterns = append(b.patterns, invalidJSONPattern(urlPath))
	return b
}

// AddMissingAuth adds a missing authentication test pattern
func (b *TestScenarioBuilder) AddMissingAuth(urlPath string) *TestScenarioBuilder {
	b.patterns = append(b.patterns, missingAuthPattern(urlPath))
	return b
}

// Build returns the constructed test patterns
func (b *TestScenarioBuilder) Build() []ErrorTestPattern {
	return b.patterns
}

// Common error patterns

// invalidUUIDPattern tests handlers with invalid UUID formats
func invalidUUIDPattern(urlPath string) ErrorTestPattern {
	return ErrorTestPattern{
		Name:           "InvalidUUID",
		Description:    "Test handler with invalid UUID format",
		ExpectedStatus: http.StatusBadRequest,
		Execute: func(t *testing.T, setupData interface{}) HTTPTestRequest {
			return HTTPTestRequest{
				Method: "GET",
				Path:   urlPath,
				URLVars: map[string]string{
					"id": "invalid-uuid-format",
				},
				Headers: map[string]string{
					"Authorization": "Bearer test",
				},
			}
		},
	}
}

// nonExistentEventPattern tests handlers with non-existent event IDs
func nonExistentEventPattern(urlPath string) ErrorTestPattern {
	return ErrorTestPattern{
		Name:           "NonExistentEvent",
		Description:    "Test handler with non-existent event ID",
		ExpectedStatus: http.StatusNotFound,
		Execute: func(t *testing.T, setupData interface{}) HTTPTestRequest {
			nonExistentID := uuid.New()
			return HTTPTestRequest{
				Method: "GET",
				Path:   urlPath,
				URLVars: map[string]string{
					"id": nonExistentID.String(),
				},
				Headers: map[string]string{
					"Authorization": "Bearer test",
				},
			}
		},
	}
}

// invalidJSONPattern tests handlers with malformed JSON
func invalidJSONPattern(urlPath string) ErrorTestPattern {
	return ErrorTestPattern{
		Name:           "InvalidJSON",
		Description:    "Test handler with malformed JSON body",
		ExpectedStatus: http.StatusBadRequest,
		Execute: func(t *testing.T, setupData interface{}) HTTPTestRequest {
			return HTTPTestRequest{
				Method: "POST",
				Path:   urlPath,
				Body:   map[string]interface{}{
					"invalid": "missing required fields",
				},
				Headers: map[string]string{
					"Authorization": "Bearer test",
				},
			}
		},
	}
}

// missingAuthPattern tests handlers without authentication
func missingAuthPattern(urlPath string) ErrorTestPattern {
	return ErrorTestPattern{
		Name:           "MissingAuth",
		Description:    "Test handler without authentication header",
		ExpectedStatus: http.StatusUnauthorized,
		Execute: func(t *testing.T, setupData interface{}) HTTPTestRequest {
			return HTTPTestRequest{
				Method:  "GET",
				Path:    urlPath,
				Headers: map[string]string{
					// No Authorization header
				},
			}
		},
	}
}

// EventTestCase defines a test case for event operations
type EventTestCase struct {
	Name           string
	Event          CreateEventRequest
	ExpectedStatus int
	ExpectedError  string
	Setup          func(t *testing.T)
	Cleanup        func(t *testing.T)
}

// RunEventTests executes a suite of event test cases
func RunEventTests(t *testing.T, handler http.HandlerFunc, testCases []EventTestCase) {
	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			if tc.Setup != nil {
				tc.Setup(t)
			}
			if tc.Cleanup != nil {
				defer tc.Cleanup(t)
			}

			httpReq, err := createHTTPRequestFromData(HTTPTestRequest{
				Method: "POST",
				Path:   "/api/v1/events",
				Body:   tc.Event,
				Headers: map[string]string{
					"Authorization": "Bearer test",
				},
			})
			if err != nil {
				t.Fatalf("Failed to create request: %v", err)
			}

			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, httpReq)

			if rr.Code != tc.ExpectedStatus {
				t.Errorf("Expected status %d, got %d. Body: %s",
					tc.ExpectedStatus, rr.Code, rr.Body.String())
			}

			if tc.ExpectedError != "" {
				assertErrorResponse(t, rr, tc.ExpectedStatus, tc.ExpectedError)
			}
		})
	}
}
