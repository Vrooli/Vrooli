// +build testing

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
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
	b.patterns = append(b.patterns, ErrorTestPattern{
		Name:           "InvalidUUID",
		Description:    "Test handler with invalid UUID format",
		ExpectedStatus: http.StatusBadRequest,
		Execute: func(t *testing.T, setupData interface{}) *HTTPTestRequest {
			return &HTTPTestRequest{
				Method:  "GET",
				Path:    urlPath,
				URLVars: map[string]string{"id": "invalid-uuid-format"},
			}
		},
	})
	return b
}

// AddNonExistentResource adds a non-existent resource test pattern
func (b *TestScenarioBuilder) AddNonExistentResource(urlPath string) *TestScenarioBuilder {
	b.patterns = append(b.patterns, ErrorTestPattern{
		Name:           "NonExistentResource",
		Description:    "Test handler with non-existent resource",
		ExpectedStatus: http.StatusNotFound,
		Execute: func(t *testing.T, setupData interface{}) *HTTPTestRequest {
			return &HTTPTestRequest{
				Method:  "GET",
				Path:    urlPath,
				URLVars: map[string]string{"resource": "non-existent-resource"},
			}
		},
	})
	return b
}

// AddInvalidJSON adds an invalid JSON test pattern
func (b *TestScenarioBuilder) AddInvalidJSON(urlPath string) *TestScenarioBuilder {
	b.patterns = append(b.patterns, ErrorTestPattern{
		Name:           "InvalidJSON",
		Description:    "Test handler with malformed JSON",
		ExpectedStatus: http.StatusBadRequest,
		Execute: func(t *testing.T, setupData interface{}) *HTTPTestRequest {
			return &HTTPTestRequest{
				Method:  "POST",
				Path:    urlPath,
				Body:    "invalid-json-content",
			}
		},
	})
	return b
}

// AddMissingRequiredField adds a missing required field test pattern
func (b *TestScenarioBuilder) AddMissingRequiredField(urlPath, fieldName string) *TestScenarioBuilder {
	b.patterns = append(b.patterns, ErrorTestPattern{
		Name:           fmt.Sprintf("Missing_%s", fieldName),
		Description:    fmt.Sprintf("Test handler with missing required field: %s", fieldName),
		ExpectedStatus: http.StatusBadRequest,
		Execute: func(t *testing.T, setupData interface{}) *HTTPTestRequest {
			return &HTTPTestRequest{
				Method: "POST",
				Path:   urlPath,
				Body:   map[string]interface{}{},
			}
		},
	})
	return b
}

// AddEmptyBody adds an empty body test pattern
func (b *TestScenarioBuilder) AddEmptyBody(urlPath string) *TestScenarioBuilder {
	b.patterns = append(b.patterns, ErrorTestPattern{
		Name:           "EmptyBody",
		Description:    "Test handler with empty request body",
		ExpectedStatus: http.StatusBadRequest,
		Execute: func(t *testing.T, setupData interface{}) *HTTPTestRequest {
			return &HTTPTestRequest{
				Method: "POST",
				Path:   urlPath,
				Body:   nil,
			}
		},
	})
	return b
}

// AddInvalidSecretType adds an invalid secret type test pattern
func (b *TestScenarioBuilder) AddInvalidSecretType(urlPath string) *TestScenarioBuilder {
	b.patterns = append(b.patterns, ErrorTestPattern{
		Name:           "InvalidSecretType",
		Description:    "Test handler with invalid secret type",
		ExpectedStatus: http.StatusBadRequest,
		Execute: func(t *testing.T, setupData interface{}) *HTTPTestRequest {
			return &HTTPTestRequest{
				Method: "POST",
				Path:   urlPath,
				Body: map[string]interface{}{
					"secret_type": "invalid_type",
					"resource":    "test-resource",
				},
			}
		},
	})
	return b
}

// AddInvalidValidationStatus adds an invalid validation status test pattern
func (b *TestScenarioBuilder) AddInvalidValidationStatus(urlPath string) *TestScenarioBuilder {
	b.patterns = append(b.patterns, ErrorTestPattern{
		Name:           "InvalidValidationStatus",
		Description:    "Test handler with invalid validation status",
		ExpectedStatus: http.StatusBadRequest,
		Execute: func(t *testing.T, setupData interface{}) *HTTPTestRequest {
			return &HTTPTestRequest{
				Method: "POST",
				Path:   urlPath,
				Body: map[string]interface{}{
					"validation_status": "invalid_status",
					"resource_secret_id": uuid.New().String(),
				},
			}
		},
	})
	return b
}

