package agenttranscript

import (
	"bytes"
	"encoding/json"
	"strings"
	"time"
)

// Message is the portable conversational portion of a native transcript line.
// Tool calls are represented as assistant text blocks so selection and search
// can retain them without teaching consumers each harness wire format.
type Message struct {
	Role      string
	Text      string
	SessionID string
	// OccurredAt is source-native message time. It remains zero when the
	// format does not provide a valid occurrence timestamp; ingestion time is
	// deliberately owned by the consuming store.
	OccurredAt time.Time
}

// TextPart is one native message part. JoinTextParts applies the shared rule
// for prose extraction while excluding tools and lifecycle records.
type TextPart struct {
	Type string
	Text string
}

// ClaudeContentBlock is one text or tool block in a Claude message.
type ClaudeContentBlock struct {
	Type  string          `json:"type"`
	Text  string          `json:"text"`
	ID    string          `json:"id"`
	Name  string          `json:"name"`
	Input json.RawMessage `json:"input"`
}

// ClaudeUsage is the native per-message token accounting payload.
type ClaudeUsage struct {
	InputTokens              int                  `json:"input_tokens"`
	OutputTokens             int                  `json:"output_tokens"`
	CacheCreationInputTokens int                  `json:"cache_creation_input_tokens,omitempty"`
	CacheReadInputTokens     int                  `json:"cache_read_input_tokens,omitempty"`
	ServerToolUse            *ClaudeServerToolUse `json:"server_tool_use,omitempty"`
}

// ClaudeServerToolUse contains provider-reported server tool counters.
type ClaudeServerToolUse struct {
	WebSearchRequests int `json:"web_search_requests,omitempty"`
}

// ClaudeContentParts decodes Claude's string or content-block message shapes.
// A bare string is returned as one text block.
func ClaudeContentParts(raw []byte) ([]ClaudeContentBlock, bool) {
	var text string
	if json.Unmarshal(raw, &text) == nil {
		return []ClaudeContentBlock{{Type: "text", Text: text}}, true
	}
	var blocks []ClaudeContentBlock
	if json.Unmarshal(raw, &blocks) != nil {
		return nil, false
	}
	return blocks, true
}

// ClaudeTextParts returns text blocks in native order. It accepts both the
// legacy bare-string message and the current content-block message shape.
// Provider-specific ANSI cleanup stays with the consuming runner.
func ClaudeTextParts(raw []byte) []string {
	blocks, ok := ClaudeContentParts(raw)
	if !ok {
		return nil
	}
	var parts []string
	for _, block := range blocks {
		if block.Type == "text" && block.Text != "" {
			parts = append(parts, block.Text)
		}
	}
	return parts
}

// JoinTextParts joins non-empty prose parts with newlines.
func JoinTextParts(parts []TextPart) string {
	var pieces []string
	for _, part := range parts {
		if part.Type == "text" && strings.TrimSpace(part.Text) != "" {
			pieces = append(pieces, strings.TrimSpace(part.Text))
		}
	}
	return strings.Join(pieces, "\n")
}

// ExtractTextContent reads text from Codex message content. It accepts strings,
// arrays, and typed text blocks while preserving the source order.
func ExtractTextContent(raw []byte) string {
	var value any
	if json.Unmarshal(raw, &value) != nil {
		return ""
	}
	var parts []string
	var collect func(any)
	collect = func(item any) {
		switch value := item.(type) {
		case string:
			if strings.TrimSpace(value) != "" {
				parts = append(parts, value)
			}
		case []any:
			for _, child := range value {
				collect(child)
			}
		case map[string]any:
			if text, ok := value["text"].(string); ok && strings.TrimSpace(text) != "" {
				parts = append(parts, text)
			}
		}
	}
	collect(value)
	return strings.Join(parts, "\n")
}

// ParseMessages extracts human-authored and assistant text plus selectable
// tool calls/results from the common JSONL dialects. Callers keep their richer
// parser for lifecycle and accounting data.
func ParseMessages(harness string, line []byte) []Message {
	line = bytes.TrimSpace(line)
	if len(line) == 0 {
		return nil
	}
	format, ok := FormatFor(harness)
	if !ok {
		return nil
	}
	switch format.MessageDialect {
	case "claude-jsonl":
		return addRecordOccurrenceTime(parseClaudeMessages(line), ParseTimestamp(line))
	case "codex-rollout":
		return addRecordOccurrenceTime(parseCodexMessages(line), ParseTimestamp(line))
	case "opencode-message":
		return parseOpenCodeMessages(line)
	case "grok-acp":
		return parseGrokMessages(line)
	case "antigravity-stdout", "antigravity-cli-jsonl":
		return parseAntigravityMessages(line)
	case "qwen-code-jsonl":
		return ParseQwenMessages(line)
	default:
		return nil
	}
}

