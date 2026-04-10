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

	// Sanitize JSON to fix common LLM issues (unescaped newlines in strings)
	jsonStr = sanitizeJSONString(jsonStr)

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
6. Return valid JSON only - no markdown code blocks, no extra text

## Examples

### Example 1
Input:
---
The agent failed because it tried to read a 50MB log file without checking the size first.

Recommendations:
- Add a file size check before reading files
- Set a maximum file size limit of 1MB
- Log a warning when files are skipped due to size
---

Output:
{"categories": [{"name": "File Handling", "recommendations": [{"text": "Add a file size check before reading files"}, {"text": "Set a maximum file size limit of 1MB"}, {"text": "Log a warning when files are skipped due to size"}]}]}

### Example 2
Input:
---
## Behavioral Analysis
The agent got stuck in an infinite loop trying to fix a syntax error.

## Recommendations
### Prompt Changes
- Add explicit instruction to limit retry attempts to 3
- Include example of when to give up and report failure

### Guardrails
- Implement loop detection in the runner
- Add timeout for individual operations
---

Output:
{"categories": [{"name": "Prompt Changes", "recommendations": [{"text": "Add explicit instruction to limit retry attempts to 3"}, {"text": "Include example of when to give up and report failure"}]}, {"name": "Guardrails", "recommendations": [{"text": "Implement loop detection in the runner"}, {"text": "Add timeout for individual operations"}]}]}

### Example 3 (No clear categories)
Input:
---
The investigation found that the agent should validate inputs, add error handling, and improve logging.
---

Output:
{"categories": [{"name": "Recommendations", "recommendations": [{"text": "Validate inputs"}, {"text": "Add error handling"}, {"text": "Improve logging"}]}]}

## Now extract from this investigation:

Investigation Output:
---
` + text + `
---`
}

// extractJSON attempts to find valid JSON in the response text.
// It handles cases where JSON is wrapped in markdown code blocks and
// properly isolates the JSON object even with surrounding text.
// It correctly handles braces inside JSON strings.
func extractJSON(text string) string {
	text = strings.TrimSpace(text)
	if text == "" {
		return ""
	}

	// Try to find JSON wrapped in code blocks first (most reliable)
	if start := strings.Index(text, "```json"); start != -1 {
		text = text[start+7:]
		if end := strings.Index(text, "```"); end != -1 {
			text = text[:end]
		}
		text = strings.TrimSpace(text)
	} else if start := strings.Index(text, "```"); start != -1 {
		text = text[start+3:]
		if end := strings.Index(text, "```"); end != -1 {
			text = text[:end]
		}
		text = strings.TrimSpace(text)
	}

	// Find the first { to start JSON extraction
	start := strings.Index(text, "{")
	if start == -1 {
		return ""
	}

	// Extract JSON object with proper string handling
	// This correctly ignores braces inside JSON strings
	depth := 0
	inString := false
	escaped := false

	for i := start; i < len(text); i++ {
		c := text[i]

		if escaped {
			// Previous character was a backslash, skip this character
			escaped = false
			continue
		}

		if c == '\\' && inString {
			// Start of escape sequence inside string
			escaped = true
			continue
		}

		if c == '"' {
			// Toggle string state
			inString = !inString
			continue
		}

		if !inString {
			// Only count braces outside of strings
			switch c {
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

	// Unclosed JSON object - return empty
	return ""
}

// sanitizeJSONString escapes unescaped control characters in JSON string values.
// LLMs often produce JSON with literal newlines inside strings instead of \n.
func sanitizeJSONString(jsonStr string) string {
	if jsonStr == "" {
		return jsonStr
	}

	var result strings.Builder
	result.Grow(len(jsonStr) + 100) // Pre-allocate with some extra space

	inString := false
	escaped := false

	for i := 0; i < len(jsonStr); i++ {
		c := jsonStr[i]

		if escaped {
			// Previous character was a backslash, this is an escape sequence
			result.WriteByte(c)
			escaped = false
			continue
		}

		if c == '\\' && inString {
			// Start of escape sequence
			result.WriteByte(c)
			escaped = true
			continue
		}

		if c == '"' {
			// Toggle string state (unless escaped, handled above)
			inString = !inString
			result.WriteByte(c)
			continue
		}

		if inString {
			// Inside a string - escape control characters
			switch c {
			case '\n':
				result.WriteString("\\n")
			case '\r':
				result.WriteString("\\r")
			case '\t':
				result.WriteString("\\t")
			default:
				if c < 32 {
					// Escape other control characters as \uXXXX
					result.WriteString(fmt.Sprintf("\\u%04x", c))
				} else {
					result.WriteByte(c)
				}
			}
		} else {
			// Outside string - keep as-is (whitespace between elements is fine)
			result.WriteByte(c)
		}
	}

	return result.String()
}
