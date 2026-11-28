package textutil

import "testing"

func TestFirstNonBlankSkipsWhitespaceAndPreservesSelectedValue(t *testing.T) {
	if got := FirstNonBlank(" \t", " selected ", "later"); got != " selected " {
		t.Fatalf("FirstNonBlank() = %q, want original selected value", got)
	}
	if got := FirstNonBlank(" ", "\t"); got != "" {
		t.Fatalf("FirstNonBlank() = %q, want empty result", got)
	}
}