// addRecordOccurrenceTime applies an explicit JSONL record timestamp to the
// messages extracted from that record. Claude and Codex attach time outside
// their message payload; the timestamp describes the containing native record.
func addRecordOccurrenceTime(messages []Message, occurredAt time.Time) []Message {
	if occurredAt.IsZero() {
		return messages
	}
	for index := range messages {
		messages[index].OccurredAt = occurredAt
	}
	return messages
}

// ContainsMessageMarker reports whether a native transcript line for harness
// contains a receipt marker in a parsed conversational message. It ignores
// malformed and unrelated records by inheriting the shared dialect parser's
// semantics; callers must supply a unique marker.
func ContainsMessageMarker(harness, marker string, lines [][]byte) bool {
	if strings.TrimSpace(marker) == "" {
		return false
	}
	for _, line := range lines {
		for _, message := range ParseMessages(harness, line) {
			if strings.Contains(message.Text, marker) {
				return true
			}
		}
	}
	return false
}

// HandoffTranscriptReceipt is derived from records observed in the target's
// own native transcript. Delivery means the user prompt appeared there;
// consumption requires the assistant to emit the same unique marker.
type HandoffTranscriptReceipt struct {
	Delivered bool
	Consumed  bool
}

func InspectHandoffTranscript(harness, marker string, lines [][]byte) HandoffTranscriptReceipt {
	var receipt HandoffTranscriptReceipt
	if strings.TrimSpace(marker) == "" {
		return receipt
	}
	for _, line := range lines {
		for _, message := range ParseMessages(harness, line) {
			if !strings.Contains(message.Text, marker) {
				continue
			}
			switch message.Role {
			case "user":
				receipt.Delivered = true
			case "assistant":
				// The assistant must respond after this marker was delivered in
				// the same target transcript. A coincidental or stale assistant
				// occurrence before delivery is not a consumption receipt.
				if receipt.Delivered {
					receipt.Consumed = true
				}
			}
		}
	}
	return receipt
}

func formatToolMessage(name, kind string, raw json.RawMessage) string {
	label := strings.TrimSpace(name)
	if label == "" {
		label = "tool"
	}
	body := strings.TrimSpace(string(raw))
	if body == "" || body == "null" {
		return "[Tool " + kind + ": " + label + "]"
	}
	var pretty any
	if json.Unmarshal(raw, &pretty) == nil {
		if encoded, err := json.MarshalIndent(pretty, "", "  "); err == nil {
			body = string(encoded)
		}
	}
	return "[Tool " + kind + ": " + label + "]\n" + body
}

// FormatToolEvent renders a tool call/result as portable assistant content for
// conversation stores that preserve only text, keeping the native ID visible.
func FormatToolEvent(name, kind, id string, raw json.RawMessage) string {
	return formatToolMessage(name, kind, raw) + toolCallIDSuffix(id)
}

func parseGrokMessages(line []byte) []Message {
	update, ok := ParseGrokUpdate(line)
	if !ok {
		return nil
	}
	role := ""
	switch update.Kind {
	case "user_message_chunk":
		role = "user"
	case "agent_message_chunk":
		role = "assistant"
	case "tool_call", "tool_call_update":
		if text := GrokToolMessage(update); text != "" {
			return []Message{{Role: "assistant", Text: text, SessionID: update.SessionID}}
		}
	}
	if role == "" || strings.TrimSpace(update.Text) == "" {
		return nil
	}
	return []Message{{Role: role, Text: update.Text, SessionID: update.SessionID}}
}

