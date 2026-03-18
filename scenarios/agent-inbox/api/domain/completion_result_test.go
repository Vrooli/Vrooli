package domain

import (
	"testing"
)

// =============================================================================
// CompletionResult Tests
// =============================================================================

func TestCompletionResult_RequiresToolExecution(t *testing.T) {
	tests := []struct {
		name     string
		result   CompletionResult
		expected bool
	}{
		{
			name:     "no tool calls, stop reason",
			result:   CompletionResult{FinishReason: "stop", ToolCalls: nil},
			expected: false,
		},
		{
			name:     "tool_calls reason but empty list",
			result:   CompletionResult{FinishReason: "tool_calls", ToolCalls: []ToolCall{}},
			expected: false,
		},
		{
			name: "tool_calls reason with calls",
			result: CompletionResult{
				FinishReason: "tool_calls",
				ToolCalls:    []ToolCall{{ID: "tc-1", Type: "function"}},
			},
			expected: true,
		},
		{
			name: "wrong reason with tool calls",
			result: CompletionResult{
				FinishReason: "stop",
				ToolCalls:    []ToolCall{{ID: "tc-1"}},
			},
			expected: false,
		},
		{
			name:     "empty result",
			result:   CompletionResult{},
			expected: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.result.RequiresToolExecution()
			if got != tc.expected {
				t.Errorf("RequiresToolExecution() = %v, want %v", got, tc.expected)
			}
		})
	}
}

func TestCompletionResult_HasContent(t *testing.T) {
	tests := []struct {
		name     string
		result   CompletionResult
		expected bool
	}{
		{"empty content", CompletionResult{Content: ""}, false},
		{"has content", CompletionResult{Content: "Hello"}, true},
		{"whitespace only", CompletionResult{Content: "   "}, true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.result.HasContent(); got != tc.expected {
				t.Errorf("HasContent() = %v, want %v", got, tc.expected)
			}
		})
	}
}

func TestCompletionResult_HasImages(t *testing.T) {
	tests := []struct {
		name     string
		result   CompletionResult
		expected bool
	}{
		{"no images", CompletionResult{Images: nil}, false},
		{"empty images", CompletionResult{Images: []string{}}, false},
		{"has images", CompletionResult{Images: []string{"data:image/png;base64,abc"}}, true},
		{"multiple images", CompletionResult{Images: []string{"img1", "img2"}}, true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.result.HasImages(); got != tc.expected {
				t.Errorf("HasImages() = %v, want %v", got, tc.expected)
			}
		})
	}
}

func TestCompletionResult_HasResponse(t *testing.T) {
	tests := []struct {
		name     string
		result   CompletionResult
		expected bool
	}{
		{"empty", CompletionResult{}, false},
		{"content only", CompletionResult{Content: "Hi"}, true},
		{"images only", CompletionResult{Images: []string{"img"}}, true},
		{"both", CompletionResult{Content: "Hi", Images: []string{"img"}}, true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.result.HasResponse(); got != tc.expected {
				t.Errorf("HasResponse() = %v, want %v", got, tc.expected)
			}
		})
	}
}

func TestCompletionResult_PreviewText(t *testing.T) {
	tests := []struct {
		name     string
		result   CompletionResult
		expected string
	}{
		{
			name:     "empty result",
			result:   CompletionResult{},
			expected: "",
		},
		{
			name:     "short content",
			result:   CompletionResult{Content: "Hello, world!"},
			expected: "Hello, world!",
		},
		{
			name:     "tool calls without content",
			result:   CompletionResult{FinishReason: "tool_calls", ToolCalls: []ToolCall{{ID: "1"}}},
			expected: "Using tools...",
		},
		{
			name:     "tool calls with content",
			result:   CompletionResult{FinishReason: "tool_calls", ToolCalls: []ToolCall{{ID: "1"}}, Content: "Let me help"},
			expected: "Let me help",
		},
		{
			name:     "images without content",
			result:   CompletionResult{Images: []string{"img1"}},
			expected: "Generated image",
		},
		{
			name:     "images with content",
			result:   CompletionResult{Images: []string{"img1"}, Content: "Here's your image"},
			expected: "Here's your image",
		},
		{
			name:     "long content truncated",
			result:   CompletionResult{Content: string(make([]byte, 150))}, // 150 null bytes
			expected: string(make([]byte, PreviewMaxLength)) + "...",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.result.PreviewText()
			if got != tc.expected {
				t.Errorf("PreviewText() = %q, want %q", got, tc.expected)
			}
		})
	}
}
