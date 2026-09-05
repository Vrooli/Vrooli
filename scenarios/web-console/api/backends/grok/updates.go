package grok

import (
	"encoding/json"
	"strings"
)

// Session-update kinds observed in grok's updates.jsonl (ACP stream). Only the
// natural-language user/assistant chunks feed the semantic conversation; thought
// chunks and tool lifecycle are parsed but intentionally not appended (see the
// sensitive-content mitigation in the implementation plan).
const (
	KindUserMessage    = "user_message_chunk"
	KindAgentMessage   = "agent_message_chunk"
	KindAgentThought   = "agent_thought_chunk"
	KindToolCall       = "tool_call"
	KindToolCallUpdate = "tool_call_update"
	KindTurnCompleted  = "turn_completed"
)

// rawUpdate is the on-disk JSONL shape. grok writes one ACP notification per
// line under either the standard `session/update` method or the x.ai-specific
// `_x.ai/session/update` method (turn lifecycle).
type rawUpdate struct {
	Method string `json:"method"`
	Params struct {
		SessionID string `json:"sessionId"`
		Update    struct {
			SessionUpdate string `json:"sessionUpdate"`
			Content       struct {
				Type string `json:"type"`
				Text string `json:"text"`
			} `json:"content"`
			StopReason string `json:"stop_reason"`
		} `json:"update"`
		Meta struct {
			EventID string `json:"eventId"`
		} `json:"_meta"`
	} `json:"params"`
}

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
}

// ParseUpdateLine decodes a single updates.jsonl line into an UpdateRecord.
// Returns ok=false when the line is not a recognized session/update record.
func ParseUpdateLine(line []byte) (UpdateRecord, bool) {
	line = trimSpace(line)
	if len(line) == 0 {
		return UpdateRecord{}, false
	}
	var ru rawUpdate
	if err := json.Unmarshal(line, &ru); err != nil {
		return UpdateRecord{}, false
	}
	kind := ru.Params.Update.SessionUpdate
	if kind == "" {
		return UpdateRecord{}, false
	}
	return UpdateRecord{
		SessionID:  ru.Params.SessionID,
		Kind:       kind,
		Text:       ru.Params.Update.Content.Text,
		StopReason: ru.Params.Update.StopReason,
		EventID:    ru.Params.Meta.EventID,
	}, true
}

func trimSpace(b []byte) []byte {
	return []byte(strings.TrimSpace(string(b)))
}

// CompletedTurn carries the accumulated natural-language text for one finished
// grok turn. Either field may be empty (e.g. a tool-only turn yields no
// assistant prose); callers drop empty halves.
type CompletedTurn struct {
	User      string
	Assistant string
}

// TurnAccumulator coalesces streamed chunk records into per-turn user and
// assistant text. grok emits a turn as: user_message_chunk(s) → interleaved
// thought/agent/tool records → turn_completed. Emitting only at the
// turn_completed boundary (rather than per chunk) is what makes the byte-offset
// checkpoint replay-safe: a checkpoint is only advanced past a boundary, so a
// restart mid-turn re-reads and re-accumulates the incomplete turn without
// having emitted anything for it yet.
type TurnAccumulator struct {
	user      strings.Builder
	assistant strings.Builder
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
	case KindTurnCompleted:
		turn := CompletedTurn{User: a.user.String(), Assistant: a.assistant.String()}
		a.user.Reset()
		a.assistant.Reset()
		return turn, true
	}
	return CompletedTurn{}, false
}
