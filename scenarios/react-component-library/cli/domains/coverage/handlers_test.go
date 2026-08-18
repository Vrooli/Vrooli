package coverage

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestPlanPrunesExpiredFilesButRetainsEvidenceArtifacts(t *testing.T) {
	root := t.TempDir()
	old := filepath.Join(root, "captures", "390x844-light.png")
	verdict := filepath.Join(root, "verdicts.json")
	recent := filepath.Join(root, "captures", "recent.png")
	for _, path := range []string{old, verdict, recent} {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(path), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Chtimes(old, time.Now().Add(-15*24*time.Hour), time.Now().Add(-15*24*time.Hour)); err != nil {
		t.Fatal(err)
	}

	report, err := plan(root, true)
	if err != nil {
		t.Fatal(err)
	}
	if !report.DryRun || len(report.Selected) != 1 || report.Selected[0].Path != old {
		t.Fatalf("report = %+v, want one expired capture in dry-run selection", report)
	}
	if len(report.RetainedArtifact) != 1 || report.RetainedArtifact[0] != "verdicts.json" {
		t.Fatalf("retained artifacts = %v, want verdicts.json", report.RetainedArtifact)
	}
	if _, err := os.Stat(old); err != nil {
		t.Fatalf("dry run removed %s: %v", old, err)
	}

	if _, err := plan(root, false); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(old); !os.IsNotExist(err) {
		t.Fatalf("apply did not remove expired capture; stat error = %v", err)
	}
	if _, err := os.Stat(verdict); err != nil {
		t.Fatalf("apply removed retained verdict: %v", err)
	}
}
