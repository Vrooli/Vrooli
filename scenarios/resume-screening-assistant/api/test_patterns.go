package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestScenario represents a single test scenario with expected results
type TestScenario struct {
	Name           string
	Method         string
	Path           string
	Body           interface{}
	ExpectedStatus int
	ExpectedError  string
	ValidateFunc   func(*testing.T, map[string]interface{})
}

// TestScenarioBuilder provides a fluent interface for building test scenarios
type TestScenarioBuilder struct {
	scenarios []TestScenario
}

// NewTestScenarioBuilder creates a new test scenario builder
func NewTestScenarioBuilder() *TestScenarioBuilder {
	return &TestScenarioBuilder{
		scenarios: make([]TestScenario, 0),
	}
}

// AddScenario adds a custom test scenario
func (b *TestScenarioBuilder) AddScenario(scenario TestScenario) *TestScenarioBuilder {
	b.scenarios = append(b.scenarios, scenario)
	return b
}

// AddInvalidMethod adds a test for invalid HTTP method
func (b *TestScenarioBuilder) AddInvalidMethod(path, invalidMethod string) *TestScenarioBuilder {
	b.scenarios = append(b.scenarios, TestScenario{
		Name:           "Invalid HTTP method: " + invalidMethod,
		Method:         invalidMethod,
		Path:           path,
		ExpectedStatus: http.StatusMethodNotAllowed,
	})
	return b
}

// AddMissingParameter adds a test for missing query parameter
func (b *TestScenarioBuilder) AddMissingParameter(path, paramName string) *TestScenarioBuilder {
	b.scenarios = append(b.scenarios, TestScenario{
		Name:           "Missing parameter: " + paramName,
		Method:         "GET",
		Path:           path,
		ExpectedStatus: http.StatusOK, // The API should handle missing params gracefully
	})
	return b
}

// AddInvalidParameter adds a test for invalid parameter value
func (b *TestScenarioBuilder) AddInvalidParameter(path, paramName, invalidValue string) *TestScenarioBuilder {
	b.scenarios = append(b.scenarios, TestScenario{
		Name:           "Invalid parameter: " + paramName + "=" + invalidValue,
		Method:         "GET",
		Path:           path + "?" + paramName + "=" + invalidValue,
		ExpectedStatus: http.StatusOK, // Should handle gracefully
	})
	return b
}

// Build returns the list of test scenarios
func (b *TestScenarioBuilder) Build() []TestScenario {
	return b.scenarios
}

// ErrorTestPattern provides systematic error condition testing
type ErrorTestPattern struct {
	scenarios []TestScenario
}

// NewErrorTestPattern creates a new error test pattern
func NewErrorTestPattern() *ErrorTestPattern {
	return &ErrorTestPattern{
		scenarios: make([]TestScenario, 0),
	}
}

// AddScenarios adds multiple scenarios to test
func (p *ErrorTestPattern) AddScenarios(scenarios []TestScenario) *ErrorTestPattern {
	p.scenarios = append(p.scenarios, scenarios...)
	return p
}

// Execute runs all error test scenarios
func (p *ErrorTestPattern) Execute(t *testing.T, router http.Handler) {
	for _, scenario := range p.scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			rr := makeHTTPRequest(t, &router, scenario.Method, scenario.Path, scenario.Body)

			if rr.Code != scenario.ExpectedStatus {
				t.Errorf("Expected status %d, got %d", scenario.ExpectedStatus, rr.Code)
			}

			if scenario.ValidateFunc != nil {
				response := assertJSONResponse(t, rr, scenario.ExpectedStatus)
				scenario.ValidateFunc(t, response)
			}
		})
	}
}

// HandlerTestSuite provides comprehensive HTTP handler testing
type HandlerTestSuite struct {
	router http.Handler
	config *Config
}

// NewHandlerTestSuite creates a new handler test suite
func NewHandlerTestSuite(config *Config) *HandlerTestSuite {
	return &HandlerTestSuite{
		router: createTestRouter(config),
		config: config,
	}
}

// TestEndpoint tests a single endpoint with given parameters
func (s *HandlerTestSuite) TestEndpoint(t *testing.T, method, path string, expectedStatus int) *httptest.ResponseRecorder {
	return makeHTTPRequest(t, &s.router, method, path, nil)
}

// TestEndpointWithBody tests an endpoint with a request body
func (s *HandlerTestSuite) TestEndpointWithBody(t *testing.T, method, path string, body interface{}, expectedStatus int) *httptest.ResponseRecorder {
	return makeHTTPRequest(t, &s.router, method, path, body)
}

// TestSuccessPath tests the happy path for an endpoint
func (s *HandlerTestSuite) TestSuccessPath(t *testing.T, method, path string) map[string]interface{} {
	rr := s.TestEndpoint(t, method, path, http.StatusOK)
	return assertSuccessResponse(t, rr)
}

// TestErrorPath tests error conditions for an endpoint
func (s *HandlerTestSuite) TestErrorPath(t *testing.T, method, path string, expectedStatus int) map[string]interface{} {
	rr := s.TestEndpoint(t, method, path, expectedStatus)
	return assertJSONResponse(t, rr, expectedStatus)
}

// RunErrorScenarios executes a set of error test scenarios
func (s *HandlerTestSuite) RunErrorScenarios(t *testing.T, scenarios []TestScenario) {
	pattern := NewErrorTestPattern()
	pattern.AddScenarios(scenarios)
	pattern.Execute(t, s.router)
}

// PerformanceTestConfig holds configuration for performance tests
type PerformanceTestConfig struct {
	Iterations      int
	MaxDuration     int64 // milliseconds
	ConcurrentUsers int
}

// DefaultPerformanceConfig returns default performance test configuration
func DefaultPerformanceConfig() *PerformanceTestConfig {
	return &PerformanceTestConfig{
		Iterations:      100,
		MaxDuration:     5000, // 5 seconds
		ConcurrentUsers: 10,
	}
}

// EdgeCaseTestBuilder builds edge case test scenarios
type EdgeCaseTestBuilder struct {
	scenarios []TestScenario
}

// NewEdgeCaseTestBuilder creates a new edge case test builder
func NewEdgeCaseTestBuilder() *EdgeCaseTestBuilder {
	return &EdgeCaseTestBuilder{
		scenarios: make([]TestScenario, 0),
	}
}

// AddEmptyInput tests empty input handling
func (b *EdgeCaseTestBuilder) AddEmptyInput(path string) *EdgeCaseTestBuilder {
	b.scenarios = append(b.scenarios, TestScenario{
		Name:           "Empty input",
		Method:         "GET",
		Path:           path,
		ExpectedStatus: http.StatusOK,
	})
	return b
}

// AddNullValue tests null value handling
func (b *EdgeCaseTestBuilder) AddNullValue(path, param string) *EdgeCaseTestBuilder {
	b.scenarios = append(b.scenarios, TestScenario{
		Name:           "Null value: " + param,
		Method:         "GET",
		Path:           path + "?" + param + "=",
		ExpectedStatus: http.StatusOK,
	})
	return b
}

// AddBoundaryValue tests boundary value handling
func (b *EdgeCaseTestBuilder) AddBoundaryValue(path, param, value string) *EdgeCaseTestBuilder {
	b.scenarios = append(b.scenarios, TestScenario{
		Name:           "Boundary value: " + param + "=" + value,
		Method:         "GET",
		Path:           path + "?" + param + "=" + value,
		ExpectedStatus: http.StatusOK,
	})
	return b
}

// Build returns the list of edge case test scenarios
func (b *EdgeCaseTestBuilder) Build() []TestScenario {
	return b.scenarios
}
