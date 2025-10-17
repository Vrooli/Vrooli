// +build testing

package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// TestScenario represents a single test scenario configuration
type TestScenario struct {
	Name           string
	Description    string
	Method         string
	Path           string
	Body           interface{}
	Headers        map[string]string
	ExpectedStatus int
	ValidateBody   func(map[string]interface{}) bool
	Setup          func(*testing.T) interface{}
	Cleanup        func(interface{})
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

// AddInvalidUUID adds a test scenario for invalid UUID handling
func (b *TestScenarioBuilder) AddInvalidUUID(path string, method string) *TestScenarioBuilder {
	b.scenarios = append(b.scenarios, TestScenario{
		Name:           "InvalidUUID",
		Description:    "Test endpoint with invalid UUID format",
		Method:         method,
		Path:           path,
		Body:           nil,
		ExpectedStatus: http.StatusBadRequest,
		ValidateBody: func(body map[string]interface{}) bool {
			_, hasError := body["error"]
			return hasError
		},
	})
	return b
}

// AddNonExistentCampaign adds a test scenario for non-existent campaign
func (b *TestScenarioBuilder) AddNonExistentCampaign(pathTemplate string, method string) *TestScenarioBuilder {
	nonExistentID := uuid.New().String()
	path := fmt.Sprintf(pathTemplate, nonExistentID)

	b.scenarios = append(b.scenarios, TestScenario{
		Name:           "NonExistentCampaign",
		Description:    "Test endpoint with non-existent campaign ID",
		Method:         method,
		Path:           path,
		Body:           nil,
		ExpectedStatus: http.StatusNotFound,
		ValidateBody: func(body map[string]interface{}) bool {
			errorMsg, ok := body["error"].(string)
			return ok && len(errorMsg) > 0
		},
	})
	return b
}

// AddInvalidJSON adds a test scenario for malformed JSON input
func (b *TestScenarioBuilder) AddInvalidJSON(path string) *TestScenarioBuilder {
	b.scenarios = append(b.scenarios, TestScenario{
		Name:           "InvalidJSON",
		Description:    "Test endpoint with malformed JSON",
		Method:         "POST",
		Path:           path,
		Body:           `{"invalid": "json"`, // Malformed JSON
		ExpectedStatus: http.StatusBadRequest,
		ValidateBody: func(body map[string]interface{}) bool {
			_, hasError := body["error"]
			return hasError
		},
	})
	return b
}

// AddMissingRequiredFields adds a test scenario for missing required fields
func (b *TestScenarioBuilder) AddMissingRequiredFields(path string, emptyBody map[string]interface{}) *TestScenarioBuilder {
	b.scenarios = append(b.scenarios, TestScenario{
		Name:           "MissingRequiredFields",
		Description:    "Test endpoint with missing required fields",
		Method:         "POST",
		Path:           path,
		Body:           emptyBody,
		ExpectedStatus: http.StatusBadRequest,
		ValidateBody: func(body map[string]interface{}) bool {
			_, hasError := body["error"]
			return hasError
		},
	})
	return b
}

// AddInvalidTone adds a test scenario for invalid tone parameter
func (b *TestScenarioBuilder) AddInvalidTone(path string) *TestScenarioBuilder {
	b.scenarios = append(b.scenarios, TestScenario{
		Name:           "InvalidTone",
		Description:    "Test template generation with invalid tone",
		Method:         "POST",
		Path:           path,
		Body: map[string]interface{}{
			"purpose": "Test email",
			"tone":    "invalid-tone",
		},
		ExpectedStatus: http.StatusBadRequest,
		ValidateBody: func(body map[string]interface{}) bool {
			errorMsg, ok := body["error"].(string)
			return ok && contains(errorMsg, "tone")
		},
	})
	return b
}

// AddDatabaseUnavailable adds a test scenario for database unavailability
func (b *TestScenarioBuilder) AddDatabaseUnavailable(path string, method string) *TestScenarioBuilder {
	b.scenarios = append(b.scenarios, TestScenario{
		Name:           "DatabaseUnavailable",
		Description:    "Test endpoint when database is unavailable",
		Method:         method,
		Path:           path,
		ExpectedStatus: http.StatusServiceUnavailable,
		Setup: func(t *testing.T) interface{} {
			// Save original db and set to nil
			originalDB := db
			db = nil
			return originalDB
		},
		Cleanup: func(data interface{}) {
			// Restore original db
			if originalDB, ok := data.(*sql.DB); ok {
				db = originalDB
			}
		},
		ValidateBody: func(body map[string]interface{}) bool {
			errorMsg, ok := body["error"].(string)
			return ok && contains(errorMsg, "Database")
		},
	})
	return b
}

// Build returns all configured test scenarios
func (b *TestScenarioBuilder) Build() []TestScenario {
	return b.scenarios
}

// ErrorTestPattern defines systematic error testing patterns
type ErrorTestPattern struct {
	Name           string
	Description    string
	Request        HTTPTestRequest
	ExpectedStatus int
	ErrorContains  string
	Setup          func(*testing.T) interface{}
	Cleanup        func(interface{})
}

// PerformanceTestPattern defines performance testing scenarios
type PerformanceTestPattern struct {
	Name           string
	Description    string
	Request        HTTPTestRequest
	MaxDuration    time.Duration
	MinThroughput  int // Requests per second
	Setup          func(*testing.T) interface{}
	Cleanup        func(interface{})
}

// RunPerformanceTest executes a performance test pattern
func RunPerformanceTest(t *testing.T, router *gin.Engine, pattern PerformanceTestPattern) {
	t.Helper()

	// Setup
	var setupData interface{}
	if pattern.Setup != nil {
		setupData = pattern.Setup(t)
	}
	if pattern.Cleanup != nil {
		defer pattern.Cleanup(setupData)
	}

	// Execute test
	startTime := time.Now()
	recorder, err := makeHTTPRequest(router, pattern.Request)
	duration := time.Since(startTime)

	if err != nil {
		t.Fatalf("%s: Failed to execute request: %v", pattern.Name, err)
	}

	// Validate duration
	if duration > pattern.MaxDuration {
		t.Errorf("%s: Request took %v, expected < %v", pattern.Name, duration, pattern.MaxDuration)
	}

	// Validate response
	if recorder.Code != http.StatusOK && recorder.Code != http.StatusCreated {
		t.Errorf("%s: Expected success status, got %d", pattern.Name, recorder.Code)
	}

	t.Logf("%s: Completed in %v", pattern.Name, duration)
}

// CampaignTestSuite provides campaign-specific test patterns
type CampaignTestSuite struct {
	Router *gin.Engine
	TestDB *TestDatabase
}

// RunCampaignErrorTests executes comprehensive error tests for campaign endpoints
func (suite *CampaignTestSuite) RunCampaignErrorTests(t *testing.T) {
	patterns := NewTestScenarioBuilder().
		AddNonExistentCampaign("/api/v1/campaigns/%s", "GET").
		AddInvalidJSON("/api/v1/campaigns").
		AddMissingRequiredFields("/api/v1/campaigns", map[string]interface{}{}).
		Build()

	for _, pattern := range patterns {
		t.Run(pattern.Name, func(t *testing.T) {
			if pattern.Setup != nil {
				setupData := pattern.Setup(t)
				if pattern.Cleanup != nil {
					defer pattern.Cleanup(setupData)
				}
			}

			var recorder *httptest.ResponseRecorder
			var err error

			// Handle different body types
			switch body := pattern.Body.(type) {
			case string:
				// Raw string body (e.g., malformed JSON)
				req, _ := http.NewRequest(pattern.Method, pattern.Path, bytes.NewBufferString(body))
				req.Header.Set("Content-Type", "application/json")
				recorder = httptest.NewRecorder()
				suite.Router.ServeHTTP(recorder, req)
			default:
				// Normal JSON body
				recorder, err = makeHTTPRequest(suite.Router, HTTPTestRequest{
					Method:  pattern.Method,
					Path:    pattern.Path,
					Body:    pattern.Body,
					Headers: pattern.Headers,
				})
			}

			if err != nil {
				t.Fatalf("Failed to make HTTP request: %v", err)
			}

			if recorder.Code != pattern.ExpectedStatus {
				t.Errorf("Expected status %d, got %d. Body: %s",
					pattern.ExpectedStatus, recorder.Code, recorder.Body.String())
			}

			if pattern.ValidateBody != nil {
				var body map[string]interface{}
				if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
					t.Errorf("Failed to parse response body: %v", err)
				} else if !pattern.ValidateBody(body) {
					t.Errorf("Body validation failed: %v", body)
				}
			}
		})
	}
}

