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
