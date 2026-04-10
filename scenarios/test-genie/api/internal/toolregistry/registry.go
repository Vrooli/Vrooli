// Package toolregistry provides the tool discovery service for test-genie.
//
// This package implements the Tool Discovery Protocol, enabling external consumers
// (like agent-inbox) to discover what tools this scenario provides.
//
// ARCHITECTURE:
// - Registry: Central coordinator that manages tool providers
// - ToolProvider: Interface for components that contribute tools
// - ToolDefinitions: Static tool definitions (test execution, scenarios, fixes, requirements)
//
// TESTING SEAMS:
// - ToolProvider interface enables mocking tool sources
// - Registry accepts providers via dependency injection
// - All dependencies are interfaces, not concrete types
package toolregistry

import (
	"context"
	"sync"

	toolspb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-inbox/v1/domain"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// ToolProvider is the interface for components that contribute tools.
// Implement this interface to add tools from different sources.
type ToolProvider interface {
	// Name returns a unique identifier for this provider.
	Name() string

	// Tools returns the tool definitions this provider contributes.
	// Context allows for cancellation of expensive operations.
	Tools(ctx context.Context) []*toolspb.ToolDefinition

	// Categories returns the category definitions for this provider's tools.
	Categories(ctx context.Context) []*toolspb.ToolCategory
}

// Registry manages tool discovery and manifest generation.
type Registry struct {
	mu        sync.RWMutex
	providers map[string]ToolProvider
	order     []string

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
		order:           make([]string, 0),
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
	if _, exists := r.providers[provider.Name()]; !exists {
		r.order = append(r.order, provider.Name())
	}
	r.providers[provider.Name()] = provider
}

// UnregisterProvider removes a tool provider from the registry.
func (r *Registry) UnregisterProvider(name string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.providers, name)
	for i, providerName := range r.order {
		if providerName == name {
			r.order = append(r.order[:i], r.order[i+1:]...)
			break
		}
	}
}

// GetManifest generates the complete tool manifest.
// This is the main entry point for the GET /api/v1/tools endpoint.
func (r *Registry) GetManifest(ctx context.Context) *toolspb.ToolManifest {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Collect tools and categories from all providers
	var allTools []*toolspb.ToolDefinition
	categoryMap := make(map[string]*toolspb.ToolCategory)
	categoryOrder := make([]string, 0)

	for _, providerName := range r.order {
		provider := r.providers[providerName]
		if provider == nil {
			continue
		}
		tools := provider.Tools(ctx)
		allTools = append(allTools, tools...)

		for _, cat := range provider.Categories(ctx) {
			if _, exists := categoryMap[cat.Id]; !exists {
				categoryOrder = append(categoryOrder, cat.Id)
			}
			// Later providers can override categories
			categoryMap[cat.Id] = cat
		}
	}

	// Convert category map to slice
	categories := make([]*toolspb.ToolCategory, 0, len(categoryOrder))
	for _, categoryID := range categoryOrder {
		if cat := categoryMap[categoryID]; cat != nil {
			categories = append(categories, cat)
		}
	}

	return &toolspb.ToolManifest{
		ProtocolVersion: ToolProtocolVersion,
		Scenario: &toolspb.ScenarioInfo{
			Name:        r.scenarioName,
			Version:     r.scenarioVersion,
			Description: r.scenarioDesc,
		},
		Tools:       allTools,
		Categories:  categories,
		GeneratedAt: timestamppb.Now(),
	}
}

// GetTool returns a specific tool by name.
// Returns nil if the tool is not found.
func (r *Registry) GetTool(ctx context.Context, toolName string) *toolspb.ToolDefinition {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, providerName := range r.order {
		provider := r.providers[providerName]
		if provider == nil {
			continue
		}
		for _, tool := range provider.Tools(ctx) {
			if tool.Name == toolName {
				return tool
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
	for _, name := range r.order {
		provider := r.providers[name]
		if provider == nil {
			continue
		}
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