func parseAntigravityMessages(line []byte) []Message {
	text := strings.TrimSpace(string(line))
	if text == "" {
		return nil
	}
	if !strings.HasPrefix(text, "{") {
		return []Message{{Role: "assistant", Text: text}}
	}
	var payload struct {
		Type           string          `json:"type"`
		Source         string          `json:"source"`
		Text           string          `json:"text"`
		Content        string          `json:"content"`
		Message        string          `json:"message"`
		Output         string          `json:"output"`
		SessionID      string          `json:"sessionId"`
		SnakeSessionID string          `json:"session_id"`
		ConversationID string          `json:"conversationId"`
		CreatedAt      json.RawMessage `json:"created_at"`
		ToolCalls      []struct {
			Name string          `json:"name"`
			Args json.RawMessage `json:"args"`
		} `json:"tool_calls"`
	}
	if json.Unmarshal([]byte(text), &payload) != nil {
		if strings.Contains(text, `"step_index"`) || strings.Contains(text, `"created_at"`) {
			return nil
		}
		return []Message{{Role: "assistant", Text: text}}
	}
	if payload.Type != "" {
		return parseAntigravityCLITranscript(payload.Type, payload.Content, payload.ToolCalls, payload.CreatedAt)
	}
	for _, candidate := range []string{payload.Text, payload.Content, payload.Message, payload.Output} {
		if strings.TrimSpace(candidate) != "" {
			sessionID := payload.SessionID
			if sessionID == "" {
				sessionID = payload.SnakeSessionID
			}
			if sessionID == "" {
				sessionID = payload.ConversationID
			}
			return []Message{{Role: "assistant", Text: candidate, SessionID: sessionID}}
		}
	}
	sessionID := payload.SessionID
	if sessionID == "" {
		sessionID = payload.SnakeSessionID
	}
	if sessionID == "" {
		sessionID = payload.ConversationID
	}
	if sessionID != "" {
		return []Message{{SessionID: sessionID}}
	}
	return nil
}

func parseAntigravityCLITranscript(recordType, content string, toolCalls []struct {
	Name string          `json:"name"`
	Args json.RawMessage `json:"args"`
}, createdAt json.RawMessage) []Message {
	role := ""
	switch recordType {
	case "USER_INPUT":
		role = "user"
	case "PLANNER_RESPONSE":
		role = "assistant"
	case "RUN_COMMAND", "VIEW_FILE", "GREP_SEARCH", "SEARCH_WEB", "READ_URL_CONTENT", "LIST_DIRECTORY", "ASK_QUESTION":
		role = "assistant"
	default:
		return nil
	}
	occurredAt := ParseTimestampValue(createdAt)
	var messages []Message
	if strings.TrimSpace(content) != "" {
		text := content
		if recordType != "USER_INPUT" && recordType != "PLANNER_RESPONSE" {
			encoded, _ := json.Marshal(content)
			text = FormatToolEvent(recordType, "result", "", encoded)
		}
		messages = append(messages, Message{Role: role, Text: text, OccurredAt: occurredAt})
	}
	if recordType == "PLANNER_RESPONSE" {
		for _, call := range toolCalls {
			if strings.TrimSpace(call.Name) == "" {
				continue
			}
			messages = append(messages, Message{
				Role: "assistant", Text: FormatToolEvent(call.Name, "call", "", call.Args), OccurredAt: occurredAt,
			})
		}
	}
	return messages
}

func parseOpenCodeMessages(line []byte) []Message {
	var record struct {
		Info struct {
			Role string `json:"role"`
			Time struct {
				Created json.RawMessage `json:"created"`
			} `json:"time"`
		} `json:"info"`
		Parts []json.RawMessage `json:"parts"`
	}
	if json.Unmarshal(line, &record) != nil || (record.Info.Role != "user" && record.Info.Role != "assistant") {
		return nil
	}
	// OpenCode's archive time.created is an explicit Unix-millisecond
	// timestamp describing the enclosing message record and therefore applies
	// to each searchable part emitted from that record.
	occurredAt := ParseTimestampValue(record.Info.Time.Created)
	var out []Message
	for _, raw := range record.Parts {
		var part struct {
			Type   string          `json:"type"`
			Text   string          `json:"text"`
			Tool   string          `json:"tool"`
			CallID string          `json:"callID"`
			State  json.RawMessage `json:"state"`
			Input  json.RawMessage `json:"input"`
		}
		if json.Unmarshal(raw, &part) != nil {
			continue
		}
		if part.Type == "text" && strings.TrimSpace(part.Text) != "" {
			out = append(out, Message{Role: record.Info.Role, Text: part.Text, OccurredAt: occurredAt})
		} else if part.Type == "tool" {
			payload := part.Input
			if len(payload) == 0 {
				payload = part.State
			}
			out = append(out, Message{Role: "assistant", Text: FormatToolEvent(part.Tool, "call", part.CallID, payload), OccurredAt: occurredAt})
		}
	}
	return out
}

