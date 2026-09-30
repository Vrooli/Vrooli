package capabilities

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"audio-tools/internal/httpc"
	"audio-tools/internal/summarize"
)

// OllamaChecker verifies that Ollama is running by hitting its /api/tags
// endpoint and that the configured summarize model is installed.
//
// The configured summarize selector may name a logical policy role (e.g.
// "summarize.default") rather than a concrete Ollama tag. Roles are resolved
// to a physical model by resource-ollama's policy SSOT — the same authority
// the summarizer's gateway path uses at chat time. ResolveRole is the seam
// for that resolution; production leaves it nil (defaulting to the shell-based
// resolver) and tests inject a fake to stay hermetic.
type OllamaChecker struct {
	BaseURL     string
	Doer        httpc.Doer
	Model       string
	ModelFn     func() string
	ResolveRole func(ctx context.Context, role string) (string, error)
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

	if resp.StatusCode != http.StatusOK {
		return StatusUnavailable, "Ollama returned unexpected status"
	}

	model := c.Model
	if c.ModelFn != nil {
		model = c.ModelFn()
	}
	if model == "" {
		return StatusAvailable, "Ollama is running"
	}

	var tags struct {
		Models []struct {
			Name string `json:"name"`
		} `json:"models"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tags); err != nil {
		return StatusUnavailable, "Ollama tags response is not valid JSON"
	}
	installed := func(name string) bool {
		for _, candidate := range tags.Models {
			if candidate.Name == name {
				return true
			}
		}
		return false
	}

	// Fast path: the selector is already a concrete tag that is installed.
	// This also keeps a literally-named model working even without a colon.
	if installed(model) {
		return StatusAvailable, fmt.Sprintf("Ollama is running and summarize model %q is available", model)
	}

	// A logical role (e.g. "summarize.default") never appears verbatim in
	// /api/tags. Resolve it through the policy SSOT to the physical model the
	// summarizer would actually run, then verify that is installed — otherwise
	// we would compare a role name against tags and always report "not
	// installed", flipping the provider rollup to degraded while ollama is
	// healthy.
	if summarize.SelectorIsRole(model) {
		resolve := c.ResolveRole
		if resolve == nil {
			resolve = summarize.ResolveRoleModel
		}
		resolved, rerr := resolve(ctx, model)
		if rerr != nil {
			return StatusUnavailable, fmt.Sprintf("Ollama is running but summarize role %q could not be resolved: %v", model, rerr)
		}
		if installed(resolved) {
			return StatusAvailable, fmt.Sprintf("Ollama is running and summarize model %q (role %q) is available", resolved, model)
		}
		return StatusUnavailable, fmt.Sprintf("Ollama is running but summarize model %q (role %q) is not installed", resolved, model)
	}

	return StatusUnavailable, fmt.Sprintf("Ollama is running but summarize model %q is not installed", model)
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
