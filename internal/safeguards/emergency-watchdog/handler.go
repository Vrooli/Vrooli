// Package emergencywatchdog installs the last-line-of-defense watchdog: the

// portable vrooli-watchdog binary plus the native scheduler that runs it every
// five minutes (systemd on Linux, launchd on macOS, and Task Scheduler on
// Windows).
//
// The watchdog itself is not new. It existed as scripts/emergency-watchdog.sh,
// was referenced by three runbooks, and was installed by nothing — its unit had
// been hand-created and hard-coded one operator's repository path, so no other
// host had it and `vrooli setup` could not reason about it. This safeguard
// makes setup the owner, which is the whole point of the setup contract. The
// shell script is gone (2026-09-02): the Go binary in cmd/vrooli-watchdog
// senses, writes ~/.vrooli/state/emergency-watchdog/last-report.json, and
// owns the unit-restart escalation the script used to carry.
//
// It runs as the invoking user, not root. The units it watches are systemd
// *user* units, and everything it writes lives under the user's own home, so
// requesting privilege here would be privilege for its own sake.
//
// Escalation is gated on host saturation. On 2026-08-19 this host reached a
// load average of 110 on 32 CPUs and autoheal restarted itself into it; the
// restart could not schedule and the machine got no better. A watchdog that
// restarts a saturated host is adding load to a load problem, so under
// sustained CPU pressure this one reports and waits.
package emergencywatchdog

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/vrooli/vrooli/internal/buildinfo"
	"github.com/vrooli/vrooli/internal/config"
	"github.com/vrooli/vrooli/internal/repocontractmeta"
	setpointpkg "github.com/vrooli/vrooli/internal/setpoint"
	"github.com/vrooli/vrooli/internal/tuning"

	platformgo "github.com/vrooli/platform-go"
	repocontract "github.com/vrooli/repo-contract-go"
	"github.com/vrooli/vrooli/internal/hostreqkit"
	"github.com/vrooli/vrooli/internal/hostreqspec"
	"github.com/vrooli/vrooli/internal/watchdoginstall"
)

const (
	serviceName = "vrooli-emergency-watchdog.service"
	timerName   = "vrooli-emergency-watchdog.timer"

	defaultDiskFloorMB   = 10240
	defaultUnitThreshold = 600
)

// settings is the resolved operator configuration.
type settings struct {
	DiskFloorMB   int
	UnitThreshold int
}

func resolveSettings(config map[string]any) settings {
	s := settings{
		DiskFloorMB:   defaultDiskFloorMB,
		UnitThreshold: defaultUnitThreshold,
	}
	if config == nil {
		return s
	}
	if v, ok := intFromConfig(config["disk_floor_mb"]); ok && v > 0 {
		s.DiskFloorMB = v
	}
	if v, ok := intFromConfig(config["unit_threshold_seconds"]); ok && v > 0 {
		s.UnitThreshold = v
	}
	return s
}

func intFromConfig(value any) (int, bool) {
	switch v := value.(type) {
	case float64:
		return int(v), true
	case int:
		return v, true
	default:
		return 0, false
	}
}

// paths groups the user-scoped install locations.
type paths struct {
	Home        string
	Binary      string
	ServiceUnit string
	TimerUnit   string
	LaunchAgent string
}

func resolvePaths(home string) paths {
	binaryName := "vrooli-watchdog"
	if os.PathSeparator == '\\' {
		binaryName += ".exe"
	}
	return paths{
		Home:        home,
		Binary:      filepath.Join(home, repocontractmeta.ProjectConfigDir, "libexec", binaryName),
		ServiceUnit: filepath.Join(home, ".config", "systemd", "user", serviceName),
		TimerUnit:   filepath.Join(home, ".config", "systemd", "user", timerName),
		LaunchAgent: filepath.Join(home, "Library", "LaunchAgents", "com.vrooli.emergency-watchdog.plist"),
	}
}

type handler struct {
	manifest hostreqkit.SafeguardManifest
}

func NewHandler(manifest hostreqkit.SafeguardManifest) hostreqkit.Handler {
	return handler{manifest: manifest}
}

func (h handler) Name() string           { return h.manifest.Name }
func (h handler) Kind() hostreqspec.Kind { return hostreqspec.KindSafeguard }

// homeDir is stubbed in tests.
var homeDir = func() (string, error) {
	if hostreqkit.RunningAsRootFn() {
		return hostreqkit.InvokingUserHomeDir()
	}
	return os.UserHomeDir()
}

var resolveWatchdogRootFn = repocontract.ResolveRepoRoot

