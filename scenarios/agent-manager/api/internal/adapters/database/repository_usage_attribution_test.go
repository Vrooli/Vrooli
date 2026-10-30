package database

import (
	"context"
	"testing"
	"time"

	"agent-manager/internal/domain"
	"agent-manager/internal/runsignal"
	"agent-manager/internal/transcriptindex"

	"github.com/google/uuid"
)

func seedUsageFile(t *testing.T, db *DB, file transcriptindex.FileRow, rows []transcriptindex.UsageRow) {
	t.Helper()
	if err := NewTranscriptIndexRepository(db).SaveTranscriptFile(context.Background(), file, rows); err != nil {
		t.Fatal(err)
	}
}

func codexReport(fileKey, runID string, offset int64, at time.Time, cumulative, amount domain.UsageCounters) transcriptindex.UsageRow {
	usage := domain.UsageEventData{
		PayloadKind: domain.PayloadKindUsage, SessionID: "thread", Model: "gpt-6.1-terra",
		InputTokens: amount.InputTokens, OutputTokens: amount.OutputTokens, CacheReadTokens: amount.CacheReadTokens, Cumulative: &cumulative,
	}
	return transcriptindex.UsageRow{FileKey: fileKey, LineOffset: offset, RunID: runID, ObservedAt: at, Usage: usage}
}

// UsageTurns folds Codex reports into short buckets without changing what
// the one accounting function credits, and reads one source: the indexer's
// usage rows, including those of run-owned files keyed to the Agent Manager
// run (D-16). Run events — an imported run's recall layer, an Agent Manager
// run's own usage events, a legacy Codex event without a cumulative counter —
// are never counted.
func TestUsageTurnsCountEachReportOnceFromTranscriptUsage(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	ctx := context.Background()
	start := time.Date(2026, 10, 28, 9, 0, 0, 0, time.UTC)
	imported, owned, legacy := uuid.NewString(), uuid.NewString(), uuid.NewString()
	for _, run := range []struct{ id, mode string }{{imported, "imported"}, {owned, "codec_pipe"}, {legacy, "interactive"}} {
		if _, err := db.ExecContext(ctx, `INSERT INTO runs (id, execution_mode, status, label, resolved_config) VALUES (?, ?, 'complete', 'run', '{"runnerType":"codex"}')`, run.id, run.mode); err != nil {
			t.Fatal(err)
		}
	}
	// A fork file whose counters start at the parent's 1000: three reports
	// over two buckets, one repeated by a re-read.
	c := func(input, cached int) domain.UsageCounters {
		return domain.UsageCounters{InputTokens: input, CacheReadTokens: cached}
	}
	reports := []transcriptindex.UsageRow{
		codexReport("1:1", imported, 10, start.Add(time.Minute), c(1100, 5000), c(100, 5000)),
		codexReport("1:1", imported, 20, start.Add(2*time.Minute), c(1300, 9000), c(200, 4000)),
		codexReport("1:1", imported, 30, start.Add(25*time.Minute), c(1350, 12000), c(50, 3000)),
	}
	seedUsageFile(t, db, transcriptindex.FileRow{Key: "1:1", Path: "/f", RootKind: transcriptindex.RootCodexHome, Harness: transcriptindex.HarnessCodex, RunID: imported, Owner: transcriptindex.OwnerIndexer, Status: transcriptindex.StatusIndexed, MtimeUnixNano: start.Add(time.Hour).UnixNano(), FirstSeenAt: start}, reports)
	seedUsageFile(t, db, transcriptindex.FileRow{Key: "3:3", Path: "/run-home/rollout.jsonl", RootKind: transcriptindex.RootRunHome, Harness: transcriptindex.HarnessCodex, RunID: owned, Owner: transcriptindex.OwnerRun, Status: transcriptindex.StatusOwned, MtimeUnixNano: start.Add(time.Hour).UnixNano(), FirstSeenAt: start},
		[]transcriptindex.UsageRow{codexReport("3:3", owned, 10, start.Add(3*time.Minute), c(10, 90), c(10, 90))})
	event := func(runID string, seq int, at time.Time, data string) {
		if _, err := db.ExecContext(ctx, `INSERT INTO run_events (id, run_id, sequence, event_type, timestamp, data) VALUES (?, ?, ?, 'metric', ?, ?)`, uuid.NewString(), runID, seq, at.Format(time.RFC3339Nano), data); err != nil {
			t.Fatal(err)
		}
	}
	event(imported, 1, start.Add(time.Minute), `{"payloadKind":"usage","inputTokens":100,"cacheReadTokens":5000,"sessionId":"thread","cumulative":{"inputTokens":1100,"outputTokens":0,"cacheReadTokens":5000}}`)
	event(owned, 1, start.Add(3*time.Minute), `{"payloadKind":"usage","runnerType":"codex","model":"gpt-6-luna","inputTokens":10,"cacheReadTokens":90,"sessionId":"t2","cumulative":{"inputTokens":10,"outputTokens":0,"cacheReadTokens":90}}`)
	event(legacy, 1, start.Add(4*time.Minute), `{"payloadKind":"usage","runnerType":"codex","inputTokens":999999,"cacheReadTokens":999999}`)

	turns, err := NewUsageAttributionRepository(db).UsageTurns(ctx, start, start.Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	events := make([]*domain.RunEvent, len(reports))
	for i := range reports {
		usage := reports[i].Usage
		events[i] = &domain.RunEvent{EventType: domain.EventTypeMetric, Data: &usage}
	}
	want := runsignal.AccountUsage(events).Total
	var got domain.UsageCounters
	byRun := map[string]int64{}
	for _, turn := range turns {
		byRun[turn.RunID] += turn.Tokens.Total()
		if turn.RunID == imported {
			got.InputTokens += turn.Tokens.InputTokens
			got.CacheReadTokens += turn.Tokens.CacheReadTokens
			if turn.Model != "gpt-6.1-terra" || turn.Runner != "codex" {
				t.Errorf("indexed turn = %+v, want terra on codex", turn)
			}
		}
	}
	if got != want {
		t.Fatalf("indexed usage = %+v, want AccountUsage over the raw reports %+v", got, want)
	}
	if byRun[owned] != 100 || byRun[legacy] != 0 {
		t.Fatalf("run usage = %v, want the run-owned file's 100 tokens once and no run event", byRun)
	}
	if len(turns) != 3 {
		t.Fatalf("turns = %d, want two operator-session buckets and one run-owned bucket", len(turns))
	}
}
