package cliutil

import "testing"

func TestNearestString(t *testing.T) {
	options := []string{"records", "captures", "backlog", "create"}
	cases := []struct {
		candidate string
		maxDist   int
		want      string
	}{
		{"record", 2, "records"},
		{"recods", 2, "records"},
		{"captres", 2, "captures"},
		{"zzz", 2, ""},
		{"", 2, ""},
		{"records", 2, "records"},
	}
	for _, tc := range cases {
		if got := NearestString(tc.candidate, options, tc.maxDist); got != tc.want {
			t.Errorf("NearestString(%q) = %q, want %q", tc.candidate, got, tc.want)
		}
	}
	// Deterministic tie-break: equal distances resolve lexicographically.
	if got := NearestString("bat", []string{"cat", "bag"}, 2); got != "bag" {
		t.Errorf("tie-break = %q, want bag", got)
	}
}
