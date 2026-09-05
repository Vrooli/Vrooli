package keyringdaemonlimits

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/vrooli/vrooli/internal/hostreqkit"
	"github.com/vrooli/vrooli/internal/hostreqspec"
)

// The safeguard keeps the invoking user's gnome-keyring-daemon from wedging on
// file-descriptor exhaustion. See doc.go for the incident that motivated it.
const (
	unitName    = "gnome-keyring-daemon.service"
	dropInDir   = ".config/systemd/user/gnome-keyring-daemon.service.d"
	dropInName  = "99-vrooli-limits.conf"
	procRoot    = "/proc"
	percent     = 100
	limitsField = "Max open files"

	defaultNoFileLimit       = 65536
	defaultRestartSaturation = 50
	minimumNoFileLimit       = 4096
	// unlimitedDescriptors stands in for a soft limit of "unlimited" so the
	// saturation arithmetic stays finite.
	unlimitedDescriptors = 1 << 30
)

type settings struct {
	NoFileLimit       int
	RestartSaturation int
}

func resolveSettings(config map[string]any) settings {
	s := settings{NoFileLimit: defaultNoFileLimit, RestartSaturation: defaultRestartSaturation}
	if v, ok := intFromConfig(config["nofile_limit"]); ok && v >= minimumNoFileLimit {
		s.NoFileLimit = v
	}
	if v, ok := intFromConfig(config["restart_saturation_percent"]); ok && v > 0 && v <= percent {
		s.RestartSaturation = v
	}
	return s
}

func intFromConfig(value any) (int, bool) {
	switch v := value.(type) {
	case int:
		return v, true
	case int64:
		return int(v), true
	case float64:
		return int(v), true
	case string:
		n, err := strconv.Atoi(strings.TrimSpace(v))
		return n, err == nil
	}
	return 0, false
}

func dropInContent(s settings) string {
	return strings.Join([]string{
		"# Managed by Vrooli (keyring_daemon_limits) -- do not edit manually",
		"# gnome-keyring-daemon leaks one eventfd per client connection under some",
		"# client patterns. At the default soft limit of 1024 descriptors a week of",
		"# leaks wedges it: accept() fails forever, every secret lookup hangs, console",
		"# login stalls in pam_gnome_keyring, and the failure line is logged tens of",
		"# thousands of times a second (320 GB of syslog on 2026-09-01). A high limit",
		"# turns a weekly wedge into a leak that takes over a year to matter, and",
		"# `vrooli setup` restarts the daemon when it passes the saturation threshold.",
		"[Service]",
		fmt.Sprintf("LimitNOFILE=%d", s.NoFileLimit),
		"",
	}, "\n")
}

// Seams. homeDir mirrors the emergency-watchdog safeguard: setup usually runs as
// root, and the unit belongs to the operator, not to root.
var (
	homeDir = func() (string, error) {
		if hostreqkit.RunningAsRootFn() {
			return hostreqkit.InvokingUserHomeDir()
		}
		return os.UserHomeDir()
	}
	readDirCountFn = func(path string) (int, error) {
		entries, err := os.ReadDir(path)
		if err != nil {
			return 0, err
		}
		return len(entries), nil
	}
)

// daemonState is what the running daemon looks like right now. Zero values mean
// "not running" or "not observable"; the caller decides what that implies.
type daemonState struct {
	PID       int
	OpenFDs   int
	SoftLimit int
}

func (d daemonState) saturationPercent() int {
	if d.SoftLimit <= 0 {
		return 0
	}
	return d.OpenFDs * percent / d.SoftLimit
}

func userSystemctl(args ...string) ([]byte, error) {
	name, commandArgs := hostreqkit.InvokingUserCommand("systemctl", append([]string{"--user"}, args...)...)
	return hostreqkit.CombinedOutputFn(name, commandArgs...)
}

func unitPresent() bool {
	out, err := userSystemctl("cat", unitName)
	return err == nil && strings.Contains(string(out), "ExecStart=")
}

func observeDaemon() daemonState {
	out, err := userSystemctl("show", "-p", "MainPID", "--value", unitName)
	if err != nil {
		return daemonState{}
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(out)))
	if err != nil || pid <= 0 {
		return daemonState{}
	}
	state := daemonState{PID: pid}
	if n, err := readDirCountFn(filepath.Join(procRoot, strconv.Itoa(pid), "fd")); err == nil {
		state.OpenFDs = n
	}
	if limits, err := hostreqkit.ReadFileFn(filepath.Join(procRoot, strconv.Itoa(pid), "limits")); err == nil {
		state.SoftLimit = parseSoftLimit(string(limits))
	}
	return state
}

// parseSoftLimit extracts the soft "Max open files" value from /proc/<pid>/limits.
func parseSoftLimit(limits string) int {
	for _, line := range strings.Split(limits, "\n") {
		if !strings.HasPrefix(line, limitsField) {
			continue
		}
		fields := strings.Fields(strings.TrimPrefix(line, limitsField))
		if len(fields) < 1 {
			return 0
		}
		if fields[0] == "unlimited" {
			return unlimitedDescriptors
		}
		n, err := strconv.Atoi(fields[0])
		if err != nil {
			return 0
		}
		return n
	}
	return 0
}

type handler struct {
	manifest hostreqkit.SafeguardManifest
}

func NewHandler(manifest hostreqkit.SafeguardManifest) hostreqkit.Handler {
	return handler{manifest: manifest}
}

func (h handler) Name() string           { return h.manifest.Name }
func (h handler) Kind() hostreqspec.Kind { return hostreqspec.KindSafeguard }

