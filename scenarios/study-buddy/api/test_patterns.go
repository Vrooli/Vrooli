// +build testing

package main

import (
	"fmt"
	"net/http"
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
	Request        HTTPTestRequest
	Validate       func(t *testing.T, statusCode int, body string)
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
func (b *TestScenarioBuilder) AddInvalidUUID(method, path string) *TestScenarioBuilder {
	b.patterns = append(b.patterns, ErrorTestPattern{
		Name:           "InvalidUUID",
		Description:    "Test with invalid UUID format",
		ExpectedStatus: http.StatusBadRequest,
		Request: HTTPTestRequest{
			Method: method,
			Path:   path,
		},
	})
	return b
}

// AddMissingRequiredField adds a missing required field test pattern
func (b *TestScenarioBuilder) AddMissingRequiredField(method, path string, body map[string]interface{}) *TestScenarioBuilder {
	b.patterns = append(b.patterns, ErrorTestPattern{
		Name:           "MissingRequiredField",
		Description:    "Test with missing required field",
		ExpectedStatus: http.StatusBadRequest,
		Request: HTTPTestRequest{
			Method: method,
			Path:   path,
			Body:   body,
		},
	})
	return b
}

// AddInvalidJSON adds an invalid JSON test pattern
func (b *TestScenarioBuilder) AddInvalidJSON(method, path string) *TestScenarioBuilder {
	b.patterns = append(b.patterns, ErrorTestPattern{
		Name:           "InvalidJSON",
		Description:    "Test with malformed JSON",
		ExpectedStatus: http.StatusBadRequest,
		Request: HTTPTestRequest{
			Method: method,
			Path:   path,
			Body:   `{"invalid": json"}`, // Malformed JSON
		},
	})
	return b
}

// AddNonExistentResource adds a non-existent resource test pattern
func (b *TestScenarioBuilder) AddNonExistentResource(method, path string) *TestScenarioBuilder {
	nonExistentID := uuid.New().String()
	pathWithID := fmt.Sprintf(path, nonExistentID)

	b.patterns = append(b.patterns, ErrorTestPattern{
		Name:           "NonExistentResource",
		Description:    "Test with non-existent resource ID",
		ExpectedStatus: http.StatusNotFound,
		Request: HTTPTestRequest{
			Method: method,
			Path:   pathWithID,
		},
	})
	return b
}

// AddEmptyBody adds an empty body test pattern
func (b *TestScenarioBuilder) AddEmptyBody(method, path string) *TestScenarioBuilder {
	b.patterns = append(b.patterns, ErrorTestPattern{
		Name:           "EmptyBody",
		Description:    "Test with empty request body",
		ExpectedStatus: http.StatusBadRequest,
		Request: HTTPTestRequest{
			Method: method,
			Path:   path,
			Body:   "",
		},
	})
	return b
}

// Build returns the constructed error test patterns
func (b *TestScenarioBuilder) Build() []ErrorTestPattern {
	return b.patterns
}

// HandlerTestSuite provides a comprehensive test framework for HTTP handlers
type HandlerTestSuite struct {
	Name        string
	Router      *gin.Engine
	SetupData   interface{}
	CleanupFunc func()
}

// RunErrorTests executes a suite of error condition tests
func (suite *HandlerTestSuite) RunErrorTests(t *testing.T, patterns []ErrorTestPattern) {
	for _, pattern := range patterns {
		t.Run(fmt.Sprintf("%s_%s", suite.Name, pattern.Name), func(t *testing.T) {
			w, err := makeHTTPRequest(suite.Router, pattern.Request)
			if err != nil {
				t.Fatalf("Failed to make HTTP request: %v", err)
			}

			if w.Code != pattern.ExpectedStatus {
				t.Errorf("Expected status %d, got %d. Body: %s",
					pattern.ExpectedStatus, w.Code, w.Body.String())
			}

			// Custom validation if provided
			if pattern.Validate != nil {
				pattern.Validate(t, w.Code, w.Body.String())
			}
		})
	}
}

// PerformanceTestPattern defines performance testing scenarios
type PerformanceTestPattern struct {
	Name           string
	Description    string
	MaxDuration    time.Duration
	Request        HTTPTestRequest
	MinIterations  int
	Setup          func(t *testing.T) interface{}
	Cleanup        func(setupData interface{})
}

// RunPerformanceTest executes a performance test
func RunPerformanceTest(t *testing.T, router *gin.Engine, pattern PerformanceTestPattern) {
	t.Run(pattern.Name, func(t *testing.T) {
		var setupData interface{}
		if pattern.Setup != nil {
			setupData = pattern.Setup(t)
		}

		if pattern.Cleanup != nil {
			defer pattern.Cleanup(setupData)
		}

		start := time.Now()
		iterations := 0

		for iterations < pattern.MinIterations {
			w, err := makeHTTPRequest(router, pattern.Request)
			if err != nil {
				t.Fatalf("Request failed: %v", err)
			}

			if w.Code >= 400 {
				t.Fatalf("Request returned error status: %d", w.Code)
			}

			iterations++
		}

		duration := time.Since(start)
		avgDuration := duration / time.Duration(iterations)

		if avgDuration > pattern.MaxDuration {
			t.Errorf("Performance test failed: average duration %v exceeds maximum %v",
				avgDuration, pattern.MaxDuration)
		}

		t.Logf("Performance: %d iterations in %v (avg: %v per request)",
			iterations, duration, avgDuration)
	})
}

// Common test patterns for study-buddy

// FlashcardGenerationPatterns returns error patterns for flashcard generation
func FlashcardGenerationPatterns() []ErrorTestPattern {
	return NewTestScenarioBuilder().
		AddMissingRequiredField("POST", "/api/flashcards/generate", map[string]interface{}{
			"content": "Test content",
			// Missing subject_id
		}).
		AddMissingRequiredField("POST", "/api/flashcards/generate", map[string]interface{}{
			"subject_id": "test-subject",
			// Missing content
		}).
		AddInvalidJSON("POST", "/api/flashcards/generate").
		AddEmptyBody("POST", "/api/flashcards/generate").
		Build()
}

// StudySessionPatterns returns error patterns for study sessions
func StudySessionPatterns() []ErrorTestPattern {
	return NewTestScenarioBuilder().
		AddMissingRequiredField("POST", "/api/study/session/start", map[string]interface{}{
			"subject_id":   "test-subject",
			"session_type": "review",
			// Missing user_id
		}).
		AddInvalidJSON("POST", "/api/study/session/start").
		AddEmptyBody("POST", "/api/study/session/start").
		Build()
}

// FlashcardAnswerPatterns returns error patterns for flashcard answers
func FlashcardAnswerPatterns() []ErrorTestPattern {
	return NewTestScenarioBuilder().
		AddMissingRequiredField("POST", "/api/study/answer", map[string]interface{}{
			"flashcard_id":  "test-card",
			"user_response": "good",
			// Missing session_id
		}).
		AddInvalidJSON("POST", "/api/study/answer").
		Build()
}

// SubjectManagementPatterns returns error patterns for subject management
func SubjectManagementPatterns() []ErrorTestPattern {
	return NewTestScenarioBuilder().
		AddMissingRequiredField("POST", "/api/subjects", map[string]interface{}{
			"name": "Test Subject",
			// Missing user_id
		}).
		AddInvalidJSON("POST", "/api/subjects").
		Build()
}

// SpacedRepetitionPerformanceTest returns a performance test for spaced repetition
func SpacedRepetitionPerformanceTest() PerformanceTestPattern {
	return PerformanceTestPattern{
		Name:          "SpacedRepetitionCalculation",
		Description:   "Test spaced repetition algorithm performance",
		MaxDuration:   100 * time.Millisecond,
		MinIterations: 100,
		Request: HTTPTestRequest{
			Method: "POST",
			Path:   "/api/study/answer",
			Body: map[string]interface{}{
				"session_id":    "test-session",
				"flashcard_id":  "test-card",
				"user_response": "good",
				"response_time": 5000,
				"user_id":       "test-user",
			},
		},
	}
}

// FlashcardGenerationPerformanceTest returns a performance test for flashcard generation
func FlashcardGenerationPerformanceTest() PerformanceTestPattern {
	return PerformanceTestPattern{
		Name:          "FlashcardGeneration",
		Description:   "Test flashcard generation performance",
		MaxDuration:   5 * time.Second,
		MinIterations: 10,
		Request: HTTPTestRequest{
			Method: "POST",
			Path:   "/api/flashcards/generate",
			Body: map[string]interface{}{
				"subject_id":       "test-subject",
				"content":          "The mitochondria is the powerhouse of the cell. It produces ATP through cellular respiration.",
				"card_count":       3,
				"difficulty_level": "intermediate",
			},
		},
	}
}

// DueCardsRetrievalPerformanceTest returns a performance test for due cards retrieval
func DueCardsRetrievalPerformanceTest() PerformanceTestPattern {
	return PerformanceTestPattern{
		Name:          "DueCardsRetrieval",
		Description:   "Test due cards retrieval performance",
		MaxDuration:   100 * time.Millisecond,
		MinIterations: 50,
		Request: HTTPTestRequest{
			Method: "GET",
			Path:   "/api/study/due-cards",
			QueryParams: map[string]string{
				"user_id": "test-user",
				"limit":   "10",
			},
		},
	}
}

// BusinessLogicTestPattern defines business logic validation scenarios
type BusinessLogicTestPattern struct {
	Name        string
	Description string
	Setup       func(t *testing.T, router *gin.Engine) interface{}
	Execute     func(t *testing.T, router *gin.Engine, setupData interface{})
	Validate    func(t *testing.T, setupData interface{}, results interface{})
	Cleanup     func(setupData interface{})
}

// RunBusinessLogicTest executes a business logic test
func RunBusinessLogicTest(t *testing.T, router *gin.Engine, pattern BusinessLogicTestPattern) {
	t.Run(pattern.Name, func(t *testing.T) {
		var setupData interface{}
		if pattern.Setup != nil {
			setupData = pattern.Setup(t, router)
		}

		if pattern.Cleanup != nil {
			defer pattern.Cleanup(setupData)
		}

		pattern.Execute(t, router, setupData)
	})
}
