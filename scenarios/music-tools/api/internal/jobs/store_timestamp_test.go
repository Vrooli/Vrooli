package jobs

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestStoreTimestampQueriesUseChronologicalOrder(t *testing.T) {
	ctx := context.Background()
	st := newStore(newJobsDB(t))
	base := time.Date(2026, time.November, 23, 12, 0, 0, 0, time.UTC)
	older := base.Add(100 * time.Millisecond)
	newer := base.Add(110 * time.Millisecond)
	for _, job := range []Job{
		{ID: "older", Operation: "compose", Lane: LaneGPU, State: StateSucceeded, CreatedAt: older, FinishedAt: &older, ResultRef: "out/older.wav"},
		{ID: "newer", Operation: "compose", Lane: LaneGPU, State: StateSucceeded, CreatedAt: newer, FinishedAt: &newer, ResultRef: "out/newer.wav"},
	} {
		if err := st.insert(ctx, job); err != nil {
			t.Fatal(err)
		}
	}

	listed, err := st.list(ctx, 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(listed) != 1 || listed[0].ID != "newer" {
		t.Errorf("newest jobs = %+v, want newer first", listed)
	}

	candidates, err := st.listRetentionCandidates(ctx, base.Add(105*time.Millisecond), 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(candidates) != 1 || candidates[0].ID != "older" {
		t.Errorf("retention candidates = %+v, want only older", candidates)
	}
}

func TestStoreMigrationCanonicalizesLegacyTimestampStrings(t *testing.T) {
	ctx := context.Background()
	db := newJobsDB(t)
	st := newStore(db)
	base := time.Date(2026, time.November, 23, 12, 0, 0, 0, time.UTC)
	older := base.Add(100 * time.Millisecond)
	newer := base.Add(110 * time.Millisecond)
	newerLocal := newer.In(time.FixedZone("owner-offset", -5*60*60))
	for _, fixture := range []struct {
		id       string
		created  string
		finished string
	}{
		{id: "legacy-older", created: older.Format(time.RFC3339Nano), finished: older.Format(time.RFC3339Nano)},
		{id: "legacy-newer", created: newerLocal.Format(time.RFC3339Nano), finished: newerLocal.Format(time.RFC3339Nano)},
	} {
		_, err := db.ExecContext(ctx, `INSERT INTO jobs (id, operation, lane, state, created_at, finished_at, result_ref) VALUES (?, ?, ?, ?, ?, ?, ?)`,
			fixture.id, "compose", string(LaneGPU), string(StateSucceeded), fixture.created, fixture.finished, "out/"+fixture.id+".wav")
		if err != nil {
			t.Fatal(err)
		}
	}
	if err := st.migrate(ctx); err != nil {
		t.Fatal(err)
	}
	if err := st.migrate(ctx); err != nil {
		t.Fatalf("second migration call: %v", err)
	}

	listed, err := st.list(ctx, 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(listed) != 1 || listed[0].ID != "legacy-newer" {
		t.Errorf("newest legacy jobs = %+v, want legacy-newer", listed)
	}
	candidates, err := st.listRetentionCandidates(ctx, base.Add(105*time.Millisecond), 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(candidates) != 1 || candidates[0].ID != "legacy-older" {
		t.Errorf("legacy retention candidates = %+v, want only legacy-older", candidates)
	}

	var storedCreated, storedFinished string
	if err := db.QueryRowContext(ctx, "SELECT created_at, finished_at FROM jobs WHERE id=?", "legacy-newer").Scan(&storedCreated, &storedFinished); err != nil {
		t.Fatal(err)
	}
	want := newer.UTC().Format("2006-01-02T15:04:05.000000000Z")
	if storedCreated != want || storedFinished != want {
		t.Errorf("canonical timestamps = %q / %q, want %q", storedCreated, storedFinished, want)
	}
}

func TestStoreMigrationRejectsMalformedHistoricalTimestamp(t *testing.T) {
	ctx := context.Background()
	db := newJobsDB(t)
	st := newStore(db)
	if _, err := db.ExecContext(ctx, `INSERT INTO jobs (id, operation, lane, state, created_at) VALUES (?, ?, ?, ?, ?)`,
		"broken-time", "compose", string(LaneCPU), string(StateQueued), "not-a-timestamp"); err != nil {
		t.Fatal(err)
	}
	if err := st.migrate(ctx); err == nil || !strings.Contains(err.Error(), `created_at for job "broken-time"`) {
		t.Fatalf("malformed timestamp migration error = %v", err)
	}
	var markers int
	if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM jobs_schema_migrations WHERE version=1").Scan(&markers); err != nil {
		t.Fatal(err)
	}
	if markers != 0 {
		t.Fatalf("migration marker count = %d, want 0", markers)
	}
}
