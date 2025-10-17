package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

// ErrorTestPattern defines a systematic approach to testing error conditions
type ErrorTestPattern struct {
	Name           string
	Description    string
	ExpectedStatus int
	Request        HTTPTestRequest
	Validate       func(t *testing.T, w *http.ResponseWriter)
}

// TestScenarioBuilder provides a fluent interface for building test scenarios
type TestScenarioBuilder struct {
	patterns []ErrorTestPattern
}

// NewTestScenarioBuilder creates a new test scenario builder
func NewTestScenarioBuilder() *TestScenarioBuilder {
	return &TestScenarioBuilder{
		patterns: []ErrorTestPattern{},
	}
}

// AddInvalidJSON adds a test for invalid JSON input
func (b *TestScenarioBuilder) AddInvalidJSON(path string) *TestScenarioBuilder {
	b.patterns = append(b.patterns, ErrorTestPattern{
		Name:           "InvalidJSON",
		Description:    "Test handler with malformed JSON",
		ExpectedStatus: http.StatusBadRequest,
		Request: HTTPTestRequest{
			Method:  "POST",
			Path:    path,
			Body:    "invalid-json",
			Headers: map[string]string{"Content-Type": "application/json"},
		},
	})
	return b
}

// AddMissingCoupleID adds a test for missing couple_id
func (b *TestScenarioBuilder) AddMissingCoupleID(path string) *TestScenarioBuilder {
	b.patterns = append(b.patterns, ErrorTestPattern{
		Name:           "MissingCoupleID",
		Description:    "Test handler with missing couple_id",
		ExpectedStatus: http.StatusBadRequest,
		Request: HTTPTestRequest{
			Method: "POST",
			Path:   path,
			Body: map[string]interface{}{
				"date_type": "romantic",
			},
		},
	})
	return b
}

// AddEmptyBody adds a test for empty request body
func (b *TestScenarioBuilder) AddEmptyBody(path string) *TestScenarioBuilder {
	b.patterns = append(b.patterns, ErrorTestPattern{
		Name:           "EmptyBody",
		Description:    "Test handler with empty request body",
		ExpectedStatus: http.StatusBadRequest,
		Request: HTTPTestRequest{
			Method: "POST",
			Path:   path,
			Body:   map[string]interface{}{},
		},
	})
	return b
}

// AddInvalidDateType adds a test for invalid date type
func (b *TestScenarioBuilder) AddInvalidDateType(path string) *TestScenarioBuilder {
	b.patterns = append(b.patterns, ErrorTestPattern{
		Name:           "InvalidDateType",
		Description:    "Test handler with invalid date type",
		ExpectedStatus: http.StatusOK, // Should still return suggestions
		Request: HTTPTestRequest{
			Method: "POST",
			Path:   path,
			Body: map[string]interface{}{
				"couple_id": "test-couple",
				"date_type": "invalid-type",
			},
		},
	})
	return b
}

// AddNegativeBudget adds a test for negative budget
func (b *TestScenarioBuilder) AddNegativeBudget(path string) *TestScenarioBuilder {
	b.patterns = append(b.patterns, ErrorTestPattern{
		Name:           "NegativeBudget",
		Description:    "Test handler with negative budget",
		ExpectedStatus: http.StatusOK, // Should handle gracefully
		Request: HTTPTestRequest{
			Method: "POST",
			Path:   path,
			Body: map[string]interface{}{
				"couple_id":  "test-couple",
				"budget_max": -100,
			},
		},
	})
	return b
}

// AddInvalidDateFormat adds a test for invalid date format
func (b *TestScenarioBuilder) AddInvalidDateFormat(path string) *TestScenarioBuilder {
	b.patterns = append(b.patterns, ErrorTestPattern{
		Name:           "InvalidDateFormat",
		Description:    "Test handler with invalid date format",
		ExpectedStatus: http.StatusOK, // Should handle gracefully
		Request: HTTPTestRequest{
			Method: "POST",
			Path:   path,
			Body: map[string]interface{}{
				"couple_id":      "test-couple",
				"preferred_date": "invalid-date",
			},
		},
	})
	return b
}

// Build returns all configured error patterns
func (b *TestScenarioBuilder) Build() []ErrorTestPattern {
	return b.patterns
}

// HandlerTestSuite provides a comprehensive test framework for HTTP handlers
type HandlerTestSuite struct {
	HandlerName string
	Handler     http.HandlerFunc
	BaseURL     string
}

