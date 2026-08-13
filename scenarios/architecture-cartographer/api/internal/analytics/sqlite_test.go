package analytics_test

import (
	"context"
	"database/sql"
	"testing"

	"architecture-cartographer/internal/analytics"
	localdb "architecture-cartographer/internal/database"
	db "github.com/vrooli/api-core/databasetest"

	"github.com/vrooli/api-core/schedule"

	apidb "github.com/vrooli/api-core/database"
)

func newSchemaDB(t *testing.T) *sql.DB {
	t.Helper()
	d := db.NewSQLite(t)
	if err := apidb.EnsureSchemas(context.Background(), d,
		apidb.SchemaProviderFunc(localdb.SystemSchema),
		apidb.SchemaProviderFunc(analytics.Schema),
	); err != nil {
		t.Fatalf("EnsureSchemas: %v", err)
	}
	return d
}

func TestSQLiteRepository_AppendAndListEvent(t *testing.T) {
	d := newSchemaDB(t)
	repo := analytics.NewSQLiteRepository(d, schedule.System())

	ev, err := repo.AppendEvent(context.Background(), analytics.Event{
		Scenario: "demo",
		Kind:     analytics.EventKindConflictDetected,
		Actor:    "agent",
	})
	if err != nil {
		t.Fatalf("AppendEvent: %v", err)
	}
	if ev.ID == "" {
		t.Fatal("expected generated id")
	}
	page, err := repo.ListEvents(context.Background(), analytics.EventFilter{Scenario: "demo"})
	if err != nil {
		t.Fatalf("ListEvents: %v", err)
	}
	if len(page.Events) != 1 || page.Events[0].ID != ev.ID {
		t.Fatalf("unexpected page: %+v", page)
	}
}

func TestSQLiteRepository_StatsSuppressesLowN(t *testing.T) {
	d := newSchemaDB(t)
	repo := analytics.NewSQLiteRepository(d, schedule.System())
	for i := 0; i < 3; i++ {
		if _, err := repo.AppendEvent(context.Background(), analytics.Event{
			Scenario: "demo", Kind: analytics.EventKindVerdictProduced,
		}); err != nil {
			t.Fatalf("seed: %v", err)
		}
	}
	stats, err := repo.Stats(context.Background(), "demo")
	if err != nil {
		t.Fatalf("Stats: %v", err)
	}
	if !stats.VerdictSuccessRateSuppressed {
		t.Fatalf("expected suppressed, got %+v", stats)
	}
}

func TestSQLiteRepository_AppendOverride(t *testing.T) {
	d := newSchemaDB(t)
	repo := analytics.NewSQLiteRepository(d, schedule.System())
	got, err := repo.AppendOverride(context.Background(), analytics.Override{
		Scenario:      "demo",
		ChunkID:       "c1",
		VerdictDomain: "graph",
		ChosenDomain:  "manifest",
		Note:          "moved manually",
	})
	if err != nil {
		t.Fatalf("AppendOverride: %v", err)
	}
	if got.ID == "" {
		t.Fatal("expected generated id")
	}
}
