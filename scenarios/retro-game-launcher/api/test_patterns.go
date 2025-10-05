// +build testing

package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
)

// ErrorTestPattern defines systematic error condition testing
type ErrorTestPattern struct {
	Name           string
	Path           string
	Method         string
	Body           interface{}
	URLVars        map[string]string
	ExpectedStatus int
	ErrorContains  string
}

// TestScenarioBuilder provides fluent interface for building test scenarios
type TestScenarioBuilder struct {
	patterns []ErrorTestPattern
}

// NewTestScenarioBuilder creates a new test scenario builder
func NewTestScenarioBuilder() *TestScenarioBuilder {
	return &TestScenarioBuilder{
		patterns: make([]ErrorTestPattern, 0),
	}
}

// AddInvalidUUID adds invalid UUID test pattern
func (b *TestScenarioBuilder) AddInvalidUUID(path string) *TestScenarioBuilder {
	b.patterns = append(b.patterns, ErrorTestPattern{
		Name:           "InvalidUUID",
		Path:           path,
		Method:         "GET",
		URLVars:        map[string]string{"id": "invalid-uuid-format"},
		ExpectedStatus: http.StatusNotFound,
		ErrorContains:  "",
	})
	return b
}

// AddNonExistentGame adds non-existent game test pattern
func (b *TestScenarioBuilder) AddNonExistentGame(path string) *TestScenarioBuilder {
	b.patterns = append(b.patterns, ErrorTestPattern{
		Name:           "NonExistentGame",
		Path:           path,
		Method:         "GET",
		URLVars:        map[string]string{"id": uuid.New().String()},
		ExpectedStatus: http.StatusNotFound,
		ErrorContains:  "not found",
	})
	return b
}

// AddInvalidJSON adds malformed JSON test pattern
func (b *TestScenarioBuilder) AddInvalidJSON(path string, method string) *TestScenarioBuilder {
	b.patterns = append(b.patterns, ErrorTestPattern{
		Name:           "InvalidJSON",
		Path:           path,
		Method:         method,
		Body:           `{"invalid": json}`,
		ExpectedStatus: http.StatusBadRequest,
		ErrorContains:  "",
	})
	return b
}

// AddMissingRequiredField adds missing field test pattern
func (b *TestScenarioBuilder) AddMissingRequiredField(path string, method string, body interface{}, fieldName string) *TestScenarioBuilder {
	b.patterns = append(b.patterns, ErrorTestPattern{
		Name:           fmt.Sprintf("Missing_%s", fieldName),
		Path:           path,
		Method:         method,
		Body:           body,
		ExpectedStatus: http.StatusBadRequest,
		ErrorContains:  "",
	})
	return b
}

