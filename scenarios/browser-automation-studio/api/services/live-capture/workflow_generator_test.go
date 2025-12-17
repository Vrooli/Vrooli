package livecapture

import (
	"testing"
)

func TestMergeConsecutiveActions_EmptySlice(t *testing.T) {
	result := MergeConsecutiveActions(nil)
	if result != nil {
		t.Errorf("Expected nil for nil input, got %v", result)
	}

	result = MergeConsecutiveActions([]RecordedAction{})
	if len(result) != 0 {
		t.Errorf("Expected empty slice for empty input, got %v", result)
	}
}

func TestMergeConsecutiveActions_SingleAction(t *testing.T) {
	actions := []RecordedAction{
		{ActionType: "click", Selector: &SelectorSet{Primary: "#btn"}},
	}
	result := MergeConsecutiveActions(actions)
	if len(result) != 1 {
		t.Errorf("Expected 1 action, got %d", len(result))
	}
}

func TestMergeConsecutiveActions_MergesConsecutiveTypeActions(t *testing.T) {
	selector := &SelectorSet{Primary: "#input"}
	actions := []RecordedAction{
		{ActionType: "type", Selector: selector, Payload: map[string]interface{}{"text": "Hello"}},
		{ActionType: "type", Selector: selector, Payload: map[string]interface{}{"text": " "}},
		{ActionType: "type", Selector: selector, Payload: map[string]interface{}{"text": "World"}},
	}

	result := MergeConsecutiveActions(actions)

	if len(result) != 1 {
		t.Fatalf("Expected 1 merged action, got %d", len(result))
	}
	if result[0].Payload["text"] != "Hello World" {
		t.Errorf("Expected merged text 'Hello World', got %v", result[0].Payload["text"])
	}
}

func TestMergeConsecutiveActions_DoesNotMergeTypeOnDifferentSelectors(t *testing.T) {
	actions := []RecordedAction{
		{ActionType: "type", Selector: &SelectorSet{Primary: "#input1"}, Payload: map[string]interface{}{"text": "First"}},
		{ActionType: "type", Selector: &SelectorSet{Primary: "#input2"}, Payload: map[string]interface{}{"text": "Second"}},
	}

	result := MergeConsecutiveActions(actions)

	if len(result) != 2 {
		t.Errorf("Expected 2 actions (different selectors), got %d", len(result))
	}
}

func TestMergeConsecutiveActions_MergesConsecutiveScrollActions(t *testing.T) {
	actions := []RecordedAction{
		{ActionType: "scroll", Payload: map[string]interface{}{"scrollY": 100.0}},
		{ActionType: "scroll", Payload: map[string]interface{}{"scrollY": 200.0}},
		{ActionType: "scroll", Payload: map[string]interface{}{"scrollY": 500.0}},
	}

	result := MergeConsecutiveActions(actions)

	if len(result) != 1 {
		t.Fatalf("Expected 1 merged scroll action, got %d", len(result))
	}
	if result[0].Payload["scrollY"] != 500.0 {
		t.Errorf("Expected final scrollY 500.0, got %v", result[0].Payload["scrollY"])
	}
}

func TestMergeConsecutiveActions_SkipsFocusBeforeType(t *testing.T) {
	selector := &SelectorSet{Primary: "#input"}
	actions := []RecordedAction{
		{ActionType: "focus", Selector: selector},
		{ActionType: "type", Selector: selector, Payload: map[string]interface{}{"text": "test"}},
	}

	result := MergeConsecutiveActions(actions)

	if len(result) != 1 {
		t.Fatalf("Expected 1 action (focus skipped), got %d", len(result))
	}
	if result[0].ActionType != "type" {
		t.Errorf("Expected type action, got %s", result[0].ActionType)
	}
}

func TestMergeConsecutiveActions_KeepsFocusWithoutFollowingType(t *testing.T) {
	selector := &SelectorSet{Primary: "#input"}
	actions := []RecordedAction{
		{ActionType: "focus", Selector: selector},
		{ActionType: "click", Selector: &SelectorSet{Primary: "#btn"}},
	}

	result := MergeConsecutiveActions(actions)

	if len(result) != 2 {
		t.Errorf("Expected 2 actions (focus kept), got %d", len(result))
	}
}

