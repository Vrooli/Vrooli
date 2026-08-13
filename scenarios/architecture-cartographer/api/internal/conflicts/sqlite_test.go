package conflicts_test

import (
	"context"
	"database/sql"
	"testing"

	"architecture-cartographer/internal/conflicts"
	localdb "architecture-cartographer/internal/database"
	"architecture-cartographer/internal/testutil/db"

	"github.com/vrooli/api-core/schedule"

	apidb "github.com/vrooli/api-core/database"
)

func newSchemaDB(t *testing.T) *sql.DB {
	t.Helper()
	d := db.NewSQLite(t)
	if err := apidb.EnsureSchemas(context.Background(), d,
		apidb.SchemaProviderFunc(localdb.SystemSchema),
		apidb.SchemaProviderFunc(conflicts.Schema),
	); err != nil {
		t.Fatalf("EnsureSchemas: %v", err)
	}
	return d
}

func TestSQLiteRepository_UpsertAndGet(t *testing.T) {
	d := newSchemaDB(t)
	repo := conflicts.NewSQLiteRepository(d, schedule.System())

	c := conflicts.Conflict{
		Scenario:  "demo",
		Detector:  "cycle",
		Type:      "cycle",
		Severity:  conflicts.SeverityError,
		Locations: []string{"a.go", "b.go"},
	}
	saved, err := repo.UpsertConflict(context.Background(), c)
	if err != nil {
		t.Fatalf("Upsert: %v", err)
	}
	got, err := repo.GetConflict(context.Background(), saved.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Type != "cycle" || len(got.Locations) != 2 {
		t.Fatalf("unexpected: %+v", got)
	}
}

func TestSQLiteRepository_ListFiltersByType(t *testing.T) {
	d := newSchemaDB(t)
	repo := conflicts.NewSQLiteRepository(d, schedule.System())

	for _, typ := range []string{"cycle", "mislocated_file"} {
		if _, err := repo.UpsertConflict(context.Background(), conflicts.Conflict{
			Scenario: "demo", Detector: typ, Type: typ,
		}); err != nil {
			t.Fatalf("seed: %v", err)
		}
	}
	page, err := repo.ListConflicts(context.Background(), conflicts.ListConflictsFilter{
		Scenario: "demo",
		Types:    []string{"mislocated_file"},
	})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(page.Conflicts) != 1 {
		t.Fatalf("want 1 conflict, got %d", len(page.Conflicts))
	}
	if page.Conflicts[0].Type != "mislocated_file" {
		t.Fatalf("type filter failed: %+v", page.Conflicts[0])
	}
}
