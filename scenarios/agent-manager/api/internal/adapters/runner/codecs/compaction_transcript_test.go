package codecs

import (
	"encoding/json"
	"testing"
	"time"

	"agent-manager/internal/adapters/runner"
	"agent-manager/internal/domain"

	"github.com/google/uuid"
)

// A compact_boundary line in a Claude transcript, and a `compacted` record in
// a Codex rollout, each yield one compaction event at the line's own time, so
// projections can end a result's residency there.
func TestTranscriptCompactionMarkersYieldCompactionEvents(t *testing.T) {
	at := time.Date(2026, 10, 28, 21, 14, 43, 552000000, time.UTC)
	claude := NewClaudeForTest().NewTranscriptParser()
	result := claude.ParseTranscriptLine(uuid.New(), `{"type":"system","subtype":"compact_boundary","content":"Conversation compacted","timestamp":"2026-10-28T21:14:43.552Z","compactMetadata":{"trigger":"auto","preTokens":287094}}`)
	if len(result.Events) != 1 {
		t.Fatalf("claude events=%d, want 1", len(result.Events))
	}
	data := requireType[*domain.CompactionEventData](t, result.Events, 0)
	if data.Trigger != "auto" || data.TokensBefore != 287094 || !result.Events[0].Timestamp.Equal(at) {
		t.Fatalf("claude compaction=%+v at %v", data, result.Events[0].Timestamp)
	}

	codex := NewCodexForTest().NewTranscriptParser()
	result = codex.ParseTranscriptLine(uuid.New(), `{"timestamp":"2026-10-11T22:40:00.000Z","type":"compacted","payload":{"message":"","replacement_history":[]}}`)
	if len(result.Events) != 1 {
		t.Fatalf("codex events=%d, want 1", len(result.Events))
	}
	requireType[*domain.CompactionEventData](t, result.Events, 0)
	if want := time.Date(2026, 10, 11, 22, 40, 0, 0, time.UTC); !result.Events[0].Timestamp.Equal(want) {
		t.Fatalf("codex compaction at %v, want %v", result.Events[0].Timestamp, want)
	}
}

// A parsed event is final when the parser returns it: sinks store it then (the
// transcript indexer flushes every 500 events and at the end of each read), so
// a value set on it by a later line would be stored or not depending on
// where a batch ended. The Claude transcript does not state the size after a
// compaction, so TokensAfter stays 0 (unknown) and the next request's usage
// event carries that size.
func TestClaudeCompactionIsNotRewrittenByALaterLine(t *testing.T) {
	parser := NewClaudeForTest().NewTranscriptParser()
	result := parser.ParseTranscriptLine(uuid.New(), `{"type":"system","subtype":"compact_boundary","content":"Conversation compacted","timestamp":"2026-10-28T21:14:43.552Z","compactMetadata":{"trigger":"auto","preTokens":287094}}`)
	data := requireType[*domain.CompactionEventData](t, result.Events, 0)
	stored := *data // what a sink that flushed here wrote
	parser.ParseTranscriptLine(uuid.New(), claudeOpenTurnEndOfTurn)
	if *data != stored || data.TokensAfter != 0 {
		t.Fatalf("a later line rewrote the returned compaction: stored %+v, now %+v", stored, *data)
	}
}

