package database

import (
	"context"
	"database/sql"
	"fmt"
	"slices"
	"sync"
	"testing"
	"time"

	"agent-manager/internal/adapters/event"
	"agent-manager/internal/domain"
	"agent-manager/internal/pricing"
	"agent-manager/internal/transcriptindex"

	"github.com/google/uuid"
)

func TestConcurrentTranscriptCaptureClaimsOneExactIntent(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	repos := NewRepositories(db, db.log)
	ctx := context.Background()
	task := &domain.Task{ID: uuid.New(), Title: "capture", ScopePath: "."}
	if err := repos.Tasks.Create(ctx, task); err != nil {
		t.Fatal(err)
	}
	run := &domain.Run{ID: uuid.New(), TaskID: task.ID, Tag: uuid.NewString(), Status: domain.RunStatusRunning, Phase: domain.RunPhaseExecuting, ExecutionMode: domain.ExecutionModeImported}
	if err := repos.Runs.Create(ctx, run); err != nil {
		t.Fatal(err)
	}
	file := transcriptindex.FileRow{Key: "capture", Path: "/capture", RunID: run.ID.String(), Owner: transcriptindex.OwnerIndexer, Status: transcriptindex.StatusIndexed, Offset: 3, FirstSeenAt: time.Now()}
	if err := repos.TranscriptIndex.SaveTranscriptFile(ctx, file, nil); err != nil {
		t.Fatal(err)
	}
	rows, err := repos.TranscriptIndex.ListTranscriptFiles(ctx)
	if err != nil || len(rows) != 1 {
		t.Fatalf("fixture: %+v %v", rows, err)
	}
	expected := rows[0]
	called := false
	evicted := expected
	evicted.RecallEvicted = true
	if err := repos.TranscriptIndex.CommitRecallEviction(ctx, expected, evicted, func(*sql.Tx) error { called = true; return nil }); err == nil || called {
		t.Fatalf("eviction admitted an active run: err=%v callback=%v", err, called)
	}
	start := make(chan struct{})
	results := make(chan error, 2)
	var ready sync.WaitGroup
	ready.Add(2)
	for _, token := range []string{"intent-a", "intent-b"} {
		go func(token string) {
			intent := expected
			intent.Status, intent.Error = transcriptindex.StatusPending, token
			ready.Done()
			<-start
			results <- repos.TranscriptIndex.BeginTranscriptCapture(ctx, &expected, intent)
		}(token)
	}
	ready.Wait()
	close(start)
	first, second := <-results, <-results
	if (first == nil) == (second == nil) {
		t.Fatalf("expected exactly one capture owner: %v %v", first, second)
	}
	rows, err = repos.TranscriptIndex.ListTranscriptFiles(ctx)
	if err != nil || len(rows) != 1 || rows[0].Status != transcriptindex.StatusPending {
		t.Fatalf("missing owner intent: %+v %v", rows, err)
	}
	winner := rows[0].Error
	called = false
	if err := repos.TranscriptIndex.CommitRecallEviction(ctx, expected, evicted, func(*sql.Tx) error { called = true; return nil }); err == nil || called {
		t.Fatalf("eviction crossed active capture: err=%v callback=%v", err, called)
	}
	fileToken, err := repos.TranscriptIndex.TranscriptReplacementToken(ctx, file.Key)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := repos.TranscriptIndex.MarkReplacementIntent(ctx, file.Key, fileToken); err == nil {
		t.Fatal("replacement stole an active capture intent")
	}
	if err := repos.TranscriptIndex.FailTranscriptCapture(ctx, file.Key, "not-"+winner, transcriptindex.StatusUnreadable, "stale failure"); err == nil {
		t.Fatal("stale owner marked failure")
	}
	rows, err = repos.TranscriptIndex.ListTranscriptFiles(ctx)
	if err != nil || len(rows) != 1 || rows[0].Error != winner {
		t.Fatalf("stale failure replaced current intent: %+v %v", rows, err)
	}
	file.Offset = 9
	if err := repos.TranscriptIndex.CommitTranscriptAndRun(ctx, file, nil, nil, "not-"+winner, func(*domain.Run) error { return nil }); err == nil {
		t.Fatal("wrong intent committed")
	}
	if err := repos.TranscriptIndex.CommitTranscriptAndRun(ctx, file, nil, nil, winner, func(*domain.Run) error { return nil }); err != nil {
		t.Fatalf("winning intent failed: %v", err)
	}
	rows, err = repos.TranscriptIndex.ListTranscriptFiles(ctx)
	if err != nil || len(rows) != 1 || rows[0].Offset != 9 || rows[0].Error != "" {
		t.Fatalf("winning capture not committed: %+v %v", rows, err)
	}
}

