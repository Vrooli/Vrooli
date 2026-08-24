package emergencywatchdog

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/vrooli/vrooli/internal/hostreqkit"
	"github.com/vrooli/vrooli/internal/hostreqspec"
)

func testManifest() hostreqkit.SafeguardManifest {
	return hostreqkit.SafeguardManifest{Name: "emergency_watchdog"}
}

func linuxHost() hostreqkit.Host {
	return hostreqkit.Host{OS: "linux", SupportsSystemd: true}
}

func req() hostreqspec.ResolvedRequirement {
	return hostreqspec.ResolvedRequirement{Name: "emergency_watchdog", Kind: hostreqspec.KindSafeguard, Required: true}
}

func defaults() settings { return resolveSettings(nil) }

// Handler fallbacks and manifest defaults must agree, or setup and the handler
// disagree about what "unconfigured" means.
func TestDefaultsMatchManifest(t *testing.T) {
	raw, err := os.ReadFile("safeguard.json")
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}
	var manifest struct {
		Config struct {
			Properties map[string]json.RawMessage `json:"properties"`
		} `json:"config"`
	}
	if err := json.Unmarshal(raw, &manifest); err != nil {
		t.Fatalf("parse manifest: %v", err)
	}
	if _, ok := manifest.Config.Properties["setpoint_path"]; !ok {
		t.Fatal("manifest must expose the shared setpoint path, not private watchdog thresholds")
	}
	for _, obsolete := range []string{"disk_floor_mb", "unit_threshold_seconds", "cpu_pressure_avg10"} {
		if _, ok := manifest.Config.Properties[obsolete]; ok {
			t.Errorf("obsolete private watchdog threshold remains in manifest: %s", obsolete)
		}
	}
}