// Codex carries a thread's native goal into its compacted history as a
// user_goal context message. The compaction event records whether that message
// kept the objective or replaced it with "[objective omitted]"; quoted text in
// other retained messages never decides it.
func TestCodexCompactionRecordsWhetherTheGoalObjectiveSurvived(t *testing.T) {
	type content struct {
		Type string `json:"type"`
		Text string `json:"text"`
	}
	type item struct {
		Type    string    `json:"type"`
		Role    string    `json:"role"`
		Content []content `json:"content"`
	}
	goalContext := func(objective string) string {
		return "<codex_internal_context source=\"user_goal\">\nUser set the goal: " + objective + ".\nUser set goal status: \"active\".\n</codex_internal_context>"
	}
	goal := func(objective string) item {
		return item{Type: "message", Role: "user", Content: []content{{Type: "input_text", Text: goalContext(objective)}}}
	}
	quoted := item{Type: "message", Role: "assistant", Content: []content{{Type: "output_text", Text: goalContext("[objective omitted; exceeds the evidence limit]")}}}
	line := func(history []item, retained ...string) string {
		payload := map[string]any{"message": "", "replacement_history": history}
		if len(retained) > 0 {
			messages := []map[string]string{}
			for _, text := range retained {
				messages = append(messages, map[string]string{"text": text})
			}
			payload["retained_context"] = map[string]any{"user_messages": messages}
		}
		raw, err := json.Marshal(map[string]any{"timestamp": "2026-11-05T13:41:13.000Z", "type": "compacted", "payload": payload})
		if err != nil {
			t.Fatal(err)
		}
		return string(raw)
	}
	omitted := "[objective omitted; exceeds the evidence limit]"
	for name, tc := range map[string]struct {
		line string
		want string
	}{
		"omitted":                 {line([]item{goal(omitted)}), domain.CompactionGoalOmitted},
		"retained":                {line([]item{goal(`"Ship the kernel"`)}), domain.CompactionGoalRetained},
		"no native goal":          {line(nil), ""},
		"quoted marker only":      {line([]item{quoted}), ""},
		"retained despite quote":  {line([]item{quoted, goal(`"Ship the kernel"`)}), domain.CompactionGoalRetained},
		"retained context record": {line(nil, goalContext(omitted)), domain.CompactionGoalOmitted},
	} {
		t.Run(name, func(t *testing.T) {
			result := NewCodexForTest().NewTranscriptParser().ParseTranscriptLine(uuid.New(), tc.line)
			data := requireType[*domain.CompactionEventData](t, result.Events, 0)
			if data.GoalObjective != tc.want {
				t.Fatalf("GoalObjective = %q, want %q", data.GoalObjective, tc.want)
			}
		})
	}
}

// retained_context lists the user messages a compaction kept. Only operator-
// typed messages carry a stance: goal events, Agent Manager wakes and injected
// instructions do not count. A sub-agent thread keeps no user messages by
// design, so its count is unknown, not zero.
func TestCodexCompactionCountsTheOperatorMessagesItKept(t *testing.T) {
	line := func(messages ...string) string {
		retained := []map[string]string{}
		for _, text := range messages {
			retained = append(retained, map[string]string{"text": text})
		}
		raw, err := json.Marshal(map[string]any{"timestamp": "2026-11-05T13:41:13.000Z", "type": "compacted", "payload": map[string]any{"message": "", "replacement_history": []any{}, "retained_context": map[string]any{"user_messages": retained}}})
		if err != nil {
			t.Fatal(err)
		}
		return string(raw)
	}
	goal := "<codex_internal_context source=\"user_goal\">\nUser set the goal: \"Ship\".\n</codex_internal_context>"
	wake := "[wake] child 1234 settled"
	stance := "Operator stance: be bold, take the whole migration."
	parse := func(parser runner.TranscriptParser, l string) *domain.CompactionEventData {
		return requireType[*domain.CompactionEventData](t, parser.ParseTranscriptLine(uuid.New(), l).Events, 0)
	}
	if kept := parse(NewCodexForTest().NewTranscriptParser(), line(goal, wake, stance)).OperatorMessagesKept; kept == nil || *kept != 1 {
		t.Fatalf("kept = %v, want 1 operator message", kept)
	}
	if kept := parse(NewCodexForTest().NewTranscriptParser(), line(goal, wake, wake)).OperatorMessagesKept; kept == nil || *kept != 0 {
		t.Fatalf("kept = %v, want 0 when only goal events and wakes survived", kept)
	}
	if kept := parse(NewCodexForTest().NewTranscriptParser(), `{"timestamp":"2026-10-11T22:40:00.000Z","type":"compacted","payload":{"message":"","replacement_history":[]}}`).OperatorMessagesKept; kept != nil {
		t.Fatalf("kept = %d for a record without retained_context, want unknown", *kept)
	}
	sub := NewCodexForTest().NewTranscriptParser()
	sub.ParseTranscriptLine(uuid.New(), `{"timestamp":"2026-11-05T13:00:00.000Z","type":"session_meta","payload":{"id":"01a1b27d-f795-7141-a484-5f4509a4ba6b","source":{"subagent":{"thread_spawn":{"parent_thread_id":"01a1ad03-daca-7c53-8d59-fef0fc7c3d07","depth":1}}}}}`)
	if kept := parse(sub, line(goal)).OperatorMessagesKept; kept != nil {
		t.Fatalf("kept = %d for a sub-agent thread, want unknown", *kept)
	}
}
