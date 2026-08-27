//go:build !windows

package codingagentshims

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vrooli/cli-core/cliutil"
	"github.com/vrooli/vrooli/internal/hostreqkit"
	"github.com/vrooli/vrooli/internal/hostreqspec"
	"github.com/vrooli/vrooli/internal/shell/shelltest"
)

// newHandlerForTest redirects HOME so the safeguard installs into a temporary
// tree, and returns the shim directory it will use.
func newHandlerForTest(t *testing.T) (hostreqkit.Handler, string, string) {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	binDir := filepath.Join(home, ".vrooli", "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatalf("mkdir bin: %v", err)
	}
	return NewHandler(hostreqkit.SafeguardManifest{Name: "coding_agent_shims"}), binDir, filepath.Join(home, ".vrooli", "shims")
}

func writeLauncher(t *testing.T, binDir string) string {
	t.Helper()
	launcher := filepath.Join(binDir, launcherBinary)
	if err := os.WriteFile(launcher, []byte(shelltest.POSIXShebang()+"exit 0\n"), 0o755); err != nil {
		t.Fatalf("write launcher: %v", err)
	}
	return launcher
}

func requirement() hostreqspec.ResolvedRequirement {
	return hostreqspec.ResolvedRequirement{Name: "coding_agent_shims"}
}

func TestInspectReportsPendingWhenLauncherIsNotBuilt(t *testing.T) {
	handler, _, _ := newHandlerForTest(t)

	status := handler.Inspect(hostreqkit.Host{OS: "linux"}, requirement())

	if status.Applied {
		t.Fatal("safeguard reported applied with no launcher to link to")
	}
	if !noteContains(status.Notes, "not built yet") {
		t.Fatalf("notes = %v, want one explaining the launcher is not built", status.Notes)
	}
}

func TestShimDirUsesInvokingUserHomeWhenElevated(t *testing.T) {
	origRoot := hostreqkit.RunningAsRootFn
	origHome := shimHomeDir
	defer func() {
		hostreqkit.RunningAsRootFn = origRoot
		shimHomeDir = origHome
	}()
	hostreqkit.RunningAsRootFn = func() bool { return true }
	shimHomeDir = func() (string, error) { return "/home/alice", nil }
	got, err := ShimDir()
	if err != nil {
		t.Fatalf("ShimDir() error = %v", err)
	}
	if got != filepath.Join("/home/alice", ".vrooli", "shims") {
		t.Fatalf("ShimDir() = %q, want invoking user's path", got)
	}
}

func TestApplyInstallsOneAliasPerSupportedAgent(t *testing.T) {
	handler, binDir, shimDir := newHandlerForTest(t)
	launcher := writeLauncher(t, binDir)

	status := handler.Inspect(hostreqkit.Host{OS: "linux"}, requirement())
	status, err := handler.Apply(hostreqkit.Host{OS: "linux"}, status, hostreqkit.EnsureOptions{})
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}
	if !status.Applied {
		t.Fatalf("Apply() did not report applied; notes = %v", status.Notes)
	}

	for _, alias := range cliutil.CodingAgentAliases() {
		path := filepath.Join(shimDir, alias)
		target, err := os.Readlink(path)
		if err != nil {
			t.Errorf("alias %q was not installed: %v", alias, err)
			continue
		}
		if target != launcher {
			t.Errorf("alias %q points at %q, want %q", alias, target, launcher)
		}
	}

	// Inspect must now agree, or setup would reinstall on every run.
	if again := handler.Inspect(hostreqkit.Host{OS: "linux"}, requirement()); !again.Applied {
		t.Fatalf("Inspect() did not see its own work; notes = %v", again.Notes)
	}
}

func TestApplyRepairsAnAliasPointingSomewhereElse(t *testing.T) {
	handler, binDir, shimDir := newHandlerForTest(t)
	launcher := writeLauncher(t, binDir)

	// A leftover from an older launcher path, or an unrelated tool that claimed
	// the name. Either way the safeguard owns this path and must reclaim it.
	if err := os.MkdirAll(shimDir, 0o755); err != nil {
		t.Fatalf("mkdir shims: %v", err)
	}
	stale := filepath.Join(shimDir, "codex")
	if err := os.Symlink("/bin/false", stale); err != nil {
		t.Fatalf("seed stale alias: %v", err)
	}

	status := handler.Inspect(hostreqkit.Host{OS: "linux"}, requirement())
	if status.Applied {
		t.Fatal("Inspect() accepted an alias pointing at the wrong target")
	}
	if !noteContains(status.Notes, "codex") {
		t.Fatalf("notes = %v, want the stale alias named", status.Notes)
	}

	if _, err := handler.Apply(hostreqkit.Host{OS: "linux"}, status, hostreqkit.EnsureOptions{}); err != nil {
		t.Fatalf("Apply() error = %v", err)
	}
	target, err := os.Readlink(stale)
	if err != nil {
		t.Fatalf("readlink after repair: %v", err)
	}
	if target != launcher {
		t.Fatalf("alias still points at %q, want %q", target, launcher)
	}
}

func TestApplyDryRunChangesNothing(t *testing.T) {
	handler, binDir, shimDir := newHandlerForTest(t)
	writeLauncher(t, binDir)

	status := handler.Inspect(hostreqkit.Host{OS: "linux"}, requirement())
	status, err := handler.Apply(hostreqkit.Host{OS: "linux"}, status, hostreqkit.EnsureOptions{DryRun: true})
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}
	if status.ExecutionState != hostreqkit.ExecutionWouldApply {
		t.Fatalf("execution state = %v, want ExecutionWouldApply", status.ExecutionState)
	}
	for _, alias := range cliutil.CodingAgentAliases() {
		if _, err := os.Lstat(filepath.Join(shimDir, alias)); err == nil {
			t.Fatalf("dry run installed alias %q", alias)
		}
	}
}

func TestApplyIsIdempotent(t *testing.T) {
	handler, binDir, shimDir := newHandlerForTest(t)
	writeLauncher(t, binDir)

	for range 3 {
		status := handler.Inspect(hostreqkit.Host{OS: "linux"}, requirement())
		if _, err := handler.Apply(hostreqkit.Host{OS: "linux"}, status, hostreqkit.EnsureOptions{}); err != nil {
			t.Fatalf("Apply() error = %v", err)
		}
	}

	entries, err := os.ReadDir(shimDir)
	if err != nil {
		t.Fatalf("read shim dir: %v", err)
	}
	// Exactly one alias per agent and no staging leftovers, however many times
	// apply runs. The launcher itself stays in the install root.
	if want := len(cliutil.CodingAgentAliases()); len(entries) != want {
		names := make([]string, 0, len(entries))
		for _, entry := range entries {
			names = append(names, entry.Name())
		}
		t.Fatalf("shim dir holds %d entries (%v), want %d", len(entries), names, want)
	}
}

func noteContains(notes []string, substring string) bool {
	for _, note := range notes {
		if strings.Contains(note, substring) {
			return true
		}
	}
	return false
}
