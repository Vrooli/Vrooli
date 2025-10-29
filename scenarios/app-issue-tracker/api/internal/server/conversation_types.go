package server

// Type-safe structures for agent conversation event data
// These provide stronger typing than map[string]interface{} for known event types

// TokenCountData represents token usage statistics from an agent conversation
type TokenCountData struct {
	InputTokens  int `json:"input_tokens,omitempty"`
	OutputTokens int `json:"output_tokens,omitempty"`
	TotalTokens  int `json:"total_tokens,omitempty"`
}

// ToolRequestData represents a tool invocation request from the agent
type ToolRequestData struct {
	ToolName   string                 `json:"tool_name,omitempty"`
	ToolID     string                 `json:"tool_id,omitempty"`
	Parameters map[string]interface{} `json:"parameters,omitempty"`
	Summary    string                 `json:"summary,omitempty"`
}

// ToolResultData represents the result of a tool execution
type ToolResultData struct {
	ToolName   string `json:"tool_name,omitempty"`
	ToolID     string `json:"tool_id,omitempty"`
	Success    bool   `json:"success"`
	Output     string `json:"output,omitempty"`
	Error      string `json:"error,omitempty"`
	Summary    string `json:"summary,omitempty"`
	ExitCode   *int   `json:"exit_code,omitempty"`
	DurationMs *int64 `json:"duration_ms,omitempty"`
}

// TaskStartedData represents the beginning of an agent task
type TaskStartedData struct {
	TaskID      string `json:"task_id,omitempty"`
	TaskType    string `json:"task_type,omitempty"`
	Description string `json:"description,omitempty"`
}

// AgentReasoningData represents internal agent reasoning/thinking
type AgentReasoningData struct {
	Reasoning string `json:"reasoning,omitempty"`
	Thought   string `json:"thought,omitempty"`
	Plan      string `json:"plan,omitempty"`
}

// TranscriptMetadata represents metadata from the transcript file header
type TranscriptMetadata struct {
	Sandbox     string `json:"sandbox,omitempty"`
	Provider    string `json:"provider,omitempty"`
	SessionID   string `json:"session_id,omitempty"`
	Generated   bool   `json:"generated,omitempty"`
	GeneratedAt string `json:"generated_at,omitempty"`
}

// ParsedConversationData attempts to parse conversation event data into typed structures
// Returns the typed data and a boolean indicating if parsing was successful
func ParsedConversationData(eventType string, raw map[string]interface{}) (interface{}, bool) {
	switch eventType {
	case "token_count":
		return parseTokenCount(raw), true
	case "tool_request":
		return parseToolRequest(raw), true
	case "tool_result", "tool_output":
		return parseToolResult(raw), true
	case "tool_error":
		return parseToolError(raw), true
	case "task_started":
		return parseTaskStarted(raw), true
	case "agent_reasoning", "reasoning":
		return parseReasoning(raw), true
	default:
		return nil, false
	}
}

func parseTokenCount(raw map[string]interface{}) *TokenCountData {
	data := &TokenCountData{}
	if v, ok := raw["input_tokens"].(float64); ok {
		data.InputTokens = int(v)
	}
	if v, ok := raw["output_tokens"].(float64); ok {
		data.OutputTokens = int(v)
	}
	if v, ok := raw["total_tokens"].(float64); ok {
		data.TotalTokens = int(v)
	}
	return data
}

func parseToolRequest(raw map[string]interface{}) *ToolRequestData {
	data := &ToolRequestData{}
	if v, ok := raw["tool_name"].(string); ok {
		data.ToolName = v
	}
	if v, ok := raw["tool_id"].(string); ok {
		data.ToolID = v
	}
	if v, ok := raw["summary"].(string); ok {
		data.Summary = v
	}
	if v, ok := raw["parameters"].(map[string]interface{}); ok {
		data.Parameters = v
	}
	return data
}

func parseToolResult(raw map[string]interface{}) *ToolResultData {
	data := &ToolResultData{}
	if v, ok := raw["tool_name"].(string); ok {
		data.ToolName = v
	}
	if v, ok := raw["tool_id"].(string); ok {
		data.ToolID = v
	}
	if v, ok := raw["success"].(bool); ok {
		data.Success = v
	}
	if v, ok := raw["output"].(string); ok {
		data.Output = v
	}
	if v, ok := raw["summary"].(string); ok {
		data.Summary = v
	}
	if v, ok := raw["exit_code"].(float64); ok {
		code := int(v)
		data.ExitCode = &code
	}
	if v, ok := raw["duration_ms"].(float64); ok {
		duration := int64(v)
		data.DurationMs = &duration
	}
	return data
}

func parseToolError(raw map[string]interface{}) *ToolResultData {
	data := parseToolResult(raw)
	data.Success = false
	if v, ok := raw["error"].(string); ok {
		data.Error = v
	}
	return data
}

func parseTaskStarted(raw map[string]interface{}) *TaskStartedData {
	data := &TaskStartedData{}
	if v, ok := raw["task_id"].(string); ok {
		data.TaskID = v
	}
	if v, ok := raw["task_type"].(string); ok {
		data.TaskType = v
	}
	if v, ok := raw["description"].(string); ok {
		data.Description = v
	}
	return data
}

func parseReasoning(raw map[string]interface{}) *AgentReasoningData {
	data := &AgentReasoningData{}
	if v, ok := raw["reasoning"].(string); ok {
		data.Reasoning = v
	}
	if v, ok := raw["thought"].(string); ok {
		data.Thought = v
	}
	if v, ok := raw["plan"].(string); ok {
		data.Plan = v
	}
	return data
}

// ParseTranscriptMetadata extracts metadata from a transcript header
func ParseTranscriptMetadata(raw map[string]interface{}) *TranscriptMetadata {
	meta := &TranscriptMetadata{}
	if v, ok := raw["sandbox"].(string); ok {
		meta.Sandbox = v
	}
	if v, ok := raw["provider"].(string); ok {
		meta.Provider = v
	}
	if v, ok := raw["session_id"].(string); ok {
		meta.SessionID = v
	}
	if v, ok := raw["generated"].(bool); ok {
		meta.Generated = v
	}
	if v, ok := raw["generated_at"].(string); ok {
		meta.GeneratedAt = v
	}
	return meta
}
