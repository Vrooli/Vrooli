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

func TestRepairUsageRowsRewritesTimeAndModelByOffset(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	ctx := context.Background()
	indexedAt := time.Date(2026, 10, 29, 7, 0, 0, 0, time.UTC)
	row := codexReport("2:2", uuid.NewString(), 40, indexedAt, domain.UsageCounters{InputTokens: 1}, domain.UsageCounters{InputTokens: 1})
	row.Usage.Model = "unknown"
	seedUsageFile(t, db, transcriptindex.FileRow{Key: "2:2", Path: "/g", Harness: transcriptindex.HarnessCodex, Owner: transcriptindex.OwnerIndexer, Status: transcriptindex.StatusIndexed, FirstSeenAt: indexedAt}, []transcriptindex.UsageRow{row})
	repo := NewTranscriptIndexRepository(db)
	files, err := repo.UsageFilesToRepair(ctx)
	if err != nil || len(files) != 1 || files[0].Path != "/g" {
		t.Fatalf("files to repair = %+v, %v", files, err)
	}
	lineTime := time.Date(2026, 10, 28, 9, 0, 2, 0, time.UTC)
	if err := repo.RepairUsageRows(ctx, "2:2", []transcriptindex.UsageFix{{LineOffset: 40, ObservedAt: lineTime, Model: "gpt-6-luna"}}); err != nil {
		t.Fatal(err)
	}
	var observed, model string
	if err := db.QueryRowContext(ctx, `SELECT observed_at, model FROM transcript_usage WHERE file_key = '2:2'`).Scan(&observed, &model); err != nil {
		t.Fatal(err)
	}
	if observed != lineTime.Format(time.RFC3339Nano) || model != "gpt-6-luna" {
		t.Fatalf("repaired row = %s %s", observed, model)
	}
	if files, _ := repo.UsageFilesToRepair(ctx); len(files) != 0 {
		t.Fatalf("a repaired file is listed again: %+v", files)
	}
}

func TestUsageRepairCompletionMarkerIsDurable(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	ctx := context.Background()
	repo := NewTranscriptIndexRepository(db)
	if done, err := repo.UsageRepairDone(ctx, "r1"); err != nil || done {
		t.Fatalf("fresh database: done=%v err=%v", done, err)
	}
	if err := repo.MarkUsageRepairDone(ctx, "r1", transcriptindex.RepairReport{Files: 2, Rows: 9}); err != nil {
		t.Fatal(err)
	}
	if err := repo.MarkUsageRepairDone(ctx, "r1", transcriptindex.RepairReport{}); err != nil {
		t.Fatalf("marking twice must be idempotent: %v", err)
	}
	var rows int
	if err := db.GetContext(ctx, &rows, `SELECT rows_repaired FROM transcript_usage_repairs WHERE name = 'r1'`); err != nil || rows != 9 {
		t.Fatalf("marker rows = %d, %v", rows, err)
	}
	if done, _ := repo.UsageRepairDone(ctx, "r1"); !done {
		t.Fatal("marker not read back")
	}
}

// F-W5-4: recommendations read only episodes of the current classifier, so an
// episode written by an older detector (a handoff continuation that counted
// every call of the run) is never priced beside current ones.
func TestUsageEpisodesReadOnlyTheCurrentClassifier(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	ctx := context.Background()
	at := time.Date(2026, 10, 29, 9, 0, 0, 0, time.UTC)
	for _, row := range []struct{ id, version string }{{"current", runsignal.EpisodeClassifierVersion}, {"stale", "friction-episode.v9"}} {
		if _, err := db.ExecContext(ctx, `INSERT INTO invocation_read_model_episodes (run_id, episode_id, occurred_at, time_basis, classifier_version, pattern, cause_scope, severity,
			honesty_flags_json, start_event_id, end_event_id, evidence_event_ids_json, turns, tokens, wall_clock_ms, owner_confidence, failed_joined_calls, fingerprint)
			VALUES ('run', ?, ?, 'event', ?, 'handoff-continuation', 'run-execution', 'one-off', '[]', 'a', 'b', '[]', 3, 0, 0, 'unknown', 0, 'fp')`,
			row.id, at.Format(time.RFC3339Nano), row.version); err != nil {
			t.Fatal(err)
		}
	}
	episodes, err := NewUsageAttributionRepository(db).UsageEpisodes(ctx, at.Add(-time.Hour), at.Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if len(episodes) != 1 || episodes[0].EpisodeID != "current" {
		t.Fatalf("episodes = %+v, want only the current classifier's", episodes)
	}
}
