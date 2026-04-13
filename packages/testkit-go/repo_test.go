package testkitgo

import (
	"os"
	"path/filepath"
	"testing"

	repocontract "github.com/vrooli/repo-contract-go"
)

func TestNewRepoFixtureUsesDefaultScenarioDir(t *testing.T) {
	fixture := NewRepoFixture(t)
	if fixture.ScenarioDir != "scenarios" {
		t.Fatalf("ScenarioDir = %q, want scenarios", fixture.ScenarioDir)
	}
	if fixture.Root == "" || fixture.Home == "" {
		t.Fatalf("fixture = %+v", fixture)
	}
}

func TestWriteRepoContractSupportsScenarioDirOverride(t *testing.T) {
	fixture := NewRepoFixture(t, WithScenarioDir("apps"))
	fixture.WriteRepoContract(t)

	if _, err := os.Stat(filepath.Join(fixture.Root, "apps")); err != nil {
		t.Fatalf("expected apps dir: %v", err)
	}

	contract, err := repocontract.LoadDefault(fixture.Root)
	if err != nil {
		t.Fatalf("LoadDefault(%s): %v", fixture.Root, err)
	}
	if got := contract.Scenario().WellKnownPaths["service"]; got != ".vrooli/service.json" {
		t.Fatalf("service path = %q", got)
	}
	if got := contract.SandboxScenarioScopePrefix(); got != "apps/" {
		t.Fatalf("SandboxScenarioScopePrefix() = %q, want apps/", got)
	}
}

func TestWriteRepoSupportDocsCreatesExpectedFiles(t *testing.T) {
	fixture := NewRepoFixture(t)
	fixture.WriteRepoSupportDocs(t, DefaultRepoSupportDocs())

	for _, rel := range []string{
		"docs/repo-contract.md",
		"docs/CONTRIBUTING.md",
		"AGENTS.md",
		"scenarios/prompt-manager/store/skills/packs/core/cross-platform-readiness/SKILL.md",
	} {
		if _, err := os.Stat(filepath.Join(fixture.Root, filepath.FromSlash(rel))); err != nil {
			t.Fatalf("expected %s: %v", rel, err)
		}
	}
}
