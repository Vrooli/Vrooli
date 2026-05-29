package prompt_manager

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"development-toolchain-validator/internal/httpc"
	vrun "development-toolchain-validator/internal/validation_run"

	"github.com/vrooli/api-core/discovery"
)

// skillGetPath is the prompt-manager REST endpoint returning a single
// skill with its full markdown content.
const skillGetPath = "/api/v1/skills/"

// SkillContentRESTAdapter implements validation_run.SkillContentSource by
// calling prompt-manager's GET /api/v1/skills/{id} endpoint. It returns
// the skill's markdown body, which the validation_run worker injects as
// the sandboxed agent's prompt so the agent executes the actual skill.
type SkillContentRESTAdapter struct {
	resolver    *discovery.Resolver
	doer        httpc.Doer
	maxAttempts int
}

// NewSkillContentRESTAdapter constructs the production adapter. Empty
// option fields fall back to the same defaults as the catalog adapter.
func NewSkillContentRESTAdapter(opts Options) *SkillContentRESTAdapter {
	a := &SkillContentRESTAdapter{
		resolver:    opts.Resolver,
		doer:        opts.Doer,
		maxAttempts: opts.MaxAttempts,
	}
	if a.resolver == nil {
		a.resolver = discovery.NewResolver(discovery.ResolverConfig{})
	}
	if a.doer == nil {
		a.doer = &http.Client{Timeout: 30 * time.Second}
	}
	if a.maxAttempts <= 0 {
		a.maxAttempts = 3
	}
	return a
}

var _ vrun.SkillContentSource = (*SkillContentRESTAdapter)(nil)

// SkillContent resolves prompt-manager's base URL via discovery and GETs
// the skill by id, returning its markdown content. Re-resolves the base
// URL on transport failure to survive scenario restart cycles.
func (a *SkillContentRESTAdapter) SkillContent(ctx context.Context, skillID string) (string, error) {
	id := strings.TrimSpace(skillID)
	if id == "" {
		return "", fmt.Errorf("skill id is empty")
	}
	var lastErr error
	for attempt := 0; attempt < a.maxAttempts; attempt++ {
		base, err := a.resolver.ResolveScenarioURLDefault(ctx, promptManagerScenario)
		if err != nil {
			if discovery.IsScenarioNotRunning(err) {
				return "", fmt.Errorf("prompt-manager is not running: %w", err)
			}
			lastErr = err
			continue
		}
		content, err := a.fetchOnce(ctx, base, id)
		if err == nil {
			return content, nil
		}
		lastErr = err
		if !isRetriable(err) {
			return "", err
		}
	}
	if lastErr == nil {
		lastErr = errors.New("exhausted retries")
	}
	return "", fmt.Errorf("fetch skill %q content: %w", id, lastErr)
}

func (a *SkillContentRESTAdapter) fetchOnce(ctx context.Context, base, id string) (string, error) {
	u := strings.TrimRight(base, "/") + skillGetPath + url.PathEscape(id)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return "", fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	resp, err := a.doer.Do(req)
	if err != nil {
		return "", fmt.Errorf("get %s: %w", u, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 500 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<14))
		return "", transportError{status: resp.StatusCode, snippet: strings.TrimSpace(string(body))}
	}
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<14))
		return "", fmt.Errorf("prompt-manager returned %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	var payload skillContentPayload
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return "", fmt.Errorf("decode response: %w", err)
	}
	content := strings.TrimSpace(payload.Content)
	if content == "" {
		return "", fmt.Errorf("skill %q returned empty content", id)
	}
	return payload.Content, nil
}

// skillContentPayload mirrors the fields we need from prompt-manager's
// single-skill response; unknown fields are tolerated.
type skillContentPayload struct {
	Content string `json:"content"`
}
