package providers

import (
	"context"
	"errors"
	"testing"
)

func TestDisabledProviderIsExplicitAndSafe(t *testing.T) {
	d := Disabled{Name: "usda"}
	if d.Status().State != "not_configured" {
		t.Fatal(d.Status())
	}
	if _, err := d.Lookup(context.Background(), NutritionQuery{Query: "beans"}); !errors.Is(err, ErrNotConfigured) {
		t.Fatal(err)
	}
	if _, err := d.Search(context.Background(), NutritionSearchQuery{Query: "beans"}); !errors.Is(err, ErrNotConfigured) {
		t.Fatal(err)
	}
	if _, err := d.Observe(context.Background(), PriceQuery{ItemID: "beans"}); !errors.Is(err, ErrNotConfigured) {
		t.Fatal(err)
	}
	if _, err := d.Parse(context.Background(), ReceiptInput{Text: "beans"}); !errors.Is(err, ErrNotConfigured) {
		t.Fatal(err)
	}
}

func TestReceiptDedupUsesSourceTransactionAndNormalizedLine(t *testing.T) {
	result := ReceiptResult{SourceID: "receipt-provider", TransactionID: "txn-7"}
	first := ReceiptLine{ItemID: "rice", Description: "Long Grain Rice", Amount: "1", Unit: "package", Price: "4.50"}
	retry := ReceiptLine{ItemID: " RICE ", Description: "long grain rice", Amount: "1", Unit: "PACKAGE", Price: "4.50"}
	if err := ValidateReceiptIdentity(result); err != nil {
		t.Fatal(err)
	}
	if ReceiptDedupKey(result, first) != ReceiptDedupKey(result, retry) {
		t.Fatal("equivalent receipt lines were not deduplicated")
	}
	if err := ValidateReceiptIdentity(ReceiptResult{SourceID: "receipt-provider"}); err == nil {
		t.Fatal("missing transaction identity accepted")
	}
}
