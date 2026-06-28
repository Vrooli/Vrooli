package validation

import (
	"os"
	"path/filepath"
	"testing"
)

func TestColorSystemFixerCreatesTokensWhenAbsentAndIsIdempotent(t *testing.T) {
	root := t.TempDir()
	// Sanity: scan should report has-color-system missing.
	if _, ok := findingByRule(ScanScenario("x", root), "has-color-system"); !ok {
		t.Fatal("precondition: expected has-color-system finding")
	}

	// Preview (dry-run) proposes a candidate but writes nothing.
	cands, _, err := BuildFixCandidates(root, []string{"has-color-system"}, false)
	if err != nil {
		t.Fatalf("preview: %v", err)
	}
	if len(cands) != 1 || cands[0].Applied {
		t.Fatalf("preview candidates = %+v", cands)
	}
	if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(designSystemCSSRel))); !os.IsNotExist(err) {
		t.Fatal("preview must not write the token file")
	}

	// Apply writes the file and the finding clears.
	applied, _, err := BuildFixCandidates(root, []string{"has-color-system"}, true)
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	if len(applied) != 1 || !applied[0].Applied {
		t.Fatalf("apply candidates = %+v", applied)
	}
	if _, ok := findingByRule(ScanScenario("x", root), "has-color-system"); ok {
		t.Fatal("has-color-system finding should clear after ApplyFix")
	}

	// Re-apply is idempotent: nothing left to fix (file now present).
	again, _, err := BuildFixCandidates(root, []string{"has-color-system"}, true)
	if err != nil {
		t.Fatalf("re-apply: %v", err)
	}
	if len(again) != 0 {
		t.Fatalf("re-apply should propose no candidates, got %+v", again)
	}
}

func TestNonFixableRuleReportsMessage(t *testing.T) {
	root := t.TempDir()
	cands, messages, err := BuildFixCandidates(root, []string{"has-logo"}, false)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if len(cands) != 0 {
		t.Fatalf("has-logo should yield no candidates, got %+v", cands)
	}
	if len(messages) == 0 {
		t.Fatal("expected a message explaining has-logo has no deterministic fixer")
	}
}
