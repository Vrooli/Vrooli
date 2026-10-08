package archtest

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestReviewEvidenceSelectionIsProducerNeutral keeps the review contract
// outcome-oriented: a criterion names the claim, while a registered action is
// discovered at runtime to produce the artifact.
func TestReviewEvidenceSelectionIsProducerNeutral(t *testing.T) {
	path := filepath.Join("..", "..", "..", "..", "prompt-manager", "store", "skills", "packs", "core", "swarm-manager-review", "SKILL.md")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read review skill: %v", err)
	}
	content := string(data)
	for _, forbidden := range []string{"browser-automation-studio", "bas.screenshot", "bas.record"} {
		if strings.Contains(strings.ToLower(content), forbidden) {
			t.Fatalf("review skill must discover a producer rather than name %q", forbidden)
		}
	}
	for _, required := range []string{"prompt-manager discover", "settlement: \"unavailable\"", "unavailable_reason"} {
		if !strings.Contains(content, required) {
			t.Fatalf("review skill must retain %q in its evidence-selection contract", required)
		}
	}
}
