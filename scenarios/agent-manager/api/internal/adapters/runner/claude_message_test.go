package runner

import (
	"encoding/json"
	"testing"
)

// =============================================================================
// CLAUDE MESSAGE CONTENT EXTRACTION TESTS
// =============================================================================

func TestClaudeMessage_ExtractTextContent_StringContent(t *testing.T) {
	// Test simple string content
	msg := &ClaudeMessage{
		Role:    "assistant",
		Content: json.RawMessage(`"Hello, world!"`),
	}

	got := msg.ExtractTextContent()
	if got != "Hello, world!" {
		t.Errorf("ExtractTextContent() = %q, want %q", got, "Hello, world!")
	}
}

func TestClaudeMessage_ExtractTextContent_ArrayContent(t *testing.T) {
	// Test array content with text blocks
	content := `[
		{"type": "text", "text": "First part"},
		{"type": "text", "text": "Second part"}
	]`
	msg := &ClaudeMessage{
		Role:    "assistant",
		Content: json.RawMessage(content),
	}

	got := msg.ExtractTextContent()
	want := "First part\nSecond part"
	if got != want {
		t.Errorf("ExtractTextContent() = %q, want %q", got, want)
	}
}

func TestClaudeMessage_ExtractTextContent_MixedContent(t *testing.T) {
	// Test array content with mixed text and tool_use blocks
	content := `[
		{"type": "text", "text": "Let me help you with that."},
		{"type": "tool_use", "id": "tool123", "name": "Bash", "input": {"command": "ls"}},
		{"type": "text", "text": "Here are the results."}
	]`
	msg := &ClaudeMessage{
		Role:    "assistant",
		Content: json.RawMessage(content),
	}

	got := msg.ExtractTextContent()
	want := "Let me help you with that.\nHere are the results."
	if got != want {
		t.Errorf("ExtractTextContent() = %q, want %q", got, want)
	}
}

func TestClaudeMessage_ExtractTextContent_EmptyContent(t *testing.T) {
	tests := []struct {
		name    string
		content json.RawMessage
	}{
		{"nil content", nil},
		{"empty content", json.RawMessage(``)},
		{"empty string", json.RawMessage(`""`)},
		{"empty array", json.RawMessage(`[]`)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := &ClaudeMessage{
				Role:    "assistant",
				Content: tt.content,
			}

			got := msg.ExtractTextContent()
			if got != "" {
				t.Errorf("ExtractTextContent() = %q, want empty string", got)
			}
		})
	}
}

func TestClaudeMessage_ExtractTextContent_OnlyToolUse(t *testing.T) {
	// Test array content with only tool_use blocks (no text)
	content := `[
		{"type": "tool_use", "id": "tool123", "name": "Bash", "input": {"command": "ls"}}
	]`
	msg := &ClaudeMessage{
		Role:    "assistant",
		Content: json.RawMessage(content),
	}

	got := msg.ExtractTextContent()
	if got != "" {
		t.Errorf("ExtractTextContent() = %q, want empty string", got)
	}
}

// =============================================================================
// CLAUDE MESSAGE TOOL USE EXTRACTION TESTS
// =============================================================================

func TestClaudeMessage_ExtractToolUses_WithToolUse(t *testing.T) {
	content := `[
		{"type": "text", "text": "Let me run that command."},
		{"type": "tool_use", "id": "tool123", "name": "Bash", "input": {"command": "ls -la", "description": "List files"}},
		{"type": "tool_use", "id": "tool456", "name": "Read", "input": {"file_path": "/tmp/test.txt"}}
	]`
	msg := &ClaudeMessage{
		Role:    "assistant",
		Content: json.RawMessage(content),
	}

	tools := msg.ExtractToolUses()
	if len(tools) != 2 {
		t.Fatalf("ExtractToolUses() returned %d tools, want 2", len(tools))
	}

	if tools[0].Name != "Bash" {
		t.Errorf("tools[0].Name = %q, want %q", tools[0].Name, "Bash")
	}
	if tools[0].ID != "tool123" {
		t.Errorf("tools[0].ID = %q, want %q", tools[0].ID, "tool123")
	}

	if tools[1].Name != "Read" {
		t.Errorf("tools[1].Name = %q, want %q", tools[1].Name, "Read")
	}
}

func TestClaudeMessage_ExtractToolUses_NoToolUse(t *testing.T) {
	content := `[
		{"type": "text", "text": "Just some text."}
	]`
	msg := &ClaudeMessage{
		Role:    "assistant",
		Content: json.RawMessage(content),
	}

	tools := msg.ExtractToolUses()
	if len(tools) != 0 {
		t.Errorf("ExtractToolUses() returned %d tools, want 0", len(tools))
	}
}

func TestClaudeMessage_ExtractToolUses_StringContent(t *testing.T) {
	// String content should return nil (no tool uses)
	msg := &ClaudeMessage{
		Role:    "assistant",
		Content: json.RawMessage(`"Just a string"`),
	}

	tools := msg.ExtractToolUses()
	if tools != nil {
		t.Errorf("ExtractToolUses() = %v, want nil for string content", tools)
	}
}

func TestClaudeMessage_ExtractToolUses_EmptyContent(t *testing.T) {
	tests := []struct {
		name    string
		content json.RawMessage
	}{
		{"nil content", nil},
		{"empty content", json.RawMessage(``)},
		{"empty array", json.RawMessage(`[]`)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := &ClaudeMessage{
				Role:    "assistant",
				Content: tt.content,
			}

			tools := msg.ExtractToolUses()
			if tools != nil && len(tools) != 0 {
				t.Errorf("ExtractToolUses() = %v, want nil or empty", tools)
			}
		})
	}
}

func TestClaudeMessage_ExtractToolUses_WithInput(t *testing.T) {
	content := `[
		{"type": "tool_use", "id": "tool789", "name": "Bash", "input": {"command": "echo hello", "timeout": 5000}}
	]`
	msg := &ClaudeMessage{
		Role:    "assistant",
		Content: json.RawMessage(content),
	}

	tools := msg.ExtractToolUses()
	if len(tools) != 1 {
		t.Fatalf("ExtractToolUses() returned %d tools, want 1", len(tools))
	}

	// Verify the input is preserved as raw JSON
	if tools[0].Input == nil {
		t.Fatal("tools[0].Input is nil")
	}

	var input map[string]interface{}
	if err := json.Unmarshal(tools[0].Input, &input); err != nil {
		t.Fatalf("Failed to unmarshal input: %v", err)
	}

	if cmd, ok := input["command"].(string); !ok || cmd != "echo hello" {
		t.Errorf("input[command] = %v, want 'echo hello'", input["command"])
	}
}

// =============================================================================
// CLAUDE CONTENT ITEM TESTS
// =============================================================================

func TestClaudeContentItem_Types(t *testing.T) {
	tests := []struct {
		name     string
		itemType string
		wantText bool
		wantTool bool
	}{
		{"text type", "text", true, false},
		{"tool_use type", "tool_use", false, true},
		{"tool_result type", "tool_result", false, false},
		{"unknown type", "unknown", false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			item := ClaudeContentItem{Type: tt.itemType}

			isText := item.Type == "text"
			isTool := item.Type == "tool_use"

			if isText != tt.wantText {
				t.Errorf("isText = %v, want %v", isText, tt.wantText)
			}
			if isTool != tt.wantTool {
				t.Errorf("isTool = %v, want %v", isTool, tt.wantTool)
			}
		})
	}
}