// NewHandlerTestSuite creates a new handler test suite
func NewHandlerTestSuite(name string, handler http.HandlerFunc, baseURL string) *HandlerTestSuite {
	return &HandlerTestSuite{
		HandlerName: name,
		Handler:     handler,
		BaseURL:     baseURL,
	}
}

// RunErrorTests executes a suite of error condition tests
func (suite *HandlerTestSuite) RunErrorTests(t *testing.T, patterns []ErrorTestPattern) {
	for _, pattern := range patterns {
		t.Run(fmt.Sprintf("%s_%s", suite.HandlerName, pattern.Name), func(t *testing.T) {
			cleanup := setupTestLogger()
			defer cleanup()

			// Execute request
			w, err := makeHTTPRequest(pattern.Request, suite.Handler)
			if err != nil {
				t.Fatalf("Failed to make HTTP request: %v", err)
			}

			// Validate status code
			if w.Code != pattern.ExpectedStatus {
				t.Errorf("Expected status %d, got %d. Body: %s",
					pattern.ExpectedStatus, w.Code, w.Body.String())
			}

			// Custom validation if provided
			if pattern.Validate != nil {
				var respWriter http.ResponseWriter = w
				pattern.Validate(t, &respWriter)
			}
		})
	}
}

// RunSuccessTest runs a single success test case
func (suite *HandlerTestSuite) RunSuccessTest(t *testing.T, name string, req HTTPTestRequest, validate func(t *testing.T, w *http.ResponseWriter)) {
	t.Run(fmt.Sprintf("%s_%s", suite.HandlerName, name), func(t *testing.T) {
		cleanup := setupTestLogger()
		defer cleanup()

		w, err := makeHTTPRequest(req, suite.Handler)
		if err != nil {
			t.Fatalf("Failed to make HTTP request: %v", err)
		}

		if validate != nil {
			var respWriter http.ResponseWriter = w
			validate(t, &respWriter)
		}
	})
}

// Common validation functions

// ValidateJSONResponse checks for valid JSON response with expected status
func ValidateJSONResponse(expectedStatus int, requiredFields []string) func(t *testing.T, w *http.ResponseWriter) {
	return func(t *testing.T, w *http.ResponseWriter) {
		recorder := (*w).(*httptest.ResponseRecorder)
		response := assertJSONResponse(t, recorder, expectedStatus)

		for _, field := range requiredFields {
			if _, ok := response[field]; !ok {
				t.Errorf("Expected field '%s' in response, but not found", field)
			}
		}
	}
}

// ValidateSuggestionsResponse validates date suggestions response
func ValidateSuggestionsResponse(t *testing.T, w *http.ResponseWriter) {
	recorder := (*w).(*httptest.ResponseRecorder)
	response := assertJSONResponse(t, recorder, http.StatusOK)

	suggestions, ok := response["suggestions"]
	if !ok {
		t.Fatal("Expected 'suggestions' field in response")
	}

	suggestionsList, ok := suggestions.([]interface{})
	if !ok {
		t.Fatal("Expected 'suggestions' to be an array")
	}

	if len(suggestionsList) == 0 {
		t.Error("Expected at least one suggestion")
	}

	// Validate first suggestion has required fields
	if len(suggestionsList) > 0 {
		suggestion, ok := suggestionsList[0].(map[string]interface{})
		if !ok {
			t.Fatal("Expected suggestion to be an object")
		}

		requiredFields := []string{"title", "description", "activities", "estimated_cost", "estimated_duration", "confidence_score"}
		for _, field := range requiredFields {
			if _, ok := suggestion[field]; !ok {
				t.Errorf("Expected field '%s' in suggestion, but not found", field)
			}
		}
	}
}

// ValidateDatePlanResponse validates date plan response
func ValidateDatePlanResponse(t *testing.T, w *http.ResponseWriter) {
	recorder := (*w).(*httptest.ResponseRecorder)
	response := assertJSONResponse(t, recorder, http.StatusCreated)

	datePlan, ok := response["date_plan"]
	if !ok {
		t.Fatal("Expected 'date_plan' field in response")
	}

	plan, ok := datePlan.(map[string]interface{})
	if !ok {
		t.Fatal("Expected 'date_plan' to be an object")
	}

	requiredFields := []string{"id", "couple_id", "title", "description", "planned_date", "status"}
	for _, field := range requiredFields {
		if _, ok := plan[field]; !ok {
			t.Errorf("Expected field '%s' in date_plan, but not found", field)
		}
	}
}