// AddEmptyBody adds empty body test pattern
func (b *TestScenarioBuilder) AddEmptyBody(path string, method string) *TestScenarioBuilder {
	b.patterns = append(b.patterns, ErrorTestPattern{
		Name:           "EmptyBody",
		Path:           path,
		Method:         method,
		Body:           "",
		ExpectedStatus: http.StatusBadRequest,
		ErrorContains:  "",
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

// RunErrorPatterns executes all error test patterns
func RunErrorPatterns(t *testing.T, env *TestEnvironment, patterns []ErrorTestPattern) {
	for _, pattern := range patterns {
		t.Run(pattern.Name, func(t *testing.T) {
			req := HTTPTestRequest{
				Method:  pattern.Method,
				Path:    pattern.Path,
				Body:    pattern.Body,
				URLVars: pattern.URLVars,
			}

			rr := makeHTTPRequest(env.Router, req)

			if rr.Code != pattern.ExpectedStatus {
				t.Errorf("Expected status %d, got %d. Body: %s",
					pattern.ExpectedStatus, rr.Code, rr.Body.String())
			}

			if pattern.ErrorContains != "" {
				body := rr.Body.String()
				if body == "" || (pattern.ErrorContains != "" && !contains(body, pattern.ErrorContains)) {
					t.Errorf("Expected error containing '%s', got: %s",
						pattern.ErrorContains, body)
				}
			}
		})
	}
}

// contains checks if a string contains a substring (case-insensitive)
func contains(s, substr string) bool {
	return len(substr) == 0 || len(s) >= len(substr) && (s == substr || contains(s[1:], substr) || s[:len(substr)] == substr)
}

// HandlerTestSuite provides comprehensive handler testing framework
type HandlerTestSuite struct {
	Name        string
	Description string
	Setup       func(t *testing.T, env *TestEnvironment) interface{}
	Tests       []HandlerTest
	Cleanup     func(setupData interface{})
}

// HandlerTest represents a single handler test case
type HandlerTest struct {
	Name           string
	Request        HTTPTestRequest
	ExpectedStatus int
	ValidateBody   func(t *testing.T, body string)
}

// Run executes the handler test suite
func (suite *HandlerTestSuite) Run(t *testing.T, env *TestEnvironment) {
	var setupData interface{}

	if suite.Setup != nil {
		setupData = suite.Setup(t, env)
	}

	if suite.Cleanup != nil {
		defer suite.Cleanup(setupData)
	}

	for _, test := range suite.Tests {
		t.Run(test.Name, func(t *testing.T) {
			rr := makeHTTPRequest(env.Router, test.Request)

			if rr.Code != test.ExpectedStatus {
				t.Errorf("Expected status %d, got %d. Body: %s",
					test.ExpectedStatus, rr.Code, rr.Body.String())
			}

			if test.ValidateBody != nil {
				test.ValidateBody(t, rr.Body.String())
			}
		})
	}
}

// Common validation patterns

// ValidateGameResponse validates a game response structure
func ValidateGameResponse(t *testing.T, body string) {
	t.Helper()

	var game Game
	if err := parseJSON(body, &game); err != nil {
		t.Errorf("Failed to parse game response: %v", err)
		return
	}

	if game.ID == "" {
		t.Error("Game ID should not be empty")
	}
	if game.Title == "" {
		t.Error("Game title should not be empty")
	}
	if game.Code == "" {
		t.Error("Game code should not be empty")
	}
}

// ValidateGamesArrayResponse validates an array of games
func ValidateGamesArrayResponse(t *testing.T, body string) {
	t.Helper()

	var games []Game
	if err := parseJSON(body, &games); err != nil {
		t.Errorf("Failed to parse games array response: %v", err)
		return
	}

	if len(games) == 0 {
		t.Log("Warning: Empty games array")
	}

	for i, game := range games {
		if game.ID == "" {
			t.Errorf("Game[%d] ID should not be empty", i)
		}
	}
}

// ValidateHealthResponse validates health check response
func ValidateHealthResponse(t *testing.T, body string) {
	t.Helper()

	var health map[string]interface{}
	if err := parseJSON(body, &health); err != nil {
		t.Errorf("Failed to parse health response: %v", err)
		return
	}

	if status, ok := health["status"]; !ok || status != "healthy" {
		t.Error("Health status should be 'healthy'")
	}

	if _, ok := health["services"]; !ok {
		t.Error("Health response should include services")
	}
}

// parseJSON is a helper to parse JSON strings
func parseJSON(jsonStr string, v interface{}) error {
	return json.Unmarshal([]byte(jsonStr), v)
}

// GameTestFactory provides factory methods for creating test games
type GameTestFactory struct{}

// NewGameTestFactory creates a new game test factory
func NewGameTestFactory() *GameTestFactory {
	return &GameTestFactory{}
}

// CreateBasicGame creates a basic test game
func (f *GameTestFactory) CreateBasicGame() *Game {
	return &Game{
		Title:     "Test Game",
		Prompt:    "Create a simple game",
		Code:      "const canvas = document.getElementById('game'); const ctx = canvas.getContext('2d'); function gameLoop() { requestAnimationFrame(gameLoop); } gameLoop();",
		Engine:    "html5",
		Published: true,
		Tags:      []string{"test"},
	}
}

// CreateUnpublishedGame creates an unpublished test game
func (f *GameTestFactory) CreateUnpublishedGame() *Game {
	game := f.CreateBasicGame()
	game.Published = false
	return game
}

// CreateHighRatedGame creates a high-rated test game
func (f *GameTestFactory) CreateHighRatedGame() *Game {
	game := f.CreateBasicGame()
	rating := 4.5
	game.Rating = &rating
	game.PlayCount = 100
	return game
}

// CreateGameWithTags creates a game with specific tags
func (f *GameTestFactory) CreateGameWithTags(tags []string) *Game {
	game := f.CreateBasicGame()
	game.Tags = tags
	return game
}

// UserTestFactory provides factory methods for creating test users
type UserTestFactory struct{}

// NewUserTestFactory creates a new user test factory
func NewUserTestFactory() *UserTestFactory {
	return &UserTestFactory{}
}

// CreateBasicUser creates a basic test user
func (f *UserTestFactory) CreateBasicUser(username, email string) *User {
	return &User{
		ID:                    uuid.New().String(),
		Username:              username,
		Email:                 email,
		SubscriptionTier:      "free",
		GamesCreatedThisMonth: 0,
		CreatedAt:             time.Now(),
	}
}

// CreatePremiumUser creates a premium test user
func (f *UserTestFactory) CreatePremiumUser(username, email string) *User {
	user := f.CreateBasicUser(username, email)
	user.SubscriptionTier = "premium"
	return user
}
