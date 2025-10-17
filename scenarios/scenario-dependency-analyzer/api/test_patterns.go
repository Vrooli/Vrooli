
package main

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
)

// TestScenarioBuilder provides a fluent interface for building test scenarios
type TestScenarioBuilder struct {
	patterns []ErrorTestPattern
}

// ErrorTestPattern represents a systematic error test case
type ErrorTestPattern struct {
	Name           string
	Method         string
	Path           string
	Body           interface{}
	ExpectedStatus int
	ErrorContains  string
}

// NewTestScenarioBuilder creates a new test scenario builder
func NewTestScenarioBuilder() *TestScenarioBuilder {
	return &TestScenarioBuilder{
		patterns: []ErrorTestPattern{},
	}
}

// AddInvalidScenario adds a test for invalid scenario name
func (b *TestScenarioBuilder) AddInvalidScenario(path string) *TestScenarioBuilder {
	b.patterns = append(b.patterns, ErrorTestPattern{
		Name:           "Invalid scenario name",
		Method:         "GET",
		Path:           path,
		ExpectedStatus: http.StatusNotFound,
		ErrorContains:  "",
	})
	return b
}

// AddNonExistentScenario adds a test for non-existent scenario
func (b *TestScenarioBuilder) AddNonExistentScenario(path string) *TestScenarioBuilder {
	b.patterns = append(b.patterns, ErrorTestPattern{
		Name:           "Non-existent scenario",
		Method:         "GET",
		Path:           path,
		ExpectedStatus: http.StatusNotFound,
		ErrorContains:  "not found",
	})
	return b
}

// AddInvalidJSON adds a test for malformed JSON request
func (b *TestScenarioBuilder) AddInvalidJSON(path string) *TestScenarioBuilder {
	b.patterns = append(b.patterns, ErrorTestPattern{
		Name:           "Invalid JSON body",
		Method:         "POST",
		Path:           path,
		Body:           "invalid-json",
		ExpectedStatus: http.StatusBadRequest,
		ErrorContains:  "",
	})
	return b
}

// AddMissingRequiredField adds a test for missing required field
func (b *TestScenarioBuilder) AddMissingRequiredField(path string, fieldName string) *TestScenarioBuilder {
	b.patterns = append(b.patterns, ErrorTestPattern{
		Name:           "Missing required field: " + fieldName,
		Method:         "POST",
		Path:           path,
		Body:           map[string]interface{}{},
		ExpectedStatus: http.StatusBadRequest,
		ErrorContains:  fieldName,
	})
	return b
}

// AddInvalidGraphType adds a test for invalid graph type
func (b *TestScenarioBuilder) AddInvalidGraphType(basePath string, invalidType string) *TestScenarioBuilder {
	b.patterns = append(b.patterns, ErrorTestPattern{
		Name:           "Invalid graph type: " + invalidType,
		Method:         "GET",
		Path:           basePath + "/" + invalidType,
		ExpectedStatus: http.StatusBadRequest,
		ErrorContains:  "Invalid graph type",
	})
	return b
}

// Build returns all configured test patterns
func (b *TestScenarioBuilder) Build() []ErrorTestPattern {
	return b.patterns
}

// HandlerTestSuite provides comprehensive HTTP handler testing
type HandlerTestSuite struct {
	Router *gin.Engine
	T      *testing.T
}

// NewHandlerTestSuite creates a new handler test suite
func NewHandlerTestSuite(t *testing.T, router *gin.Engine) *HandlerTestSuite {
	return &HandlerTestSuite{
		Router: router,
		T:      t,
	}
}

// TestSuccessPath tests the happy path scenario
func (s *HandlerTestSuite) TestSuccessPath(method, path string, body interface{}, expectedStatus int, validator func(*testing.T, map[string]interface{})) {
	s.T.Run("SuccessPath", func(t *testing.T) {
		recorder := makeHTTPRequest(t, s.Router, method, path, body)

		if recorder.Code != expectedStatus {
			t.Errorf("Expected status %d, got %d. Body: %s", expectedStatus, recorder.Code, recorder.Body.String())
			return
		}

		if validator != nil {
			var response map[string]interface{}
			if err := json.Unmarshal(recorder.Body.Bytes(), &response); err == nil {
				validator(t, response)
			}
		}
	})
}

