package main

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

// ErrorTestPattern defines systematic error condition testing
type ErrorTestPattern struct {
	Name           string
	Description    string
	Method         string
	Path           string
	Body           interface{}
	ExpectedStatus int
}

// TestScenarioBuilder provides fluent interface for building test scenarios
type TestScenarioBuilder struct {
	scenarios []ErrorTestPattern
}

// NewTestScenarioBuilder creates a new test scenario builder
func NewTestScenarioBuilder() *TestScenarioBuilder {
	return &TestScenarioBuilder{
		scenarios: []ErrorTestPattern{},
	}
}

// AddInvalidJSON adds test for malformed JSON
func (b *TestScenarioBuilder) AddInvalidJSON(method, path string) *TestScenarioBuilder {
	b.scenarios = append(b.scenarios, ErrorTestPattern{
		Name:           "InvalidJSON",
		Description:    "Test with malformed JSON body",
		Method:         method,
		Path:           path,
		Body:           "invalid-json",
		ExpectedStatus: http.StatusBadRequest,
	})
	return b
}

// AddMissingRequiredField adds test for missing required field
func (b *TestScenarioBuilder) AddMissingRequiredField(method, path, field string, body interface{}) *TestScenarioBuilder {
	b.scenarios = append(b.scenarios, ErrorTestPattern{
		Name:           fmt.Sprintf("Missing_%s", field),
		Description:    fmt.Sprintf("Test with missing required field: %s", field),
		Method:         method,
		Path:           path,
		Body:           body,
		ExpectedStatus: http.StatusBadRequest,
	})
	return b
}

// AddEmptyField adds test for empty field value
func (b *TestScenarioBuilder) AddEmptyField(method, path, field string, body interface{}) *TestScenarioBuilder {
	b.scenarios = append(b.scenarios, ErrorTestPattern{
		Name:           fmt.Sprintf("Empty_%s", field),
		Description:    fmt.Sprintf("Test with empty field: %s", field),
		Method:         method,
		Path:           path,
		Body:           body,
		ExpectedStatus: http.StatusBadRequest,
	})
	return b
}

// AddMethodNotAllowed adds test for wrong HTTP method
func (b *TestScenarioBuilder) AddMethodNotAllowed(path, wrongMethod string) *TestScenarioBuilder {
	b.scenarios = append(b.scenarios, ErrorTestPattern{
		Name:           fmt.Sprintf("MethodNotAllowed_%s", wrongMethod),
		Description:    fmt.Sprintf("Test with wrong HTTP method: %s", wrongMethod),
		Method:         wrongMethod,
		Path:           path,
		Body:           nil,
		ExpectedStatus: http.StatusMethodNotAllowed,
	})
	return b
}

// Build returns the built test scenarios
func (b *TestScenarioBuilder) Build() []ErrorTestPattern {
	return b.scenarios
}

// RunErrorPatternTests executes a set of error pattern tests
func RunErrorPatternTests(t *testing.T, patterns []ErrorTestPattern, handler http.HandlerFunc) {
	for _, pattern := range patterns {
		t.Run(pattern.Name, func(t *testing.T) {
			var req HTTPTestRequest

			if pattern.Body == "invalid-json" {
				// Create request with truly invalid JSON
				req = HTTPTestRequest{
					Method: pattern.Method,
					Path:   pattern.Path,
					Body:   nil,
				}
				w := httptest.NewRecorder()
				httpReq, _ := http.NewRequest(pattern.Method, pattern.Path, bytes.NewBufferString("{invalid-json}"))
				httpReq.Header.Set("Content-Type", "application/json")
				handler(w, httpReq)

				if w.Code != pattern.ExpectedStatus {
					t.Errorf("Expected status %d, got %d. Body: %s", pattern.ExpectedStatus, w.Code, w.Body.String())
				}
				return
			}

			req = HTTPTestRequest{
				Method: pattern.Method,
				Path:   pattern.Path,
				Body:   pattern.Body,
			}

			w, err := makeHTTPRequest(req, handler)
			if err != nil {
				t.Fatalf("Failed to make HTTP request: %v", err)
			}

			if w.Code != pattern.ExpectedStatus {
				t.Errorf("[%s] Expected status %d, got %d. Body: %s",
					pattern.Description, pattern.ExpectedStatus, w.Code, w.Body.String())
			}
		})
	}
}

// HandlerTestSuite provides comprehensive HTTP handler testing framework
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

// TestSuccessCase runs a success case test
func (s *HandlerTestSuite) TestSuccessCase(t *testing.T, req HTTPTestRequest, validator func(*testing.T, *httptest.ResponseRecorder)) {
	t.Run(fmt.Sprintf("%s_Success", s.HandlerName), func(t *testing.T) {
		w, err := makeHTTPRequest(req, s.Handler)
		if err != nil {
			t.Fatalf("Failed to make HTTP request: %v", err)
		}

		if validator != nil {
			validator(t, w)
		}
	})
}

// TestErrorCases runs multiple error case tests
func (s *HandlerTestSuite) TestErrorCases(t *testing.T, patterns []ErrorTestPattern) {
	RunErrorPatternTests(t, patterns, s.Handler)
}
