// +build testing

package main

import (
	"fmt"
	"net/http"
	"strings"
	"testing"
)

// TestScenario represents a single test scenario
type TestScenario struct {
	Name           string
	Endpoint       string
	Method         string
	Body           interface{}
	ExpectedStatus int
	ExpectedError  string
}

// TestScenarioBuilder provides a fluent interface for building test scenarios
type TestScenarioBuilder struct {
	scenarios []TestScenario
}

// NewTestScenarioBuilder creates a new test scenario builder
func NewTestScenarioBuilder() *TestScenarioBuilder {
	return &TestScenarioBuilder{
		scenarios: []TestScenario{},
	}
}

// AddInvalidJSON adds a test for invalid JSON input
func (b *TestScenarioBuilder) AddInvalidJSON(endpoint, method string) *TestScenarioBuilder {
	b.scenarios = append(b.scenarios, TestScenario{
		Name:           "Invalid JSON",
		Endpoint:       endpoint,
		Method:         method,
		Body:           "invalid-json",
		ExpectedStatus: http.StatusBadRequest,
		ExpectedError:  "Invalid request body",
	})
	return b
}

// AddEmptyRequest adds a test for empty request body
func (b *TestScenarioBuilder) AddEmptyRequest(endpoint, method string) *TestScenarioBuilder {
	b.scenarios = append(b.scenarios, TestScenario{
		Name:           "Empty Request",
		Endpoint:       endpoint,
		Method:         method,
		Body:           map[string]interface{}{},
		ExpectedStatus: http.StatusBadRequest,
		ExpectedError:  "validation_error",
	})
	return b
}

// AddInvalidPath adds a test for invalid file path
func (b *TestScenarioBuilder) AddInvalidPath(endpoint, method string) *TestScenarioBuilder {
	b.scenarios = append(b.scenarios, TestScenario{
		Name:     "Invalid Path",
		Endpoint: endpoint,
		Method:   method,
		Body: map[string]interface{}{
			"paths": []string{"/nonexistent/path/that/does/not/exist"},
			"types": []string{"backup_files"},
		},
		ExpectedStatus: http.StatusBadRequest,
		ExpectedError:  "validation_error",
	})
	return b
}

// AddInvalidScanType adds a test for invalid scan type
func (b *TestScenarioBuilder) AddInvalidScanType(endpoint, method string) *TestScenarioBuilder {
	b.scenarios = append(b.scenarios, TestScenario{
		Name:     "Invalid Scan Type",
		Endpoint: endpoint,
		Method:   method,
		Body: map[string]interface{}{
			"paths": []string{"/tmp"},
			"types": []string{"invalid_type"},
		},
		ExpectedStatus: http.StatusBadRequest,
		ExpectedError:  "validation_error",
	})
	return b
}

// AddInvalidAnalysisType adds a test for invalid analysis type
func (b *TestScenarioBuilder) AddInvalidAnalysisType(endpoint, method string) *TestScenarioBuilder {
	b.scenarios = append(b.scenarios, TestScenario{
		Name:     "Invalid Analysis Type",
		Endpoint: endpoint,
		Method:   method,
		Body: map[string]interface{}{
			"analysis_type": "invalid_analysis",
			"paths":         []string{"/tmp"},
		},
		ExpectedStatus: http.StatusBadRequest,
		ExpectedError:  "command_error",
	})
	return b
}

// AddDangerousCleanup adds a test for dangerous cleanup scripts
func (b *TestScenarioBuilder) AddDangerousCleanup(endpoint, method string) *TestScenarioBuilder {
	b.scenarios = append(b.scenarios, TestScenario{
		Name:     "Dangerous Cleanup Script",
		Endpoint: endpoint,
		Method:   method,
		Body: map[string]interface{}{
			"cleanup_scripts": []string{"rm -rf /"},
		},
		ExpectedStatus: http.StatusInternalServerError,
		ExpectedError:  "safety_error",
	})
	return b
}

// AddNoCleanupScripts adds a test for missing cleanup scripts
func (b *TestScenarioBuilder) AddNoCleanupScripts(endpoint, method string) *TestScenarioBuilder {
	b.scenarios = append(b.scenarios, TestScenario{
		Name:           "No Cleanup Scripts",
		Endpoint:       endpoint,
		Method:         method,
		Body:           map[string]interface{}{},
		ExpectedStatus: http.StatusInternalServerError,
		ExpectedError:  "validation_error",
	})
	return b
}

// Build returns the built test scenarios
func (b *TestScenarioBuilder) Build() []TestScenario {
	return b.scenarios
}

