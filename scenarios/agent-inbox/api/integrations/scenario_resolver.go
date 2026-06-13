// Package integrations provides clients for external services.
// This file contains URL resolution and scenario discovery utilities.
package integrations

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"

	vroolicli "github.com/vrooli/vrooli-cli-go"
)

// cliClient is the shared typed Vrooli CLI client for scenario discovery. It
// decodes the vrooli.cli.v1 contracts instead of hand-parsing CLI JSON, so a
// CLI output change is a compile error here rather than a silently empty or
// wrong result.
var cliClient = vroolicli.New()

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

	// Fall back to the vrooli CLI's typed scenario-port lookup.
	resp, err := cliClient.ScenarioPort(ctx, scenarioName, "API_PORT")
	if err != nil {
		return "", fmt.Errorf("scenario %s not available: %w", scenarioName, err)
	}
	if !resp.GetSuccess() || resp.GetPort() == 0 {
		return "", fmt.Errorf("scenario %s returned no API_PORT: %s", scenarioName, resp.GetError())
	}

	return fmt.Sprintf("http://localhost:%d", resp.GetPort()), nil
}

// DiscoverToolScenarios uses vrooli CLI to find all running scenarios
// and probes each for /api/v1/tools endpoints.
// Returns the names of scenarios that implement the Tool Discovery Protocol.
func (c *ScenarioClient) DiscoverToolScenarios(ctx context.Context) ([]string, error) {
	// 1. List scenarios via the typed CLI client.
	resp, err := cliClient.ListScenarios(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list vrooli scenarios: %w", err)
	}

	// 2. Filter running scenarios (exclude [UNREGISTERED] and [MISSING]).
	var candidates []string
	for _, s := range resp.GetScenarios() {
		status := s.GetStatus()
		if s.GetName() != "" && status != "[UNREGISTERED]" && status != "[MISSING]" {
			candidates = append(candidates, s.GetName())
		}
	}

	// 3. Probe each /api/v1/tools (parallel, with timeout).
	return c.probeForTools(ctx, candidates)
}

// probeForTools probes each scenario's /api/v1/tools endpoint in parallel.
// Returns only the scenarios that successfully return a tool manifest.
func (c *ScenarioClient) probeForTools(ctx context.Context, scenarios []string) ([]string, error) {
	var wg sync.WaitGroup
	var mu sync.Mutex
	var withTools []string

	for _, name := range scenarios {
		wg.Add(1)
		go func(scenarioName string) {
			defer wg.Done()

			// Use a shorter timeout context for probing
			probeCtx, cancel := context.WithTimeout(ctx, c.timeout)
			defer cancel()

			_, err := c.FetchToolManifest(probeCtx, scenarioName)
			if err == nil {
				mu.Lock()
				withTools = append(withTools, scenarioName)
				mu.Unlock()
			}
			// Silent skip on error (404/timeout = no tools)
		}(name)
	}

	wg.Wait()
	return withTools, nil
}
