// Package services provides application services for the Agent Inbox scenario.
//
// This file implements the ToolRegistry service for dynamic tool discovery.
// The registry aggregates tools from multiple scenarios and manages user
// configurations for enabling/disabling tools.
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
	"context"
	"fmt"
	"log"
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

	// Cache for aggregated tool set
	mu          sync.RWMutex
	cachedTools *domain.ToolSet
	cacheTime   time.Time
}

// NewToolRegistry creates a new ToolRegistry with default dependencies.
func NewToolRegistry(repo *persistence.Repository) *ToolRegistry {
	return &ToolRegistry{
		scenarioClient: integrations.NewScenarioClient(),
		repo:           repo,
		cfg:            config.Default(),
	}
}

// NewToolRegistryWithDeps creates a ToolRegistry with injected dependencies.
// This is the constructor for testing.
func NewToolRegistryWithDeps(client *integrations.ScenarioClient, repo *persistence.Repository, cfg *config.Config) *ToolRegistry {
	return &ToolRegistry{
		scenarioClient: client,
		repo:           repo,
		cfg:            cfg,
	}
}

// RefreshTools fetches tools from all configured scenarios.
// This is typically called on startup and periodically.
func (r *ToolRegistry) RefreshTools(ctx context.Context) error {
	scenarios := r.cfg.Integration.ToolDiscovery.Scenarios

	manifests, errors := r.scenarioClient.FetchMultiple(ctx, scenarios)

	// Log any errors but continue with available manifests
	for scenario, err := range errors {
		log.Printf("warning: failed to fetch tools from %s: %v", scenario, err)
	}

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

// GetEffectiveTools returns tools with user configurations applied.
// Pass empty chatID for global defaults, or a chatID for chat-specific settings.
func (r *ToolRegistry) GetEffectiveTools(ctx context.Context, chatID string) ([]domain.EffectiveTool, error) {
	toolSet, err := r.GetToolSet(ctx)
	if err != nil {
		return nil, err
	}

	// Get user configurations
	configs, err := r.repo.ListToolConfigurations(ctx, chatID)
	if err != nil {
		return nil, fmt.Errorf("failed to load tool configurations: %w", err)
	}

	// Build config lookup maps (global and chat-specific)
	globalConfigMap := make(map[string]*domain.ToolConfiguration)
	chatConfigMap := make(map[string]*domain.ToolConfiguration)
	for _, cfg := range configs {
		key := cfg.Scenario + "/" + cfg.ToolName
		if cfg.ChatID != "" {
			chatConfigMap[key] = cfg
		} else {
			globalConfigMap[key] = cfg
		}
	}

	// Apply configurations to tools
	result := make([]domain.EffectiveTool, len(toolSet.Tools))
	for i, tool := range toolSet.Tools {
		result[i] = tool
		key := tool.Scenario + "/" + tool.Tool.Name

		// Determine effective enabled state
		if chatCfg, ok := chatConfigMap[key]; ok {
			result[i].Enabled = chatCfg.Enabled
			result[i].Source = domain.ScopeChat
		} else if globalCfg, ok := globalConfigMap[key]; ok {
			result[i].Enabled = globalCfg.Enabled
			result[i].Source = domain.ScopeGlobal
		}

		// Determine effective approval requirement
		// Priority: chat-specific > global > tool metadata default
		metadataDefault := tool.Tool.Metadata.RequiresApproval
		result[i].RequiresApproval = metadataDefault

		if chatCfg, ok := chatConfigMap[key]; ok && chatCfg.ApprovalOverride != "" {
			result[i].RequiresApproval = chatCfg.ApprovalOverride == domain.ApprovalRequire
			result[i].ApprovalSource = domain.ScopeChat
			result[i].ApprovalOverride = chatCfg.ApprovalOverride
		} else if globalCfg, ok := globalConfigMap[key]; ok && globalCfg.ApprovalOverride != "" {
			result[i].RequiresApproval = globalCfg.ApprovalOverride == domain.ApprovalRequire
			result[i].ApprovalSource = domain.ScopeGlobal
			result[i].ApprovalOverride = globalCfg.ApprovalOverride
		}
		// If no override, ApprovalSource and ApprovalOverride stay empty (meaning tool default)
	}

	return result, nil
}

// GetEnabledTools returns only the tools that are currently enabled.
func (r *ToolRegistry) GetEnabledTools(ctx context.Context, chatID string) ([]domain.EffectiveTool, error) {
	tools, err := r.GetEffectiveTools(ctx, chatID)
	if err != nil {
		return nil, err
	}

	var enabled []domain.EffectiveTool
	for _, tool := range tools {
		if tool.Enabled {
			enabled = append(enabled, tool)
		}
	}

	return enabled, nil
}

// GetToolsForOpenAI returns enabled tools in OpenAI function-calling format.
// This is used when making requests to OpenRouter.
func (r *ToolRegistry) GetToolsForOpenAI(ctx context.Context, chatID string) ([]map[string]interface{}, error) {
	tools, err := r.GetEnabledTools(ctx, chatID)
	if err != nil {
		return nil, err
	}

	result := make([]map[string]interface{}, len(tools))
	for i, tool := range tools {
		result[i] = tool.Tool.ToOpenAIFunction()
	}

	return result, nil
}

// SetToolEnabled updates the enabled state for a tool.
// Pass empty chatID for global configuration.
func (r *ToolRegistry) SetToolEnabled(ctx context.Context, chatID, scenario, toolName string, enabled bool) error {
	cfg := &domain.ToolConfiguration{
		ChatID:   chatID,
		Scenario: scenario,
		ToolName: toolName,
		Enabled:  enabled,
	}

	return r.repo.SaveToolConfiguration(ctx, cfg)
}

// ResetToolConfiguration removes a tool configuration, reverting to default.
// Pass empty chatID for global configuration.
func (r *ToolRegistry) ResetToolConfiguration(ctx context.Context, chatID, scenario, toolName string) error {
	return r.repo.DeleteToolConfiguration(ctx, chatID, scenario, toolName)
}

// GetScenarioStatuses checks availability of all configured scenarios.
func (r *ToolRegistry) GetScenarioStatuses(ctx context.Context) []*domain.ScenarioStatus {
	scenarios := r.cfg.Integration.ToolDiscovery.Scenarios
	statuses := make([]*domain.ScenarioStatus, len(scenarios))

	var wg sync.WaitGroup
	for i, scenario := range scenarios {
		wg.Add(1)
		go func(idx int, name string) {
			defer wg.Done()
			statuses[idx] = r.scenarioClient.CheckScenarioStatus(ctx, name)
		}(i, scenario)
	}

	wg.Wait()
	return statuses
}

// GetTool looks up a specific tool by scenario and name.
func (r *ToolRegistry) GetTool(ctx context.Context, scenario, toolName string) (*domain.ToolDefinition, error) {
	toolSet, err := r.GetToolSet(ctx)
	if err != nil {
		return nil, err
	}

	for _, tool := range toolSet.Tools {
		if tool.Scenario == scenario && tool.Tool.Name == toolName {
			return &tool.Tool, nil
		}
	}

	return nil, fmt.Errorf("tool not found: %s/%s", scenario, toolName)
}

// GetToolByName looks up a tool by name only (across all scenarios).
// If multiple scenarios provide tools with the same name, returns the first found.
func (r *ToolRegistry) GetToolByName(ctx context.Context, toolName string) (*domain.ToolDefinition, string, error) {
	toolSet, err := r.GetToolSet(ctx)
	if err != nil {
		return nil, "", err
	}

	for _, tool := range toolSet.Tools {
		if tool.Tool.Name == toolName {
			return &tool.Tool, tool.Scenario, nil
		}
	}

	return nil, "", fmt.Errorf("tool not found: %s", toolName)
}

// GetToolApprovalRequired checks if a tool requires approval before execution.
// This considers YOLO mode, user overrides, and tool metadata defaults.
// Returns (requiresApproval, source, error).
func (r *ToolRegistry) GetToolApprovalRequired(ctx context.Context, chatID, toolName string) (bool, domain.ToolConfigurationScope, error) {
	// First check YOLO mode - if enabled, never require approval
	yoloMode, err := r.repo.GetYoloMode(ctx)
	if err != nil {
		log.Printf("warning: failed to check YOLO mode: %v", err)
	}
	if yoloMode {
		return false, "", nil
	}

	// Look up the tool to get its metadata default and scenario
	tool, scenario, err := r.GetToolByName(ctx, toolName)
	if err != nil {
		// Tool not found, default to not requiring approval
		return false, "", nil
	}

	// Check user overrides via repository
	metadataDefault := tool.Metadata.RequiresApproval
	return r.repo.GetEffectiveToolApproval(ctx, chatID, scenario, toolName, metadataDefault)
}

// SetToolApprovalOverride updates the approval override for a tool.
// Pass empty chatID for global configuration.
// Pass empty override to reset to default (use tool metadata).
func (r *ToolRegistry) SetToolApprovalOverride(ctx context.Context, chatID, scenario, toolName string, override domain.ApprovalOverride) error {
	return r.repo.SetToolApprovalOverride(ctx, chatID, scenario, toolName, override)
}

// buildToolSet aggregates manifests from multiple scenarios into a ToolSet.
func (r *ToolRegistry) buildToolSet(manifests map[string]*domain.ToolManifest) *domain.ToolSet {
	var scenarios []domain.ScenarioInfo
	var tools []domain.EffectiveTool
	categoryMap := make(map[string]domain.ToolCategory)

	for scenarioName, manifest := range manifests {
		// Add scenario info (with base URL if we know it)
		info := manifest.Scenario
		scenarios = append(scenarios, info)

		// Add tools with default enabled state and approval requirement
		for _, tool := range manifest.Tools {
			tools = append(tools, domain.EffectiveTool{
				Scenario:         scenarioName,
				Tool:             tool,
				Enabled:          tool.Metadata.EnabledByDefault,
				Source:           "",                             // Empty means using tool's default
				RequiresApproval: tool.Metadata.RequiresApproval, // Default from tool metadata
				ApprovalSource:   "",                             // Empty means using tool's default
			})
		}

		// Merge categories
		for _, cat := range manifest.Categories {
			categoryMap[cat.ID] = cat
		}
	}

	// Convert category map to slice
	categories := make([]domain.ToolCategory, 0, len(categoryMap))
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
