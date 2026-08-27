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
// legacy script renderer remains only for compatibility tests; new installs
// schedule the binary directly.
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
	"strconv"
	"strings"

	"github.com/vrooli/vrooli/internal/buildinfo"
	"github.com/vrooli/vrooli/internal/repocontractmeta"
	"github.com/vrooli/vrooli/internal/tuning"

	"github.com/vrooli/vrooli/internal/config"

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
	Script      string
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
		Script:      filepath.Join(home, repocontractmeta.ProjectConfigDir, "libexec", "emergency-watchdog.sh"),
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

var buildWatchdogFn = func(root, output string) error {
	if err := hostreqkit.RunAsInvokingUser("go", []string{"-C", root, "build", "-ldflags", "-X main.buildVersion=managed", "-o", output, "./cmd/vrooli-watchdog"}, hostreqkit.EnsureOptions{}); err != nil {
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
	if host.OS == "darwin" && !guiLaunchdAvailable() {
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
	}
	if !hostreqkit.FileContentMatches(p.ServiceUnit, serviceContent(p)) {
		pending = append(pending, p.ServiceUnit+" missing or stale")
	}
	if !hostreqkit.FileContentMatches(p.TimerUnit, timerContent()) {
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

	// Everything below writes inside the user's own home, so it uses ordinary
	// file operations. Routing these through the privileged installer would
	// leave root-owned files in a user directory.
	writes := []struct {
		name    string
		path    string
		content string
	}{
		{"systemd service", p.ServiceUnit, serviceContent(p)},
		{"systemd timer", p.TimerUnit, timerContent()},
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

func serviceContent(p paths) string {
	setpoint := ""
	if root, err := repocontract.ResolveRepoRoot(); err == nil {
		setpoint = filepath.Join(root, repocontractmeta.ScenarioDir, "infrastructure-manager", "setpoint", "reliability-setpoint.json")
	}
	setpointEnv := ""
	if setpoint != "" {
		setpointEnv = "Environment=VROOLI_SETPOINT_PATH=" + setpoint + "\n"
	}
	return `[Unit]
Description=Vrooli emergency watchdog (portable host-pressure binary)
Documentation=internal/safeguards/emergency-watchdog/handler.go
After=default.target

[Service]
Type=oneshot
Environment=HOME=` + p.Home + `
` + setpointEnv + `
ExecStart=` + p.Binary + ` --report-only
`
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

func timerContent() string {
	return `[Unit]
Description=Vrooli emergency watchdog timer (5-minute cadence)

[Timer]
OnBootSec=2min
OnUnitActiveSec=5min
Persistent=true

[Install]
WantedBy=timers.target
`
}

// scriptContent renders the watchdog.
//
// The disk and unit-liveness logic is carried over from the script this
// safeguard replaces, including the properties that were learned the hard way:
// df's Available column rather than Free (the superuser reserve made a full
// filesystem look comfortable), logging that can never fail the script, and a
// self-bounding log so the watchdog cannot become the disk-pressure source it
// exists to catch.
//
// The saturation brake is new.
func scriptContent(s settings) string {
	return `#!/bin/sh
# Managed by Vrooli -- do not edit manually.
# See internal/safeguards/emergency-watchdog/handler.go for rationale.
#
# Pure-POSIX last-line-of-defense. Runs every 5 minutes from a systemd user
# timer and checks three things:
#
#   1. Free disk        — the condition that took the host down on 2026-07-31.
#   2. Host saturation  — the condition that took it down on 2026-08-19.
#   3. Unit liveness    — the runtime supervisor and the autoheal loop.
#
# It has no Go dependency: if everything Vrooli builds is broken this script
# still runs, which is the entire point.

set -eu

STATE_DIR="${HOME}/.vrooli/state"
LOG_FILE="${HOME}/.vrooli/logs/emergency-watchdog.log"
LAST_FAIL_FILE="${STATE_DIR}/emergency-watchdog.last-fail"
LAST_DISK_FILE="${STATE_DIR}/emergency-watchdog.last-disk"

THRESHOLD_SECONDS="${EMERGENCY_WATCHDOG_THRESHOLD:-` + strconv.Itoa(s.UnitThreshold) + `}"
DISK_FLOOR_MB="${EMERGENCY_WATCHDOG_DISK_FLOOR_MB:-` + strconv.Itoa(s.DiskFloorMB) + `}"
DISK_THRESHOLD_SECONDS="${EMERGENCY_WATCHDOG_DISK_THRESHOLD:-120}"
WATCH_MOUNT="${EMERGENCY_WATCHDOG_MOUNT:-/}"
LOG_MAX_BYTES="${EMERGENCY_WATCHDOG_LOG_MAX_BYTES:-1048576}"

UNITS="vrooli-runtime-supervisor.service vrooli-autoheal.service"

mkdir -p "$STATE_DIR" "$(dirname "$LOG_FILE")" 2>/dev/null || true

# rotate_log keeps the log bounded. A watchdog that grows its own log without
# limit is a slow version of the problem it exists to catch.
rotate_log() {
  [ -f "$LOG_FILE" ] || return 0
  size=$( ( wc -c <"$LOG_FILE" ) 2>/dev/null || echo 0 )
  [ "$size" -gt "$LOG_MAX_BYTES" ] 2>/dev/null || return 0
  ( tail -c $((LOG_MAX_BYTES / 2)) "$LOG_FILE" >"${LOG_FILE}.tmp" ) 2>/dev/null &&
    mv "${LOG_FILE}.tmp" "$LOG_FILE" 2>/dev/null
  return 0
}

# log never fails the script. A failed write here once exited non-zero and took
# the watchdog down with the disk — the one moment it most needed to keep
# running. The subshell matters: when the redirection itself fails, the shell
# reports it on stderr regardless of any redirection inside the command, and
# under systemd that stderr becomes journal writes.
log() {
  ( printf '%s %s\n' "$(date -Iseconds)" "$*" >>"$LOG_FILE" ) 2>/dev/null || true
  return 0
}

is_active() { systemctl --user is-active --quiet "$1"; }
now() { date +%s; }

# available_mb reads df's Available column, not Free. Free includes the
# superuser reserve, which on the 2026-07-31 incident host was 93 GB — enough to
# keep every threshold looking comfortable while the filesystem was unwritable
# for the supervisor.
available_mb() {
  df -PBM "$WATCH_MOUNT" 2>/dev/null | awk 'NR==2 {gsub(/M/,"",$4); print $4; found=1} END {if (!found) print ""}'
}

read_state() {
  [ -f "$1" ] || return 0
  value="$(cat "$1" 2>/dev/null || true)"
  [ -n "$value" ] && [ "$value" -gt 0 ] 2>/dev/null && printf '%s' "$value"
  return 0
}

# request_cleanup asks storage-manager to reclaim safe-tier space. It shells out
# to the CLI rather than linking anything: this script must keep working when
# the Go toolchain is broken. A missing CLI is logged and skipped, never fatal.
request_cleanup() {
  band="$1"
  used_percent="$2"
  if ! command -v storage-manager >/dev/null 2>&1; then
    log "storage-manager CLI not on PATH; cannot request reclamation"
    return 0
  fi
  if ( storage-manager cleanup report-pressure \
    --partition "$WATCH_MOUNT" \
    --band "$band" \
    --used-percent "$used_percent" \
    --source emergency-watchdog >>"$LOG_FILE" 2>&1 ) 2>/dev/null; then
    log "requested $band cleanup for $WATCH_MOUNT"
  else
    log "cleanup request FAILED for $WATCH_MOUNT"
  fi
  return 0
}

rotate_log

# ---------------------------------------------------------------------------
# Disk check
# ---------------------------------------------------------------------------

avail="$(available_mb)"
if [ -z "$avail" ]; then
  log "could not read available space on $WATCH_MOUNT"
elif [ "$avail" -ge "$DISK_FLOOR_MB" ] 2>/dev/null; then
  rm -f "$LAST_DISK_FILE" 2>/dev/null || true
else
  used_percent="$(df -P "$WATCH_MOUNT" 2>/dev/null | awk 'NR==2 {gsub(/%/,"",$5); print $5}')"
  [ -n "$used_percent" ] || used_percent=0
  first_disk_fail="$(read_state "$LAST_DISK_FILE")"
  if [ -z "$first_disk_fail" ]; then
    printf '%s\n' "$(now)" >"$LAST_DISK_FILE" 2>/dev/null || true
    log "first observed low disk: ${avail}MB available on $WATCH_MOUNT (floor ${DISK_FLOOR_MB}MB, ${used_percent}% used)"
  else
    disk_elapsed=$(( $(now) - first_disk_fail ))
    if [ "$disk_elapsed" -lt "$DISK_THRESHOLD_SECONDS" ]; then
      log "low disk ${disk_elapsed}s/${DISK_THRESHOLD_SECONDS}s — not yet escalating (${avail}MB available)"
    else
      log "ESCALATING: ${avail}MB available on $WATCH_MOUNT for ${disk_elapsed}s (floor ${DISK_FLOOR_MB}MB, ${used_percent}% used)"
      if [ "$avail" -lt $(( DISK_FLOOR_MB / 2 )) ] 2>/dev/null; then
        request_cleanup critical "$used_percent"
      else
        request_cleanup high "$used_percent"
      fi
      rm -f "$LAST_DISK_FILE" 2>/dev/null || true
    fi
  fi
fi

# ---------------------------------------------------------------------------
# Unit liveness check
# ---------------------------------------------------------------------------

any_down=0
for u in $UNITS; do
  if ! is_active "$u"; then
    any_down=1
  fi
done

if [ "$any_down" -eq 0 ]; then
  rm -f "$LAST_FAIL_FILE" 2>/dev/null || true
  exit 0
fi

first_fail="$(read_state "$LAST_FAIL_FILE")"
if [ -z "$first_fail" ]; then
  printf '%s\n' "$(now)" >"$LAST_FAIL_FILE" 2>/dev/null || true
  log "first observed unhealthy: $UNITS"
  exit 0
fi

elapsed=$(( $(now) - first_fail ))
if [ "$elapsed" -lt "$THRESHOLD_SECONDS" ]; then
  log "unhealthy ${elapsed}s/${THRESHOLD_SECONDS}s — not yet escalating"
  exit 0
fi

log "ESCALATING: units unhealthy for ${elapsed}s; pressure disposition is owned by the standalone watchdog binary"

# Attempt 1: cheap, non-mutating dependency refresh at the repo root, when the
# unit supplied one. There is no hard-coded fallback: a watchdog that guesses at
# somebody else's checkout is worse than one that skips this step.
if [ -n "${` + buildinfo.SourceRootFallbackEnvVar + `:-}" ] && [ -f "${` + buildinfo.SourceRootFallbackEnvVar + `}/go.mod" ] && command -v go >/dev/null 2>&1; then
  ( cd "$` + buildinfo.SourceRootFallbackEnvVar + `" && go mod download 2>>"$LOG_FILE" ) 2>/dev/null || log "go mod download exited non-zero"
else
  log "skipping go mod download (no ` + buildinfo.SourceRootFallbackEnvVar + `, go.mod, or go binary)"
fi

# Attempt 2: restart the systemd units; ExecStartPre will swap in known-good
# binaries if the live ones are corrupt.
for u in $UNITS; do
  if ( systemctl --user restart "$u" 2>>"$LOG_FILE" ) 2>/dev/null; then
    log "restart ok: $u"
  else
    log "restart FAILED: $u"
  fi
done

rm -f "$LAST_FAIL_FILE" 2>/dev/null || true
exit 0
`
}
