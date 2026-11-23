package codecs

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestClaudeParserCarriesRecordTimeToMessageEvent(t *testing.T) {
	parser := NewClaudeForTest().NewTranscriptParser()
	want := time.Date(2026, 7, 22, 2, 36, 55, 108000000, time.UTC)
	result := parser.ParseTranscriptLine(uuid.New(), `{"type":"assistant","timestamp":"2026-07-22T02:36:55.108Z","session_id":"session-safe","message":{"role":"assistant","content":[{"type":"text","text":"answer"}]}}`)
	if result.Err != nil || len(result.Events) != 1 || !result.Timestamp.Equal(want) || !result.Events[0].Timestamp.Equal(want) {
		t.Fatalf("Claude parse result=%+v, want message timestamp %s", result, want)
	}
}

func TestCodexParserCarriesRolloutRecordTimeToMessageEvent(t *testing.T) {
	parser := NewCodexForTest().NewTranscriptParser()
	want := time.Date(2026, 7, 22, 2, 36, 55, 108000000, time.UTC)
	result := parser.ParseTranscriptLine(uuid.New(), `{"timestamp":"2026-07-22T02:36:55.108Z","type":"response_item","payload":{"type":"message","role":"assistant","content":[{"type":"output_text","text":"answer"}]}}`)
	if result.Err != nil || len(result.Events) != 1 || !result.Timestamp.Equal(want) || !result.Events[0].Timestamp.Equal(want) {
		t.Fatalf("Codex parse result=%+v, want message timestamp %s", result, want)
	}
}
