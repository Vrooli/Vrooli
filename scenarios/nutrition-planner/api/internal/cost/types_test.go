package cost

import (
	"testing"

	"nutrition-planner/internal/decimalx"
	"nutrition-planner/internal/money"
)

func mustDecimal(t *testing.T, value string) decimalx.Decimal {
	t.Helper()
	d, err := decimalx.Parse(value)
	if err != nil {
		t.Fatal(err)
	}
	return d
}

func TestCalculateSeparatesAllocatedAndCheckoutCost(t *testing.T) {
	price, err := money.New(400, "USD", 2)
	if err != nil {
		t.Fatal(err)
	}
	result, err := Calculate(Requirement{ItemID: "tofu", Amount: mustDecimal(t, "800"), Unit: "g"}, Stock{Amount: mustDecimal(t, "200"), Unit: "g"}, []Package{{ItemID: "tofu", Amount: mustDecimal(t, "400"), Unit: "g", Price: price}})
	if err != nil {
		t.Fatal(err)
	}
	if result.Missing.String() != "600" || result.PackagesToBuy.String() != "2" || result.CheckoutMinorCost.String() != "800" || result.AllocatedMinorCost.String() != "800" {
		t.Fatalf("result=%#v", result)
	}
}

func TestCalculateKeepsUnknownStockUnknown(t *testing.T) {
	price, _ := money.New(200, "USD", 2)
	result, err := Calculate(Requirement{ItemID: "sauce", Amount: mustDecimal(t, "80"), Unit: "g"}, Stock{Amount: decimalx.Unknown, Unit: "g"}, []Package{{ItemID: "sauce", Amount: mustDecimal(t, "200"), Unit: "g", Price: price}})
	if err != nil {
		t.Fatal(err)
	}
	if !result.Missing.IsUnknown() || !result.CheckoutMinorCost.IsUnknown() {
		t.Fatalf("unknown stock was fabricated: %#v", result)
	}
}

func TestCalculateManyKeepsMissingPricesOutOfTotals(t *testing.T) {
	known, _ := money.New(200, "USD", 2)
	result, err := CalculateMany([]Requirement{{ItemID: "rice", Amount: mustDecimal(t, "500"), Unit: "g"}, {ItemID: "unknown", Amount: mustDecimal(t, "1"), Unit: "count"}}, map[string]Stock{"rice": {Amount: mustDecimal(t, "0"), Unit: "g"}}, map[string][]Package{"rice": {{ItemID: "rice", Amount: mustDecimal(t, "500"), Unit: "g", Price: known}}, "unknown": nil})
	if err != nil {
		t.Fatal(err)
	}
	if result.Complete || result.CheckoutMinorCost.String() != "200" || len(result.Unknown) != 1 || result.Unknown[0] != "unknown" {
		t.Fatalf("result=%#v", result)
	}
}

func TestCalculateDoesNotRenderUnknownPriceAsZero(t *testing.T) {
	result, err := Calculate(Requirement{ItemID: "beans", Amount: mustDecimal(t, "1"), Unit: "count"}, Stock{Amount: mustDecimal(t, "0"), Unit: "count"}, []Package{{ItemID: "beans", Amount: mustDecimal(t, "1"), Unit: "count", Price: money.Unknown("USD", 2)}})
	if err != nil {
		t.Fatal(err)
	}
	if !result.AllocatedMinorCost.IsUnknown() || !result.CheckoutMinorCost.IsUnknown() || result.PriceKnown {
		t.Fatalf("unknown price was rendered as known: %#v", result)
	}
}
