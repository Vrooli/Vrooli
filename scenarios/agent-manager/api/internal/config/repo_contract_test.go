package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/vrooli/repo-contract-go/repocontracttest"
)

func TestResolveOrchestrationSettingsPathCanonicalizesContractDescendant(t *testing.T) {
	root := newConfigContractFixtureRepo(t)
	nested := filepath.Join(root, "scenarios", "agent-manager", "api")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatalf("mkdir nested: %v", err)
	}

	t.Setenv("PROJECT_ROOT", "")
	t.Setenv("ORCHESTRATION_SETTINGS_PATH", "")
	t.Setenv("VROOLI_SOURCE_ROOT", nested)
	t.Setenv("VROOLI_ROOT", "")

	got, err := ResolveOrchestrationSettingsPath()
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(root, "scenarios", "agent-manager", "config", "orchestration.json")
	if got != want {
		t.Fatalf("ResolveOrchestrationSettingsPath() = %q, want %q", got, want)
	}
}

func newConfigContractFixtureRepo(t *testing.T) string {
	t.Helper()

	root := t.TempDir()
	repoRoot := configRepoRoot(t)
	contractData, err := os.ReadFile(filepath.Join(repoRoot, ".vrooli", "repo-contract.json"))
	if err != nil {
		t.Fatalf("read repo contract: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(root, ".vrooli"), 0o755); err != nil {
		t.Fatalf("mkdir .vrooli: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, ".vrooli", "repo-contract.json"), contractData, 0o644); err != nil {
		t.Fatalf("write repo contract: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/agent-manager-config-test\n\ngo 1.24.0\n"), 0o644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}
	for _, dir := range []string{"templates", "scenarios", "resources", "packages", "cmd", "internal"} {
		if err := os.MkdirAll(filepath.Join(root, dir), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", dir, err)
		}
	}
	return root
}

func configRepoRoot(t *testing.T) string {
	t.Helper()
	return repocontracttest.ProjectRoot(t)
}

func TestCanonicalConfigurationUsesRetainedOwnerSource(t *testing.T) {
	fixture := repocontracttest.NewRepoFixture(t, repocontracttest.WithScenarioDir("apps"))
	fixture.WriteRepoContract(t)
	fixture.WriteScenarioStub(t, "agent-manager")
	source := filepath.Join(fixture.Root, "apps", "agent-manager")
	retained := t.TempDir()
	for _, key := range []string{"PROJECT_ROOT", "VROOLI_SOURCE_ROOT", "VROOLI_ROOT", "AGENT_MANAGER_CONFIG", "AGENT_MANAGER_CONVERSATION_POLICY_FILE", "ORCHESTRATION_SETTINGS_PATH"} {
		t.Setenv(key, "")
	}
	t.Setenv("SCENARIO_NAME", "agent-manager")
	t.Setenv("VROOLI_SCENARIO", "agent-manager")
	t.Setenv("SCENARIO_PATH", source)
	t.Setenv("VROOLI_SCENARIO_DIR", source)
	state := t.TempDir()
	t.Setenv("SCENARIO_DATA_DIR", state)
	repocontracttest.WriteFile(t, filepath.Join(source, ".vrooli", "conversation-policy.json"), `{"rules":[{"name":"canonical-private","action":"exclude","enforceAt":"import"}]}`)
	t.Chdir(retained)
	got, err := RepoRoot()
	if err != nil || got != fixture.Root {
		t.Fatalf("root=%q err=%v", got, err)
	}
	settings, err := ResolveOrchestrationSettingsPath()
	if err != nil || settings != filepath.Join(source, "config", "orchestration.json") {
		t.Fatalf("settings=%q err=%v", settings, err)
	}
	levers, err := LoadLevers()
	if err != nil {
		t.Fatal(err)
	}
	if len(levers.ConversationPolicy.Rules) != 1 || levers.ConversationPolicy.Rules[0].Name != "canonical-private" {
		t.Fatalf("canonical conversation rules not loaded: %+v", levers.ConversationPolicy)
	}
	if cwd, _ := os.Getwd(); cwd != retained || os.Getenv("SCENARIO_DATA_DIR") != state {
		t.Fatal("configuration resolution changed runtime or state roots")
	}
}

func TestCanonicalConfigurationOverridesAndInvalidRoots(t *testing.T) {
	root := newConfigContractFixtureRepo(t)
	t.Setenv("PROJECT_ROOT", root)
	t.Setenv("VROOLI_SOURCE_ROOT", filepath.Join(root, "missing"))
	got, err := RepoRoot()
	if err != nil || got != root {
		t.Fatalf("explicit root=%q err=%v", got, err)
	}
	t.Setenv("PROJECT_ROOT", filepath.Join(root, "missing"))
	t.Setenv("AGENT_MANAGER_CONFIG", "")
	t.Setenv("AGENT_MANAGER_CONVERSATION_POLICY_FILE", "")
	t.Setenv("ORCHESTRATION_SETTINGS_PATH", "")
	if _, err := RepoRoot(); err == nil {
		t.Fatal("invalid explicit root accepted")
	}
	if _, err := ResolveOrchestrationSettingsPath(); err == nil {
		t.Fatal("invalid root became a writable relative settings path")
	}
	if _, err := LoadLevers(); err == nil {
		t.Fatal("invalid root silently dropped conversation policy")
	}
	override := filepath.Join(t.TempDir(), "conversation-policy.json")
	repocontracttest.WriteFile(t, override, `{"rules":[{"name":"file-override","action":"exclude","enforceAt":"import"}]}`)
	t.Setenv("AGENT_MANAGER_CONVERSATION_POLICY_FILE", override)
	levers, err := LoadLevers()
	if err != nil || len(levers.ConversationPolicy.Rules) != 1 || levers.ConversationPolicy.Rules[0].Name != "file-override" {
		t.Fatalf("explicit policy override failed: %v", err)
	}
	t.Setenv("ORCHESTRATION_SETTINGS_PATH", override)
	path, err := ResolveOrchestrationSettingsPath()
	if err != nil || path != override {
		t.Fatalf("explicit settings override=%q err=%v", path, err)
	}
}
