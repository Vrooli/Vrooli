package structure

import (
	"os"
	"path/filepath"
	"testing"

	repocontract "github.com/vrooli/repo-contract-go"
)

func TestResolveCandidateUsesRepoContractForScenarioName(t *testing.T) {
	repoRoot, err := repocontract.FindRepoRootFromCWD()
	if err != nil {
		t.Fatalf("FindRepoRootFromCWD() error: %v", err)
	}
	t.Setenv("VROOLI_ROOT", repoRoot)
	chdirRuleTest(t, filepath.Join(repoRoot, "scenarios", "scenario-auditor", "api"))

	got := resolveCandidate("test-genie")
	want := filepath.Join(repoRoot, "scenarios", "test-genie")
	if got != want {
		t.Fatalf("resolveCandidate() = %q, want %q", got, want)
	}
}

func TestSearchRuleDirFromUsesRepoContract(t *testing.T) {
	repoRoot, err := repocontract.FindRepoRootFromCWD()
	if err != nil {
		t.Fatalf("FindRepoRootFromCWD() error: %v", err)
	}

	got := searchRuleDirFrom(filepath.Join(repoRoot, "scenarios", "test-genie"))
	want := filepath.Join(repoRoot, "scenarios", "scenario-auditor", "api", "rules", "structure")
	if got != want {
		t.Fatalf("searchRuleDirFrom() = %q, want %q", got, want)
	}
}

func chdirRuleTest(t *testing.T, dir string) {
	t.Helper()
	orig, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir %s: %v", dir, err)
	}
	t.Cleanup(func() { _ = os.Chdir(orig) })
}