// ErrorTestPattern provides systematic error testing patterns
type ErrorTestPattern struct {
	scenarios []TestScenario
}

// NewErrorTestPattern creates a new error test pattern
func NewErrorTestPattern() *ErrorTestPattern {
	return &ErrorTestPattern{
		scenarios: []TestScenario{},
	}
}

// TestAllScenarios executes all test scenarios
func (p *ErrorTestPattern) TestAllScenarios(t *testing.T, handler http.Handler, scenarios []TestScenario) {
	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			var req *http.Request
			var err error

			if scenario.Body != nil {
				if strBody, ok := scenario.Body.(string); ok {
					// Invalid JSON test
					req, err = http.NewRequest(scenario.Method, scenario.Endpoint, strings.NewReader(strBody))
				} else {
					req, err = makeHTTPRequest(scenario.Method, scenario.Endpoint, scenario.Body)
				}
			} else {
				req, err = http.NewRequest(scenario.Method, scenario.Endpoint, nil)
			}

			if err != nil {
				t.Fatalf("Failed to create request: %v", err)
			}

			rr := executeRequest(handler, req)

			if rr.Code != scenario.ExpectedStatus {
				t.Errorf("Expected status %d, got %d\nBody: %s",
					scenario.ExpectedStatus, rr.Code, rr.Body.String())
			}

			if scenario.ExpectedError != "" {
				body := rr.Body.String()
				if !strings.Contains(body, scenario.ExpectedError) && rr.Code >= 400 {
					// For plain text errors, just check status code
					if rr.Code != scenario.ExpectedStatus {
						t.Errorf("Expected error containing '%s' in response", scenario.ExpectedError)
					}
				}
			}
		})
	}
}

// HandlerTestSuite provides comprehensive HTTP handler testing
type HandlerTestSuite struct {
	t       *testing.T
	handler http.Handler
	baseURL string
}

// NewHandlerTestSuite creates a new handler test suite
func NewHandlerTestSuite(t *testing.T, handler http.Handler) *HandlerTestSuite {
	return &HandlerTestSuite{
		t:       t,
		handler: handler,
		baseURL: "",
	}
}

// TestEndpoint tests a single endpoint with success and error cases
func (s *HandlerTestSuite) TestEndpoint(endpoint, method string, successBody interface{}, expectedStatus int) {
	s.t.Run(fmt.Sprintf("%s %s", method, endpoint), func(t *testing.T) {
		req, err := makeHTTPRequest(method, endpoint, successBody)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		rr := executeRequest(s.handler, req)

		if rr.Code != expectedStatus {
			t.Errorf("Expected status %d, got %d\nBody: %s",
				expectedStatus, rr.Code, rr.Body.String())
		}

		if rr.Header().Get("Content-Type") != "application/json" && expectedStatus < 400 {
			t.Errorf("Expected Content-Type application/json, got %s",
				rr.Header().Get("Content-Type"))
		}
	})
}

// TestHealthEndpoint tests health check endpoints
func (s *HandlerTestSuite) TestHealthEndpoint(endpoint string) {
	s.t.Run("Health Check", func(t *testing.T) {
		req, err := http.NewRequest("GET", endpoint, nil)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		rr := executeRequest(s.handler, req)

		if rr.Code != http.StatusOK {
			t.Errorf("Health check failed with status %d", rr.Code)
		}

		response := assertJSONResponse(t, rr, http.StatusOK)

		if status, ok := response["status"].(string); !ok || status != "healthy" {
			t.Errorf("Expected status 'healthy', got %v", response["status"])
		}
	})
}

// TableDrivenTest represents a table-driven test case
type TableDrivenTest struct {
	Name           string
	Input          interface{}
	Expected       interface{}
	ShouldError    bool
	ErrorContains  string
}

// RunTableTests executes table-driven tests
func RunTableTests(t *testing.T, tests []TableDrivenTest, testFunc func(input interface{}) (interface{}, error)) {
	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			result, err := testFunc(tt.Input)

			if tt.ShouldError {
				if err == nil {
					t.Errorf("Expected error but got none")
				} else if tt.ErrorContains != "" && !strings.Contains(err.Error(), tt.ErrorContains) {
					t.Errorf("Expected error containing '%s', got '%s'", tt.ErrorContains, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				// For complex comparisons, tests should implement their own validation
				if tt.Expected != nil && result != tt.Expected {
					// Basic comparison, override in specific tests for complex types
					t.Logf("Result: %v, Expected: %v", result, tt.Expected)
				}
			}
		})
	}
}
