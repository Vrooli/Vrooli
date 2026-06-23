package autofix

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"ui-health/internal/services/manifestvalidation"

	autofixcore "github.com/vrooli/maturity-go/autofix"
)

// diskAwareValidator re-derives slot-dir findings from disk under root, mirroring
// the real manifest validator: a slot_dir_missing / slot_parent_dir_missing
// finding is emitted only while the directory is still absent. This lets the
// tests exercise the real preview → apply → re-preview idempotency contract.
type diskAwareValidator struct {
	root  string
	slots []slotSpec
	extra []manifestvalidation.Finding
}

type slotSpec struct {
	code string
	rel  string
}

func (v diskAwareValidator) ValidateScenario(_ context.Context, _ string) (manifestvalidation.Report, error) {
	rep := manifestvalidation.Report{Scenario: filepath.Base(v.root)}
	for _, s := range v.slots {
		abs := filepath.Join(v.root, filepath.FromSlash(s.rel))
		if _, err := os.Stat(abs); os.IsNotExist(err) {
			rep.Findings = append(rep.Findings, manifestvalidation.Finding{
				Severity: manifestvalidation.SeverityWarning,
				Code:     s.code,
				Location: abs,
				Message:  "slot directory missing",
			})
		}
	}
	rep.Findings = append(rep.Findings, v.extra...)
	return rep, nil
}

func newFixer(root string) *Fixer {
	return New(diskAwareValidator{
		root: root,
		slots: []slotSpec{
			{code: RuleSlotDirMissing, rel: "ui/src/widgets"},
			{code: RuleSlotParentDirMissing, rel: "ui/src/features"},
		},
		// A non-fixable finding must never produce a candidate.
		extra: []manifestvalidation.Finding{
			{Severity: manifestvalidation.SeverityError, Code: "contract_kind_mismatch", Location: filepath.Join(root, "ui", "manifest.json"), Message: "kind mismatch"},
		},
	})
}

func TestFixClassFor(t *testing.T) {
	for _, code := range []string{RuleSlotDirMissing, RuleSlotParentDirMissing} {
		if got := FixClassFor(code); got != autofixcore.FixClassAutofix {
			t.Fatalf("FixClassFor(%q)=%q, want autofix", code, got)
		}
	}
	if got := FixClassFor("contract_kind_mismatch"); got != autofixcore.FixClassDetectionOnly {
		t.Fatalf("FixClassFor(non-fixable)=%q, want detection_only", got)
	}
}

func TestPreviewProducesCandidatesWithoutWriting(t *testing.T) {
	root := t.TempDir()
	f := newFixer(root)

	resp, err := f.PreviewFixResponse("demo", root, nil)
	if err != nil {
		t.Fatalf("PreviewFixResponse: %v", err)
	}
	if got := len(resp.GetCandidates()); got != 2 {
		t.Fatalf("preview candidates=%d, want 2 (the two missing slot dirs)", got)
	}
	// Preview must not touch the filesystem.
	for _, rel := range []string{"ui/src/widgets", "ui/src/features"} {
		if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(rel), ".gitkeep")); !os.IsNotExist(err) {
			t.Fatalf("preview wrote %s/.gitkeep; preview must be read-only", rel)
		}
	}
	// Every candidate must be a registered fixable rule, never the non-fixable one.
	for _, c := range resp.GetCandidates() {
		if c.GetRuleId() != RuleSlotDirMissing && c.GetRuleId() != RuleSlotParentDirMissing {
			t.Fatalf("unexpected candidate rule %q", c.GetRuleId())
		}
	}
}

func TestApplyCreatesDirsAndIsIdempotent(t *testing.T) {
	root := t.TempDir()
	f := newFixer(root)

	resp, err := f.ApplyFixResponse("demo", root, nil)
	if err != nil {
		t.Fatalf("ApplyFixResponse: %v", err)
	}
	if !resp.GetApplied() {
		t.Fatal("first apply: Applied=false, want true")
	}
	if got := len(resp.GetCandidates()); got != 2 {
		t.Fatalf("first apply candidates=%d, want 2", got)
	}
	for _, rel := range []string{"ui/src/widgets", "ui/src/features"} {
		if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(rel), ".gitkeep")); err != nil {
			t.Fatalf("apply did not create %s/.gitkeep: %v", rel, err)
		}
	}

	// Second apply must be a no-op: the dirs now exist so the validator emits no
	// slot-dir findings, so the registry produces no candidates.
	resp2, err := f.ApplyFixResponse("demo", root, nil)
	if err != nil {
		t.Fatalf("second ApplyFixResponse: %v", err)
	}
	if got := len(resp2.GetCandidates()); got != 0 {
		t.Fatalf("second apply candidates=%d, want 0 (idempotent)", got)
	}
}

func TestCanFixScopedToSpecificFinding(t *testing.T) {
	root := t.TempDir()
	f := newFixer(root)

	missingDir := filepath.Join(root, "ui", "src", "widgets")
	if !f.CanFix(root, RuleSlotDirMissing, missingDir) {
		t.Fatal("CanFix should be true for a missing slot dir with a registered fixer")
	}
	// A non-fixable code never reports an available fix.
	if f.CanFix(root, "contract_kind_mismatch", filepath.Join(root, "ui", "manifest.json")) {
		t.Fatal("CanFix must be false for a code with no registered fixer")
	}
	// A finding path that is not among the current candidates is not fixable.
	if f.CanFix(root, RuleSlotDirMissing, filepath.Join(root, "ui", "src", "nonexistent-slot")) {
		t.Fatal("CanFix must be false for a path that has no matching candidate")
	}

	// Once the directory exists, the rule can no longer fix it (no no-op claims).
	if err := os.MkdirAll(missingDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if f.CanFix(root, RuleSlotDirMissing, missingDir) {
		t.Fatal("CanFix must be false once the directory exists")
	}
}
