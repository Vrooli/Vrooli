package componenttests

import (
	"strings"
	"testing"
)

func TestFixtureNameIsStable(t *testing.T) {
	first := fixtureName("positive")
	second := fixtureName("positive")
	if first != second {
		t.Fatalf("fixtureName is not stable: %q != %q", first, second)
	}
	if !strings.HasPrefix(first, generatedFixtureNamePrefix) {
		t.Fatalf("fixtureName = %q, want prefix %q", first, generatedFixtureNamePrefix)
	}
	if strings.ContainsAny(first, "0123456789") {
		t.Fatalf("fixtureName = %q contains a timestamp-like numeric suffix", first)
	}
}
