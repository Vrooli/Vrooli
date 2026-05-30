package skills

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestSwarmManagerReviewSkill_BaselineDeltaSection guards the before/after
// baseline-diff contract the swarm-manager finalization pipeline depends on:
// the review agent is handed a `baseline-diff-results` attachment and must
// prioritize regressions this item introduced over pre-existing failures.
// If these markers drift out of the skill, the regression-attribution feature
// silently degrades to guesswork.
func TestSwarmManagerReviewSkill_BaselineDeltaSection(t *testing.T) {
	path := filepath.Join("..", "..", "store", "skills", "packs", "core", "swarm-manager-review", "SKILL.md")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read swarm-manager-review skill: %v", err)
	}
	content := string(data)

	required := []string{
		"baseline-diff-results",   // the context attachment key
		"Step 2.6",                // the evaluation section
		"Evaluate Baseline Delta", // section title
		"regression_introduced",   // the round schema flag
		"preexisting",             // must explain pre-existing failures
		"comparable",              // must handle the not-comparable case
		"caused by THIS item",     // attribution wording
	}
	for _, marker := range required {
		if !strings.Contains(content, marker) {
			t.Errorf("swarm-manager-review SKILL.md missing required marker %q", marker)
		}
	}
}
