package inventory

import (
	"context"
	"database/sql"
	"testing"
	"time"

	_ "modernc.org/sqlite"
	"nutrition-planner/internal/decimalx"
)

func TestSQLiteInventoryEventsAreWorkspaceScopedAndIdempotent(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err = db.Exec(Schema()); err != nil {
		t.Fatal(err)
	}
	repo := NewSQLiteRepository(db)
	amount, _ := decimalx.Parse("500")
	event := Event{ID: "purchase-1", Kind: Purchase, ItemID: "rice", Amount: amount, Unit: "g", CreatedAt: time.Date(2026, 9, 18, 0, 0, 0, 0, time.UTC)}
	if err := repo.Append(context.Background(), "w1", event); err != nil {
		t.Fatal(err)
	}
	if err := repo.Append(context.Background(), "w1", event); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.List(context.Background(), "w2"); err != nil {
		t.Fatal(err)
	}
	items, err := repo.List(context.Background(), "w1")
	if err != nil || len(items) != 1 || items[0].Amount.String() != "500" {
		t.Fatalf("items=%#v err=%v", items, err)
	}
	changed := event
	changed.Amount, _ = decimalx.Parse("600")
	if err := repo.Append(context.Background(), "w1", changed); err == nil {
		t.Fatal("expected idempotency conflict")
	}
}

func TestSQLiteBatchPreparationConsumesRawOnceAndUndoIsIdempotent(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err = db.Exec(Schema()); err != nil {
		t.Fatal(err)
	}
	repo := NewSQLiteRepository(db).(BatchRepository)
	yield, _ := decimalx.Parse("4")
	raw, _ := decimalx.Parse("800")
	batch, err := repo.PrepareBatch(context.Background(), "w1", "prep-1", "batch-1", "recipe-1", 3, yield, "serving", []Event{{ID: "prep-1:tofu", Kind: Preparation, ItemID: "tofu", Amount: raw, Unit: "g"}})
	if err != nil || batch.Available.String() != "4" {
		t.Fatalf("batch=%#v err=%v", batch, err)
	}
	portion, _ := decimalx.Parse("2")
	batch, err = repo.ConsumeBatchPortion(context.Background(), "w1", "eat-1", "batch-1", portion, "serving", "recipe-1", false)
	if err != nil || batch.Available.String() != "2" {
		t.Fatalf("portion=%#v err=%v", batch, err)
	}
	if _, err = repo.ConsumeBatchPortion(context.Background(), "w1", "eat-1", "batch-1", portion, "serving", "recipe-1", false); err != nil {
		t.Fatal(err)
	}
	batch, err = repo.ConsumeBatchPortion(context.Background(), "w1", "undo-eat-1", "batch-1", portion, "serving", "recipe-1", true)
	if err != nil || batch.Available.String() != "4" {
		t.Fatalf("undo=%#v err=%v", batch, err)
	}
	events, err := repo.List(context.Background(), "w1")
	if err != nil || len(events) != 3 {
		t.Fatalf("events=%#v err=%v", events, err)
	}
}

func TestSQLiteReceiptProposalsStageWithoutStockAndApplyOnce(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err = db.Exec(Schema()); err != nil {
		t.Fatal(err)
	}
	repo := NewSQLiteRepository(db).(ReceiptRepository)
	amount, _ := decimalx.Parse("2")
	proposal := ReceiptProposal{SourceID: "fixture", TransactionID: "tx-1", Description: "Organic rice", ItemID: "rice", Amount: amount, Unit: "kg", Price: "4.50"}
	first, err := repo.StageReceiptProposal(context.Background(), "w1", proposal)
	if err != nil {
		t.Fatal(err)
	}
	retry, err := repo.StageReceiptProposal(context.Background(), "w1", proposal)
	if err != nil {
		t.Fatal(err)
	}
	if first.ID != retry.ID || first.EventID != retry.EventID || retry.Status != ReceiptProposalPending {
		t.Fatalf("first=%#v retry=%#v", first, retry)
	}
	events, err := NewSQLiteRepository(db).List(context.Background(), "w1")
	if err != nil || len(events) != 0 {
		t.Fatalf("staging changed events=%#v err=%v", events, err)
	}
	applied, err := repo.ApplyReceiptProposal(context.Background(), "w1", first.ID)
	if err != nil {
		t.Fatal(err)
	}
	if applied.Status != ReceiptProposalApplied {
		t.Fatalf("applied=%#v", applied)
	}
	if _, err = repo.ApplyReceiptProposal(context.Background(), "w1", first.ID); err != nil {
		t.Fatal(err)
	}
	events, err = NewSQLiteRepository(db).List(context.Background(), "w1")
	if err != nil || len(events) != 1 || events[0].Kind != Purchase || events[0].Amount.String() != "2" {
		t.Fatalf("events=%#v err=%v", events, err)
	}
	proposals, err := repo.ListReceiptProposals(context.Background(), "w1")
	if err != nil || len(proposals) != 1 || proposals[0].Status != ReceiptProposalApplied {
		t.Fatalf("proposals=%#v err=%v", proposals, err)
	}
}

func TestSQLiteReceiptProposalSeparatesTransactionsAndLines(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err = db.Exec(Schema()); err != nil {
		t.Fatal(err)
	}
	repo := NewSQLiteRepository(db).(ReceiptRepository)
	amount, _ := decimalx.Parse("1")
	base := ReceiptProposal{SourceID: "fixture", TransactionID: "tx-1", Description: "Rice", ItemID: "rice", Amount: amount, Unit: "kg"}
	a, err := repo.StageReceiptProposal(context.Background(), "w1", base)
	if err != nil {
		t.Fatal(err)
	}
	b := base
	b.TransactionID = "tx-2"
	bp, err := repo.StageReceiptProposal(context.Background(), "w1", b)
	if err != nil {
		t.Fatal(err)
	}
	c := base
	c.Description = "Beans"
	c.ItemID = "beans"
	cp, err := repo.StageReceiptProposal(context.Background(), "w1", c)
	if err != nil {
		t.Fatal(err)
	}
	if a.ID == bp.ID || a.ID == cp.ID || bp.ID == cp.ID {
		t.Fatalf("expected distinct proposal IDs: %q %q %q", a.ID, bp.ID, cp.ID)
	}
}
