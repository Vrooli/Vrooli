package agenttranscript

import (
	"bytes"
	"encoding/json"
	"strings"
)

// GrokUpdate is the portable part of one ACP session/update notification.
// Content and RawInput retain their native JSON representation for consumers
// that need richer tool semantics.
type GrokUpdate struct {
	Method     string
	SessionID  string
	Kind       string
	Text       string
	StopReason string
	EventID    string
	ToolCallID string
	Title      string
	Status     string
	Content    json.RawMessage
	RawInput   json.RawMessage
	RawOutput  string
}

// GrokStreamEvent is one line from Grok's headless streaming-json output.
// The event envelope is shared transcript data; consumers retain decisions
// about buffering, terminal state, and which event kinds to expose.
type GrokStreamEvent struct {
	Type       string `json:"type"`
	Data       string `json:"data,omitempty"`
	StopReason string `json:"stopReason,omitempty"`
	SessionID  string `json:"sessionId,omitempty"`
	RequestID  string `json:"requestId,omitempty"`
	Message    string `json:"message,omitempty"`
}

// ParseGrokStreamEvent decodes a single line from Grok's headless
// streaming-json output. Blank, non-object, and malformed lines are ignored;
// non-JSON banners are emitted by some CLI versions before the stream starts.
func ParseGrokStreamEvent(line []byte) (GrokStreamEvent, bool) {
	line = bytes.TrimSpace(line)
	if len(line) == 0 || line[0] != '{' {
		return GrokStreamEvent{}, false
	}
	var event GrokStreamEvent
	if json.Unmarshal(line, &event) != nil {
		return GrokStreamEvent{}, false
	}
	return event, true
}

// ParseGrokUpdate decodes Grok's on-disk ACP stream. Both standard and
// vendor-namespaced turn completion methods end in session/update.
func ParseGrokUpdate(line []byte) (GrokUpdate, bool) {
	line = bytes.TrimSpace(line)
	if len(line) == 0 {
		return GrokUpdate{}, false
	}
	var wire struct {
		Method string `json:"method"`
		Params struct {
			SessionID string `json:"sessionId"`
			Update    struct {
				Kind       string          `json:"sessionUpdate"`
				Content    json.RawMessage `json:"content"`
				StopReason string          `json:"stop_reason"`
				ToolCallID string          `json:"toolCallId"`
				Title      string          `json:"title"`
				Status     string          `json:"status"`
				RawInput   json.RawMessage `json:"rawInput"`
				RawOutput  struct {
					OutputForPrompt string `json:"output_for_prompt"`
				} `json:"rawOutput"`
			} `json:"update"`
			Meta struct {
				EventID string `json:"eventId"`
			} `json:"_meta"`
		} `json:"params"`
	}
	if json.Unmarshal(line, &wire) != nil || !strings.HasSuffix(wire.Method, "session/update") || wire.Params.Update.Kind == "" {
		return GrokUpdate{}, false
	}
	return GrokUpdate{
		Method: wire.Method, SessionID: wire.Params.SessionID, Kind: wire.Params.Update.Kind,
		Text: GrokContentText(wire.Params.Update.Content), StopReason: wire.Params.Update.StopReason,
		EventID: wire.Params.Meta.EventID, ToolCallID: wire.Params.Update.ToolCallID,
		Title: wire.Params.Update.Title, Status: wire.Params.Update.Status,
		Content: wire.Params.Update.Content, RawInput: wire.Params.Update.RawInput,
		RawOutput: wire.Params.Update.RawOutput.OutputForPrompt,
	}, true
}

// GrokContentText returns prose from a native ACP message chunk or tool result.
func GrokContentText(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var object struct {
		Type string `json:"type"`
		Text string `json:"text"`
	}
	if json.Unmarshal(raw, &object) == nil && object.Text != "" {
		return object.Text
	}
	var parts []struct {
		Content struct {
			Text string `json:"text"`
		} `json:"content"`
	}
	if json.Unmarshal(raw, &parts) != nil {
		return ""
	}
	var text strings.Builder
	for _, part := range parts {
		text.WriteString(part.Content.Text)
	}
	return text.String()
}

// GrokToolMessage formats the selectable tool content already decoded from one
// ACP update. Keeping this next to ParseGrokUpdate lets consumers avoid
// parsing the same line again through ParseMessages.
func GrokToolMessage(update GrokUpdate) string {
	switch update.Kind {
	case "tool_call":
		return FormatToolEvent(update.Title, "call", update.ToolCallID, update.RawInput)
	case "tool_call_update":
		if update.RawOutput == "" {
			return ""
		}
		raw, _ := json.Marshal(update.RawOutput)
		return FormatToolEvent(update.Title, "result", update.ToolCallID, raw)
	default:
		return ""
	}
}