func TestMergeConsecutiveActions_MixedActions(t *testing.T) {
	inputSelector := &SelectorSet{Primary: "#input"}
	actions := []RecordedAction{
		{ActionType: "click", Selector: &SelectorSet{Primary: "#btn"}},
		{ActionType: "focus", Selector: inputSelector},
		{ActionType: "type", Selector: inputSelector, Payload: map[string]interface{}{"text": "Hello"}},
		{ActionType: "type", Selector: inputSelector, Payload: map[string]interface{}{"text": " World"}},
		{ActionType: "scroll", Payload: map[string]interface{}{"scrollY": 100.0}},
		{ActionType: "scroll", Payload: map[string]interface{}{"scrollY": 300.0}},
		{ActionType: "click", Selector: &SelectorSet{Primary: "#submit"}},
	}

	result := MergeConsecutiveActions(actions)

	// Expected: click, type (merged, focus skipped), scroll (merged), click
	if len(result) != 4 {
		t.Fatalf("Expected 4 actions after merging, got %d", len(result))
	}

	if result[0].ActionType != "click" {
		t.Errorf("Action 0: expected click, got %s", result[0].ActionType)
	}
	if result[1].ActionType != "type" || result[1].Payload["text"] != "Hello World" {
		t.Errorf("Action 1: expected merged type with 'Hello World', got %v", result[1])
	}
	if result[2].ActionType != "scroll" || result[2].Payload["scrollY"] != 300.0 {
		t.Errorf("Action 2: expected merged scroll with 300.0, got %v", result[2])
	}
	if result[3].ActionType != "click" {
		t.Errorf("Action 3: expected click, got %s", result[3].ActionType)
	}
}

func TestApplyActionRange_EmptySlice(t *testing.T) {
	result := ApplyActionRange(nil, 0, 5)
	if result != nil {
		t.Errorf("Expected nil for nil input, got %v", result)
	}

	result = ApplyActionRange([]RecordedAction{}, 0, 5)
	if len(result) != 0 {
		t.Errorf("Expected empty slice, got %v", result)
	}
}

func TestApplyActionRange_ValidRange(t *testing.T) {
	actions := []RecordedAction{
		{ActionType: "a"},
		{ActionType: "b"},
		{ActionType: "c"},
		{ActionType: "d"},
		{ActionType: "e"},
	}

	result := ApplyActionRange(actions, 1, 3)

	if len(result) != 3 {
		t.Fatalf("Expected 3 actions, got %d", len(result))
	}
	if result[0].ActionType != "b" || result[1].ActionType != "c" || result[2].ActionType != "d" {
		t.Errorf("Expected [b,c,d], got %v", result)
	}
}

func TestApplyActionRange_ClampsNegativeStart(t *testing.T) {
	actions := []RecordedAction{
		{ActionType: "a"},
		{ActionType: "b"},
		{ActionType: "c"},
	}

	result := ApplyActionRange(actions, -5, 1)

	if len(result) != 2 {
		t.Fatalf("Expected 2 actions, got %d", len(result))
	}
	if result[0].ActionType != "a" {
		t.Errorf("Expected first action 'a', got %s", result[0].ActionType)
	}
}

func TestApplyActionRange_ClampsEndBeyondLength(t *testing.T) {
	actions := []RecordedAction{
		{ActionType: "a"},
		{ActionType: "b"},
		{ActionType: "c"},
	}

	result := ApplyActionRange(actions, 1, 100)

	if len(result) != 2 {
		t.Fatalf("Expected 2 actions, got %d", len(result))
	}
	if result[0].ActionType != "b" || result[1].ActionType != "c" {
		t.Errorf("Expected [b,c], got %v", result)
	}
}

func TestApplyActionRange_FullRange(t *testing.T) {
	actions := []RecordedAction{
		{ActionType: "a"},
		{ActionType: "b"},
		{ActionType: "c"},
	}

	result := ApplyActionRange(actions, 0, 2)

	if len(result) != 3 {
		t.Errorf("Expected all 3 actions, got %d", len(result))
	}
}

func TestSelectorsMatch(t *testing.T) {
	tests := []struct {
		name     string
		a        *SelectorSet
		b        *SelectorSet
		expected bool
	}{
		{
			name:     "both nil",
			a:        nil,
			b:        nil,
			expected: false,
		},
		{
			name:     "first nil",
			a:        nil,
			b:        &SelectorSet{Primary: "#test"},
			expected: false,
		},
		{
			name:     "second nil",
			a:        &SelectorSet{Primary: "#test"},
			b:        nil,
			expected: false,
		},
		{
			name:     "matching selectors",
			a:        &SelectorSet{Primary: "#input"},
			b:        &SelectorSet{Primary: "#input"},
			expected: true,
		},
		{
			name:     "different selectors",
			a:        &SelectorSet{Primary: "#input1"},
			b:        &SelectorSet{Primary: "#input2"},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := selectorsMatch(tt.a, tt.b)
			if result != tt.expected {
				t.Errorf("selectorsMatch(%v, %v) = %v, expected %v", tt.a, tt.b, result, tt.expected)
			}
		})
	}
}

