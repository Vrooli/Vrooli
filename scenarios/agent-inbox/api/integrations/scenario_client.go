// Package integrations provides clients for external services.
//
// This file implements the ScenarioClient for discovering tools from
// other scenarios via the Tool Discovery Protocol.
//
// ARCHITECTURE:
// - ScenarioClient: HTTP client for fetching tool manifests
// - Discovery: URL resolution via environment or vrooli CLI
// - Caching: TTL-based caching to reduce network traffic
//
// TESTING SEAMS:
// - HTTPClient interface for mocking HTTP calls
// - URLResolver interface for mocking scenario discovery
package integrations

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"agent-inbox/config"
	"agent-inbox/domain"

	toolspb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-inbox/v1/domain"
	"google.golang.org/protobuf/encoding/protojson"
)

// HTTPClient defines the interface for HTTP operations.
// This enables testing without real network calls.
type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// URLResolver resolves scenario names to API base URLs.
// This enables testing without real service discovery.
type URLResolver interface {
	ResolveScenarioURL(ctx context.Context, scenarioName string) (string, error)
}

// ScenarioClient fetches tool manifests from external scenarios.
type ScenarioClient struct {
	httpClient  HTTPClient
	urlResolver URLResolver
	timeout     time.Duration
	cacheTTL    time.Duration

	// Cache for manifests
	mu    sync.RWMutex
	cache map[string]*cachedManifest
}

// cachedManifest stores a manifest with its fetch time.
type cachedManifest struct {
	manifest  *toolspb.ToolManifest
	fetchedAt time.Time
}

// ScenarioClientConfig holds configuration for creating a ScenarioClient.
type ScenarioClientConfig struct {
	Timeout  time.Duration
	CacheTTL time.Duration
}

// NewScenarioClient creates a new ScenarioClient with default configuration.
func NewScenarioClient() *ScenarioClient {
	cfg := config.Default()
	return NewScenarioClientWithConfig(ScenarioClientConfig{
		Timeout:  cfg.Integration.ToolDiscovery.DiscoveryTimeout,
		CacheTTL: cfg.Integration.ToolDiscovery.CacheTTL,
	})
}

// NewScenarioClientWithConfig creates a ScenarioClient with explicit configuration.
func NewScenarioClientWithConfig(cfg ScenarioClientConfig) *ScenarioClient {
	return &ScenarioClient{
		httpClient: &http.Client{
			Timeout: cfg.Timeout,
		},
		urlResolver: &defaultURLResolver{},
		timeout:     cfg.Timeout,
		cacheTTL:    cfg.CacheTTL,
		cache:       make(map[string]*cachedManifest),
	}
}

// NewScenarioClientWithDeps creates a ScenarioClient with injected dependencies.
// This is the constructor for testing.
func NewScenarioClientWithDeps(httpClient HTTPClient, urlResolver URLResolver, cfg ScenarioClientConfig) *ScenarioClient {
	return &ScenarioClient{
		httpClient:  httpClient,
		urlResolver: urlResolver,
		timeout:     cfg.Timeout,
		cacheTTL:    cfg.CacheTTL,
		cache:       make(map[string]*cachedManifest),
	}
}

// FetchToolManifest retrieves the tool manifest from a scenario.
// Uses caching to avoid redundant network calls.
func (c *ScenarioClient) FetchToolManifest(ctx context.Context, scenarioName string) (*toolspb.ToolManifest, error) {
	// Check cache first
	if manifest := c.getCached(scenarioName); manifest != nil {
		return manifest, nil
	}

	// Resolve scenario URL
	baseURL, err := c.urlResolver.ResolveScenarioURL(ctx, scenarioName)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve %s URL: %w", scenarioName, err)
	}

	// Fetch manifest
	manifest, err := c.fetchManifest(ctx, baseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch tools from %s: %w", scenarioName, err)
	}

	// Update cache
	c.setCache(scenarioName, manifest)

	return manifest, nil
}

// FetchMultiple fetches tool manifests from multiple scenarios concurrently.
// Returns a map of scenario name to manifest, and any errors encountered.
func (c *ScenarioClient) FetchMultiple(ctx context.Context, scenarioNames []string) (map[string]*toolspb.ToolManifest, map[string]error) {
	results := make(map[string]*toolspb.ToolManifest)
	errors := make(map[string]error)

	var wg sync.WaitGroup
	var mu sync.Mutex

	for _, name := range scenarioNames {
		wg.Add(1)
		go func(scenarioName string) {
			defer wg.Done()

			manifest, err := c.FetchToolManifest(ctx, scenarioName)

			mu.Lock()
			defer mu.Unlock()

			if err != nil {
				errors[scenarioName] = err
			} else {
				results[scenarioName] = manifest
			}
		}(name)
	}

	wg.Wait()
	return results, errors
}

