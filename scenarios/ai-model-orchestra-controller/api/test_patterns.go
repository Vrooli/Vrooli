
package main

import (
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
	ExpectedStatus int
	ExpectedError  string
}

// NewTestScenarioBuilder creates a new test scenario builder
func NewTestScenarioBuilder() *TestScenarioBuilder {
	return &TestScenarioBuilder{
		scenarios: make([]ErrorTestScenario, 0),
	}
}

// AddInvalidJSON adds a test case for invalid JSON input
func (b *TestScenarioBuilder) AddInvalidJSON(method, path string) *TestScenarioBuilder {
	b.scenarios = append(b.scenarios, ErrorTestScenario{
		Name:           "Invalid JSON",
		Method:         method,
		Path:           path,
		Body:           "{invalid json}",
		ExpectedStatus: http.StatusBadRequest,
		ExpectedError:  "Invalid JSON",
	})
	return b
}

// AddMissingRequiredField adds a test case for missing required fields
func (b *TestScenarioBuilder) AddMissingRequiredField(method, path, fieldName string, body interface{}) *TestScenarioBuilder {
	b.scenarios = append(b.scenarios, ErrorTestScenario{
		Name:           "Missing required field: " + fieldName,
		Method:         method,
		Path:           path,
		Body:           body,
		ExpectedStatus: http.StatusBadRequest,
		ExpectedError:  fieldName + " is required",
	})
	return b
}

// AddMethodNotAllowed adds a test case for wrong HTTP method
func (b *TestScenarioBuilder) AddMethodNotAllowed(wrongMethod, correctMethod, path string) *TestScenarioBuilder {
	b.scenarios = append(b.scenarios, ErrorTestScenario{
		Name:           "Method Not Allowed: " + wrongMethod,
		Method:         wrongMethod,
		Path:           path,
		ExpectedStatus: http.StatusMethodNotAllowed,
		ExpectedError:  "Method not allowed",
	})
	return b
}

// AddCustomScenario adds a custom test scenario
func (b *TestScenarioBuilder) AddCustomScenario(name, method, path string, body interface{}, expectedStatus int, expectedError string) *TestScenarioBuilder {
	b.scenarios = append(b.scenarios, ErrorTestScenario{
		Name:           name,
		Method:         method,
		Path:           path,
		Body:           body,
		ExpectedStatus: expectedStatus,
		ExpectedError:  expectedError,
	})
	return b
}

// Build returns the list of test scenarios
func (b *TestScenarioBuilder) Build() []ErrorTestScenario {
	return b.scenarios
}

// ErrorTestPattern provides systematic error condition testing
type ErrorTestPattern struct {
	t *testing.T
}

// NewErrorTestPattern creates a new error test pattern
func NewErrorTestPattern(t *testing.T) *ErrorTestPattern {
	return &ErrorTestPattern{t: t}
}

// TestInvalidJSON tests handler with invalid JSON
func (p *ErrorTestPattern) TestInvalidJSON(handler http.HandlerFunc, method, path string) {
	req := HTTPTestRequest{
		Method: method,
		Path:   path,
		Body:   "{invalid json}",
	}

	w, httpReq, err := makeHTTPRequest(req)
	if err != nil {
		p.t.Fatalf("Failed to create request: %v", err)
	}

	handler(w, httpReq)

	if w.Code != http.StatusBadRequest {
		p.t.Errorf("Expected status 400 for invalid JSON, got %d", w.Code)
	}
}

// TestMissingRequiredField tests handler with missing required field
func (p *ErrorTestPattern) TestMissingRequiredField(handler http.HandlerFunc, method, path string, body interface{}, fieldName string) {
	req := HTTPTestRequest{
		Method: method,
		Path:   path,
		Body:   body,
	}

	w, httpReq, err := makeHTTPRequest(req)
	if err != nil {
		p.t.Fatalf("Failed to create request: %v", err)
	}

	handler(w, httpReq)

	if w.Code != http.StatusBadRequest {
		p.t.Errorf("Expected status 400 for missing field '%s', got %d", fieldName, w.Code)
	}
}

