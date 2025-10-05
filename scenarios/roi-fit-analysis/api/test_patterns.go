package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

// ErrorTestPattern defines a systematic approach to testing error conditions
type ErrorTestPattern struct {
	Name           string
	Description    string
	Method         string
	Path           string
	Body           interface{}
	ExpectedStatus int
	ShouldContain  string
	Setup          func(t *testing.T) interface{}
	Validate       func(t *testing.T, recorder *httptest.ResponseRecorder, setupData interface{})
	Cleanup        func(setupData interface{})
}

// HandlerTestSuite provides a comprehensive test framework for HTTP handlers
type HandlerTestSuite struct {
	HandlerName string
	Handler     http.HandlerFunc
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

			// Create request
			var bodyBytes []byte
			var err error
			if pattern.Body != nil {
				bodyBytes, err = json.Marshal(pattern.Body)
				if err != nil {
					t.Fatalf("Failed to marshal request body: %v", err)
				}
			}

			req, err := http.NewRequest(pattern.Method, pattern.Path, bytes.NewBuffer(bodyBytes))
			if err != nil {
				t.Fatalf("Failed to create HTTP request: %v", err)
			}

			if pattern.Body != nil {
				req.Header.Set("Content-Type", "application/json")
			}

			recorder := httptest.NewRecorder()

			// Execute handler
			suite.Handler(recorder, req)

			// Validate status code
			if recorder.Code != pattern.ExpectedStatus {
				t.Errorf("Expected status %d, got %d. Body: %s", pattern.ExpectedStatus, recorder.Code, recorder.Body.String())
			}

			// Custom validation
			if pattern.Validate != nil {
				pattern.Validate(t, recorder, setupData)
			}
		})
	}
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

// AddInvalidJSON adds a test for invalid JSON input
func (b *TestScenarioBuilder) AddInvalidJSON(path string, method string) *TestScenarioBuilder {
	b.patterns = append(b.patterns, ErrorTestPattern{
		Name:           "InvalidJSON",
		Description:    "Test handler with invalid JSON input",
		Method:         method,
		Path:           path,
		Body:           nil, // Will send raw invalid JSON
		ExpectedStatus: http.StatusBadRequest,
	})
	return b
}

// AddMissingRequiredField adds a test for missing required fields
func (b *TestScenarioBuilder) AddMissingRequiredField(path string, method string, field string) *TestScenarioBuilder {
	b.patterns = append(b.patterns, ErrorTestPattern{
		Name:           fmt.Sprintf("Missing_%s", field),
		Description:    fmt.Sprintf("Test handler with missing %s field", field),
		Method:         method,
		Path:           path,
		Body:           map[string]interface{}{}, // Empty body
		ExpectedStatus: http.StatusBadRequest,
	})
	return b
}

// AddInvalidMethodTest adds a test for wrong HTTP method
func (b *TestScenarioBuilder) AddInvalidMethodTest(path string, invalidMethod string) *TestScenarioBuilder {
	b.patterns = append(b.patterns, ErrorTestPattern{
		Name:           fmt.Sprintf("Invalid_Method_%s", invalidMethod),
		Description:    fmt.Sprintf("Test handler with invalid HTTP method: %s", invalidMethod),
		Method:         invalidMethod,
		Path:           path,
		Body:           nil,
		ExpectedStatus: http.StatusMethodNotAllowed,
	})
	return b
}

// AddEmptyBodyTest adds a test for empty request body
func (b *TestScenarioBuilder) AddEmptyBodyTest(path string, method string) *TestScenarioBuilder {
	b.patterns = append(b.patterns, ErrorTestPattern{
		Name:           "EmptyBody",
		Description:    "Test handler with empty request body",
		Method:         method,
		Path:           path,
		Body:           nil,
		ExpectedStatus: http.StatusBadRequest,
	})
	return b
}

// AddInvalidValueTest adds a test for invalid field values
func (b *TestScenarioBuilder) AddInvalidValueTest(path string, method string, field string, value interface{}, reason string) *TestScenarioBuilder {
	body := map[string]interface{}{
		field: value,
	}

	b.patterns = append(b.patterns, ErrorTestPattern{
		Name:           fmt.Sprintf("Invalid_%s_%s", field, reason),
		Description:    fmt.Sprintf("Test handler with invalid %s: %s", field, reason),
		Method:         method,
		Path:           path,
		Body:           body,
		ExpectedStatus: http.StatusBadRequest,
	})
	return b
}

