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

// =============================================================================
// StreamingAccumulator Tests
// =============================================================================

func TestNewStreamingAccumulator(t *testing.T) {
	acc := NewStreamingAccumulator()

	if acc == nil {
		t.Fatal("NewStreamingAccumulator() returned nil")
	}
	if acc.ToolCalls == nil {
		t.Error("ToolCalls should be initialized to empty slice")
	}
	if len(acc.ToolCalls) != 0 {
		t.Errorf("ToolCalls should be empty, got %d", len(acc.ToolCalls))
	}
	if acc.ContentBuilder != "" {
		t.Errorf("ContentBuilder should be empty, got %q", acc.ContentBuilder)
	}
}

func TestStreamingAccumulator_AppendContent(t *testing.T) {
	acc := NewStreamingAccumulator()

	acc.AppendContent("Hello")
	acc.AppendContent(", ")
	acc.AppendContent("world!")

	if acc.ContentBuilder != "Hello, world!" {
		t.Errorf("ContentBuilder = %q, want %q", acc.ContentBuilder, "Hello, world!")
	}
}

func TestStreamingAccumulator_AppendImage(t *testing.T) {
	acc := NewStreamingAccumulator()

	acc.AppendImage("data:image/png;base64,abc")
	acc.AppendImage("data:image/jpeg;base64,def")

	if len(acc.Images) != 2 {
		t.Fatalf("expected 2 images, got %d", len(acc.Images))
	}
	if acc.Images[0] != "data:image/png;base64,abc" {
		t.Errorf("Images[0] = %q, want %q", acc.Images[0], "data:image/png;base64,abc")
	}
}

func TestStreamingAccumulator_AppendToolCallDelta(t *testing.T) {
	acc := NewStreamingAccumulator()

	// First delta - new tool call with index, id, and partial args
	acc.AppendToolCallDelta(ToolCall{
		Index: 0,
		ID:    "tc-1",
		Type:  "function",
		Function: struct {
			Name      string `json:"name"`
			Arguments string `json:"arguments"`
		}{
			Name:      "get_weather",
			Arguments: `{"location":`,
		},
	})

	if len(acc.ToolCalls) != 1 {
		t.Fatalf("expected 1 tool call, got %d", len(acc.ToolCalls))
	}
	if acc.ToolCalls[0].ID != "tc-1" {
		t.Errorf("ID = %q, want %q", acc.ToolCalls[0].ID, "tc-1")
	}
	if acc.ToolCalls[0].Function.Name != "get_weather" {
		t.Errorf("Name = %q, want %q", acc.ToolCalls[0].Function.Name, "get_weather")
	}

	// Second delta - continuation (same index, more args)
	acc.AppendToolCallDelta(ToolCall{
		Index: 0,
		Function: struct {
			Name      string `json:"name"`
			Arguments string `json:"arguments"`
		}{
			Arguments: `"NYC"}`,
		},
	})

	if len(acc.ToolCalls) != 1 {
		t.Fatalf("expected 1 tool call after merge, got %d", len(acc.ToolCalls))
	}
	expectedArgs := `{"location":"NYC"}`
	if acc.ToolCalls[0].Function.Arguments != expectedArgs {
		t.Errorf("Arguments = %q, want %q", acc.ToolCalls[0].Function.Arguments, expectedArgs)
	}
}

func TestStreamingAccumulator_AppendToolCallDelta_MultipleTools(t *testing.T) {
	acc := NewStreamingAccumulator()

	// First tool call
	acc.AppendToolCallDelta(ToolCall{Index: 0, ID: "tc-1", Function: struct {
		Name      string `json:"name"`
		Arguments string `json:"arguments"`
	}{Name: "tool_a", Arguments: `{"a":1}`}})

	// Second tool call
	acc.AppendToolCallDelta(ToolCall{Index: 1, ID: "tc-2", Function: struct {
		Name      string `json:"name"`
		Arguments string `json:"arguments"`
	}{Name: "tool_b", Arguments: `{"b":2}`}})

	// Update first tool call
	acc.AppendToolCallDelta(ToolCall{Index: 0, Function: struct {
		Name      string `json:"name"`
		Arguments string `json:"arguments"`
	}{Arguments: `,extra`}})

	if len(acc.ToolCalls) != 2 {
		t.Fatalf("expected 2 tool calls, got %d", len(acc.ToolCalls))
	}
	if acc.ToolCalls[0].Function.Arguments != `{"a":1},extra` {
		t.Errorf("tool 0 args = %q", acc.ToolCalls[0].Function.Arguments)
	}
	if acc.ToolCalls[1].Function.Name != "tool_b" {
		t.Errorf("tool 1 name = %q", acc.ToolCalls[1].Function.Name)
	}
}

