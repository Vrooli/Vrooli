package grok

import (
	"strings"

	"github.com/vrooli/cli-core/agenttranscript"
)

// Session-update kinds observed in grok's updates.jsonl (ACP stream). User and
// assistant prose plus explicit tool calls/results are selectable transcript
// events; thought chunks remain private and are not appended.
const (
	KindUserMessage    = "user_message_chunk"
	KindAgentMessage   = "agent_message_chunk"
	KindAgentThought   = "agent_thought_chunk"
	KindToolCall       = "tool_call"
	KindToolCallUpdate = "tool_call_update"
	KindTurnCompleted  = "turn_completed"
)

// UpdateRecord is the normalized form of one updates.jsonl line. Ok is false
// for blank lines, malformed JSON, or records that carry no recognized
// session-update kind, so callers can skip without advancing past unparseable
// content.
type UpdateRecord struct {
	SessionID  string
	Kind       string
	Text       string // content.text for chunk kinds; empty otherwise
	StopReason string // turn_completed only
	EventID    string
	ToolText   string
}

// ParseUpdateLine decodes a single updates.jsonl line into an UpdateRecord.
// Returns ok=false when the line is not a recognized session/update record.
func ParseUpdateLine(line []byte) (UpdateRecord, bool) {
	record, ok := agenttranscript.ParseGrokUpdate(line)
	if !ok {
		return UpdateRecord{}, false
	}
	out := UpdateRecord{
		SessionID:  record.SessionID,
		Kind:       record.Kind,
		Text:       record.Text,
		StopReason: record.StopReason,
		EventID:    record.EventID,
	}
	if record.Kind == KindToolCall || record.Kind == KindToolCallUpdate {
		out.ToolText = agenttranscript.GrokToolMessage(record)
	}
	return out, true
}

// CompletedTurn carries the user/assistant prose and tool records for one
// finished Grok turn. Either prose field may be empty for tool-only turns.
type CompletedTurn struct {
	User      string
	Assistant string
	ToolCalls []string
}

// TurnAccumulator coalesces streamed chunk records into per-turn user and
// assistant text plus tool records. Grok emits a turn as: user_message_chunk(s) → interleaved
// thought/agent/tool records → turn_completed. Emitting only at the
// turn_completed boundary (rather than per chunk) is what makes the byte-offset
// checkpoint replay-safe: a checkpoint is only advanced past a boundary, so a
// restart mid-turn re-reads and re-accumulates the incomplete turn without
// having emitted anything for it yet.
type TurnAccumulator struct {
	user      strings.Builder
	assistant strings.Builder
	toolCalls []string
}

// Add folds one record into the in-progress turn. When the record is a
// turn-completion boundary it returns the accumulated turn and resets; ok is
// false for every non-boundary record.
func (a *TurnAccumulator) Add(rec UpdateRecord) (CompletedTurn, bool) {
	switch rec.Kind {
	case KindUserMessage:
		a.user.WriteString(rec.Text)
	case KindAgentMessage:
		a.assistant.WriteString(rec.Text)
	case KindToolCall, KindToolCallUpdate:
		if rec.ToolText != "" {
			a.toolCalls = append(a.toolCalls, rec.ToolText)
		}
	case KindTurnCompleted:
		turn := CompletedTurn{User: a.user.String(), Assistant: a.assistant.String(), ToolCalls: append([]string(nil), a.toolCalls...)}
		a.user.Reset()
		a.assistant.Reset()
		a.toolCalls = nil
		return turn, true
	}
	return CompletedTurn{}, false
}
