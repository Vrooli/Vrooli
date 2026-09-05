package workspace

import (
	"os"
	"path/filepath"
	"testing"

	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
)

func TestTargetKindNameUsesRepoContractToken(t *testing.T) {
	got := targetKindName(commonv1.ValidationTargetKind_VALIDATION_TARGET_KIND_CONTROL_PLANE)
	if got != "control-plane" {
		t.Fatalf("targetKindName() = %q, want control-plane", got)
	}
}

func TestNewTargetWorkspace(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	t.Setenv("HOME", home)
	scenarioDir := filepath.Join(root, "demo")
	if err := os.MkdirAll(scenarioDir, 0o755); err != nil {
		t.Fatalf("failed to create scenario dir: %v", err)
	}

	workspace, err := New(root, "demo")
	if err != nil {
		t.Fatalf("expected workspace, got error: %v", err)
	}
	if workspace.ScenarioDir != scenarioDir {
		t.Fatalf("unexpected scenario dir %s", workspace.ScenarioDir)
	}
	if workspace.CoverageDir == filepath.Join(scenarioDir, "coverage") {
		t.Fatalf("artifact workspace must not be created in the scenario tree: %s", workspace.CoverageDir)
	}
	expectedPhaseDir := filepath.Join(home, ".vrooli", "test-runs", "demo", "coverage", "phases")
	if workspace.PhaseDir != expectedPhaseDir {
		t.Fatalf("unexpected phase dir %s", workspace.PhaseDir)
	}
	if workspace.AppRoot == "" {
		t.Fatalf("expected app root to be set")
	}

	env := workspace.Environment()
	if env.ScenarioName != "demo" || env.ScenarioDir == "" || env.CoverageDir == "" {
		t.Fatalf("environment missing data: %#v", env)
	}
	if info, err := os.Stat(workspace.CoverageDir); err != nil || !info.IsDir() {
		t.Fatalf("coverage dir missing: %v", err)
	}

	artifactDir, err := workspace.EnsureArtifactDir()
	if err != nil {
		t.Fatalf("artifact dir error: %v", err)
	}
	if info, err := os.Stat(artifactDir); err != nil || !info.IsDir() {
		t.Fatalf("artifact dir missing: %v", err)
	}
}

func TestNewTargetWorkspaceValidatesNames(t *testing.T) {
	root := t.TempDir()
	if _, err := New(root, ""); err == nil {
		t.Fatalf("expected error for empty scenario")
	}
	if _, err := New(root, "invalid name"); err == nil {
		t.Fatalf("expected error for invalid characters")
	}
}

func TestNewTargetWorkspaceWithPhysicalOverride(t *testing.T) {
	root := t.TempDir()
	scenarioDir := filepath.Join(t.TempDir(), "scenarios", "demo")
	if err := os.MkdirAll(scenarioDir, 0o755); err != nil {
		t.Fatalf("failed to create scenario dir: %v", err)
	}

	workspace, err := NewWithOptions(root, "demo", Options{ScenarioPath: scenarioDir})
	if err != nil {
		t.Fatalf("NewWithOptions() error = %v", err)
	}
	if workspace.ScenarioDir != scenarioDir {
		t.Fatalf("ScenarioDir = %q, want %q", workspace.ScenarioDir, scenarioDir)
	}
	if workspace.Mapping.HasLogicalPlacement() {
		t.Fatal("physical-only override should not have logical placement")
	}
}

func TestWorkspaceMappingResolvesMappedLinks(t *testing.T) {
	repoRoot := t.TempDir()
	physicalScenario := filepath.Join(t.TempDir(), "scenarios", "demo")
	if err := os.MkdirAll(filepath.Join(physicalScenario, "docs", "reference"), 0o755); err != nil {
		t.Fatalf("failed to create physical scenario: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(repoRoot, "docs", "reference"), 0o755); err != nil {
		t.Fatalf("failed to create repo docs: %v", err)
	}
	if err := os.WriteFile(filepath.Join(physicalScenario, "docs", "reference", "local.md"), []byte("# Local\n"), 0o644); err != nil {
		t.Fatalf("write local doc: %v", err)
	}
	if err := os.WriteFile(filepath.Join(repoRoot, "docs", "reference", "port-allocation.md"), []byte("# Ports\n"), 0o644); err != nil {
		t.Fatalf("write repo doc: %v", err)
	}

	mapping, err := NewMapping(physicalScenario, filepath.Dir(filepath.Dir(physicalScenario)), repoRoot, "scenarios/demo", "demo")
	if err != nil {
		t.Fatalf("NewMapping() error = %v", err)
	}
	source := filepath.Join(physicalScenario, "docs", "reference", "configuration.md")

	local := mapping.ResolveLocalLink(source, "local.md")
	if !local.Exists || local.OutsideScenario {
		t.Fatalf("local resolution = %#v", local)
	}
	if local.PhysicalPath != filepath.Join(physicalScenario, "docs", "reference", "local.md") {
		t.Fatalf("local physical path = %q", local.PhysicalPath)
	}

	repo := mapping.ResolveLocalLink(source, "../../../../docs/reference/port-allocation.md")
	if !repo.Exists || !repo.OutsideScenario {
		t.Fatalf("repo resolution = %#v", repo)
	}
	if repo.PhysicalPath != filepath.Join(repoRoot, "docs", "reference", "port-allocation.md") {
		t.Fatalf("repo physical path = %q", repo.PhysicalPath)
	}

	escaped := mapping.ResolveLocalLink(source, "../../../../../outside.md")
	if escaped.Exists || !escaped.EscapesRoot {
		t.Fatalf("escaped resolution = %#v", escaped)
	}
}
