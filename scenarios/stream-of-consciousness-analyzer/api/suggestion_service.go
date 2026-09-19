// DOC: docs/concepts/ARCHITECTURE.md#llm-integration
// DOC: docs/reference/api-endpoints.md#suggestions
// DOC: docs/internal/SEAMS.md#environment-reader
package main

import (
	"database/sql"
	"fmt"
	"os"
	"time"
)

// Suggestion represents an LLM-generated thought connection suggestion
type Suggestion struct {
	ID         string    `json:"id"`
	SchemeID   string    `json:"scheme_id"`
	SourceID   string    `json:"source_id"`
	TargetID   string    `json:"target_id"`
	Label      string    `json:"label"`
	Confidence float64   `json:"confidence"`
	Dismissed  bool      `json:"dismissed"`
	CreatedAt  time.Time `json:"created_at"`
}

// LLMProvider represents an LLM backend with its availability status.
//
// The Active and Fallback flags form a two-tier priority system:
//   - Active=true, Fallback=false → primary provider, preferred when healthy
//   - Active=true, Fallback=true  → fallback provider, used only when no primary is active
//   - Active=false                → unavailable, skipped during selection
type LLMProvider struct {
	Name     string `json:"name"`
	URL      string `json:"url"`
	Active   bool   `json:"active"`
	Fallback bool   `json:"fallback"`
}

// EnvReader abstracts environment variable access for testability.
// Production code uses os.Getenv; tests can substitute controlled values.
type EnvReader func(key string) string

// SuggestionService handles LLM-based thought suggestions with provider fallback.
//
// Provider selection strategy (see GetActiveProvider):
//   - Primary providers (Fallback=false) are tried first — these are local/fast.
//   - Fallback providers (Fallback=true) are used only when no primary is available.
//   - Within each tier, the first Active provider wins.
//
// CHANGE AXIS: Adding a new LLM provider requires only a new entry in
// NewSuggestionServiceWithEnv. The selection logic in GetActiveProvider
// handles any number of providers without modification.
type SuggestionService struct {
	db        *sql.DB
	providers []LLMProvider
}

// NewSuggestionService creates a new SuggestionService with provider configuration.
// Uses os.Getenv for environment access; use NewSuggestionServiceWithEnv for testing.
func NewSuggestionService(db *sql.DB) *SuggestionService {
	return NewSuggestionServiceWithEnv(db, os.Getenv)
}

// NewSuggestionServiceWithEnv creates a SuggestionService with injectable env reader.
// This seam allows tests to control environment-dependent provider configuration
// without modifying process-level environment variables.
func NewSuggestionServiceWithEnv(db *sql.DB, env EnvReader) *SuggestionService {
	providers := []LLMProvider{
		{
			Name:     "ollama",
			URL:      OllamaProviderTransport,
			Active:   true,
			Fallback: false,
		},
		{
			Name:     "openrouter",
			URL:      OpenRouterURL,
			Active:   env("OPENROUTER_API_KEY") != "",
			Fallback: true,
		},
	}
	return &SuggestionService{db: db, providers: providers}
}

// GetProviders returns the current provider configuration
func (s *SuggestionService) GetProviders() []LLMProvider {
	return s.providers
}

// GetActiveProvider selects the best available LLM provider using a two-tier strategy:
//  1. Prefer any active primary provider (local, low-latency).
//  2. Fall back to an active fallback provider (remote, higher-latency).
//  3. If neither tier has an active provider, return an error.
//
// This two-pass approach ensures local providers are always preferred when healthy,
// while still providing service when local infra is down.
func (s *SuggestionService) GetActiveProvider() (*LLMProvider, error) {
	for i := range s.providers {
		if s.providers[i].Active && !s.providers[i].Fallback {
			return &s.providers[i], nil
		}
	}
	for i := range s.providers {
		if s.providers[i].Active && s.providers[i].Fallback {
			return &s.providers[i], nil
		}
	}
	return nil, fmt.Errorf("no LLM provider available")
}

// GenerateSuggestions generates thought connection suggestions for a scheme.
//
// STUB: Currently returns an empty slice to confirm provider fallback logic.
// When implemented, this will:
//  1. Fetch all thoughts+edges for the scheme
//  2. Build a prompt describing the thought graph
//  3. Call the selected LLM provider's chat/completion endpoint
//  4. Parse the response into Suggestion structs with confidence scores
//
// The provider is selected via GetActiveProvider (primary-first, fallback-second).
func (s *SuggestionService) GenerateSuggestions(schemeID string) ([]Suggestion, *LLMProvider, error) {
	provider, err := s.GetActiveProvider()
	if err != nil {
		return nil, nil, err
	}
	return []Suggestion{}, provider, nil
}
