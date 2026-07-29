package commerce

import (
	"errors"
	"testing"
)

func TestNormalizeStripeImportSelections(t *testing.T) {
	selections, validationErrors, err := NormalizeStripeImportSelections([]ImportPlanSelection{
		{PriceID: " price_alpha ", Action: " IMPORT "},
		{PriceID: "price_alpha", Action: "overwrite"},
		{PriceID: "", Action: "skip"},
		{PriceID: "price_beta", Action: "invalid"},
	})
	if err != nil {
		t.Fatalf("NormalizeStripeImportSelections() error = %v", err)
	}
	if len(selections) != 1 || selections[0].PriceID != "price_alpha" || selections[0].Action != "import" {
		t.Fatalf("normalized selections = %#v", selections)
	}
	if len(validationErrors) != 3 {
		t.Fatalf("validation errors = %#v", validationErrors)
	}
}

func TestNormalizeStripeImportSelectionsRejectsNoUsableSelections(t *testing.T) {
	_, validationErrors, err := NormalizeStripeImportSelections([]ImportPlanSelection{{PriceID: "", Action: "import"}})
	if !errors.Is(err, ErrStripeImportNoValidSelections) {
		t.Fatalf("error = %v, want ErrStripeImportNoValidSelections", err)
	}
	if len(validationErrors) != 1 {
		t.Fatalf("validation errors = %#v", validationErrors)
	}
}
