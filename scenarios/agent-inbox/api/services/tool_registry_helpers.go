// Package services provides application services for the Agent Inbox scenario.
//
// This file contains validation, conversion, and helper functions for the ToolRegistry.
package services

import (
	"context"
	"log"
	"time"

	"agent-inbox/domain"

	toolspb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-inbox/v1/domain"
)

// difference returns elements in a that are not in b.
func difference(a, b []string) []string {
	bSet := make(map[string]bool)
	for _, s := range b {
		bSet[s] = true
	}
	var diff []string
	for _, s := range a {
		if !bSet[s] {
			diff = append(diff, s)
		}
	}
	return diff
}

// resolveScenarioURL resolves a scenario name to its API base URL.
// It uses the same resolution logic as the ScenarioClient's URLResolver.
func (r *ToolRegistry) resolveScenarioURL(ctx context.Context, scenarioName string) (string, error) {
	resolver := &defaultURLResolver{}
	return resolver.ResolveScenarioURL(ctx, scenarioName)
}

// registerHandlers registers protocol handlers for all discovered scenarios.
func (r *ToolRegistry) registerHandlers(ctx context.Context, manifests map[string]*toolspb.ToolManifest) {
	if r.toolExecutor == nil {
		return
	}

	for scenarioName, manifest := range manifests {
		// Resolve the scenario URL
		baseURL, err := r.resolveScenarioURL(ctx, scenarioName)
		if err != nil {
			log.Printf("warning: failed to resolve URL for %s: %v", scenarioName, err)
			continue
		}

		// Extract tool names from manifest
		var toolNames []string
		for _, tool := range manifest.Tools {
			toolNames = append(toolNames, tool.Name)
		}

		// Register the protocol handler with URL resolver for re-resolution on failure
		resolver := &defaultURLResolver{}
		r.toolExecutor.RegisterProtocolHandler(scenarioName, baseURL, toolNames, resolver)
		log.Printf("Registered protocol handler for %s with %d tools", scenarioName, len(toolNames))
	}
}

// getScenarioNames returns the names of scenarios in the current cache.
// Thread-safe: acquires read lock internally.
func (r *ToolRegistry) getScenarioNames() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if r.cachedTools == nil {
		return nil
	}
	var names []string
	for _, s := range r.cachedTools.Scenarios {
		if s != nil {
			names = append(names, s.Name)
		}
	}
	return names
}

// buildToolSet aggregates manifests from multiple scenarios into a ToolSet.
func (r *ToolRegistry) buildToolSet(manifests map[string]*toolspb.ToolManifest) *domain.ToolSet {
	var scenarios []*toolspb.ScenarioInfo
	var tools []domain.EffectiveTool
	categoryMap := make(map[string]*toolspb.ToolCategory)

	for scenarioName, manifest := range manifests {
		// Add scenario info (with base URL if we know it)
		info := manifest.Scenario
		scenarios = append(scenarios, info)

		// Add tools with default enabled state and approval requirement
		for _, tool := range manifest.Tools {
			enabledByDefault := false
			requiresApproval := false
			if tool.Metadata != nil {
				enabledByDefault = tool.Metadata.EnabledByDefault
				requiresApproval = tool.Metadata.RequiresApproval
			}
			tools = append(tools, domain.EffectiveTool{
				Scenario:         scenarioName,
				Tool:             tool,
				Enabled:          enabledByDefault,
				Source:           "",               // Empty means using tool's default
				RequiresApproval: requiresApproval, // Default from tool metadata
				ApprovalSource:   "",               // Empty means using tool's default
			})
		}

		// Merge categories
		for _, cat := range manifest.Categories {
			categoryMap[cat.Id] = cat
		}
	}

	// Convert category map to slice
	categories := make([]*toolspb.ToolCategory, 0, len(categoryMap))
	for _, cat := range categoryMap {
		categories = append(categories, cat)
	}

	return &domain.ToolSet{
		Scenarios:   scenarios,
		Tools:       tools,
		Categories:  categories,
		GeneratedAt: time.Now(),
	}
}
