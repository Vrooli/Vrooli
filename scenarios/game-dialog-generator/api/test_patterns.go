package main

import (
	"fmt"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// ErrorTestPattern defines a systematic approach to testing error conditions
type ErrorTestPattern struct {
	Name           string
	Description    string
	ExpectedStatus int
	Setup          func(t *testing.T, router *gin.Engine) interface{}
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
				setupData = pattern.Setup(t, suite.Router)
			}

			// Cleanup
			if pattern.Cleanup != nil {
				defer pattern.Cleanup(setupData)
			}

			// Execute
			w := pattern.Execute(t, suite.Router, setupData)

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

// Common error patterns that can be reused across handlers

// invalidUUIDPattern tests handlers with invalid UUID formats
func invalidUUIDPattern(method, urlPath string) ErrorTestPattern {
	return ErrorTestPattern{
		Name:           "InvalidUUID",
		Description:    "Test handler with invalid UUID format",
		ExpectedStatus: 400, // BadRequest or similar
		Setup:          nil,
		Execute: func(t *testing.T, router *gin.Engine, setupData interface{}) *httptest.ResponseRecorder {
			req := HTTPTestRequest{
				Method: method,
				Path:   urlPath,
			}
			return makeHTTPRequest(router, req)
		},
	}
}

// nonExistentCharacterPattern tests handlers with non-existent character IDs
func nonExistentCharacterPattern(method, urlPathTemplate string) ErrorTestPattern {
	return ErrorTestPattern{
		Name:           "NonExistentCharacter",
		Description:    "Test handler with non-existent character ID",
		ExpectedStatus: 404,
		Setup:          nil,
		Execute: func(t *testing.T, router *gin.Engine, setupData interface{}) *httptest.ResponseRecorder {
			nonExistentID := uuid.New()
			path := fmt.Sprintf(urlPathTemplate, nonExistentID.String())
			req := HTTPTestRequest{
				Method: method,
				Path:   path,
			}
			return makeHTTPRequest(router, req)
		},
	}
}

// invalidJSONPattern tests handlers with malformed JSON
func invalidJSONPattern(method, urlPath string) ErrorTestPattern {
	return ErrorTestPattern{
		Name:           "InvalidJSON",
		Description:    "Test handler with malformed JSON input",
		ExpectedStatus: 400,
		Setup:          nil,
		Execute: func(t *testing.T, router *gin.Engine, setupData interface{}) *httptest.ResponseRecorder {
			req := HTTPTestRequest{
				Method: method,
				Path:   urlPath,
				Body:   `{"invalid": "json"`, // Malformed JSON
			}
			return makeHTTPRequest(router, req)
		},
	}
}

// missingRequiredFieldsPattern tests handlers with missing required fields
func missingRequiredFieldsPattern(method, urlPath string, incompleteBody interface{}) ErrorTestPattern {
	return ErrorTestPattern{
		Name:           "MissingRequiredFields",
		Description:    "Test handler with missing required fields",
		ExpectedStatus: 400,
		Setup:          nil,
		Execute: func(t *testing.T, router *gin.Engine, setupData interface{}) *httptest.ResponseRecorder {
			req := HTTPTestRequest{
				Method: method,
				Path:   urlPath,
				Body:   incompleteBody,
			}
			return makeHTTPRequest(router, req)
		},
	}
}

// PerformanceTestPattern defines performance testing scenarios
type PerformanceTestPattern struct {
	Name           string
	Description    string
	MaxDuration    time.Duration
	Setup          func(t *testing.T) interface{}
	Execute        func(t *testing.T, setupData interface{}) time.Duration
	Validate       func(t *testing.T, duration time.Duration, setupData interface{})
	Cleanup        func(setupData interface{})
}

// RunPerformanceTest executes a performance test pattern
func RunPerformanceTest(t *testing.T, pattern PerformanceTestPattern) {
	t.Run(pattern.Name, func(t *testing.T) {
		// Setup
		var setupData interface{}
		if pattern.Setup != nil {
			setupData = pattern.Setup(t)
		}

		// Cleanup
		if pattern.Cleanup != nil {
			defer pattern.Cleanup(setupData)
		}

		// Execute and measure
		duration := pattern.Execute(t, setupData)

		// Validate performance
		if duration > pattern.MaxDuration {
			t.Errorf("Performance test '%s' exceeded max duration: %v > %v",
				pattern.Name, duration, pattern.MaxDuration)
		}

		// Custom validation
		if pattern.Validate != nil {
			pattern.Validate(t, duration, setupData)
		}
	})
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

// AddInvalidUUID adds invalid UUID test pattern
func (b *TestScenarioBuilder) AddInvalidUUID(method, urlPath string) *TestScenarioBuilder {
	b.patterns = append(b.patterns, invalidUUIDPattern(method, urlPath))
	return b
}

// AddNonExistentCharacter adds non-existent character test pattern
func (b *TestScenarioBuilder) AddNonExistentCharacter(method, urlPathTemplate string) *TestScenarioBuilder {
	b.patterns = append(b.patterns, nonExistentCharacterPattern(method, urlPathTemplate))
	return b
}

// AddInvalidJSON adds invalid JSON test pattern
func (b *TestScenarioBuilder) AddInvalidJSON(method, urlPath string) *TestScenarioBuilder {
	b.patterns = append(b.patterns, invalidJSONPattern(method, urlPath))
	return b
}

// AddMissingRequiredFields adds missing fields test pattern
func (b *TestScenarioBuilder) AddMissingRequiredFields(method, urlPath string, incompleteBody interface{}) *TestScenarioBuilder {
	b.patterns = append(b.patterns, missingRequiredFieldsPattern(method, urlPath, incompleteBody))
	return b
}

// AddCustom adds a custom test pattern
func (b *TestScenarioBuilder) AddCustom(pattern ErrorTestPattern) *TestScenarioBuilder {
	b.patterns = append(b.patterns, pattern)
	return b
}

// Build returns the configured test patterns
func (b *TestScenarioBuilder) Build() []ErrorTestPattern {
	return b.patterns
}

// Template for comprehensive handler testing
func TemplateComprehensiveHandlerTest(t *testing.T, handlerName string, router *gin.Engine, testCases func(*testing.T, *gin.Engine)) {
	t.Run(fmt.Sprintf("%s_Comprehensive", handlerName), func(t *testing.T) {
		// 1. Setup
		cleanup := setupTestLogger()
		defer cleanup()

		// 2. Run test cases
		testCases(t, router)
	})
}

// BatchTestPattern for testing batch operations
type BatchTestPattern struct {
	Name          string
	Description   string
	BatchSize     int
	Setup         func(t *testing.T) interface{}
	ExecuteBatch  func(t *testing.T, setupData interface{}, batchSize int) error
	Validate      func(t *testing.T, setupData interface{}, results interface{})
	Cleanup       func(setupData interface{})
}

// RunBatchTest executes a batch test pattern
func RunBatchTest(t *testing.T, pattern BatchTestPattern) {
	t.Run(pattern.Name, func(t *testing.T) {
		// Setup
		var setupData interface{}
		if pattern.Setup != nil {
			setupData = pattern.Setup(t)
		}

		// Cleanup
		if pattern.Cleanup != nil {
			defer pattern.Cleanup(setupData)
		}

		// Execute batch
		err := pattern.ExecuteBatch(t, setupData, pattern.BatchSize)
		if err != nil {
			t.Errorf("Batch test '%s' failed: %v", pattern.Name, err)
		}

		// Validate
		if pattern.Validate != nil {
			pattern.Validate(t, setupData, nil)
		}
	})
}

// Edge case test patterns

// emptyBodyPattern tests handlers with empty request body
func emptyBodyPattern(method, urlPath string) ErrorTestPattern {
	return ErrorTestPattern{
		Name:           "EmptyBody",
		Description:    "Test handler with empty request body",
		ExpectedStatus: 400,
		Setup:          nil,
		Execute: func(t *testing.T, router *gin.Engine, setupData interface{}) *httptest.ResponseRecorder {
			req := HTTPTestRequest{
				Method: method,
				Path:   urlPath,
				Body:   "",
			}
			return makeHTTPRequest(router, req)
		},
	}
}

// nullFieldsPattern tests handlers with null values in fields
func nullFieldsPattern(method, urlPath string, bodyWithNulls map[string]interface{}) ErrorTestPattern {
	return ErrorTestPattern{
		Name:           "NullFields",
		Description:    "Test handler with null field values",
		ExpectedStatus: 400,
		Setup:          nil,
		Execute: func(t *testing.T, router *gin.Engine, setupData interface{}) *httptest.ResponseRecorder {
			req := HTTPTestRequest{
				Method: method,
				Path:   urlPath,
				Body:   bodyWithNulls,
			}
			return makeHTTPRequest(router, req)
		},
	}
}

// Example usage pattern for dialog generation testing
func DialogGenerationTestPatterns(characterID string) *TestScenarioBuilder {
	return NewTestScenarioBuilder().
		AddInvalidJSON("POST", "/api/v1/dialog/generate").
		AddMissingRequiredFields("POST", "/api/v1/dialog/generate", map[string]interface{}{
			"scene_context": "A jungle clearing",
			// Missing character_id
		}).
		AddCustom(ErrorTestPattern{
			Name:           "InvalidCharacterID",
			Description:    "Test with invalid character ID format",
			ExpectedStatus: 400,
			Execute: func(t *testing.T, router *gin.Engine, setupData interface{}) *httptest.ResponseRecorder {
				req := HTTPTestRequest{
					Method: "POST",
					Path:   "/api/v1/dialog/generate",
					Body: map[string]interface{}{
						"character_id":  "not-a-uuid",
						"scene_context": "Test scene",
					},
				}
				return makeHTTPRequest(router, req)
			},
		})
}
