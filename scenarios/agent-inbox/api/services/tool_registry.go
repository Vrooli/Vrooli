// Package services provides application services for the Agent Inbox scenario.
//
// This file implements tool discovery and refresh for the ToolRegistry service.
//
// ARCHITECTURE:
// - ToolRegistry: Central coordinator for tool discovery and configuration
// - Uses ScenarioClient for fetching manifests from scenarios
// - Uses Repository for persisting user preferences
// - Provides effective tool sets with merged configuration
//
// TESTING SEAMS:
// - ScenarioClient interface for mocking network calls
// - Repository interface for mocking database access
package services

import (
	"agent-inbox/domain"
	"context"
	"fmt"
	"log"
	"time"
)

// RefreshTools fetches tools from scenarios and updates the cache.
// If AutoDiscovery is enabled, it discovers scenarios via vrooli CLI.
// Otherwise, it uses the explicit Scenarios list from config.
// It auto-registers protocol handlers for all discovered scenarios.
func (r *ToolRegistry) RefreshTools(ctx context.Context) error {
	var scenarios []string

	// Determine which scenarios to fetch from
	if r.cfg.Integration.ToolDiscovery.AutoDiscovery {
		// Dynamic discovery via vrooli CLI
		discovered, err := r.scenarioClient.DiscoverToolScenarios(ctx)
		if err != nil {
			log.Printf("warning: auto-discovery failed, using fallback: %v", err)
			scenarios = r.cfg.Integration.ToolDiscovery.Scenarios
		} else {
			scenarios = discovered
			log.Printf("Auto-discovery found %d scenarios with tools", len(scenarios))
		}
	} else {
		// Use explicit list from config
		scenarios = r.cfg.Integration.ToolDiscovery.Scenarios
	}

	// Fetch tool manifests from discovered scenarios
	manifests, errors := r.scenarioClient.FetchMultiple(ctx, scenarios)

	// Log any errors but continue with available manifests
	for scenario, err := range errors {
		log.Printf("warning: failed to fetch tools from %s: %v", scenario, err)
	}

	// Auto-register protocol handlers for discovered scenarios
	r.registerHandlers(ctx, manifests)

	// Build aggregated tool set
	toolSet := r.buildToolSet(manifests)

	// Update cache
	r.mu.Lock()
	r.cachedTools = toolSet
	r.cacheTime = time.Now()
	r.mu.Unlock()

	log.Printf("Tool registry refreshed: %d tools from %d scenarios",
		len(toolSet.Tools), len(toolSet.Scenarios))

	return nil
}

// StartAsyncDiscovery triggers background discovery without blocking.
// Call this on startup if you want cached tools available immediately.
func (r *ToolRegistry) StartAsyncDiscovery() {
	if r.cfg.Integration.ToolDiscovery.AutoDiscovery {
		go r.discoverAsync(context.Background())
	}
}

// discoverAsync runs discovery in background and updates cache.
func (r *ToolRegistry) discoverAsync(ctx context.Context) {
	discovered, err := r.scenarioClient.DiscoverToolScenarios(ctx)
	if err != nil {
		log.Printf("warning: async auto-discovery failed: %v", err)
		return
	}

	// Fetch manifests from discovered scenarios
	manifests, errors := r.scenarioClient.FetchMultiple(ctx, discovered)
	for scenario, err := range errors {
		log.Printf("warning: failed to fetch tools from %s: %v", scenario, err)
	}

	// Update cache atomically
	toolSet := r.buildToolSet(manifests)
	r.mu.Lock()
	r.cachedTools = toolSet
	r.cacheTime = time.Now()
	r.mu.Unlock()

	// Register protocol handlers
	r.registerHandlers(ctx, manifests)

	log.Printf("Async discovery complete: %d scenarios with %d tools", len(discovered), len(toolSet.Tools))
}

// SyncTools performs full discovery synchronously (for manual sync button).
// Returns detailed results about what was discovered.
func (r *ToolRegistry) SyncTools(ctx context.Context) (*domain.DiscoveryResult, error) {
	// Clear caches to force fresh discovery
	r.scenarioClient.InvalidateAllCache()

	// Get previous scenario list for comparison (getScenarioNames handles its own locking)
	previousScenarios := r.getScenarioNames()

	// Discover scenarios with tools
	discovered, err := r.scenarioClient.DiscoverToolScenarios(ctx)
	if err != nil {
		return nil, fmt.Errorf("discovery failed: %w", err)
	}

	// Fetch manifests
	manifests, errors := r.scenarioClient.FetchMultiple(ctx, discovered)
	for scenario, err := range errors {
		log.Printf("warning: failed to fetch tools from %s: %v", scenario, err)
	}

	// Build and update cache
	toolSet := r.buildToolSet(manifests)
	r.mu.Lock()
	r.cachedTools = toolSet
	r.cacheTime = time.Now()
	r.mu.Unlock()

	// Register protocol handlers
	r.registerHandlers(ctx, manifests)

	// Calculate diff
	newScenarios := difference(discovered, previousScenarios)
	removedScenarios := difference(previousScenarios, discovered)

	return &domain.DiscoveryResult{
		ScenariosWithTools: len(discovered),
		NewScenarios:       newScenarios,
		RemovedScenarios:   removedScenarios,
		TotalTools:         len(toolSet.Tools),
	}, nil
}

// GetToolSet returns the current aggregated tool set.
// If the cache is stale, it triggers a background refresh.
func (r *ToolRegistry) GetToolSet(ctx context.Context) (*domain.ToolSet, error) {
	r.mu.RLock()
	cached := r.cachedTools
	cacheAge := time.Since(r.cacheTime)
	r.mu.RUnlock()

	// Return cached if still valid
	if cached != nil && cacheAge < r.cfg.Integration.ToolDiscovery.CacheTTL {
		return cached, nil
	}

	// Refresh and return
	if err := r.RefreshTools(ctx); err != nil {
		// If refresh fails but we have stale cache, use it
		if cached != nil {
			log.Printf("warning: using stale tool cache due to refresh error: %v", err)
			return cached, nil
		}
		return nil, err
	}

	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.cachedTools, nil
}
