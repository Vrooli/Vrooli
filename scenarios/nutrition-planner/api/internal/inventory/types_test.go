package inventory

import (
	"testing"

	"nutrition-planner/internal/decimalx"
)

func invDecimal(t *testing.T, value string) decimalx.Decimal {
	t.Helper()
	d, err := decimalx.Parse(value)
	if err != nil {
		t.Fatal(err)
	}
	return d
}

func TestPreparationConsumesRawOnceAndPortionsConsumeBatch(t *testing.T) {
	state := NewState()
	var err error
	state, err = Apply(state, Event{ID: "purchase-tofu", Kind: Purchase, ItemID: "tofu", Amount: invDecimal(t, "800"), Unit: "g"})
	if err != nil {
		t.Fatal(err)
	}
	state, err = Prepare(state, "prep-1", "batch-1", "recipe-1", 3, []Event{{Kind: Preparation, ItemID: "tofu", Amount: invDecimal(t, "800"), Unit: "g"}}, invDecimal(t, "4"), "serving")
	if err != nil {
		t.Fatal(err)
	}
	if state.OnHand["tofu"].String() != "0" || state.Batches["batch-1"].Available.String() != "4" {
		t.Fatalf("prepared state=%#v", state)
	}
	state, err = Apply(state, Event{ID: "eat-1", Kind: BatchPortion, BatchID: "batch-1", Amount: invDecimal(t, "2"), Unit: "serving"})
	if err != nil {
		t.Fatal(err)
	}
	if state.OnHand["tofu"].String() != "0" || state.Batches["batch-1"].Available.String() != "2" {
		t.Fatalf("portion state=%#v", state)
	}
}

func TestInventoryRetryIsIdempotentAndPayloadReuseConflicts(t *testing.T) {
	state := NewState()
	event := Event{ID: "purchase-1", Kind: Purchase, ItemID: "rice", Amount: invDecimal(t, "500"), Unit: "g"}
	var err error
	state, err = Apply(state, event)
	if err != nil {
		t.Fatal(err)
	}
	state, err = Apply(state, event)
	if err != nil || len(state.Events) != 1 || state.OnHand["rice"].String() != "500" {
		t.Fatalf("retry changed state: %#v err=%v", state, err)
	}
	_, err = Apply(state, Event{ID: "purchase-1", Kind: Purchase, ItemID: "rice", Amount: invDecimal(t, "600"), Unit: "g"})
	if err == nil {
		t.Fatal("expected changed-payload idempotency conflict")
	}
}

func TestUndoPortionRestoresBatchExactlyOnce(t *testing.T) {
	state := NewState()
	var err error
	state, err = Prepare(NewState(), "prep", "batch", "recipe", 1, nil, invDecimal(t, "2"), "serving")
	if err != nil {
		t.Fatal(err)
	}
	state, err = Apply(state, Event{ID: "eat", Kind: BatchPortion, BatchID: "batch", Amount: invDecimal(t, "1"), Unit: "serving"})
	if err != nil {
		t.Fatal(err)
	}
	state, err = Apply(state, Event{ID: "undo-eat", Kind: PortionUndo, BatchID: "batch", Amount: invDecimal(t, "1"), Unit: "serving"})
	if err != nil || state.Batches["batch"].Available.String() != "2" {
		t.Fatalf("undo state=%#v err=%v", state, err)
	}
}

func TestQualitativeAssertionsPreserveEvidenceWithoutNumericStockEffects(t *testing.T) {
	state := NewState()
	expression := "About half a bag"
	updated, err := Apply(state, Event{ID: "qual-1", Kind: Assertion, ItemID: "rice", Amount: decimalx.Unknown, AmountMode: AmountQualitative, Expression: expression})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := updated.OnHand["rice"]; ok {
		t.Fatalf("qualitative evidence created numeric stock: %#v", updated.OnHand)
	}
	if len(updated.Events) != 1 || updated.Events[0].Expression != expression || updated.Events[0].AmountMode != AmountQualitative {
		t.Fatalf("evidence not retained: %#v", updated.Events)
	}
	for _, mode := range []AmountMode{AmountOut, AmountUnknown} {
		updated, err = Apply(updated, Event{ID: string(mode), Kind: Assertion, ItemID: "rice", Amount: decimalx.Unknown, AmountMode: mode})
		if err != nil {
			t.Fatalf("%s assertion: %v", mode, err)
		}
	}
	if _, ok := updated.OnHand["rice"]; ok {
		t.Fatalf("non-numeric evidence changed stock: %#v", updated.OnHand)
	}
	if _, err := Apply(updated, Event{ID: "bad-consumption", Kind: Preparation, ItemID: "rice", Amount: decimalx.Unknown, AmountMode: AmountQualitative, Expression: expression}); err == nil {
		t.Fatal("qualitative evidence was accepted as consumption")
	}
}

func TestEstimatedAssertionsKeepNumericArithmetic(t *testing.T) {
	amount, _ := decimalx.Parse("2")
	state, err := Apply(NewState(), Event{ID: "est-1", Kind: Assertion, ItemID: "beans", Amount: amount, AmountMode: AmountEstimated, Unit: "cans"})
	if err != nil {
		t.Fatal(err)
	}
	if got := state.OnHand["beans"].String(); got != "2" {
		t.Fatalf("estimated stock=%s, want 2", got)
	}
}
