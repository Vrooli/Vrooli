package steer

import (
	"reflect"
	"testing"
)

func TestBuildQuery(t *testing.T) {
	tests := []struct {
		skill, dim, want string
	}{
		{"", "", ""},
		{"lint-fix", "", "skill=lint-fix"},
		{"", "standards", "dimension=standards"},
		{"lint fix", "standards", "skill=lint+fix&dimension=standards"},
	}
	for _, tt := range tests {
		if got := buildQuery(tt.skill, tt.dim); got != tt.want {
			t.Fatalf("buildQuery(%q,%q) = %q, want %q", tt.skill, tt.dim, got, tt.want)
		}
	}
}

func TestSumCounts(t *testing.T) {
	if got := sumCounts(map[string]int{"standards": 2, "tests": 3}); got != 5 {
		t.Fatalf("sumCounts = %d, want 5", got)
	}
	if got := sumCounts(nil); got != 0 {
		t.Fatalf("sumCounts(nil) = %d, want 0", got)
	}
}

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
