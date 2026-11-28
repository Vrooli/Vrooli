package workspace

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/vrooli/api-core/identity"
	"github.com/vrooli/api-core/schedule"
	_ "modernc.org/sqlite"
)

func workspaceTestContext(subject string) context.Context {
	return identity.WithPrincipal(context.Background(), identity.Principal{Kind: identity.ActorHuman, Subject: subject, Verified: true})
}

func workspaceTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", "file:workspace-ownership-"+t.Name()+"?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if _, err := db.Exec(Schema()); err != nil {
		t.Fatal(err)
	}
	for _, query := range []string{
		`INSERT INTO planning_profiles VALUES ('default','legacy-zone','sunday',300,20,30,8,111)`,
		`INSERT INTO availability_rules VALUES ('legacy-window',2,500,600,'UTC','','',0,8)`,
		`INSERT INTO availability_exceptions VALUES ('legacy-exception','2030-01-02',700,800,'protected','legacy',8)`,
	} {
		if _, err := db.Exec(query); err != nil {
			t.Fatal(err)
		}
	}
	return db
}

func TestSQLiteWorkspaceProfilesAndAvailabilityAreIsolatedByVerifiedOwner(t *testing.T) {
	db := workspaceTestDB(t)
	repo := NewSQLiteRepository(db, schedule.System())
	ownerA, ownerB := workspaceTestContext("owner-a"), workspaceTestContext("owner-b")

	a, err := repo.Get(ownerA)
	if err != nil {
		t.Fatal(err)
	}
	b, err := repo.Get(ownerB)
	if err != nil {
		t.Fatal(err)
	}
	if a.ID != profileID || b.ID != profileID || a.Timezone != "UTC" || b.Timezone != "UTC" || a.Revision != 1 || b.Revision != 1 {
		t.Fatalf("profiles should start from isolated defaults: A=%#v B=%#v", a, b)
	}
	updated, err := repo.Update(ownerA, UpdateInput{Timezone: "America/New_York", WeekStart: "monday", DailyCapacityMinutes: 420, ReserveMinutes: 60, FocusSessionMinutes: 45, ExpectedRevision: a.Revision})
	if err != nil {
		t.Fatal(err)
	}
	if updated.ID != a.ID || updated.Revision != 2 || updated.Timezone != "America/New_York" {
		t.Fatalf("updated profile=%#v", updated)
	}
	if _, err := repo.Update(ownerA, UpdateInput{Timezone: "UTC", WeekStart: "monday", DailyCapacityMinutes: 480, ReserveMinutes: 60, FocusSessionMinutes: 45, ExpectedRevision: a.Revision}); !errors.As(err, new(ErrRevisionConflict)) {
		t.Fatalf("stale profile update error=%v", err)
	}
	if got, err := repo.Get(ownerB); err != nil || got.Revision != 1 || got.Timezone != "UTC" {
		t.Fatalf("owner B changed after owner A update: profile=%#v err=%v", got, err)
	}
	initialAAvailability, err := repo.ListAvailability(ownerA)
	if err != nil {
		t.Fatal(err)
	}
	if len(initialAAvailability.Windows) != 0 || len(initialAAvailability.Exceptions) != 0 {
		t.Fatalf("owner A received legacy availability: %#v", initialAAvailability)
	}

	availability, err := repo.ReplaceAvailability(ownerA, AvailabilityInput{ExpectedRevision: updated.Revision,
		Windows:    []AvailabilityWindow{{ID: "window", Weekday: 1, StartMinute: 540, EndMinute: 600, Timezone: "UTC"}},
		Exceptions: []AvailabilityException{{ID: "exception", Date: "2030-01-03", StartMinute: 700, EndMinute: 760, Kind: "protected", Reason: "A", Revision: 1}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if availability.Revision != 3 || len(availability.Windows) != 1 || len(availability.Exceptions) != 1 || availability.Windows[0].ID != "window" {
		t.Fatalf("owner A availability=%#v", availability)
	}
	bAvailability, err := repo.ListAvailability(ownerB)
	if err != nil {
		t.Fatal(err)
	}
	if bAvailability.Revision != 1 || len(bAvailability.Windows) != 0 || len(bAvailability.Exceptions) != 0 {
		t.Fatalf("owner B saw owner A or legacy availability: %#v", bAvailability)
	}
	bAvailability, err = repo.ReplaceAvailability(ownerB, AvailabilityInput{ExpectedRevision: b.Revision,
		Windows: []AvailabilityWindow{{ID: "window", Weekday: 2, StartMinute: 600, EndMinute: 660, Timezone: "America/Los_Angeles"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if bAvailability.Revision != 2 || len(bAvailability.Windows) != 1 || bAvailability.Windows[0].Timezone != "America/Los_Angeles" {
		t.Fatalf("owner B availability=%#v", bAvailability)
	}
	aAvailability, err := repo.ListAvailability(ownerA)
	if err != nil {
		t.Fatal(err)
	}
	if aAvailability.Revision != 3 || len(aAvailability.Windows) != 1 || aAvailability.Windows[0].Timezone != "UTC" || len(aAvailability.Exceptions) != 1 {
		t.Fatalf("owner B availability update changed owner A: %#v", aAvailability)
	}
	if _, err := repo.ReplaceAvailability(ownerA, AvailabilityInput{ExpectedRevision: 1, Windows: []AvailabilityWindow{{ID: "bad", Weekday: 1, StartMinute: 540, EndMinute: 600, Timezone: "UTC"}}}); !errors.As(err, new(ErrRevisionConflict)) {
		t.Fatalf("stale availability replacement error=%v", err)
	}

	var legacyID, legacyZone, legacyWeekStart string
	var legacyCapacity, legacyReserve, legacyFocus, legacyProfileRevision int
	var legacyUpdated int64
	if err := db.QueryRow(`SELECT id,timezone,week_start,daily_capacity_minutes,reserve_minutes,focus_session_minutes,revision,updated_at FROM planning_profiles WHERE id='default'`).Scan(&legacyID, &legacyZone, &legacyWeekStart, &legacyCapacity, &legacyReserve, &legacyFocus, &legacyProfileRevision, &legacyUpdated); err != nil {
		t.Fatal(err)
	}
	if legacyID != "default" || legacyZone != "legacy-zone" || legacyWeekStart != "sunday" || legacyCapacity != 300 || legacyReserve != 20 || legacyFocus != 30 || legacyProfileRevision != 8 || legacyUpdated != 111 {
		t.Fatalf("legacy profile changed: %q %q %q %d %d %d %d %d", legacyID, legacyZone, legacyWeekStart, legacyCapacity, legacyReserve, legacyFocus, legacyProfileRevision, legacyUpdated)
	}
	var ruleID, ruleZone, ruleStartDate, ruleEndDate string
	var ruleWeekday, ruleStart, ruleEnd, rulePriority, ruleRevision int
	if err := db.QueryRow(`SELECT id,weekday,start_minute,end_minute,timezone,effective_start_date,effective_end_date,priority,revision FROM availability_rules WHERE id='legacy-window'`).Scan(&ruleID, &ruleWeekday, &ruleStart, &ruleEnd, &ruleZone, &ruleStartDate, &ruleEndDate, &rulePriority, &ruleRevision); err != nil {
		t.Fatal(err)
	}
	if ruleID != "legacy-window" || ruleWeekday != 2 || ruleStart != 500 || ruleEnd != 600 || ruleZone != "UTC" || ruleStartDate != "" || ruleEndDate != "" || rulePriority != 0 || ruleRevision != 8 {
		t.Fatalf("legacy availability rule changed: %#v", []any{ruleID, ruleWeekday, ruleStart, ruleEnd, ruleZone, ruleStartDate, ruleEndDate, rulePriority, ruleRevision})
	}
	var exceptionID, exceptionDate, exceptionKind, exceptionReason string
	var exceptionStart, exceptionEnd, exceptionRevision int
	if err := db.QueryRow(`SELECT id,date,start_minute,end_minute,kind,reason,revision FROM availability_exceptions WHERE id='legacy-exception'`).Scan(&exceptionID, &exceptionDate, &exceptionStart, &exceptionEnd, &exceptionKind, &exceptionReason, &exceptionRevision); err != nil {
		t.Fatal(err)
	}
	if exceptionID != "legacy-exception" || exceptionDate != "2030-01-02" || exceptionStart != 700 || exceptionEnd != 800 || exceptionKind != "protected" || exceptionReason != "legacy" || exceptionRevision != 8 {
		t.Fatalf("legacy availability exception changed: %#v", []any{exceptionID, exceptionDate, exceptionStart, exceptionEnd, exceptionKind, exceptionReason, exceptionRevision})
	}
}

func TestReplaceAvailabilityRollsBackAfterInsertFailure(t *testing.T) {
	db := workspaceTestDB(t)
	ctx := workspaceTestContext("owner")
	repo := NewSQLiteRepository(db, schedule.System())
	prior, err := repo.ReplaceAvailability(ctx, AvailabilityInput{ExpectedRevision: 1,
		Windows:    []AvailabilityWindow{{ID: "prior", Weekday: 1, StartMinute: 540, EndMinute: 600, Timezone: "UTC"}},
		Exceptions: []AvailabilityException{{ID: "prior-exception", Date: "2030-01-03", StartMinute: 700, EndMinute: 760, Kind: "protected", Reason: "prior"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := repo.ReplaceAvailability(ctx, AvailabilityInput{ExpectedRevision: prior.Revision, Windows: []AvailabilityWindow{
		{ID: "partial", Weekday: 2, StartMinute: 600, EndMinute: 660, Timezone: "UTC"},
		{ID: "partial", Weekday: 3, StartMinute: 660, EndMinute: 720, Timezone: "UTC"},
	}}); err == nil {
		t.Fatal("duplicate availability key unexpectedly succeeded")
	}
	got, err := repo.ListAvailability(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if got.Revision != prior.Revision || len(got.Windows) != 1 || got.Windows[0].ID != "prior" || len(got.Exceptions) != 1 || got.Exceptions[0].ID != "prior-exception" {
		t.Fatalf("failed replacement changed prior availability or revision: %#v", got)
	}
}

type revisionRacingDB struct {
	*sql.DB
	fired bool
}

func (db *revisionRacingDB) BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error) {
	if !db.fired {
		db.fired = true
		if _, err := db.ExecContext(ctx, `UPDATE actor_planning_profiles SET timezone='Europe/London',revision=revision+1,updated_at=222 WHERE owner_subject='owner' AND id='default'`); err != nil {
			return nil, err
		}
	}
	return db.DB.BeginTx(ctx, opts)
}

func TestReplaceAvailabilityRollsBackWhenRevisionChangesAfterRead(t *testing.T) {
	db := workspaceTestDB(t)
	ctx := workspaceTestContext("owner")
	repo := NewSQLiteRepository(db, schedule.System())
	prior, err := repo.ReplaceAvailability(ctx, AvailabilityInput{ExpectedRevision: 1,
		Windows: []AvailabilityWindow{{ID: "prior", Weekday: 1, StartMinute: 540, EndMinute: 600, Timezone: "UTC"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	racingRepo := NewSQLiteRepository(&revisionRacingDB{DB: db}, schedule.System())
	if _, err := racingRepo.ReplaceAvailability(ctx, AvailabilityInput{ExpectedRevision: prior.Revision,
		Windows: []AvailabilityWindow{{ID: "replacement", Weekday: 2, StartMinute: 600, EndMinute: 660, Timezone: "UTC"}},
	}); !errors.As(err, new(ErrRevisionConflict)) {
		t.Fatalf("late revision change error=%v", err)
	}
	profile, err := repo.Get(ctx)
	if err != nil {
		t.Fatal(err)
	}
	got, err := repo.ListAvailability(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if profile.Revision != prior.Revision+1 || profile.Timezone != "Europe/London" {
		t.Fatalf("winning profile revision was not preserved: %#v", profile)
	}
	if got.Revision != profile.Revision || len(got.Windows) != 1 || got.Windows[0].ID != "prior" {
		t.Fatalf("late conflict changed prior availability: %#v", got)
	}
}

func TestSQLiteWorkspaceRequiresVerifiedOwnerAndNeverFallsBackToLegacy(t *testing.T) {
	db := workspaceTestDB(t)
	repo := NewSQLiteRepository(db, schedule.System())
	unverified := identity.WithPrincipal(context.Background(), identity.Principal{Kind: identity.ActorHuman, Subject: "claimed", Verified: false})
	for name, ctx := range map[string]context.Context{"missing": context.Background(), "unverified": unverified, "blank subject": identity.WithPrincipal(context.Background(), identity.Principal{Kind: identity.ActorHuman, Verified: true})} {
		t.Run(name, func(t *testing.T) {
			if _, err := repo.Get(ctx); !errors.Is(err, ErrUnverifiedWorkspacePrincipal) {
				t.Fatalf("Get error=%v", err)
			}
			if _, err := repo.Update(ctx, UpdateInput{ExpectedRevision: 8}); !errors.Is(err, ErrUnverifiedWorkspacePrincipal) {
				t.Fatalf("Update error=%v", err)
			}
			if _, err := repo.ListAvailability(ctx); !errors.Is(err, ErrUnverifiedWorkspacePrincipal) {
				t.Fatalf("ListAvailability error=%v", err)
			}
			if _, err := repo.ReplaceAvailability(ctx, AvailabilityInput{ExpectedRevision: 8, Windows: []AvailabilityWindow{{ID: "x"}}}); !errors.Is(err, ErrUnverifiedWorkspacePrincipal) {
				t.Fatalf("ReplaceAvailability error=%v", err)
			}
		})
	}
	var count int
	if err := db.QueryRow(`SELECT count(*) FROM actor_planning_profiles`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("invalid identity created %d actor profiles", count)
	}
	var legacyZone string
	if err := db.QueryRow(`SELECT timezone FROM planning_profiles WHERE id='default'`).Scan(&legacyZone); err != nil {
		t.Fatal(err)
	}
	if legacyZone != "legacy-zone" {
		t.Fatalf("legacy singleton was exposed or changed: %q", legacyZone)
	}
}