// AddDatabaseUnavailableTest adds a test for database unavailability
func (b *TestScenarioBuilder) AddDatabaseUnavailableTest(path string, method string) *TestScenarioBuilder {
	b.patterns = append(b.patterns, ErrorTestPattern{
		Name:           "DatabaseUnavailable",
		Description:    "Test handler when database is unavailable",
		Method:         method,
		Path:           path,
		Body:           createTestAnalysisRequest(),
		ExpectedStatus: http.StatusServiceUnavailable,
		Setup: func(t *testing.T) interface{} {
			// Save original db and set to nil
			originalDB := db
			db = nil
			return originalDB
		},
		Cleanup: func(setupData interface{}) {
			// Restore original db
			if originalDB, ok := setupData.(*sql.DB); ok {
				db = originalDB
			}
		},
	})
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

// Common pre-built test scenarios

// CreateAnalyzeEndpointErrorTests creates error tests for the /analyze endpoint
func CreateAnalyzeEndpointErrorTests() []ErrorTestPattern {
	return NewTestScenarioBuilder().
		AddInvalidMethodTest("/analyze", http.MethodGet).
		AddInvalidMethodTest("/analyze", http.MethodPut).
		AddInvalidMethodTest("/analyze", http.MethodDelete).
		AddEmptyBodyTest("/analyze", http.MethodPost).
		AddMissingRequiredField("/analyze", http.MethodPost, "idea").
		AddMissingRequiredField("/analyze", http.MethodPost, "budget").
		AddMissingRequiredField("/analyze", http.MethodPost, "timeline").
		AddInvalidValueTest("/analyze", http.MethodPost, "budget", -1000, "negative").
		AddInvalidValueTest("/analyze", http.MethodPost, "budget", 0, "zero").
		AddInvalidValueTest("/analyze", http.MethodPost, "idea", "", "empty").
		Build()
}

// CreateComprehensiveAnalysisErrorTests creates error tests for /comprehensive-analysis
func CreateComprehensiveAnalysisErrorTests() []ErrorTestPattern {
	return NewTestScenarioBuilder().
		AddInvalidMethodTest("/comprehensive-analysis", http.MethodGet).
		AddInvalidMethodTest("/comprehensive-analysis", http.MethodPut).
		AddInvalidMethodTest("/comprehensive-analysis", http.MethodDelete).
		AddEmptyBodyTest("/comprehensive-analysis", http.MethodPost).
		AddMissingRequiredField("/comprehensive-analysis", http.MethodPost, "idea").
		AddMissingRequiredField("/comprehensive-analysis", http.MethodPost, "budget").
		AddInvalidValueTest("/comprehensive-analysis", http.MethodPost, "budget", -5000, "negative").
		AddInvalidValueTest("/comprehensive-analysis", http.MethodPost, "timeline", "", "empty").
		Build()
}

// CreateOpportunitiesEndpointErrorTests creates error tests for /opportunities
func CreateOpportunitiesEndpointErrorTests() []ErrorTestPattern {
	return NewTestScenarioBuilder().
		AddInvalidMethodTest("/opportunities", http.MethodPost).
		AddInvalidMethodTest("/opportunities", http.MethodPut).
		AddInvalidMethodTest("/opportunities", http.MethodDelete).
		Build()
}

// CreateReportsEndpointErrorTests creates error tests for /reports
func CreateReportsEndpointErrorTests() []ErrorTestPattern {
	return NewTestScenarioBuilder().
		AddInvalidMethodTest("/reports", http.MethodPost).
		AddInvalidMethodTest("/reports", http.MethodPut).
		AddInvalidMethodTest("/reports", http.MethodDelete).
		Build()
}

// CreateHealthEndpointErrorTests creates error tests for /health
func CreateHealthEndpointErrorTests() []ErrorTestPattern {
	return NewTestScenarioBuilder().
		AddInvalidMethodTest("/health", http.MethodPost).
		AddInvalidMethodTest("/health", http.MethodPut).
		AddInvalidMethodTest("/health", http.MethodDelete).
		Build()
}

// CreateAnalysisResultsErrorTests creates error tests for /analysis/results
func CreateAnalysisResultsErrorTests() []ErrorTestPattern {
	return NewTestScenarioBuilder().
		AddInvalidMethodTest("/analysis/results", http.MethodPost).
		AddInvalidMethodTest("/analysis/results", http.MethodPut).
		AddInvalidMethodTest("/analysis/results", http.MethodDelete).
		AddDatabaseUnavailableTest("/analysis/results", http.MethodGet).
		Build()
}

// EdgeCaseScenarios provides edge case test scenarios
type EdgeCaseScenarios struct {
	patterns []ErrorTestPattern
}

// NewEdgeCaseScenarios creates edge case test scenarios
func NewEdgeCaseScenarios() *EdgeCaseScenarios {
	return &EdgeCaseScenarios{
		patterns: []ErrorTestPattern{},
	}
}

// AddVeryLargeBudgetTest tests with extremely large budget
func (e *EdgeCaseScenarios) AddVeryLargeBudgetTest(path string) *EdgeCaseScenarios {
	req := createTestAnalysisRequest()
	req.Budget = 999999999999.99

	e.patterns = append(e.patterns, ErrorTestPattern{
		Name:           "VeryLargeBudget",
		Description:    "Test with extremely large budget",
		Method:         http.MethodPost,
		Path:           path,
		Body:           req,
		ExpectedStatus: http.StatusOK,
	})
	return e
}

// AddVerySmallBudgetTest tests with very small budget
func (e *EdgeCaseScenarios) AddVerySmallBudgetTest(path string) *EdgeCaseScenarios {
	req := createTestAnalysisRequest()
	req.Budget = 0.01

	e.patterns = append(e.patterns, ErrorTestPattern{
		Name:           "VerySmallBudget",
		Description:    "Test with very small budget",
		Method:         http.MethodPost,
		Path:           path,
		Body:           req,
		ExpectedStatus: http.StatusOK,
	})
	return e
}

// AddLongIdeaTextTest tests with very long idea description
func (e *EdgeCaseScenarios) AddLongIdeaTextTest(path string) *EdgeCaseScenarios {
	req := createTestAnalysisRequest()
	req.Idea = string(make([]byte, 10000)) // 10KB of text

	e.patterns = append(e.patterns, ErrorTestPattern{
		Name:           "LongIdeaText",
		Description:    "Test with very long idea description",
		Method:         http.MethodPost,
		Path:           path,
		Body:           req,
		ExpectedStatus: http.StatusOK,
	})
	return e
}

// AddManySkillsTest tests with many skills
func (e *EdgeCaseScenarios) AddManySkillsTest(path string) *EdgeCaseScenarios {
	req := createTestAnalysisRequest()
	req.Skills = make([]string, 100)
	for i := 0; i < 100; i++ {
		req.Skills[i] = fmt.Sprintf("Skill_%d", i)
	}

	e.patterns = append(e.patterns, ErrorTestPattern{
		Name:           "ManySkills",
		Description:    "Test with many skills",
		Method:         http.MethodPost,
		Path:           path,
		Body:           req,
		ExpectedStatus: http.StatusOK,
	})
	return e
}

// AddEmptySkillsTest tests with no skills
func (e *EdgeCaseScenarios) AddEmptySkillsTest(path string) *EdgeCaseScenarios {
	req := createTestAnalysisRequest()
	req.Skills = []string{}

	e.patterns = append(e.patterns, ErrorTestPattern{
		Name:           "EmptySkills",
		Description:    "Test with empty skills array",
		Method:         http.MethodPost,
		Path:           path,
		Body:           req,
		ExpectedStatus: http.StatusOK,
	})
	return e
}

// AddSpecialCharactersTest tests with special characters in inputs
func (e *EdgeCaseScenarios) AddSpecialCharactersTest(path string) *EdgeCaseScenarios {
	req := createTestAnalysisRequest()
	req.Idea = "Test with special chars: <script>alert('xss')</script> & @#$%^&*()"
	req.MarketFocus = "Test & Special <> Characters"

	e.patterns = append(e.patterns, ErrorTestPattern{
		Name:           "SpecialCharacters",
		Description:    "Test with special characters in input",
		Method:         http.MethodPost,
		Path:           path,
		Body:           req,
		ExpectedStatus: http.StatusOK,
	})
	return e
}

// Build returns the edge case patterns
func (e *EdgeCaseScenarios) Build() []ErrorTestPattern {
	return e.patterns
}
