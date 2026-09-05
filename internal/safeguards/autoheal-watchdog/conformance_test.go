package autohealwatchdog

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/vrooli/vrooli/internal/hostreqkit"
	"github.com/vrooli/vrooli/internal/hostreqkit/hostreqkittest"
	"github.com/vrooli/vrooli/internal/hostreqspec"
)

func TestConformance(t *testing.T) {
	hostreqkittest.RunSuite(t, hostreqkittest.Case{
		NewHandler:         func() hostreqkit.Handler { return NewHandler(hostreqkit.SafeguardManifest{Name: "autoheal_watchdog"}) },
		Name:               "autoheal_watchdog",
		Kind:               hostreqspec.KindSafeguard,
		SupportedPlatforms: []string{"linux", "macos", "windows"},
		Seams:              conformanceSeams,
		Checks:             []string{"name_and_kind", "inspect_reports_validator_verdict", "apply_reverifies"},
	})
}

// conformanceSeams stands up a stamped loop component, an installed unit, a
// healthy fake scheduler, and a running process whose image matches the
// binary, so the shared checks exercise the real Inspect and Apply paths
// without touching the host.
func conformanceSeams(t *testing.T) {
	root, loop := writeLoopComponent(t)
	home := t.TempDir()
	installedUnit(t, root, home)
	stampLoopManifest(t, root, loop)
	restore := testSeams(root, home, healthyUnit(""))
	originalLinger, originalSetup, originalRun, originalProc := lingeringEnabledFn, setupScenarioFn, hostreqkit.RunCommandFn, procRoot
	lingeringEnabledFn = func(string) bool { return true }
	setupScenarioFn = func(string, string, hostreqkit.EnsureOptions) error { return nil }
	hostreqkit.RunCommandFn = func(string, []string, hostreqkit.EnsureOptions) error { return nil }
	procRoot = t.TempDir()
	if err := os.MkdirAll(filepath.Join(procRoot, "4242"), 0o755); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(loop)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(procRoot, "4242", "exe"), data, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		restore()
		lingeringEnabledFn, setupScenarioFn, hostreqkit.RunCommandFn, procRoot = originalLinger, originalSetup, originalRun, originalProc
	})
}
