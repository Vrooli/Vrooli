package apply_test

import (
	"context"
	"database/sql"
	"testing"

	"architecture-cartographer/internal/apply"
	"architecture-cartographer/internal/clock"
	localdb "architecture-cartographer/internal/database"
	"architecture-cartographer/internal/testutil/db"

	apidb "github.com/vrooli/api-core/database"
)

func newSchemaDB(t *testing.T) *sql.DB {
	t.Helper()
	d := db.NewSQLite(t)
	if err := apidb.EnsureSchemas(context.Background(), d,
		apidb.SchemaProviderFunc(localdb.SystemSchema),
		apidb.SchemaProviderFunc(apply.Schema),
	); err != nil {
		t.Fatalf("EnsureSchemas: %v", err)
	}
	return d
}

func TestSQLiteRepository_SaveAndGetPlan(t *testing.T) {
	d := newSchemaDB(t)
	repo := apply.NewSQLiteRepository(d, clock.System{})

	saved, err := repo.SavePlan(context.Background(), apply.Plan{
		Scenario: "demo",
		Domain:   "graph",
		Operations: []apply.Operation{{
			ID:   "op-1",
			Kind: apply.OperationKindMoveFile,
		}},
		ConflictIDs: []string{"c-1"},
	})
	if err != nil {
		t.Fatalf("Save: %v", err)
	}
	got, err := repo.GetPlan(context.Background(), saved.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if len(got.Operations) != 1 || got.Operations[0].Kind != apply.OperationKindMoveFile {
		t.Fatalf("unexpected: %+v", got)
	}
}

func TestSQLiteRepository_BaselineMissingReturnsEmpty(t *testing.T) {
	d := newSchemaDB(t)
	repo := apply.NewSQLiteRepository(d, clock.System{})
	b, err := repo.GetBaseline(context.Background(), "demo")
	if err != nil {
		t.Fatalf("GetBaseline: %v", err)
	}
	if b.Green {
		t.Fatal("empty baseline must not be green")
	}
}
