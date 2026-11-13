package rehearsals

import (
	"crypto/sha256"
	"fmt"
	"regexp"
	"strings"
	"testing"
)

func TestPublic17164ImportRetainsRSSBoundaryWithoutRunCredit(t *testing.T) {
	workflowJSON := readFixture(t, "public-workflows/17164.json")
	if got := fmt.Sprintf("%x", sha256.Sum256([]byte(workflowJSON))); got != "2788a548183fb6a9cf96f9c16a590ab7da4e87ae7850fac7c0dd014e6194f715" {
		t.Fatalf("sanitized fixture hash changed without provenance update: %s", got)
	}
	if strings.Contains(workflowJSON, `"credentials"`) || strings.Contains(workflowJSON, "googleapis.com") || strings.Contains(workflowJSON, "slack.com") {
		t.Fatal("sanitized graph retained credentials or live Google/Slack service hostnames")
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
	if !strings.Contains(err.Error(), "RSS Feed Read requires a simple current-item JSON path") {
		t.Fatalf("expected the generic RSS URL boundary to block this complete graph; got %v", err)
	}
	t.Logf("zero public run credit: full import stopped at generic RSS URL boundary: %v", err)
}
