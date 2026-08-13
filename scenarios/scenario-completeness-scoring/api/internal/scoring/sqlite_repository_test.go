package scoring

import (
	"context"
	"database/sql"
	"strconv"
	"strings"
	"testing"
	"time"

	apidb "github.com/vrooli/api-core/database"

	testdb "github.com/vrooli/api-core/databasetest"
	localdb "scenario-completeness-scoring/internal/database"
)

func TestSQLiteSnapshotRepositoryUpsertAndLatest(t *testing.T) {
	repo := newTestSnapshotRepo(t)
	ctx := context.Background()
	now := time.Date(2026, 6, 11, 12, 0, 0, 0, time.UTC)
	importance := 0.7

	inserted, err := repo.UpsertSnapshot(ctx, Snapshot{
		Scenario:       "cli-health",
		Category:       "utility",
		Digest:         "td:one",
		Composite:      72,
		Classification: "mostly_complete",
		WorkingRung:    "R2",
		BreakdownJSON:  `{"score":72}`,
		Importance:     &importance,
		Source:         "test",
		CreatedAt:      now,
	})
	if err != nil {
		t.Fatalf("UpsertSnapshot() error = %v", err)
	}
	if !inserted {
		t.Fatalf("UpsertSnapshot() inserted = false, want true")
	}
	inserted, err = repo.UpsertSnapshot(ctx, Snapshot{
		Scenario:       "cli-health",
		Category:       "utility",
		Digest:         "td:one",
		Composite:      99,
		Classification: "production_ready",
		CreatedAt:      now.Add(time.Minute),
	})
	if err != nil {
		t.Fatalf("duplicate UpsertSnapshot() error = %v", err)
	}
	if inserted {
		t.Fatalf("duplicate UpsertSnapshot() inserted = true, want false")
	}

	got, ok, err := repo.LatestFor(ctx, "cli-health")
	if err != nil {
		t.Fatalf("LatestFor() error = %v", err)
	}
	if !ok {
		t.Fatalf("LatestFor() ok = false, want true")
	}
	if got.Composite != 72 || got.Digest != "td:one" || got.Importance == nil || *got.Importance != importance {
		t.Fatalf("LatestFor() = %+v, want inserted snapshot", got)
	}
}

// TestSQLiteSnapshotRepositoryImportanceUpsertOntoExistingRow locks in the
// INSERT-OR-IGNORE importance fix: the fast score sweep writes importance=NULL,
// and a later importance-refresh for the same (scenario, digest) must land its
// computed importance onto that existing row rather than being silently ignored.
func TestSQLiteSnapshotRepositoryImportanceUpsertOntoExistingRow(t *testing.T) {
	repo := newTestSnapshotRepo(t)
	ctx := context.Background()
	now := time.Date(2026, 6, 22, 12, 0, 0, 0, time.UTC)

	// Fast sweep: no importance.
	if _, err := repo.UpsertSnapshot(ctx, Snapshot{
		Scenario: "cli-health", Category: "utility", Digest: "td:x",
		Composite: 60, Classification: "developing", CreatedAt: now,
	}); err != nil {
		t.Fatalf("seed UpsertSnapshot: %v", err)
	}
	if latest, _, _ := repo.LatestFor(ctx, "cli-health"); latest.Importance != nil {
		t.Fatalf("expected nil importance after fast sweep, got %v", *latest.Importance)
	}

	// Importance-refresh: same (scenario, digest), now carrying importance.
	importance := 0.83
	if _, err := repo.UpsertSnapshot(ctx, Snapshot{
		Scenario: "cli-health", Category: "utility", Digest: "td:x",
		Composite: 60, Classification: "developing", Importance: &importance,
		Source: "importance-refresh", CreatedAt: now.Add(time.Hour),
	}); err != nil {
		t.Fatalf("refresh UpsertSnapshot: %v", err)
	}
	latest, ok, err := repo.LatestFor(ctx, "cli-health")
	if err != nil || !ok {
		t.Fatalf("LatestFor: ok=%v err=%v", ok, err)
	}
	if latest.Importance == nil || *latest.Importance != importance {
		t.Fatalf("importance not upserted onto existing row: %+v", latest.Importance)
	}
}

