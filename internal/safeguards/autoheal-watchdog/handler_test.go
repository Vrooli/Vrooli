package autohealwatchdog

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/vrooli/vrooli/internal/hostreqkit"
	"github.com/vrooli/vrooli/internal/hostreqspec"
)

// installedUnit writes the rendered unit for root/home so the definition
// reads as current, and returns the unit path.
func installedUnit(t *testing.T, root, home string) string {
	t.Helper()
	definition, path, _, err := nativeDefinition("linux", root, home)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(definition), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

// healthyUnit answers systemctl probes for an enabled, active unit with a
// clean run state; mainPID is reported when non-empty.
func healthyUnit(mainPID string) func(name string, args ...string) ([]byte, error) {
	return func(name string, args ...string) ([]byte, error) {
		if name == "loginctl" {
			return []byte("Linger=yes\n"), nil
		}
		joined := strings.Join(args, " ")
		if strings.HasSuffix(joined, "is-enabled vrooli-autoheal.service") {
			return []byte("enabled\n"), nil
		}
		if strings.Contains(joined, "show vrooli-autoheal.service") {
			state := "ActiveState=active\nNRestarts=0\nResult=success\n"
			if mainPID != "" {
				state += "MainPID=" + mainPID + "\n"
			}
			return []byte(state), nil
		}
		return []byte("active\n"), nil
	}
}

func dedicatedRequirement() hostreqspec.ResolvedRequirement {
	return hostreqspec.ResolvedRequirement{Name: "autoheal_watchdog", Kind: hostreqspec.KindSafeguard, Required: true, Config: map[string]any{"boot_policy": "dedicated"}}
}

func linuxHost() hostreqkit.Host { return hostreqkit.Host{OS: "linux", SupportsSystemd: true} }

func TestInspectDedicatedRequiresLinuxLingering(t *testing.T) {
	root, loop := writeLoopComponent(t)
	stampLoopManifest(t, root, loop)
	home := t.TempDir()
	installedUnit(t, root, home)
	restore := testSeams(root, home, func(name string, args ...string) ([]byte, error) {
		if name == "loginctl" {
			return []byte("Linger=no\n"), nil
		}
		if strings.HasSuffix(strings.Join(args, " "), "is-enabled vrooli-autoheal.service") {
			return []byte("enabled\n"), nil
		}
		return []byte("active\n"), nil
	})
	defer restore()

	status := NewHandler(hostreqkit.SafeguardManifest{Name: "autoheal_watchdog"}).Inspect(linuxHost(), dedicatedRequirement())
	if status.Applied || !strings.Contains(strings.Join(status.Notes, " "), "boot protection is incomplete") {
		t.Fatalf("status=%+v, want incomplete dedicated boot protection", status)
	}
}

func TestInspectHeadlessMacDoesNotRequireGuiLaunchAgent(t *testing.T) {
	original := commandOutputFn
	commandOutputFn = func(name string, args ...string) ([]byte, error) {
		if name == "launchctl" && strings.HasPrefix(strings.Join(args, " "), "print gui/") {
			return nil, os.ErrNotExist
		}
		return []byte(""), nil
	}
	t.Cleanup(func() { commandOutputFn = original })

	status := NewHandler(hostreqkit.SafeguardManifest{Name: "autoheal_watchdog"}).Inspect(
		hostreqkit.Host{OS: "macos"},
		hostreqspec.ResolvedRequirement{Name: "autoheal_watchdog", Kind: hostreqspec.KindSafeguard, Required: true},
	)
	if status.ExecutionState != hostreqkit.ExecutionNotApplicable || status.SupportClass != hostreqkit.SupportNotApplicable {
		t.Fatalf("status=%+v, want headless macOS not-applicable", status)
	}
}

func TestAsUserArgsTargetsInvokingUserBus(t *testing.T) {
	uid, _, ok := hostreqkit.InvokingUserIDs()
	if !ok {
		t.Skip("invoking user uid is unavailable in this test environment")
	}
	name, args := asUserArgs("systemctl", "--user", "is-active", "vrooli-autoheal.service")
	joined := name + " " + strings.Join(args, " ")
	if !strings.Contains(joined, "XDG_RUNTIME_DIR=/run/user/"+strconv.Itoa(uid)) {
		t.Fatalf("scheduler command %q does not target invoking user runtime", joined)
	}
	if !strings.Contains(joined, "DBUS_SESSION_BUS_ADDRESS=unix:path=/run/user/"+strconv.Itoa(uid)+"/bus") {
		t.Fatalf("scheduler command %q does not target invoking user bus", joined)
	}
}

func TestInspectSharedDoesNotRequireLingering(t *testing.T) {
	root, loop := writeLoopComponent(t)
	stampLoopManifest(t, root, loop)
	home := t.TempDir()
	installedUnit(t, root, home)
	restore := testSeams(root, home, func(name string, args ...string) ([]byte, error) {
		if strings.HasSuffix(strings.Join(args, " "), "is-enabled vrooli-autoheal.service") {
			return []byte("enabled\n"), nil
		}
		return []byte("active\n"), nil
	})
	defer restore()

	status := NewHandler(hostreqkit.SafeguardManifest{Name: "autoheal_watchdog"}).Inspect(linuxHost(), hostreqspec.ResolvedRequirement{Name: "autoheal_watchdog", Kind: hostreqspec.KindSafeguard, Required: true, Config: map[string]any{"boot_policy": "shared"}})
	if !status.Applied {
		t.Fatalf("status=%+v, want shared policy satisfied", status)
	}
}

func testSeams(root, home string, command func(string, ...string) ([]byte, error)) func() {
	originalRoot, originalHome, originalCommand, originalLinger, originalProc := resolveRootFn, userHomeFn, commandOutputFn, lingeringEnabledFn, procRoot
	resolveRootFn = func() (string, error) { return root, nil }
	userHomeFn = func() (string, error) { return home, nil }
	commandOutputFn = command
	lingeringEnabledFn = func(string) bool { return false }
	procRoot = filepath.Join(root, "proc-fixture-unset")
	return func() {
		resolveRootFn, userHomeFn, commandOutputFn, lingeringEnabledFn, procRoot = originalRoot, originalHome, originalCommand, originalLinger, originalProc
	}
}

// [REQ:BOOT-RECOVERY-001] An enabled, active unit whose content drifted from
// the rendered definition is not applied: that was this host's state on
// 2026-09-02 and every signal read green.
func TestInspectStaleDefinitionIsNotApplied(t *testing.T) {
	root, loop := writeLoopComponent(t)
	stampLoopManifest(t, root, loop)
	home := t.TempDir()
	_, path, _, err := nativeDefinition("linux", root, home)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("[Service]\nExecStart=/old/loop --no-stale-check\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	restore := testSeams(root, home, healthyUnit(""))
	defer restore()
	lingeringEnabledFn = func(string) bool { return true }

	status := NewHandler(hostreqkit.SafeguardManifest{Name: "autoheal_watchdog"}).Inspect(linuxHost(), dedicatedRequirement())
	if status.Applied || !strings.Contains(strings.Join(status.Notes, " "), "missing or stale") {
		t.Fatalf("status=%+v, want not applied with a stale-definition note", status)
	}
}

// A fully healthy unit whose binary the engine's manifest reports stale is
// not applied: the engine, not an mtime walk, is the freshness authority.
func TestInspectReportsStaleWhenManifestSaysSo(t *testing.T) {
	root, loop := writeLoopComponent(t)
	stampLoopManifest(t, root, loop)
	if err := os.WriteFile(filepath.Join(loopModuleDir(root), "main.go"), []byte("package main\n\n// a fix that never shipped\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	home := t.TempDir()
	installedUnit(t, root, home)
	restore := testSeams(root, home, healthyUnit(""))
	defer restore()
	lingeringEnabledFn = func(string) bool { return true }

	status := NewHandler(hostreqkit.SafeguardManifest{Name: "autoheal_watchdog"}).Inspect(linuxHost(), dedicatedRequirement())
	if status.Applied {
		t.Fatalf("status=%+v, want not applied when the manifest says stale", status)
	}
	notes := strings.Join(status.Notes, " ")
	if !strings.Contains(notes, "binary is stale") {
		t.Fatalf("notes=%q, want a stale-binary note", notes)
	}
	evidence, _ := status.Evidence["loop_freshness"].(map[string]any)
	if evidence["verdict"] != verdictStale {
		t.Fatalf("loop_freshness evidence=%v, want stale verdict", evidence)
	}
}

// A binary with no manifest is unknown, and unknown is not applied: the
// safeguard must not vouch for a build the engine never stamped.
func TestInspectReportsUnknownWithoutManifest(t *testing.T) {
	root, _ := writeLoopComponent(t)
	home := t.TempDir()
	installedUnit(t, root, home)
	restore := testSeams(root, home, healthyUnit(""))
	defer restore()
	lingeringEnabledFn = func(string) bool { return true }

	status := NewHandler(hostreqkit.SafeguardManifest{Name: "autoheal_watchdog"}).Inspect(linuxHost(), dedicatedRequirement())
	if status.Applied {
		t.Fatalf("status=%+v, want not applied without a manifest", status)
	}
	notes := strings.Join(status.Notes, " ")
	if !strings.Contains(notes, "freshness is unknown") || !strings.Contains(notes, "binary is unknown") {
		t.Fatalf("notes=%q, want unknown-freshness notes", notes)
	}
	evidence, _ := status.Evidence["loop_freshness"].(map[string]any)
	if evidence["verdict"] != verdictUnknown {
		t.Fatalf("loop_freshness evidence=%v, want unknown verdict", evidence)
	}
}

// The binary on disk is fresh and stamped, but the unit's main process was
// exec'd from an older build: setup rebuilt the file and nothing restarted
// the unit. The process protecting boot is the old one, so not applied.
func TestInspectDetectsRunningProcessOlderThanBinary(t *testing.T) {
	root, loop := writeLoopComponent(t)
	stampLoopManifest(t, root, loop)
	home := t.TempDir()
	installedUnit(t, root, home)
	restore := testSeams(root, home, healthyUnit("4242"))
	defer restore()
	lingeringEnabledFn = func(string) bool { return true }
	procRoot = filepath.Join(root, "proc")
	if err := os.MkdirAll(filepath.Join(procRoot, "4242"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(procRoot, "4242", "exe"), []byte("older binary image"), 0o755); err != nil {
		t.Fatal(err)
	}

	status := NewHandler(hostreqkit.SafeguardManifest{Name: "autoheal_watchdog"}).Inspect(linuxHost(), dedicatedRequirement())
	if status.Applied {
		t.Fatalf("status=%+v, want not applied when the process is older than the binary", status)
	}
	notes := strings.Join(status.Notes, " ")
	if !strings.Contains(notes, "process older than binary") {
		t.Fatalf("notes=%q, want a process-older-than-binary note", notes)
	}
	evidence, _ := status.Evidence["process_identity"].(map[string]any)
	if evidence["pid"] != "4242" || evidence["match"] != false {
		t.Fatalf("process_identity evidence=%v, want pid 4242 mismatch", evidence)
	}

	// Once the process runs the on-disk image the same host is applied.
	if err := os.WriteFile(filepath.Join(procRoot, "4242", "exe"), []byte("binary"), 0o755); err != nil {
		t.Fatal(err)
	}
	status = NewHandler(hostreqkit.SafeguardManifest{Name: "autoheal_watchdog"}).Inspect(linuxHost(), dedicatedRequirement())
	if !status.Applied {
		t.Fatalf("status=%+v, want applied once the process matches the binary", status)
	}
}

// Apply rebuilds only through the lifecycle engine and restarts the unit when
// the rebuild changed the binary.
func TestApplyRebuildsThroughLifecycleAndRestartsOnChange(t *testing.T) {
	root, loop := writeLoopComponent(t)
	home := t.TempDir()
	installedUnit(t, root, home)
	var ran []string
	restore := testSeams(root, home, healthyUnit(""))
	defer restore()
	lingeringEnabledFn = func(string) bool { return true }
	originalSetup, originalRun := setupScenarioFn, hostreqkit.RunCommandFn
	setupScenarioFn = func(gotRoot, gotHome string, _ hostreqkit.EnsureOptions) error {
		if gotRoot != root || gotHome != home {
			t.Fatalf("setup invoked with root=%q home=%q", gotRoot, gotHome)
		}
		// The engine builds a new binary and stamps its manifest.
		if err := os.WriteFile(loop, []byte("rebuilt binary"), 0o755); err != nil {
			return err
		}
		stampLoopManifest(t, root, loop)
		return nil
	}
	hostreqkit.RunCommandFn = func(name string, args []string, _ hostreqkit.EnsureOptions) error {
		ran = append(ran, name+" "+strings.Join(args, " "))
		return nil
	}
	t.Cleanup(func() { setupScenarioFn, hostreqkit.RunCommandFn = originalSetup, originalRun })

	h := NewHandler(hostreqkit.SafeguardManifest{Name: "autoheal_watchdog"})
	before := h.Inspect(linuxHost(), dedicatedRequirement())
	if before.Applied {
		t.Fatalf("unstamped binary must not be applied before setup: %+v", before)
	}
	after, err := h.Apply(linuxHost(), before, hostreqkit.EnsureOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if !after.Applied {
		t.Fatalf("status=%+v, want applied after the engine rebuilt the loop", after)
	}
	joined := strings.Join(ran, "\n")
	if !strings.Contains(joined, "restart vrooli-autoheal.service") {
		t.Fatalf("commands=%q, want a unit restart after the binary changed", joined)
	}
	if strings.Contains(joined, "go ") {
		t.Fatalf("commands=%q, the safeguard must never run go build itself", joined)
	}
	notes := strings.Join(after.Notes, " ")
	if !strings.Contains(notes, "rebuilt autoheal loop binary through the lifecycle engine") {
		t.Fatalf("notes=%q, want a lifecycle rebuild note", notes)
	}
}
