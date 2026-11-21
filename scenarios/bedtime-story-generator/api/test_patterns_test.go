package main

import (
	"database/sql"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
)

// ErrorTestPattern defines a systematic approach to testing error conditions
type ErrorTestPattern struct {
	Name           string
	Description    string
	ExpectedStatus int
	Setup          func(t *testing.T) interface{}
	Execute        func(t *testing.T, setupData interface{}) *HTTPTestRequest
	Validate       func(t *testing.T, req *HTTPTestRequest, setupData interface{})
	Cleanup        func(setupData interface{})
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
			w, httpReq, err := makeHTTPRequest(*req)
			if err != nil {
				t.Fatalf("Failed to create HTTP request: %v", err)
			}

			suite.Handler(w, httpReq)

			// Validate status code
			if w.Code != pattern.ExpectedStatus {
				t.Errorf("Expected status %d, got %d. Response: %s",
					pattern.ExpectedStatus, w.Code, w.Body.String())
			}

			// Additional validation
			if pattern.Validate != nil {
				pattern.Validate(t, req, setupData)
			}
		})
	}
}

// Common error patterns that can be reused across handlers

// invalidUUIDPattern tests handlers with invalid UUID formats
func invalidUUIDPattern(urlPath string, method string) ErrorTestPattern {
	return ErrorTestPattern{
		Name:           "InvalidUUID",
		Description:    "Test handler with invalid UUID format",
		ExpectedStatus: http.StatusBadRequest,
		Setup:          nil,
		Execute: func(t *testing.T, setupData interface{}) *HTTPTestRequest {
			return &HTTPTestRequest{
				Method:  method,
				Path:    urlPath,
				URLVars: map[string]string{"id": "invalid-uuid-format"},
			}
		},
		Validate: nil,
	}
}

// nonExistentStoryPattern tests handlers with non-existent story IDs
func nonExistentStoryPattern(urlPath string, method string) ErrorTestPattern {
	return ErrorTestPattern{
		Name:           "NonExistentStory",
		Description:    "Test handler with non-existent story ID",
		ExpectedStatus: http.StatusNotFound,
		Setup: func(t *testing.T) interface{} {
			// Setup mock database
			testDB := setupTestDatabase(t)
			return testDB
		},
		Execute: func(t *testing.T, setupData interface{}) *HTTPTestRequest {
			nonExistentID := uuid.New()
			return &HTTPTestRequest{
				Method:  method,
				Path:    urlPath,
				URLVars: map[string]string{"id": nonExistentID.String()},
			}
		},
		Cleanup: func(setupData interface{}) {
			if testDB, ok := setupData.(*TestDatabase); ok {
				testDB.Cleanup()
			}
		},
	}
}

// invalidJSONPattern tests handlers with malformed JSON
func invalidJSONPattern(urlPath string) ErrorTestPattern {
	return ErrorTestPattern{
		Name:           "InvalidJSON",
		Description:    "Test handler with malformed JSON input",
		ExpectedStatus: http.StatusBadRequest,
		Setup:          nil,
		Execute: func(t *testing.T, setupData interface{}) *HTTPTestRequest {
			return &HTTPTestRequest{
				Method: "POST",
				Path:   urlPath,
				Body:   `{"invalid": "json"`, // Malformed JSON
			}
		},
	}
}

// emptyRequestPattern tests handlers with empty request body
func emptyRequestPattern(urlPath string) ErrorTestPattern {
	return ErrorTestPattern{
		Name:           "EmptyRequest",
		Description:    "Test handler with empty request body",
		ExpectedStatus: http.StatusBadRequest,
		Setup:          nil,
		Execute: func(t *testing.T, setupData interface{}) *HTTPTestRequest {
			return &HTTPTestRequest{
				Method: "POST",
				Path:   urlPath,
				Body:   "",
			}
		},
	}
}

// invalidAgeGroupPattern tests handlers with invalid age group
func invalidAgeGroupPattern(urlPath string) ErrorTestPattern {
	return ErrorTestPattern{
		Name:           "InvalidAgeGroup",
		Description:    "Test handler with invalid age group",
		ExpectedStatus: http.StatusBadRequest,
		Setup:          nil,
		Execute: func(t *testing.T, setupData interface{}) *HTTPTestRequest {
			req := GenerateStoryRequest{
				AgeGroup:       "invalid-age-group",
				Theme:          "adventure",
				Length:         "medium",
				CharacterNames: []string{"Test"},
			}
			return &HTTPTestRequest{
				Method: "POST",
				Path:   urlPath,
				Body:   req,
			}
		},
	}
}

// invalidThemePattern tests handlers with invalid theme
func invalidThemePattern(urlPath string) ErrorTestPattern {
	return ErrorTestPattern{
		Name:           "InvalidTheme",
		Description:    "Test handler with invalid theme",
		ExpectedStatus: http.StatusBadRequest,
		Setup:          nil,
		Execute: func(t *testing.T, setupData interface{}) *HTTPTestRequest {
			req := GenerateStoryRequest{
				AgeGroup:       "6-8",
				Theme:          "invalid-theme",
				Length:         "medium",
				CharacterNames: []string{"Test"},
			}
			return &HTTPTestRequest{
				Method: "POST",
				Path:   urlPath,
				Body:   req,
			}
		},
	}
}

