package docaccess_test

import (
	"context"
	"testing"

	apidb "github.com/vrooli/api-core/database"

	"knowledge-observatory/internal/dbtest"
	"knowledge-observatory/internal/docaccess"
)

func newRepo(t *testing.T) *docaccess.SQLite {
	t.Helper()
	return docaccess.NewSQLite(dbtest.New(t, apidb.SchemaProviderFunc(docaccess.Schema)))
}

func TestAccessRoundTripCoversEveryColumn(t *testing.T) {
	repo := newRepo(t)
	ctx := context.Background()

	if err := repo.LogAccess(ctx, docaccess.Access{
		ScenarioName: "vrooli-autoheal",
		DocType:      "reference",
		Operation:    "read",
	}); err != nil {
		t.Fatalf("log: %v", err)
	}

	rows, err := repo.Recent(ctx, 10)
	if err != nil {
		t.Fatalf("recent: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("got %d rows, want 1", len(rows))
	}
	got := rows[0]
	if got.ID == "" {
		t.Error("id was not generated")
	}
	if got.ScenarioName != "vrooli-autoheal" || got.DocType != "reference" || got.Operation != "read" {
		t.Errorf("row = %+v", got)
	}
	if got.CreatedAt.IsZero() {
		t.Error("created_at was not defaulted")
	}
}

func TestInvalidOperationIsRejected(t *testing.T) {
	repo := newRepo(t)
	err := repo.LogAccess(context.Background(), docaccess.Access{
		ScenarioName: "s", DocType: "d", Operation: "delete",
	})
	if err == nil {
		t.Fatal("expected the CHECK constraint to reject an unknown operation")
	}
}

// TestQueryStatsFilterClause proves the aggregate FILTER syntax carried over
// from Postgres unchanged — SQLite has supported it since 3.30.
func TestQueryStatsFilterClause(t *testing.T) {
	repo := newRepo(t)
	ctx := context.Background()

	writes := []docaccess.Access{
		{ScenarioName: "alpha", DocType: "guide", Operation: "read"},
		{ScenarioName: "alpha", DocType: "guide", Operation: "read"},
		{ScenarioName: "alpha", DocType: "guide", Operation: "write"},
		{ScenarioName: "alpha", DocType: "guide", Operation: "reset"},
		{ScenarioName: "beta", DocType: "guide", Operation: "read"},
	}
	for _, w := range writes {
		if err := repo.LogAccess(ctx, w); err != nil {
			t.Fatal(err)
		}
	}

	stats, err := repo.QueryStats(ctx, docaccess.Filter{})
	if err != nil {
		t.Fatalf("stats: %v", err)
	}
	if len(stats) != 2 {
		t.Fatalf("got %d groups, want 2", len(stats))
	}
	if stats[0].ScenarioName != "alpha" {
		t.Fatalf("stats[0] = %q, want alpha", stats[0].ScenarioName)
	}
	if stats[0].ReadCount != 2 || stats[0].WriteCount != 1 || stats[0].ResetCount != 1 {
		t.Errorf("alpha tallies = %d/%d/%d, want 2/1/1",
			stats[0].ReadCount, stats[0].WriteCount, stats[0].ResetCount)
	}

	filtered, err := repo.QueryStats(ctx, docaccess.Filter{ScenarioName: "beta"})
	if err != nil {
		t.Fatal(err)
	}
	if len(filtered) != 1 || filtered[0].ReadCount != 1 {
		t.Errorf("filtered = %+v, want only beta with 1 read", filtered)
	}
}
