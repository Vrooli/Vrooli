package deepsearch

// DOC: docs/reference/configuration.md#api-runtime-configuration
import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// OllamaParser uses Ollama to coerce unstructured output into JSON results.
type OllamaParser struct {
	BaseURL string
	Model   string
	Client  *http.Client
}

type ollamaGenerateRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
	Stream bool   `json:"stream"`
}

type ollamaGenerateResponse struct {
	Response string `json:"response"`
}

func (o *OllamaParser) Parse(ctx context.Context, raw string) ([]DeepSearchResult, error) {
	baseURL := strings.TrimRight(strings.TrimSpace(o.BaseURL), "/")
	model := strings.TrimSpace(o.Model)
	if baseURL == "" || model == "" {
		return nil, fmt.Errorf("ollama parser not configured")
	}
	prompt := buildOllamaPrompt(raw)
	payload := ollamaGenerateRequest{
		Model:  model,
		Prompt: prompt,
		Stream: false,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal ollama request: %w", err)
	}
	client := o.Client
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Second}
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/api/generate", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create ollama request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("ollama request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		rawBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("ollama returned status %d: %s", resp.StatusCode, strings.TrimSpace(string(rawBody)))
	}

	var decoded ollamaGenerateResponse
	if err := json.NewDecoder(resp.Body).Decode(&decoded); err != nil {
		return nil, fmt.Errorf("failed to decode ollama response: %w", err)
	}

	parsed, ok := parseJSONResults(decoded.Response)
	if !ok {
		return nil, fmt.Errorf("ollama response did not contain valid JSON")
	}
	return parsed, nil
}

func buildOllamaPrompt(raw string) string {
	return strings.TrimSpace(fmt.Sprintf(`Convert the following agent output into JSON only.

Return a JSON array of objects with fields:
- path (string)
- relevance (number 0-1)
- summary (string)
- match_reason (string)
- references (array of strings)
- snippet (string)

If a field is missing, use an empty string or empty array. Return JSON only.

OUTPUT:
%s`, raw))
}
