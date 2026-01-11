package main

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/google/uuid"
)

// ErrorTestPattern defines a systematic approach to testing error conditions
type ErrorTestPattern struct {
	Name           string
	Description    string
	ExpectedStatus int
	Setup          func(t *testing.T) interface{}
	Execute        func(t *testing.T, setupData interface{}) *HTTPTestRequest
	Validate       func(t *testing.T, w interface{}, setupData interface{})
	Cleanup        func(setupData interface{})
}

// HandlerTestSuite provides a comprehensive test framework for HTTP handlers
type HandlerTestSuite struct {
	HandlerName string
	Handler     http.HandlerFunc
	Server      *Server
	BaseURL     string
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

// AddInvalidUUID adds an invalid UUID test pattern
func (b *TestScenarioBuilder) AddInvalidUUID(urlPath string) *TestScenarioBuilder {
	pattern := ErrorTestPattern{
		Name:           "InvalidUUID",
		Description:    "Test handler with invalid UUID format",
		ExpectedStatus: http.StatusBadRequest,
		Setup:          nil,
		Execute: func(t *testing.T, setupData interface{}) *HTTPTestRequest {
			return &HTTPTestRequest{
				Method:  "GET",
				Path:    urlPath,
				URLVars: map[string]string{"build_id": "invalid-uuid-format"},
			}
		},
		Validate: func(t *testing.T, w interface{}, setupData interface{}) {
			// Additional validation can be added here
		},
	}
	b.patterns = append(b.patterns, pattern)
	return b
}

// AddNonExistentBuild adds a non-existent build ID test pattern
func (b *TestScenarioBuilder) AddNonExistentBuild(urlPath string) *TestScenarioBuilder {
	pattern := ErrorTestPattern{
		Name:           "NonExistentBuild",
		Description:    "Test handler with non-existent build ID",
		ExpectedStatus: http.StatusNotFound,
		Setup:          nil,
		Execute: func(t *testing.T, setupData interface{}) *HTTPTestRequest {
			nonExistentID := uuid.New()
			return &HTTPTestRequest{
				Method:  "GET",
				Path:    urlPath,
				URLVars: map[string]string{"build_id": nonExistentID.String()},
			}
		},
		Validate: func(t *testing.T, w interface{}, setupData interface{}) {
			// Validate error message contains appropriate information
		},
	}
	b.patterns = append(b.patterns, pattern)
	return b
}

// AddInvalidJSON adds an invalid JSON test pattern
func (b *TestScenarioBuilder) AddInvalidJSON(urlPath string, method string) *TestScenarioBuilder {
	pattern := ErrorTestPattern{
		Name:           "InvalidJSON",
		Description:    "Test handler with malformed JSON body",
		ExpectedStatus: http.StatusBadRequest,
		Setup:          nil,
		Execute: func(t *testing.T, setupData interface{}) *HTTPTestRequest {
			return &HTTPTestRequest{
				Method: method,
				Path:   urlPath,
				Body:   "this is not valid JSON {{{",
			}
		},
		Validate: func(t *testing.T, w interface{}, setupData interface{}) {
			// Validate error message mentions JSON parsing issue
		},
	}
	b.patterns = append(b.patterns, pattern)
	return b
}

// AddMissingRequiredField adds a missing required field test pattern
func (b *TestScenarioBuilder) AddMissingRequiredField(urlPath string, method string, fieldName string) *TestScenarioBuilder {
	pattern := ErrorTestPattern{
		Name:           fmt.Sprintf("MissingField_%s", fieldName),
		Description:    fmt.Sprintf("Test handler with missing required field: %s", fieldName),
		ExpectedStatus: http.StatusBadRequest,
		Setup:          nil,
		Execute: func(t *testing.T, setupData interface{}) *HTTPTestRequest {
			// Create incomplete request missing the specified field
			body := map[string]interface{}{}
			switch fieldName {
			case "app_name":
				body = map[string]interface{}{
					"framework":     "electron",
					"template_type": "basic",
					"output_path":   "/tmp/test",
				}
			case "framework":
				body = map[string]interface{}{
					"app_name":      "TestApp",
					"template_type": "basic",
					"output_path":   "/tmp/test",
				}
			case "template_type":
				body = map[string]interface{}{
					"app_name":    "TestApp",
					"framework":   "electron",
					"output_path": "/tmp/test",
				}
			case "output_path":
				body = map[string]interface{}{
					"app_name":      "TestApp",
					"framework":     "electron",
					"template_type": "basic",
				}
			}
			return &HTTPTestRequest{
				Method: method,
				Path:   urlPath,
				Body:   body,
			}
		},
		Validate: func(t *testing.T, w interface{}, setupData interface{}) {
			// Validate error message mentions missing field
		},
	}
	b.patterns = append(b.patterns, pattern)
	return b
}

// AddInvalidEnumValue adds an invalid enum value test pattern
func (b *TestScenarioBuilder) AddInvalidEnumValue(urlPath string, method string, fieldName string, invalidValue string) *TestScenarioBuilder {
	pattern := ErrorTestPattern{
		Name:           fmt.Sprintf("InvalidEnum_%s", fieldName),
		Description:    fmt.Sprintf("Test handler with invalid enum value for %s", fieldName),
		ExpectedStatus: http.StatusBadRequest,
		Setup:          nil,
		Execute: func(t *testing.T, setupData interface{}) *HTTPTestRequest {
			body := createValidGenerateRequest()
			body[fieldName] = invalidValue
			return &HTTPTestRequest{
				Method: method,
				Path:   urlPath,
				Body:   body,
			}
		},
		Validate: func(t *testing.T, w interface{}, setupData interface{}) {
			// Validate error message mentions invalid value
		},
	}
	b.patterns = append(b.patterns, pattern)
	return b
}

// AddEmptyBody adds an empty body test pattern
func (b *TestScenarioBuilder) AddEmptyBody(urlPath string, method string) *TestScenarioBuilder {
	pattern := ErrorTestPattern{
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
		Validate: func(t *testing.T, w interface{}, setupData interface{}) {
			// Validate appropriate error response
		},
	}
	b.patterns = append(b.patterns, pattern)
	return b
}

// AddNullFieldValue adds a null field value test pattern
func (b *TestScenarioBuilder) AddNullFieldValue(urlPath string, method string, fieldName string) *TestScenarioBuilder {
	pattern := ErrorTestPattern{
		Name:           fmt.Sprintf("NullField_%s", fieldName),
		Description:    fmt.Sprintf("Test handler with null value for %s", fieldName),
		ExpectedStatus: http.StatusBadRequest,
		Setup:          nil,
		Execute: func(t *testing.T, setupData interface{}) *HTTPTestRequest {
			body := createValidGenerateRequest()
			body[fieldName] = nil
			return &HTTPTestRequest{
				Method: method,
				Path:   urlPath,
				Body:   body,
			}
		},
		Validate: func(t *testing.T, w interface{}, setupData interface{}) {
			// Validate error response
		},
	}
	b.patterns = append(b.patterns, pattern)
	return b
}

// AddCustomPattern adds a custom error test pattern
func (b *TestScenarioBuilder) AddCustomPattern(pattern ErrorTestPattern) *TestScenarioBuilder {
	b.patterns = append(b.patterns, pattern)
	return b
}

// Build returns the built test patterns
func (b *TestScenarioBuilder) Build() []ErrorTestPattern {
	return b.patterns
}
