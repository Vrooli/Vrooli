package testgenie

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/vrooli/api-core/discovery"
)

const (
	defaultScenarioName = "test-genie"
	defaultTimeout      = 30 * time.Second
)

// SeedApplyResponse is returned by test-genie seed apply.
type SeedApplyResponse struct {
	Status       string                 `json:"status"`
	Scenario     string                 `json:"scenario"`
	RunID        string                 `json:"run_id"`
	SeedState    map[string]any         `json:"seed_state"`
	CleanupToken string                 `json:"cleanup_token"`
	Resources    []map[string]any       `json:"resources"`
}

// SeedCleanupResponse is returned by test-genie seed cleanup.
type SeedCleanupResponse struct {
	Status   string `json:"status"`
	Scenario string `json:"scenario"`
	RunID    string `json:"run_id"`
}

// Client calls test-genie seed lifecycle endpoints.
type Client struct {
	resolver   *discovery.Resolver
	httpClient *http.Client
}

// NewClient creates a test-genie client using api-core discovery.
func NewClient(resolver *discovery.Resolver, httpClient *http.Client) *Client {
	if resolver == nil {
		resolver = discovery.NewResolver(discovery.ResolverConfig{})
	}
	if httpClient == nil {
		httpClient = &http.Client{Timeout: defaultTimeout}
	}
	return &Client{
		resolver:   resolver,
		httpClient: httpClient,
	}
}

// ApplySeed runs the test-genie seed lifecycle for the scenario.
func (c *Client) ApplySeed(ctx context.Context, scenario string, retain bool) (*SeedApplyResponse, error) {
	baseURL, err := c.resolveBaseURL(ctx)
	if err != nil {
		return nil, err
	}
	payload := map[string]any{"retain": retain}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("encode seed apply: %w", err)
	}
	path := fmt.Sprintf("%s/scenarios/%s/playbooks/seed/apply", baseURL, url.PathEscape(scenario))
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, path, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("build seed apply request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("seed apply request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("seed apply failed (%s): %s", resp.Status, strings.TrimSpace(string(respBody)))
	}

	var parsed SeedApplyResponse
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return nil, fmt.Errorf("parse seed apply response: %w", err)
	}
	return &parsed, nil
}

// CleanupSeed clears isolation resources for a prior seed session.
func (c *Client) CleanupSeed(ctx context.Context, scenario, cleanupToken string) (*SeedCleanupResponse, error) {
	baseURL, err := c.resolveBaseURL(ctx)
	if err != nil {
		return nil, err
	}
	payload := map[string]any{"cleanup_token": cleanupToken}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("encode seed cleanup: %w", err)
	}
	path := fmt.Sprintf("%s/scenarios/%s/playbooks/seed/cleanup", baseURL, url.PathEscape(scenario))
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, path, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("build seed cleanup request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("seed cleanup request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("seed cleanup failed (%s): %s", resp.Status, strings.TrimSpace(string(respBody)))
	}

	var parsed SeedCleanupResponse
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return nil, fmt.Errorf("parse seed cleanup response: %w", err)
	}
	return &parsed, nil
}

func (c *Client) resolveBaseURL(ctx context.Context) (string, error) {
	url, err := c.resolver.ResolveScenarioURLDefault(ctx, defaultScenarioName)
	if err != nil {
		return "", err
	}
	return strings.TrimRight(url, "/") + "/api/v1", nil
}
