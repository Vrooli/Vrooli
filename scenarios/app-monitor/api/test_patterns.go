// +build testing

package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

// ErrorTestPattern defines a systematic approach to testing error conditions
type ErrorTestPattern struct {
	Name           string
	Description    string
	ExpectedStatus int
	Setup          func(t *testing.T) interface{}
	Execute        func(t *testing.T, router *gin.Engine, setupData interface{}) *httptest.ResponseRecorder
	Validate       func(t *testing.T, w *httptest.ResponseRecorder, setupData interface{})
	Cleanup        func(setupData interface{})
}

// HandlerTestSuite provides a comprehensive test framework for HTTP handlers
type HandlerTestSuite struct {
	HandlerName string
	Router      *gin.Engine
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
			w := pattern.Execute(t, suite.Router, setupData)

			// Validate status code
			if w.Code != pattern.ExpectedStatus {
				t.Errorf("Expected status %d, got %d. Body: %s", pattern.ExpectedStatus, w.Code, w.Body.String())
			}

			// Additional validation
			if pattern.Validate != nil {
				pattern.Validate(t, w, setupData)
			}
		})
	}
}

// TestScenarioBuilder helps build comprehensive test scenarios
type TestScenarioBuilder struct {
	patterns []ErrorTestPattern
}

// NewTestScenarioBuilder creates a new builder
func NewTestScenarioBuilder() *TestScenarioBuilder {
	return &TestScenarioBuilder{
		patterns: []ErrorTestPattern{},
	}
}

// AddInvalidID adds a test for invalid app ID
func (b *TestScenarioBuilder) AddInvalidID(path string, method string) *TestScenarioBuilder {
	b.patterns = append(b.patterns, ErrorTestPattern{
		Name:           "InvalidID",
		Description:    "Test handler with invalid app ID format",
		ExpectedStatus: http.StatusNotFound,
		Execute: func(t *testing.T, router *gin.Engine, setupData interface{}) *httptest.ResponseRecorder {
			req, _ := makeHTTPRequest(HTTPTestRequest{
				Method: method,
				Path:   path,
			})
			router.ServeHTTP(req, httptest.NewRequest(method, path, nil))
			return req
		},
	})
	return b
}

// AddNonExistentApp adds a test for non-existent app
func (b *TestScenarioBuilder) AddNonExistentApp(path string, method string) *TestScenarioBuilder {
	b.patterns = append(b.patterns, ErrorTestPattern{
		Name:           "NonExistentApp",
		Description:    "Test handler with non-existent app ID",
		ExpectedStatus: http.StatusNotFound,
		Execute: func(t *testing.T, router *gin.Engine, setupData interface{}) *httptest.ResponseRecorder {
			req, _ := makeHTTPRequest(HTTPTestRequest{
				Method: method,
				Path:   path,
			})
			router.ServeHTTP(req, httptest.NewRequest(method, path, nil))
			return req
		},
	})
	return b
}

// AddInvalidJSON adds a test for invalid JSON payload
func (b *TestScenarioBuilder) AddInvalidJSON(path string, method string) *TestScenarioBuilder {
	b.patterns = append(b.patterns, ErrorTestPattern{
		Name:           "InvalidJSON",
		Description:    "Test handler with malformed JSON",
		ExpectedStatus: http.StatusBadRequest,
		Execute: func(t *testing.T, router *gin.Engine, setupData interface{}) *httptest.ResponseRecorder {
			req := httptest.NewRequest(method, path, strings.NewReader("{invalid json}"))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
			return w
		},
	})
	return b
}

// AddMissingRequiredField adds a test for missing required fields
func (b *TestScenarioBuilder) AddMissingRequiredField(path string, method string, fieldName string) *TestScenarioBuilder {
	b.patterns = append(b.patterns, ErrorTestPattern{
		Name:           fmt.Sprintf("Missing%s", fieldName),
		Description:    fmt.Sprintf("Test handler with missing %s field", fieldName),
		ExpectedStatus: http.StatusBadRequest,
		Execute: func(t *testing.T, router *gin.Engine, setupData interface{}) *httptest.ResponseRecorder {
			req, _ := makeHTTPRequest(HTTPTestRequest{
				Method: method,
				Path:   path,
				Body:   map[string]interface{}{}, // Empty body
			})
			router.ServeHTTP(req, httptest.NewRequest(method, path, nil))
			return req
		},
	})
	return b
}

