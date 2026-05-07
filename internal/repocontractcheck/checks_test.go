package repocontractcheck

import (
	"path/filepath"
	"strings"
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

func TestRunFailsWhenPersonalAbsolutePathAppearsInActiveGoSource(t *testing.T) {
	fixture := newValidationFixtureRepo(t)
	writeValidationScenarioSource(t, fixture, "example", "api/main.go", "package main\n\nconst root = \"/home/carol.dev/Vrooli\"\n")

	report, err := Run(fixture.Root)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if report.Success {
		t.Fatalf("expected failure, got success: %+v", report.Checks)
	}
	if !hasFailedCheck(report, "personal_absolute_paths") {
		t.Fatalf("expected personal_absolute_paths failure, got %+v", report.Checks)
	}
	message := failedCheckMessage(report, "personal_absolute_paths")
	if !strings.Contains(message, "scenarios/example/api/main.go:3") {
		t.Fatalf("expected relative file:line in message, got %q", message)
	}
}

func TestRunFailsWhenMacOSPersonalAbsolutePathAppearsInActiveGoSource(t *testing.T) {
	fixture := newValidationFixtureRepo(t)
	writeValidationScenarioSource(t, fixture, "example", "api/main.go", "package main\n\nconst root = \"/Users/carol.dev/Vrooli\"\n")

	report, err := Run(fixture.Root)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if report.Success {
		t.Fatalf("expected failure, got success: %+v", report.Checks)
	}
	if !hasFailedCheck(report, "personal_absolute_paths") {
		t.Fatalf("expected personal_absolute_paths failure, got %+v", report.Checks)
	}
	message := failedCheckMessage(report, "personal_absolute_paths")
	if !strings.Contains(message, "scenarios/example/api/main.go:3") {
		t.Fatalf("expected relative file:line in message, got %q", message)
	}
}

func TestRunFailsWhenPersonalAbsolutePathAppearsInActiveScriptOrConfig(t *testing.T) {
	fixture := newValidationFixtureRepo(t)
	testkitgo.WriteRelativeFile(t, fixture.Root, "templates/scenarios/example/perf/capture.js", "const root = \"/home/carol.dev/Vrooli\";\n")
	testkitgo.WriteRelativeFile(t, fixture.Root, ".vrooli/resources/example.json", "{\n  \"root\": \"/Users/carol.dev/Vrooli\"\n}\n")

	report, err := Run(fixture.Root)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if report.Success {
		t.Fatalf("expected failure, got success: %+v", report.Checks)
	}
	message := failedCheckMessage(report, "personal_absolute_paths")
	for _, want := range []string{
		"templates/scenarios/example/perf/capture.js:1",
		".vrooli/resources/example.json:2",
	} {
		if !strings.Contains(message, want) {
			t.Fatalf("expected %q in message %q", want, message)
		}
	}
}

func TestRunFailsWhenPersonalAbsolutePathAppearsInPromptMarkdown(t *testing.T) {
	fixture := newValidationFixtureRepo(t)
	testkitgo.WriteRelativeFile(t, fixture.Root, "scenarios/example/prompts/templates/build.md", "Run `rg thing /home/carol.dev/Vrooli` before generating.\n")
	testkitgo.WriteRelativeFile(t, fixture.Root, "scenarios/prompt-manager/store/skills/packs/core/example/SKILL.md", "const root = \"/home/alice/Vrooli\";\n")

	report, err := Run(fixture.Root)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if report.Success {
		t.Fatalf("expected failure, got success: %+v", report.Checks)
	}
	message := failedCheckMessage(report, "personal_absolute_paths")
	for _, want := range []string{
		"scenarios/example/prompts/templates/build.md:1",
		"scenarios/prompt-manager/store/skills/packs/core/example/SKILL.md:1",
	} {
		if !strings.Contains(message, want) {
			t.Fatalf("expected %q in message %q", want, message)
		}
	}
}

func TestRunFailsWhenOperatorIdentityAppearsInPromptOrGeneratedState(t *testing.T) {
	fixture := newValidationFixtureRepo(t)
	testkitgo.WriteRelativeFile(t, fixture.Root, "scenarios/example/prompts/templates/operator.md", "Ask Matt Halloran to approve this.\n")
	testkitgo.WriteRelativeFile(t, fixture.Root, "scenarios/swarm-manager/ideas/example/review/round-001.json", "{\"operator\":\"matthalloran8\"}\n")

	report, err := Run(fixture.Root)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if report.Success {
		t.Fatalf("expected failure, got success: %+v", report.Checks)
	}
	message := failedCheckMessage(report, "personal_absolute_paths")
	for _, want := range []string{
		"scenarios/example/prompts/templates/operator.md:1",
		"scenarios/swarm-manager/ideas/example/review/round-001.json:1",
	} {
		if !strings.Contains(message, want) {
			t.Fatalf("expected %q in message %q", want, message)
		}
	}
}

func TestRunAllowsIntentionalDetectorPaths(t *testing.T) {
	fixture := newValidationFixtureRepo(t)
	testkitgo.WriteRelativeFile(t, fixture.Root, "scenarios/code-smell/initialization/rules/vrooli-specific.yaml", "pattern: /home/carol.dev/Vrooli\n")

	report, err := Run(fixture.Root)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if hasFailedCheck(report, "personal_absolute_paths") {
		t.Fatalf("did not expect personal_absolute_paths failure, got %+v", report.Checks)
	}
}

func TestRunFailsWhenPersonalAbsolutePathAppearsInGeneratedSwarmManagerState(t *testing.T) {
	fixture := newValidationFixtureRepo(t)
	testkitgo.WriteRelativeFile(t, fixture.Root, "scenarios/swarm-manager/ideas/example/.swarm/last-research-prompt-trace.json", "{\"root\":\"/home/carol.dev/Vrooli\"}\n")
	testkitgo.WriteRelativeFile(t, fixture.Root, "scenarios/swarm-manager/ideas/example/review/captures/output.txt", "error at /Users/carol.dev/Vrooli/scenarios/app/main.go\n")
	testkitgo.WriteRelativeFile(t, fixture.Root, "scenarios/swarm-manager/ideas/example/handoff/brief.md", "Plan: /home/carol.dev/Vrooli/scenarios/app/plan.md\n")

	report, err := Run(fixture.Root)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if report.Success {
		t.Fatalf("expected failure, got success: %+v", report.Checks)
	}
	message := failedCheckMessage(report, "personal_absolute_paths")
	for _, want := range []string{
		"scenarios/swarm-manager/ideas/example/.swarm/last-research-prompt-trace.json:1",
		"scenarios/swarm-manager/ideas/example/review/captures/output.txt:1",
		"scenarios/swarm-manager/ideas/example/handoff/brief.md:1",
	} {
		if !strings.Contains(message, want) {
			t.Fatalf("expected %q in message %q", want, message)
		}
	}
}

func TestRunAllowsPortableGeneratedSwarmManagerState(t *testing.T) {
	fixture := newValidationFixtureRepo(t)
	testkitgo.WriteRelativeFile(t, fixture.Root, "scenarios/swarm-manager/ideas/example/handoff/manifest.json", "{\"item_folder\":\"path:scenarios/swarm-manager/ideas/example\"}\n")
	testkitgo.WriteRelativeFile(t, fixture.Root, "scenarios/swarm-manager/ideas/example/review/captures/output.txt", "error at path:scenarios/app/main.go\n")

	report, err := Run(fixture.Root)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if hasFailedCheck(report, "personal_absolute_paths") {
		t.Fatalf("did not expect personal_absolute_paths failure, got %+v", report.Checks)
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

func failedCheckMessage(report Report, name string) string {
	for _, check := range report.Checks {
		if check.Name == name {
			return check.Message
		}
	}
	return ""
}

func TestRunFailsWhenScenarioContainsRawOllamaEmbeddingsLiteral(t *testing.T) {
	fixture := newValidationFixtureRepo(t)
	writeValidationScenarioSource(t, fixture, "swarm-manager", "api/internal/aisearch/embedder.go",
		"package aisearch\n\nconst path = \"/api/embeddings\"\n")

	report, err := Run(fixture.Root)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if report.Success {
		t.Fatalf("expected failure, got success: %+v", report.Checks)
	}
	if !hasFailedCheck(report, "ollama_gateway_only") {
		t.Fatalf("expected ollama_gateway_only failure, got %+v", report.Checks)
	}
	message := failedCheckMessage(report, "ollama_gateway_only")
	if !strings.Contains(message, "/api/embeddings") {
		t.Fatalf("expected message to mention banned literal, got %q", message)
	}
}

func TestRunPassesWhenScenarioStaysOffRawOllamaEndpoints(t *testing.T) {
	fixture := newValidationFixtureRepo(t)
	writeValidationScenarioSource(t, fixture, "swarm-manager", "api/internal/aisearch/embedder.go",
		"package aisearch\n\n// uses resource-ollama gateway, no raw HTTP\nconst placeholder = \"ok\"\n")

	report, err := Run(fixture.Root)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	for _, check := range report.Checks {
		if check.Name == "ollama_gateway_only" && !check.Passed {
			t.Fatalf("ollama_gateway_only should pass; message=%q", check.Message)
		}
	}
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

## Allowed `+".vrooli/"+` Surface

- `+"`~/.vrooli/secrets.json`"+`

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
