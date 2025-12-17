// Package livecapture provides action type configuration for workflow generation.
package livecapture

import "fmt"

// ActionTypeConfig defines how an action type should be converted to a workflow node.
// This centralizes action type configuration so adding new action types requires
// only one entry in the registry instead of changes across multiple functions.
type ActionTypeConfig struct {
	// NodeType is the workflow node type to create (e.g., "click", "type", "navigate")
	NodeType string

	// NeedsSelectorWait indicates whether this action type needs a wait-for-selector
	// node inserted before it when preceded by a DOM-changing action.
	NeedsSelectorWait bool

	// TriggersDOMChanges indicates whether this action type might change the DOM,
	// requiring a wait node before the next action.
	TriggersDOMChanges bool

	// BuildNode creates the node data and config for this action type.
	// Returns (data, config) maps.
	BuildNode func(action RecordedAction) (data map[string]any, config map[string]any)

	// GenerateLabel creates a human-readable label for the action.
	GenerateLabel func(action RecordedAction) string
}

// actionRegistry maps action type strings to their configuration.
// This is the single source of truth for action type handling.
var actionRegistry = map[string]ActionTypeConfig{
	"click": {
		NodeType:           "click",
		NeedsSelectorWait:  true,
		TriggersDOMChanges: true,
		BuildNode:          buildClickNode,
		GenerateLabel:      generateClickLabel,
	},
	"type": {
		NodeType:           "type",
		NeedsSelectorWait:  true,
		TriggersDOMChanges: true,
		BuildNode:          buildTypeNode,
		GenerateLabel:      generateTypeLabel,
	},
	"navigate": {
		NodeType:           "navigate",
		NeedsSelectorWait:  false,
		TriggersDOMChanges: true,
		BuildNode:          buildNavigateNode,
		GenerateLabel:      generateNavigateLabel,
	},
	"scroll": {
		NodeType:           "scroll",
		NeedsSelectorWait:  true,
		TriggersDOMChanges: false,
		BuildNode:          buildScrollNode,
		GenerateLabel: func(_ RecordedAction) string {
			return "Scroll"
		},
	},
	"select": {
		NodeType:           "select",
		NeedsSelectorWait:  true,
		TriggersDOMChanges: true,
		BuildNode:          buildSelectNode,
		GenerateLabel: func(_ RecordedAction) string {
			return "Select option"
		},
	},
	"focus": {
		NodeType:           "click", // Focus falls back to click
		NeedsSelectorWait:  true,
		TriggersDOMChanges: false,
		BuildNode:          buildFocusNode,
		GenerateLabel: func(_ RecordedAction) string {
			return "Focus element"
		},
	},
	"hover": {
		NodeType:           "hover",
		NeedsSelectorWait:  true,
		TriggersDOMChanges: false,
		BuildNode:          buildHoverNode,
		GenerateLabel: func(_ RecordedAction) string {
			return "Hover element"
		},
	},
	"keypress": {
		NodeType:           "keyboard",
		NeedsSelectorWait:  false,
		TriggersDOMChanges: true,
		BuildNode:          buildKeypressNode,
		GenerateLabel:      generateKeypressLabel,
	},
	"dragDrop": {
		NodeType:           "dragDrop",
		NeedsSelectorWait:  true,
		TriggersDOMChanges: true,
		BuildNode:          buildDragDropNode,
		GenerateLabel: func(_ RecordedAction) string {
			return "Drag and drop"
		},
	},
}

// GetActionConfig returns the configuration for an action type.
// Returns the default config for unknown action types.
func GetActionConfig(actionType string) ActionTypeConfig {
	if cfg, ok := actionRegistry[actionType]; ok {
		return cfg
	}
	return defaultActionConfig
}

// NeedsSelectorWait returns whether the given action type needs a selector wait.
func NeedsSelectorWait(actionType string) bool {
	if cfg, ok := actionRegistry[actionType]; ok {
		return cfg.NeedsSelectorWait
	}
	return false
}

// TriggersDOMChanges returns whether the given action type might trigger DOM changes.
func TriggersDOMChanges(actionType string) bool {
	if cfg, ok := actionRegistry[actionType]; ok {
		return cfg.TriggersDOMChanges
	}
	return false
}

// defaultActionConfig is used for unknown action types.
var defaultActionConfig = ActionTypeConfig{
	NodeType:           "click",
	NeedsSelectorWait:  false,
	TriggersDOMChanges: false,
	BuildNode:          buildDefaultNode,
	GenerateLabel: func(action RecordedAction) string {
		return action.ActionType
	},
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
