package nutrition

import (
	"testing"

	"nutrition-planner/internal/decimalx"
)

func d(t *testing.T, value string) decimalx.Decimal {
	t.Helper()
	v, err := decimalx.Parse(value)
	if err != nil {
		t.Fatal(err)
	}
	return v
}

func TestAggregateScalesExactAndPreservesUnknown(t *testing.T) {
	result, err := Aggregate("protein", []Contribution{
		{NutrientID: "protein", Amount: d(t, "83"), Basis: d(t, "2"), Quantity: d(t, "1"), Unit: "serving", BasisUnit: "serving", Source: "recipe"},
		{NutrientID: "protein", Amount: decimalx.Unknown, Basis: d(t, "1"), Quantity: d(t, "1"), Unit: "serving", BasisUnit: "serving", Source: "unknown sauce"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Known.String() != "41.5" || result.Complete || len(result.Unresolved) != 1 {
		t.Fatalf("unexpected result: %+v", result)
	}
}

func TestEvaluateUnknownRules(t *testing.T) {
	partial := Result{NutrientID: "energy", Known: d(t, "400"), Complete: false}
	if got := Evaluate(partial, Target{NutrientID: "energy", Lower: d(t, "300"), Upper: d(t, "600")}); got.Status != Unknown {
		t.Fatalf("partial range status = %s", got.Status)
	}
	if got := Evaluate(partial, Target{NutrientID: "energy", Lower: decimalx.Unknown, Upper: d(t, "300")}); got.Status != Fail {
		t.Fatalf("known upper-bound violation status = %s", got.Status)
	}
	if got := Evaluate(partial, Target{NutrientID: "energy", Lower: d(t, "500"), Upper: decimalx.Unknown}); got.Status != Unknown {
		t.Fatalf("unresolved lower-bound status = %s", got.Status)
	}
}

func TestEvaluateCompleteRange(t *testing.T) {
	result := Result{NutrientID: "protein", Known: d(t, "41.5"), Complete: true}
	got := Evaluate(result, Target{NutrientID: "protein", Lower: d(t, "40"), Upper: d(t, "50")})
	if got.Status != Pass {
		t.Fatalf("status = %s, reason = %s", got.Status, got.Reason)
	}
}

func TestAggregateScopeUsesActualInsteadOfPlanAndKeepsPastUnknown(t *testing.T) {
	planned, _ := decimalx.Parse("50")
	actual, _ := decimalx.Parse("40")
	result, err := AggregateScope("protein", "expected_day", []Intake{
		{Date: "2026-09-18", NutrientID: "protein", Planned: planned, Actual: actual, Recorded: true, Past: true},
		{Date: "2026-09-17", NutrientID: "protein", Planned: planned, Actual: decimalx.Unknown, Past: true},
		{Date: "2026-09-19", NutrientID: "protein", Planned: planned, Actual: decimalx.Unknown},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Known.String() != "90" || result.Complete || len(result.Unresolved) != 1 {
		t.Fatalf("expected actual 40 + future plan 50 and one unresolved past intake, got %+v", result)
	}
}
