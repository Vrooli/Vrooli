// Package toolregistry provides the tool discovery service for system-monitor.
//
// This file defines the configuration tools that are exposed via the Tool Discovery Protocol.
package toolregistry

import (
	"context"

	toolspb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-inbox/v1/domain"
)

// ConfigurationToolProvider provides the monitoring configuration tools.
type ConfigurationToolProvider struct{}

// NewConfigurationToolProvider creates a new ConfigurationToolProvider.
func NewConfigurationToolProvider() *ConfigurationToolProvider {
	return &ConfigurationToolProvider{}
}

// Name returns the provider identifier.
func (p *ConfigurationToolProvider) Name() string {
	return "system-monitor-configuration"
}

// Categories returns the tool categories for configuration tools.
func (p *ConfigurationToolProvider) Categories(_ context.Context) []*toolspb.ToolCategory {
	return []*toolspb.ToolCategory{
		{
			Id:          "configuration",
			Name:        "Configuration",
			Description: "Tools for managing investigation triggers and monitoring settings",
			Icon:        "settings",
		},
	}
}

// Tools returns the tool definitions for configuration tools.
func (p *ConfigurationToolProvider) Tools(_ context.Context) []*toolspb.ToolDefinition {
	return []*toolspb.ToolDefinition{
		p.getTriggersTool(),
		p.updateTriggerTool(),
		p.getCooldownStatusTool(),
		p.resetCooldownTool(),
	}
}

func (p *ConfigurationToolProvider) getTriggersTool() *toolspb.ToolDefinition {
	return &toolspb.ToolDefinition{
		Name:        "get_triggers",
		Description: "Get all configured investigation triggers. Triggers define thresholds that automatically start investigations when exceeded (e.g., high CPU usage, memory pressure, low disk space).",
		Category:    "configuration",
		Parameters:  NewEmptyParams(),
		Metadata: &toolspb.ToolMetadata{
			EnabledByDefault:   true,
			RequiresApproval:   false,
			TimeoutSeconds:     10,
			RateLimitPerMinute: 60,
			CostEstimate:       "low",
			LongRunning:        false,
			Idempotent:         true,
			Tags:               []string{"triggers", "configuration", "thresholds"},
			Examples: []*toolspb.ToolExample{
				NewToolExample(
					"List all investigation triggers",
					map[string]interface{}{},
				),
			},
		},
	}
}

func (p *ConfigurationToolProvider) updateTriggerTool() *toolspb.ToolDefinition {
	return &toolspb.ToolDefinition{
		Name:        "update_trigger",
		Description: "Update an investigation trigger's configuration. You can enable/disable triggers, toggle auto-fix behavior, or adjust threshold values.",
		Category:    "configuration",
		Parameters: NewObjectParams(
			map[string]*toolspb.ParameterSchema{
				"trigger_id": NewStringParam(
					"The ID of the trigger to update. Available triggers: high_cpu, memory_pressure, disk_space, network_connections, process_anomaly",
				),
				"enabled": NewBoolParam(
					"Whether this trigger is enabled. Disabled triggers won't start automatic investigations.",
				),
				"auto_fix": NewBoolParam(
					"Whether investigations triggered by this should automatically apply fixes.",
				),
				"threshold": NewFloatParam(
					"The threshold value that triggers an investigation. The meaning depends on the trigger type (e.g., 85.0 for 85% CPU usage).",
				),
			},
			[]string{"trigger_id"},
		),
		Metadata: &toolspb.ToolMetadata{
			EnabledByDefault:   true,
			RequiresApproval:   true, // Modifying monitoring behavior requires approval
			TimeoutSeconds:     10,
			RateLimitPerMinute: 30,
			CostEstimate:       "low",
			LongRunning:        false,
			Idempotent:         true,
			ModifiesState:      true,
			Tags:               []string{"triggers", "configuration", "thresholds", "settings"},
			Examples: []*toolspb.ToolExample{
				NewToolExample(
					"Enable high CPU trigger with 90% threshold",
					map[string]interface{}{
						"trigger_id": "high_cpu",
						"enabled":    true,
						"threshold":  90.0,
					},
				),
				NewToolExample(
					"Disable network connections trigger",
					map[string]interface{}{
						"trigger_id": "network_connections",
						"enabled":    false,
					},
				),
				NewToolExample(
					"Enable auto-fix for memory pressure",
					map[string]interface{}{
						"trigger_id": "memory_pressure",
						"auto_fix":   true,
					},
				),
			},
		},
	}
}

func (p *ConfigurationToolProvider) getCooldownStatusTool() *toolspb.ToolDefinition {
	return &toolspb.ToolDefinition{
		Name:        "get_cooldown_status",
		Description: "Check the investigation cooldown status. After triggering an investigation, there's a cooldown period before another can be started. This prevents spam and allows previous investigations to complete.",
		Category:    "configuration",
		Parameters:  NewEmptyParams(),
		Metadata: &toolspb.ToolMetadata{
			EnabledByDefault:   true,
			RequiresApproval:   false,
			TimeoutSeconds:     10,
			RateLimitPerMinute: 60,
			CostEstimate:       "low",
			LongRunning:        false,
			Idempotent:         true,
			Tags:               []string{"cooldown", "rate-limiting", "status"},
			Examples: []*toolspb.ToolExample{
				NewToolExample(
					"Check if cooldown is active",
					map[string]interface{}{},
				),
			},
		},
	}
}

func (p *ConfigurationToolProvider) resetCooldownTool() *toolspb.ToolDefinition {
	return &toolspb.ToolDefinition{
		Name:        "reset_cooldown",
		Description: "Reset the investigation cooldown timer, allowing an investigation to be triggered immediately. Use this after maintenance or when you need to run an investigation before the cooldown expires.",
		Category:    "configuration",
		Parameters:  NewEmptyParams(),
		Metadata: &toolspb.ToolMetadata{
			EnabledByDefault:   true,
			RequiresApproval:   false, // Low risk operation
			TimeoutSeconds:     10,
			RateLimitPerMinute: 10,
			CostEstimate:       "low",
			LongRunning:        false,
			Idempotent:         true,
			ModifiesState:      true,
			Tags:               []string{"cooldown", "rate-limiting", "reset"},
			Examples: []*toolspb.ToolExample{
				NewToolExample(
					"Reset the cooldown to allow immediate investigation",
					map[string]interface{}{},
				),
			},
		},
	}
}
