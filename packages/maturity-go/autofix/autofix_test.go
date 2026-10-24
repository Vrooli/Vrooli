package autofix

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/vrooli/maturity-go/assessment"
)

// fixedFileFixer remediates a target file to a desired content; it is idempotent
// because Preview emits nothing once the file already matches.
func fixedFileFixer(ruleID, rel, want string) Fixer {
	read := func(root string) (string, bool) {
		raw, err := os.ReadFile(filepath.Join(root, rel))
		if err != nil {
			return "", false
		}
		return string(raw), true
	}
	return Fixer{
		RuleID: ruleID,
		Preview: func(root string) ([]Candidate, error) {
			before, _ := read(root)
			if before == want {
				return nil, nil
			}
			return []Candidate{{
				RuleID:      ruleID,
				FilePath:    filepath.Join(root, rel),
				Description: "set " + rel,
				Before:      before,
				After:       want,
			}}, nil
		},
		CanFix: func(root, _ string) bool {
			before, _ := read(root)
			return before != want
		},
	}
}

func TestRegistryPreviewApplyIdempotent(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "a.txt"), []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}
	reg := NewRegistry(
		fixedFileFixer("rule-a", "a.txt", "new"),
		fixedFileFixer("rule-b", "b.txt", "made"),
	)

	preview, err := reg.Preview(root, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(preview) != 2 {
		t.Fatalf("want 2 candidates, got %d", len(preview))
	}
	// Sorted by file path: a.txt before b.txt.
	if preview[0].FilePath >= preview[1].FilePath {
		t.Fatalf("candidates not sorted: %v", preview)
	}
	for _, c := range preview {
		if c.Applied {
			t.Fatalf("preview must not mark applied: %+v", c)
		}
	}

	applied, err := reg.Apply(root, nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range applied {
		if !c.Applied {
			t.Fatalf("apply must mark applied: %+v", c)
		}
	}
	if got, _ := os.ReadFile(filepath.Join(root, "b.txt")); string(got) != "made" {
		t.Fatalf("b.txt not written: %q", got)
	}

	// Idempotency: a second apply changes nothing.
	again, err := reg.Apply(root, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(again) != 0 {
		t.Fatalf("second apply must be a no-op, got %d candidates", len(again))
	}
}

func TestRegistryRuleFilterAndCanFix(t *testing.T) {
	root := t.TempDir()
	reg := NewRegistry(
		fixedFileFixer("rule-a", "a.txt", "new"),
		fixedFileFixer("rule-b", "b.txt", "made"),
	)
	preview, err := reg.Preview(root, []string{"rule-b"})
	if err != nil {
		t.Fatal(err)
	}
	if len(preview) != 1 || preview[0].RuleID != "rule-b" {
		t.Fatalf("rule filter failed: %+v", preview)
	}
	if !reg.CanFix(root, "rule-a", "") {
		t.Fatal("rule-a should be fixable on empty tree")
	}
	if reg.CanFix(root, "rule-missing", "") {
		t.Fatal("unknown rule must not be fixable")
	}
}

func TestFixClassAndAutofixableCount(t *testing.T) {
	if !FixClassAutofix.Autofixable() {
		t.Fatal("autofix class must be autofixable")
	}
	if FixClassDetectionOnly.Autofixable() {
		t.Fatal("detection-only class must not be autofixable")
	}
	findings := []assessment.Finding{
		{Code: "x", AutofixAvailable: true},
		{Code: "y", AutofixAvailable: false},
		{Code: "z", AutofixAvailable: true},
	}
	if got := AutofixableCount(findings); got != 2 {
		t.Fatalf("AutofixableCount = %d, want 2", got)
	}
}
