package livecapture

import (
	"testing"
)

func TestGetActionConfig_KnownTypes(t *testing.T) {
	knownTypes := []string{"click", "type", "navigate", "scroll", "select", "focus", "hover", "keypress", "dragDrop"}

	for _, actionType := range knownTypes {
		t.Run(actionType, func(t *testing.T) {
			cfg := GetActionConfig(actionType)
			if cfg.NodeType == "" {
				t.Errorf("GetActionConfig(%q) returned empty NodeType", actionType)
			}
			if cfg.BuildNode == nil {
				t.Errorf("GetActionConfig(%q) returned nil BuildNode", actionType)
			}
			if cfg.GenerateLabel == nil {
				t.Errorf("GetActionConfig(%q) returned nil GenerateLabel", actionType)
			}
		})
	}
}

func TestGetActionConfig_UnknownType(t *testing.T) {
	cfg := GetActionConfig("unknown_action_type")

	// Should return default config
	if cfg.NodeType != "click" {
		t.Errorf("Expected default NodeType 'click', got %q", cfg.NodeType)
	}
	if cfg.NeedsSelectorWait != false {
		t.Errorf("Expected default NeedsSelectorWait false, got %v", cfg.NeedsSelectorWait)
	}
}

func TestNeedsSelectorWait(t *testing.T) {
	tests := []struct {
		actionType string
		expected   bool
	}{
		{"click", true},
		{"type", true},
		{"select", true},
		{"focus", true},
		{"hover", true},
		{"scroll", true},
		{"dragDrop", true},
		{"navigate", false},
		{"keypress", false},
		{"unknown", false},
	}

	for _, tt := range tests {
		t.Run(tt.actionType, func(t *testing.T) {
			result := NeedsSelectorWait(tt.actionType)
			if result != tt.expected {
				t.Errorf("NeedsSelectorWait(%q) = %v, expected %v", tt.actionType, result, tt.expected)
			}
		})
	}
}

func TestTriggersDOMChanges(t *testing.T) {
	tests := []struct {
		actionType string
		expected   bool
	}{
		{"click", true},
		{"type", true},
		{"navigate", true},
		{"select", true},
		{"keypress", true},
		{"dragDrop", true},
		{"scroll", false},
		{"focus", false},
		{"hover", false},
		{"unknown", false},
	}

	for _, tt := range tests {
		t.Run(tt.actionType, func(t *testing.T) {
			result := TriggersDOMChanges(tt.actionType)
			if result != tt.expected {
				t.Errorf("TriggersDOMChanges(%q) = %v, expected %v", tt.actionType, result, tt.expected)
			}
		})
	}
}

func TestBuildClickNode(t *testing.T) {
	action := RecordedAction{
		ActionType: "click",
		Selector:   &SelectorSet{Primary: "#submit-btn"},
		Payload: map[string]interface{}{
			"button":     "left",
			"clickCount": 2,
			"modifiers":  []string{"ctrl"},
		},
	}

	data, config := buildClickNode(action)

	if data["selector"] != "#submit-btn" {
		t.Errorf("Expected selector '#submit-btn', got %v", data["selector"])
	}
	if data["button"] != "left" {
		t.Errorf("Expected button 'left', got %v", data["button"])
	}

	clickConfig, ok := config["click"].(map[string]any)
	if !ok {
		t.Fatal("Expected click config map")
	}
	if clickConfig["selector"] != "#submit-btn" {
		t.Errorf("Expected config selector '#submit-btn', got %v", clickConfig["selector"])
	}
	if clickConfig["click_count"] != 2 {
		t.Errorf("Expected click_count 2, got %v", clickConfig["click_count"])
	}
}

func TestBuildTypeNode(t *testing.T) {
	action := RecordedAction{
		ActionType: "type",
		Selector:   &SelectorSet{Primary: "#email"},
		Payload: map[string]interface{}{
			"text":   "test@example.com",
			"submit": true,
		},
	}

	data, config := buildTypeNode(action)

	if data["selector"] != "#email" {
		t.Errorf("Expected selector '#email', got %v", data["selector"])
	}
	if data["text"] != "test@example.com" {
		t.Errorf("Expected text 'test@example.com', got %v", data["text"])
	}

	inputConfig, ok := config["input"].(map[string]any)
	if !ok {
		t.Fatal("Expected input config map")
	}
	if inputConfig["value"] != "test@example.com" {
		t.Errorf("Expected config value 'test@example.com', got %v", inputConfig["value"])
	}
	if inputConfig["submit"] != true {
		t.Errorf("Expected submit true, got %v", inputConfig["submit"])
	}
}