// TestErrorPatterns tests all configured error patterns
func (s *HandlerTestSuite) TestErrorPatterns(patterns []ErrorTestPattern) {
	for _, pattern := range patterns {
		s.T.Run(pattern.Name, func(t *testing.T) {
			recorder := makeHTTPRequest(t, s.Router, pattern.Method, pattern.Path, pattern.Body)

			if recorder.Code != pattern.ExpectedStatus {
				t.Errorf("Expected status %d, got %d. Body: %s",
					pattern.ExpectedStatus, recorder.Code, recorder.Body.String())
			}

			if pattern.ErrorContains != "" {
				assertErrorResponse(t, recorder, pattern.ExpectedStatus, pattern.ErrorContains)
			}
		})
	}
}

// TestEdgeCases tests boundary and edge cases
func (s *HandlerTestSuite) TestEdgeCases(cases map[string]struct {
	Method         string
	Path           string
	Body           interface{}
	ExpectedStatus int
	Description    string
}) {
	for name, tc := range cases {
		s.T.Run(name, func(t *testing.T) {
			recorder := makeHTTPRequest(t, s.Router, tc.Method, tc.Path, tc.Body)

			if recorder.Code != tc.ExpectedStatus {
				t.Errorf("%s: Expected status %d, got %d. Body: %s",
					tc.Description, tc.ExpectedStatus, recorder.Code, recorder.Body.String())
			}
		})
	}
}

// DatabaseTestPattern provides database operation testing patterns
type DatabaseTestPattern struct {
	Name          string
	Setup         func(*testing.T)
	Operation     func(*testing.T) error
	Verification  func(*testing.T, error)
	Cleanup       func(*testing.T)
}

// RunDatabaseTests executes a series of database test patterns
func RunDatabaseTests(t *testing.T, patterns []DatabaseTestPattern) {
	for _, pattern := range patterns {
		t.Run(pattern.Name, func(t *testing.T) {
			if pattern.Setup != nil {
				pattern.Setup(t)
			}

			err := pattern.Operation(t)

			if pattern.Verification != nil {
				pattern.Verification(t, err)
			}

			if pattern.Cleanup != nil {
				pattern.Cleanup(t)
			}
		})
	}
}

// AnalysisTestScenarios provides pre-built test scenarios for dependency analysis
func AnalysisTestScenarios() map[string]struct {
	Resources map[string]Resource
	Expected  struct {
		ResourceCount int
		ScenarioCount int
		WorkflowCount int
	}
} {
	return map[string]struct {
		Resources map[string]Resource
		Expected  struct {
			ResourceCount int
			ScenarioCount int
			WorkflowCount int
		}
	}{
		"simple-postgres": {
			Resources: map[string]Resource{
				"postgres": {
					Type:     "postgres",
					Enabled:  true,
					Required: true,
					Purpose:  "Primary database",
				},
			},
			Expected: struct {
				ResourceCount int
				ScenarioCount int
				WorkflowCount int
			}{
				ResourceCount: 1,
				ScenarioCount: 0,
				WorkflowCount: 0,
			},
		},
		"multi-resource": {
			Resources: map[string]Resource{
				"postgres": {
					Type:     "postgres",
					Enabled:  true,
					Required: true,
					Purpose:  "Primary database",
				},
				"redis": {
					Type:     "redis",
					Enabled:  true,
					Required: false,
					Purpose:  "Caching layer",
				},
				"ollama": {
					Type:     "ollama",
					Enabled:  true,
					Required: false,
					Purpose:  "AI inference",
				},
			},
			Expected: struct {
				ResourceCount int
				ScenarioCount int
				WorkflowCount int
			}{
				ResourceCount: 3,
				ScenarioCount: 0,
				WorkflowCount: 0,
			},
		},
		"empty-scenario": {
			Resources: map[string]Resource{},
			Expected: struct {
				ResourceCount int
				ScenarioCount int
				WorkflowCount int
			}{
				ResourceCount: 0,
				ScenarioCount: 0,
				WorkflowCount: 0,
			},
		},
	}
}
