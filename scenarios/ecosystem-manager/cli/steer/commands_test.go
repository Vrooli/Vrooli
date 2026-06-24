package steer

import (
	"reflect"
	"testing"
)

func TestOrNone(t *testing.T) {
	if orNone("") != "(no skill)" {
		t.Fatal("empty should render (no skill)")
	}
	if orNone("refactor") != "refactor" {
		t.Fatal("non-empty should pass through")
	}
}

func TestGapLinesIncludesKnownUncoveredTracking(t *testing.T) { // [REQ:EM-P1-006]
	got := gapLines([]CoverageDimensionGap{{
		Dimension:   "dependencies",
		Reason:      "no skill exists yet",
		TrackingRef: "knw-test",
	}})
	want := []string{"dependencies — known uncovered (knw-test): no skill exists yet"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("gapLines = %v, want %v", got, want)
	}
}

func TestCompactList(t *testing.T) {
	if got := compactList(nil, "none"); !reflect.DeepEqual(got, []string{"none"}) {
		t.Fatalf("compactList empty = %v", got)
	}
	if got := compactList([]string{"a", "b"}, "none"); !reflect.DeepEqual(got, []string{"a, b"}) {
		t.Fatalf("compactList values = %v", got)
	}
}
