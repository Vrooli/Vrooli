package repocontractcheck

import (
	"path/filepath"
	"testing"

	"github.com/vrooli/vrooli/internal/resources"
	manifestpkg "github.com/vrooli/vrooli/internal/resources/manifest"
	testkitgo "github.com/vrooli/vrooli/packages/testkit-go"
	testresource "github.com/vrooli/vrooli/packages/testkit-go/resourcefixture"
)

func TestRunReportsChecksAgainstLiveRepo(t *testing.T) {
	report, err := Run(repoRoot(t))
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if len(report.Checks) == 0 {
		t.Fatal("expected checks to be populated")
	}
	found := false
	for _, check := range report.Checks {
		if check.Name == "resource_schema_artifacts" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected resource_schema_artifacts check, got %+v", report.Checks)
	}
}

func TestRunRequiresRoot(t *testing.T) {
	if _, err := Run(""); err == nil {
		t.Fatal("expected error for empty root")
	}
}

func TestRunFailsWhenUnexpectedAdoptionViolationAppears(t *testing.T) {
	fixture := newValidationFixtureRepo(t)
	root := fixture.Root
	writeValidationScenarioSource(t, fixture, "example", "api/main.go", "package main\n\nimport \"path/filepath\"\n\nfunc getVrooliRoot() string {\n\thome := \"/tmp\"\n\treturn filepath.Join(home, \"Vrooli\")\n}\n")

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
	fixture := newValidationFixtureRepo(t)
	root := fixture.Root
	writeValidationScenarioSource(t, fixture, "example", "api/main.go", "package main\n\nimport (\n\t\"os\"\n\t\"path/filepath\"\n)\n\nfunc resolveRepoRoot(path string) bool {\n\t_, err := os.Stat(filepath.Join(path, \".git\"))\n\treturn err == nil\n}\n")

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
	fixture := newValidationFixtureRepo(t)
	root := fixture.Root
	writeValidationScenarioSource(t, fixture, "example", "api/main.go", "package main\n\nimport (\n\t\"os\"\n\t\"path/filepath\"\n)\n\nfunc resolveRepoRoot(path string) bool {\n\t_, err := os.Stat(filepath.Join(path, \"pnpm-workspace.yaml\"))\n\treturn err == nil\n}\n")

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

func newValidationFixtureRepo(t *testing.T) testkitgo.RepoFixture {
	t.Helper()

	fixture := testkitgo.NewRepoFixture(t)
	fixture.WriteRepoContract(t)
	testkitgo.WriteRelativeFile(t, fixture.Root, "docs/repo-contract.md", `# Repo Contract

## Validation

- `+"`vrooli contract validate`"+`
- `+"`vrooli contract show`"+`
- `+"`vrooli contract resolve scenario <name> --file service`"+`
- `+"`vrooli contract match-glob <pattern> <path>`"+`
- `+"`make validate-repo-contract` remains the CI/automation entrypoint"+`

## Landed Consumer Migrations

- `+"`swarm-manager`"+`
`)
	fixture.WriteScenarioStub(t, "alpha")
	testresource.WriteResourceManifest(t, fixture.Root, "redis", testresource.ResourceManifest(
		"redis",
		testresource.WithResourceDriver("docker-service"),
		testresource.WithResourceTemplate("docker-service"),
		testresource.WithResourceDisplayName("Redis"),
		testresource.WithResourceDescription("Cache"),
		testresource.WithResourceRuntime(manifestpkg.ResourceRuntime{
			Image: "redis:7-alpine",
		}),
	))
	if _, err := resources.SyncSchemaArtifacts(fixture.Root); err != nil {
		t.Fatalf("SyncSchemaArtifacts: %v", err)
	}

	return fixture
}

func writeValidationScenarioSource(t *testing.T, fixture testkitgo.RepoFixture, scenarioName, relPath, contents string) {
	t.Helper()
	testkitgo.WriteRelativeFile(t, fixture.Root, filepath.ToSlash(filepath.Join(fixture.ScenarioDir, scenarioName, filepath.FromSlash(relPath))), contents)
}