// AddUnauthorized adds a test for unauthorized access
func (b *TestScenarioBuilder) AddUnauthorized(path string, method string) *TestScenarioBuilder {
	b.patterns = append(b.patterns, ErrorTestPattern{
		Name:           "Unauthorized",
		Description:    "Test handler with missing or invalid API key",
		ExpectedStatus: http.StatusUnauthorized,
		Execute: func(t *testing.T, router *gin.Engine, setupData interface{}) *httptest.ResponseRecorder {
			req := httptest.NewRequest(method, path, nil)
			// Don't set API key header
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
			return w
		},
	})
	return b
}

// AddServerError adds a test for simulated server errors
func (b *TestScenarioBuilder) AddServerError(path string, method string) *TestScenarioBuilder {
	b.patterns = append(b.patterns, ErrorTestPattern{
		Name:           "ServerError",
		Description:    "Test handler with simulated server error",
		ExpectedStatus: http.StatusInternalServerError,
		Execute: func(t *testing.T, router *gin.Engine, setupData interface{}) *httptest.ResponseRecorder {
			req, _ := makeHTTPRequest(HTTPTestRequest{
				Method: method,
				Path:   path,
			})
			router.ServeHTTP(req, httptest.NewRequest(method, path, nil))
			return req
		},
	})
	return b
}

// Build returns the built patterns
func (b *TestScenarioBuilder) Build() []ErrorTestPattern {
	return b.patterns
}

// Common error patterns that can be reused

// invalidIDPattern tests handlers with invalid app IDs
func invalidIDPattern(urlPath string, method string) ErrorTestPattern {
	return ErrorTestPattern{
		Name:           "InvalidID",
		Description:    "Test handler with invalid app ID",
		ExpectedStatus: http.StatusNotFound,
		Setup:          nil,
		Execute: func(t *testing.T, router *gin.Engine, setupData interface{}) *httptest.ResponseRecorder {
			req := httptest.NewRequest(method, urlPath, nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
			return w
		},
	}
}

// nonExistentAppPattern tests handlers with non-existent app IDs
func nonExistentAppPattern(urlPath string, method string) ErrorTestPattern {
	return ErrorTestPattern{
		Name:           "NonExistentApp",
		Description:    "Test handler with non-existent app ID",
		ExpectedStatus: http.StatusNotFound,
		Setup:          nil,
		Execute: func(t *testing.T, router *gin.Engine, setupData interface{}) *httptest.ResponseRecorder {
			req := httptest.NewRequest(method, urlPath, nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
			return w
		},
	}
}

// invalidJSONPattern tests handlers with malformed JSON
func invalidJSONPattern(urlPath string, method string) ErrorTestPattern {
	return ErrorTestPattern{
		Name:           "InvalidJSON",
		Description:    "Test handler with malformed JSON payload",
		ExpectedStatus: http.StatusBadRequest,
		Setup:          nil,
		Execute: func(t *testing.T, router *gin.Engine, setupData interface{}) *httptest.ResponseRecorder {
			req := httptest.NewRequest(method, urlPath, strings.NewReader("{invalid json"))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
			return w
		},
	}
}

// emptyBodyPattern tests handlers with empty request body
func emptyBodyPattern(urlPath string, method string) ErrorTestPattern {
	return ErrorTestPattern{
		Name:           "EmptyBody",
		Description:    "Test handler with empty request body",
		ExpectedStatus: http.StatusBadRequest,
		Setup:          nil,
		Execute: func(t *testing.T, router *gin.Engine, setupData interface{}) *httptest.ResponseRecorder {
			req := httptest.NewRequest(method, urlPath, nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
			return w
		},
	}
}
