// Package testing provides LLM-based skill testing via Ollama.
package testing

// TestRepository defines the interface for test result storage operations.
// This is the testing seam for the testing domain.
// Implementations: Repository (database, production), MockRepository (testing).
type TestRepository interface {
	// Save stores a test result in the database.
	Save(result *TestResult) error

	// GetHistory retrieves test history for a skill, newest first.
	GetHistory(skillID string, limit int) ([]TestResult, error)
}

// LLMClient defines the interface for LLM operations.
// This is the testing seam for LLM interactions.
// Implementations: OllamaClient (production), MockLLMClient (testing).
type LLMClient interface {
	// IsEnabled returns true if the LLM client is configured.
	IsEnabled() bool

	// Generate runs a prompt through the LLM and returns the response.
	// Returns (response, responseTimeMs, error).
	Generate(role, prompt string, maxTokens int, temperature float64) (*OllamaResponse, float64, error)
}
