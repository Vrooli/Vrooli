package lifecycle

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/vrooli/cli-core/cliutil"
	"github.com/vrooli/vrooli/internal/scenario"
)

func TestAgentManagerUIOnlySetupBuildsUIWithoutOwnerAPISetup(t *testing.T) {
	t.Setenv(cliutil.EnvIdentityToken, "identified-ui-setup-test")
	t.Setenv("VROOLI_SETUP_COMPONENTS", "ui")
	root, home := t.TempDir(), t.TempDir()
	manifest := lifecycleFixtureManifest("agent-manager")
	manifest.Components = map[string]scenario.Component{
		"api": {Role: "api", Build: scenario.ComponentBuild{Kind: "go_module", Dir: "api", Output: "api/mock-api"}, Run: scenario.ComponentRun{Argv: []string{"{{bin.api}}"}, CWD: "api", Port: "api"}},
		"ui":  {Role: "ui", Build: scenario.ComponentBuild{Kind: "go_module", Dir: "ui", Output: "ui/mock-ui"}, Run: scenario.ComponentRun{Argv: []string{"{{bin.ui}}"}, CWD: "ui", Port: "ui"}},
	}
	writeLifecycleFixtureManifest(t, root, manifest)
	uiDir := filepath.Join(root, "scenarios", "agent-manager", "ui")
	if err := os.MkdirAll(uiDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(uiDir, "go.mod"), []byte("module fixture-ui\n\ngo 1.25.0\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(uiDir, "main.go"), []byte("package main\nfunc main() {}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	runner := newLifecycleRunnerForTest(t, root, home, func(deps *lifecycleDeps) {
		deps.readOwnerMaintenance = func(context.Context, scenario.Scenario) (ownerMaintenanceStanding, error) {
			t.Fatal("UI-only setup must not consult the API owner fence")
			return ownerMaintenanceStanding{}, nil
		}
		deps.signalPID = func(int, bool) error { t.Fatal("setup must not stop an owner process"); return nil }
		deps.signalProcessGroup = func(int, bool) error { t.Fatal("setup must not stop an owner process group"); return nil }
	})
	if _, err := runner.RunPhaseDetailed("agent-manager", "setup", PhaseOptions{}); err != nil {
		t.Fatalf("UI-only setup failed: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "scenarios", "agent-manager", "ui", "mock-ui")); err != nil {
		t.Fatalf("UI output was not built: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "scenarios", "agent-manager", "api", "mock-api")); !os.IsNotExist(err) {
		t.Fatalf("API output exists after UI-only setup: %v", err)
	}
}

func TestAgentManagerFullAndAPISetupKeepOwnerFence(t *testing.T) {
	for _, selection := range []struct{ name, value string }{{name: "full"}, {name: "api", value: "api"}} {
		t.Run(selection.name, func(t *testing.T) {
			t.Setenv(cliutil.EnvIdentityToken, "identified-api-setup-test")
			t.Setenv("VROOLI_SETUP_COMPONENTS", selection.value)
			root, home := t.TempDir(), t.TempDir()
			manifest := lifecycleFixtureManifest("agent-manager")
			manifest.Components["ui"] = scenario.Component{Role: "ui", Build: scenario.ComponentBuild{Kind: "go_module", Dir: "ui", Output: "ui/mock-ui"}}
			writeLifecycleFixtureManifest(t, root, manifest)
			runner := newLifecycleRunnerForTest(t, root, home, func(deps *lifecycleDeps) {
				deps.readOwnerMaintenance = func(context.Context, scenario.Scenario) (ownerMaintenanceStanding, error) {
					t.Fatal("identified executor must be refused before owner read")
					return ownerMaintenanceStanding{}, nil
				}
			})
			if _, err := runner.RunPhaseDetailed("agent-manager", "setup", PhaseOptions{}); !errors.Is(err, ErrOwnerMaintenanceRequired) {
				t.Fatalf("setup error=%v, want owner maintenance refusal", err)
			}
			for _, output := range []string{"api/mock-api", "ui/mock-ui"} {
				if _, err := os.Stat(filepath.Join(root, "scenarios", "agent-manager", output)); !os.IsNotExist(err) {
					t.Fatalf("output %s exists after refused setup: %v", output, err)
				}
			}
		})
	}
}

func TestAgentManagerUISelectionWithAuthoredSetupCommandKeepsOwnerFence(t *testing.T) {
	t.Setenv(cliutil.EnvIdentityToken, "identified-ui-setup-test")
	t.Setenv("VROOLI_SETUP_COMPONENTS", "ui")
	root, home := t.TempDir(), t.TempDir()
	manifest := lifecycleFixtureManifest("agent-manager")
	manifest.Components["ui"] = scenario.Component{Role: "ui", Build: scenario.ComponentBuild{Kind: "go_module", Dir: "ui", Output: "ui/mock-ui"}}
	marker := filepath.Join(t.TempDir(), "setup-command-ran")
	manifest.Lifecycle.Setup.Steps = []scenario.PhaseStep{{
		Name: "must-not-run-before-maintenance-admission",
		Exec: []string{"sh", "-c", `touch "$MARKER"`},
		Env:  map[string]string{"MARKER": marker},
	}}
	writeLifecycleFixtureManifest(t, root, manifest)
	runner := newLifecycleRunnerForTest(t, root, home, func(deps *lifecycleDeps) {
		deps.readOwnerMaintenance = func(context.Context, scenario.Scenario) (ownerMaintenanceStanding, error) {
			t.Fatal("authored setup command must retain the owner fence")
			return ownerMaintenanceStanding{}, nil
		}
	})
	if _, err := runner.RunPhaseDetailed("agent-manager", "setup", PhaseOptions{}); !errors.Is(err, ErrOwnerMaintenanceRequired) {
		t.Fatalf("setup error=%v, want owner maintenance refusal", err)
	}
	if _, err := os.Stat(marker); !os.IsNotExist(err) {
		t.Fatalf("authored setup command ran before owner admission: %v", err)
	}
}
