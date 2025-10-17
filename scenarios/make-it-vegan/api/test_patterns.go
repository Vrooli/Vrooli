// +build !test

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
	Setup          func(t *testing.T) interface{}
	Execute        func(t *testing.T, setupData interface{}) *HTTPTestRequest
	Validate       func(t *testing.T, w *httptest.ResponseRecorder, setupData interface{})
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
			req := pattern.Execute(t, setupData)
			w, err := makeHTTPRequest(*req)
			if err != nil {
				t.Fatalf("Failed to create HTTP request: %v", err)
			}

			// Call handler
			suite.Handler(w, w.Result().Request)

			// Validate status
			if w.Code != pattern.ExpectedStatus {
				t.Errorf("Expected status %d, got %d. Body: %s", pattern.ExpectedStatus, w.Code, w.Body.String())
			}

			// Custom validation
			if pattern.Validate != nil {
				pattern.Validate(t, w, setupData)
			}
		})
	}
}

// Common error patterns that can be reused across handlers

// invalidJSONPattern tests handlers with malformed JSON
func invalidJSONPattern(method, urlPath string) ErrorTestPattern {
	return ErrorTestPattern{
		Name:           "InvalidJSON",
		Description:    "Test handler with malformed JSON body",
		ExpectedStatus: http.StatusBadRequest,
		Setup:          nil,
		Execute: func(t *testing.T, setupData interface{}) *HTTPTestRequest {
			return &HTTPTestRequest{
				Method: method,
				Path:   urlPath,
				Body:   "invalid-json",
			}
		},
		Validate: nil,
	}
}

// emptyBodyPattern tests handlers with empty request body
func emptyBodyPattern(method, urlPath string) ErrorTestPattern {
	return ErrorTestPattern{
		Name:           "EmptyBody",
		Description:    "Test handler with empty request body",
		ExpectedStatus: http.StatusBadRequest,
		Setup:          nil,
		Execute: func(t *testing.T, setupData interface{}) *HTTPTestRequest {
			return &HTTPTestRequest{
				Method: method,
				Path:   urlPath,
				Body:   nil,
			}
		},
		Validate: nil,
	}
}

// missingFieldPattern tests handlers with missing required fields
func missingFieldPattern(method, urlPath string, body map[string]interface{}) ErrorTestPattern {
	return ErrorTestPattern{
		Name:           "MissingRequiredField",
		Description:    "Test handler with missing required field",
		ExpectedStatus: http.StatusBadRequest,
		Setup:          nil,
		Execute: func(t *testing.T, setupData interface{}) *HTTPTestRequest {
			return &HTTPTestRequest{
				Method: method,
				Path:   urlPath,
				Body:   body,
			}
		},
		Validate: nil,
	}
}

// emptyStringPattern tests handlers with empty string values
func emptyStringPattern(method, urlPath string, body map[string]interface{}) ErrorTestPattern {
	return ErrorTestPattern{
		Name:           "EmptyStringValue",
		Description:    "Test handler with empty string value",
		ExpectedStatus: http.StatusOK, // Most handlers should handle empty strings gracefully
		Setup:          nil,
		Execute: func(t *testing.T, setupData interface{}) *HTTPTestRequest {
			return &HTTPTestRequest{
				Method: method,
				Path:   urlPath,
				Body:   body,
			}
		},
		Validate: nil,
	}
}

// TestScenarioBuilder helper methods for common patterns

// AddCheckIngredientsScenarios adds test scenarios for ingredient checking
func (b *TestScenarioBuilder) AddCheckIngredientsScenarios() *TestScenarioBuilder {
	// Valid vegan ingredients
	b.AddScenario(HTTPTestRequest{
		Method: "POST",
		Path:   "/api/check",
		Body:   CheckRequest{Ingredients: "flour, sugar, salt"},
	})

	// Non-vegan ingredients
	b.AddScenario(HTTPTestRequest{
		Method: "POST",
		Path:   "/api/check",
		Body:   CheckRequest{Ingredients: "milk, eggs, butter"},
	})

	// Mixed ingredients
	b.AddScenario(HTTPTestRequest{
		Method: "POST",
		Path:   "/api/check",
		Body:   CheckRequest{Ingredients: "flour, milk, sugar"},
	})

	// Empty ingredients
	b.AddScenario(HTTPTestRequest{
		Method: "POST",
		Path:   "/api/check",
		Body:   CheckRequest{Ingredients: ""},
	})

	return b
}

// AddSubstituteScenarios adds test scenarios for substitute finding
func (b *TestScenarioBuilder) AddSubstituteScenarios() *TestScenarioBuilder {
	// Common non-vegan ingredient
	b.AddScenario(HTTPTestRequest{
		Method: "POST",
		Path:   "/api/substitute",
		Body:   SubstituteRequest{Ingredient: "milk", Context: "baking"},
	})

	// Another common ingredient
	b.AddScenario(HTTPTestRequest{
		Method: "POST",
		Path:   "/api/substitute",
		Body:   SubstituteRequest{Ingredient: "eggs", Context: "baking"},
	})

	// Without context
	b.AddScenario(HTTPTestRequest{
		Method: "POST",
		Path:   "/api/substitute",
		Body:   SubstituteRequest{Ingredient: "butter", Context: ""},
	})

	// Unknown ingredient
	b.AddScenario(HTTPTestRequest{
		Method: "POST",
		Path:   "/api/substitute",
		Body:   SubstituteRequest{Ingredient: "unknown-ingredient", Context: "cooking"},
	})

	return b
}

// AddVeganizeScenarios adds test scenarios for recipe veganization
func (b *TestScenarioBuilder) AddVeganizeScenarios() *TestScenarioBuilder {
	// Recipe with non-vegan ingredients
	b.AddScenario(HTTPTestRequest{
		Method: "POST",
		Path:   "/api/veganize",
		Body:   VeganizeRequest{Recipe: "Mix milk, eggs, and butter to make pancakes"},
	})

	// Already vegan recipe
	b.AddScenario(HTTPTestRequest{
		Method: "POST",
		Path:   "/api/veganize",
		Body:   VeganizeRequest{Recipe: "Mix flour, water, and oil to make bread"},
	})

	// Empty recipe
	b.AddScenario(HTTPTestRequest{
		Method: "POST",
		Path:   "/api/veganize",
		Body:   VeganizeRequest{Recipe: ""},
	})

	return b
}
