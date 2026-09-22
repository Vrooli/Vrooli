package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/santhosh-tekuri/jsonschema/v5"
	"github.com/sirupsen/logrus"
)

const defaultOllamaRole = "chat.small"

// The provider and parser enforce the same contract. Array responses are wrapped
// at the parser boundary to retain the previously supported response shape.
const suggestionResponseSchema = `{
 "type":"object", "required":["suggestions"], "additionalProperties":false,
 "properties":{"suggestions":{"type":"array","items":{
  "type":"object", "required":["action","confidence","category"],
  "additionalProperties":false,
  "properties":{
   "action":{"type":"string","pattern":"^[\\s\\S]*\\S[\\s\\S]*$"},
   "description":{"type":"string"}, "elementText":{"type":"string"},
   "selector":{"type":"string"}, "reasoning":{"type":"string"},
   "confidence":{"type":"number","minimum":0,"maximum":1},
   "category":{"type":"string","enum":["authentication","navigation","data-entry","actions","content"]}
  }
 }}}
}`

var compiledSuggestionSchema = jsonschema.MustCompileString("bas-suggestions.json", suggestionResponseSchema)

// ollamaSuggestionGenerator handles AI-powered workflow suggestions using Ollama.
type ollamaSuggestionGenerator struct {
	client OllamaClient
	role   string
}

// OllamaSuggestionOption configures the ollamaSuggestionGenerator.
type OllamaSuggestionOption func(*ollamaSuggestionGenerator)

// WithOllamaClient sets a custom Ollama client.
func WithOllamaClient(client OllamaClient) OllamaSuggestionOption {
	return func(g *ollamaSuggestionGenerator) {
		g.client = client
	}
}

// WithOllamaRole sets the Ollama role to use.
func WithOllamaRole(role string) OllamaSuggestionOption {
	return func(g *ollamaSuggestionGenerator) {
		g.role = role
	}
}

// newOllamaSuggestionGenerator creates a new Ollama suggestion generator.
func newOllamaSuggestionGenerator(log *logrus.Logger, opts ...OllamaSuggestionOption) *ollamaSuggestionGenerator {
	generator := &ollamaSuggestionGenerator{
		role: defaultOllamaRole,
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
	if len(elements) == 0 {
		return []AISuggestion{}, nil
	}
	// Build prompt for Ollama
	prompt := g.buildElementAnalysisPrompt(elements, pageContext)

	// Query Ollama via the client interface
	suggestionsPayload, err := g.client.Query(ctx, g.role, prompt, suggestionResponseSchema)
	if err != nil {
		return nil, fmt.Errorf("failed to call ollama: %w", err)
	}

	payload := strings.TrimSpace(suggestionsPayload)
	if strings.HasPrefix(payload, "[") {
		payload = `{"suggestions":` + payload + `}`
	}
	var value any
	if err := json.Unmarshal([]byte(payload), &value); err != nil {
		return nil, fmt.Errorf("decode Ollama suggestions: %w", err)
	}
	if err := compiledSuggestionSchema.Validate(value); err != nil {
		return nil, fmt.Errorf("invalid Ollama suggestions: %w", err)
	}
	var response struct {
		Suggestions []AISuggestion `json:"suggestions"`
	}
	if err := json.Unmarshal([]byte(payload), &response); err != nil {
		return nil, fmt.Errorf("decode validated Ollama suggestions: %w", err)
	}
	return response.Suggestions, nil
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