func TestConcurrentLegacyRelinkAdmissionHasOneOwner(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	repos := NewRepositories(db, db.log)
	ctx := context.Background()
	task := &domain.Task{ID: uuid.New(), Title: "import", ScopePath: "."}
	if err := repos.Tasks.Create(ctx, task); err != nil {
		t.Fatal(err)
	}
	run := &domain.Run{ID: uuid.New(), TaskID: task.ID, Tag: uuid.NewString(), Status: domain.RunStatusComplete, Phase: domain.RunPhaseCompleted, ExecutionMode: domain.ExecutionModeImported, ImportSourceHarness: "codex", ImportSourceSessionID: "session", TranscriptPath: "/private/transcript.ndjson"}
	if err := repos.Runs.Create(ctx, run); err != nil {
		t.Fatal(err)
	}
	results := make(chan error, 2)
	for _, key := range []string{"source-a", "source-b"} {
		file := transcriptindex.FileRow{Key: key, Path: "/" + key, RunID: run.ID.String(), Harness: "codex", SessionID: "session", Owner: transcriptindex.OwnerIndexer, Status: transcriptindex.StatusUnreadable, Error: transcriptindex.ReplacementReconciliationRequired + "legacy relink", FirstSeenAt: time.Now()}
		go func() { results <- repos.TranscriptIndex.AdmitLegacyRelink(ctx, nil, file, run.TranscriptPath) }()
	}
	wins := 0
	for range 2 {
		if <-results == nil {
			wins++
		}
	}
	if wins != 1 {
		t.Fatalf("legacy admission winners = %d, want 1", wins)
	}
	rows, err := repos.TranscriptIndex.ListTranscriptFiles(ctx)
	if err != nil || len(rows) != 1 || rows[0].RunID != run.ID.String() {
		t.Fatalf("legacy checkpoints: %+v %v", rows, err)
	}
}

