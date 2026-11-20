package runannotations

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	_ "modernc.org/sqlite"
)

// selectorFixture creates only the tables the selector reads: the run census
// and the annotation store it consults for rotation.
func selectorFixture(t *testing.T) (*sqlx.DB, *Store) {
	t.Helper()
	db, err := sqlx.Connect("sqlite", "file:"+filepath.Join(t.TempDir(), "selector.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if _, err := db.Exec(`CREATE TABLE runs (id TEXT PRIMARY KEY, started_at TEXT)`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`CREATE TABLE run_events (run_id TEXT NOT NULL, event_type TEXT NOT NULL)`); err != nil {
		t.Fatal(err)
	}
	store := NewStore(db)
	if err := store.EnsureSchema(context.Background()); err != nil {
		t.Fatal(err)
	}
	return db, store
}

func TestCandidateRunsWindowForRotatesAcrossUnannotatedRuns(t *testing.T) {
	db, store := selectorFixture(t)
	ctx := context.Background()
	base := time.Date(2026, 11, 18, 9, 0, 0, 0, time.UTC)
	runs := map[string]time.Time{
		"run-1": base,
		"run-2": base.Add(time.Hour),
		"run-3": base.Add(2 * time.Hour),
		"run-4": base.Add(3 * time.Hour),
	}
	for id, started := range runs {
		if _, err := db.Exec(`INSERT INTO runs (id, started_at) VALUES (?, ?)`, id, formatTime(started)); err != nil {
			t.Fatal(err)
		}
	}
	const hash = "hash-current"
	record := func(runID string, at time.Time) {
		t.Helper()
		probability := 0.9
		annotation := Annotation{
			ScopeKind: ScopeRun, ScopeID: runID, LabelID: "agent.spinning", DefinitionHash: hash,
			Model: "model-a", ModelVersion: "role-a", Method: "decision", State: StateObserved,
			Value: Value{Kind: "noul", Noul: &probability}, Probability: &probability, ObservedAt: at,
		}
		if err := store.Save(ctx, []Annotation{annotation}); err != nil {
			t.Fatal(err)
		}
	}
	// The two newest runs are already answered by this pack.
	record("run-3", base.Add(4*time.Hour))
	record("run-4", base.Add(5*time.Hour))

	selector := NewSQLiteRunSelector(db)
	window := [2]time.Time{base.Add(-time.Hour), base.Add(24 * time.Hour)}
	got, err := selector.CandidateRunsWindowFor(ctx, window[0], window[1], 2, []string{hash})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0] != "run-2" || got[1] != "run-1" {
		t.Fatalf("first rotated pass = %v, want the two unannotated runs newest-first", got)
	}

	// Answering the newest unannotated run must advance to the next one rather
	// than re-selecting the same pair.
	record("run-2", base.Add(6*time.Hour))
	got, err = selector.CandidateRunsWindowFor(ctx, window[0], window[1], 2, []string{hash})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0] != "run-1" || got[1] != "run-3" {
		t.Fatalf("second rotated pass = %v, want the remaining run then the oldest answer", got)
	}

	// A foreign hash does not count as answered by this pack, so every run is
	// a candidate again.
	got, err = selector.CandidateRunsWindowFor(ctx, window[0], window[1], 2, []string{"hash-other"})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0] != "run-4" || got[1] != "run-3" {
		t.Fatalf("foreign-hash pass = %v, want newest-first", got)
	}

	// No hashes falls back to plain recent order instead of a query that can
	// never rotate.
	got, err = selector.CandidateRunsWindowFor(ctx, window[0], window[1], 2, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0] != "run-4" || got[1] != "run-3" {
		t.Fatalf("empty-hash fallback = %v, want newest-first", got)
	}
}

// recordingHashedSelector proves Evaluate hands the pack's own definition
// hashes to a rotation-capable selector.
type recordingHashedSelector struct{ hashes []string }

func (s *recordingHashedSelector) CandidateRuns(context.Context, time.Time, int) ([]string, error) {
	return nil, nil
}

func (s *recordingHashedSelector) CandidateRunsWindowFor(_ context.Context, _, _ time.Time, _ int, hashes []string) ([]string, error) {
	s.hashes = append([]string(nil), hashes...)
	return nil, nil
}

func TestEvaluatePassesPackHashesToRotationCapableSelector(t *testing.T) {
	selector := &recordingHashedSelector{}
	service, _ := testService(t, decisionLabels, nil, nil, Options{Selector: selector})
	if _, err := service.Evaluate(context.Background(), EvaluateOptions{}); err != nil {
		t.Fatal(err)
	}
	want := service.DefinitionHashes()
	if len(want) == 0 {
		t.Fatal("pack reported no definition hashes")
	}
	if len(selector.hashes) != len(want) {
		t.Fatalf("selector received %d hashes, want %d", len(selector.hashes), len(want))
	}
	for i := range want {
		if selector.hashes[i] != want[i] {
			t.Fatalf("selector hash %d = %q, want %q", i, selector.hashes[i], want[i])
		}
	}
}

func TestCandidateRunsPreferEvidenceBearingRuns(t *testing.T) {
	db, _ := selectorFixture(t)
	ctx := context.Background()
	base := time.Date(2026, 11, 18, 9, 0, 0, 0, time.UTC)
	// Newer synthetic runs recorded only metrics; older ones carry messages.
	for index, id := range []string{"metric-1", "metric-2", "metric-3", "talk-1", "talk-2"} {
		started := base.Add(time.Duration(index) * time.Hour)
		if _, err := db.Exec(`INSERT INTO runs (id, started_at) VALUES (?, ?)`, id, formatTime(started)); err != nil {
			t.Fatal(err)
		}
	}
	for _, id := range []string{"metric-1", "metric-2", "metric-3"} {
		if _, err := db.Exec(`INSERT INTO run_events (run_id, event_type) VALUES (?, 'metric')`, id); err != nil {
			t.Fatal(err)
		}
	}
	for _, id := range []string{"talk-1", "talk-2"} {
		if _, err := db.Exec(`INSERT INTO run_events (run_id, event_type) VALUES (?, 'message')`, id); err != nil {
			t.Fatal(err)
		}
	}
	selector := NewSQLiteRunSelector(db)
	got, err := selector.CandidateRunsWindowFor(ctx, time.Time{}, time.Time{}, 2, []string{"hash-current"})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0] != "talk-2" || got[1] != "talk-1" {
		t.Fatalf("selection = %v, want the evidence-bearing runs before metric-only runs", got)
	}
	// The plain recency fallback follows the same rule so a caller without pack
	// hashes cannot starve interactive runs either.
	plain, err := selector.CandidateRunsWindow(ctx, time.Time{}, time.Time{}, 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(plain) != 2 || plain[0] != "talk-2" || plain[1] != "talk-1" {
		t.Fatalf("plain selection = %v, want evidence first", plain)
	}
}
