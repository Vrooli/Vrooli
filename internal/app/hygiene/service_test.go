package hygiene

import (
	"context"
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

func TestServiceRunIncludesDriftCheckWhenNoScenarios(t *testing.T) {
	root := t.TempDir()
	runGit(t, root, "init")
	writeFile(t, filepath.Join(root, ".vrooli", "repo-contract.json"), `{"version":"v0","required_dirs":[],"required_files":[],"checks":[]}`)
	report, err := Service{Root: root, Home: root, DependencyFreshnessRunner: fakeDependencyFreshnessRunner{}}.Run(Request{
		FailOn:          SeverityError,
		IncludeContract: false,
		IncludePlans:    false,
		IncludeDrift:    true,
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if report.SharedDrift == nil {
		t.Fatalf("SharedDrift report missing")
	}
	if !report.SharedDrift.Clean {
		t.Fatalf("expected clean drift with no scenarios, got %#v", report.SharedDrift)
	}
	var sawCheck bool
	for _, c := range report.Checks {
		if c.Name == "dependency_freshness" {
			sawCheck = true
			if !c.Passed {
				t.Fatalf("dependency_freshness check failed: %s", c.Message)
			}
		}
	}
	if !sawCheck {
		t.Fatalf("expected dependency_freshness check entry, got %+v", report.Checks)
	}
}

type fakeDependencyFreshnessRunner struct{}

func (fakeDependencyFreshnessRunner) CheckDependencyFreshness(_ context.Context, root string) (sdaFreshnessReport, error) {
	return sdaFreshnessReport{Clean: true, Root: root, Mode: "touched"}, nil
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

func TestIsArtifactResiduePathClassifiesGeneratedCaptures(t *testing.T) {
	residue := []string{
		"docs/architecture/evidence/baseline-p4.json",
		"docs/architecture/internal-debt-atlas.html",
		"docs/reference/cross-platform-effort/evidence/machines/04b-scope-audit.png",
		"docs/reports/supervision-authority-baseline.md",
		"scenarios/react-component-library/docs/evidence/final-gate-report.json",
		"scenarios/backdrop-studio/docs/evidence/catalog/sheet-tonal.png",
		"evidence/phase1-lifecycle-race.txt",
		"coverage/browser-soak-1787056948739551951.json",
		".artifacts/rcl-screenshots/shot.png",
	}
	for _, rel := range residue {
		if !isArtifactResiduePath(rel) {
			t.Errorf("isArtifactResiduePath(%q) = false, want true", rel)
		}
	}
}

func TestIsArtifactResiduePathSparesRealSource(t *testing.T) {
	// "evidence", "captures", "reports" and "phases" are all real package names
	// in this repository. Only a docs tree makes them capture directories, so a
	// name-only sweep would delete working code.
	source := []string{
		"scenarios/deployment-manager/api/internal/evidence/service.go",
		"packages/proto/gen/go/git-control-tower/v1/evidence/evidence.pb.go",
		"packages/proto/schemas/treasury/v1/evidence/evidence.proto",
		"scenarios/swarm-manager/cli/domains/captures/handlers.go",
		"scenarios/test-genie/api/internal/orchestrator/phases/docs.go",
		"scenarios/ui-health/ui/src/features/captures/index.ts",
		"docs/concepts/ARCHITECTURE.md",
		"docs/reference/machine-readable-references.md",
		"README.md",
	}
	for _, rel := range source {
		if isArtifactResiduePath(rel) {
			t.Errorf("isArtifactResiduePath(%q) = true, want false", rel)
		}
	}
}
