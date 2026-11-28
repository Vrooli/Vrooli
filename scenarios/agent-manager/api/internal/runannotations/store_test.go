package runannotations

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	_ "modernc.org/sqlite"
)

func testStore(t *testing.T) (*Store, *sqlx.DB) {
	t.Helper()
	db, err := sqlx.Connect("sqlite", "file:"+filepath.Join(t.TempDir(), "annotations.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	store := NewStore(db)
	if err := store.EnsureSchema(context.Background()); err != nil {
		t.Fatal(err)
	}
	return store, db
}

func annotation(scope, label, hash string, at time.Time, probability *float64) Annotation {
	return Annotation{
		ScopeKind: ScopeRun, ScopeID: scope, LabelID: label, DefinitionHash: hash,
		Model: "model-a", ModelVersion: "role-a", Method: "decision",
		Value:       Value{Kind: "noul", Noul: probability},
		Probability: probability, ObservedAt: at, State: StateObserved,
		EvidenceEventIDs: []string{"event-1"},
	}
}

func TestEnsureSchemaIsIdempotent(t *testing.T) {
	store, _ := testStore(t)
	ctx := context.Background()
	for range 3 {
		if err := store.EnsureSchema(ctx); err != nil {
			t.Fatalf("reapply: %v", err)
		}
	}
}

func TestSaveIsIdempotentAndKeepsProbabilityNullable(t *testing.T) {
	store, _ := testStore(t)
	ctx := context.Background()
	at := time.Date(2026, 11, 18, 12, 0, 0, 0, time.UTC)
	row := annotation("run-1", "agent.stalled", "hash-1", at, floatPointer(0.72))
	for range 3 {
		if err := store.Save(ctx, []Annotation{row}); err != nil {
			t.Fatal(err)
		}
	}
	count, err := store.Count(ctx)
	if err != nil || count != 1 {
		t.Fatalf("count = %d %v, want 1", count, err)
	}
	latest, err := store.Current(ctx, ScopeRun, "run-1", "agent.stalled", "hash-1")
	if err != nil || latest == nil {
		t.Fatalf("current = %v %v", latest, err)
	}
	if latest.Probability == nil || *latest.Probability != 0.72 {
		t.Fatalf("probability = %v", latest.Probability)
	}
	if len(latest.EvidenceEventIDs) != 1 || latest.EvidenceEventIDs[0] != "event-1" {
		t.Fatalf("evidence = %v", latest.EvidenceEventIDs)
	}

	noProbability := annotation("run-2", "run.type", "hash-2", at, nil)
	noProbability.Method = "deterministic"
	if err := store.Save(ctx, []Annotation{noProbability}); err != nil {
		t.Fatal(err)
	}
	stored, err := store.Current(ctx, ScopeRun, "run-2", "run.type", "hash-2")
	if err != nil || stored == nil {
		t.Fatalf("stored = %v %v", stored, err)
	}
	if stored.Probability != nil {
		t.Fatalf("uncalibrated probability stored as %v, want nil", *stored.Probability)
	}
}

func TestSaveRefusesToMoveAnObservationBackwards(t *testing.T) {
	store, _ := testStore(t)
	ctx := context.Background()
	newer := time.Date(2026, 11, 18, 12, 0, 0, 0, time.UTC)
	older := newer.Add(-time.Hour)
	if err := store.Save(ctx, []Annotation{annotation("run-1", "agent.stalled", "hash-1", newer, floatPointer(0.8))}); err != nil {
		t.Fatal(err)
	}
	if err := store.Save(ctx, []Annotation{annotation("run-1", "agent.stalled", "hash-1", older, floatPointer(0.1))}); err != nil {
		t.Fatal(err)
	}
	latest, _ := store.Current(ctx, ScopeRun, "run-1", "agent.stalled", "hash-1")
	if latest.Probability == nil || *latest.Probability != 0.8 {
		t.Fatalf("older replay overwrote newer observation: %v", latest.Probability)
	}
}

func TestCompatibleLookupUsesOnlyDeclaredHashes(t *testing.T) {
	store, _ := testStore(t)
	ctx := context.Background()
	at := time.Date(2026, 11, 18, 12, 0, 0, 0, time.UTC)
	if err := store.Save(ctx, []Annotation{annotation("run-1", "operator.valence", "old-hash", at, floatPointer(0.4))}); err != nil {
		t.Fatal(err)
	}
	found, err := store.Compatible(ctx, ScopeRun, "run-1", "operator.valence", []string{"old-hash"})
	if err != nil || found == nil || found.DefinitionHash != "old-hash" {
		t.Fatalf("compatible = %v %v", found, err)
	}
	notFound, err := store.Compatible(ctx, ScopeRun, "run-1", "operator.valence", []string{"other-hash"})
	if err != nil || notFound != nil {
		t.Fatalf("undeclared hash reused: %v %v", notFound, err)
	}
}

func TestLatestForScopePicksNewestPerLabel(t *testing.T) {
	store, _ := testStore(t)
	ctx := context.Background()
	base := time.Date(2026, 11, 18, 12, 0, 0, 0, time.UTC)
	rows := []Annotation{
		annotation("run-1", "a", "old", base, floatPointer(0.1)),
		annotation("run-1", "a", "new", base.Add(time.Minute), floatPointer(0.9)),
		annotation("run-1", "b", "b1", base, floatPointer(0.5)),
	}
	if err := store.Save(ctx, rows); err != nil {
		t.Fatal(err)
	}
	latest, err := store.LatestForScope(ctx, ScopeRun, "run-1")
	if err != nil {
		t.Fatal(err)
	}
	if len(latest) != 2 {
		t.Fatalf("labels = %d, want 2", len(latest))
	}
	for _, row := range latest {
		if row.LabelID == "a" && (row.DefinitionHash != "new" || *row.Probability != 0.9) {
			t.Fatalf("latest a = %+v", row)
		}
	}
}

func TestStoreOrderingSurvivesSameSecondFractions(t *testing.T) {
	store, _ := testStore(t)
	ctx := context.Background()
	base := time.Date(2026, 11, 18, 12, 0, 0, 0, time.UTC)
	rows := []Annotation{
		annotation("run-1", "a", "whole", base, floatPointer(0.1)),
		annotation("run-1", "a", "fraction", base.Add(500*time.Millisecond), floatPointer(0.9)),
	}
	if err := store.Save(ctx, rows); err != nil {
		t.Fatal(err)
	}
	latest, err := store.LatestForScope(ctx, ScopeRun, "run-1")
	if err != nil {
		t.Fatal(err)
	}
	if len(latest) != 1 || latest[0].DefinitionHash != "fraction" {
		t.Fatalf("latest = %+v, want the fractional timestamp", latest)
	}
}

// latestRows collects ForEachLatest for a window.
func latestRows(t *testing.T, store *Store, since, until time.Time) []Annotation {
	t.Helper()
	var rows []Annotation
	err := store.ForEachLatest(context.Background(), since, until, nil, func(row Annotation) error {
		rows = append(rows, row)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return rows
}

// latestRow is the newest row of one label for one run, under any definition.
func latestRow(store *Store, runID, labelID string) (*Annotation, error) {
	rows, err := store.LatestForScope(context.Background(), ScopeRun, runID)
	if err != nil {
		return nil, err
	}
	for index := range rows {
		if rows[index].LabelID == labelID {
			return &rows[index], nil
		}
	}
	return nil, nil
}

func TestForEachLatestReadsOneRowPerScopeAndLabelInTheWindow(t *testing.T) {
	store, _ := testStore(t)
	ctx := context.Background()
	base := time.Date(2026, 11, 18, 12, 0, 0, 0, time.UTC)
	rows := []Annotation{
		annotation("run-1", "a", "h1", base, floatPointer(0.7)),
		annotation("run-2", "a", "h1", base, nil),
		annotation("run-3", "a", "h1", base, nil),
		annotation("run-1", "a", "h1", base.Add(time.Minute), floatPointer(0.9)),
	}
	rows[1].State = StateUnavailable
	rows[2].State = StateCompatible
	if err := store.Save(ctx, rows); err != nil {
		t.Fatal(err)
	}
	latest := latestRows(t, store, time.Time{}, time.Time{})
	counts := StateCounts{}
	for _, row := range latest {
		addCount(&counts, row.State)
		if row.ScopeID == "run-1" && (row.Probability == nil || *row.Probability != 0.9) {
			t.Fatalf("run-1 latest = %+v, want the newer observation", row)
		}
	}
	if counts.Observed != 1 || counts.Compatible != 1 || counts.Unavailable != 1 || counts.NotObserved != 0 {
		t.Fatalf("counts = %+v", counts)
	}
	if future := latestRows(t, store, base.Add(time.Hour), time.Time{}); len(future) != 0 {
		t.Fatalf("future window = %+v", future)
	}
}

func TestObservedScopesKeepsOnlyTheCurrentDefinition(t *testing.T) {
	store, _ := testStore(t)
	ctx := context.Background()
	now := time.Now().UTC()
	confidence := 0.9
	current := annotation("run-current", "agent.stalled", "current-hash", now, &confidence)
	stale := annotation("run-stale", "agent.stalled", "old-hash", now, &confidence)
	unavailable := annotation("run-unavailable", "agent.stalled", "current-hash", now, &confidence)
	unavailable.State = StateUnavailable
	if err := store.Save(ctx, []Annotation{current, stale, unavailable}); err != nil {
		t.Fatal(err)
	}
	scopes, err := store.ObservedScopes(ctx, ScopeRun, now.Add(-time.Hour), now.Add(time.Hour), "agent.stalled", "current-hash", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(scopes) != 1 || scopes[0].ScopeID != "run-current" || scopes[0].RunID != "run-current" {
		t.Fatalf("observed scopes = %v, want only the current observed run", scopes)
	}
}

func TestRunCoverageCountsOnlyTheCurrentDefinitionAsObserved(t *testing.T) {
	store, _ := testStore(t)
	ctx := context.Background()
	base := time.Date(2026, 11, 18, 12, 0, 0, 0, time.UTC)
	rows := []Annotation{
		annotation("run-1", "a", "current", base, floatPointer(0.9)),
		annotation("run-2", "a", "old", base, floatPointer(0.9)),
		annotation("run-3", "a", "old", base, floatPointer(0.9)),
	}
	if err := store.Save(ctx, rows); err != nil {
		t.Fatal(err)
	}
	eligible, observed, err := store.RunCoverageFor(ctx, time.Time{}, time.Time{}, map[string]string{"a": "current"})
	if err != nil {
		t.Fatal(err)
	}
	if eligible != 3 || observed != 1 {
		t.Fatalf("run coverage = %d/%d, want 1 of 3 observed for the current hash", observed, eligible)
	}
}

func TestTrialsAreAnotherTableAndNeverMixWithProduction(t *testing.T) {
	store, _ := testStore(t)
	ctx := context.Background()
	at := time.Date(2026, 11, 18, 12, 0, 0, 0, time.UTC)
	production := annotation("run-1", "a", "h1", at, floatPointer(0.9))
	trial := annotation("run-1", "a", "h1", at.Add(time.Minute), floatPointer(0.2))
	trial.Model, trial.ModelVersion = "candidate", "decide.local.candidate"
	if err := store.Save(ctx, []Annotation{production}); err != nil {
		t.Fatal(err)
	}
	if err := store.Trials().Save(ctx, []Annotation{trial}); err != nil {
		t.Fatal(err)
	}
	current, err := store.Current(ctx, ScopeRun, "run-1", "a", "h1")
	if err != nil || current == nil || current.Model != "model-a" {
		t.Fatalf("production read = %+v (%v), want its own row untouched by the trial", current, err)
	}
	trialRow, err := store.Trials().CurrentFor(ctx, ScopeRun, "run-1", "a", "h1", "decide.local.candidate")
	if err != nil || trialRow == nil || trialRow.Model != "candidate" {
		t.Fatalf("trial read = %+v (%v)", trialRow, err)
	}
	if production, _ := store.Count(ctx); production != 1 {
		t.Fatalf("production rows = %d, want 1", production)
	}
	if latest := latestRows(t, store, time.Time{}, time.Time{}); len(latest) != 1 || latest[0].Model != "model-a" {
		t.Fatalf("production coverage rows = %+v, want only the production row", latest)
	}
}

func TestMessageRowsCarryTheirRunAndTimestamp(t *testing.T) {
	store, _ := testStore(t)
	ctx := context.Background()
	at := time.Date(2026, 11, 18, 12, 0, 0, 0, time.UTC)
	older := annotation("msg-1", "mood", "h1", at, floatPointer(0.4))
	older.ScopeKind, older.RunID, older.OccurredAt = ScopeMessage, "run-1", at.Add(-2*time.Hour)
	newer := annotation("msg-2", "mood", "h1", at, floatPointer(0.6))
	newer.ScopeKind, newer.RunID, newer.OccurredAt = ScopeMessage, "run-1", at.Add(-time.Hour)
	if err := store.Save(ctx, []Annotation{newer, older}); err != nil {
		t.Fatal(err)
	}
	rows, err := store.LatestMessagesForRun(ctx, "run-1")
	if err != nil || len(rows) != 2 {
		t.Fatalf("rows = %+v (%v)", rows, err)
	}
	if rows[0].ScopeID != "msg-1" || !rows[0].OccurredAt.Equal(at.Add(-2*time.Hour)) {
		t.Fatalf("messages are not oldest first with their timestamps: %+v", rows)
	}
	series, err := store.MessageObservations(ctx, "mood", "h1", at.Add(-90*time.Minute), time.Time{})
	if err != nil || len(series) != 1 || series[0].ScopeID != "msg-2" {
		t.Fatalf("series window = %+v (%v), want only the message written inside it", series, err)
	}
}

func TestEnsureSchemaAddsEvidenceHashToAnOlderTable(t *testing.T) {
	db, err := sqlx.Connect("sqlite", "file:"+filepath.Join(t.TempDir(), "legacy.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if _, err := db.Exec(`CREATE TABLE run_annotations (
    scope_kind TEXT NOT NULL, scope_id TEXT NOT NULL, label_id TEXT NOT NULL,
    definition_hash TEXT NOT NULL, model TEXT NOT NULL DEFAULT '', model_version TEXT NOT NULL DEFAULT '',
    value_json TEXT NOT NULL DEFAULT '{}', probability REAL, confidence REAL,
    method TEXT NOT NULL, provider TEXT NOT NULL DEFAULT '',
    evidence_event_ids_json TEXT NOT NULL DEFAULT '[]', observed_at TEXT NOT NULL,
    state TEXT NOT NULL, reason TEXT NOT NULL DEFAULT '',
    PRIMARY KEY (scope_kind, scope_id, label_id, definition_hash, model, model_version))`); err != nil {
		t.Fatal(err)
	}
	store := NewStore(db)
	if err := store.EnsureSchema(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := store.EnsureSchema(context.Background()); err != nil {
		t.Fatalf("a second ensure must be idempotent: %v", err)
	}
	annotation := annotation("run-1", "a", "h1", time.Now().UTC(), floatPointer(0.7))
	annotation.EvidenceHash = "fingerprint"
	if err := store.Save(context.Background(), []Annotation{annotation}); err != nil {
		t.Fatal(err)
	}
	current, err := store.Current(context.Background(), ScopeRun, "run-1", "a", "h1")
	if err != nil {
		t.Fatal(err)
	}
	if current == nil || current.EvidenceHash != "fingerprint" {
		t.Fatalf("current = %+v, want the stored fingerprint", current)
	}
}

// A gateway deadline or an exhausted budget writes an unavailable row after an
// answer already stands. Every read of "what is known about this subject" must
// keep the answer, or coverage and the mood card report a message as lost that
// was answered.
func TestATransientUnavailableRowNeverShadowsAnAnswerOfTheSameDefinition(t *testing.T) {
	store, _ := testStore(t)
	ctx := context.Background()
	at := time.Date(2026, 11, 18, 12, 0, 0, 0, time.UTC)
	message := func(hash string, state State, when time.Time) Annotation {
		row := annotation("msg-1", "m.label", hash, when, floatPointer(0.4))
		row.ScopeKind, row.RunID, row.OccurredAt, row.State = ScopeMessage, "run-1", at, state
		if state == StateUnavailable {
			row.Model, row.ModelVersion, row.Probability, row.Value, row.Reason = "", "", nil, Value{}, "decision_unavailable: deadline"
		}
		return row
	}
	if err := store.Save(ctx, []Annotation{
		message("hash-1", StateObserved, at),
		message("hash-1", StateUnavailable, at.Add(time.Hour)),
	}); err != nil {
		t.Fatal(err)
	}
	stateOf := func(rows []Annotation) State {
		t.Helper()
		if len(rows) != 1 {
			t.Fatalf("rows = %d, want one per label", len(rows))
		}
		return rows[0].State
	}
	forScope, err := store.LatestForScope(ctx, ScopeMessage, "msg-1")
	if err != nil || stateOf(forScope) != StateObserved {
		t.Fatalf("LatestForScope = %v %v, want the answer", forScope, err)
	}
	forRun, err := store.LatestMessagesForRun(ctx, "run-1")
	if err != nil || stateOf(forRun) != StateObserved {
		t.Fatalf("LatestMessagesForRun = %v %v, want the answer", forRun, err)
	}
	var streamed []Annotation
	if err := store.ForEachLatest(ctx, time.Time{}, time.Time{}, func(string) bool { return false }, func(row Annotation) error {
		streamed = append(streamed, row)
		return nil
	}); err != nil || stateOf(streamed) != StateObserved {
		t.Fatalf("ForEachLatest = %v %v, want the answer", streamed, err)
	}
	scopes, err := store.ObservedScopes(ctx, ScopeMessage, time.Time{}, time.Time{}, "m.label", "hash-1", 10)
	if err != nil || len(scopes) != 1 {
		t.Fatalf("ObservedScopes = %v %v, want the answered message", scopes, err)
	}

	// A newer definition that could not be answered is a different matter: the
	// old answer belongs to a definition nobody asks any more.
	if err := store.Save(ctx, []Annotation{message("hash-2", StateUnavailable, at.Add(2*time.Hour))}); err != nil {
		t.Fatal(err)
	}
	forScope, err = store.LatestForScope(ctx, ScopeMessage, "msg-1")
	if err != nil || stateOf(forScope) != StateUnavailable || forScope[0].DefinitionHash != "hash-2" {
		t.Fatalf("LatestForScope after a newer definition = %v %v, want its unavailable row", forScope, err)
	}
}
