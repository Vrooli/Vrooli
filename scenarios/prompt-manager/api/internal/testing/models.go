// Package testing provides LLM-based skill testing via Ollama.
// This allows users to test skills with local LLMs and track results in SQLite.
package testing

import "time"

// TestResult represents a test result stored in the database.
type TestResult struct {
	ID           string    `json:"id"`
	SkillID      string    `json:"skillId"`
	Role         string    `json:"role"`
	InputVars    *string   `json:"inputVariables,omitempty"`
	Response     *string   `json:"response,omitempty"`
	ResponseTime *float64  `json:"responseTime,omitempty"`
	TokenCount   *int      `json:"tokenCount,omitempty"`
	Rating       *int      `json:"rating,omitempty"`
	Notes        *string   `json:"notes,omitempty"`
	TestedAt     time.Time `json:"testedAt"`
}

// TestRequest is the request body for testing a skill.
type TestRequest struct {
	Role        string            `json:"role"`
	Variables   map[string]string `json:"variables"`
	MaxTokens   *int              `json:"maxTokens"`
	Temperature *float64          `json:"temperature"`
}

// TestResponse is returned after running a skill test.
type TestResponse struct {
	TestID       string    `json:"testId"`
	Role         string    `json:"role"`
	Response     string    `json:"response"`
	ResponseTime float64   `json:"responseTime"`
	TokenCount   int       `json:"tokenCount"`
	TestedAt     time.Time `json:"testedAt"`
}

// OllamaResponse is the response from resource-ollama gateway generate.
type OllamaResponse struct {
	Response  string `json:"response"`
	EvalCount int    `json:"eval_count"`
}