// PerformanceTestPattern defines performance testing scenarios
type PerformanceTestPattern struct {
	Name           string
	Description    string
	MaxDuration    time.Duration
	Iterations     int
	Setup          func(t *testing.T) interface{}
	Execute        func(t *testing.T, setupData interface{}) time.Duration
	Cleanup        func(setupData interface{})
}

// ConcurrencyTestPattern defines concurrency testing scenarios
type ConcurrencyTestPattern struct {
	Name           string
	Description    string
	Concurrency    int
	Iterations     int
	Setup          func(t *testing.T) interface{}
	Execute        func(t *testing.T, setupData interface{}, iteration int) error
	Validate       func(t *testing.T, setupData interface{}, results []error)
	Cleanup        func(setupData interface{})
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
func (b *TestScenarioBuilder) AddInvalidUUID(urlPath string, method string) *TestScenarioBuilder {
	b.patterns = append(b.patterns, invalidUUIDPattern(urlPath, method))
	return b
}

// AddNonExistentStory adds non-existent story test pattern
func (b *TestScenarioBuilder) AddNonExistentStory(urlPath string, method string) *TestScenarioBuilder {
	b.patterns = append(b.patterns, nonExistentStoryPattern(urlPath, method))
	return b
}

// AddInvalidJSON adds invalid JSON test pattern
func (b *TestScenarioBuilder) AddInvalidJSON(urlPath string) *TestScenarioBuilder {
	b.patterns = append(b.patterns, invalidJSONPattern(urlPath))
	return b
}

// AddEmptyRequest adds empty request test pattern
func (b *TestScenarioBuilder) AddEmptyRequest(urlPath string) *TestScenarioBuilder {
	b.patterns = append(b.patterns, emptyRequestPattern(urlPath))
	return b
}

// AddInvalidAgeGroup adds invalid age group test pattern
func (b *TestScenarioBuilder) AddInvalidAgeGroup(urlPath string) *TestScenarioBuilder {
	b.patterns = append(b.patterns, invalidAgeGroupPattern(urlPath))
	return b
}

// AddInvalidTheme adds invalid theme test pattern
func (b *TestScenarioBuilder) AddInvalidTheme(urlPath string) *TestScenarioBuilder {
	b.patterns = append(b.patterns, invalidThemePattern(urlPath))
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

// DatabaseErrorPattern tests database connection failures
func DatabaseErrorPattern(urlPath string, method string, handler http.HandlerFunc) ErrorTestPattern {
	return ErrorTestPattern{
		Name:           "DatabaseError",
		Description:    "Test handler with database connection failure",
		ExpectedStatus: http.StatusInternalServerError,
		Setup: func(t *testing.T) interface{} {
			// Close the database to simulate connection error
			oldDB := db
			db = nil
			return oldDB
		},
		Execute: func(t *testing.T, setupData interface{}) *HTTPTestRequest {
			return &HTTPTestRequest{
				Method: method,
				Path:   urlPath,
			}
		},
		Cleanup: func(setupData interface{}) {
			if oldDB, ok := setupData.(*sql.DB); ok {
				db = oldDB
			}
		},
	}
}

// Cache test patterns

// CacheHitPattern tests cache hit scenario
func CacheHitPattern(t *testing.T) ErrorTestPattern {
	return ErrorTestPattern{
		Name:           "CacheHit",
		Description:    "Test story retrieval from cache",
		ExpectedStatus: http.StatusOK,
		Setup: func(t *testing.T) interface{} {
			story := setupTestStory(t, "Cached Story", "6-8")
			storyCache.Set(story.Story.ID, story.Story)
			return story
		},
		Execute: func(t *testing.T, setupData interface{}) *HTTPTestRequest {
			story := setupData.(*TestStory)
			return &HTTPTestRequest{
				Method:  "GET",
				Path:    "/api/v1/stories/" + story.Story.ID,
				URLVars: map[string]string{"id": story.Story.ID},
			}
		},
		Cleanup: func(setupData interface{}) {
			if story, ok := setupData.(*TestStory); ok {
				story.Cleanup()
			}
		},
	}
}

// Template for comprehensive handler testing
func TemplateComprehensiveHandlerTest(t *testing.T, handlerName string, handler http.HandlerFunc) {
	t.Run(fmt.Sprintf("%s_Comprehensive", handlerName), func(t *testing.T) {
		// 1. Setup
		loggerCleanup := setupTestLogger()
		defer loggerCleanup()

		env := setupTestDirectory(t)
		defer env.Cleanup()

		// 2. Test successful scenarios
		t.Run("Success_Cases", func(t *testing.T) {
			// Add success test cases here
		})

		// 3. Test error conditions
		t.Run("Error_Cases", func(t *testing.T) {
			// Add error test cases here
		})

		// 4. Test edge cases
		t.Run("Edge_Cases", func(t *testing.T) {
			// Add edge case tests here
		})

		// 5. Test performance (if needed)
		t.Run("Performance", func(t *testing.T) {
			// Add performance tests here
		})
	})
}
