package hygiene

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestDetectPlanCandidatesClassifiesLegacyScratchPlans(t *testing.T) {
	root := t.TempDir()
	runGit(t, root, "init")
	writeFile(t, filepath.Join(root, "docs", "plans", "scratch-plan.md"), "# Scratch\n")
	writeFile(t, filepath.Join(root, "scenarios", "demo", "docs", "plans", "scenario-plan.md"), "# Scenario\n")
	writeFile(t, filepath.Join(root, "scenarios", "swarm-manager", "backlog", "items", "item-1", "plan.md"), "# Backlog\n")
	writeFile(t, filepath.Join(root, "docs", "product", "plans.md"), "# Product\n")

	candidates, err := DetectPlanCandidates(root)
	if err != nil {
		t.Fatalf("DetectPlanCandidates: %v", err)
	}
	paths := map[string]bool{}
	for _, candidate := range candidates {
		paths[candidate.Path] = true
		if candidate.Status != "untracked" {
			t.Fatalf("candidate %s status = %q, want untracked", candidate.Path, candidate.Status)
		}
	}
	for _, want := range []string{
		"docs/plans/scratch-plan.md",
		"scenarios/demo/docs/plans/scenario-plan.md",
	} {
		if !paths[want] {
			t.Fatalf("missing candidate %q in %#v", want, candidates)
		}
	}
	if paths["scenarios/swarm-manager/backlog/items/item-1/plan.md"] {
		t.Fatalf("swarm backlog plan should be allowlisted")
	}
	if paths["docs/product/plans.md"] {
		t.Fatalf("product docs should not be classified as scratch plans")
	}
}

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}
