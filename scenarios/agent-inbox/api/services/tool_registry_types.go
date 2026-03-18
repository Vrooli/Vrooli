// Package services provides application services for the Agent Inbox scenario.
//
// This file defines types and interfaces for the ToolRegistry service.
package services

import (
	"context"
	"sync"
	"time"

	"agent-inbox/config"
	"agent-inbox/domain"
	"agent-inbox/integrations"
	"agent-inbox/persistence"
)

// ToolRegistry manages tool discovery and user configurations.
type ToolRegistry struct {
	scenarioClient *integrations.ScenarioClient
	repo           *persistence.Repository
	cfg            *config.Config
	toolExecutor   *integrations.ToolExecutor // Injected for auto-registering protocol handlers

	// Cache for aggregated tool set
	mu          sync.RWMutex
	cachedTools *domain.ToolSet
	cacheTime   time.Time
}

// NewToolRegistry creates a new ToolRegistry with default dependencies.
// Pass nil for toolExecutor if you don't need auto-registration of protocol handlers.
func NewToolRegistry(repo *persistence.Repository, toolExecutor *integrations.ToolExecutor) *ToolRegistry {
	return &ToolRegistry{
		scenarioClient: integrations.NewScenarioClient(),
		repo:           repo,
		cfg:            config.Default(),
		toolExecutor:   toolExecutor,
	}
}

// defaultURLResolver implements URL resolution for scenarios.
// This duplicates the logic from scenario_client.go's defaultURLResolver
// to avoid a circular dependency.
type defaultURLResolver struct{}

// ResolveScenarioURL resolves a scenario name to its API base URL.
func (r *defaultURLResolver) ResolveScenarioURL(ctx context.Context, scenarioName string) (string, error) {
	// Use the integrations package's URL resolution
	return integrations.ResolveScenarioURL(ctx, scenarioName)
}
