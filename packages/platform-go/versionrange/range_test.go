package versionrange

import "testing"

func TestSatisfies(t *testing.T) {
	tests := []struct {
		name       string
		version    string
		constraint string
		want       bool
	}{
		{name: "blank range preserves legacy behavior", version: "1.2.3", constraint: "", want: true},
		{name: "whitespace range preserves legacy behavior", version: "1.2.3", constraint: " \t ", want: true},
		{name: "wildcard", version: "1.2.3", constraint: "*", want: true},
		{name: "all operators", version: "1.4.2", constraint: ">=1.2.0 <2.0.0", want: true},
		{name: "upper bound", version: "2.0.0", constraint: ">=1.2.0 <2.0.0", want: false},
		{name: "caret", version: "1.2.0", constraint: "^1.2.0", want: true},
		{name: "tilde rejects next minor", version: "1.3.0", constraint: "~1.2.0", want: false},
		{name: "comma conjunction", version: "1.2.3", constraint: ">=1.2, <2", want: true},
		{name: "bare version remains exact", version: "1.2.3", constraint: "1.2", want: false},
		{name: "comma only", version: "1.2.3", constraint: ",", want: false},
		{name: "leading empty clause", version: "1.2.3", constraint: ",>=1.2", want: false},
		{name: "trailing empty clause", version: "1.2.3", constraint: ">=1.2,", want: false},
		{name: "repeated empty clauses", version: "1.2.3", constraint: ">=1.2,,<2", want: false},
		{name: "whitespace empty clause", version: "1.2.3", constraint: ", ,", want: false},
		{name: "malformed term", version: "1.2.3", constraint: ">=bad", want: false},
		{name: "invalid version", version: "invalid", constraint: ">=1.0.0", want: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := Satisfies(tc.version, tc.constraint); got != tc.want {
				t.Errorf("Satisfies(%q, %q) = %t, want %t", tc.version, tc.constraint, got, tc.want)
			}
		})
	}
}
