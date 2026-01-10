package recommendation

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"agent-manager/internal/domain"

	"github.com/google/uuid"
)

// OllamaExtractor uses the resource-ollama CLI to extract recommendations.
type OllamaExtractor struct {
	model       string
	temperature string
	maxTokens   string
}

// NewOllamaExtractor creates a new extractor using resource-ollama CLI.
func NewOllamaExtractor() *OllamaExtractor {
	// Use environment variable for model if set, otherwise default to qwen2.5-coder:14b
	model := os.Getenv("OLLAMA_RECOMMENDATION_MODEL")
	if model == "" {
		model = "qwen2.5-coder:14b"
	}
	return &OllamaExtractor{
		model:       model,
		temperature: "0.1",
		maxTokens:   "2000",
	}
}

// extractedData represents the structured JSON we expect from the LLM.
type extractedData struct {
	Categories []struct {
		Name            string `json:"name"`
		Recommendations []struct {
			Text string `json:"text"`
		} `json:"recommendations"`
	} `json:"categories"`
}

// Extract parses investigation text and returns categorized recommendations.
func (e *OllamaExtractor) Extract(ctx context.Context, req domain.ExtractionRequest) (*domain.ExtractionResult, error) {
	// Build extraction prompt
	prompt := e.buildPrompt(req.InvestigationText)

	// Execute: resource-ollama query --model X --json "prompt"
	// Note: temperature and max_tokens are set via environment variables
	cmd := exec.CommandContext(ctx, "resource-ollama", "query",
		"--model", e.model,
		"--json",
		"--", // Use -- to separate flags from prompt (which may start with -)
		prompt,
	)

	// Set environment variables for temperature and max_tokens
	cmd.Env = append(os.Environ(),
		"TEMPERATURE="+e.temperature,
		"MAX_TOKENS="+e.maxTokens,
	)

	output, err := cmd.Output()
	if err != nil {
		// Try to get stderr for more details
		errMsg := fmt.Sprintf("ollama extraction failed: %v", err)
		if exitErr, ok := err.(*exec.ExitError); ok && len(exitErr.Stderr) > 0 {
			errMsg = fmt.Sprintf("ollama extraction failed: %s", string(exitErr.Stderr))
		}
		return &domain.ExtractionResult{
			Success:       false,
			RawText:       req.InvestigationText,
			ExtractedFrom: "summary",
			Error:         errMsg,
		}, nil
	}

	// The output from resource-ollama query is raw text (the LLM's response)
	responseText := strings.TrimSpace(string(output))

	// Try to find JSON in the response (it might be wrapped in markdown code blocks)
	jsonStr := extractJSON(responseText)
	if jsonStr == "" {
		return &domain.ExtractionResult{
			Success:       false,
			RawText:       req.InvestigationText,
			ExtractedFrom: "summary",
			Error:         "no valid JSON found in LLM response",
		}, nil
	}

	// Parse the extracted JSON
	var data extractedData
	if err := json.Unmarshal([]byte(jsonStr), &data); err != nil {
		return &domain.ExtractionResult{
			Success:       false,
			RawText:       req.InvestigationText,
			ExtractedFrom: "summary",
			Error:         fmt.Sprintf("failed to parse extracted JSON: %v", err),
		}, nil
	}

	// Convert to domain types with generated UUIDs
	categories := make([]domain.RecommendationCategory, 0, len(data.Categories))
	for _, cat := range data.Categories {
		recs := make([]domain.Recommendation, 0, len(cat.Recommendations))
		for _, rec := range cat.Recommendations {
			text := strings.TrimSpace(rec.Text)
			if text == "" {
				continue
			}
			recs = append(recs, domain.Recommendation{
				ID:       uuid.New().String(),
				Text:     text,
				Selected: true,
			})
		}
		if len(recs) == 0 {
			continue
		}
		categories = append(categories, domain.RecommendationCategory{
			ID:              uuid.New().String(),
			Name:            strings.TrimSpace(cat.Name),
			Recommendations: recs,
		})
	}

	return &domain.ExtractionResult{
		Success:       true,
		Categories:    categories,
		RawText:       req.InvestigationText,
		ExtractedFrom: "summary",
	}, nil
}

// IsAvailable checks if resource-ollama is accessible.
func (e *OllamaExtractor) IsAvailable(ctx context.Context) bool {
	cmd := exec.CommandContext(ctx, "resource-ollama", "status")
	return cmd.Run() == nil
}

// buildPrompt constructs the extraction prompt for the LLM.
func (e *OllamaExtractor) buildPrompt(text string) string {
	return `You are an expert at analyzing investigation reports and extracting actionable recommendations.

Given the following investigation output, extract all recommendations into structured JSON.
Group recommendations by natural categories discovered in the text.

Output ONLY valid JSON (no markdown, no explanation):
{
  "categories": [
    {"name": "Category Name", "recommendations": [{"text": "Specific recommendation"}]}
  ]
}

Rules:
1. Extract ONLY actionable items (things to do)
2. Preserve original wording where possible
3. Group under logical category names discovered in the text
4. If no categories exist, use a single "Recommendations" category
5. Do NOT include background context or explanations

Investigation Output:
---
` + text + `
---`
}

// extractJSON attempts to find valid JSON in the response text.
// It handles cases where JSON is wrapped in markdown code blocks.
func extractJSON(text string) string {
	text = strings.TrimSpace(text)

	// Try to find JSON wrapped in code blocks
	if start := strings.Index(text, "```json"); start != -1 {
		text = text[start+7:]
		if end := strings.Index(text, "```"); end != -1 {
			text = text[:end]
		}
	} else if start := strings.Index(text, "```"); start != -1 {
		text = text[start+3:]
		if end := strings.Index(text, "```"); end != -1 {
			text = text[:end]
		}
	}

	text = strings.TrimSpace(text)

	// Verify it's valid JSON starting with {
	if strings.HasPrefix(text, "{") {
		return text
	}

	// Try to extract JSON object from anywhere in the text
	if start := strings.Index(text, "{"); start != -1 {
		depth := 0
		for i := start; i < len(text); i++ {
			switch text[i] {
			case '{':
				depth++
			case '}':
				depth--
				if depth == 0 {
					return text[start : i+1]
				}
			}
		}
	}

	return ""
}