// CheckScenarioStatus checks if a scenario's tool endpoint is reachable.
func (c *ScenarioClient) CheckScenarioStatus(ctx context.Context, scenarioName string) *domain.ScenarioStatus {
	status := &domain.ScenarioStatus{
		Scenario:    scenarioName,
		LastChecked: time.Now(),
	}

	manifest, err := c.FetchToolManifest(ctx, scenarioName)
	if err != nil {
		status.Available = false
		status.Error = err.Error()
		return status
	}

	status.Available = true
	status.ToolCount = len(manifest.Tools)
	return status
}

// InvalidateCache removes a scenario from the cache.
func (c *ScenarioClient) InvalidateCache(scenarioName string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.cache, scenarioName)
}

// InvalidateAllCache clears the entire cache.
func (c *ScenarioClient) InvalidateAllCache() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.cache = make(map[string]*cachedManifest)
}

// getCached retrieves a manifest from cache if still valid.
func (c *ScenarioClient) getCached(scenarioName string) *toolspb.ToolManifest {
	c.mu.RLock()
	defer c.mu.RUnlock()

	cached, ok := c.cache[scenarioName]
	if !ok {
		return nil
	}

	if time.Since(cached.fetchedAt) > c.cacheTTL {
		return nil
	}

	return cached.manifest
}

// setCache stores a manifest in the cache.
func (c *ScenarioClient) setCache(scenarioName string, manifest *toolspb.ToolManifest) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.cache[scenarioName] = &cachedManifest{
		manifest:  manifest,
		fetchedAt: time.Now(),
	}
}

// fetchManifest performs the HTTP request to get the tool manifest.
func (c *ScenarioClient) fetchManifest(ctx context.Context, baseURL string) (*toolspb.ToolManifest, error) {
	url := baseURL + "/api/v1/tools"

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var manifest toolspb.ToolManifest
	// Use protojson for proper proto JSON handling (e.g., timestamps)
	if err := protojson.Unmarshal(body, &manifest); err != nil {
		return nil, fmt.Errorf("failed to parse manifest: %w", err)
	}

	// Validate protocol version
	if manifest.ProtocolVersion != domain.ToolProtocolVersion {
		return nil, fmt.Errorf("unsupported protocol version: got %s, want %s",
			manifest.ProtocolVersion, domain.ToolProtocolVersion)
	}

	return &manifest, nil
}

// defaultURLResolver implements URLResolver using environment variables and vrooli CLI.
type defaultURLResolver struct{}

// ResolveScenarioURL resolves a scenario name to its API base URL.
// Priority:
// 1. Environment variable: <SCENARIO_NAME>_API_URL (e.g., AGENT_MANAGER_API_URL)
// 2. Vrooli CLI: vrooli scenario port <name> API_PORT
func (r *defaultURLResolver) ResolveScenarioURL(ctx context.Context, scenarioName string) (string, error) {
	return ResolveScenarioURL(ctx, scenarioName)
}

// ResolveScenarioURL resolves a scenario name to its API base URL.
// This is exported for use by other packages (e.g., tool_registry.go).
// Priority:
// 1. Environment variable: <SCENARIO_NAME>_API_URL (e.g., AGENT_MANAGER_API_URL)
// 2. Vrooli CLI: vrooli scenario port <name> API_PORT
func ResolveScenarioURL(ctx context.Context, scenarioName string) (string, error) {
	// Try environment variable first
	envKey := strings.ToUpper(strings.ReplaceAll(scenarioName, "-", "_")) + "_API_URL"
	if url := os.Getenv(envKey); url != "" {
		return url, nil
	}

	// Fall back to vrooli CLI
	cmd := exec.CommandContext(ctx, "vrooli", "scenario", "port", scenarioName, "API_PORT")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("scenario %s not available: %w", scenarioName, err)
	}

	port := strings.TrimSpace(string(output))
	if port == "" {
		return "", fmt.Errorf("scenario %s returned empty port", scenarioName)
	}

	return fmt.Sprintf("http://localhost:%s", port), nil
}
