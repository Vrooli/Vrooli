package rehearsals

import (
	"crypto/sha256"
	"fmt"
	"strings"
	"testing"
)

func TestPublic8011ComplexAgentGraphFailsClosedUntilItsServiceAndToolEdgesAreModeled(t *testing.T) {
	fixture := readFixture(t, "public-workflows/8011.json")
	if got := fmt.Sprintf("%x", sha256.Sum256([]byte(fixture))); got != "43f7814e7d9a0181002b335cffdbb17162f2935d1fa6b97269efe76436591ffb" {
		t.Fatalf("sanitized public fixture hash changed: %s", got)
	}
	_, _, _, err := normalizeWorkflow(fixture)
	if err == nil || !strings.Contains(err.Error(), "unsupported executable node") {
		t.Fatalf("complex public Agent graph must fail closed while its service/tool semantics are unmodeled, got %v", err)
	}
}
