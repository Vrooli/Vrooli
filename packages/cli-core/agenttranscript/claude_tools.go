package agenttranscript

import "encoding/json"

// ClaudeToolBlock is the native tool-facing subset of a Claude content block.
type ClaudeToolBlock struct {
	Type      string          `json:"type"`
	Text      string          `json:"text,omitempty"`
	ID        string          `json:"id,omitempty"`
	Name      string          `json:"name,omitempty"`
	Input     json.RawMessage `json:"input,omitempty"`
	ToolUseID string          `json:"tool_use_id,omitempty"`
	Content   string          `json:"content,omitempty"`
	IsError   bool            `json:"is_error,omitempty"`
}

// ClaudeToolUses selects native tool_use blocks from one message's content.
func ClaudeToolUses(raw []byte) []ClaudeToolBlock {
	return claudeToolBlocksOfType(raw, "tool_use")
}

// ClaudeToolResults selects native tool_result blocks from one message's content.
func ClaudeToolResults(raw []byte) []ClaudeToolBlock {
	return claudeToolBlocksOfType(raw, "tool_result")
}

func claudeToolBlocksOfType(raw []byte, kind string) []ClaudeToolBlock {
	var blocks []ClaudeToolBlock
	if json.Unmarshal(raw, &blocks) != nil {
		return nil
	}
	var selected []ClaudeToolBlock
	for _, block := range blocks {
		if block.Type == kind {
			selected = append(selected, block)
		}
	}
	return selected
}
