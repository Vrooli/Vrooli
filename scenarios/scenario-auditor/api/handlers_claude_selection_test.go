package main

import (
	"reflect"
	"testing"
)

func TestSelectStoredFindingsKeepsOrderAndCollectsSelectedIDs(t *testing.T) {
	items := []StandardsViolation{{ID: "second"}, {ID: "first"}, {ID: ""}}
	selected, ids := selectStoredFindings(items, []string{"first", "second"}, func(item StandardsViolation) string { return item.ID })
	if !reflect.DeepEqual(ids, []string{"second", "first"}) {
		t.Fatalf("selected IDs = %v, want [second first]", ids)
	}
	if len(selected) != 2 || selected[0].ID != "second" || selected[1].ID != "first" {
		t.Fatalf("selected findings = %+v", selected)
	}
}

func TestSelectStoredFindingsCopiesAllAndSkipsBlankIDs(t *testing.T) {
	items := []StoredVulnerability{{ID: "a"}, {ID: ""}}
	selected, ids := selectStoredFindings(items, nil, func(item StoredVulnerability) string { return item.ID })
	if len(selected) != len(items) || &selected[0] == &items[0] {
		t.Fatal("all-findings selection did not return a copy")
	}
	if !reflect.DeepEqual(ids, []string{"a"}) {
		t.Fatalf("selected IDs = %v, want [a]", ids)
	}
}

func TestSelectStoredFindingsEmptyInput(t *testing.T) {
	selected, ids := selectStoredFindings([]StandardsViolation(nil), []string{"missing"}, func(item StandardsViolation) string { return item.ID })
	if selected != nil || ids != nil {
		t.Fatalf("empty selection = (%v, %v), want nil slices", selected, ids)
	}
}
