package executionevidence

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestCutoverArchivesOnlyWithExplicitConfirmation(t *testing.T) {
	dir := t.TempDir()
	coverage := filepath.Join(dir, "coverage")
	if err := os.MkdirAll(filepath.Join(coverage, "runs", "old"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(coverage, "runs", "old", "findings.json"), []byte("detail"), 0o644); err != nil {
		t.Fatal(err)
	}
	plan, err := PlanCutover(dir, filepath.Join(dir, "archive"))
	if err != nil {
		t.Fatal(err)
	}
	if plan.Files != 1 || plan.Bytes != int64(len("detail")) || plan.Digest == "" {
		t.Fatalf("plan=%+v", plan)
	}
	if err := ApplyCutover(plan, "no"); !errors.Is(err, ErrCutoverNotConfirmed) {
		t.Fatalf("confirmation error=%v", err)
	}
	if _, err := os.Stat(filepath.Join(coverage, "runs", "old", "findings.json")); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(coverage, "new.json"), []byte("changed"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := ApplyCutover(plan, CutoverConfirmation); err == nil {
		t.Fatal("changed inventory must reject cutover")
	}
	plan, err = PlanCutover(dir, filepath.Join(dir, "archive"))
	if err != nil {
		t.Fatal(err)
	}
	if err := ApplyCutover(plan, CutoverConfirmation); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "archive", "runs", "old", "findings.json")); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "archive", CutoverReceiptFile)); err != nil {
		t.Fatalf("expected operator receipt in archive: %v", err)
	}
	if _, err := os.Stat(coverage); err != nil {
		t.Fatal(err)
	}
}