// watchdogSourceInputs are the source trees whose content decides what the
// watchdog reports. Their fingerprint is stamped into the binary at build
// time and compared on every inspection, so a watchdog built before an
// attribution or setpoint change is "stale", not "present". Extend the list
// when the watchdog gains a dependency whose behavior it surfaces.
var watchdogSourceInputs = []string{
	"cmd/vrooli-watchdog",
	"internal/hostpressure",
	"internal/setpoint",
	"internal/workloadowner",
	"internal/hostinventory",
	"packages/platform-go",
}

const watchdogVersionPrefix = "managed:"

// expectedWatchdogVersion is the version string a binary built from this
// checkout carries.
func expectedWatchdogVersion(root string) (string, error) {
	fingerprint, err := buildinfo.ComputeSourceFingerprintForPaths(root, watchdogSourceInputs...)
	if err != nil {
		return "", fmt.Errorf("fingerprint watchdog sources: %w", err)
	}
	return watchdogVersionPrefix + fingerprint, nil
}

// installedWatchdogVersion asks the installed binary for its stamp; "" when
// it cannot answer, which pendingState treats as stale.
var installedWatchdogVersion = func(binary string) string {
	out, err := hostreqkit.CombinedOutputFn(binary, "--version")
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

var buildWatchdogFn = func(root, output string) error {
	version, err := expectedWatchdogVersion(root)
	if err != nil {
		return err
	}
	if err := hostreqkit.RunAsInvokingUser("go", []string{"-C", root, "build", "-ldflags", "-X main.buildVersion=" + version, "-o", output, "./cmd/vrooli-watchdog"}, hostreqkit.EnsureOptions{}); err != nil {
		return fmt.Errorf("build watchdog: %w", err)
	}
	return nil
}

func (h handler) Inspect(host hostreqkit.Host, requirement hostreqspec.ResolvedRequirement) hostreqkit.ItemStatus {
	status := hostreqkit.BaseStatus(requirement)
	status.SupportClass = hostreqkit.SupportSupported

	if requirement.Manual {
		status.SupportClass = hostreqkit.SupportManualOnly
		status.ExecutionState = hostreqkit.ExecutionManualActionRequired
		return status
	}
	if hostreqspec.PlatformFromGOOS(host.OS) != hostreqspec.PlatformLinux && !nativeSchedulerAvailable(host.OS) {
		schedule := watchdoginstall.For(host.OS, tuning.EmergencyWatchdogInterval())
		status.SupportClass = hostreqkit.SupportUnsupported
		status.ExecutionState = hostreqkit.ExecutionUnsupported
		status.Notes = append(status.Notes,
			"the native scheduler for "+host.OS+" is unavailable: "+schedule.Remediation)
		return status
	}
	if hostreqspec.PlatformFromGOOS(host.OS) == hostreqspec.PlatformLinux && !host.SupportsSystemd {
		status.SupportClass = hostreqkit.SupportUnsupported
		status.ExecutionState = hostreqkit.ExecutionUnsupported
		status.Notes = append(status.Notes, "the emergency watchdog requires systemd user service/timer support")
		return status
	}
	if host.OS == string(hostreqspec.PlatformDarwin) && !guiLaunchdAvailable() {
		status.SupportClass = hostreqkit.SupportNotApplicable
		status.ExecutionState = hostreqkit.ExecutionNotApplicable
		status.Notes = append(status.Notes, "the invoking user's GUI launchd domain is unavailable; this user LaunchAgent is not applicable to the current SSH/headless session")
		return status
	}

	home, err := homeDir()
	if err != nil || strings.TrimSpace(home) == "" {
		status.ExecutionState = hostreqkit.ExecutionFailed
		status.Notes = append(status.Notes, "could not determine the invoking user's home directory")
		return status
	}

	p := resolvePaths(home)
	resolved := resolveSettings(requirement.Config)
	if hostreqspec.PlatformFromGOOS(host.OS) != hostreqspec.PlatformLinux {
		pending := nativePending(host.OS, p)
		if len(pending) == 0 {
			status.Applied = true
			status.ExecutionState = hostreqkit.ExecutionAlreadyPresent
			status.Notes = append(status.Notes, "emergency watchdog is installed in the native "+host.OS+" scheduler")
			return status
		}
		status.Notes = append(status.Notes, "emergency watchdog pending: "+strings.Join(pending, ", "))
		return status
	}
	pending := pendingState(p, resolved)
	if artifact, err := renderedArtifact("linux", p); err == nil {
		recordVerdict(&status, platformgo.ValidateArtifact(artifact, platformgo.ScopeUser))
	}
	if len(pending) == 0 {
		status.Applied = true
		status.ExecutionState = hostreqkit.ExecutionAlreadyPresent
		status.Notes = append(status.Notes, "emergency watchdog binary and timer are installed")
		return status
	}

	status.Notes = append(status.Notes, "emergency watchdog pending: "+strings.Join(pending, ", "))
	return status
}

func pendingState(p paths, s settings) []string {
	var pending []string
	_ = s // thresholds are enforced by the standalone watchdog/setpoint.
	if _, err := os.Stat(p.Binary); err != nil {
		pending = append(pending, p.Binary+" missing")
	} else if root, err := resolveWatchdogRootFn(); err == nil && strings.TrimSpace(root) != "" {
		if want, err := expectedWatchdogVersion(root); err == nil {
			if got := installedWatchdogVersion(p.Binary); got != want {
				pending = append(pending, fmt.Sprintf("%s stale (built %q, source %q)", p.Binary, got, want))
			}
		}
	}
	service, timer, err := renderedSystemdUnits(p)
	if err != nil {
		return append(pending, "render systemd units: "+err.Error())
	}
	if !hostreqkit.FileContentMatches(p.ServiceUnit, service) {
		pending = append(pending, p.ServiceUnit+" missing or stale")
	}
	if !hostreqkit.FileContentMatches(p.TimerUnit, timer) {
		pending = append(pending, p.TimerUnit+" missing or stale")
	}
	if !timerEnabled() {
		pending = append(pending, timerName+" not enabled")
	}
	return pending
}

func timerEnabled() bool {
	name, args := hostreqkit.InvokingUserCommand("systemctl", "--user", "is-enabled", timerName)
	out, err := hostreqkit.CombinedOutputFn(name, args...)
	return err == nil && strings.TrimSpace(string(out)) == "enabled"
}

func (h handler) Apply(host hostreqkit.Host, status hostreqkit.ItemStatus, opts hostreqkit.EnsureOptions) (hostreqkit.ItemStatus, error) {
	switch status.SupportClass {
	case hostreqkit.SupportUnsupported:
		status.ExecutionState = hostreqkit.ExecutionUnsupported
		return status, nil
	case hostreqkit.SupportNotApplicable:
		status.ExecutionState = hostreqkit.ExecutionNotApplicable
		return status, nil
	case hostreqkit.SupportManualOnly:
		status.ExecutionState = hostreqkit.ExecutionManualActionRequired
		return status, nil
	}
	if status.Applied {
		status.ExecutionState = hostreqkit.ExecutionAlreadyPresent
		return status, nil
	}

	home, err := homeDir()
	if err != nil || strings.TrimSpace(home) == "" {
		status.ExecutionState = hostreqkit.ExecutionFailed
		status.Notes = append(status.Notes, "could not determine the invoking user's home directory")
		return status, nil
	}
	p := resolvePaths(home)
	if opts.DryRun {
		status.ExecutionState = hostreqkit.ExecutionWouldApply
		status.Notes = append(status.Notes,
			fmt.Sprintf("dry-run: would build %s and install the native watchdog scheduler for %s", p.Binary, host.OS))
		return status, nil
	}
	if err := buildAndInstallWatchdog(p); err != nil {
		status.ExecutionState = hostreqkit.ExecutionFailed
		status.Notes = append(status.Notes, "degraded: previous watchdog binary was preserved: "+err.Error())
		return status, nil
	}
	if hostreqspec.PlatformFromGOOS(host.OS) != hostreqspec.PlatformLinux {
		return applyNative(host.OS, p, status, opts)
	}

	// Render once, ask systemd whether it would load the result, and only
	// then touch the unit directory: a rejected render must never replace a
	// working unit.
	artifact, err := renderedArtifact("linux", p)
	if err != nil {
		status.ExecutionState = hostreqkit.ExecutionFailed
		status.Notes = append(status.Notes, "render systemd units: "+err.Error())
		return status, nil
	}
	verdict := platformgo.ValidateArtifact(artifact, platformgo.ScopeUser)
	recordVerdict(&status, verdict)
	if verdict.Rejected() {
		status.ExecutionState = hostreqkit.ExecutionFailed
		status.Notes = append(status.Notes, "systemd rejected the rendered emergency watchdog units; nothing was installed: "+verdict.Output)
		return status, nil
	}
	service, _ := artifact.File(serviceName)
	timer, _ := artifact.File(timerName)
	// Everything below writes inside the user's own home, so it uses ordinary
	// file operations. Routing these through the privileged installer would
	// leave root-owned files in a user directory.
	writes := []struct {
		name    string
		path    string
		content string
	}{
		{"systemd service", p.ServiceUnit, service.Content},
		{"systemd timer", p.TimerUnit, timer.Content},
	}
	for _, w := range writes {
		if err := installUserFile(w.path, w.content, opts); err != nil {
			status.ExecutionState = hostreqkit.ExecutionFailed
			status.Notes = append(status.Notes, "install "+w.name+" failed: "+err.Error())
			return status, nil
		}
	}

	for _, args := range [][]string{
		{"--user", "daemon-reload"},
		{"--user", "enable", "--now", timerName},
	} {
		name, commandArgs := hostreqkit.InvokingUserCommand("systemctl", args...)
		if _, err := hostreqkit.CombinedOutputFn(name, commandArgs...); err != nil {
			status.ExecutionState = hostreqkit.ExecutionFailed
			status.Notes = append(status.Notes, "systemctl "+strings.Join(args, " ")+" failed: "+err.Error())
			return status, nil
		}
	}

	status.Applied = true
	status.ExecutionState = hostreqkit.ExecutionApplied
	status.Notes = append(status.Notes, "emergency watchdog installed and timer enabled")
	return status, nil
}

// definition builds the watchdog's one ServiceDefinition for a target. The
// same argv renders on every platform; the interval comes from tuning so an
// operator override reaches the native scheduler too.
func definition(target string, p paths) (platformgo.ServiceDefinition, error) {
	setpoint := ""
	if root, err := resolveWatchdogRootFn(); err == nil && strings.TrimSpace(root) != "" {
		setpoint = filepath.Join(root, filepath.FromSlash(setpointpkg.RelativePath))
	}
	return platformgo.EmergencyWatchdogDefinition(target, platformgo.EmergencyWatchdogOptions{
		Home:         p.Home,
		Binary:       p.Binary,
		SetpointPath: setpoint,
		Interval:     tuning.EmergencyWatchdogInterval(),
		Username:     hostreqkit.InvokingUser(),
	})
}

// renderedArtifact renders the watchdog units for a target.
func renderedArtifact(target string, p paths) (platformgo.RenderedArtifact, error) {
	d, err := definition(target, p)
	if err != nil {
		return platformgo.RenderedArtifact{}, err
	}
	return platformgo.RenderDefinition(d, target)
}

// renderedSystemdUnits returns the service and timer bodies the Linux
// install writes, so Inspect compares against exactly what Apply installs.
func renderedSystemdUnits(p paths) (string, string, error) {
	artifact, err := renderedArtifact("linux", p)
	if err != nil {
		return "", "", err
	}
	service, ok := artifact.File(serviceName)
	if !ok {
		return "", "", fmt.Errorf("rendered artifact lacks %s", serviceName)
	}
	timer, ok := artifact.File(timerName)
	if !ok {
		return "", "", fmt.Errorf("rendered artifact lacks %s", timerName)
	}
	return service.Content, timer.Content, nil
}

// recordVerdict stores the native validator's answer as evidence so the
// readiness inspection can read it back, and notes an unavailable
// validator: unproven is not accepted.
func recordVerdict(status *hostreqkit.ItemStatus, verdict platformgo.Verdict) {
	if status.Evidence == nil {
		status.Evidence = map[string]any{}
	}
	status.Evidence["validator_verdict"] = verdict
	if verdict.State == platformgo.VerdictUnavailable {
		status.Notes = append(status.Notes, "native validator unavailable; rendered units are unproven: "+verdict.Output)
	}
}

func buildAndInstallWatchdog(p paths) error {
	root, err := resolveWatchdogRootFn()
	if err != nil {
		return fmt.Errorf("resolve Vrooli source root: %w", err)
	}
	if strings.TrimSpace(root) == "" {
		return fmt.Errorf("Vrooli source root is empty")
	}
	if err := hostreqkit.RunAsInvokingUser("mkdir", []string{"-p", filepath.Dir(p.Binary)}, hostreqkit.EnsureOptions{}); err != nil {
		return fmt.Errorf("create watchdog install directory: %w", err)
	}
	tmp := p.Binary + fmt.Sprintf(".%d.tmp", os.Getpid())
	defer os.Remove(tmp)
	if err := buildWatchdogFn(root, tmp); err != nil {
		return err
	}
	data, err := os.ReadFile(tmp)
	if err != nil {
		return fmt.Errorf("read built watchdog: %w", err)
	}
	if err := config.WriteOwnedFileAtomic(p.Binary, data, tuning.PermDir); err != nil {
		return fmt.Errorf("install watchdog atomically: %w", err)
	}
	return nil
}

func installUserFile(path, content string, opts hostreqkit.EnsureOptions) error {
	return hostreqkit.InstallUserFile(path, content, opts)
}