func TestGenerateWorkflow_CreatesNodesAndEdges(t *testing.T) {
	gen := NewWorkflowGenerator()
	actions := []RecordedAction{
		{ActionType: "navigate", URL: "https://example.com"},
		{ActionType: "click", Selector: &SelectorSet{Primary: "#btn"}},
	}

	result := gen.GenerateWorkflow(actions)

	nodes, ok := result["nodes"].([]map[string]interface{})
	if !ok {
		t.Fatal("Expected nodes to be []map[string]interface{}")
	}

	edges, ok := result["edges"].([]map[string]interface{})
	if !ok {
		t.Fatal("Expected edges to be []map[string]interface{}")
	}

	// Should have at least 2 action nodes (may have wait nodes inserted)
	if len(nodes) < 2 {
		t.Errorf("Expected at least 2 nodes, got %d", len(nodes))
	}

	// Should have at least 1 edge connecting them
	if len(edges) < 1 {
		t.Errorf("Expected at least 1 edge, got %d", len(edges))
	}
}

func TestGenerateWorkflow_EmptyActions(t *testing.T) {
	gen := NewWorkflowGenerator()
	result := gen.GenerateWorkflow([]RecordedAction{})

	// Verify the result has nodes and edges keys (may be nil or empty)
	if _, ok := result["nodes"]; !ok {
		t.Error("Expected result to have 'nodes' key")
	}
	if _, ok := result["edges"]; !ok {
		t.Error("Expected result to have 'edges' key")
	}
}

func TestTruncateString(t *testing.T) {
	tests := []struct {
		input    string
		maxLen   int
		expected string
	}{
		{"hello", 10, "hello"},
		{"hello world", 5, "hello..."},
		{"", 5, ""},
		{"abc", 3, "abc"},
		{"abcd", 3, "abc..."},
	}

	for _, tt := range tests {
		result := truncateString(tt.input, tt.maxLen)
		if result != tt.expected {
			t.Errorf("truncateString(%q, %d) = %q, expected %q", tt.input, tt.maxLen, result, tt.expected)
		}
	}
}

func TestExtractHostname(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"https://example.com", "https://example.com"},
		{"https://example.com/path", "https://example.com/path"},
		{"https://very-long-domain-name-that-exceeds-fifty-characters.example.com/path", "https://very-long-domain-name-that-exceeds-fifty-c..."},
	}

	for _, tt := range tests {
		result := extractHostname(tt.input)
		if result != tt.expected {
			t.Errorf("extractHostname(%q) = %q, expected %q", tt.input, result, tt.expected)
		}
	}
}

func TestGenerateClickLabel(t *testing.T) {
	tests := []struct {
		name     string
		action   RecordedAction
		expected string
	}{
		{
			name:     "no element meta",
			action:   RecordedAction{ActionType: "click"},
			expected: "Click element",
		},
		{
			name: "with inner text",
			action: RecordedAction{
				ActionType:  "click",
				ElementMeta: &ElementMeta{InnerText: "Submit", TagName: "BUTTON"},
			},
			expected: "Click: Submit",
		},
		{
			name: "with aria label",
			action: RecordedAction{
				ActionType:  "click",
				ElementMeta: &ElementMeta{AriaLabel: "Close dialog", TagName: "BUTTON"},
			},
			expected: "Click: Close dialog",
		},
		{
			name: "with tag name only",
			action: RecordedAction{
				ActionType:  "click",
				ElementMeta: &ElementMeta{TagName: "BUTTON"},
			},
			expected: "Click BUTTON",
		},
		{
			name: "long inner text truncated",
			action: RecordedAction{
				ActionType:  "click",
				ElementMeta: &ElementMeta{InnerText: "This is a very long button text that should be truncated"},
			},
			expected: "Click: This is a very long ...",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := generateClickLabel(tt.action)
			if result != tt.expected {
				t.Errorf("generateClickLabel() = %q, expected %q", result, tt.expected)
			}
		})
	}
}
