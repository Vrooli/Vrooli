package main

import (
	"reflect"
	"testing"
)

type sampleFinding struct {
	id       string
	severity string
}

func TestFilterFindingsBySeverity(t *testing.T) {
	findings := []sampleFinding{
		{id: "high", severity: " High "},
		{id: "low", severity: "low"},
	}
	got := filterFindingsBySeverity(findings, map[string]struct{}{"high": {}}, func(item sampleFinding) string { return item.severity })
	if len(got) != 1 || got[0].id != "high" {
		t.Fatalf("filtered findings = %#v", got)
	}
	if got := filterFindingsBySeverity(findings, nil, func(item sampleFinding) string { return item.severity }); len(got) != len(findings) {
		t.Fatalf("empty filter should retain all findings, got %#v", got)
	}
}

func TestUniqueFindingIDs(t *testing.T) {
	findings := []sampleFinding{{id: " first "}, {id: "first"}, {id: ""}, {id: "second"}}
	got := uniqueFindingIDs(findings, func(item sampleFinding) string { return item.id })
	if want := []string{"first", "second"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("ids = %v, want %v", got, want)
	}
}

func TestOrderedUniqueFindings(t *testing.T) {
	findings := []sampleFinding{
		{id: "medium", severity: "medium"},
		{id: "critical", severity: "critical"},
		{id: "medium", severity: "low"},
		{id: "high", severity: "high"},
	}
	ids, severities := orderedUniqueFindings(findings, "high_first", func(item sampleFinding) string { return item.id }, func(item sampleFinding) string { return item.severity })
	if want := []string{"critical", "high", "medium"}; !reflect.DeepEqual(ids, want) {
		t.Fatalf("ordered ids = %v, want %v", ids, want)
	}
	if severities["medium"] != "medium" {
		t.Fatalf("duplicate ID severity = %q, want the first sorted occurrence", severities["medium"])
	}
	if len(severities) != 3 {
		t.Fatalf("severity map = %#v", severities)
	}
}
