// Package domain defines core domain types for the Agent Inbox scenario.
package domain

// CompletionResult represents the outcome of an AI completion request.
// This type makes explicit the two possible completion outcomes:
// 1. Regular content response (text reply from the model)
// 2. Tool call response (model wants to invoke tools)
type CompletionResult struct {
	Content      string     `json:"content"`
	TokenCount   int        `json:"token_count"`
	FinishReason string     `json:"finish_reason"`
	ToolCalls    []ToolCall `json:"tool_calls,omitempty"`
	ResponseID   string     `json:"response_id,omitempty"`
}

// RequiresToolExecution returns true if the model requested tool calls.
// This is the primary decision boundary for completion handling:
// - true: we must execute tools and potentially continue the conversation
// - false: the response is complete and can be shown to the user
func (r *CompletionResult) RequiresToolExecution() bool {
	return r.FinishReason == "tool_calls" && len(r.ToolCalls) > 0
}

// HasContent returns true if the response contains text content.
func (r *CompletionResult) HasContent() bool {
	return r.Content != ""
}

// PreviewText returns a truncated version of the content for chat list display.
// Returns a fallback message if the response has tool calls but no content.
// Uses domain.PreviewMaxLength for consistent truncation across the codebase.
func (r *CompletionResult) PreviewText() string {
	if r.RequiresToolExecution() && !r.HasContent() {
		return "Using tools..."
	}
	if r.Content == "" {
		return ""
	}
	return TruncatePreview(r.Content)
}

// StreamingAccumulator collects chunks from a streaming response.
// Streaming responses arrive in pieces that must be assembled into a complete result.
type StreamingAccumulator struct {
	ContentBuilder string
	TokenCount     int
	ToolCalls      []ToolCall
	FinishReason   string
	ResponseID     string
}

// NewStreamingAccumulator creates a new accumulator for streaming responses.
func NewStreamingAccumulator() *StreamingAccumulator {
	return &StreamingAccumulator{
		ToolCalls: make([]ToolCall, 0),
	}
}

// AppendContent adds content from a streaming chunk.
func (a *StreamingAccumulator) AppendContent(content string) {
	a.ContentBuilder += content
	a.TokenCount++
}

// AppendToolCallDelta merges a tool call delta into the accumulated tool calls.
// Tool calls in streaming responses arrive as partial deltas that must be assembled.
func (a *StreamingAccumulator) AppendToolCallDelta(delta ToolCall) {
	// Find existing tool call to extend, or add new one
	for i := range a.ToolCalls {
		if a.ToolCalls[i].ID == delta.ID {
			a.ToolCalls[i].Function.Arguments += delta.Function.Arguments
			return
		}
	}
	// New tool call
	if delta.ID != "" {
		a.ToolCalls = append(a.ToolCalls, delta)
	}
}

// SetResponseID sets the response ID if not already set.
func (a *StreamingAccumulator) SetResponseID(id string) {
	if a.ResponseID == "" && id != "" {
		a.ResponseID = id
	}
}

// SetFinishReason records the finish reason when received.
func (a *StreamingAccumulator) SetFinishReason(reason string) {
	if reason != "" {
		a.FinishReason = reason
	}
}

// ToResult converts the accumulated data into a CompletionResult.
func (a *StreamingAccumulator) ToResult() *CompletionResult {
	return &CompletionResult{
		Content:      a.ContentBuilder,
		TokenCount:   a.TokenCount,
		FinishReason: a.FinishReason,
		ToolCalls:    a.ToolCalls,
		ResponseID:   a.ResponseID,
	}
}

// ToolExecutionResult captures the outcome of executing a single tool.
type ToolExecutionResult struct {
	ToolCallID string      `json:"tool_call_id"`
	ToolName   string      `json:"tool_name"`
	Status     string      `json:"status"`
	Result     interface{} `json:"result,omitempty"`
	Error      string      `json:"error,omitempty"`
}

// Succeeded returns true if the tool executed successfully.
func (r *ToolExecutionResult) Succeeded() bool {
	return r.Status == StatusCompleted
}
