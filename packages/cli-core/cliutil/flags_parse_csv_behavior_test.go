package cliutil

import (
	"reflect"
	"testing"
)

func TestParseCSVCommandFlagContract(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []string
	}{
		{name: "empty", input: "", want: nil},
		{name: "whitespace only", input: " \t\n", want: nil},
		{name: "trims drops blanks and preserves duplicates", input: " alpha, ,beta,alpha ", want: []string{"alpha", "beta", "alpha"}},
		{name: "trims unicode whitespace", input: "\u00a0alpha,\tbeta\u2003", want: []string{"alpha", "beta"}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := ParseCSV(test.input); !reflect.DeepEqual(got, test.want) {
				t.Fatalf("ParseCSV(%q) = %#v, want %#v", test.input, got, test.want)
			}
		})
	}
}
