// Package livecapture provides action type configuration for workflow generation.
// This file provides recording-specific functionality (node building, label generation)
// while using the unified automation/actions package for action type definitions
// and behavioral metadata.
package livecapture

import (
	"fmt"

	"github.com/vrooli/browser-automation-studio/automation/actions"
)

// ActionNodeConfig defines how to convert a recorded action type to a workflow node.
// This includes the node type mapping, data building, and label generation.
// Behavioral metadata (NeedsSelectorWait, TriggersDOMChanges) comes from the
// unified actions.Registry.
type ActionNodeConfig struct {
	// NodeType is the workflow node type to create (e.g., "click", "type", "navigate")
	NodeType string

	// BuildNode creates the node data and config for this action type.
	// Returns (data, config) maps.
	BuildNode func(action RecordedAction) (data map[string]any, config map[string]any)

	// GenerateLabel creates a human-readable label for the action.
	GenerateLabel func(action RecordedAction) string
}

// actionNodeRegistry maps action types to their node conversion configuration.
// Behavioral metadata is retrieved from the unified actions.Registry.
var actionNodeRegistry = map[actions.ActionType]ActionNodeConfig{
	actions.Click: {
		NodeType:      "click",
		BuildNode:     buildClickNode,
		GenerateLabel: generateClickLabel,
	},
	actions.TypeInput: {
		NodeType:      "type",
		BuildNode:     buildTypeNode,
		GenerateLabel: generateTypeLabel,
	},
	actions.Navigate: {
		NodeType:      "navigate",
		BuildNode:     buildNavigateNode,
		GenerateLabel: generateNavigateLabel,
	},
	actions.Scroll: {
		NodeType:  "scroll",
		BuildNode: buildScrollNode,
		GenerateLabel: func(_ RecordedAction) string {
			return "Scroll"
		},
	},
	actions.Select: {
		NodeType:  "select",
		BuildNode: buildSelectNode,
		GenerateLabel: func(_ RecordedAction) string {
			return "Select option"
		},
	},
	actions.Focus: {
		NodeType:  "click", // Focus falls back to click for execution
		BuildNode: buildFocusNode,
		GenerateLabel: func(_ RecordedAction) string {
			return "Focus element"
		},
	},
	actions.Hover: {
		NodeType:  "hover",
		BuildNode: buildHoverNode,
		GenerateLabel: func(_ RecordedAction) string {
			return "Hover element"
		},
	},
	actions.Keyboard: {
		NodeType:      "keyboard",
		BuildNode:     buildKeypressNode,
		GenerateLabel: generateKeypressLabel,
	},
	actions.DragDrop: {
		NodeType:  "dragDrop",
		BuildNode: buildDragDropNode,
		GenerateLabel: func(_ RecordedAction) string {
			return "Drag and drop"
		},
	},
	actions.Wait: {
		NodeType:  "wait",
		BuildNode: buildWaitNode,
		GenerateLabel: func(_ RecordedAction) string {
			return "Wait"
		},
	},
	actions.Assert: {
		NodeType:  "assert",
		BuildNode: buildAssertNode,
		GenerateLabel: func(_ RecordedAction) string {
			return "Assert"
		},
	},
	actions.Screenshot: {
		NodeType:  "screenshot",
		BuildNode: buildScreenshotNode,
		GenerateLabel: func(_ RecordedAction) string {
			return "Screenshot"
		},
	},
}

// defaultActionNodeConfig is used for action types without specific node config.
var defaultActionNodeConfig = ActionNodeConfig{
	NodeType:  "click",
	BuildNode: buildDefaultNode,
	GenerateLabel: func(action RecordedAction) string {
		return action.ActionType
	},
}

// GetActionNodeConfig returns the node conversion configuration for an action type.
// Use actions.GetMetadata for behavioral metadata (NeedsSelectorWait, etc.).
func GetActionNodeConfig(actionType actions.ActionType) ActionNodeConfig {
	if cfg, ok := actionNodeRegistry[actionType]; ok {
		return cfg
	}
	return defaultActionNodeConfig
}

