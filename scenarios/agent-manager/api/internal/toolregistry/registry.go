// Package toolregistry provides the tool discovery service for agent-manager.
//
// This package implements the Tool Discovery Protocol, enabling external consumers
// (like agent-inbox) to discover what tools this scenario provides.
//
// ARCHITECTURE:
// - Registry: Central coordinator that manages tool providers
// - ToolProvider: Interface for components that contribute tools
// - ToolDefinitions: Static tool definitions (agent lifecycle, status, etc.)
//
// TESTING SEAMS:
// - ToolProvider interface enables mocking tool sources
// - Registry accepts providers via dependency injection
// - All dependencies are interfaces, not concrete types
package toolregistry

import (
	"context"
	"sync"
	"time"

	"agent-manager/internal/domain"
)

// ToolProvider is the interface for components that contribute tools.
// Implement this interface to add tools from different sources.
type ToolProvider interface {
	// Name returns a unique identifier for this provider.
	Name() string

	// Tools returns the tool definitions this provider contributes.
	// Context allows for cancellation of expensive operations.
	Tools(ctx context.Context) []domain.ToolDefinition

	// Categories returns the category definitions for this provider's tools.
	Categories(ctx context.Context) []domain.ToolCategory
}

// Registry manages tool discovery and manifest generation.
type Registry struct {
	mu        sync.RWMutex
	providers map[string]ToolProvider

	// Scenario metadata
	scenarioName    string
	scenarioVersion string
	scenarioDesc    string
}

// RegistryConfig holds configuration for creating a Registry.
type RegistryConfig struct {
	ScenarioName        string
	ScenarioVersion     string
	ScenarioDescription string
}

// NewRegistry creates a new tool registry with the given configuration.
func NewRegistry(cfg RegistryConfig) *Registry {
	return &Registry{
		providers:       make(map[string]ToolProvider),
		scenarioName:    cfg.ScenarioName,
		scenarioVersion: cfg.ScenarioVersion,
		scenarioDesc:    cfg.ScenarioDescription,
	}
}

// RegisterProvider adds a tool provider to the registry.
// If a provider with the same name exists, it is replaced.
func (r *Registry) RegisterProvider(provider ToolProvider) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.providers[provider.Name()] = provider
}

// UnregisterProvider removes a tool provider from the registry.
func (r *Registry) UnregisterProvider(name string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.providers, name)
}

// GetManifest generates the complete tool manifest.
// This is the main entry point for the GET /api/v1/tools endpoint.
func (r *Registry) GetManifest(ctx context.Context) *domain.ToolManifest {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Collect tools and categories from all providers
	var allTools []domain.ToolDefinition
	categoryMap := make(map[string]domain.ToolCategory)

	for _, provider := range r.providers {
		tools := provider.Tools(ctx)
		allTools = append(allTools, tools...)

		for _, cat := range provider.Categories(ctx) {
			// Later providers can override categories
			categoryMap[cat.ID] = cat
		}
	}

	// Convert category map to slice
	categories := make([]domain.ToolCategory, 0, len(categoryMap))
	for _, cat := range categoryMap {
		categories = append(categories, cat)
	}

	return &domain.ToolManifest{
		ProtocolVersion: domain.ProtocolVersion,
		Scenario: domain.ScenarioInfo{
			Name:        r.scenarioName,
			Version:     r.scenarioVersion,
			Description: r.scenarioDesc,
		},
		Tools:       allTools,
		Categories:  categories,
		GeneratedAt: time.Now().UTC(),
	}
}

// GetTool returns a specific tool by name.
// Returns nil if the tool is not found.
func (r *Registry) GetTool(ctx context.Context, name string) *domain.ToolDefinition {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, provider := range r.providers {
		for _, tool := range provider.Tools(ctx) {
			if tool.Name == name {
				return &tool
			}
		}
	}
	return nil
}

// ListToolNames returns the names of all registered tools.
func (r *Registry) ListToolNames(ctx context.Context) []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var names []string
	for _, provider := range r.providers {
		for _, tool := range provider.Tools(ctx) {
			names = append(names, tool.Name)
		}
	}
	return names
}

// ProviderCount returns the number of registered providers.
func (r *Registry) ProviderCount() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.providers)
}
