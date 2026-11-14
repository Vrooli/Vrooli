package aisearch

import (
	"reflect"
	"testing"
)

func TestFixedEvaluationSplitUsesOnlyOwnerAuthoredMembership(t *testing.T) {
	ids := []string{"a", "b", "c", "d", "e", "f", "g", "a"}
	heldout := map[string]bool{"g": true}
	excluded := map[string]bool{"f": true}
	tuningA, heldoutA, excludedA := FixedEvaluationSplit(ids, heldout, excluded)
	tuningB, heldoutB, excludedB := FixedEvaluationSplit([]string{"g", "f", "e", "d", "c", "b", "a"}, heldout, excluded)
	if !reflect.DeepEqual(tuningA, tuningB) || !reflect.DeepEqual(heldoutA, heldoutB) || !reflect.DeepEqual(excludedA, excludedB) {
		t.Fatalf("split depends on input ordering/duplicates: tuning %v/%v heldout %v/%v excluded %v/%v", tuningA, tuningB, heldoutA, heldoutB, excludedA, excludedB)
	}
	if !reflect.DeepEqual(tuningA, []string{"a", "b", "c", "d", "e"}) {
		t.Fatalf("untagged cases must remain tuning cases, got %v", tuningA)
	}
	if !reflect.DeepEqual(heldoutA, []string{"g"}) {
		t.Fatalf("only explicitly owner-heldout cases may enter heldout, got %v", heldoutA)
	}
	if !reflect.DeepEqual(excludedA, []string{"f"}) {
		t.Fatalf("excluded cases must enter neither fold, got %v", excludedA)
	}
}

func TestFixedEvaluationSplitDoesNotInventHeldoutFromUntaggedCases(t *testing.T) {
	ids := []string{"a", "b", "c", "d", "e", "f", "g", "h", "i", "j"}
	tuning, heldout, excluded := FixedEvaluationSplit(ids, nil, nil)
	if len(heldout) != 0 {
		t.Fatalf("untagged cases cannot be treated as heldout, got %v", heldout)
	}
	if !reflect.DeepEqual(tuning, ids) || len(excluded) != 0 {
		t.Fatalf("untagged cases must be tuning and no cases excluded, got tuning=%v excluded=%v", tuning, excluded)
	}
}
