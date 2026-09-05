package validation

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"unit-health/internal/discovery"
	"unit-health/internal/executor"
)

type fakeExecutorByName struct {
	byName map[string]executor.Result
}

func (f fakeExecutorByName) Run(_ context.Context, cmd executor.Command) executor.Result {
	r := f.byName[cmd.Name]
	r.WorkspaceID = cmd.WorkspaceID
	r.Name = cmd.Name
	return r
}

func TestBuildExecutionPlanUsesTypedCommandsAndNeverInstallsDependencies(t *testing.T) {
	root := t.TempDir()
	ui := filepath.Join(root, "ui")
	if err := writeFileForTypedPlan(filepath.Join(ui, "package.json"), `{
  "scripts": {"test": "vitest run", "test:coverage": "vitest run --coverage"},
  "devDependencies": {"vitest": "^3.0.0"}
}`); err != nil {
		t.Fatal(err)
	}
	inv := discovery.Inventory{Scenario: "demo", TargetKind: "scenario", RootPath: root, Surfaces: []discovery.Surface{{
		ID: "ui", Kind: "ui", Language: "typescript", RootPath: ui, PackageManager: "pnpm", Status: "known",
	}}}
	_, workspaces, plan, _ := buildPlan("demo", inv, "2026-08-21T00:00:00Z")
	if len(workspaces) != 1 || len(plan.Commands) != 1 {
		t.Fatalf("workspaces=%+v plan=%+v", workspaces, plan)
	}
	if workspaces[0].AdapterID != "react-vitest" || workspaces[0].AdapterVersion == "" || workspaces[0].TestKind != "unit" {
		t.Fatalf("adapter projection=%+v", workspaces[0])
	}
	cmd := plan.Commands[0]
	if !strings.HasPrefix(strings.ToLower(filepath.Base(cmd.Executable)), "pnpm") || len(cmd.Args) != 2 || cmd.Args[0] != "run" || cmd.Args[1] != "test:coverage" {
		t.Fatalf("typed command=%+v", cmd)
	}
	if cmd.Kind != kindTest {
		t.Fatalf("kind=%q, want %q", cmd.Kind, kindTest)
	}
	if cmd.Executable == "npm" || cmd.Executable == "yarn" || cmd.Executable == "pnpm" && cmd.Args[0] == "install" {
		t.Fatalf("dependency installation leaked into validation plan: %+v", cmd)
	}
}

func TestBuildExecutionPlanProjectsDeclaredRunnerProfile(t *testing.T) {
	root := t.TempDir()
	apiRoot := filepath.Join(root, "api")
	if err := writeFileForTypedPlan(filepath.Join(apiRoot, "go.mod"), "module example.test\n\ngo 1.25\n"); err != nil {
		t.Fatal(err)
	}
	if err := writeFileForTypedPlan(filepath.Join(root, ".vrooli", "testing.json"), `{
  "unit": {"policy_profile": {
    "version": "2.0.0",
    "template": {"id": "go", "scenario_class": "go"},
    "required_roles": [{"role": "api", "policy_class": "go_api", "match": {"surface_id": "api", "kind": "api", "language": "go"}}],
    "policy_classes": {"go_api": {"language": "go", "framework": "go test", "adapter": {"id": "go", "version": "1.0.0"}, "runner_profile": "bounded", "test_kind": "unit", "hermetic": {"network": "deny", "filesystem": "temporary_root", "restore_environment": true}}},
    "runner_profiles": {"bounded": {"cpu_weight": 2, "memory_bytes": 1048576, "max_workers": 2, "timeout_seconds": 17, "no_output_timeout_seconds": 5, "network": "deny", "filesystem": "temporary_root"}},
    "customization": {"mode": "monotonic", "waivers": []}
  }}
}`); err != nil {
		t.Fatal(err)
	}
	inv := discovery.Inventory{Scenario: "demo", TargetKind: "scenario", RootPath: root, Surfaces: []discovery.Surface{{
		ID: "api", Kind: "api", Language: "go", RootPath: apiRoot, Status: "known",
	}}}
	_, workspaces, plan, findings := buildPlan("demo", inv, "2026-08-21T00:00:00Z")
	if len(findings) != 0 {
		t.Fatalf("unexpected profile findings: %+v", findings)
	}
	if len(workspaces) != 1 || len(plan.Commands) != 1 {
		t.Fatalf("workspaces=%+v plan=%+v", workspaces, plan)
	}
	ws := workspaces[0]
	if ws.RunnerProfile != "bounded" || ws.Resource.MaxWorkers != 2 || ws.TimeoutSeconds != 17 || ws.NoOutputTimeoutSeconds != 5 {
		t.Fatalf("runner projection=%+v", ws)
	}
	if ws.Hermetic.Network != "deny" || ws.Hermetic.Filesystem != "temporary_root" || !ws.Hermetic.RestoreEnvironment {
		t.Fatalf("hermetic projection=%+v", ws.Hermetic)
	}
}

func writeFileForTypedPlan(path, contents string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(contents), 0o644)
}