// RunTemplateErrorTests executes comprehensive error tests for template endpoints
func (suite *CampaignTestSuite) RunTemplateErrorTests(t *testing.T) {
	patterns := NewTestScenarioBuilder().
		AddInvalidTone("/api/v1/templates/generate").
		AddMissingRequiredFields("/api/v1/templates/generate", map[string]interface{}{
			"tone": "professional",
			// Missing "purpose" field
		}).
		Build()

	for _, pattern := range patterns {
		t.Run(pattern.Name, func(t *testing.T) {
			recorder, err := makeHTTPRequest(suite.Router, HTTPTestRequest{
				Method:  pattern.Method,
				Path:    pattern.Path,
				Body:    pattern.Body,
				Headers: pattern.Headers,
			})

			if err != nil {
				t.Fatalf("Failed to make HTTP request: %v", err)
			}

			if recorder.Code != pattern.ExpectedStatus {
				t.Errorf("Expected status %d, got %d. Body: %s",
					pattern.ExpectedStatus, recorder.Code, recorder.Body.String())
			}

			if pattern.ValidateBody != nil {
				var body map[string]interface{}
				if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
					t.Errorf("Failed to parse response body: %v", err)
				} else if !pattern.ValidateBody(body) {
					t.Errorf("Body validation failed: %v", body)
				}
			}
		})
	}
}
