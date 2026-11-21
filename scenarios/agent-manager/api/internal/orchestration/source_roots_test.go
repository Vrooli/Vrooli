package orchestration

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/vrooli/repo-contract-go/repocontracttest"
)

func TestDeclarationsAndHarnessManifestsUseRetainedOwnerSource(t *testing.T) {
	fixture := repocontracttest.NewRepoFixture(t)
	fixture.WriteRepoContract(t)
	fixture.WriteScenarioStub(t, "agent-manager")
	source := filepath.Join(fixture.Root, "scenarios", "agent-manager")
	other := filepath.Join(fixture.Root, "scenarios", "fixture-scn")
	repocontracttest.WriteFile(t, filepath.Join(other, ".vrooli", "service.json"), declarationManifest(".vrooli/agent-manager/default.json"))
	repocontracttest.WriteFile(t, filepath.Join(other, ".vrooli", "agent-manager", "default.json"), fixtureProfile)
	for _, key := range []string{"PROJECT_ROOT", "VROOLI_SOURCE_ROOT", "VROOLI_ROOT"} {
		t.Setenv(key, "")
	}
	t.Setenv("SCENARIO_NAME", "agent-manager")
	t.Setenv("VROOLI_SCENARIO", "agent-manager")
	t.Setenv("SCENARIO_PATH", source)
	t.Setenv("VROOLI_SCENARIO_DIR", source)
	o := newDeclarationOrchestrator(t)
	t.Chdir(t.TempDir())
	if got := defaultHarnessResourcesDir(); got != filepath.Join(fixture.Root, "resources") {
		t.Fatalf("manifest source=%q", got)
	}
	result, err := o.ReconcileScenarioDeclarations(context.Background(), ReconcileScenarioDeclarationsRequest{Scenario: "fixture-scn", DryRun: true, ValidateOnly: true})
	if err != nil {
		t.Fatal(err)
	}
	if result.ProfilesFailed != 0 || len(result.ProfileResults) != 1 {
		t.Fatalf("declaration result=%+v", result)
	}
	t.Setenv("PROJECT_ROOT", filepath.Join(fixture.Root, "missing"))
	if got := defaultHarnessResourcesDir(); got != "" {
		t.Fatalf("invalid root became %q", got)
	}
	if _, err := o.ReconcileScenarioDeclarations(context.Background(), ReconcileScenarioDeclarationsRequest{Scenario: "fixture-scn", DryRun: true, ValidateOnly: true}); err == nil {
		t.Fatal("declarations accepted invalid root")
	}
}
