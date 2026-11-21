package rehearsals

import (
	"crypto/sha256"
	"fmt"
	"strings"
	"testing"
)

// This records a generic qualification attempt. Community-node incompatibility
// earns no public workflow run credit; it is left visible for central interpreter work.
func TestPublic7950DiscordGitHubGeminiQualificationAttempt(t *testing.T) {
	raw := readFixture(t, "public-workflows/7950.json")
	if got := fmt.Sprintf("%x", sha256Bytes(raw)); got != "408b57fb4df5e0f70f9efc4f7b342e2fbbc4c93b3122a0194985cb8590be88fb" {
		t.Fatalf("sanitized fixture changed without provenance update: %s", got)
	}
	if strings.Contains(raw, "n8n Blog Master") || strings.Contains(raw, "praneel7015") || strings.Contains(raw, "gravatar.com") {
		t.Fatal("fixture retained source prompt or author profile metadata")
	}
	workflow, _, _, err := normalizeWorkflow(raw)
	if err != nil {
		t.Logf("no run credit: generic import rejects unchanged Discord community-trigger graph: %v", err)
		return
	}
	if len(workflow.Nodes) != 10 || len(workflow.EntryNodeIDs) != 1 {
		t.Fatalf("unexpected sanitized graph shape: nodes=%d entries=%v", len(workflow.Nodes), workflow.EntryNodeIDs)
	}
	t.Log("import succeeded; full synthetic execution profile remains required before run credit")
}

func sha256Bytes(value string) []byte {
	sum := sha256.Sum256([]byte(value))
	return sum[:]
}
