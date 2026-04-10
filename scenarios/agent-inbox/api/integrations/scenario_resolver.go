// Package integrations provides clients for external services.
// This file contains URL resolution and scenario discovery utilities.
package integrations

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
)

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

// DiscoverToolScenarios uses vrooli CLI to find all running scenarios
// and probes each for /api/v1/tools endpoints.
// Returns the names of scenarios that implement the Tool Discovery Protocol.
func (c *ScenarioClient) DiscoverToolScenarios(ctx context.Context) ([]string, error) {
	// 1. Run: vrooli scenario list --json
	cmd := exec.CommandContext(ctx, "vrooli", "scenario", "list", "--json")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to run vrooli scenario list: %w", err)
	}

	// 2. Parse JSON response
	var response struct {
		Scenarios []struct {
			Name   string `json:"name"`
			Status string `json:"status"`
		} `json:"scenarios"`
	}
	if err := json.Unmarshal(output, &response); err != nil {
		return nil, fmt.Errorf("failed to parse vrooli response: %w", err)
	}

	// 3. Filter running scenarios (exclude [UNREGISTERED] and [MISSING])
	var candidates []string
	for _, s := range response.Scenarios {
		if s.Status != "[UNREGISTERED]" && s.Status != "[MISSING]" && s.Name != "" {
			candidates = append(candidates, s.Name)
		}
	}

	// 4. Probe each /api/v1/tools (parallel, with timeout)
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
