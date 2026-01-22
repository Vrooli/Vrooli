// Package testing provides LLM-based prompt testing via Ollama.
// This allows users to test prompts with local LLMs and track results.
package testing

import "time"

// TestResult represents a test result stored in the database.
type TestResult struct {
	ID           string    `json:"id"`
	PromptID     string    `json:"promptId"`
	Model        string    `json:"model"`
	InputVars    *string   `json:"inputVariables,omitempty"`
	Response     *string   `json:"response,omitempty"`
	ResponseTime *float64  `json:"responseTime,omitempty"`
	TokenCount   *int      `json:"tokenCount,omitempty"`
	Rating       *int      `json:"rating,omitempty"`
	Notes        *string   `json:"notes,omitempty"`
	TestedAt     time.Time `json:"testedAt"`
}

// TestRequest is the request body for testing a prompt.
type TestRequest struct {
	Model       string            `json:"model"`
	Variables   map[string]string `json:"variables"`
	MaxTokens   *int              `json:"maxTokens"`
	Temperature *float64          `json:"temperature"`
}

// TestResponse is returned after running a prompt test.
type TestResponse struct {
	TestID       string    `json:"testId"`
	Model        string    `json:"model"`
	Response     string    `json:"response"`
	ResponseTime float64   `json:"responseTime"`
	TokenCount   int       `json:"tokenCount"`
	TestedAt     time.Time `json:"testedAt"`
}

// OllamaRequest is the request body for Ollama's generate endpoint.
type OllamaRequest struct {
	Model   string                 `json:"model"`
	Prompt  string                 `json:"prompt"`
	Stream  bool                   `json:"stream"`
	Options map[string]interface{} `json:"options"`
}

// OllamaResponse is the response from Ollama's generate endpoint.
type OllamaResponse struct {
	Response  string `json:"response"`
	EvalCount int    `json:"eval_count"`
}