func TestLegacyCleanupObligationRollsBackWithPromotion(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	repos := NewRepositories(db, db.log)
	ctx := context.Background()
	task := &domain.Task{ID: uuid.New(), Title: "import", ScopePath: "."}
	if err := repos.Tasks.Create(ctx, task); err != nil {
		t.Fatal(err)
	}
	run := &domain.Run{ID: uuid.New(), TaskID: task.ID, Tag: uuid.NewString(), Status: domain.RunStatusComplete, Phase: domain.RunPhaseCompleted, ExecutionMode: domain.ExecutionModeImported, ImportSourceHarness: "codex", ImportSourceSessionID: "session", TranscriptPath: "/private/transcript.ndjson"}
	if err := repos.Runs.Create(ctx, run); err != nil {
		t.Fatal(err)
	}
	file := transcriptindex.FileRow{Key: "source", Path: "/source", RunID: run.ID.String(), Harness: "codex", SessionID: "session", Owner: transcriptindex.OwnerIndexer, Status: transcriptindex.StatusUnreadable, Error: transcriptindex.ReplacementReconciliationRequired + "legacy relink", FirstSeenAt: time.Now()}
	if err := repos.TranscriptIndex.AdmitLegacyRelink(ctx, nil, file, run.TranscriptPath); err != nil {
		t.Fatal(err)
	}
	token, err := repos.TranscriptIndex.TranscriptReplacementToken(ctx, file.Key)
	if err != nil {
		t.Fatal(err)
	}
	file.Status, file.Error = transcriptindex.StatusPending, transcriptindex.ReplacementEffectsPending
	err = repos.TranscriptIndex.CommitTranscriptReplacement(ctx, file.Key, token, file, nil, func(current *domain.Run) error { current.TranscriptPath = file.Path; return nil }, func(tx *sql.Tx) error {
		if err := repos.TranscriptIndex.RecordLegacyCleanupInTx(ctx, tx, file, run.TranscriptPath); err != nil {
			return err
		}
		return fmt.Errorf("injected failure after cleanup obligation")
	})
	if err == nil {
		t.Fatal("expected promotion failure")
	}
	stored, err := repos.Runs.Get(ctx, run.ID)
	if err != nil || stored.TranscriptPath != run.TranscriptPath {
		t.Fatalf("run escaped rollback: %+v %v", stored, err)
	}
	issue, err := repos.TranscriptIndex.LegacyCleanupIssue(ctx, run.ID.String())
	if err != nil || issue != nil {
		t.Fatalf("cleanup escaped rollback: %+v %v", issue, err)
	}
	rows, err := repos.TranscriptIndex.ListTranscriptFiles(ctx)
	if err != nil || len(rows) != 1 || rows[0].Status != transcriptindex.StatusUnreadable {
		t.Fatalf("checkpoint escaped rollback: %+v %v", rows, err)
	}
	tx, err := db.BeginTxx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := repos.TranscriptIndex.RecordLegacyCleanupInTx(ctx, tx.Tx, file, run.TranscriptPath); err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `DELETE FROM runs WHERE id = ?`, run.ID.String()); err != nil {
		t.Fatal(err)
	}
	issue, err = repos.TranscriptIndex.LegacyCleanupIssue(ctx, run.ID.String())
	if err != nil || issue == nil || issue.Path != run.TranscriptPath {
		t.Fatalf("cleanup obligation lost when run was deleted: %+v %v", issue, err)
	}
}

