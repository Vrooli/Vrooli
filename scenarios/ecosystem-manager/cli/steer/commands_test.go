package steer

import "testing"

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
