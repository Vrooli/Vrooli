package stringsx

import "testing"

func TestFirstNonEmptyTrimsAndSkipsBlankValues(t *testing.T) {
	if got := FirstNonEmpty(" ", "\n", "  selected  ", "later"); got != "selected" {
		t.Fatalf("FirstNonEmpty = %q", got)
	}
}