func TestTranscriptReplacementRollsBackEventsRunAndUsage(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	repos := NewRepositories(db, db.log)
	events := event.NewSQLiteStore(db, db.log)
	ctx := context.Background()
	task := &domain.Task{ID: uuid.New(), Title: "import", ScopePath: "src/", CreatedAt: time.Now(), UpdatedAt: time.Now()}
	if err := repos.Tasks.Create(ctx, task); err != nil {
		t.Fatal(err)
	}
	run := &domain.Run{ID: uuid.New(), TaskID: task.ID, Tag: uuid.NewString(), Status: domain.RunStatusRunning, Phase: domain.RunPhaseExecuting, ExecutionMode: domain.ExecutionModeImported}
	if err := repos.Runs.Create(ctx, run); err != nil {
		t.Fatal(err)
	}
	file := transcriptindex.FileRow{Key: "old", Path: "/import", RunID: run.ID.String(), Owner: transcriptindex.OwnerIndexer, Status: transcriptindex.StatusIndexed, Offset: 3, FirstSeenAt: time.Now()}
	if err := repos.TranscriptIndex.SaveTranscriptFile(ctx, file, nil); err != nil {
		t.Fatal(err)
	}
	old := domain.NewLogEvent(run.ID, "info", "old")
	if err := events.Append(ctx, run.ID, old); err != nil {
		t.Fatal(err)
	}
	fileToken, err := repos.TranscriptIndex.TranscriptReplacementToken(ctx, file.Key)
	if err != nil {
		t.Fatal(err)
	}
	usage := transcriptindex.UsageRow{FileKey: file.Key, LineOffset: 1, RunID: run.ID.String(), ObservedAt: time.Now()}
	if err := repos.TranscriptIndex.SaveTranscriptFile(ctx, file, []transcriptindex.UsageRow{usage}); err != nil {
		t.Fatal(err)
	}
	applyRun := func(current *domain.Run) error {
		current.Status, current.Phase = domain.RunStatusComplete, domain.RunPhaseCompleted
		if current.LabelSource != domain.RunLabelSourceManual {
			current.Label = "from transcript"
		}
		return nil
	}
	called := false
	if err := repos.TranscriptIndex.CommitTranscriptReplacement(ctx, file.Key, fileToken, file, nil, applyRun, func(*sql.Tx) error { called = true; return nil }); err == nil || called {
		t.Fatalf("changed usage generation was accepted: err=%v event callback=%v", err, called)
	}
	fileToken, err = repos.TranscriptIndex.TranscriptReplacementToken(ctx, file.Key)
	if err != nil {
		t.Fatal(err)
	}
	eventToken, err := events.StreamToken(ctx, run.ID)
	if err != nil {
		t.Fatal(err)
	}
	if err := repos.Runs.(*runRepository).UpdateRunLabel(ctx, run.ID, "manual", domain.RunLabelSourceManual); err != nil {
		t.Fatal(err)
	}
	file.Key, file.Offset = "new", 9
	if _, err := db.Exec(`CREATE TRIGGER reject_replacement BEFORE INSERT ON transcript_files BEGIN SELECT RAISE(ABORT, 'checkpoint failure'); END`); err != nil {
		t.Fatal(err)
	}
	guard := events.LockReplacement()
	quotaRepo := NewPricingRepository(db, db.log)
	quota := quotaRepo.(interface {
		RecordQuotaObservationInTx(context.Context, *sql.Tx, *pricing.QuotaObservation) error
	})
	quotaStore := quotaRepo.(pricing.QuotaObservationRepository)
	base := time.Now().UTC()
	oldPercent, otherPercent, newPercent := 10.0, 20.0, 30.0
	oldQuota := &pricing.QuotaObservation{Provider: "openai", AccountScope: "scope", Pool: "primary", Window: "300m", ObservedAt: base.Add(-2 * time.Hour), Standing: pricing.QuotaStandingAvailable, UsedPercent: &oldPercent, Provenance: "test", SourceRunID: run.ID.String()}
	otherQuota := &pricing.QuotaObservation{Provider: "openai", AccountScope: "scope", Pool: "primary", Window: "300m", ObservedAt: base.Add(-time.Hour), Standing: pricing.QuotaStandingAvailable, UsedPercent: &otherPercent, Provenance: "test", SourceRunID: uuid.NewString()}
	for _, item := range []*pricing.QuotaObservation{oldQuota, otherQuota} {
		if err := quotaStore.RecordQuotaObservation(ctx, item); err != nil {
			guard.Finish(run.ID, false)
			t.Fatal(err)
		}
	}
	observation := &pricing.QuotaObservation{Provider: "openai", AccountScope: "scope", Pool: "primary", Window: "300m", ObservedAt: base, Standing: pricing.QuotaStandingAvailable, UsedPercent: &newPercent, Provenance: "test", SourceRunID: run.ID.String()}
	apply := func(tx *sql.Tx) error {
		if err := guard.ReplaceInTx(ctx, tx, run.ID, eventToken, []*domain.RunEvent{domain.NewLogEvent(run.ID, "info", "new")}); err != nil {
			return err
		}
		return quota.RecordQuotaObservationInTx(ctx, tx, observation)
	}
	err = repos.TranscriptIndex.CommitTranscriptReplacement(ctx, "old", fileToken, file, nil, applyRun, func(tx *sql.Tx) error {
		return apply(tx)
	})
	guard.Finish(run.ID, false)
	if err == nil {
		t.Fatal("expected checkpoint failure")
	}
	stored, err := repos.Runs.Get(ctx, run.ID)
	if err != nil || stored.Status != domain.RunStatusRunning {
		t.Fatalf("run escaped rollback: %+v %v", stored, err)
	}
	rows, err := repos.TranscriptIndex.ListTranscriptFiles(ctx)
	if err != nil || len(rows) != 1 || rows[0].Key != "old" || rows[0].Offset != 3 {
		t.Fatalf("checkpoint escaped rollback: %+v %v", rows, err)
	}
	storedUsage, err := repos.TranscriptIndex.TranscriptUsageForRun(ctx, run.ID.String())
	if err != nil || len(storedUsage) != 1 {
		t.Fatalf("usage escaped rollback: %+v %v", storedUsage, err)
	}
	got, err := events.Get(ctx, run.ID, event.GetOptions{AfterSequence: -1})
	if err != nil || len(got) != 1 || got[0].ID != old.ID {
		t.Fatalf("events escaped rollback: %+v %v", got, err)
	}
	var quotaCount int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM quota_observations`).Scan(&quotaCount); err != nil || quotaCount != 2 {
		t.Fatalf("quota escaped rollback: %d %v", quotaCount, err)
	}
	if _, err := db.Exec(`DROP TRIGGER reject_replacement`); err != nil {
		t.Fatal(err)
	}
	guard = events.LockReplacement()
	err = repos.TranscriptIndex.CommitTranscriptReplacement(ctx, "old", fileToken, file, nil, applyRun, apply)
	guard.Finish(run.ID, err == nil)
	if err != nil {
		t.Fatalf("successful replacement: %v", err)
	}
	stored, err = repos.Runs.Get(ctx, run.ID)
	if err != nil || stored.Label != "manual" || stored.LabelSource != domain.RunLabelSourceManual {
		t.Fatalf("replacement overwrote manual label: %+v %v", stored, err)
	}
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM quota_observations`).Scan(&quotaCount); err != nil || quotaCount != 3 {
		t.Fatalf("quota not committed: %d %v", quotaCount, err)
	}
	for _, item := range []*pricing.QuotaObservation{oldQuota, otherQuota} {
		var count int
		if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM quota_observations WHERE id = ?`, item.ID).Scan(&count); err != nil || count != 1 {
			t.Fatalf("historic quota observation lost: %s %d %v", item.ID, count, err)
		}
	}
}

func TestCommitTranscriptAndRunRollsBackBothOwners(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	repos := NewRepositories(db, db.log)
	ctx := context.Background()
	task := &domain.Task{ID: uuid.New(), Title: "import", ScopePath: "src/", CreatedAt: time.Now(), UpdatedAt: time.Now()}
	if err := repos.Tasks.Create(ctx, task); err != nil {
		t.Fatal(err)
	}
	run := &domain.Run{ID: uuid.New(), TaskID: task.ID, Tag: uuid.NewString(), Status: domain.RunStatusRunning, Phase: domain.RunPhaseExecuting, ExecutionMode: domain.ExecutionModeImported}
	if err := repos.Runs.Create(ctx, run); err != nil {
		t.Fatal(err)
	}
	file := transcriptindex.FileRow{Key: "atomic", Path: "/import", RunID: run.ID.String(), Owner: transcriptindex.OwnerIndexer, Status: transcriptindex.StatusIndexed, Offset: 3, FirstSeenAt: time.Now()}
	if err := repos.TranscriptIndex.SaveTranscriptFile(ctx, file, nil); err != nil {
		t.Fatal(err)
	}
	applyRun := func(current *domain.Run) error {
		current.Status, current.Phase = domain.RunStatusComplete, domain.RunPhaseCompleted
		return nil
	}
	initialRows, err := repos.TranscriptIndex.ListTranscriptFiles(ctx)
	if err != nil || len(initialRows) != 1 {
		t.Fatalf("initial checkpoint: %+v %v", initialRows, err)
	}
	expected := initialRows[0]
	file.Offset = 9
	if _, err := db.Exec(`CREATE TRIGGER reject_transcript_checkpoint BEFORE UPDATE ON transcript_files BEGIN SELECT RAISE(ABORT, 'checkpoint failure'); END`); err != nil {
		t.Fatal(err)
	}
	if err := repos.TranscriptIndex.CommitTranscriptAndRun(ctx, file, nil, &expected, "", applyRun); err == nil {
		t.Fatal("expected checkpoint failure")
	}
	after, err := repos.Runs.Get(ctx, run.ID)
	if err != nil || after.Status != domain.RunStatusRunning {
		t.Fatalf("run update escaped rollback: %+v %v", after, err)
	}
	rows, err := repos.TranscriptIndex.ListTranscriptFiles(ctx)
	if err != nil || len(rows) != 1 || rows[0].Offset != 3 {
		t.Fatalf("checkpoint escaped rollback: %+v %v", rows, err)
	}
	if _, err := db.Exec(`DROP TRIGGER reject_transcript_checkpoint`); err != nil {
		t.Fatal(err)
	}
	if err := repos.Runs.(*runRepository).UpdateRunLabel(ctx, run.ID, "manual", domain.RunLabelSourceManual); err != nil {
		t.Fatal(err)
	}
	if err := repos.TranscriptIndex.CommitTranscriptAndRun(ctx, file, nil, &expected, "", applyRun); err != nil {
		t.Fatal(err)
	}
	after, err = repos.Runs.Get(ctx, run.ID)
	if err != nil || after.Status != domain.RunStatusComplete || after.Label != "manual" || after.LabelSource != domain.RunLabelSourceManual {
		t.Fatalf("run not committed: %+v %v", after, err)
	}
	rows, err = repos.TranscriptIndex.ListTranscriptFiles(ctx)
	if err != nil || len(rows) != 1 || rows[0].Offset != 9 {
		t.Fatalf("checkpoint not committed: %+v %v", rows, err)
	}
	file.Offset = 12
	expected = rows[0]
	if err := repos.TranscriptIndex.CommitTranscriptAndRun(ctx, file, nil, &expected, "", func(current *domain.Run) error {
		current.LifecycleVersion--
		current.Status, current.Phase = domain.RunStatusRunning, domain.RunPhaseExecuting
		return nil
	}); err == nil {
		t.Fatal("stale run lifecycle update accepted")
	}
	rows, err = repos.TranscriptIndex.ListTranscriptFiles(ctx)
	if err != nil || len(rows) != 1 || rows[0].Offset != 9 {
		t.Fatalf("stale run update advanced checkpoint: %+v %v", rows, err)
	}
}

func TestListByWebConsoleSessionIDsReturnsEveryReferencingRun(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	repos := NewRepositories(db, db.log)
	ctx := context.Background()
	task := &domain.Task{ID: uuid.New(), Title: "web-console lookup", ScopePath: "src/", CreatedAt: time.Now(), UpdatedAt: time.Now()}
	if err := repos.Tasks.Create(ctx, task); err != nil {
		t.Fatal(err)
	}
	ended := time.Date(2026, 10, 1, 9, 0, 0, 0, time.UTC)
	create := func(sessionID string, status domain.RunStatus) *domain.Run {
		t.Helper()
		run := &domain.Run{ID: uuid.New(), TaskID: task.ID, Tag: uuid.NewString(), Status: status, Phase: domain.RunPhaseCompleted, ExecutionMode: domain.ExecutionModeInteractive, WebConsoleSessionID: sessionID, EndedAt: &ended}
		if err := repos.Runs.Create(ctx, run); err != nil {
			t.Fatal(err)
		}
		return run
	}
	first := create("wc-shared", domain.RunStatusComplete)
	second := create("wc-shared", domain.RunStatusFailed)
	single := create("wc-single", domain.RunStatusCancelled)
	create("wc-unrequested", domain.RunStatusComplete)
	create("", domain.RunStatusComplete)

	runs, err := repos.Runs.ListByWebConsoleSessionIDs(ctx, []string{"wc-shared", "wc-single", "wc-single", "wc-missing", ""})
	if err != nil {
		t.Fatal(err)
	}
	got := map[uuid.UUID]*domain.Run{}
	for _, run := range runs {
		got[run.ID] = run
	}
	if len(runs) != 3 || got[first.ID] == nil || got[second.ID] == nil || got[single.ID] == nil {
		t.Fatalf("runs = %d %v, want exactly the two shared-session runs and the single-session run", len(runs), got)
	}
	if r := got[single.ID]; r.WebConsoleSessionID != "wc-single" || r.Status != domain.RunStatusCancelled || r.EndedAt == nil || !r.EndedAt.Equal(ended) {
		t.Fatalf("single-session run = %+v, want session, status, and end time round-tripped", r)
	}

	none, err := repos.Runs.ListByWebConsoleSessionIDs(ctx, nil)
	if err != nil || len(none) != 0 {
		t.Fatalf("empty lookup = %v err=%v, want no runs", none, err)
	}
}

func TestListByWebConsoleSessionIDsSpansLookupChunks(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	repos := NewRepositories(db, db.log)
	ctx := context.Background()
	task := &domain.Task{ID: uuid.New(), Title: "web-console chunked lookup", ScopePath: "src/", CreatedAt: time.Now(), UpdatedAt: time.Now()}
	if err := repos.Tasks.Create(ctx, task); err != nil {
		t.Fatal(err)
	}
	ids := make([]string, webConsoleSessionLookupChunk+3)
	for i := range ids {
		ids[i] = fmt.Sprintf("wc-%04d", i)
	}
	for _, sessionID := range []string{ids[0], ids[webConsoleSessionLookupChunk+2]} {
		run := &domain.Run{ID: uuid.New(), TaskID: task.ID, Tag: uuid.NewString(), Status: domain.RunStatusComplete, Phase: domain.RunPhaseCompleted, WebConsoleSessionID: sessionID}
		if err := repos.Runs.Create(ctx, run); err != nil {
			t.Fatal(err)
		}
	}

	runs, err := repos.Runs.ListByWebConsoleSessionIDs(ctx, ids)
	if err != nil {
		t.Fatal(err)
	}
	var found []string
	for _, run := range runs {
		found = append(found, run.WebConsoleSessionID)
	}
	slices.Sort(found)
	if want := []string{ids[0], ids[webConsoleSessionLookupChunk+2]}; !slices.Equal(found, want) {
		t.Fatalf("sessions found = %v, want runs from both lookup chunks %v", found, want)
	}
}

func TestFindImportProvenancePreservesAmbiguityAndHarnessIdentity(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	repos := NewRepositories(db, db.log)
	ctx := context.Background()
	task := &domain.Task{ID: uuid.New(), Title: "import lookup", ScopePath: "src/", CreatedAt: time.Now(), UpdatedAt: time.Now()}
	if err := repos.Tasks.Create(ctx, task); err != nil {
		t.Fatal(err)
	}
	create := func(harness, session string) uuid.UUID {
		run := &domain.Run{ID: uuid.New(), TaskID: task.ID, Tag: uuid.NewString(), Status: domain.RunStatusComplete, Phase: domain.RunPhaseCompleted, ExecutionMode: domain.ExecutionModeImported, ImportSourceHarness: harness, ImportSourceSessionID: session}
		if err := repos.Runs.Create(ctx, run); err != nil {
			t.Fatal(err)
		}
		return run.ID
	}
	first := create("codex", "same")
	create("claude-code", "same")
	got, err := repos.Runs.FindImportProvenance(ctx, "resource:codex/sessions", "same")
	if err != nil || len(got) != 1 || got[0] != first {
		t.Fatalf("single match = %v, %v", got, err)
	}
	second := create("resource:codex/sessions", "same")
	got, err = repos.Runs.FindImportProvenance(ctx, "codex", "same")
	if err != nil || len(got) != 2 || !slices.Contains(got, first) || !slices.Contains(got, second) {
		t.Fatalf("ambiguous matches = %v, %v", got, err)
	}
	create("resource:codex/alternate", "same")
	got, err = repos.Runs.FindImportProvenance(ctx, "codex", "same")
	if err != nil || len(got) != 2 {
		t.Fatalf("ambiguous matches must be bounded: %v, %v", got, err)
	}
	got, err = repos.Runs.FindImportProvenance(ctx, "codex", "missing")
	if err != nil || len(got) != 0 {
		t.Fatalf("missing match = %v, %v", got, err)
	}
	if _, err = repos.Runs.FindImportProvenance(ctx, "codex", ""); err == nil {
		t.Fatal("empty identity accepted")
	}
}