// NeedsSelectorWait returns whether the given action type needs a selector wait.
// Delegates to the unified actions package.
func NeedsSelectorWait(actionType string) bool {
	return actions.NeedsSelectorWait(actions.ActionType(actionType))
}

// TriggersDOMChanges returns whether the given action type might trigger DOM changes.
// Delegates to the unified actions package.
func TriggersDOMChanges(actionType string) bool {
	return actions.TriggersDOMChanges(actions.ActionType(actionType))
}

// Node builder functions

func buildClickNode(action RecordedAction) (map[string]any, map[string]any) {
	data := map[string]any{}
	config := map[string]any{}

	if action.Selector != nil {
		data["selector"] = action.Selector.Primary
		config["click"] = map[string]any{
			"selector": action.Selector.Primary,
		}
	}
	if action.Payload != nil {
		if btn, ok := action.Payload["button"]; ok {
			data["button"] = btn
			ensureConfigKey(config, "click")["button"] = btn
		}
		if mods, ok := action.Payload["modifiers"]; ok {
			data["modifiers"] = mods
		}
		if count, ok := action.Payload["clickCount"]; ok {
			ensureConfigKey(config, "click")["click_count"] = count
		}
		if delay, ok := action.Payload["delay"]; ok {
			ensureConfigKey(config, "click")["delay_ms"] = delay
		}
	}
	return data, config
}

func buildTypeNode(action RecordedAction) (map[string]any, map[string]any) {
	data := map[string]any{}
	config := map[string]any{}

	if action.Selector != nil {
		data["selector"] = action.Selector.Primary
		config["input"] = map[string]any{
			"selector": action.Selector.Primary,
		}
	}
	if action.Payload != nil {
		if text, ok := action.Payload["text"].(string); ok {
			data["text"] = text
			ensureConfigKey(config, "input")["value"] = text
		}
		if submit, ok := action.Payload["submit"]; ok {
			ensureConfigKey(config, "input")["submit"] = submit
		}
	}
	return data, config
}

func buildNavigateNode(action RecordedAction) (map[string]any, map[string]any) {
	data := map[string]any{
		"url": action.URL,
	}
	config := map[string]any{
		"navigate": map[string]any{
			"url": action.URL,
		},
	}
	if action.Payload != nil {
		if waitFor, ok := action.Payload["waitForSelector"]; ok {
			ensureConfigKey(config, "navigate")["wait_for_selector"] = waitFor
		}
		if timeout, ok := action.Payload["timeoutMs"]; ok {
			ensureConfigKey(config, "navigate")["timeout_ms"] = timeout
		}
	}
	return data, config
}

func buildScrollNode(action RecordedAction) (map[string]any, map[string]any) {
	data := map[string]any{}
	if action.Selector != nil {
		data["selector"] = action.Selector.Primary
	}
	if action.Payload != nil {
		if y, ok := action.Payload["scrollY"].(float64); ok {
			data["y"] = y
		}
	}
	config := map[string]any{
		"custom": map[string]any{
			"kind": "scroll",
			"payload": map[string]any{
				"y": data["y"],
			},
		},
	}
	return data, config
}

func buildSelectNode(action RecordedAction) (map[string]any, map[string]any) {
	data := map[string]any{}
	if action.Selector != nil {
		data["selector"] = action.Selector.Primary
	}
	if action.Payload != nil {
		if val, ok := action.Payload["value"]; ok {
			data["value"] = val
		}
	}
	config := map[string]any{
		"custom": map[string]any{
			"kind":    "select",
			"payload": data,
		},
	}
	return data, config
}

func buildFocusNode(action RecordedAction) (map[string]any, map[string]any) {
	data := map[string]any{}
	config := map[string]any{}
	if action.Selector != nil {
		data["selector"] = action.Selector.Primary
		config["click"] = map[string]any{
			"selector": action.Selector.Primary,
		}
	}
	return data, config
}

