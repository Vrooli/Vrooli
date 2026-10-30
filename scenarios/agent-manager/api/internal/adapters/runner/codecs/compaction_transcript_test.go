package codecs

import (
	"testing"
	"time"

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
