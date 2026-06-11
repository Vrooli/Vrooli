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

func TestNoRuntimeHomeLiteralsPassesOnLiveRepo(t *testing.T) {
	report, err := Run(repoRoot(t))
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	for _, check := range report.Checks {
		if check.Name == "no_runtime_home_literals" {
			if !check.Passed {
				t.Fatalf("no_runtime_home_literals must pass on the live repo (a home-dir .vrooli literal slipped in): %s", check.Message)
			}
			return
		}
	}
	t.Fatal("no_runtime_home_literals check missing from report")
}

func TestNoRuntimeHomeLiteralsTripsOnReintroducedLiteral(t *testing.T) {
	fixture := newValidationFixtureRepo(t)
	root := fixture.Root
	// Reintroduce a home-dir .vrooli join in the platform surface (internal/).
	testkitgo.WriteRelativeFile(t, root, "internal/drift/drift.go",
		"package drift\n\nimport \"path/filepath\"\n\nfunc logsDir(home string) string {\n\treturn filepath.Join(home, \".vrooli\", \"logs\")\n}\n")

	report, err := Run(root)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if !hasFailedCheck(report, "no_runtime_home_literals") {
		t.Fatalf("expected no_runtime_home_literals failure, got %+v", report.Checks)
	}
}

func TestNoRuntimeHomeLiteralsAllowsAnnotatedProjectConfig(t *testing.T) {
	fixture := newValidationFixtureRepo(t)
	root := fixture.Root
	// A home-join that is genuinely the repo-project dir, annotated, must pass.
	testkitgo.WriteRelativeFile(t, root, "internal/okpkg/ok.go",
		"package okpkg\n\nimport \"path/filepath\"\n\nfunc projectConfig(home string) string {\n\treturn filepath.Join(home, \".vrooli\") // repo-contract:project-config\n}\n")

	report, err := Run(root)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if hasFailedCheck(report, "no_runtime_home_literals") {
		t.Fatalf("annotated repo-project use must not trip the guard: %+v", report.Checks)
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

func TestRunFailsWhenKnownScenarioDebtContainsRawOllamaChatLiteral(t *testing.T) {
	fixture := newValidationFixtureRepo(t)
	writeValidationScenarioSource(t, fixture, "audio-tools", "api/internal/summarize/summarizer.go",
		"package summarize\n\nconst path = \"/api/chat\"\n")

	report, err := Run(fixture.Root)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if report.Success {
		t.Fatalf("expected failure, got success: %+v", report.Checks)
	}
	message := failedCheckMessage(report, "ollama_gateway_only")
	if !strings.Contains(message, "audio-tools/api/internal/summarize/summarizer.go") || !strings.Contains(message, "/api/chat") {
		t.Fatalf("expected known raw caller path to fail, got %q", message)
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

func TestRunFailsWhenScenarioDefinesLocalOllamaVectorSize(t *testing.T) {
	fixture := newValidationFixtureRepo(t)
	writeValidationScenarioSource(t, fixture, "fresh-search", "api/internal/aisearch/vectorstore_test.go",
		"package aisearch\n\nfunc TestVector(t *testing.T) {\n\t_ = CollectionSpec{DenseSize: 768}\n}\n")

	report, err := Run(fixture.Root)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if report.Success {
		t.Fatalf("expected failure, got success: %+v", report.Checks)
	}
	if !hasFailedCheck(report, "ollama_policy_facts") {
		t.Fatalf("expected ollama_policy_facts failure, got %+v", report.Checks)
	}
	message := failedCheckMessage(report, "ollama_policy_facts")
	if !strings.Contains(message, "DenseSize") {
		t.Fatalf("expected message to mention DenseSize, got %q", message)
	}
}

func TestRunAllowsClearlyNamedFixtureOllamaVectorSize(t *testing.T) {
	fixture := newValidationFixtureRepo(t)
	writeValidationScenarioSource(t, fixture, "fresh-search", "api/internal/aisearch/vectorstore_test.go",
		"package aisearch\n\nconst fixtureEmbeddingDimensions = 768\n")

	report, err := Run(fixture.Root)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if hasFailedCheck(report, "ollama_policy_facts") {
		t.Fatalf("fixture-labeled test dimension should pass, got %+v", report.Checks)
	}
}

func TestRunFailsWhenScenarioSQLDefinesPgvectorDimension(t *testing.T) {
	fixture := newValidationFixtureRepo(t)
	testkitgo.WriteRelativeFile(t, fixture.Root, "scenarios/fresh-search/initialization/storage/postgres/schema.sql",
		"CREATE TABLE tasks (embedding VECTOR(768));\n")

	report, err := Run(fixture.Root)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if report.Success {
		t.Fatalf("expected failure, got success: %+v", report.Checks)
	}
	message := failedCheckMessage(report, "ollama_policy_facts")
	if !strings.Contains(message, "schema.sql") || !strings.Contains(message, "vector") {
		t.Fatalf("expected message to mention SQL vector dimension, got %q", message)
	}
}

func TestRunFailsWhenScenarioTestNamesPhysicalOllamaModel(t *testing.T) {
	fixture := newValidationFixtureRepo(t)
	writeValidationScenarioSource(t, fixture, "prompt-manager", "api/aisearch/embedder_test.go",
		"package aisearch\n\nconst model = \"nomic-embed-text:latest\"\n")

	report, err := Run(fixture.Root)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if report.Success {
		t.Fatalf("expected failure, got success: %+v", report.Checks)
	}
	message := failedCheckMessage(report, "ollama_policy_facts")
	if !strings.Contains(message, "physical Ollama model literal") {
		t.Fatalf("expected physical-model message, got %q", message)
	}
}

func TestRunDoesNotFlagNonOllamaModelNameSubstrings(t *testing.T) {
	fixture := newValidationFixtureRepo(t)
	writeValidationScenarioSource(t, fixture, "agent-inbox", "api/integrations/openrouter_types.go",
		"package integrations\n\nconst engine = \"mistral/ocr\"\nconst task = \"pdf-text or mistral-ocr\"\n")
	writeValidationScenarioSource(t, fixture, "data-tools", "initialization/configuration/app-config.json",
		`{"vision":["llama3.2-vision:11b"]}`)

	report, err := Run(fixture.Root)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if hasFailedCheck(report, "ollama_policy_facts") {
		t.Fatalf("non-Ollama substrings should not trip ollama_policy_facts, got %q", failedCheckMessage(report, "ollama_policy_facts"))
	}
}

func TestRunFailsWhenScenarioNamesDecimalPhysicalOllamaModel(t *testing.T) {
	fixture := newValidationFixtureRepo(t)
	writeValidationScenarioSource(t, fixture, "audio-tools", "api/internal/summarize/summarizer_test.go",
		"package summarize\n\nconst model = \"qwen3:1.7b\"\n")

	report, err := Run(fixture.Root)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if report.Success {
		t.Fatalf("expected failure, got success: %+v", report.Checks)
	}
	message := failedCheckMessage(report, "ollama_policy_facts")
	if !strings.Contains(message, "physical Ollama model literal") {
		t.Fatalf("expected physical-model message, got %q", message)
	}
}

func TestRunFailsWhenMigratedInitializationStoresQdrantVectorSize(t *testing.T) {
	fixture := newValidationFixtureRepo(t)
	testkitgo.WriteRelativeFile(t, fixture.Root, "scenarios/seo-optimizer/initialization/qdrant/collections.json",
		`{"collections":[{"name":"seo_content","config":{"vectors":{"size":768,"distance":"Cosine"}}}]}`)

	report, err := Run(fixture.Root)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if report.Success {
		t.Fatalf("expected failure, got success: %+v", report.Checks)
	}
	message := failedCheckMessage(report, "ollama_policy_facts")
	if !strings.Contains(message, "seo-optimizer/initialization/qdrant/collections.json") {
		t.Fatalf("expected message to mention initialization payload, got %q", message)
	}
}

func TestRunFailsWhenScenarioUsesGatewayDirectModel(t *testing.T) {
	fixture := newValidationFixtureRepo(t)
	writeValidationScenarioSource(t, fixture, "prompt-manager", "api/internal/embed.sh",
		"resource-ollama gateway embed --model nomic-embed-text:latest --input hi\n")

	report, err := Run(fixture.Root)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if report.Success {
		t.Fatalf("expected failure, got success: %+v", report.Checks)
	}
	message := failedCheckMessage(report, "ollama_policy_facts")
	if !strings.Contains(message, "--model") {
		t.Fatalf("expected message to mention --model, got %q", message)
	}
}

func TestRunFailsWhenResourceMaintainsOllamaDimensionMap(t *testing.T) {
	fixture := newValidationFixtureRepo(t)
	testkitgo.WriteRelativeFile(t, fixture.Root, "resources/qdrant/lib/models.sh",
		"declare -A KNOWN_MODEL_DIMENSIONS=([nomic]=768)\n")

	report, err := Run(fixture.Root)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if report.Success {
		t.Fatalf("expected failure, got success: %+v", report.Checks)
	}
	message := failedCheckMessage(report, "ollama_policy_facts")
	if !strings.Contains(message, "model-dimension map") {
		t.Fatalf("expected message to mention model-dimension map, got %q", message)
	}
}

func TestRunFailsWhenLocalHostProbeAppearsOutsideHostInventory(t *testing.T) {
	fixture := newValidationFixtureRepo(t)
	writeValidationScenarioSource(t, fixture, "example", "api/internal/metrics/memory.go",
		"package metrics\n\nimport \"os\"\n\nfunc readMemory() ([]byte, error) {\n\treturn os.ReadFile(\"/proc/meminfo\")\n}\n")

	report, err := Run(fixture.Root)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if report.Success {
		t.Fatalf("expected failure, got success: %+v", report.Checks)
	}
	if !hasFailedCheck(report, "host_inventory_authority") {
		t.Fatalf("expected host_inventory_authority failure, got %+v", report.Checks)
	}
	message := failedCheckMessage(report, "host_inventory_authority")
	if !strings.Contains(message, "scenarios/example/api/internal/metrics/memory.go:6") || !strings.Contains(message, "proc_meminfo") {
		t.Fatalf("expected message to identify proc_meminfo violation, got %q", message)
	}
}

func TestRunAllowsMarkedRemoteHostSnapshotParser(t *testing.T) {
	fixture := newValidationFixtureRepo(t)
	writeValidationScenarioSource(t, fixture, "example", "api/internal/vps/metrics.go",
		"package vps\n\nfunc remoteCommands() []string {\n\t// hostinventory:remote-snapshot-parser\n\treturn []string{\"cat /proc/meminfo 2>/dev/null\", \"cat /proc/loadavg 2>/dev/null\"}\n}\n")

	report, err := Run(fixture.Root)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if hasFailedCheck(report, "host_inventory_authority") {
		t.Fatalf("marked remote parser should pass host_inventory_authority: %+v", report.Checks)
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
- `+"`make hygiene` remains the CI/automation entrypoint"+`

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
