package model

import "testing"

func TestModelConstants(t *testing.T) {
	if SchemaVersion <= 0 {
		t.Fatalf("SchemaVersion must be positive, got %d", SchemaVersion)
	}
	if SelfTarget == "" {
		t.Fatal("SelfTarget must not be empty")
	}
}

func TestFlowZeroValueIsUsable(t *testing.T) {
	f := Flow{}
	// Zero-value Flow should have empty collections — exercising the
	// struct shape catches accidental pointer fields that would panic
	// on naive iteration.
	if len(f.States) != 0 || len(f.Events) != 0 || len(f.Transitions) != 0 {
		t.Fatal("zero-value Flow should have empty slices")
	}
}
