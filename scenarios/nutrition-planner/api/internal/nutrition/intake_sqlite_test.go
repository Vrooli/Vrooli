package nutrition

import (
	"context"
	"database/sql"
	"testing"

	_ "modernc.org/sqlite"
	"nutrition-planner/internal/decimalx"
)

func TestSQLiteIntakeEventsAreImmutableAndIdempotent(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err = db.Exec(Schema()); err != nil {
		t.Fatal(err)
	}
	repo := NewSQLiteIntakeRepository(db)
	amount, _ := decimalx.Parse("41.5")
	event := IntakeEvent{ID: "eat-1", Date: "2026-09-18", RecipeID: "r1", RecipeRevision: 2, NutrientID: "protein", Amount: amount, Unit: "g", Reason: "actual portion"}
	if err = repo.Append(context.Background(), "w1", event); err != nil {
		t.Fatal(err)
	}
	if err = repo.Append(context.Background(), "w1", event); err != nil {
		t.Fatal(err)
	}
	items, err := repo.List(context.Background(), "w1")
	if err != nil || len(items) != 1 || items[0].Amount.String() != "41.5" {
		t.Fatalf("items=%#v err=%v", items, err)
	}
	changed := event
	changed.Amount, _ = decimalx.Parse("42")
	if err = repo.Append(context.Background(), "w1", changed); err == nil {
		t.Fatal("expected idempotency conflict")
	}
	correction := IntakeEvent{ID: "corr-1", Date: event.Date, NutrientID: event.NutrientID, Amount: amount, Unit: "g", CorrectionOf: event.ID, Reason: "user correction"}
	if err = repo.Append(context.Background(), "w1", correction); err != nil {
		t.Fatal(err)
	}
	if got, _ := repo.List(context.Background(), "w2"); len(got) != 0 {
		t.Fatal("events crossed workspace boundary")
	}
}
