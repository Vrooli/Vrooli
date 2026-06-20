package selfhealthsnapshots

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

func newTestRepo(t *testing.T) *SqliteRepository {
	t.Helper()
	dsn := "file:" + filepath.Join(t.TempDir(), "snapshots.db")
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if _, err := db.ExecContext(context.Background(), Schema()); err != nil {
		t.Fatalf("apply schema: %v", err)
	}
	return NewSqliteRepository(db)
}

func TestInsertDedupsOnDigest(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()
	base := time.Date(2026, 6, 20, 12, 0, 0, 0, time.UTC)

	snap := Snapshot{CapturedAt: base, WindowDays: 30, RunCount: 10, Availability: 0.9, Digest: "abc", PayloadJSON: `{"a":1}`}
	inserted, err := repo.Insert(ctx, snap)
	if err != nil || !inserted {
		t.Fatalf("first insert: inserted=%v err=%v", inserted, err)
	}

	// Identical digest at a later time must be skipped (idempotent dedup).
	snap.CapturedAt = base.Add(time.Hour)
	inserted, err = repo.Insert(ctx, snap)
	if err != nil {
		t.Fatalf("second insert err: %v", err)
	}
	if inserted {
		t.Fatal("identical-digest snapshot must not be inserted again")
	}

	// A differing digest inserts.
	snap2 := Snapshot{CapturedAt: base.Add(2 * time.Hour), RunCount: 12, Availability: 0.95, Digest: "def", PayloadJSON: `{"a":2}`}
	inserted, err = repo.Insert(ctx, snap2)
	if err != nil || !inserted {
		t.Fatalf("third insert: inserted=%v err=%v", inserted, err)
	}
}

func TestLatestAndLatestDifferingDigest(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()
	base := time.Date(2026, 6, 20, 12, 0, 0, 0, time.UTC)

	if _, ok, err := repo.Latest(ctx); err != nil || ok {
		t.Fatalf("empty Latest: ok=%v err=%v", ok, err)
	}

	for i, d := range []struct {
		digest string
		avail  float64
	}{{"d1", 0.80}, {"d2", 0.85}, {"d3", 0.90}} {
		if _, err := repo.Insert(ctx, Snapshot{CapturedAt: base.Add(time.Duration(i) * time.Hour), Availability: d.avail, RunCount: i, Digest: d.digest, PayloadJSON: "{}"}); err != nil {
			t.Fatalf("seed insert %d: %v", i, err)
		}
	}

	latest, ok, err := repo.Latest(ctx)
	if err != nil || !ok {
		t.Fatalf("Latest: ok=%v err=%v", ok, err)
	}
	if latest.Digest != "d3" || latest.Availability != 0.90 {
		t.Fatalf("Latest = %+v, want d3/0.90", latest)
	}

	// Baseline for a trend delta against the newest digest is the prior differing one.
	prev, ok, err := repo.LatestDifferingDigest(ctx, "d3")
	if err != nil || !ok {
		t.Fatalf("LatestDifferingDigest: ok=%v err=%v", ok, err)
	}
	if prev.Digest != "d2" {
		t.Fatalf("LatestDifferingDigest = %q, want d2", prev.Digest)
	}
}

func TestSeriesNewestFirstAndBounded(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()
	base := time.Date(2026, 6, 20, 12, 0, 0, 0, time.UTC)
	for i := 0; i < 5; i++ {
		if _, err := repo.Insert(ctx, Snapshot{CapturedAt: base.Add(time.Duration(i) * time.Hour), RunCount: i, Digest: string(rune('a' + i)), PayloadJSON: "{}"}); err != nil {
			t.Fatalf("seed %d: %v", i, err)
		}
	}

	all, err := repo.Series(ctx, SeriesQuery{})
	if err != nil {
		t.Fatalf("Series: %v", err)
	}
	if len(all) != 5 {
		t.Fatalf("len(all) = %d, want 5", len(all))
	}
	if all[0].RunCount != 4 || all[4].RunCount != 0 {
		t.Fatalf("series not newest-first: %v", all)
	}

	limited, err := repo.Series(ctx, SeriesQuery{Limit: 2})
	if err != nil {
		t.Fatalf("Series limited: %v", err)
	}
	if len(limited) != 2 || limited[0].RunCount != 4 {
		t.Fatalf("limited series wrong: %v", limited)
	}

	since, err := repo.Series(ctx, SeriesQuery{Since: base.Add(3 * time.Hour)})
	if err != nil {
		t.Fatalf("Series since: %v", err)
	}
	if len(since) != 2 {
		t.Fatalf("since series len = %d, want 2", len(since))
	}
}

func TestSweeperRunOnceDedups(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()
	now := time.Date(2026, 6, 20, 12, 0, 0, 0, time.UTC)

	// First build returns one rollup; second build returns the SAME content
	// (identical payload → identical digest → skip); third differs.
	calls := 0
	build := func(context.Context) (Rollup, error) {
		calls++
		switch calls {
		case 1, 2:
			return Rollup{WindowDays: 30, RunCount: 10, Availability: 0.9, Payload: map[string]any{"availability": 0.9, "run_count": 10}}, nil
		default:
			return Rollup{WindowDays: 30, RunCount: 11, Availability: 0.91, Payload: map[string]any{"availability": 0.91, "run_count": 11}}, nil
		}
	}
	sweeper, err := NewSweeper(SweeperConfig{
		Repository: repo,
		Build:      build,
		Now:        func() time.Time { return now },
	})
	if err != nil {
		t.Fatalf("NewSweeper: %v", err)
	}

	if _, inserted, err := sweeper.RunOnce(ctx); err != nil || !inserted {
		t.Fatalf("first sweep: inserted=%v err=%v", inserted, err)
	}
	if _, inserted, err := sweeper.RunOnce(ctx); err != nil || inserted {
		t.Fatalf("identical-content sweep must skip: inserted=%v err=%v", inserted, err)
	}
	if _, inserted, err := sweeper.RunOnce(ctx); err != nil || !inserted {
		t.Fatalf("changed-content sweep must insert: inserted=%v err=%v", inserted, err)
	}

	all, err := repo.Series(ctx, SeriesQuery{})
	if err != nil {
		t.Fatalf("Series: %v", err)
	}
	if len(all) != 2 {
		t.Fatalf("expected 2 distinct snapshots, got %d", len(all))
	}
}
