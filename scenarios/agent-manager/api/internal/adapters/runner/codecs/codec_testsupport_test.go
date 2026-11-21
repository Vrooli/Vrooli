package codecs

import (
	"testing"

	"agent-manager/internal/adapters/runner"
	"agent-manager/internal/domain"

	"github.com/google/uuid"
)

// codec_testsupport_test.go is a shared, test-only toolkit for the codec decode
// tests. The four codec test files historically hand-rolled the same two shapes
// hundreds of times: raw inline JSONL string literals, and
// `data, ok := events[i].Data.(*domain.XData); if !ok || ... { t.Fatalf }`
// type-assert-and-compare blocks. Both are brittle — a wire-format tweak or an
// event-struct rename means a find-replace across thousands of lines — and both
// copy-paste silently, so a bug in one assertion hides everywhere.
//
// The helpers here collapse those idioms to one source. A builder bug or an
// assertion bug is now caught in one place rather than per copy. These are pure
// test infrastructure: they do NOT touch any production parse/emit path, and the
// golden/corpus fixtures remain the byte-identical safety spine for the
// refactor.

// eventsOfType returns every event payload of the concrete domain pointer type
// P, in emission order. It is the collector most decode tests need ("give me all
// the messages / tool calls this trace produced") without an inline type-switch
// loop. P is a pointer type (e.g. *domain.MessageEventData) because that is what
// RunEvent.Data holds.
func eventsOfType[P domain.EventPayload](events []*domain.RunEvent) []P {
	var out []P
	for _, e := range events {
		if e == nil {
			continue
		}
		if d, ok := e.Data.(P); ok {
			out = append(out, d)
		}
	}
	return out
}

// firstOfType returns the first event payload of pointer type P, failing the
// test if none is present. Replaces the repeated "loop, find the one XData,
// t.Fatal if absent" scaffold.
func firstOfType[P domain.EventPayload](t *testing.T, events []*domain.RunEvent) P {
	t.Helper()
	got := eventsOfType[P](events)
	if len(got) == 0 {
		var zero P
		t.Fatalf("expected at least one %T event, got none in %d events", zero, len(events))
	}
	return got[0]
}

// requireType type-asserts events[idx] to the concrete domain pointer type P
// with a precise failure message, replacing the `data, ok := events[i].Data.(*X);
// if !ok { t.Fatalf(...) }` block. It also bounds-checks idx.
func requireType[P domain.EventPayload](t *testing.T, events []*domain.RunEvent, idx int) P {
	t.Helper()
	if idx < 0 || idx >= len(events) {
		var zero P
		t.Fatalf("event index %d out of range (have %d events), want %T", idx, len(events), zero)
	}
	d, ok := events[idx].Data.(P)
	if !ok {
		var zero P
		t.Fatalf("event[%d] = %T, want %T", idx, events[idx].Data, zero)
	}
	return d
}

// requireMessage asserts events[idx] is a message with the given role and
// content. role=="" skips the role check; content is always checked.
func requireMessage(t *testing.T, events []*domain.RunEvent, idx int, role, content string) *domain.MessageEventData {
	t.Helper()
	msg := requireType[*domain.MessageEventData](t, events, idx)
	if role != "" && msg.Role != role {
		t.Fatalf("event[%d] role = %q, want %q", idx, msg.Role, role)
	}
	if msg.Content != content {
		t.Fatalf("event[%d] content = %q, want %q", idx, msg.Content, content)
	}
	return msg
}

// requireToolCall asserts events[idx] is a tool call with the given tool name
// and returns its payload so callers can inspect Input. name=="" skips the name
// check.
func requireToolCall(t *testing.T, events []*domain.RunEvent, idx int, name string) *domain.ToolCallEventData {
	t.Helper()
	tc := requireType[*domain.ToolCallEventData](t, events, idx)
	if name != "" && tc.ToolName != name {
		t.Fatalf("event[%d] tool name = %q, want %q", idx, tc.ToolName, name)
	}
	return tc
}

// feedStream returns a feeder that drives one line through the live stream
// decoder (DecodeStreamLine). Decode must never error for a drift line — the
// line is dropped and the failure recorded — so an error fails the test.
func feedStream(newCodec func() Codec) func(t *testing.T, runID uuid.UUID, line string) {
	return func(t *testing.T, runID uuid.UUID, line string) {
		t.Helper()
		c := newCodec()
		if _, err := c.DecodeStreamLine(c.NewState(), runID, line); err != nil {
			t.Fatalf("DecodeStreamLine returned error (behavior changed): %v", err)
		}
	}
}

// feedTranscript returns a feeder that drives one line through the transcript
// (replay) parser, for sites whose instrumentation lives on the on-disk dialect
// rather than the stdout stream.
func feedTranscript(newCodec func() Codec) func(t *testing.T, runID uuid.UUID, line string) {
	return func(t *testing.T, runID uuid.UUID, line string) {
		t.Helper()
		_ = newCodec().NewTranscriptParser().ParseTranscriptLine(runID, line)
	}
}

// countMessages counts message events matching role and content. role=="" or
// content=="" wildcards that field. Replaces the repeated "loop, type-assert,
// if role==x && content==y { n++ }" occurrence counters.
func countMessages(events []*domain.RunEvent, role, content string) int {
	n := 0
	for _, m := range eventsOfType[*domain.MessageEventData](events) {
		if role != "" && m.Role != role {
			continue
		}
		if content != "" && m.Content != content {
			continue
		}
		n++
	}
	return n
}

// requireTerminalUsage returns the authoritative (ReconciliationAuthority) usage
// event from a transcript parse result, failing if none is present. The codex
// transcript tests repeatedly need "the terminal receipt" and had re-declared
// this closure inline in several functions.
func requireTerminalUsage(t *testing.T, result runner.TranscriptParseResult) *domain.UsageEventData {
	t.Helper()
	for _, evt := range result.Events {
		if usage, ok := evt.Data.(*domain.UsageEventData); ok && usage.ReconciliationAuthority {
			return usage
		}
	}
	t.Fatalf("terminal receipt missing: %+v", result.Events)
	return nil
}

// decodeLines replays lines through a single codec state (the live scan loop)
// and returns the accumulated events, failing on any decode error. Codec-
// specific trace helpers (grokDecodeTrace etc.) stay where format handling
// differs; this is the plain shared loop for already-split lines.
func decodeLines(t *testing.T, c Codec, runID uuid.UUID, lines []string) []*domain.RunEvent {
	t.Helper()
	state := c.NewState()
	var events []*domain.RunEvent
	for _, line := range lines {
		evs, err := c.DecodeStreamLine(state, runID, line)
		if err != nil {
			t.Fatalf("DecodeStreamLine(%q): %v", line, err)
		}
		events = append(events, evs...)
	}
	return events
}
