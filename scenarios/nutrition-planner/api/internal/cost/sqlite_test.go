package cost

import (
	"context"
	"database/sql"
	"testing"
	"time"

	_ "modernc.org/sqlite"
	"nutrition-planner/internal/decimalx"
	"nutrition-planner/internal/money"
)

func TestSQLitePriceObservationsPreserveHistory(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err = db.Exec(Schema()); err != nil {
		t.Fatal(err)
	}
	repo := NewSQLiteRepository(db)
	amount, _ := decimalx.Parse("400")
	price1, _ := money.New(400, "USD", 2)
	price2, _ := money.New(500, "USD", 2)
	base := Observation{WorkspaceID: "w", ItemID: "tofu", PackageAmount: amount, PackageUnit: "g", Price: price1, ObservedAt: time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC), Source: "manual"}
	if _, err := repo.Create(context.Background(), base); err != nil {
		t.Fatal(err)
	}
	base.Price = price2
	base.ObservedAt = base.ObservedAt.Add(24 * time.Hour)
	if _, err := repo.Create(context.Background(), base); err != nil {
		t.Fatal(err)
	}
	items, err := repo.List(context.Background(), "w", "tofu")
	if err != nil || len(items) != 2 || items[0].Price.Minor != 500 || items[1].Price.Minor != 400 {
		t.Fatalf("items=%#v err=%v", items, err)
	}
}
