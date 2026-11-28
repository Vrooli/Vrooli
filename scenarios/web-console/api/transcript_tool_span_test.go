package main

import (
	"strings"
	"testing"
	"time"

	"web-console/backends/grok"
	"web-console/internal/tailer"
)

func TestClaudeAndCodexTranscriptSourcesEmitSelectableToolEvents(t *testing.T) {
	cases := []struct {
		name   string
		source interface {
			DecodeLine(string, []byte) ([]tailer.Event, error)
		}
		line string
		id   string
	}{
		{"Claude", claudeTranscriptSource{}, `{"type":"assistant","message":{"content":[{"type":"tool_use","id":"claude-call","name":"Bash","input":{"command":"pwd"}}]}}`, "claude-call"},
		{"Codex", codexTranscriptSource{}, `{"type":"response_item","payload":{"type":"function_call","call_id":"codex-call","name":"exec","arguments":"{\"cmd\":\"pwd\"}"}}`, "codex-call"},
	}
	for _, tc := range cases {
		events, err := tc.source.DecodeLine("", []byte(tc.line))
		if err != nil || len(events) != 1 || events[0].Role != "assistant" || !strings.Contains(events[0].Text, tc.id) || !strings.Contains(events[0].Text, "pwd") {
			t.Errorf("%s tool span = %#v, err=%v", tc.name, events, err)
		}
	}
}

func TestClaudeAndCodexLiveTranscriptSourcesPreserveOccurrenceTime(t *testing.T) {
	want := time.Date(2026, 7, 22, 2, 36, 55, 108000000, time.UTC)
	cases := []struct {
		name   string
		source interface {
			DecodeLine(string, []byte) ([]tailer.Event, error)
		}
		line string
	}{
		{"Claude", claudeTranscriptSource{}, `{"type":"assistant","timestamp":"2026-07-22T02:36:55.108Z","message":{"content":"answer"}}`},
		{"Codex", codexTranscriptSource{}, `{"timestamp":"2026-07-22T02:36:55.108Z","type":"event_msg","payload":{"type":"user_message","message":"question"}}`},
	}
	for _, tc := range cases {
		events, err := tc.source.DecodeLine("", []byte(tc.line))
		if err != nil || len(events) != 1 || !events[0].OccurredAt.Equal(want) {
			t.Errorf("%s live decode = %#v, err=%v; want source time %s", tc.name, events, err, want)
		}
	}
}

func TestGrokTranscriptSourceEmitsToolCallsAndResultsAtTurnBoundary(t *testing.T) {
	lines := [][]byte{
		[]byte(`{"method":"session/update","params":{"sessionId":"local-target","update":{"sessionUpdate":"user_message_chunk","content":{"type":"text","text":"question"}}}}`),
		[]byte(`{"method":"session/update","params":{"sessionId":"local-target","update":{"sessionUpdate":"tool_call","toolCallId":"grok-call","title":"Shell","rawInput":{"command":"pwd"}}}}`),
		[]byte(`{"method":"session/update","params":{"sessionId":"local-target","update":{"sessionUpdate":"tool_call_update","toolCallId":"grok-call","title":"Shell","rawOutput":{"output_for_prompt":"/operator-owned"}}}}`),
		[]byte(`{"method":"session/update","params":{"sessionId":"local-target","update":{"sessionUpdate":"agent_message_chunk","content":{"type":"text","text":"done"}}}}`),
		[]byte(`{"method":"session/update","params":{"sessionId":"local-target","update":{"sessionUpdate":"turn_completed"}}}`),
	}
	source := &grokTranscriptSource{turns: make(map[string]grok.TurnAccumulator)}
	var events []tailer.Event
	for _, line := range lines {
		got, err := source.DecodeLine("local-transcript", line)
		if err != nil {
			t.Fatal(err)
		}
		events = append(events, got...)
	}
	if len(events) != 4 || !strings.Contains(events[1].Text, "grok-call") || !strings.Contains(events[2].Text, "/operator-owned") {
		t.Fatalf("Grok emitted selectable tool span incorrectly: %#v", events)
	}
}
