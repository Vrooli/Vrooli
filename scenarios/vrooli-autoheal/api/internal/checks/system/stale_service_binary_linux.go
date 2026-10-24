//go:build linux

package system

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

// linuxProcResolver asks systemd which process belongs to the unit, then reads
// that process's executable. The kernel appends " (deleted)" to
// /proc/<pid>/exe once the file is unlinked, which is precisely the "running
// replaced code" signal — no hashing or version comparison needed.
type linuxProcResolver struct{}

func (linuxProcResolver) Resolve(ctx context.Context, unit string) (string, bool, bool) {
	pid, ok := unitMainPID(ctx, unit)
	if !ok {
		return "", false, false
	}
	link, err := os.Readlink(filepath.Join("/proc", strconv.Itoa(pid), "exe"))
	if err != nil {
		// A process we cannot introspect is not evidence of staleness.
		return "", false, false
	}
	if trimmed := strings.TrimSuffix(link, " (deleted)"); trimmed != link {
		return trimmed, true, true
	}
	return link, false, true
}

// unitMainPID returns the unit's main process. systemd reports 0 for a unit
// that is not running, which is the same answer as "no process to inspect".
func unitMainPID(ctx context.Context, unit string) (int, bool) {
	out, err := exec.CommandContext(ctx, "systemctl", "--user", "show", "-p", "MainPID", "--value", unit).Output()
	if err != nil {
		return 0, false
	}
	pid, convErr := strconv.Atoi(strings.TrimSpace(string(out)))
	if convErr != nil || pid <= 0 {
		return 0, false
	}
	return pid, true
}

// restartUserUnit restarts a systemd user unit. Autoheal runs as the project
// user and these are user units, so this needs no privilege.
func restartUserUnit(ctx context.Context, unit string) (string, error) {
	out, err := exec.CommandContext(ctx, "systemctl", "--user", "restart", unit).CombinedOutput()
	return string(out), err
}