// Build returns the built error patterns
func (b *TestScenarioBuilder) Build() []ErrorTestPattern {
	return b.patterns
}

// HandlerTestSuite provides a comprehensive test framework for HTTP handlers
type HandlerTestSuite struct {
	HandlerName    string
	Handler        http.HandlerFunc
	BaseURL        string
	RequiredURLVars []string
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

			// Execute handler
			if suite.Handler != nil {
				suite.Handler.ServeHTTP(w, req.ToHTTPRequest())
			}

			// Validate status code
			if w.Code != pattern.ExpectedStatus {
				t.Errorf("Expected status %d, got %d. Response: %s",
					pattern.ExpectedStatus, w.Code, w.Body.String())
			}

			// Custom validation
			if pattern.Validate != nil {
				pattern.Validate(t, w, setupData)
			}
		})
	}
}

// ToHTTPRequest converts HTTPTestRequest to http.Request
func (req *HTTPTestRequest) ToHTTPRequest() *http.Request {
	var bodyReader io.Reader
	if req.Body != nil {
		if str, ok := req.Body.(string); ok {
			bodyReader = strings.NewReader(str)
		} else {
			bodyBytes, _ := json.Marshal(req.Body)
			bodyReader = bytes.NewReader(bodyBytes)
		}
	}

	httpReq, _ := http.NewRequest(req.Method, req.Path, bodyReader)

	if req.Headers != nil {
		for key, value := range req.Headers {
			httpReq.Header.Set(key, value)
		}
	}

	if httpReq.Header.Get("Content-Type") == "" && req.Body != nil {
		httpReq.Header.Set("Content-Type", "application/json")
	}

	if req.URLVars != nil {
		httpReq = mux.SetURLVars(httpReq, req.URLVars)
	}

	return httpReq
}

// Common error patterns

// InvalidUUIDPattern creates a standard invalid UUID test pattern
func InvalidUUIDPattern(urlPath string) ErrorTestPattern {
	return ErrorTestPattern{
		Name:           "InvalidUUID",
		Description:    "Test with invalid UUID format",
		ExpectedStatus: http.StatusBadRequest,
		Execute: func(t *testing.T, setupData interface{}) *HTTPTestRequest {
			return &HTTPTestRequest{
				Method:  "GET",
				Path:    urlPath,
				URLVars: map[string]string{"id": "not-a-valid-uuid"},
			}
		},
	}
}

// NonExistentResourcePattern creates a standard non-existent resource test pattern
func NonExistentResourcePattern(urlPath string) ErrorTestPattern {
	return ErrorTestPattern{
		Name:           "NonExistentResource",
		Description:    "Test with non-existent resource ID",
		ExpectedStatus: http.StatusNotFound,
		Execute: func(t *testing.T, setupData interface{}) *HTTPTestRequest {
			return &HTTPTestRequest{
				Method:  "GET",
				Path:    urlPath,
				URLVars: map[string]string{"id": uuid.New().String()},
			}
		},
	}
}

// MalformedJSONPattern creates a standard malformed JSON test pattern
func MalformedJSONPattern(urlPath string) ErrorTestPattern {
	return ErrorTestPattern{
		Name:           "MalformedJSON",
		Description:    "Test with malformed JSON body",
		ExpectedStatus: http.StatusBadRequest,
		Execute: func(t *testing.T, setupData interface{}) *HTTPTestRequest {
			return &HTTPTestRequest{
				Method: "POST",
				Path:   urlPath,
				Body:   "{invalid json}",
			}
		},
	}
}

// EmptyBodyPattern creates a standard empty body test pattern
func EmptyBodyPattern(urlPath string) ErrorTestPattern {
	return ErrorTestPattern{
		Name:           "EmptyBody",
		Description:    "Test with empty request body",
		ExpectedStatus: http.StatusBadRequest,
		Execute: func(t *testing.T, setupData interface{}) *HTTPTestRequest {
			return &HTTPTestRequest{
				Method: "POST",
				Path:   urlPath,
				Body:   nil,
			}
		},
	}
}
