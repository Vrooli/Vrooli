package recipe

import (
	"testing"

	"nutrition-planner/internal/decimalx"
)

func TestScaleQuantitiesDoesNotChangeCanonicalValues(t *testing.T) {
	one, _ := decimalx.Parse("1")
	two, _ := decimalx.Parse("2")
	quantity, _ := decimalx.Parse("3")
	out, err := ScaleQuantities([]ScalableQuantity{{ComponentID: "beans", Amount: quantity, Unit: "count"}}, one, two)
	if err != nil || out[0].Amount.String() != "6" || quantity.String() != "3" {
		t.Fatalf("out=%#v err=%v canonical=%s", out, err, quantity.String())
	}
}

func TestScaleQuantitiesRejectsFractionalDiscreteUnits(t *testing.T) {
	one, _ := decimalx.Parse("1")
	oneHalf, _ := decimalx.Parse("1.5")
	two, _ := decimalx.Parse("2")
	if _, err := ScaleQuantities([]ScalableQuantity{{ComponentID: "egg", Amount: one, Unit: "count", Discrete: true}}, two, oneHalf); err == nil {
		t.Fatal("fractional discrete quantity accepted")
	}
}
