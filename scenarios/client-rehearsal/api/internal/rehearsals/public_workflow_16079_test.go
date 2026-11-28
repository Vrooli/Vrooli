package rehearsals

import (
	"crypto/sha256"
	"fmt"
	"strings"
	"testing"
)

func TestPublic16079RejectedBecauseItsPinnedSlackResourceIsInvalid(t *testing.T) {
	workflowJSON := readFixture(t, "public-workflows/16079.json")
	if got := fmt.Sprintf("%x", sha256.Sum256([]byte(workflowJSON))); got != "d4ecd7ed4abca9361d430d8d716db56790ceb01bca904eb74defdcec30943584" {
		t.Fatalf("sanitized public fixture changed without provenance update: %s", got)
	}
	_, _, _, err := normalizeWorkflow(workflowJSON)
	if err == nil || !strings.Contains(err.Error(), `Slack — Duplicate Warning`) || !strings.Contains(err.Error(), `resource="chat"`) {
		t.Fatalf("import must fail closed on the public graph's unsupported Slack V2.3 resource=chat node, got %v", err)
	}
}