func parseClaudeMessages(line []byte) []Message {
	var entry struct {
		Type    string `json:"type"`
		Message struct {
			Content json.RawMessage `json:"content"`
		} `json:"message"`
	}
	if json.Unmarshal(line, &entry) != nil || (entry.Type != "user" && entry.Type != "assistant") {
		return nil
	}
	blocks, ok := ClaudeContentParts(entry.Message.Content)
	if ok {
		for _, block := range blocks {
			if entry.Type == "user" && block.Type == "tool_result" {
				return nil
			}
		}
		out := make([]Message, 0, len(blocks))
		for _, block := range blocks {
			if block.Type == "text" && strings.TrimSpace(block.Text) != "" {
				out = append(out, Message{Role: entry.Type, Text: block.Text})
			} else if entry.Type == "assistant" && block.Type == "tool_use" &&
				strings.TrimSpace(block.ID) != "" && strings.TrimSpace(block.Name) != "" {
				out = append(out, Message{Role: "assistant", Text: FormatToolEvent(block.Name, "call", block.ID, block.Input)})
			}
		}
		return out
	}
	var text string
	if json.Unmarshal(entry.Message.Content, &text) == nil && strings.TrimSpace(text) != "" {
		return []Message{{Role: entry.Type, Text: text}}
	}
	return nil
}

func parseCodexMessages(line []byte) []Message {
	var record map[string]json.RawMessage
	if json.Unmarshal(line, &record) != nil {
		return nil
	}
	var payload struct {
		Type    string `json:"type"`
		Text    string `json:"text"`
		Message string `json:"message"`
		Role    string `json:"role"`
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
		Name      string          `json:"name"`
		CallID    string          `json:"call_id"`
		Input     json.RawMessage `json:"input"`
		Arguments json.RawMessage `json:"arguments"`
		Output    json.RawMessage `json:"output"`
	}
	if json.Unmarshal(record["payload"], &payload) == nil {
		switch payload.Type {
		case "function_call", "custom_tool_call":
			input := payload.Input
			if len(input) == 0 {
				input = payload.Arguments
			}
			return []Message{{Role: "assistant", Text: FormatToolEvent(payload.Name, "call", payload.CallID, input)}}
		case "function_call_output", "custom_tool_call_output":
			return []Message{{Role: "assistant", Text: FormatToolEvent(payload.Name, "result", payload.CallID, payload.Output)}}
		case "user_message":
			text := payload.Text
			if text == "" {
				text = payload.Message
			}
			if strings.TrimSpace(text) != "" {
				return []Message{{Role: "user", Text: text}}
			}
		case "agent_message", "assistant_message":
			if strings.TrimSpace(payload.Text) != "" {
				return []Message{{Role: "assistant", Text: payload.Text}}
			}
		}
		if payload.Role == "user" || payload.Role == "assistant" {
			var out []Message
			for _, content := range payload.Content {
				// Codex uses input_text for user content and output_text for
				// assistant output. Keep the neutral text form for older logs,
				// but don't surface the opposite role's typed content.
				if payload.Role == "user" && content.Type != "input_text" && content.Type != "text" {
					continue
				}
				if payload.Role == "assistant" && content.Type != "output_text" && content.Type != "text" {
					continue
				}
				if strings.TrimSpace(content.Text) != "" {
					out = append(out, Message{Role: payload.Role, Text: content.Text})
				}
			}
			return out
		}
	}
	var item struct {
		Type    string `json:"type"`
		Role    string `json:"role"`
		Text    string `json:"text"`
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
	}
	if json.Unmarshal(record["item"], &item) == nil {
		role := ""
		if item.Type == "user_message" || item.Role == "user" {
			role = "user"
		}
		if item.Type == "agent_message" || item.Type == "assistant_message" || item.Role == "assistant" {
			role = "assistant"
		}
		if role != "" {
			if strings.TrimSpace(item.Text) != "" {
				return []Message{{Role: role, Text: item.Text}}
			}
			var out []Message
			for _, content := range item.Content {
				if content.Type == "text" && strings.TrimSpace(content.Text) != "" {
					out = append(out, Message{Role: role, Text: content.Text})
				}
			}
			return out
		}
	}
	return nil
}

func toolCallIDSuffix(id string) string {
	if strings.TrimSpace(id) == "" {
		return ""
	}
	return "\nCall ID: " + id
}