func TestStreamingAccumulator_SetResponseID(t *testing.T) {
	acc := NewStreamingAccumulator()

	// First set
	acc.SetResponseID("gen-123")
	if acc.ResponseID != "gen-123" {
		t.Errorf("ResponseID = %q, want %q", acc.ResponseID, "gen-123")
	}

	// Should not overwrite
	acc.SetResponseID("gen-456")
	if acc.ResponseID != "gen-123" {
		t.Errorf("ResponseID should not be overwritten, got %q", acc.ResponseID)
	}

	// Empty string should not set
	acc2 := NewStreamingAccumulator()
	acc2.SetResponseID("")
	if acc2.ResponseID != "" {
		t.Errorf("Empty string should not set ResponseID")
	}
}

func TestStreamingAccumulator_SetFinishReason(t *testing.T) {
	acc := NewStreamingAccumulator()

	acc.SetFinishReason("stop")
	if acc.FinishReason != "stop" {
		t.Errorf("FinishReason = %q, want %q", acc.FinishReason, "stop")
	}

	// Can be overwritten (unlike ResponseID)
	acc.SetFinishReason("tool_calls")
	if acc.FinishReason != "tool_calls" {
		t.Errorf("FinishReason = %q, want %q", acc.FinishReason, "tool_calls")
	}

	// Empty string doesn't set
	acc2 := NewStreamingAccumulator()
	acc2.SetFinishReason("")
	if acc2.FinishReason != "" {
		t.Errorf("Empty string should not set FinishReason")
	}
}

func TestStreamingAccumulator_SetUsage(t *testing.T) {
	acc := NewStreamingAccumulator()

	// All zeros - should not set
	acc.SetUsage(0, 0, 0)
	if acc.Usage != nil {
		t.Error("Usage should be nil when all zeros")
	}

	// Valid usage
	acc.SetUsage(100, 50, 150)
	if acc.Usage == nil {
		t.Fatal("Usage should be set")
	}
	if acc.Usage.PromptTokens != 100 {
		t.Errorf("PromptTokens = %d, want 100", acc.Usage.PromptTokens)
	}
	if acc.Usage.CompletionTokens != 50 {
		t.Errorf("CompletionTokens = %d, want 50", acc.Usage.CompletionTokens)
	}
	if acc.Usage.TotalTokens != 150 {
		t.Errorf("TotalTokens = %d, want 150", acc.Usage.TotalTokens)
	}
}

func TestStreamingAccumulator_ToResult(t *testing.T) {
	acc := NewStreamingAccumulator()
	acc.AppendContent("Test response")
	acc.SetResponseID("gen-abc")
	acc.SetFinishReason("stop")
	acc.SetUsage(100, 25, 125)
	acc.AppendImage("img1")

	result := acc.ToResult()

	if result.Content != "Test response" {
		t.Errorf("Content = %q", result.Content)
	}
	if result.ResponseID != "gen-abc" {
		t.Errorf("ResponseID = %q", result.ResponseID)
	}
	if result.FinishReason != "stop" {
		t.Errorf("FinishReason = %q", result.FinishReason)
	}
	if result.TokenCount != 25 {
		t.Errorf("TokenCount = %d, want 25 (from Usage.CompletionTokens)", result.TokenCount)
	}
	if len(result.Images) != 1 {
		t.Errorf("Images count = %d, want 1", len(result.Images))
	}
}

func TestStreamingAccumulator_ToResult_NoUsage(t *testing.T) {
	acc := NewStreamingAccumulator()
	acc.AppendContent("Test")
	acc.TokenCount = 10 // Manual token count

	result := acc.ToResult()

	if result.TokenCount != 10 {
		t.Errorf("TokenCount = %d, want 10 (from manual TokenCount)", result.TokenCount)
	}
}

// =============================================================================
// ToolExecutionResult Tests
// =============================================================================

func TestToolExecutionResult_Succeeded(t *testing.T) {
	tests := []struct {
		name     string
		result   ToolExecutionResult
		expected bool
	}{
		{"completed status", ToolExecutionResult{Status: StatusCompleted}, true},
		{"failed status", ToolExecutionResult{Status: StatusFailed}, false},
		{"running status", ToolExecutionResult{Status: StatusRunning}, false},
		{"pending status", ToolExecutionResult{Status: StatusPending}, false},
		{"empty status", ToolExecutionResult{Status: ""}, false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.result.Succeeded(); got != tc.expected {
				t.Errorf("Succeeded() = %v, want %v", got, tc.expected)
			}
		})
	}
}
