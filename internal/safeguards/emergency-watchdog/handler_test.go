package emergencywatchdog

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	repocontract "github.com/vrooli/repo-contract-go"
	"github.com/vrooli/vrooli/internal/hostreqkit"
	"github.com/vrooli/vrooli/internal/hostreqspec"
)

func testManifest() hostreqkit.SafeguardManifest {
	return hostreqkit.SafeguardManifest{Name: "emergency_watchdog"}
}

var linuxHost = emergencyWatchdogLinuxHost

func emergencyWatchdogLinuxHost() hostreqkit.Host {
	return hostreqkit.Host{OS: "linux", SupportsSystemd: true}
}

func req() hostreqspec.ResolvedRequirement {
	return hostreqspec.ResolvedRequirement{Name: "emergency_watchdog", Kind: hostreqspec.KindSafeguard, Required: true}
}

func defaults() settings { return resolveSettings(nil) }

// Handler fallbacks and manifest defaults must agree, or setup and the handler
// disagree about what "unconfigured" means.

func TestInspectNonLinuxReportsMechanism(t *testing.T) {
	status := NewHandler(testManifest()).Inspect(hostreqkit.Host{OS: "darwin"}, req())
	if status.SupportClass != hostreqkit.SupportUnsupported && status.SupportClass != hostreqkit.SupportManualOnly && status.SupportClass != hostreqkit.SupportNotApplicable {
		t.Fatalf("SupportClass = %q, want unsupported, manual-only, or not-applicable", status.SupportClass)
	}
	if !strings.Contains(strings.Join(status.Notes, " "), "native scheduler") && !strings.Contains(strings.Join(status.Notes, " "), "GUI launchd domain") {
		t.Errorf("the unsupported note should name the missing platform mechanism; got %v", status.Notes)
	}
}

func TestInspectRequiresSystemd(t *testing.T) {
	host := linuxHost()
	host.SupportsSystemd = false
	if status := NewHandler(testManifest()).Inspect(host, req()); status.SupportClass != hostreqkit.SupportUnsupported {
		t.Fatalf("SupportClass = %q, want unsupported without systemd", status.SupportClass)
	}
}

func TestTimerProbeTargetsInvokingUserBus(t *testing.T) {
	origRoot := hostreqkit.RunningAsRootFn
	origCombined := hostreqkit.CombinedOutputFn
	defer func() {
		hostreqkit.RunningAsRootFn = origRoot
		hostreqkit.CombinedOutputFn = origCombined
	}()
	hostreqkit.RunningAsRootFn = func() bool { return true }
	os.Setenv("SUDO_USER", "alice")
	os.Setenv("SUDO_UID", "1000")
	os.Setenv("SUDO_GID", "1000")
	defer os.Unsetenv("SUDO_USER")
	defer os.Unsetenv("SUDO_UID")
	defer os.Unsetenv("SUDO_GID")
	var gotName string
	var gotArgs []string
	hostreqkit.CombinedOutputFn = func(name string, args ...string) ([]byte, error) {
		gotName, gotArgs = name, args
		return []byte("enabled\n"), nil
	}
	if !timerEnabled() {
		t.Fatal("timerEnabled() = false, want enabled")
	}
	joined := gotName + " " + strings.Join(gotArgs, " ")
	if !strings.Contains(joined, "-u alice") || !strings.Contains(joined, "DBUS_SESSION_BUS_ADDRESS=unix:path=/run/user/1000/bus") {
		t.Fatalf("timer probe command = %q, want invoking user's bus", joined)
	}
}

func TestSettingsRejectNonsense(t *testing.T) {
	got := resolveSettings(map[string]any{
		"disk_floor_mb":          float64(0),
		"unit_threshold_seconds": float64(-5),
	})
	if got != defaults() {
		t.Fatalf("nonsensical config should fall back to defaults, got %#v", got)
	}
}

