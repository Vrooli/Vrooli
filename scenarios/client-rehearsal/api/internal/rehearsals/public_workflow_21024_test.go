package rehearsals

import (
	"crypto/sha256"
	"fmt"
	"regexp"
	"strings"
	"testing"
)

func TestPublic21024ImportRetainsSafeCodeBoundaryWithoutRunCredit(t *testing.T) {
	workflowJSON := readFixture(t, "public-workflows/21024.json")
	if got := fmt.Sprintf("%x", sha256.Sum256([]byte(workflowJSON))); got != "00c63ca77cbe0631d55b19e8633a4ee704b0a09d48ec28dd151316e92db34cfa" {
		t.Fatalf("sanitized fixture hash changed without provenance update: %s", got)
	}
	if strings.Contains(workflowJSON, `"credentials"`) {
		t.Fatal("sanitized graph retained credential objects")
	}
	for _, match := range regexp.MustCompile(`https?://([^/\s"'<>]+)`).FindAllStringSubmatch(workflowJSON, -1) {
		if len(match) < 2 || !strings.HasSuffix(match[1], ".synthetic.invalid") {
			t.Fatalf("sanitized graph retained a non-synthetic URL host: %q", match)
		}
	}
	_, _, _, err := normalizeWorkflow(workflowJSON)
	if err == nil {
		t.Fatal("source workflow unexpectedly imports; add synthetic profiles and prove a complete simulator run before claiming credit")
	}
	if !strings.Contains(err.Error(), "only synthetic Date() or bounded Set() is supported") {
		t.Fatalf("expected unsupported Date and Set syntax to block this complete graph; got %v", err)
	}
	t.Logf("zero public run credit: full import stopped at generic safe-code boundary: %v", err)
}
