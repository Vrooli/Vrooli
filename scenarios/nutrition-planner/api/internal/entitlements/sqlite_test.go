package entitlements

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	_ "modernc.org/sqlite"
)

func entitlementDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(Schema()); err != nil {
		db.Close()
		t.Fatal(err)
	}
	return db
}

func TestSQLiteEntitlementsDefaultEnabledAndReservationIsIdempotent(t *testing.T) {
	db := entitlementDB(t)
	defer db.Close()
	repo := NewSQLiteRepository(db)
	ctx := context.Background()
	state, err := repo.Get(ctx, "w1")
	if err != nil || !state.OptionalCompute || state.Used != 0 {
		t.Fatalf("state=%#v err=%v", state, err)
	}
	state, err = repo.ReserveOptional(ctx, "w1", "job-1", 3)
	if err != nil || state.Used != 3 {
		t.Fatalf("reserved=%#v err=%v", state, err)
	}
	state, err = repo.ReserveOptional(ctx, "w1", "job-1", 3)
	if err != nil || state.Used != 3 {
		t.Fatalf("retry=%#v err=%v", state, err)
	}
	if _, err = repo.ReserveOptional(ctx, "w1", "job-1", 4); err == nil {
		t.Fatal("expected operation payload conflict")
	}
}

func TestSQLiteEntitlementsServerBudgetAndOutOfOrderEvents(t *testing.T) {
	db := entitlementDB(t)
	defer db.Close()
	repo := NewSQLiteRepository(db)
	ctx := context.Background()
	if _, err := repo.ApplyEvent(ctx, "w1", Event{ID: "plan-2", Version: 2, OptionalCompute: false, MonthlyLimit: 2}); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.ReserveOptional(ctx, "w1", "job-1", 1); err == nil {
		t.Fatal("expected disabled optional compute refusal")
	}
	state, err := repo.ApplyEvent(ctx, "w1", Event{ID: "plan-1", Version: 1, OptionalCompute: true, MonthlyLimit: 100})
	if err != nil {
		t.Fatal(err)
	}
	if state.Version != 2 || state.OptionalCompute {
		t.Fatalf("old event rewrote state: %#v", state)
	}
	_, err = repo.ReserveOptional(ctx, "w1", "job-2", 3)
	var budget BudgetError
	if !errors.As(err, &budget) || budget.Remaining != 2 {
		t.Fatalf("err=%v, want remaining-budget message", err)
	}
}
