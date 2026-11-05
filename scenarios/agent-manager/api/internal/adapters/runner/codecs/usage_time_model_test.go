package codecs

import (
	"testing"
	"time"

	"agent-manager/internal/adapters/runner"
	"agent-manager/internal/domain"

	"github.com/google/uuid"
)

func usageEvents(t *testing.T, parser runner.TranscriptParser, lines ...string) []*domain.RunEvent {
	t.Helper()
	runID := uuid.New()
	var out []*domain.RunEvent
	for _, line := range lines {
		for _, event := range parser.ParseTranscriptLine(runID, line).Events {
			if usage, ok := event.Data.(*domain.UsageEventData); ok && usage.PayloadKind == domain.PayloadKindUsage {
				out = append(out, event)
			}
		}
	}
	return out
}

// A transcript read days later (indexer back-fill, recovery replay) must
// stamp each usage report with its line's time and the model the turn ran,
// not the parse time and the run's configured (or unknown) model.
func TestTranscriptUsageCarriesLineTimeAndTheTurnsModel(t *testing.T) {
	want := time.Date(2026, 10, 28, 9, 0, 2, 0, time.UTC)

	codex := NewCodexForTest().NewTranscriptParser()
	codex.(runner.TranscriptModelSetter).SetTranscriptModel("")
	events := usageEvents(t, codex,
		`{"timestamp":"2026-10-28T09:00:00Z","type":"session_meta","payload":{"id":"thread-1"}}`,
		`{"timestamp":"2026-10-28T09:00:01Z","type":"turn_context","payload":{"model":"gpt-6.1-terra"}}`,
		`{"timestamp":"2026-10-28T09:00:02Z","type":"event_msg","payload":{"type":"token_count","info":{"total_token_usage":{"input_tokens":100,"cached_input_tokens":40,"output_tokens":5},"last_token_usage":{"input_tokens":100,"cached_input_tokens":40,"output_tokens":5}}}}`,
	)
	if len(events) != 1 {
		t.Fatalf("codex usage events = %d, want 1", len(events))
	}
	if usage := events[0].Data.(*domain.UsageEventData); usage.Model != "gpt-6.1-terra" || !events[0].Timestamp.Equal(want) {
		t.Errorf("codex usage = model %q at %s, want gpt-6.1-terra at %s", usage.Model, events[0].Timestamp, want)
	}

	claude := NewClaudeForTest().NewTranscriptParser()
	claude.(runner.TranscriptModelSetter).SetTranscriptModel("")
	events = usageEvents(t, claude,
		`{"type":"assistant","sessionId":"s","timestamp":"2026-10-28T09:00:02Z","message":{"id":"msg_1","model":"claude-haiku-5-5","role":"assistant","content":[{"type":"text","text":"hi"}],"usage":{"input_tokens":3,"output_tokens":2}}}`,
	)
	if len(events) != 1 {
		t.Fatalf("claude usage events = %d, want 1", len(events))
	}
	if usage := events[0].Data.(*domain.UsageEventData); usage.Model != "claude-haiku-5-5" || !events[0].Timestamp.Equal(want) {
		t.Errorf("claude usage = model %q at %s, want claude-haiku-5-5 at %s", usage.Model, events[0].Timestamp, want)
	}
}
