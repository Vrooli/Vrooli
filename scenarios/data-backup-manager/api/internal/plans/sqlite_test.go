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

	testdb "github.com/vrooli/api-core/databasetest"

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

func TestEnsureColumnsBootstrapsWeeklyDrillsOnlyForEnabledPlans(t *testing.T) {
	ctx := context.Background()
	db := newPlansDB(t)
	_, err := db.ExecContext(ctx, `INSERT INTO plans
		(id,name,schedule,keep_latest,keep_hourly,keep_daily,keep_weekly,keep_monthly,keep_annual,enabled,protection_tier,recovery_drill_schedule,created_at,updated_at)
		VALUES ('enabled-empty','enabled','24h',1,0,0,0,0,0,1,'full_primary','','2026-01-01T00:00:00Z','2026-01-01T00:00:00Z'),
		('disabled-empty','disabled','24h',1,0,0,0,0,0,0,'full_primary','','2026-01-01T00:00:00Z','2026-01-01T00:00:00Z'),
		('enabled-custom','custom','24h',1,0,0,0,0,0,1,'full_primary','72h','2026-01-01T00:00:00Z','2026-01-01T00:00:00Z')`)
	if err != nil {
		t.Fatal(err)
	}
	if err := plans.EnsureColumns(ctx, db); err != nil {
		t.Fatalf("bootstrap: %v", err)
	}
	if err := plans.EnsureColumns(ctx, db); err != nil {
		t.Fatalf("repeat bootstrap: %v", err)
	}
	for id, want := range map[string]string{"enabled-empty": plans.DefaultRecoveryDrillSchedule, "disabled-empty": "", "enabled-custom": "72h"} {
		var got string
		if err := db.QueryRowContext(ctx, `SELECT recovery_drill_schedule FROM plans WHERE id=?`, id).Scan(&got); err != nil {
			t.Fatal(err)
		}
		if got != want {
			t.Errorf("%s drill schedule=%q, want %q", id, got, want)
		}
	}
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
