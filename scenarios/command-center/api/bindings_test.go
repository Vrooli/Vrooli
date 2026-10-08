package main

import "testing"

func TestValidateSignalBinding(t *testing.T) {
	rows := MetricEntry{ID: "funnel", Shape: "rows", Columns: map[string]ColumnSpec{"key": {Type: "string"}, "share": {Type: "number"}}}
	if err := ValidateSignalBinding(rows, "rows", []string{"key", "share"}, "funnel"); err != nil {
		t.Fatalf("valid rows binding rejected: %v", err)
	}
	if err := ValidateSignalBinding(rows, "rows", []string{"key", "value"}, "funnel"); err == nil {
		t.Fatal("missing required column accepted")
	}
	if err := ValidateSignalBinding(MetricEntry{ID: "count", Shape: "scalar"}, "rows", nil, "funnel"); err == nil {
		t.Fatal("shape mismatch accepted")
	}
}

func TestBundledSignalsAndRoomBindingsAreTyped(t *testing.T) {
	reg, err := LoadRegistry("../config/outcome-registry.json")
	if err != nil {
		t.Fatal(err)
	}
	if len(reg.Rooms) != 6 {
		t.Fatalf("rooms=%d, want six defaults", len(reg.Rooms))
	}
	for _, room := range reg.Rooms {
		for slot, signalID := range room.Bind {
			if slot == "" || !contains(room.MetricIDs, signalID) {
				t.Errorf("room %s bind %q points outside its signals: %q", room.ID, slot, signalID)
			}
		}
	}
}
