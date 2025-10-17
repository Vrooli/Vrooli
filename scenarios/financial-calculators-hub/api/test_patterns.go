package main

import (
	"net/http"
	"testing"
)

// ErrorTestScenario defines a systematic error test case
type ErrorTestScenario struct {
	Name           string
	Request        HTTPTestRequest
	ExpectedStatus int
	Description    string
}

// TestScenarioBuilder provides a fluent interface for building test scenarios
type TestScenarioBuilder struct {
	scenarios []ErrorTestScenario
}

// NewTestScenarioBuilder creates a new scenario builder
func NewTestScenarioBuilder() *TestScenarioBuilder {
	return &TestScenarioBuilder{
		scenarios: make([]ErrorTestScenario, 0),
	}
}

// AddInvalidJSON adds a test case for invalid JSON input
func (b *TestScenarioBuilder) AddInvalidJSON(path string, method string) *TestScenarioBuilder {
	b.scenarios = append(b.scenarios, ErrorTestScenario{
		Name: "InvalidJSON",
		Request: HTTPTestRequest{
			Method: method,
			Path:   path,
			Body:   `{"invalid": "json"`, // Malformed JSON
		},
		ExpectedStatus: http.StatusBadRequest,
		Description:    "Should reject malformed JSON",
	})
	return b
}

// AddMissingRequiredFields adds a test case for missing required fields
func (b *TestScenarioBuilder) AddMissingRequiredFields(path string, method string, emptyBody interface{}) *TestScenarioBuilder {
	b.scenarios = append(b.scenarios, ErrorTestScenario{
		Name: "MissingRequiredFields",
		Request: HTTPTestRequest{
			Method: method,
			Path:   path,
			Body:   emptyBody,
		},
		ExpectedStatus: http.StatusBadRequest,
		Description:    "Should reject request with missing required fields",
	})
	return b
}

// AddInvalidMethod adds a test case for invalid HTTP method
func (b *TestScenarioBuilder) AddInvalidMethod(path string, invalidMethod string) *TestScenarioBuilder {
	b.scenarios = append(b.scenarios, ErrorTestScenario{
		Name: "InvalidMethod",
		Request: HTTPTestRequest{
			Method: invalidMethod,
			Path:   path,
		},
		ExpectedStatus: http.StatusMethodNotAllowed,
		Description:    "Should reject invalid HTTP method",
	})
	return b
}

// AddNegativeValues adds a test case for negative input values
func (b *TestScenarioBuilder) AddNegativeValues(path string, method string, negativeInput interface{}) *TestScenarioBuilder {
	b.scenarios = append(b.scenarios, ErrorTestScenario{
		Name: "NegativeValues",
		Request: HTTPTestRequest{
			Method: method,
			Path:   path,
			Body:   negativeInput,
		},
		ExpectedStatus: http.StatusBadRequest,
		Description:    "Should reject negative values where inappropriate",
	})
	return b
}

// AddZeroValues adds a test case for zero input values
func (b *TestScenarioBuilder) AddZeroValues(path string, method string, zeroInput interface{}) *TestScenarioBuilder {
	b.scenarios = append(b.scenarios, ErrorTestScenario{
		Name: "ZeroValues",
		Request: HTTPTestRequest{
			Method: method,
			Path:   path,
			Body:   zeroInput,
		},
		ExpectedStatus: http.StatusBadRequest,
		Description:    "Should reject zero values where inappropriate",
	})
	return b
}

// Build returns the built scenarios
func (b *TestScenarioBuilder) Build() []ErrorTestScenario {
	return b.scenarios
}

// RunErrorScenarios executes a suite of error test scenarios
func RunErrorScenarios(t *testing.T, handler http.HandlerFunc, scenarios []ErrorTestScenario) {
	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			w, err := makeHTTPRequest(scenario.Request, handler)
			if err != nil {
				t.Fatalf("Failed to make request: %v", err)
			}

			if w.Code != scenario.ExpectedStatus {
				t.Errorf("%s: expected status %d, got %d",
					scenario.Description, scenario.ExpectedStatus, w.Code)
				t.Logf("Response body: %s", w.Body.String())
			}
		})
	}
}

// PerformanceTestScenario defines a performance test case
type PerformanceTestScenario struct {
	Name            string
	Request         HTTPTestRequest
	MaxResponseTime int64 // in milliseconds
	Description     string
}
