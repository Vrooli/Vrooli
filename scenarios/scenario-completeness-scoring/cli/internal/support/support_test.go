package support

import "testing"

func TestNormalizeInterspersedFlags(t *testing.T) {
	got := NormalizeInterspersedFlags([]string{"demo", "--json", "--verbose"})
	if len(got) != 3 || got[0] != "--json" || got[1] != "--verbose" || got[2] != "demo" {
		t.Fatalf("unexpected normalized args: %#v", got)
	}
}

func TestJSONLines(t *testing.T) {
	lines := JSONLines([]byte(`{"a":1,"b":{"c":2}}`))
	if len(lines) == 0 {
		t.Fatal("expected pretty JSON lines")
	}
}
