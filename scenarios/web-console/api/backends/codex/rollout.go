package codex

import "encoding/json"

// RolloutLine represents a single line from a Codex rollout JSONL file.
type RolloutLine struct {
	Timestamp string          `json:"timestamp"`
	Type      string          `json:"type"`
	Payload   json.RawMessage `json:"payload"`
}

// ResponsePayload is the parsed payload for response_item lines.
type ResponsePayload struct {
	Role    string        `json:"role"`
	Content []ContentItem `json:"content"`
}

// ContentItem represents a single content element within a response payload.
type ContentItem struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// ExtractUserText parses a rollout JSONL line and returns the user's
// text prompt, or "" if this line isn't a user text message.
func ExtractUserText(line []byte) string { return ExtractUserInput(line).Text }

// ExtractUserInput retains the origin of a captured coding user message.
func ExtractUserInput(line []byte) UserInput {
	var rl RolloutLine
	if err := json.Unmarshal(line, &rl); err != nil {
		return UserInput{}
	}
	if rl.Type != "response_item" {
		return UserInput{}
	}
	var payload ResponsePayload
	if err := json.Unmarshal(rl.Payload, &payload); err != nil {
		return UserInput{}
	}
	if payload.Role != "user" {
		return UserInput{}
	}
	var result string
	for _, item := range payload.Content {
		if item.Type == "input_text" || item.Type == "text" {
			result += item.Text
		}
	}
	return ParseUserInput(result)
}

// ExtractAssistantText parses a rollout JSONL line and returns the assistant's
// text response, or "" if this line isn't an assistant text message.
func ExtractAssistantText(line []byte) string {
	var rl RolloutLine
	if err := json.Unmarshal(line, &rl); err != nil {
		return ""
	}
	if rl.Type != "response_item" {
		return ""
	}
	var payload ResponsePayload
	if err := json.Unmarshal(rl.Payload, &payload); err != nil {
		return ""
	}
	if payload.Role != "assistant" {
		return ""
	}
	var result string
	for _, item := range payload.Content {
		if item.Type == "output_text" {
			result += item.Text
		}
	}
	return result
}
