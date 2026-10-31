package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/santhosh-tekuri/jsonschema/v5"
	"github.com/sirupsen/logrus"
	modelai "github.com/vrooli/browser-automation-studio/services/ai"
)

const defaultAIModelRole = "chat.small"

// The provider and parser enforce the same contract. Array responses are wrapped
// at the parser boundary to retain the previously supported response shape.
// Optional text may be absent or null; both decode to the existing empty string.
// Required fields remain non-null and are never synthesized from invalid output.
const suggestionResponseSchema = `{
 "type":"object", "required":["suggestions"], "additionalProperties":false,
 "properties":{"suggestions":{"type":"array","items":{
  "type":"object", "required":["action","confidence","category"],
  "additionalProperties":false,
  "properties":{
   "action":{"type":"string","pattern":"^[\\s\\S]*\\S[\\s\\S]*$"},
   "description":{"type":["string","null"]}, "elementText":{"type":["string","null"]},
   "selector":{"type":["string","null"]}, "reasoning":{"type":["string","null"]},
   "confidence":{"type":"number","minimum":0,"maximum":1},
   "category":{"type":"string","enum":["authentication","navigation","data-entry","actions","content"]}
  }
 }}}
}`

var compiledSuggestionSchema = jsonschema.MustCompileString("bas-suggestions.json", suggestionResponseSchema)

// aiSuggestionGenerator handles AI-powered workflow suggestions through the shared AI model service.
type aiSuggestionGenerator struct {
	client modelai.RolePromptClient
	role   string
}

// AISuggestionOption configures the aiSuggestionGenerator.
type AISuggestionOption func(*aiSuggestionGenerator)

// WithAISuggestionModelClient sets the shared model service.
func WithAISuggestionModelClient(client modelai.RolePromptClient) AISuggestionOption {
	return func(g *aiSuggestionGenerator) {
		g.client = client
	}
}

// WithAIModelRole sets the AI model role to use.
func WithAIModelRole(role string) AISuggestionOption {
	return func(g *aiSuggestionGenerator) {
		g.role = role
	}
}

// newAISuggestionGenerator creates a model-backed suggestion generator.
func newAISuggestionGenerator(log *logrus.Logger, opts ...AISuggestionOption) *aiSuggestionGenerator {
	generator := &aiSuggestionGenerator{
		role: defaultAIModelRole,
	}

	// Apply options first
	for _, opt := range opts {
		opt(generator)
	}

	// Create default client if not provided
	if generator.client == nil {
		generator.client = modelai.NewOpenRouterClient(log)
	}

	return generator
}

// generateAISuggestions uses the shared model service to generate automation suggestions
// based on page elements and context.
func (g *aiSuggestionGenerator) generateAISuggestions(ctx context.Context, elements []ElementInfo, pageContext PageContext) ([]AISuggestion, error) {
	if len(elements) == 0 {
		return []AISuggestion{}, nil
	}
	// Build the structured suggestion prompt.
	prompt := g.buildElementAnalysisPrompt(elements, pageContext)

	// Query the shared model service using the established role.
	suggestionsPayload, err := g.client.ExecutePromptWithRole(ctx, g.role, prompt)
	if err != nil {
		return nil, fmt.Errorf("failed to call AI model service: %w", err)
	}

	payload := strings.TrimSpace(suggestionsPayload)
	if strings.HasPrefix(payload, "[") {
		payload = `{"suggestions":` + payload + `}`
	}
	var value any
	if err := json.Unmarshal([]byte(payload), &value); err != nil {
		return nil, fmt.Errorf("decode AI suggestions: %w", err)
	}
	if err := compiledSuggestionSchema.Validate(value); err != nil {
		return nil, fmt.Errorf("invalid AI suggestions: %w", err)
	}
	var response struct {
		Suggestions []AISuggestion `json:"suggestions"`
	}
	if err := json.Unmarshal([]byte(payload), &response); err != nil {
		return nil, fmt.Errorf("decode validated AI suggestions: %w", err)
	}
	return response.Suggestions, nil
}

// buildElementAnalysisPrompt creates a structured prompt for element analysis.
func (g *aiSuggestionGenerator) buildElementAnalysisPrompt(elements []ElementInfo, pageContext PageContext) string {
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