func TestBuildNavigateNode(t *testing.T) {
	action := RecordedAction{
		ActionType: "navigate",
		URL:        "https://example.com/page",
		Payload: map[string]interface{}{
			"waitForSelector": "#content",
			"timeoutMs":       5000,
		},
	}

	data, config := buildNavigateNode(action)

	if data["url"] != "https://example.com/page" {
		t.Errorf("Expected url 'https://example.com/page', got %v", data["url"])
	}

	navConfig, ok := config["navigate"].(map[string]any)
	if !ok {
		t.Fatal("Expected navigate config map")
	}
	if navConfig["url"] != "https://example.com/page" {
		t.Errorf("Expected config url, got %v", navConfig["url"])
	}
	if navConfig["wait_for_selector"] != "#content" {
		t.Errorf("Expected wait_for_selector '#content', got %v", navConfig["wait_for_selector"])
	}
}

func TestBuildScrollNode(t *testing.T) {
	action := RecordedAction{
		ActionType: "scroll",
		Selector:   &SelectorSet{Primary: "#container"},
		Payload: map[string]interface{}{
			"scrollY": 500.0,
		},
	}

	data, config := buildScrollNode(action)

	if data["selector"] != "#container" {
		t.Errorf("Expected selector '#container', got %v", data["selector"])
	}
	if data["y"] != 500.0 {
		t.Errorf("Expected y 500.0, got %v", data["y"])
	}

	customConfig, ok := config["custom"].(map[string]any)
	if !ok {
		t.Fatal("Expected custom config map")
	}
	if customConfig["kind"] != "scroll" {
		t.Errorf("Expected kind 'scroll', got %v", customConfig["kind"])
	}
}

func TestBuildSelectNode(t *testing.T) {
	action := RecordedAction{
		ActionType: "select",
		Selector:   &SelectorSet{Primary: "#country"},
		Payload: map[string]interface{}{
			"value": "US",
		},
	}

	data, config := buildSelectNode(action)

	if data["selector"] != "#country" {
		t.Errorf("Expected selector '#country', got %v", data["selector"])
	}
	if data["value"] != "US" {
		t.Errorf("Expected value 'US', got %v", data["value"])
	}

	customConfig, ok := config["custom"].(map[string]any)
	if !ok {
		t.Fatal("Expected custom config map")
	}
	if customConfig["kind"] != "select" {
		t.Errorf("Expected kind 'select', got %v", customConfig["kind"])
	}
}

func TestBuildKeypressNode(t *testing.T) {
	action := RecordedAction{
		ActionType: "keypress",
		Payload: map[string]interface{}{
			"key": "Enter",
		},
	}

	data, config := buildKeypressNode(action)

	if data["key"] != "Enter" {
		t.Errorf("Expected key 'Enter', got %v", data["key"])
	}

	customConfig, ok := config["custom"].(map[string]any)
	if !ok {
		t.Fatal("Expected custom config map")
	}
	if customConfig["kind"] != "keypress" {
		t.Errorf("Expected kind 'keypress', got %v", customConfig["kind"])
	}
}

func TestBuildDefaultNode(t *testing.T) {
	action := RecordedAction{
		ActionType: "custom_action",
		Selector:   &SelectorSet{Primary: "#element"},
	}

	data, config := buildDefaultNode(action)

	if data["selector"] != "#element" {
		t.Errorf("Expected selector '#element', got %v", data["selector"])
	}

	customConfig, ok := config["custom"].(map[string]any)
	if !ok {
		t.Fatal("Expected custom config map")
	}
	if customConfig["kind"] != "custom_action" {
		t.Errorf("Expected kind 'custom_action', got %v", customConfig["kind"])
	}
}

