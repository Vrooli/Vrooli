package cliutil

import "testing"

func TestSanitizePortOutput(t *testing.T) {
	cases := []struct {
		name   string
		input  string
		expect string
	}{
		{name: "numeric", input: "3000\n", expect: "3000"},
		{name: "with label", input: "api port: 4567", expect: "4567"},
		{name: "no digits", input: "not running", expect: ""},
		{name: "empty", input: "", expect: ""},
	}

	for _, tc := range cases {
		if got := sanitizePortOutput(tc.input); got != tc.expect {
			t.Fatalf("%s: expected %q, got %q", tc.name, tc.expect, got)
		}
	}
}
