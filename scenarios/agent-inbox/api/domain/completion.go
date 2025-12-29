// Package domain defines core domain types for the Agent Inbox scenario.
package domain

// CompletionResult represents the outcome of an AI completion request.
// This type makes explicit the possible completion outcomes:
// 1. Regular content response (text reply from the model)
// 2. Tool call response (model wants to invoke tools)
// 3. Image generation response (model generated images)
type CompletionResult struct {
	Content      string     `json:"content"`
	TokenCount   int        `json:"token_count"`
	FinishReason string     `json:"finish_reason"`
	ToolCalls    []ToolCall `json:"tool_calls,omitempty"`
	ResponseID   string     `json:"response_id,omitempty"`
	Usage        *Usage     `json:"usage,omitempty"`
	Images       []string   `json:"images,omitempty"` // AI-generated images as base64 data URLs
}

// Usage contains token usage information from a completion.
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
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
	Usage          *Usage
	Images         []string // AI-generated images accumulated during streaming
}

// NewStreamingAccumulator creates a new accumulator for streaming responses.
func NewStreamingAccumulator() *StreamingAccumulator {
	return &StreamingAccumulator{
		ToolCalls: make([]ToolCall, 0),
	}
}

// AppendContent adds content from a streaming chunk.
// Note: TokenCount is set from Usage data in the final chunk, not incremented per chunk.
// The per-chunk increment was incorrect as chunks don't correspond to tokens.
func (a *StreamingAccumulator) AppendContent(content string) {
	a.ContentBuilder += content
	// TokenCount is populated from Usage data via SetUsage, not incremented here
}

// AppendImage adds a generated image to the accumulator.
func (a *StreamingAccumulator) AppendImage(dataURL string) {
	a.Images = append(a.Images, dataURL)
}

// AppendToolCallDelta merges a tool call delta into the accumulated tool calls.
// Tool calls in streaming responses arrive as partial deltas that must be assembled.
//
// OpenRouter/OpenAI streaming format:
// - First delta for a tool call includes: index, id, type, function.name, partial arguments
// - Subsequent deltas include: index, partial arguments (id/name may be empty)
// - The Index field correlates deltas across chunks, not the ID
func (a *StreamingAccumulator) AppendToolCallDelta(delta ToolCall) {
	// Use Index for correlation - OpenRouter streams tool calls with index field
	// where ID is only present in the first delta
	for i := range a.ToolCalls {
		if a.ToolCalls[i].Index == delta.Index {
			// Append arguments fragment to existing tool call
			a.ToolCalls[i].Function.Arguments += delta.Function.Arguments
			// Also capture name if it arrives (first delta has it)
			if delta.Function.Name != "" && a.ToolCalls[i].Function.Name == "" {
				a.ToolCalls[i].Function.Name = delta.Function.Name
			}
			// Capture ID if it arrives (first delta has it)
			if delta.ID != "" && a.ToolCalls[i].ID == "" {
				a.ToolCalls[i].ID = delta.ID
			}
			return
		}
	}
	// New tool call - add it to the accumulator
	// The first delta for a tool call always has the index
	a.ToolCalls = append(a.ToolCalls, delta)
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

// SetUsage sets usage data when received (typically in final chunk).
func (a *StreamingAccumulator) SetUsage(promptTokens, completionTokens, totalTokens int) {
	if promptTokens > 0 || completionTokens > 0 || totalTokens > 0 {
		a.Usage = &Usage{
			PromptTokens:     promptTokens,
			CompletionTokens: completionTokens,
			TotalTokens:      totalTokens,
		}
	}
}

// ToResult converts the accumulated data into a CompletionResult.
// TokenCount is derived from Usage.CompletionTokens when available (accurate),
// otherwise falls back to the accumulated TokenCount (may be 0 for streaming).
func (a *StreamingAccumulator) ToResult() *CompletionResult {
	tokenCount := a.TokenCount
	if a.Usage != nil && a.Usage.CompletionTokens > 0 {
		tokenCount = a.Usage.CompletionTokens
	}
	return &CompletionResult{
		Content:      a.ContentBuilder,
		TokenCount:   tokenCount,
		FinishReason: a.FinishReason,
		ToolCalls:    a.ToolCalls,
		ResponseID:   a.ResponseID,
		Usage:        a.Usage,
		Images:       a.Images,
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
