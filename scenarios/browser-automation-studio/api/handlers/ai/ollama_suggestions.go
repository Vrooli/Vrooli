package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"

	"github.com/sirupsen/logrus"
)

// ollamaSuggestionGenerator handles AI-powered workflow suggestions using Ollama.
type ollamaSuggestionGenerator struct {
	log *logrus.Logger
}

// newOllamaSuggestionGenerator creates a new Ollama suggestion generator.
func newOllamaSuggestionGenerator(log *logrus.Logger) *ollamaSuggestionGenerator {
	return &ollamaSuggestionGenerator{log: log}
}

// generateAISuggestions uses Ollama to generate intelligent automation suggestions
// based on page elements and context.
func (g *ollamaSuggestionGenerator) generateAISuggestions(ctx context.Context, elements []ElementInfo, pageContext PageContext) ([]AISuggestion, error) {
	// Build prompt for Ollama
	prompt := g.buildElementAnalysisPrompt(elements, pageContext)

	// Create temporary file for Ollama output
	tmpOllamaFile, err := os.CreateTemp("", "ollama-suggestions-*.json")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp ollama file: %w", err)
	}
	defer os.Remove(tmpOllamaFile.Name())
	defer tmpOllamaFile.Close()

	// Call resource-ollama using the query command
	ollamaCmd := exec.CommandContext(ctx, "resource-ollama", "query",
		"--model", "llama3.2",
		"--prompt", prompt)

	output, err := ollamaCmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("failed to call ollama: %w, output: %s", err, string(output))
	}

	// Write ollama output to temp file for parsing
	if err := os.WriteFile(tmpOllamaFile.Name(), output, 0600); err != nil {
		return nil, fmt.Errorf("failed to write ollama output: %w", err)
	}

	// Read and parse Ollama response
	ollamaData, err := os.ReadFile(tmpOllamaFile.Name())
	if err != nil {
		return nil, fmt.Errorf("failed to read ollama output: %w", err)
	}

	var ollamaResponse struct {
		Suggestions []AISuggestion `json:"suggestions"`
	}

	if err := json.Unmarshal(ollamaData, &ollamaResponse); err != nil {
		// If JSON parsing fails, try to extract from the raw response
		g.log.WithError(err).Warn("Failed to parse Ollama JSON response, trying fallback parsing")
		return g.parseOllamaFallback(string(ollamaData))
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
