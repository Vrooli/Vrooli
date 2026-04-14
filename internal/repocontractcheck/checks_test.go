package repocontractcheck

import (
	"testing"

	testkitgo "github.com/vrooli/vrooli/packages/testkit-go"
)

func TestRunPassesAgainstLiveRepo(t *testing.T) {
	report, err := Run(repoRoot(t))
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if !report.Success {
		t.Fatalf("report.Success = false, checks = %+v", report.Checks)
	}
	if len(report.Checks) == 0 {
		t.Fatal("expected checks to be populated")
	}
}

func TestRunRequiresRoot(t *testing.T) {
	if _, err := Run(""); err == nil {
		t.Fatal("expected error for empty root")
	}
}

func TestRunFailsWhenAgentGuidanceMissing(t *testing.T) {
	root := newValidationFixtureRepo(t)
	testkitgo.WriteRelativeFile(t, root, "AGENTS.md", "# AGENTS.md\n")

	report, err := Run(root)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if report.Success {
		t.Fatalf("expected failure, got success: %+v", report.Checks)
	}
	if !hasFailedCheck(report, "adoption_rules_alignment") {
		t.Fatalf("expected adoption_rules_alignment failure, got %+v", report.Checks)
	}
}

func TestRunFailsWhenUnexpectedAdoptionViolationAppears(t *testing.T) {
	root := newValidationFixtureRepo(t)
	testkitgo.WriteRelativeFile(t, root, "scenarios/example/api/main.go", "package main\n\nimport \"path/filepath\"\n\nfunc getVrooliRoot() string {\n\thome := \"/tmp\"\n\treturn filepath.Join(home, \"Vrooli\")\n}\n")

	report, err := Run(root)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if report.Success {
		t.Fatalf("expected failure, got success: %+v", report.Checks)
	}
	if !hasFailedCheck(report, "adoption_rules_alignment") {
		t.Fatalf("expected adoption_rules_alignment failure, got %+v", report.Checks)
	}
}

func TestRunFailsWhenGitMarkerRootProbeAppears(t *testing.T) {
	root := newValidationFixtureRepo(t)
	testkitgo.WriteRelativeFile(t, root, "scenarios/example/api/main.go", "package main\n\nimport (\n\t\"os\"\n\t\"path/filepath\"\n)\n\nfunc resolveRepoRoot(path string) bool {\n\t_, err := os.Stat(filepath.Join(path, \".git\"))\n\treturn err == nil\n}\n")

	report, err := Run(root)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if report.Success {
		t.Fatalf("expected failure, got success: %+v", report.Checks)
	}
	if !hasFailedCheck(report, "adoption_rules_alignment") {
		t.Fatalf("expected adoption_rules_alignment failure, got %+v", report.Checks)
	}
}

func TestRunFailsWhenPNPMWorkspaceRootProbeAppears(t *testing.T) {
	root := newValidationFixtureRepo(t)
	testkitgo.WriteRelativeFile(t, root, "scenarios/example/api/main.go", "package main\n\nimport (\n\t\"os\"\n\t\"path/filepath\"\n)\n\nfunc resolveRepoRoot(path string) bool {\n\t_, err := os.Stat(filepath.Join(path, \"pnpm-workspace.yaml\"))\n\treturn err == nil\n}\n")

	report, err := Run(root)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if report.Success {
		t.Fatalf("expected failure, got success: %+v", report.Checks)
	}
	if !hasFailedCheck(report, "adoption_rules_alignment") {
		t.Fatalf("expected adoption_rules_alignment failure, got %+v", report.Checks)
	}
}

func repoRoot(t *testing.T) string {
	return testkitgo.ProjectRoot(t)
}

func hasFailedCheck(report Report, name string) bool {
	for _, check := range report.Checks {
		if check.Name == name && !check.Passed {
			return true
		}
	}
	return false
}

func newValidationFixtureRepo(t *testing.T) string {
	t.Helper()

	fixture := testkitgo.NewRepoFixture(t)
	fixture.WriteRepoContract(t)
	fixture.WriteRepoSupportDocs(t, testkitgo.DefaultRepoSupportDocs())
	fixture.WriteScenarioStub(t, "alpha")
	fixture.WriteResourceStub(t, "redis")

	return fixture.Root
}