func TestInspectNonLinuxUnsupported(t *testing.T) {
	status := NewHandler(testManifest()).Inspect(hostreqkit.Host{OS: "darwin"}, req())
	if status.SupportClass != hostreqkit.SupportUnsupported {
		t.Fatalf("SupportClass = %q, want unsupported", status.SupportClass)
	}
	if !strings.Contains(strings.Join(status.Notes, " "), "native scheduler") {
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

// The unit must not carry a hard-coded repository path — that is what made the
// script this safeguard replaces unusable on any other host.
func TestNoHardCodedOperatorPaths(t *testing.T) {
	script := scriptContent(defaults())
	for _, leak := range []string{"/home/matthalloran8", "matthalloran8"} {
		if strings.Contains(script, leak) {
			t.Errorf("script leaks an operator-specific path: %s", leak)
		}
	}
	if !strings.Contains(script, `[ -n "${VROOLI_ROOT:-}" ]`) {
		t.Error("script should skip the repo step when VROOLI_ROOT is unset rather than guessing")
	}
}

func TestSettingsFlowIntoTheScript(t *testing.T) {
	script := scriptContent(resolveSettings(map[string]any{
		"disk_floor_mb":          float64(2048),
		"unit_threshold_seconds": float64(90),
	}))
	for _, want := range []string{"2048", "90"} {
		if !strings.Contains(script, want) {
			t.Errorf("script should embed configured value %s", want)
		}
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
	unit := serviceContent(p)
	if !strings.Contains(unit, "ExecStart="+p.Binary+" --report-only") {
		t.Errorf("service must run the portable watchdog binary, got:\n%s", unit)
	}
	if !strings.Contains(timerContent(), "OnUnitActiveSec=5min") {
		t.Error("timer should keep the five-minute cadence")
	}
	if !strings.Contains(timerContent(), "Persistent=true") {
		t.Error("timer should be persistent so a missed window still fires")
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

// ---------------------------------------------------------------------------
// Executing the generated script
// ---------------------------------------------------------------------------

// sandbox renders the script with stubbed externals so it can run as a test.
// systemctl reports the watched units according to unitsActive, and
// /proc/pressure/cpu is replaced by a fixture the script reads through an
// overridden path.
func sandbox(t *testing.T, s settings, unitsActive bool, cpuAvg10 string) (script, home, logPath string) {
	t.Helper()
	if runtime.GOOS != "linux" {
		t.Skip("the watchdog is a Linux shell script")
	}
	root := t.TempDir()
	home = filepath.Join(root, "home")
	binDir := filepath.Join(root, "bin")
	for _, dir := range []string{home, binDir} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", dir, err)
		}
	}

	activeExit := "1"
	if unitsActive {
		activeExit = "0"
	}
	writeExec(t, filepath.Join(binDir, "systemctl"), "#!/bin/sh\n"+
		"for a in \"$@\"; do case \"$a\" in is-active) exit "+activeExit+";; restart) echo \"restart $*\" >> \""+root+"/restarts\"; exit 0;; esac; done\nexit 0\n")
	// storage-manager absent is a supported condition; keep it that way.

	body := scriptContent(s)
	// Point the CPU pressure probe at a fixture rather than the real host.
	pressurePath := filepath.Join(root, "cpu-pressure")
	if cpuAvg10 != "" {
		if err := os.WriteFile(pressurePath, []byte("some avg10="+cpuAvg10+" avg60=1.00 avg300=1.00 total=1\nfull avg10=0.00 avg60=0.00 avg300=0.00 total=0\n"), 0o644); err != nil {
			t.Fatalf("write pressure fixture: %v", err)
		}
	}
	body = strings.ReplaceAll(body, "/proc/pressure/cpu", pressurePath)
	body = strings.Replace(body, "#!/bin/sh\n", "#!/bin/sh\nPATH=\""+binDir+":$PATH\"\n", 1)

	script = filepath.Join(root, "watchdog.sh")
	writeExec(t, script, body)
	return script, home, filepath.Join(home, ".vrooli", "logs", "emergency-watchdog.log")
}

func writeExec(t *testing.T, path, body string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(body), 0o755); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func run(t *testing.T, script, home string) string {
	t.Helper()
	cmd := exec.Command("/bin/sh", script)
	cmd.Env = append(os.Environ(), "HOME="+home)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("watchdog exited non-zero: %v\n%s", err, out)
	}
	return string(out)
}

func readLog(t *testing.T, logPath string) string {
	t.Helper()
	raw, err := os.ReadFile(logPath)
	if err != nil {
		return ""
	}
	return string(raw)
}

// The brake must not become a permanent excuse: once pressure clears, the very
// next tick escalates.
func TestWatchdogEscalatesOncePressureClears(t *testing.T) {
	s := defaults()
	s.UnitThreshold = 0
	script, home, logPath := sandbox(t, s, false, "3.00")

	run(t, script, home)
	run(t, script, home)

	log := readLog(t, logPath)
	if !strings.Contains(log, "ESCALATING") {
		t.Fatalf("expected escalation on an unsaturated host; log:\n%s", log)
	}
	if strings.Contains(log, "HOLDING restart") {
		t.Errorf("must not hold when the host is idle; log:\n%s", log)
	}
}

// A host with no PSI support must still be able to recover.
func TestWatchdogEscalatesWhenPressureIsUnavailable(t *testing.T) {
	s := defaults()
	s.UnitThreshold = 0
	script, home, logPath := sandbox(t, s, false, "")

	run(t, script, home)
	run(t, script, home)

	if log := readLog(t, logPath); !strings.Contains(log, "ESCALATING") {
		t.Fatalf("missing PSI must not disable recovery; log:\n%s", log)
	}
}

// Healthy units clear hysteresis and produce no escalation.
func TestWatchdogQuietWhenHealthy(t *testing.T) {
	script, home, logPath := sandbox(t, defaults(), true, "3.00")

	run(t, script, home)
	run(t, script, home)

	if log := readLog(t, logPath); strings.Contains(log, "ESCALATING") || strings.Contains(log, "unhealthy") {
		t.Fatalf("a healthy host should stay quiet; log:\n%s", log)
	}
}

// Hysteresis: one unhealthy observation is not enough to act on.
func TestWatchdogWaitsOutTheThresholdBeforeActing(t *testing.T) {
	s := defaults()
	s.UnitThreshold = 3600
	script, home, logPath := sandbox(t, s, false, "3.00")

	run(t, script, home)
	run(t, script, home)

	log := readLog(t, logPath)
	if !strings.Contains(log, "not yet escalating") {
		t.Fatalf("expected hysteresis to hold; log:\n%s", log)
	}
	if strings.Contains(log, "ESCALATING") {
		t.Errorf("must not act before the threshold; log:\n%s", log)
	}
}
