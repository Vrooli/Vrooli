package keyringdaemonlimits

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/vrooli/vrooli/internal/hostreqkit"
	"github.com/vrooli/vrooli/internal/hostreqkit/hostreqkittest"
	"github.com/vrooli/vrooli/internal/hostreqspec"
)

const unitText = "# /usr/lib/systemd/user/gnome-keyring-daemon.service\n[Service]\nExecStart=/usr/bin/gnome-keyring-daemon --foreground\n"

func limitsText(soft int) string {
	return "Limit                     Soft Limit           Hard Limit           Units\n" +
		"Max open files            " + strconv.Itoa(soft) + "                 1048576              files\n"
}

func newTestHandler() hostreqkit.Handler {
	return NewHandler(hostreqkit.SafeguardManifest{Name: "keyring_daemon_limits", Handler: "keyring_daemon_limits"})
}

func linuxReq() hostreqspec.ResolvedRequirement {
	return hostreqspec.ResolvedRequirement{Name: "keyring_daemon_limits", Kind: hostreqspec.KindSafeguard, Required: true}
}

func linuxHost() hostreqkit.Host {
	return hostreqkit.Host{OS: "linux", PackageManager: "apt-get", SupportsSysctl: true, SupportsSystemd: true}
}

type fixture struct {
	home     string
	files    map[string]string
	fdCount  int
	pid      string
	unit     bool
	commands []string
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	restore := hostreqkittest.StubLookups(t)
	f := &fixture{home: "/home/op", files: map[string]string{}, pid: "4242", unit: true, fdCount: 10}
	f.files["/proc/4242/limits"] = limitsText(1024)

	origHome, origReadDir, origRoot := homeDir, readDirCountFn, hostreqkit.RunningAsRootFn
	homeDir = func() (string, error) { return f.home, nil }
	readDirCountFn = func(path string) (int, error) {
		if path == filepath.Join("/proc", f.pid, "fd") {
			return f.fdCount, nil
		}
		return 0, os.ErrNotExist
	}
	hostreqkit.RunningAsRootFn = func() bool { return false }
	hostreqkit.ReadFileFn = func(path string) ([]byte, error) {
		if c, ok := f.files[path]; ok {
			return []byte(c), nil
		}
		return nil, os.ErrNotExist
	}
	hostreqkit.CombinedOutputFn = func(name string, args ...string) ([]byte, error) {
		joined := name + " " + strings.Join(args, " ")
		f.commands = append(f.commands, joined)
		switch {
		case strings.Contains(joined, "cat "+unitName):
			if f.unit {
				return []byte(unitText), nil
			}
			return []byte("No files found for " + unitName), os.ErrNotExist
		case strings.Contains(joined, "show -p MainPID"):
			return []byte(f.pid + "\n"), nil
		case strings.Contains(joined, "restart "+unitName):
			// A restarted daemon starts with the new limit and a clean table.
			f.files["/proc/4242/limits"] = limitsText(65536)
			f.fdCount = 10
		}
		return nil, nil
	}
	hostreqkit.RunCommandFn = func(name string, args []string, _ hostreqkit.EnsureOptions) error {
		f.commands = append(f.commands, name+" "+strings.Join(args, " "))
		if name == "install" {
			content, err := os.ReadFile(args[len(args)-2])
			if err != nil {
				return err
			}
			f.files[args[len(args)-1]] = string(content)
		}
		return nil
	}
	t.Cleanup(func() {
		restore()
		homeDir, readDirCountFn, hostreqkit.RunningAsRootFn = origHome, origReadDir, origRoot
	})
	return f
}

func (f *fixture) dropIn() string { return filepath.Join(f.home, dropInDir, dropInName) }

func (f *fixture) ran(substr string) bool {
	for _, c := range f.commands {
		if strings.Contains(c, substr) {
			return true
		}
	}
	return false
}

func TestParseSoftLimit(t *testing.T) {
	if got := parseSoftLimit(limitsText(1024)); got != 1024 {
		t.Fatalf("parseSoftLimit = %d, want 1024", got)
	}
	if got := parseSoftLimit("Max open files            unlimited            unlimited            files\n"); got != 1<<30 {
		t.Fatalf("unlimited parsed as %d", got)
	}
	if got := parseSoftLimit("nothing here"); got != 0 {
		t.Fatalf("missing field parsed as %d", got)
	}
}

func TestResolveSettings(t *testing.T) {
	s := resolveSettings(nil)
	if s.NoFileLimit != 65536 || s.RestartSaturation != 50 {
		t.Fatalf("defaults = %+v", s)
	}
	s = resolveSettings(map[string]any{"nofile_limit": float64(8192), "restart_saturation_percent": "75"})
	if s.NoFileLimit != 8192 || s.RestartSaturation != 75 {
		t.Fatalf("overrides = %+v", s)
	}
	s = resolveSettings(map[string]any{"nofile_limit": 100, "restart_saturation_percent": 500})
	if s.NoFileLimit != 65536 || s.RestartSaturation != 50 {
		t.Fatalf("invalid overrides leaked: %+v", s)
	}
}

func TestInspectPendingWithoutDropIn(t *testing.T) {
	f := newFixture(t)
	status := newTestHandler().Inspect(linuxHost(), linuxReq())
	if status.Applied {
		t.Fatalf("expected pending, notes %v", status.Notes)
	}
	joined := strings.Join(status.Notes, "\n")
	if !strings.Contains(joined, f.dropIn()) || !strings.Contains(joined, "holds 10 of 1024 descriptors (0%)") {
		t.Fatalf("notes = %v", status.Notes)
	}
}