func TestServiceUnitPointsAtTheInstalledScript(t *testing.T) {
	p := resolvePaths("/home/u")
	unit, timer, err := renderedSystemdUnits(p)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(unit, "ExecStart=\""+p.Binary+"\" --report-only --request-pressure") {
		t.Errorf("service must run the portable watchdog binary with the shared argv, got:\n%s", unit)
	}
	if !strings.Contains(unit, "Documentation=https://github.com/Vrooli/Vrooli/blob/master/internal/safeguards/emergency-watchdog/handler.go") {
		t.Errorf("service must carry the URL Documentation convention, got:\n%s", unit)
	}
	if strings.Contains(unit, "After=default.target") || strings.Contains(unit, "network-online") {
		t.Errorf("a user oneshot must not order after targets the user manager never reaches, got:\n%s", unit)
	}
	if !strings.Contains(unit, "/usr/local/go/bin") {
		t.Errorf("unit PATH must include the Go toolchain, got:\n%s", unit)
	}
	if !strings.Contains(timer, "OnUnitActiveSec=300s") {
		t.Errorf("timer should keep the five-minute cadence, got:\n%s", timer)
	}
	if !strings.Contains(timer, "Persistent=true") {
		t.Error("timer should be persistent so a missed window still fires")
	}
}

// [REQ:BOOT-RECOVERY-001] Every platform runs the watchdog with the same argv.
func TestEveryPlatformRendersTheSameWatchdogArgv(t *testing.T) {
	p := resolvePaths("/home/u")
	for _, target := range []string{"linux", "darwin", "windows"} {
		d, err := definition(target, p)
		if err != nil {
			t.Fatalf("%s: %v", target, err)
		}
		if got := strings.Join(d.Args, " "); got != "--report-only --request-pressure" {
			t.Errorf("%s argv = %q, want the shared argv", target, got)
		}
	}
}

func TestFailedBuildPreservesPreviousWatchdog(t *testing.T) {
	origRoot := resolveWatchdogRootFn
	origBuild := buildWatchdogFn
	t.Cleanup(func() {
		resolveWatchdogRootFn = origRoot
		buildWatchdogFn = origBuild
	})
	root := t.TempDir()
	resolveWatchdogRootFn = func() (string, error) { return root, nil }
	buildWatchdogFn = func(string, string) error { return fmt.Errorf("compiler unavailable") }
	p := resolvePaths(root)
	if err := os.MkdirAll(filepath.Dir(p.Binary), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p.Binary, []byte("known-good"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := buildAndInstallWatchdog(p); err == nil {
		t.Fatal("expected build failure")
	}
	data, err := os.ReadFile(p.Binary)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "known-good" {
		t.Fatalf("previous watchdog was not preserved: %q", data)
	}
}

// An installed watchdog whose stamp differs from the checkout's source
// fingerprint is pending as stale, so setup rebuilds it instead of reading
// "present" from a binary that predates the findings it must report.
func TestPendingStateReportsStaleBinary(t *testing.T) {
	root, err := repocontract.ResolveRepoRoot()
	if err != nil {
		t.Skip("no repository root")
	}
	previousRoot, previousVersion := resolveWatchdogRootFn, installedWatchdogVersion
	defer func() { resolveWatchdogRootFn, installedWatchdogVersion = previousRoot, previousVersion }()
	resolveWatchdogRootFn = func() (string, error) { return root, nil }
	dir := t.TempDir()
	binary := filepath.Join(dir, "vrooli-watchdog")
	if err := os.WriteFile(binary, []byte("managed:old\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	p := paths{Binary: binary, ServiceUnit: filepath.Join(dir, "svc"), TimerUnit: filepath.Join(dir, "timer"), Home: dir}
	installedWatchdogVersion = func(string) string { return "managed:old" }
	var stale bool
	for _, item := range pendingState(p, settings{}) {
		if strings.Contains(item, "stale (built \"managed:old\"") {
			stale = true
		}
	}
	if !stale {
		t.Fatalf("stale binary not reported: %v", pendingState(p, settings{}))
	}
	want, err := expectedWatchdogVersion(root)
	if err != nil || !strings.HasPrefix(want, watchdogVersionPrefix) {
		t.Fatalf("expected version = %q, %v", want, err)
	}
	installedWatchdogVersion = func(string) string { return want }
	for _, item := range pendingState(p, settings{}) {
		if strings.HasPrefix(item, binary+" stale") {
			t.Fatalf("fresh binary reported stale: %v", item)
		}
	}
}