func (h handler) Inspect(host hostreqkit.Host, requirement hostreqspec.ResolvedRequirement) hostreqkit.ItemStatus {
	status := hostreqkit.BaseStatus(requirement)
	status.SupportClass = hostreqkit.SupportSupported

	if requirement.Manual {
		status.SupportClass = hostreqkit.SupportManualOnly
		status.ExecutionState = hostreqkit.ExecutionManualActionRequired
		return status
	}
	if host.OS != string(hostreqspec.PlatformLinux) {
		status.SupportClass = hostreqkit.SupportUnsupported
		status.ExecutionState = hostreqkit.ExecutionUnsupported
		status.Notes = append(status.Notes, "gnome-keyring-daemon limits apply to Linux desktops only")
		return status
	}
	if !host.SupportsSystemd {
		status.SupportClass = hostreqkit.SupportNotApplicable
		status.ExecutionState = hostreqkit.ExecutionNotApplicable
		status.Notes = append(status.Notes, "requires a systemd user manager to own the keyring unit")
		return status
	}
	if !unitPresent() {
		status.SupportClass = hostreqkit.SupportNotApplicable
		status.ExecutionState = hostreqkit.ExecutionNotApplicable
		status.Notes = append(status.Notes, "no "+unitName+" user unit; the keyring is not systemd-managed on this host")
		return status
	}
	home, err := homeDir()
	if err != nil || strings.TrimSpace(home) == "" {
		status.ExecutionState = hostreqkit.ExecutionFailed
		status.Notes = append(status.Notes, "could not determine the invoking user's home directory")
		return status
	}

	s := resolveSettings(requirement.Config)
	dropIn := filepath.Join(home, dropInDir, dropInName)
	state := observeDaemon()
	var pending []string
	if !hostreqkit.FileContentMatches(dropIn, dropInContent(s)) {
		pending = append(pending, fmt.Sprintf("%s missing or stale (wants LimitNOFILE=%d)", dropIn, s.NoFileLimit))
	}
	if state.PID > 0 && state.SoftLimit > 0 {
		sat := state.saturationPercent()
		status.Notes = append(status.Notes, fmt.Sprintf("daemon pid %d holds %d of %d descriptors (%d%%)", state.PID, state.OpenFDs, state.SoftLimit, sat))
		if sat >= s.RestartSaturation {
			pending = append(pending, fmt.Sprintf("daemon is at %d%% of its descriptor limit; a restart is due before it wedges", sat))
		}
	}

	if len(pending) == 0 {
		status.Applied = true
		status.ExecutionState = hostreqkit.ExecutionAlreadyPresent
		status.Notes = append(status.Notes, fmt.Sprintf("keyring daemon limit raised to %d descriptors", s.NoFileLimit))
		return status
	}
	status.Notes = append(status.Notes, pending...)
	return status
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
		status.Notes = append(status.Notes, "manual safeguard action required by manifest declaration")
		return status, nil
	}
	if status.Applied {
		status.ExecutionState = hostreqkit.ExecutionAlreadyPresent
		return status, nil
	}

	s := resolveSettings(status.Config)
	if opts.DryRun {
		status.ExecutionState = hostreqkit.ExecutionWouldApply
		status.Notes = append(status.Notes,
			fmt.Sprintf("dry-run: would set LimitNOFILE=%d on %s and restart it if it is at or above %d%% of its descriptor limit", s.NoFileLimit, unitName, s.RestartSaturation))
		return status, nil
	}

	home, err := homeDir()
	if err != nil || strings.TrimSpace(home) == "" {
		status.ExecutionState = hostreqkit.ExecutionFailed
		status.Notes = append(status.Notes, "could not determine the invoking user's home directory")
		return status, nil
	}
	fail := func(step string, err error) (hostreqkit.ItemStatus, error) {
		status.ExecutionState = hostreqkit.ExecutionFailed
		status.Notes = append(status.Notes, step+": "+err.Error())
		return status, nil
	}

	dropIn := filepath.Join(home, dropInDir, dropInName)
	installed := false
	if !hostreqkit.FileContentMatches(dropIn, dropInContent(s)) {
		// The file lives in the operator's home, so it is written as the operator.
		if err := hostreqkit.InstallUserFile(dropIn, dropInContent(s), opts); err != nil {
			return fail("install "+dropIn, err)
		}
		if _, err := userSystemctl("daemon-reload"); err != nil {
			return fail("systemctl --user daemon-reload", err)
		}
		installed = true
		status.Notes = append(status.Notes, fmt.Sprintf("LimitNOFILE=%d installed for %s", s.NoFileLimit, unitName))
	}

	// The new limit only binds a freshly started daemon. Restart when the
	// running one is already near its limit or the operator declared a
	// maintenance window; otherwise leave a live, healthy daemon alone and let
	// the next login or restart pick the limit up.
	state := observeDaemon()
	needsRestart := state.PID > 0 && state.SoftLimit > 0 && state.saturationPercent() >= s.RestartSaturation
	limitLow := state.PID > 0 && state.SoftLimit > 0 && state.SoftLimit < s.NoFileLimit
	switch {
	case needsRestart || (opts.MaintenanceWindow && limitLow):
		if _, err := userSystemctl("restart", unitName); err != nil {
			return fail("restart "+unitName, err)
		}
		status.Notes = append(status.Notes, fmt.Sprintf("restarted %s (was at %d of %d descriptors)", unitName, state.OpenFDs, state.SoftLimit))
	case limitLow:
		status.Notes = append(status.Notes, fmt.Sprintf("running daemon still has a %d-descriptor limit at %d%% use; the new limit binds at its next restart", state.SoftLimit, state.saturationPercent()))
	case installed && state.PID == 0:
		status.Notes = append(status.Notes, "daemon not running in this session; the limit binds when it starts")
	}

	status.Applied = true
	status.ExecutionState = hostreqkit.ExecutionApplied
	return status, nil
}
