// +build testing

package main

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
)

// ErrorTestScenario defines a reusable error test case
type ErrorTestScenario struct {
	Name           string
	Request        HTTPTestRequest
	ExpectedStatus int
	Description    string
}

// TestScenarioBuilder provides fluent interface for building test scenarios
type TestScenarioBuilder struct {
	scenarios []ErrorTestScenario
}

// NewTestScenarioBuilder creates a new scenario builder
func NewTestScenarioBuilder() *TestScenarioBuilder {
	return &TestScenarioBuilder{
		scenarios: []ErrorTestScenario{},
	}
}

// AddInvalidJSON adds a test for invalid JSON input
func (b *TestScenarioBuilder) AddInvalidJSON(method, path string) *TestScenarioBuilder {
	b.scenarios = append(b.scenarios, ErrorTestScenario{
		Name: "InvalidJSON",
		Request: HTTPTestRequest{
			Method: method,
			Path:   path,
			Body:   `{"invalid": "json"`, // Malformed JSON
		},
		ExpectedStatus: http.StatusBadRequest,
		Description:    "Request with malformed JSON should return 400",
	})
	return b
}

// AddEmptyBody adds a test for empty request body
func (b *TestScenarioBuilder) AddEmptyBody(method, path string) *TestScenarioBuilder {
	b.scenarios = append(b.scenarios, ErrorTestScenario{
		Name: "EmptyBody",
		Request: HTTPTestRequest{
			Method: method,
			Path:   path,
			Body:   "",
		},
		ExpectedStatus: http.StatusBadRequest,
		Description:    "Request with empty body should return 400",
	})
	return b
}

// AddMissingRequiredField adds a test for missing required fields
func (b *TestScenarioBuilder) AddMissingRequiredField(method, path, fieldName string, body interface{}) *TestScenarioBuilder {
	b.scenarios = append(b.scenarios, ErrorTestScenario{
		Name: fmt.Sprintf("Missing%s", fieldName),
		Request: HTTPTestRequest{
			Method: method,
			Path:   path,
			Body:   body,
		},
		ExpectedStatus: http.StatusBadRequest,
		Description:    fmt.Sprintf("Request without required field '%s' should return 400", fieldName),
	})
	return b
}

// AddInvalidQueryParam adds a test for invalid query parameters
func (b *TestScenarioBuilder) AddInvalidQueryParam(path, param, value string) *TestScenarioBuilder {
	b.scenarios = append(b.scenarios, ErrorTestScenario{
		Name: fmt.Sprintf("Invalid%sParam", param),
		Request: HTTPTestRequest{
			Method: "GET",
			Path:   fmt.Sprintf("%s?%s=%s", path, param, value),
		},
		ExpectedStatus: http.StatusBadRequest,
		Description:    fmt.Sprintf("Request with invalid %s parameter should handle gracefully", param),
	})
	return b
}

// Build returns the constructed scenarios
func (b *TestScenarioBuilder) Build() []ErrorTestScenario {
	return b.scenarios
}

// PerformanceTestPattern defines performance testing requirements
type PerformanceTestPattern struct {
	Name            string
	Description     string
	MaxDuration     time.Duration
	MinRequestsPerSec int
	Setup           func(t *testing.T) interface{}
	Execute         func(t *testing.T, setupData interface{}) time.Duration
	Validate        func(t *testing.T, duration time.Duration, setupData interface{})
	Cleanup         func(setupData interface{})
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

		// Execute
		start := time.Now()
		duration := pattern.Execute(t, setupData)
		_ = start // Keep for potential future use

		// Validate duration
		if duration > pattern.MaxDuration {
			t.Errorf("Performance test %s exceeded max duration: took %v, expected < %v",
				pattern.Name, duration, pattern.MaxDuration)
		}

		// Custom validation
		if pattern.Validate != nil {
			pattern.Validate(t, duration, setupData)
		}
	})
}

// Common test data generators

// generateTestScores creates multiple test scores
func generateTestScores(count int) []Score {
	scores := make([]Score, count)
	for i := 0; i < count; i++ {
		scores[i] = Score{
			Name:       fmt.Sprintf("Player%d", i+1),
			Score:      (50 + i*5) * (80 + i) / 100,
			WPM:        50 + i*5,
			Accuracy:   80 + i,
			MaxCombo:   10 + i,
			Difficulty: []string{"easy", "medium", "hard"}[i%3],
			Mode:       "classic",
			CreatedAt:  time.Now().Add(-time.Duration(i) * time.Hour),
		}
	}
	return scores
}

// generateTestSessions creates multiple test sessions
func generateTestSessions(count int) []SessionStats {
	sessions := make([]SessionStats, count)
	for i := 0; i < count; i++ {
		sessions[i] = SessionStats{
			SessionID:       uuid.New().String(),
			WPM:             40 + i*10,
			Accuracy:        80.0 + float64(i)*2,
			CharactersTyped: 400 + i*50,
			ErrorCount:      20 - i,
			TimeSpent:       60,
			Difficulty:      []string{"easy", "medium", "hard"}[i%3],
			TextCompleted:   true,
		}
	}
	return sessions
}

// Boundary value test helpers

// BoundaryTestCase represents a boundary value test
type BoundaryTestCase struct {
	Name        string
	Value       interface{}
	ShouldPass  bool
	Description string
}