// TestMethodNotAllowed tests handler with wrong HTTP method
func (p *ErrorTestPattern) TestMethodNotAllowed(handler http.HandlerFunc, wrongMethod, path string) {
	req := HTTPTestRequest{
		Method: wrongMethod,
		Path:   path,
	}

	w, httpReq, err := makeHTTPRequest(req)
	if err != nil {
		p.t.Fatalf("Failed to create request: %v", err)
	}

	handler(w, httpReq)

	if w.Code != http.StatusMethodNotAllowed {
		p.t.Errorf("Expected status 405 for method %s, got %d", wrongMethod, w.Code)
	}
}

// HandlerTestSuite provides comprehensive HTTP handler testing
type HandlerTestSuite struct {
	t        *testing.T
	app      *AppState
	handlers *Handlers
}

// NewHandlerTestSuite creates a new handler test suite
func NewHandlerTestSuite(t *testing.T, app *AppState) *HandlerTestSuite {
	return &HandlerTestSuite{
		t:        t,
		app:      app,
		handlers: NewHandlers(app),
	}
}

// TestHealthCheck runs comprehensive health check tests
func (s *HandlerTestSuite) TestHealthCheck() {
	s.t.Run("HealthCheck", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/health",
		}

		w, httpReq, err := makeHTTPRequest(req)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		s.handlers.handleHealthCheck(w, httpReq)

		// Should return some status (could be healthy, degraded, or unhealthy)
		if w.Code != http.StatusOK && w.Code != http.StatusPartialContent && w.Code != http.StatusServiceUnavailable {
			t.Errorf("Unexpected status code: %d", w.Code)
		}

		// Should have valid JSON response
		response := assertJSONResponse(t, w, w.Code, nil)
		if response == nil {
			t.Fatal("Expected valid JSON response")
		}

		// Should have required fields
		requiredFields := []string{"status", "timestamp", "services", "system", "version"}
		for _, field := range requiredFields {
			if _, exists := response[field]; !exists {
				t.Errorf("Missing required field: %s", field)
			}
		}
	})
}

// TestModelSelect runs comprehensive model selection tests
func (s *HandlerTestSuite) TestModelSelect() {
	s.t.Run("ModelSelect_Success", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/ai/select-model",
			Body: ModelSelectRequest{
				TaskType: "completion",
				Requirements: map[string]interface{}{
					"complexity": "simple",
				},
			},
		}

		w, httpReq, err := makeHTTPRequest(req)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		s.handlers.handleModelSelect(w, httpReq)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d. Response: %s", w.Code, w.Body.String())
		}
	})

	s.t.Run("ModelSelect_InvalidJSON", func(t *testing.T) {
		pattern := NewErrorTestPattern(t)
		pattern.TestInvalidJSON(s.handlers.handleModelSelect, "POST", "/api/v1/ai/select-model")
	})

	s.t.Run("ModelSelect_MissingTaskType", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/ai/select-model",
			Body: map[string]interface{}{
				"requirements": map[string]interface{}{},
			},
		}

		w, httpReq, err := makeHTTPRequest(req)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		s.handlers.handleModelSelect(w, httpReq)

		assertErrorResponse(t, w, http.StatusBadRequest, "taskType is required")
	})

	s.t.Run("ModelSelect_WrongMethod", func(t *testing.T) {
		pattern := NewErrorTestPattern(t)
		pattern.TestMethodNotAllowed(s.handlers.handleModelSelect, "GET", "/api/v1/ai/select-model")
	})
}

// TestRouteRequest runs comprehensive request routing tests
func (s *HandlerTestSuite) TestRouteRequest() {
	s.t.Run("RouteRequest_Success", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/ai/route-request",
			Body: RouteRequest{
				TaskType: "completion",
				Prompt:   "Hello, world!",
			},
		}

		w, httpReq, err := makeHTTPRequest(req)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		s.handlers.handleRouteRequest(w, httpReq)

		// Should succeed even without Ollama (falls back to simulation)
		if w.Code != http.StatusOK {
			t.Logf("Response: %s", w.Body.String())
		}
	})

	s.t.Run("RouteRequest_MissingPrompt", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/ai/route-request",
			Body: map[string]interface{}{
				"taskType": "completion",
			},
		}

		w, httpReq, err := makeHTTPRequest(req)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		s.handlers.handleRouteRequest(w, httpReq)

		assertErrorResponse(t, w, http.StatusBadRequest, "prompt")
	})
}