func TestInspectPendingWhenDaemonSaturated(t *testing.T) {
	f := newFixture(t)
	f.files[f.dropIn()] = dropInContent(resolveSettings(nil))
	f.fdCount = 1015
	status := newTestHandler().Inspect(linuxHost(), linuxReq())
	if status.Applied {
		t.Fatalf("a daemon at 99%% must be pending; notes %v", status.Notes)
	}
	if !strings.Contains(strings.Join(status.Notes, "\n"), "99% of its descriptor limit") {
		t.Fatalf("notes = %v", status.Notes)
	}
}

func TestInspectAppliedWhenDropInPresentAndDaemonHealthy(t *testing.T) {
	f := newFixture(t)
	f.files[f.dropIn()] = dropInContent(resolveSettings(nil))
	status := newTestHandler().Inspect(linuxHost(), linuxReq())
	if !status.Applied || status.ExecutionState != hostreqkit.ExecutionAlreadyPresent {
		t.Fatalf("state %q notes %v", status.ExecutionState, status.Notes)
	}
}

func TestInspectNotApplicableWithoutUnit(t *testing.T) {
	f := newFixture(t)
	f.unit = false
	status := newTestHandler().Inspect(linuxHost(), linuxReq())
	if status.ExecutionState != hostreqkit.ExecutionNotApplicable {
		t.Fatalf("state %q", status.ExecutionState)
	}
}

func TestApplyInstallsDropInAndLeavesHealthyDaemonRunning(t *testing.T) {
	f := newFixture(t)
	h := newTestHandler()
	status := h.Inspect(linuxHost(), linuxReq())
	status, err := h.Apply(linuxHost(), status, hostreqkit.EnsureOptions{})
	if err != nil || status.ExecutionState != hostreqkit.ExecutionApplied {
		t.Fatalf("err %v state %q notes %v", err, status.ExecutionState, status.Notes)
	}
	if f.files[f.dropIn()] != dropInContent(resolveSettings(nil)) {
		t.Fatalf("drop-in not installed: %q", f.files[f.dropIn()])
	}
	if !f.ran("daemon-reload") {
		t.Fatalf("daemon-reload not run: %v", f.commands)
	}
	if f.ran("restart " + unitName) {
		t.Fatalf("a healthy daemon at 0%% must not be restarted outside a maintenance window: %v", f.commands)
	}
	if !strings.Contains(strings.Join(status.Notes, "\n"), "binds at its next restart") {
		t.Fatalf("operator not told the limit is not yet live: %v", status.Notes)
	}
}

func TestApplyRestartsSaturatedDaemon(t *testing.T) {
	f := newFixture(t)
	f.fdCount = 1015
	h := newTestHandler()
	status := h.Inspect(linuxHost(), linuxReq())
	status, _ = h.Apply(linuxHost(), status, hostreqkit.EnsureOptions{})
	if status.ExecutionState != hostreqkit.ExecutionApplied {
		t.Fatalf("state %q notes %v", status.ExecutionState, status.Notes)
	}
	if !f.ran("restart " + unitName) {
		t.Fatalf("saturated daemon not restarted: %v", f.commands)
	}
	if !strings.Contains(strings.Join(status.Notes, "\n"), "was at 1015 of 1024") {
		t.Fatalf("notes = %v", status.Notes)
	}
	// After the restart the host inspects clean with the new limit live.
	status = h.Inspect(linuxHost(), linuxReq())
	if !status.Applied || !strings.Contains(strings.Join(status.Notes, "\n"), "of 65536 descriptors") {
		t.Fatalf("post-restart inspect: applied=%v notes %v", status.Applied, status.Notes)
	}
}

func TestApplyRestartsInMaintenanceWindowWhenLimitIsLow(t *testing.T) {
	f := newFixture(t)
	f.files[f.dropIn()] = dropInContent(resolveSettings(nil))
	h := newTestHandler()
	// Inspect says applied (drop-in present, daemon healthy); force Apply anyway
	// as setup does when the operator declares a maintenance window.
	status := hostreqkit.BaseStatus(linuxReq())
	status.SupportClass = hostreqkit.SupportSupported
	status, _ = h.Apply(linuxHost(), status, hostreqkit.EnsureOptions{MaintenanceWindow: true})
	if !f.ran("restart " + unitName) {
		t.Fatalf("maintenance window with a low live limit should restart: %v", f.commands)
	}
	if f.ran("daemon-reload") {
		t.Fatalf("drop-in already present; daemon-reload should be skipped: %v", f.commands)
	}
	_ = status
}

func TestApplyDryRunTouchesNothing(t *testing.T) {
	f := newFixture(t)
	h := newTestHandler()
	status := h.Inspect(linuxHost(), linuxReq())
	before := len(f.commands)
	status, _ = h.Apply(linuxHost(), status, hostreqkit.EnsureOptions{DryRun: true})
	if status.ExecutionState != hostreqkit.ExecutionWouldApply || status.Applied || len(f.commands) != before {
		t.Fatalf("dry-run state %q applied %v commands %v", status.ExecutionState, status.Applied, f.commands[before:])
	}
}
