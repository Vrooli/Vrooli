// Package services provides application services for the Agent Inbox scenario.
//
// This file implements tool configuration, lookup, and approval methods
// for the ToolRegistry service.
package services

import (
	"agent-inbox/domain"
	"context"
	"fmt"
	"log"
	"sync"

	toolspb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-inbox/v1/domain"
)

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
		metadataDefault := false
		if tool.Tool.Metadata != nil {
			metadataDefault = tool.Tool.Metadata.RequiresApproval
		}
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

	var result []map[string]interface{}
	for _, tool := range tools {
		// Skip internal tools (status polling, cancellation, etc.)
		// These are used by the async tracker but should not be visible to AI.
		if tool.Tool.Metadata != nil && tool.Tool.Metadata.InternalOnly {
			continue
		}
		result = append(result, domain.ToOpenAIFunction(tool.Tool))
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

// GetScenarioStatuses checks availability of all discovered scenarios.
// Uses scenarios from the cached toolSet, not the hardcoded config.
func (r *ToolRegistry) GetScenarioStatuses(ctx context.Context) []*domain.ScenarioStatus {
	// Get scenarios from cache (discovered scenarios)
	scenarios := r.getScenarioNames()
	if len(scenarios) == 0 {
		// Fallback to config if cache is empty
		scenarios = r.cfg.Integration.ToolDiscovery.Scenarios
	}

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
func (r *ToolRegistry) GetTool(ctx context.Context, scenario, toolName string) (*toolspb.ToolDefinition, error) {
	toolSet, err := r.GetToolSet(ctx)
	if err != nil {
		return nil, err
	}

	for _, tool := range toolSet.Tools {
		if tool.Scenario == scenario && tool.Tool.Name == toolName {
			return tool.Tool, nil
		}
	}

	return nil, fmt.Errorf("tool not found: %s/%s", scenario, toolName)
}

// GetToolByName looks up a tool by name only (across all scenarios).
// If multiple scenarios provide tools with the same name, returns the first found.
func (r *ToolRegistry) GetToolByName(ctx context.Context, toolName string) (*toolspb.ToolDefinition, string, error) {
	toolSet, err := r.GetToolSet(ctx)
	if err != nil {
		return nil, "", err
	}

	for _, tool := range toolSet.Tools {
		if tool.Tool.Name == toolName {
			return tool.Tool, tool.Scenario, nil
		}
	}

	return nil, "", fmt.Errorf("tool not found: %s", toolName)
}

// GetScenarioInfo returns the ScenarioInfo for a given scenario name.
// Returns nil if the scenario is not found in the cached tool set.
func (r *ToolRegistry) GetScenarioInfo(ctx context.Context, scenarioName string) (*toolspb.ScenarioInfo, error) {
	toolSet, err := r.GetToolSet(ctx)
	if err != nil {
		return nil, err
	}

	for _, scenario := range toolSet.Scenarios {
		if scenario != nil && scenario.Name == scenarioName {
			return scenario, nil
		}
	}

	return nil, fmt.Errorf("scenario not found: %s", scenarioName)
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
	metadataDefault := false
	if tool.Metadata != nil {
		metadataDefault = tool.Metadata.RequiresApproval
	}
	return r.repo.GetEffectiveToolApproval(ctx, chatID, scenario, toolName, metadataDefault)
}

// SetToolApprovalOverride updates the approval override for a tool.
// Pass empty chatID for global configuration.
// Pass empty override to reset to default (use tool metadata).
func (r *ToolRegistry) SetToolApprovalOverride(ctx context.Context, chatID, scenario, toolName string, override domain.ApprovalOverride) error {
	return r.repo.SetToolApprovalOverride(ctx, chatID, scenario, toolName, override)
}
