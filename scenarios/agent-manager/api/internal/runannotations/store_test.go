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
	row := annotation("run-1", "agent.spinning", "hash-1", at, floatPointer(0.72))
	for range 3 {
		if err := store.Save(ctx, []Annotation{row}); err != nil {
			t.Fatal(err)
		}
	}
	count, err := store.Count(ctx)
	if err != nil || count != 1 {
		t.Fatalf("count = %d %v, want 1", count, err)
	}
	latest, err := store.Current(ctx, ScopeRun, "run-1", "agent.spinning", "hash-1")
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
	if err := store.Save(ctx, []Annotation{annotation("run-1", "agent.spinning", "hash-1", newer, floatPointer(0.8))}); err != nil {
		t.Fatal(err)
	}
	if err := store.Save(ctx, []Annotation{annotation("run-1", "agent.spinning", "hash-1", older, floatPointer(0.1))}); err != nil {
		t.Fatal(err)
	}
	latest, _ := store.Current(ctx, ScopeRun, "run-1", "agent.spinning", "hash-1")
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

func TestCoverageCountsLatestStatePerScope(t *testing.T) {
	store, _ := testStore(t)
	ctx := context.Background()
	base := time.Date(2026, 11, 18, 12, 0, 0, 0, time.UTC)
	rows := []Annotation{
		annotation("run-1", "a", "h1", base, floatPointer(0.7)),
		annotation("run-2", "a", "h1", base, nil),
		annotation("run-3", "a", "h1", base, nil),
	}
	rows[1].State = StateUnavailable
	rows[2].State = StateCompatible
	if err := store.Save(ctx, rows); err != nil {
		t.Fatal(err)
	}
	coverage, err := store.Coverage(ctx, time.Time{})
	if err != nil {
		t.Fatal(err)
	}
	counts := coverage["a"]
	if counts.Observed != 1 || counts.Compatible != 1 || counts.Unavailable != 1 || counts.NotObserved != 0 {
		t.Fatalf("counts = %+v", counts)
	}
	window, err := store.Coverage(ctx, base.Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if window["a"].Total() != 0 {
		t.Fatalf("future window = %+v", window["a"])
	}
}

func TestPositivesAppliesASignedScoreCeiling(t *testing.T) {
	store, _ := testStore(t)
	ctx := context.Background()
	now := time.Now().UTC()
	confidence := 0.9
	ceiling := -1.0
	save := func(scope string, score float64) {
		t.Helper()
		value := score
		item := annotation(scope, "operator.valence", "hash", now, &confidence)
		item.Value = Value{Kind: "score", Score: &value}
		if err := store.Save(ctx, []Annotation{item}); err != nil {
			t.Fatal(err)
		}
	}
	save("run-negative", -1.5)
	save("run-positive", 1.5)
	save("run-barely-negative", -0.5)
	level := -1 // index whose score (offset 0) is at or below the ceiling
	count, err := store.Positives(ctx, now.Add(-time.Hour), now.Add(time.Hour), "operator.valence", FrictionSelector{
		Threshold: confidence, LevelBound: &level, ScoreBound: &ceiling,
	})
	if err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("positives = %d, want only the observation at or below the ceiling", count)
	}
}

func TestPositivesUsesTheProbabilityMassInTheFrictionRegion(t *testing.T) {
	store, _ := testStore(t)
	ctx := context.Background()
	now := time.Now().UTC()
	save := func(scope string, score float64, probabilities map[string]float64) {
		t.Helper()
		value := score
		confidence := 0.8
		item := annotation(scope, "agent.spinning", "hash", now, &confidence)
		item.Value = Value{Kind: "score", Score: &value, Probabilities: probabilities}
		if err := store.Save(ctx, []Annotation{item}); err != nil {
			t.Fatal(err)
		}
	}
	// A confident top level is normal on a five-level scale; the mass at or
	// above level 3 is what the friction rule reads.
	save("run-spinning", 3, map[string]float64{"0": 0.02, "1": 0.08, "2": 0.15, "3": 0.45, "4": 0.30})
	save("run-mild", 2, map[string]float64{"0": 0.20, "1": 0.40, "2": 0.30, "3": 0.08, "4": 0.02})
	bound := 3
	count, err := store.Positives(ctx, now.Add(-time.Hour), now.Add(time.Hour), "agent.spinning", FrictionSelector{
		Threshold: 0.6, LevelBound: &bound, LevelAtLeast: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("positives = %d, want only the run whose mass at or above level 3 clears 0.6", count)
	}
}

func TestObservedScopesKeepsOnlyTheCurrentDefinition(t *testing.T) {
	store, _ := testStore(t)
	ctx := context.Background()
	now := time.Now().UTC()
	confidence := 0.9
	current := annotation("run-current", "agent.spinning", "current-hash", now, &confidence)
	stale := annotation("run-stale", "agent.spinning", "old-hash", now, &confidence)
	unavailable := annotation("run-unavailable", "agent.spinning", "current-hash", now, &confidence)
	unavailable.State = StateUnavailable
	if err := store.Save(ctx, []Annotation{current, stale, unavailable}); err != nil {
		t.Fatal(err)
	}
	scopes, err := store.ObservedScopes(ctx, now.Add(-time.Hour), now.Add(time.Hour), "agent.spinning", "current-hash", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(scopes) != 1 || scopes[0] != "run-current" {
		t.Fatalf("observed scopes = %v, want only the current observed run", scopes)
	}
}

func TestCoverageSeparatesRowsFromAnotherDefinition(t *testing.T) {
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
	legacy, err := store.CoverageWindow(ctx, time.Time{}, time.Time{})
	if err != nil {
		t.Fatal(err)
	}
	if legacy["a"].Observed != 3 {
		t.Fatalf("legacy coverage = %+v, want every row counted", legacy["a"])
	}
	current, err := store.CoverageWindowFor(ctx, time.Time{}, time.Time{}, map[string]string{"a": "current"})
	if err != nil {
		t.Fatal(err)
	}
	if current["a"].Observed != 1 || current["a"].Stale != 2 {
		t.Fatalf("hash-aware coverage = %+v, want 1 observed and 2 stale", current["a"])
	}
	if current["a"].Total() != 1 {
		t.Fatalf("total = %d, want stale rows excluded", current["a"].Total())
	}
	eligible, observed, err := store.RunCoverageFor(ctx, time.Time{}, time.Time{}, map[string]string{"a": "current"})
	if err != nil {
		t.Fatal(err)
	}
	if eligible != 3 || observed != 1 {
		t.Fatalf("run coverage = %d/%d, want 1 of 3 observed for the current hash", observed, eligible)
	}
	positives, err := store.PositivesFor(ctx, time.Time{}, time.Time{}, "a", "current", FrictionSelector{Threshold: 0.6})
	if err != nil {
		t.Fatal(err)
	}
	if positives != 1 {
		t.Fatalf("positives = %d, want only the current definition", positives)
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