func buildHoverNode(action RecordedAction) (map[string]any, map[string]any) {
	data := map[string]any{}
	config := map[string]any{}
	if action.Selector != nil {
		data["selector"] = action.Selector.Primary
		config["hover"] = map[string]any{
			"selector": action.Selector.Primary,
		}
	}
	return data, config
}

func buildKeypressNode(action RecordedAction) (map[string]any, map[string]any) {
	data := map[string]any{}
	if action.Payload != nil {
		if key, ok := action.Payload["key"].(string); ok {
			data["key"] = key
		}
	}
	config := map[string]any{
		"custom": map[string]any{
			"kind":    "keypress",
			"payload": data,
		},
	}
	return data, config
}

func buildDragDropNode(action RecordedAction) (map[string]any, map[string]any) {
	data := map[string]any{}
	config := map[string]any{}
	if action.Selector != nil {
		data["selector"] = action.Selector.Primary
	}
	if action.Payload != nil {
		if target, ok := action.Payload["targetSelector"].(string); ok {
			data["targetSelector"] = target
		}
	}
	config["custom"] = map[string]any{
		"kind":    "dragDrop",
		"payload": data,
	}
	return data, config
}

func buildWaitNode(action RecordedAction) (map[string]any, map[string]any) {
	data := map[string]any{}
	if action.Selector != nil {
		data["selector"] = action.Selector.Primary
	}
	if action.Payload != nil {
		if timeout, ok := action.Payload["timeoutMs"]; ok {
			data["timeoutMs"] = timeout
		}
	}
	config := map[string]any{
		"custom": map[string]any{
			"kind":    "wait",
			"payload": data,
		},
	}
	return data, config
}

func buildAssertNode(action RecordedAction) (map[string]any, map[string]any) {
	data := map[string]any{}
	if action.Selector != nil {
		data["selector"] = action.Selector.Primary
	}
	if action.Payload != nil {
		for k, v := range action.Payload {
			data[k] = v
		}
	}
	config := map[string]any{
		"custom": map[string]any{
			"kind":    "assert",
			"payload": data,
		},
	}
	return data, config
}

func buildScreenshotNode(action RecordedAction) (map[string]any, map[string]any) {
	data := map[string]any{}
	if action.Payload != nil {
		if name, ok := action.Payload["name"].(string); ok {
			data["name"] = name
		}
	}
	config := map[string]any{
		"custom": map[string]any{
			"kind":    "screenshot",
			"payload": data,
		},
	}
	return data, config
}

func buildDefaultNode(action RecordedAction) (map[string]any, map[string]any) {
	data := map[string]any{}
	if action.Selector != nil {
		data["selector"] = action.Selector.Primary
	}
	config := map[string]any{
		"custom": map[string]any{
			"kind":    action.ActionType,
			"payload": data,
		},
	}
	return data, config
}

// Label generator functions

func generateTypeLabel(action RecordedAction) string {
	if action.Payload != nil {
		if text, ok := action.Payload["text"].(string); ok {
			return fmt.Sprintf("Type: %q", truncateString(text, 20))
		}
	}
	return "Type text"
}

func generateNavigateLabel(action RecordedAction) string {
	return fmt.Sprintf("Navigate to %s", extractHostname(action.URL))
}

func generateKeypressLabel(action RecordedAction) string {
	if action.Payload != nil {
		if key, ok := action.Payload["key"].(string); ok {
			return fmt.Sprintf("Press %s", key)
		}
	}
	return "Press key"
}

// ensureConfigKey ensures a nested config key exists and returns the inner map.
func ensureConfigKey(config map[string]any, key string) map[string]any {
	if existing, ok := config[key]; ok {
		if typed, ok := existing.(map[string]any); ok {
			return typed
		}
	}
	typed := map[string]any{}
	config[key] = typed
	return typed
}
