package agentharness

import "testing"

func TestSemverMajorAcceptsVersionCommandOutput(t *testing.T) {
	cases := map[string]string{
		"1.19.6":               "1",
		"opencode v2.3.2":      "2",
		"claude 1.2.3 (build)": "1",
		"no version":           "",
	}
	for input, want := range cases {
		if got := semverMajor(input); got != want {
			t.Fatalf("semverMajor(%q) = %q, want %q", input, got, want)
		}
	}
}
