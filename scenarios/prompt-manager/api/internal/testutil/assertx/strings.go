package assertx

import (
	"strings"
	"testing"
)

func Contains(t testing.TB, got, wantSubstring, contract string) {
	t.Helper()
	if !strings.Contains(got, wantSubstring) {
		if contract == "" {
			contract = "string contract"
		}
		t.Fatalf("%s: expected %q to contain %q", contract, got, wantSubstring)
	}
}
