package ai

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/sirupsen/logrus"
)

const defaultOllamaModel = "llama3.2"

// ollamaSuggestionGenerator handles AI-powered workflow suggestions using Ollama.
type ollamaSuggestionGenerator struct {
	log    *logrus.Logger
	client OllamaClient
	model  string
}

// OllamaSuggestionOption configures the ollamaSuggestionGenerator.
type OllamaSuggestionOption func(*ollamaSuggestionGenerator)

// WithOllamaClient sets a custom Ollama client.
func WithOllamaClient(client OllamaClient) OllamaSuggestionOption {
	return func(g *ollamaSuggestionGenerator) {
		g.client = client
	}
}

// WithOllamaModel sets the Ollama model to use.
func WithOllamaModel(model string) OllamaSuggestionOption {
	return func(g *ollamaSuggestionGenerator) {
		g.model = model
	}
}

// newOllamaSuggestionGenerator creates a new Ollama suggestion generator.
func newOllamaSuggestionGenerator(log *logrus.Logger, opts ...OllamaSuggestionOption) *ollamaSuggestionGenerator {
	generator := &ollamaSuggestionGenerator{
		log:   log,
		model: defaultOllamaModel,
	}

	// Apply options first
	for _, opt := range opts {
		opt(generator)
	}

	// Create default client if not provided
	if generator.client == nil {
		generator.client = NewDefaultOllamaClient(log)
	}

	return generator
}

// generateAISuggestions uses Ollama to generate intelligent automation suggestions
// based on page elements and context.
func (g *ollamaSuggestionGenerator) generateAISuggestions(ctx context.Context, elements []ElementInfo, pageContext PageContext) ([]AISuggestion, error) {
	// Build prompt for Ollama
	prompt := g.buildElementAnalysisPrompt(elements, pageContext)

	// Query Ollama via the client interface
	response, err := g.client.Query(ctx, g.model, prompt)
	if err != nil {
		return nil, fmt.Errorf("failed to call ollama: %w", err)
	}

	// Parse the response
	var ollamaResponse struct {
		Suggestions []AISuggestion `json:"suggestions"`
	}

	if err := json.Unmarshal([]byte(response), &ollamaResponse); err != nil {
		// If JSON parsing fails, try to extract from the raw response
		g.log.WithError(err).Warn("Failed to parse Ollama JSON response, trying fallback parsing")
		return g.parseOllamaFallback(response)
	}

	return ollamaResponse.Suggestions, nil
}

// buildElementAnalysisPrompt creates a structured prompt for element analysis.
func (g *ollamaSuggestionGenerator) buildElementAnalysisPrompt(elements []ElementInfo, pageContext PageContext) string {
	// Build elements summary
	elementsJSON, _ := json.MarshalIndent(elements, "  ", "  ")
	contextJSON, _ := json.MarshalIndent(pageContext, "  ", "  ")

	return fmt.Sprintf(`Analyze this webpage and suggest the most likely automation actions a user would want to perform.

Page Information:
%s

Available Interactive Elements:
%s

Provide suggestions in this exact JSON format:
{
  "suggestions": [
    {
      "action": "Login to account",
      "description": "Click the login button to authenticate user",
      "elementText": "Login",
      "selector": "#login-btn",
      "confidence": 0.95,
      "category": "authentication",
      "reasoning": "Page has password input and login button, indicating authentication workflow"
    }
  ]
}

Categories to use:
- "authentication": Login, logout, register, password reset
- "navigation": Menu items, page links, tabs, breadcrumbs
- "data-entry": Forms, search, input fields, text areas
- "actions": Submit, save, delete, edit, download buttons
- "content": Read more, expand, filter, sort options

Focus on:
1. Common user workflows and practical automation scenarios
2. Element text and semantic meaning over position
3. Rank by likelihood of user intent (0.0 to 1.0 confidence)
4. Provide clear reasoning for each suggestion
5. Prefer robust selectors (ID > data attributes > semantic classes)

Return only valid JSON without additional text.`, string(contextJSON), string(elementsJSON))
}

// parseOllamaFallback attempts to extract suggestions from malformed Ollama response.
func (g *ollamaSuggestionGenerator) parseOllamaFallback(response string) ([]AISuggestion, error) {
	g.log.WithField("response", response).Debug("Attempting fallback parsing of Ollama response")

	// For now, return empty suggestions if we can't parse
	// In a real implementation, you might try regex extraction or other parsing strategies
	return []AISuggestion{}, nil
}
