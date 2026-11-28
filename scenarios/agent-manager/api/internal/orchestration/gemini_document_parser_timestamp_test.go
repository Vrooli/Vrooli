package orchestration

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestGeminiDocumentParserKeepsSourceOccurrenceTime(t *testing.T) {
	parser := &geminiDocumentParser{source: "fixture.gemini"}
	result := parser.ParseTranscriptDocument(uuid.New(), []byte(`{"sessionId":"session-safe","messages":[{"type":"user","content":"older question","timestamp":"2025-10-17T14:27:00.123456789Z"},{"type":"gemini","content":"unknown time"}]}`))
	if result.Err != nil || len(result.Events) != 2 {
		t.Fatalf("parse result = %+v", result)
	}
	want := time.Date(2025, 10, 17, 14, 27, 0, 123456789, time.UTC)
	if !result.Events[0].Timestamp.Equal(want) {
		t.Fatalf("source event time = %s, want %s", result.Events[0].Timestamp, want)
	}
	if result.Events[1].Timestamp.Equal(want) {
		t.Fatal("missing source timestamp inherited another message's occurrence time")
	}
}
