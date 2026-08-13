package plans_test

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	apidb "github.com/vrooli/api-core/database"

	localdb "data-backup-manager/internal/database"
	"data-backup-manager/internal/plans"
	testdb "data-backup-manager/internal/testutil/db"

	"github.com/vrooli/api-core/scheduletest"
)

// newPlansDB returns a fresh sqlite handle with system + plans schema applied.
func newPlansDB(t *testing.T) *sql.DB {
	t.Helper()
	d := testdb.NewSQLite(t)
	if err := apidb.EnsureSchemas(context.Background(), d,
		apidb.SchemaProviderFunc(localdb.SystemSchema),
		apidb.SchemaProviderFunc(plans.Schema),
	); err != nil {
		t.Fatalf("ensure schemas: %v", err)
	}
	return d
}

// TestSQLiteRepository_MembershipRoundTrip proves membership tables persist and
// reload: create plan with 2 targets + 2 destinations; reload; assert lists.
func TestSQLiteRepository_MembershipRoundTrip(t *testing.T) {
	ctx := context.Background()
	clk := scheduletest.New(time.Time{})
	repo := plans.NewSQLiteRepository(newPlansDB(t), clk)

	created, err := repo.Create(ctx, plans.Plan{
		Name:           "nightly",
		TargetIDs:      []string{"tgt-1", "tgt-2"},
		DestinationIDs: []string{"dst-a", "dst-b"},
		Schedule:       "24h",
		Enabled:        true,
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if created.ID == "" || created.CreatedAt.IsZero() {
		t.Fatalf("create did not populate id/timestamps: %+v", created)
	}

	// Reload via GetByID.
	reloaded, err := repo.GetByID(ctx, created.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if len(reloaded.TargetIDs) != 2 {
		t.Fatalf("target_ids = %v, want 2", reloaded.TargetIDs)
	}
	if len(reloaded.DestinationIDs) != 2 {
		t.Fatalf("destination_ids = %v, want 2", reloaded.DestinationIDs)
	}
	if !reloaded.Enabled {
		t.Fatal("enabled not persisted")
	}

	// A target can appear in two plans.
	_, err = repo.Create(ctx, plans.Plan{
		Name:           "weekly",
		TargetIDs:      []string{"tgt-1"}, // tgt-1 shared with plan 1
		DestinationIDs: []string{"dst-offsite"},
		Schedule:       "168h",
		Enabled:        true,
	})
	if err != nil {
		t.Fatalf("create second plan: %v", err)
	}

	all, err := repo.List(ctx, 100)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(all) != 2 {
		t.Fatalf("list count = %d, want 2", len(all))
	}

	// Delete first plan; membership rows should cascade.
	removed, err := repo.Delete(ctx, created.ID)
	if err != nil || !removed {
		t.Fatalf("delete: removed=%v err=%v", removed, err)
	}
	var notFound plans.ErrPlanNotFound
	if _, err := repo.GetByID(ctx, created.ID); !errors.As(err, &notFound) {
		t.Fatalf("expected ErrPlanNotFound after delete, got %v", err)
	}
}

// TestSQLiteRepository_UpdateReplacesMembers proves Update fully replaces
// membership lists.
func TestSQLiteRepository_UpdateReplacesMembers(t *testing.T) {
	ctx := context.Background()
	clk := scheduletest.New(time.Time{})
	repo := plans.NewSQLiteRepository(newPlansDB(t), clk)

	p, err := repo.Create(ctx, plans.Plan{
		Name:           "plan-a",
		TargetIDs:      []string{"tgt-1", "tgt-2"},
		DestinationIDs: []string{"dst-a"},
		Enabled:        true,
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	clk.Advance(time.Second)
	updated, err := repo.Update(ctx, plans.Plan{
		ID:             p.ID,
		Name:           "plan-a-updated",
		TargetIDs:      []string{"tgt-3"}, // replaced
		DestinationIDs: []string{"dst-b", "dst-c"},
		Enabled:        false,
	})
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if !updated.UpdatedAt.After(p.CreatedAt) {
		t.Fatalf("UpdatedAt not advanced: %v <= %v", updated.UpdatedAt, p.CreatedAt)
	}

	reloaded, err := repo.GetByID(ctx, p.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if len(reloaded.TargetIDs) != 1 || reloaded.TargetIDs[0] != "tgt-3" {
		t.Fatalf("target_ids after update = %v, want [tgt-3]", reloaded.TargetIDs)
	}
	if len(reloaded.DestinationIDs) != 2 {
		t.Fatalf("destination_ids after update = %v, want 2", reloaded.DestinationIDs)
	}
	if reloaded.Enabled {
		t.Fatal("enabled should be false after update")
	}
}
