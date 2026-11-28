package shopping

import (
	"context"
	"database/sql"
	"testing"

	"github.com/vrooli/api-core/schedule"
	_ "modernc.org/sqlite"
	"nutrition-planner/internal/decimalx"
	"nutrition-planner/internal/inventory"
)

func TestSQLitePurchaseReviewIsAtomicIdempotentAndDistinctFromHaveThis(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	ctx := context.Background()
	// Simulate an existing populated store before applying the additive schema.
	if _, err = db.Exec(`CREATE TABLE shopping_checks(workspace_id TEXT NOT NULL,line_key TEXT NOT NULL,checked INTEGER NOT NULL DEFAULT 0,updated_at TEXT NOT NULL,PRIMARY KEY(workspace_id,line_key)); INSERT INTO shopping_checks VALUES('w','ingredient:rice',1,'before');`); err != nil {
		t.Fatal(err)
	}
	if _, err = db.Exec(Schema() + "\n" + inventory.Schema()); err != nil {
		t.Fatal(err)
	}
	repo := NewSQLiteRepository(db, schedule.System())
	checked, err := repo.Checked(ctx, "w")
	if err != nil || !checked["ingredient:rice"] {
		t.Fatalf("existing checklist row lost: %#v err=%v", checked, err)
	}
	if err = repo.SetHaveThis(ctx, "w", "ingredient:rice", true); err != nil {
		t.Fatal(err)
	}
	have, err := repo.HaveThis(ctx, "w")
	if err != nil || !have["ingredient:rice"] {
		t.Fatalf("qualitative assertion=%#v err=%v", have, err)
	}
	events, err := inventory.NewSQLiteRepository(db).List(ctx, "w")
	if err != nil || len(events) != 0 {
		t.Fatalf("Have this must not invent stock: events=%#v err=%v", events, err)
	}

	amount, _ := decimalx.Parse("40")
	rows := []PurchaseLine{{Key: "ingredient:rice", ItemID: "rice", Amount: amount, Unit: "g", Price: "1.20"}, {Key: "ingredient:beans", ItemID: "beans", Unit: "can", Price: "", Omitted: true}}
	if err = repo.ConfirmPurchases(ctx, "w", "trip-1", rows); err != nil {
		t.Fatal(err)
	}
	if err = repo.ConfirmPurchases(ctx, "w", "trip-1", rows); err != nil {
		t.Fatalf("identical retry: %v", err)
	}
	events, err = inventory.NewSQLiteRepository(db).List(ctx, "w")
	if err != nil || len(events) != 1 || events[0].Kind != inventory.Purchase || events[0].Amount.String() != "40" {
		t.Fatalf("purchase events=%#v err=%v", events, err)
	}
	var storedRows int
	if err = db.QueryRow(`SELECT count(*) FROM shopping_purchase_review_lines WHERE workspace_id='w' AND review_id='trip-1'`).Scan(&storedRows); err != nil || storedRows != 2 {
		t.Fatalf("review rows=%d err=%v", storedRows, err)
	}
	changed := append([]PurchaseLine(nil), rows...)
	changed[0].Price = "1.25"
	if err = repo.ConfirmPurchases(ctx, "w", "trip-1", changed); err == nil {
		t.Fatal("changed retry should conflict")
	}
	var eatingEvents int
	if err = db.QueryRow(`SELECT count(*) FROM inventory_events WHERE kind IN ('meal_intake','batch_portion')`).Scan(&eatingEvents); err != nil || eatingEvents != 0 {
		t.Fatalf("purchase recorded eating: count=%d err=%v", eatingEvents, err)
	}
}

func TestSQLitePurchaseReviewRollsBackAllRowsOnFailure(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err = db.Exec(Schema() + "\n" + inventory.Schema()); err != nil {
		t.Fatal(err)
	}
	if _, err = db.Exec(`CREATE TRIGGER fail_second BEFORE INSERT ON inventory_events BEGIN SELECT RAISE(ABORT,'injected failure'); END;`); err != nil {
		t.Fatal(err)
	}
	a, _ := decimalx.Parse("10")
	b, _ := decimalx.Parse("20")
	repo := NewSQLiteRepository(db, schedule.System())
	err = repo.ConfirmPurchases(context.Background(), "w", "trip", []PurchaseLine{{Key: "a", ItemID: "a", Amount: a, Unit: "g"}, {Key: "b", ItemID: "b", Amount: b, Unit: "g"}})
	if err == nil {
		t.Fatal("expected injected write failure")
	}
	var reviews, lines, events int
	if err = db.QueryRow(`SELECT (SELECT count(*) FROM shopping_purchase_reviews),(SELECT count(*) FROM shopping_purchase_review_lines),(SELECT count(*) FROM inventory_events)`).Scan(&reviews, &lines, &events); err != nil {
		t.Fatal(err)
	}
	if reviews != 0 || lines != 0 || events != 0 {
		t.Fatalf("partial review escaped rollback: reviews=%d lines=%d events=%d", reviews, lines, events)
	}
}