// generateWPMBoundaryTests creates WPM boundary test cases
func generateWPMBoundaryTests() []BoundaryTestCase {
	return []BoundaryTestCase{
		{Name: "ZeroWPM", Value: 0, ShouldPass: true, Description: "Zero WPM should be handled"},
		{Name: "NegativeWPM", Value: -1, ShouldPass: false, Description: "Negative WPM should be rejected"},
		{Name: "MaxWPM", Value: 300, ShouldPass: true, Description: "High WPM should be accepted"},
		{Name: "ExtremeWPM", Value: 10000, ShouldPass: false, Description: "Unrealistic WPM should be rejected"},
	}
}

// generateAccuracyBoundaryTests creates accuracy boundary test cases
func generateAccuracyBoundaryTests() []BoundaryTestCase {
	return []BoundaryTestCase{
		{Name: "ZeroAccuracy", Value: 0.0, ShouldPass: true, Description: "Zero accuracy should be handled"},
		{Name: "NegativeAccuracy", Value: -1.0, ShouldPass: false, Description: "Negative accuracy should be rejected"},
		{Name: "PerfectAccuracy", Value: 100.0, ShouldPass: true, Description: "100% accuracy should be accepted"},
		{Name: "OverAccuracy", Value: 101.0, ShouldPass: false, Description: "Accuracy > 100 should be rejected"},
	}
}

// Edge case helpers

// createEdgeCaseScore creates edge case test scores
func createEdgeCaseScore(caseType string) Score {
	baseScore := Score{
		Name:       "EdgeCase",
		Mode:       "classic",
		Difficulty: "medium",
		CreatedAt:  time.Now(),
	}

	switch caseType {
	case "minimal":
		baseScore.WPM = 1
		baseScore.Accuracy = 1
		baseScore.Score = 1
		baseScore.MaxCombo = 0
	case "maximum":
		baseScore.WPM = 200
		baseScore.Accuracy = 100
		baseScore.Score = 20000
		baseScore.MaxCombo = 1000
	case "empty_name":
		baseScore.Name = ""
		baseScore.WPM = 50
		baseScore.Accuracy = 90
		baseScore.Score = 4500
	case "long_name":
		baseScore.Name = "ThisIsAReallyLongNameThatExceedsNormalLimitsForTestingPurposes"
		baseScore.WPM = 50
		baseScore.Accuracy = 90
		baseScore.Score = 4500
	}

	return baseScore
}

// createEdgeCaseAdaptiveRequest creates edge case adaptive text requests
func createEdgeCaseAdaptiveRequest(caseType string) AdaptiveTextRequest {
	baseReq := AdaptiveTextRequest{
		UserID:     uuid.New().String(),
		Difficulty: "medium",
		UserLevel:  "intermediate",
		TextLength: "medium",
	}

	switch caseType {
	case "no_target_words":
		baseReq.TargetWords = []string{}
		baseReq.ProblemChars = []string{}
	case "many_target_words":
		baseReq.TargetWords = []string{"word1", "word2", "word3", "word4", "word5", "word6", "word7", "word8", "word9", "word10"}
	case "many_mistakes":
		mistakes := make([]struct {
			Word       string `json:"word"`
			Char       string `json:"char"`
			Position   string `json:"position"`
			ErrorCount int    `json:"errorCount"`
		}, 20)
		for i := range mistakes {
			mistakes[i].Word = fmt.Sprintf("word%d", i)
			mistakes[i].Char = "x"
			mistakes[i].Position = "0"
			mistakes[i].ErrorCount = 5
		}
		baseReq.PreviousMistakes = mistakes
	case "empty_strings":
		baseReq.UserID = ""
		baseReq.Difficulty = ""
		baseReq.UserLevel = ""
		baseReq.TextLength = ""
	}

	return baseReq
}

// Assertion helpers for complex validations

// assertValidCoachingResponse validates coaching response structure
func assertValidCoachingResponse(t *testing.T, response CoachingResponse) {
	t.Helper()

	if response.Feedback == "" {
		t.Error("Coaching response missing feedback")
	}
	if len(response.Tips) == 0 {
		t.Error("Coaching response missing tips")
	}
	if response.EncouragingNote == "" {
		t.Error("Coaching response missing encouraging note")
	}
	if response.NextChallenge == "" {
		t.Error("Coaching response missing next challenge")
	}
}

// assertValidProcessedStats validates processed stats structure
func assertValidProcessedStats(t *testing.T, stats ProcessedStats) {
	t.Helper()

	if stats.SessionID == "" {
		t.Error("Processed stats missing session ID")
	}
	if stats.Timestamp == "" {
		t.Error("Processed stats missing timestamp")
	}
	if stats.Metrics.WPM < 0 {
		t.Error("Invalid WPM in processed stats")
	}
	if stats.Metrics.Accuracy < 0 || stats.Metrics.Accuracy > 100 {
		t.Error("Invalid accuracy in processed stats")
	}
	if stats.Analysis.PerformanceLevel == "" {
		t.Error("Processed stats missing performance level")
	}
}

// assertValidAdaptiveResponse validates adaptive text response
func assertValidAdaptiveResponse(t *testing.T, response AdaptiveTextResponse) {
	t.Helper()

	if response.Text == "" {
		t.Error("Adaptive response missing text")
	}
	if response.WordCount <= 0 {
		t.Error("Adaptive response has invalid word count")
	}
	if response.Difficulty == "" {
		t.Error("Adaptive response missing difficulty")
	}
	if !response.IsAdaptive {
		t.Error("Adaptive response should be marked as adaptive")
	}
	if response.Timestamp == "" {
		t.Error("Adaptive response missing timestamp")
	}
}
