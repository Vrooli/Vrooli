package capabilities

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"audio-tools/internal/httpc"
)

// OllamaChecker verifies that Ollama is running by hitting its /api/tags
// endpoint.
type OllamaChecker struct {
	BaseURL string
	Doer    httpc.Doer
	Model   string
	ModelFn func() string
}

func (c *OllamaChecker) Check(ctx context.Context) (Status, string) {
	req, err := http.NewRequestWithContext(ctx, "GET", c.BaseURL+"/api/tags", nil)
	if err != nil {
		return StatusUnavailable, "failed to create request: " + err.Error()
	}

	resp, err := c.Doer.Do(req)
	if err != nil {
		return StatusUnavailable, "Ollama is not responding"
	}
	defer resp.Body.Close()

	model := c.Model
	if c.ModelFn != nil {
		model = c.ModelFn()
	}
	if resp.StatusCode == http.StatusOK && model == "" {
		return StatusAvailable, "Ollama is running"
	}
	if resp.StatusCode == http.StatusOK {
		var tags struct {
			Models []struct {
				Name string `json:"name"`
			} `json:"models"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&tags); err != nil {
			return StatusUnavailable, "Ollama tags response is not valid JSON"
		}
		for _, candidate := range tags.Models {
			if candidate.Name == model {
				return StatusAvailable, fmt.Sprintf("Ollama is running and summarize model %q is available", model)
			}
		}
		return StatusUnavailable, fmt.Sprintf("Ollama is running but summarize model %q is not installed", model)
	}

	return StatusUnavailable, "Ollama returned unexpected status"
}

// OpenRouterChecker verifies that OpenRouter is configured and reachable.
type OpenRouterChecker struct {
	APIKey  string
	BaseURL string
	Doer    httpc.Doer
}

func (c *OpenRouterChecker) Check(ctx context.Context) (Status, string) {
	if c.APIKey == "" {
		return StatusUnavailable, "OPENROUTER_API_KEY not configured"
	}

	baseURL := c.BaseURL
	if baseURL == "" {
		baseURL = "https://openrouter.ai"
	}

	req, err := http.NewRequestWithContext(ctx, "GET", baseURL+"/api/v1/models", nil)
	if err != nil {
		return StatusUnavailable, "failed to create request: " + err.Error()
	}
	req.Header.Set("Authorization", "Bearer "+c.APIKey)

	resp, err := c.Doer.Do(req)
	if err != nil {
		return StatusUnavailable, "OpenRouter is not reachable"
	}
	resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		return StatusAvailable, "OpenRouter is configured and reachable"
	}
	if resp.StatusCode == http.StatusUnauthorized {
		return StatusUnavailable, "OpenRouter API key is invalid"
	}

	return StatusUnavailable, "OpenRouter returned unexpected status"
}
