package conflicts_test

import (
	"context"
	"database/sql"
	"testing"

	"architecture-cartographer/internal/clock"
	"architecture-cartographer/internal/conflicts"
	localdb "architecture-cartographer/internal/database"
	"architecture-cartographer/internal/testutil/db"

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
	repo := conflicts.NewSQLiteRepository(d, clock.System{})

	c := conflicts.Conflict{
		Scenario:  "demo",
		Detector:  "cycle",
		Type:      "cycle",
		Severity:  conflicts.SeverityError,
		Status:    conflicts.ResolutionStatusDetected,
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

func TestSQLiteRepository_UpdateStatus(t *testing.T) {
	d := newSchemaDB(t)
	repo := conflicts.NewSQLiteRepository(d, clock.System{})

	saved, err := repo.UpsertConflict(context.Background(), conflicts.Conflict{
		Scenario: "demo",
		Detector: "cycle",
		Type:     "cycle",
		Status:   conflicts.ResolutionStatusDetected,
	})
	if err != nil {
		t.Fatalf("seed: %v", err)
	}
	updated, err := repo.UpdateStatus(context.Background(), saved.ID, conflicts.ResolutionStatusAssigned, "ok", "graph")
	if err != nil {
		t.Fatalf("UpdateStatus: %v", err)
	}
	if updated.Status != conflicts.ResolutionStatusAssigned || updated.AssignedDomain != "graph" {
		t.Fatalf("unexpected: %+v", updated)
	}
}

func TestSQLiteRepository_ListFiltersByStatus(t *testing.T) {
	d := newSchemaDB(t)
	repo := conflicts.NewSQLiteRepository(d, clock.System{})

	for _, status := range []conflicts.ResolutionStatus{
		conflicts.ResolutionStatusDetected,
		conflicts.ResolutionStatusResolved,
	} {
		if _, err := repo.UpsertConflict(context.Background(), conflicts.Conflict{
			Scenario: "demo", Detector: "cycle", Type: "cycle", Status: status,
		}); err != nil {
			t.Fatalf("seed: %v", err)
		}
	}
	page, err := repo.ListConflicts(context.Background(), conflicts.ListConflictsFilter{
		Scenario: "demo",
		Statuses: []conflicts.ResolutionStatus{conflicts.ResolutionStatusResolved},
	})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(page.Conflicts) != 1 {
		t.Fatalf("want 1 conflict, got %d", len(page.Conflicts))
	}
	if page.Conflicts[0].Status != conflicts.ResolutionStatusResolved {
		t.Fatalf("status filter failed: %+v", page.Conflicts[0])
	}
}