func TestSQLiteSnapshotRepositorySeriesFor(t *testing.T) {
	repo := newTestSnapshotRepo(t)
	ctx := context.Background()
	base := time.Date(2026, 6, 11, 12, 0, 0, 0, time.UTC)
	mustInsertSnapshots(t, repo,
		snapshotAt("alpha", "td:old", 40, base),
		snapshotAt("alpha", "td:new", 60, base.Add(time.Hour)),
		snapshotAt("beta", "td:other", 80, base.Add(2*time.Hour)),
	)

	got, err := repo.SeriesFor(ctx, TrendQuery{Scenario: "alpha", Limit: 10, Since: base.Add(30 * time.Minute)})
	if err != nil {
		t.Fatalf("SeriesFor() error = %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("SeriesFor() len = %d, want 1: %+v", len(got), got)
	}
	if got[0].Digest != "td:new" {
		t.Fatalf("SeriesFor()[0].Digest = %q, want td:new", got[0].Digest)
	}
}

func TestSQLiteSnapshotRepositoryLatestDifferingDigest(t *testing.T) {
	repo := newTestSnapshotRepo(t)
	ctx := context.Background()
	base := time.Date(2026, 6, 11, 12, 0, 0, 0, time.UTC)
	mustInsertSnapshots(t, repo,
		snapshotAt("alpha", "td:old", 40, base),
		snapshotAt("alpha", "td:new", 60, base.Add(time.Hour)),
	)

	got, ok, err := repo.LatestDifferingDigest(ctx, "alpha", "td:new")
	if err != nil {
		t.Fatalf("LatestDifferingDigest() error = %v", err)
	}
	if !ok {
		t.Fatalf("LatestDifferingDigest() ok = false, want true")
	}
	if got.Digest != "td:old" {
		t.Fatalf("LatestDifferingDigest().Digest = %q, want td:old", got.Digest)
	}
}

func TestSQLiteSnapshotRepositoryListPageSortFilterPagination(t *testing.T) {
	repo := newTestSnapshotRepo(t)
	ctx := context.Background()
	base := time.Date(2026, 6, 11, 12, 0, 0, 0, time.UTC)
	mustInsertSnapshots(t, repo,
		snapshotAt("alpha", "td:alpha-old", 20, base),
		snapshotAt("alpha", "td:alpha-new", 55, base.Add(time.Hour)),
		snapshotAt("beta", "td:beta", 80, base.Add(2*time.Hour)),
		snapshotAt("gamma", "td:gamma", 65, base.Add(3*time.Hour)),
	)

	minScore := 50
	page, err := repo.ListPage(ctx, ListQuery{
		SortBy:   SortByComposite,
		Order:    SortDesc,
		Limit:    2,
		MinScore: &minScore,
	})
	if err != nil {
		t.Fatalf("ListPage() error = %v", err)
	}
	if !page.HasNext || page.NextOffset != 2 {
		t.Fatalf("ListPage() page metadata = %+v, want has next offset 2", page)
	}
	if got := snapshotNames(page.Snapshots); got != "beta,gamma" {
		t.Fatalf("ListPage() names = %q, want beta,gamma", got)
	}

	page, err = repo.ListPage(ctx, ListQuery{
		SortBy:   SortByComposite,
		Order:    SortDesc,
		Limit:    2,
		Offset:   page.NextOffset,
		MinScore: &minScore,
	})
	if err != nil {
		t.Fatalf("ListPage(next) error = %v", err)
	}
	if page.HasNext {
		t.Fatalf("ListPage(next) HasNext = true, want false")
	}
	if got := snapshotNames(page.Snapshots); got != "alpha" {
		t.Fatalf("ListPage(next) names = %q, want alpha", got)
	}
}

func TestSQLiteSnapshotRepositoryListPageUsesNewestCreatedAt(t *testing.T) {
	repo := newTestSnapshotRepo(t)
	ctx := context.Background()
	base := time.Date(2026, 6, 11, 12, 0, 0, 0, time.UTC)
	mustInsertSnapshots(t, repo,
		snapshotAt("alpha", "td:newer", 90, base.Add(time.Hour)),
		snapshotAt("alpha", "td:older-inserted-later", 10, base),
		snapshotAt("beta", "td:beta", 50, base.Add(2*time.Hour)),
	)

	page, err := repo.ListPage(ctx, ListQuery{SortBy: SortByComposite, Order: SortDesc, Limit: 10})
	if err != nil {
		t.Fatalf("ListPage() error = %v", err)
	}
	if got := snapshotNames(page.Snapshots); got != "alpha,beta" {
		t.Fatalf("ListPage() names = %q, want alpha,beta", got)
	}
	if page.Snapshots[0].Digest != "td:newer" || page.Snapshots[0].Composite != 90 {
		t.Fatalf("ListPage() selected alpha snapshot = %+v, want newest created_at", page.Snapshots[0])
	}
}

func TestSQLiteSnapshotRepositoryListPagePrioritySort(t *testing.T) {
	repo := newTestSnapshotRepo(t)
	ctx := context.Background()
	base := time.Date(2026, 6, 11, 12, 0, 0, 0, time.UTC)
	highImportance := 0.9
	lowImportance := 0.2
	mustInsertSnapshots(t, repo,
		snapshotAt("higher-gap", "td:gap", 40, base, &lowImportance),
		snapshotAt("higher-priority", "td:priority", 70, base.Add(time.Hour), &highImportance),
		snapshotAt("no-importance", "td:none", 10, base.Add(2*time.Hour)),
	)

	page, err := repo.ListPage(ctx, ListQuery{SortBy: SortByPriority, Order: SortDesc, Limit: 10})
	if err != nil {
		t.Fatalf("ListPage(priority) error = %v", err)
	}
	if got := snapshotNames(page.Snapshots); got != "higher-priority,higher-gap,no-importance" {
		t.Fatalf("ListPage(priority) names = %q, want priority order", got)
	}
}

func TestSQLiteSnapshotRepositoryListPageLargeFixture(t *testing.T) {
	if testing.Short() {
		t.Skip("large persisted-read fixture")
	}
	repo := newTestSnapshotRepo(t)
	ctx := context.Background()
	base := time.Date(2026, 6, 11, 12, 0, 0, 0, time.UTC)
	snaps := make([]Snapshot, 0, 10_000)
	for i := 0; i < 10_000; i++ {
		snaps = append(snaps, snapshotAt(
			"scenario-"+strings.Repeat("0", 5-lenInt(i))+strconv.Itoa(i),
			"td:large-"+strconv.Itoa(i),
			i%101,
			base.Add(time.Duration(i)*time.Second),
		))
	}
	mustInsertSnapshotsFast(t, repo, snaps...)

	start := time.Now()
	page, err := repo.ListPage(ctx, ListQuery{SortBy: SortByComposite, Order: SortDesc, Limit: 25})
	elapsed := time.Since(start)
	if err != nil {
		t.Fatalf("ListPage(large) error = %v", err)
	}
	if len(page.Snapshots) != 25 || !page.HasNext {
		t.Fatalf("ListPage(large) metadata = len %d page %+v, want 25 with next page", len(page.Snapshots), page)
	}
	if elapsed > 500*time.Millisecond {
		t.Fatalf("ListPage(large) took %s, want persisted page read under 500ms", elapsed)
	}
}

func TestSQLiteSnapshotRepositoryMeasureAggregates(t *testing.T) {
	repo := newTestSnapshotRepo(t)
	ctx := context.Background()
	base := time.Date(2026, 6, 11, 12, 0, 0, 0, time.UTC)
	mustInsertSnapshots(t, repo,
		snapshotWithRung("alpha", "td:a-old", 30, "R0 Runnable & green", base),
		snapshotWithRung("alpha", "td:a-new", 50, "R1 Standards-clean", base.Add(time.Hour)),
		snapshotWithRung("beta", "td:b", 80, "R3 Product-ready", base.Add(2*time.Hour)),
		snapshotWithRung("clean", "td:c", 100, "", base.Add(3*time.Hour)),
		snapshotWithRung("outside", "td:o", 10, "R0 Runnable & green", base.Add(-48*time.Hour)),
	)
	window := MeasureWindow{From: base.Add(-time.Minute), To: base.Add(24 * time.Hour)}

	belowR2, err := repo.CountLatestBelowRung(ctx, 2, window)
	if err != nil {
		t.Fatalf("CountLatestBelowRung() error = %v", err)
	}
	if belowR2 != 1 {
		t.Fatalf("CountLatestBelowRung() = %d, want 1", belowR2)
	}

	avg, ok, err := repo.AverageLatestComposite(ctx, window)
	if err != nil {
		t.Fatalf("AverageLatestComposite() error = %v", err)
	}
	if !ok {
		t.Fatal("AverageLatestComposite() ok = false, want true")
	}
	if avg < 76.66 || avg > 76.67 {
		t.Fatalf("AverageLatestComposite() = %v, want about 76.67", avg)
	}

	series, err := repo.FleetScoreSeries(ctx, window)
	if err != nil {
		t.Fatalf("FleetScoreSeries() error = %v", err)
	}
	if len(series) != 1 {
		t.Fatalf("FleetScoreSeries() len = %d, want 1", len(series))
	}
	if series[0].Count != 4 || series[0].Score != 65 {
		t.Fatalf("FleetScoreSeries()[0] = %+v, want count 4 score 65", series[0])
	}
}

func TestSQLiteSnapshotRepositoryMeasureAggregatesUseNewestCreatedAt(t *testing.T) {
	repo := newTestSnapshotRepo(t)
	ctx := context.Background()
	base := time.Date(2026, 6, 11, 12, 0, 0, 0, time.UTC)
	mustInsertSnapshots(t, repo,
		snapshotWithRung("alpha", "td:newer", 80, "R3 Product-ready", base.Add(time.Hour)),
		snapshotWithRung("alpha", "td:older-inserted-later", 10, "R0 Runnable & green", base),
		snapshotWithRung("beta", "td:beta", 60, "R1 Standards-clean", base.Add(2*time.Hour)),
	)
	window := MeasureWindow{From: base.Add(-time.Minute), To: base.Add(24 * time.Hour)}

	belowR2, err := repo.CountLatestBelowRung(ctx, 2, window)
	if err != nil {
		t.Fatalf("CountLatestBelowRung() error = %v", err)
	}
	if belowR2 != 1 {
		t.Fatalf("CountLatestBelowRung() = %d, want only beta below R2", belowR2)
	}

	avg, ok, err := repo.AverageLatestComposite(ctx, window)
	if err != nil {
		t.Fatalf("AverageLatestComposite() error = %v", err)
	}
	if !ok {
		t.Fatal("AverageLatestComposite() ok = false, want true")
	}
	if avg != 70 {
		t.Fatalf("AverageLatestComposite() = %v, want newest-created_at average 70", avg)
	}
}

func newTestSnapshotRepo(t *testing.T) *SQLiteSnapshotRepository {
	t.Helper()
	d := testdb.NewSQLite(t)
	if err := apidb.EnsureSchemas(context.Background(), d,
		apidb.SchemaProviderFunc(localdb.SystemSchema),
		apidb.SchemaProviderFunc(Schema),
	); err != nil {
		t.Fatalf("EnsureSchemas() error = %v", err)
	}
	return NewSQLiteSnapshotRepository(d)
}

func mustInsertSnapshots(t *testing.T, repo *SQLiteSnapshotRepository, snapshots ...Snapshot) {
	t.Helper()
	for _, snap := range snapshots {
		inserted, err := repo.UpsertSnapshot(context.Background(), snap)
		if err != nil {
			t.Fatalf("UpsertSnapshot(%s/%s) error = %v", snap.Scenario, snap.Digest, err)
		}
		if !inserted {
			t.Fatalf("UpsertSnapshot(%s/%s) inserted = false, want true", snap.Scenario, snap.Digest)
		}
	}
}

func mustInsertSnapshotsFast(t *testing.T, repo *SQLiteSnapshotRepository, snapshots ...Snapshot) {
	t.Helper()
	db, ok := repo.db.(interface {
		Begin() (*sql.Tx, error)
	})
	if !ok {
		t.Fatalf("fixture database does not support transactions")
	}
	tx, err := db.Begin()
	if err != nil {
		t.Fatalf("begin fixture transaction: %v", err)
	}
	stmt, err := tx.Prepare(`INSERT INTO score_snapshots
		(scenario, category, digest, composite, classification, working_rung, breakdown_json, importance, source, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		_ = tx.Rollback()
		t.Fatalf("prepare fixture insert: %v", err)
	}
	for _, snap := range snapshots {
		if _, err := stmt.Exec(
			snap.Scenario,
			defaultString(snap.Category, "utility"),
			snap.Digest,
			snap.Composite,
			snap.Classification,
			snap.WorkingRung,
			defaultString(snap.BreakdownJSON, "{}"),
			nullableFloat(snap.Importance),
			defaultString(snap.Source, "test"),
			formatSnapshotTime(snap.CreatedAt),
		); err != nil {
			_ = stmt.Close()
			_ = tx.Rollback()
			t.Fatalf("fixture insert %s/%s: %v", snap.Scenario, snap.Digest, err)
		}
	}
	if err := stmt.Close(); err != nil {
		_ = tx.Rollback()
		t.Fatalf("close fixture insert: %v", err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatalf("commit fixture transaction: %v", err)
	}
}

func snapshotAt(scenario, digest string, score int, createdAt time.Time, importance ...*float64) Snapshot {
	var imp *float64
	if len(importance) > 0 {
		imp = importance[0]
	}
	return Snapshot{
		Scenario:       scenario,
		Category:       "utility",
		Digest:         digest,
		Composite:      score,
		Classification: "test",
		WorkingRung:    "R2",
		BreakdownJSON:  "{}",
		Importance:     imp,
		Source:         "test",
		CreatedAt:      createdAt,
	}
}

func snapshotWithRung(scenario, digest string, score int, rung string, createdAt time.Time) Snapshot {
	snap := snapshotAt(scenario, digest, score, createdAt)
	snap.WorkingRung = rung
	return snap
}

func snapshotNames(snaps []Snapshot) string {
	names := make([]string, 0, len(snaps))
	for _, snap := range snaps {
		names = append(names, snap.Scenario)
	}
	return strings.Join(names, ",")
}

func lenInt(v int) int {
	if v == 0 {
		return 1
	}
	n := 0
	for v > 0 {
		n++
		v /= 10
	}
	return n
}
