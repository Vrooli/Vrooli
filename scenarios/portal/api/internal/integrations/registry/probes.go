package registry

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/vrooli/api-core/discovery"

	"portal/internal/integrations/openrouter"
)

type Probe interface {
	Probe(ctx context.Context) ProbeResult
}

type ProbeFunc func(ctx context.Context) ProbeResult

func (f ProbeFunc) Probe(ctx context.Context) ProbeResult {
	return f(ctx)
}

type HTTPDoer interface {
	Do(req *http.Request) (*http.Response, error)
}

type ScenarioProbe struct {
	Slug   string
	EnvVar string
	Client HTTPDoer
}

func (p ScenarioProbe) Probe(ctx context.Context) ProbeResult {
	baseURL, err := p.resolveBaseURL(ctx)
	if err != nil {
		return ProbeResult{OK: false, Reason: err.Error()}
	}
	client := p.Client
	if client == nil {
		client = &http.Client{Timeout: 2 * time.Second}
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(baseURL, "/")+"/health", nil)
	if err != nil {
		return ProbeResult{OK: false, Reason: err.Error()}
	}
	resp, err := client.Do(req)
	if err != nil {
		return ProbeResult{OK: false, Reason: err.Error()}
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return ProbeResult{OK: false, Reason: fmt.Sprintf("health returned HTTP %d", resp.StatusCode)}
	}
	return ProbeResult{OK: true}
}

func (p ScenarioProbe) resolveBaseURL(ctx context.Context) (string, error) {
	if p.EnvVar != "" {
		if value := strings.TrimSpace(os.Getenv(p.EnvVar)); value != "" {
			return strings.TrimRight(value, "/"), nil
		}
	}
	slug := strings.TrimSpace(p.Slug)
	if slug == "" {
		return "", fmt.Errorf("scenario slug is required")
	}
	return discovery.ResolveScenarioURLDefault(ctx, slug)
}

type OpenRouterProbe struct {
	Client HTTPDoer
}

func (p OpenRouterProbe) Probe(ctx context.Context) ProbeResult {
	cfg, err := openrouter.ResolveConfig()
	if err != nil {
		return ProbeResult{OK: false, Reason: err.Error()}
	}
	baseURL := strings.TrimRight(strings.TrimSpace(cfg.BaseURL), "/")
	if baseURL == "" {
		baseURL = "https://openrouter.ai/api/v1"
	}
	client := p.Client
	if client == nil {
		client = &http.Client{Timeout: 3 * time.Second}
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+"/models", nil)
	if err != nil {
		return ProbeResult{OK: false, Reason: err.Error()}
	}
	req.Header.Set("Authorization", "Bearer "+cfg.APIKey)
	resp, err := client.Do(req)
	if err != nil {
		return ProbeResult{OK: false, Reason: err.Error()}
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return ProbeResult{OK: false, Reason: "OpenRouter rejected the configured API key"}
	}
	if resp.StatusCode >= 500 {
		return ProbeResult{OK: false, Degraded: true, Reason: fmt.Sprintf("OpenRouter returned HTTP %d", resp.StatusCode)}
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return ProbeResult{OK: false, Reason: fmt.Sprintf("OpenRouter returned HTTP %d", resp.StatusCode)}
	}
	return ProbeResult{OK: true}
}

func DefaultProbes() map[IntegrationID]Probe {
	return map[IntegrationID]Probe{
		IntegrationSearchHub:    ScenarioProbe{Slug: "search-hub", EnvVar: "SEARCH_HUB_API_URL"},
		IntegrationOpenRouter:   OpenRouterProbe{},
		IntegrationAgentManager: ScenarioProbe{Slug: "agent-manager", EnvVar: "AGENT_MANAGER_API_URL"},
		IntegrationPromptMgr:    ScenarioProbe{Slug: "prompt-manager", EnvVar: "PROMPT_MANAGER_API_URL"},
	}
}
