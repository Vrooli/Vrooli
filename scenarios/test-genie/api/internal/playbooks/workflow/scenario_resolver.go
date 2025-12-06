package workflow

import (
	"context"
	"fmt"
	"strings"
)

// ScenarioURLResolver is a function that resolves a scenario name to its UI base URL.
type ScenarioURLResolver func(ctx context.Context, scenario string) (string, error)

// ResolveScenarioURLs transforms navigate nodes with destinationType=scenario
// to destinationType=url by resolving the target scenario's UI URL.
//
// This transformation is required because:
// 1. Playbooks use destinationType=scenario for portability (no hardcoded URLs)
// 2. BAS's validator rejects unresolved scenario references (WF_UNRESOLVED_SCENARIO_URL)
// 3. The actual UI URL depends on runtime port allocation
//
// The function modifies the definition in place and returns an error if any
// scenario URL cannot be resolved.
func ResolveScenarioURLs(
	ctx context.Context,
	definition map[string]any,
	resolver ScenarioURLResolver,
) error {
	if resolver == nil {
		return nil // No resolver means we can't resolve - let BAS validation catch it
	}

	nodes, ok := definition["nodes"].([]any)
	if !ok {
		return nil // No nodes to process
	}

	for i, node := range nodes {
		nodeMap, ok := node.(map[string]any)
		if !ok {
			continue
		}

		nodeType, _ := nodeMap["type"].(string)

		// Process navigate nodes
		if nodeType == "navigate" {
			data, ok := nodeMap["data"].(map[string]any)
			if !ok {
				continue
			}

			if err := resolveNavigateNode(ctx, data, resolver, i); err != nil {
				nodeID, _ := nodeMap["id"].(string)
				return fmt.Errorf("node %q (index %d): %w", nodeID, i, err)
			}
		}

		// Recursively process nested workflowDefinitions (from fixture expansion)
		data, ok := nodeMap["data"].(map[string]any)
		if ok {
			if nestedDef, ok := data["workflowDefinition"].(map[string]any); ok {
				if err := ResolveScenarioURLs(ctx, nestedDef, resolver); err != nil {
					nodeID, _ := nodeMap["id"].(string)
					return fmt.Errorf("nested workflow in node %q: %w", nodeID, err)
				}
			}
		}
	}

	return nil
}

// resolveNavigateNode transforms a single navigate node if it uses destinationType=scenario.
func resolveNavigateNode(
	ctx context.Context,
	data map[string]any,
	resolver ScenarioURLResolver,
	nodeIndex int,
) error {
	destType, _ := data["destinationType"].(string)
	destType = strings.ToLower(strings.TrimSpace(destType))

	if destType != "scenario" {
		return nil // Not a scenario destination, nothing to do
	}

	// Extract scenario name
	scenario, _ := data["scenario"].(string)
	scenario = strings.TrimSpace(scenario)

	if scenario == "" {
		return fmt.Errorf("destinationType=scenario but scenario name is empty")
	}

	// Resolve the scenario's UI base URL
	baseURL, err := resolver(ctx, scenario)
	if err != nil {
		return fmt.Errorf("failed to resolve URL for scenario %q: %w", scenario, err)
	}

	if baseURL == "" {
		return fmt.Errorf("resolved URL for scenario %q is empty (scenario may not have a UI)", scenario)
	}

	// Build the full URL by combining base URL with scenarioPath
	scenarioPath, _ := data["scenarioPath"].(string)
	fullURL := buildFullURL(baseURL, scenarioPath)

	// Transform the node: change destinationType and set url
	data["destinationType"] = "url"
	data["url"] = fullURL

	// Clean up scenario-specific fields that are no longer needed
	delete(data, "scenario")
	delete(data, "scenarioPath")

	return nil
}

// buildFullURL combines a base URL with a path, handling slashes correctly.
func buildFullURL(baseURL, path string) string {
	baseURL = strings.TrimSpace(baseURL)
	path = strings.TrimSpace(path)

	if path == "" {
		return baseURL
	}

	// Ensure proper slash handling
	baseHasSlash := strings.HasSuffix(baseURL, "/")
	pathHasSlash := strings.HasPrefix(path, "/")

	switch {
	case baseHasSlash && pathHasSlash:
		// Both have slash: remove one
		return baseURL + path[1:]
	case !baseHasSlash && !pathHasSlash:
		// Neither has slash: add one
		return baseURL + "/" + path
	default:
		// Exactly one has slash: concatenate directly
		return baseURL + path
	}
}
