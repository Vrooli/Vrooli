package grok

import "testing"

// A representative turn from grok's updates.jsonl: user prompt, a thought chunk,
// the assistant reply, a tool call + update, then turn completion.
var sampleLines = [][]byte{
	[]byte(`{"timestamp":1,"method":"session/update","params":{"sessionId":"s1","update":{"sessionUpdate":"user_message_chunk","content":{"type":"text","text":"hi there"}},"_meta":{"eventId":"s1-1"}}}`),
	[]byte(`{"timestamp":2,"method":"session/update","params":{"sessionId":"s1","update":{"sessionUpdate":"agent_thought_chunk","content":{"type":"text","text":"thinking privately"}}}}`),
	[]byte(`{"timestamp":3,"method":"session/update","params":{"sessionId":"s1","update":{"sessionUpdate":"agent_message_chunk","content":{"type":"text","text":"Hello! "}}}}`),
	[]byte(`{"timestamp":4,"method":"session/update","params":{"sessionId":"s1","update":{"sessionUpdate":"agent_message_chunk","content":{"type":"text","text":"How can I help?"}}}}`),
	[]byte(`{"timestamp":5,"method":"session/update","params":{"sessionId":"s1","update":{"sessionUpdate":"tool_call","toolCallId":"c1","title":"Shell"}}}`),
	[]byte(`{"timestamp":6,"method":"session/update","params":{"sessionId":"s1","update":{"sessionUpdate":"tool_call_update","toolCallId":"c1"}}}`),
	[]byte(`{"timestamp":7,"method":"_x.ai/session/update","params":{"sessionId":"s1","update":{"sessionUpdate":"turn_completed","stop_reason":"end_turn"},"_meta":{"eventId":"s1-8"}}}`),
}

func TestParseUpdateLine_AllObservedKinds(t *testing.T) {
	wantKinds := []string{
		KindUserMessage, KindAgentThought, KindAgentMessage, KindAgentMessage,
		KindToolCall, KindToolCallUpdate, KindTurnCompleted,
	}
	for i, line := range sampleLines {
		rec, ok := ParseUpdateLine(line)
		if !ok {
			t.Fatalf("line %d: expected parse ok", i)
		}
		if rec.Kind != wantKinds[i] {
			t.Fatalf("line %d: kind = %q, want %q", i, rec.Kind, wantKinds[i])
		}
		if rec.SessionID != "s1" {
			t.Fatalf("line %d: sessionID = %q", i, rec.SessionID)
		}
	}
}

func TestParseUpdateLine_RejectsMalformedAndBlank(t *testing.T) {
	for _, line := range [][]byte{
		nil,
		[]byte("   "),
		[]byte("{not json"),
		[]byte(`{"method":"session/update","params":{"sessionId":"s1","update":{}}}`), // no sessionUpdate
	} {
		if _, ok := ParseUpdateLine(line); ok {
			t.Fatalf("expected parse to fail for %q", string(line))
		}
	}
}

func TestTurnAccumulator_EmitsConcatenatedTurnAtBoundary(t *testing.T) {
	var acc TurnAccumulator
	var got *CompletedTurn
	for _, line := range sampleLines {
		rec, ok := ParseUpdateLine(line)
		if !ok {
			t.Fatalf("parse failed for %q", string(line))
		}
		if turn, done := acc.Add(rec); done {
			turnCopy := turn
			got = &turnCopy
		}
	}
	if got == nil {
		t.Fatal("expected a completed turn at turn_completed")
	}
	if got.User != "hi there" {
		t.Fatalf("user text = %q, want %q", got.User, "hi there")
	}
	// Assistant chunks concatenate; thought + tool records contribute nothing.
	if got.Assistant != "Hello! How can I help?" {
		t.Fatalf("assistant text = %q, want %q", got.Assistant, "Hello! How can I help?")
	}
}

func TestTurnAccumulator_ResetsBetweenTurns(t *testing.T) {
	var acc TurnAccumulator
	feed := func(kind, text string) (CompletedTurn, bool) {
		return acc.Add(UpdateRecord{Kind: kind, Text: text})
	}
	feed(KindUserMessage, "first")
	feed(KindAgentMessage, "reply one")
	if _, done := feed(KindTurnCompleted, ""); !done {
		t.Fatal("expected first turn to complete")
	}
	feed(KindUserMessage, "second")
	feed(KindAgentMessage, "reply two")
	turn, done := feed(KindTurnCompleted, "")
	if !done {
		t.Fatal("expected second turn to complete")
	}
	if turn.User != "second" || turn.Assistant != "reply two" {
		t.Fatalf("turn carried stale state: %+v", turn)
	}
}
