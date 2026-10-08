package targets_test

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	apidb "github.com/vrooli/api-core/database"

	localdb "data-backup-manager/internal/database"
	"data-backup-manager/internal/sources"
	"data-backup-manager/internal/targets"

	db "github.com/vrooli/api-core/databasetest"

	"github.com/vrooli/api-core/scheduletest"
)

// newSchemaDB returns a fresh sqlite handle with system + targets schema
// applied, so repository tests get a real table without touching the central
// registry.
func newSchemaDB(t *testing.T) *sql.DB {
	t.Helper()
	d := db.NewSQLite(t)
	if err := apidb.EnsureSchemas(context.Background(), d,
		apidb.SchemaProviderFunc(localdb.SystemSchema),
		apidb.SchemaProviderFunc(targets.Schema),
	); err != nil {
		t.Fatalf("ensure schemas: %v", err)
	}
	return d
}

// TestSQLiteRepository_UpsertRoundTrip pins the SQL-level semantics the service
// relies on: insert, lookup by owner+name and by id, the (owner, name) unique
// constraint, update-in-place, list ordering, and delete.
func TestSQLiteRepository_UpsertRoundTrip(t *testing.T) {
	ctx := context.Background()
	clk := scheduletest.New(time.Time{})
	repo := targets.NewSQLiteRepository(newSchemaDB(t), clk)

	created, err := repo.Create(ctx, targets.Target{
		Owner: "prompt-manager", Name: "store",
		SourceKind: sources.KindFilesystem, Locator: "store/teams",
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if created.ID == "" || created.CreatedAt.IsZero() || !created.CreatedAt.Equal(created.UpdatedAt) {
		t.Fatalf("create did not populate id/timestamps: %+v", created)
	}

	gotKey, err := repo.GetByOwnerName(ctx, "prompt-manager", "store")
	if err != nil {
		t.Fatalf("get by owner/name: %v", err)
	}
	if gotKey.ID != created.ID || gotKey.Locator != "store/teams" {
		t.Fatalf("get by owner/name mismatch: %+v", gotKey)
	}

	gotID, err := repo.GetByID(ctx, created.ID)
	if err != nil {
		t.Fatalf("get by id: %v", err)
	}
	if gotID.Owner != "prompt-manager" || gotID.SourceKind != sources.KindFilesystem {
		t.Fatalf("get by id mismatch: %+v", gotID)
	}

	// Update in place — advance the clock so UpdatedAt moves past CreatedAt.
	clk.Advance(time.Second)
	created.Locator = "store/teams/x"
	updated, err := repo.Update(ctx, created)
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if !updated.UpdatedAt.After(updated.CreatedAt) {
		t.Fatalf("update did not advance UpdatedAt: %+v", updated)
	}
	reread, _ := repo.GetByID(ctx, created.ID)
	if reread.Locator != "store/teams/x" {
		t.Fatalf("update not persisted: %q", reread.Locator)
	}

	// List ordering + owner filter.
	if _, err := repo.Create(ctx, targets.Target{Owner: "swarm-manager", Name: "db", SourceKind: sources.KindSQLite, Locator: "d.db"}); err != nil {
		t.Fatalf("create second: %v", err)
	}
	all, err := repo.List(ctx, "", 100)
	if err != nil || len(all) != 2 {
		t.Fatalf("list all: n=%d err=%v", len(all), err)
	}
	if all[0].Owner != "prompt-manager" || all[1].Owner != "swarm-manager" {
		t.Fatalf("list not ordered by owner: %+v", all)
	}
	mine, _ := repo.List(ctx, "swarm-manager", 100)
	if len(mine) != 1 || mine[0].Name != "db" {
		t.Fatalf("owner filter wrong: %+v", mine)
	}

	// Delete.
	removed, err := repo.DeleteByOwnerName(ctx, "prompt-manager", "store")
	if err != nil || !removed {
		t.Fatalf("delete: removed=%v err=%v", removed, err)
	}
	var notFound targets.ErrTargetNotFound
	if _, err := repo.GetByID(ctx, created.ID); !errors.As(err, &notFound) {
		t.Fatalf("expected ErrTargetNotFound after delete, got %v", err)
	}
}

// TestSQLiteRepository_UniqueOwnerName proves the (owner, name) constraint
// rejects a duplicate insert (the service avoids this via upsert, but the
// constraint is the backstop).
func TestSQLiteRepository_UniqueOwnerName(t *testing.T) {
	ctx := context.Background()
	repo := targets.NewSQLiteRepository(newSchemaDB(t), scheduletest.New(time.Time{}))
	base := targets.Target{Owner: "o", Name: "n", SourceKind: sources.KindFilesystem, Locator: "p"}
	if _, err := repo.Create(ctx, base); err != nil {
		t.Fatalf("first create: %v", err)
	}
	if _, err := repo.Create(ctx, base); err == nil {
		t.Fatal("expected unique-constraint failure on duplicate (owner, name)")
	}
}