func TestGenerateTypeLabel(t *testing.T) {
	tests := []struct {
		name     string
		action   RecordedAction
		expected string
	}{
		{
			name:     "no payload",
			action:   RecordedAction{ActionType: "type"},
			expected: "Type text",
		},
		{
			name: "with text",
			action: RecordedAction{
				ActionType: "type",
				Payload:    map[string]interface{}{"text": "hello"},
			},
			expected: `Type: "hello"`,
		},
		{
			name: "long text truncated",
			action: RecordedAction{
				ActionType: "type",
				Payload:    map[string]interface{}{"text": "this is a very long text that should be truncated"},
			},
			expected: `Type: "this is a very long ..."`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := generateTypeLabel(tt.action)
			if result != tt.expected {
				t.Errorf("generateTypeLabel() = %q, expected %q", result, tt.expected)
			}
		})
	}
}

func TestGenerateNavigateLabel(t *testing.T) {
	action := RecordedAction{
		ActionType: "navigate",
		URL:        "https://example.com",
	}

	result := generateNavigateLabel(action)
	expected := "Navigate to https://example.com"

	if result != expected {
		t.Errorf("generateNavigateLabel() = %q, expected %q", result, expected)
	}
}

func TestGenerateKeypressLabel(t *testing.T) {
	tests := []struct {
		name     string
		action   RecordedAction
		expected string
	}{
		{
			name:     "no payload",
			action:   RecordedAction{ActionType: "keypress"},
			expected: "Press key",
		},
		{
			name: "with key",
			action: RecordedAction{
				ActionType: "keypress",
				Payload:    map[string]interface{}{"key": "Enter"},
			},
			expected: "Press Enter",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := generateKeypressLabel(tt.action)
			if result != tt.expected {
				t.Errorf("generateKeypressLabel() = %q, expected %q", result, tt.expected)
			}
		})
	}
}

func TestMapActionToNode_UsesRegistry(t *testing.T) {
	// Test that mapActionToNode correctly uses the registry
	tests := []struct {
		name             string
		action           RecordedAction
		expectedNodeType string
	}{
		{
			name:             "click action",
			action:           RecordedAction{ActionType: "click", Selector: &SelectorSet{Primary: "#btn"}},
			expectedNodeType: "click",
		},
		{
			name:             "type action",
			action:           RecordedAction{ActionType: "type", Selector: &SelectorSet{Primary: "#input"}, Payload: map[string]interface{}{"text": "test"}},
			expectedNodeType: "type",
		},
		{
			name:             "navigate action",
			action:           RecordedAction{ActionType: "navigate", URL: "https://example.com"},
			expectedNodeType: "navigate",
		},
		{
			name:             "scroll action",
			action:           RecordedAction{ActionType: "scroll", Payload: map[string]interface{}{"scrollY": 100.0}},
			expectedNodeType: "scroll",
		},
		{
			name:             "keypress action",
			action:           RecordedAction{ActionType: "keypress", Payload: map[string]interface{}{"key": "Enter"}},
			expectedNodeType: "keyboard",
		},
		{
			name:             "unknown action falls back to click",
			action:           RecordedAction{ActionType: "unknown_type"},
			expectedNodeType: "click",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			node := mapActionToNode(tt.action, "node_1", 0)

			nodeType, ok := node["type"].(string)
			if !ok {
				t.Fatal("Expected node type to be string")
			}
			if nodeType != tt.expectedNodeType {
				t.Errorf("Expected node type %q, got %q", tt.expectedNodeType, nodeType)
			}

			// Verify node has required fields
			if node["id"] != "node_1" {
				t.Errorf("Expected node id 'node_1', got %v", node["id"])
			}
			if node["position"] == nil {
				t.Error("Expected node to have position")
			}
			if node["data"] == nil {
				t.Error("Expected node to have data")
			}

			// Verify label is generated
			data, ok := node["data"].(map[string]any)
			if !ok {
				t.Fatal("Expected data to be map")
			}
			if data["label"] == nil || data["label"] == "" {
				t.Error("Expected node to have a label")
			}
		})
	}
}

func TestEnsureConfigKey(t *testing.T) {
	config := map[string]any{}

	// First call should create the key
	result := ensureConfigKey(config, "click")
	result["selector"] = "#btn"

	// Second call should return existing map
	result2 := ensureConfigKey(config, "click")
	if result2["selector"] != "#btn" {
		t.Errorf("Expected existing map to be returned, but selector was %v", result2["selector"])
	}

	// Different key should create new map
	result3 := ensureConfigKey(config, "input")
	result3["value"] = "test"
	// Verify they are separate maps by checking the click map wasn't modified
	if result["value"] != nil {
		t.Error("Expected different map for different key - click map was modified")
	}
}
