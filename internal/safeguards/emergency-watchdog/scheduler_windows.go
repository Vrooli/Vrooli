//go:build windows

package emergencywatchdog

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/vrooli/vrooli/internal/hostreqkit"
	"github.com/vrooli/vrooli/internal/hostreqspec"
)

const windowsTaskName = "Vrooli Emergency Watchdog"

func nativeSchedulerAvailable(goos string) bool {
	return goos == string(hostreqspec.PlatformWindows) && commandAvailable("schtasks.exe")
}

func guiLaunchdAvailable() bool { return false }

func nativePending(goos string, _ paths) []string {
	if goos != string(hostreqspec.PlatformWindows) {
		return []string{"unsupported scheduler"}
	}
	if _, err := hostreqkit.CombinedOutputFn("schtasks.exe", "/Query", "/TN", windowsTaskName); err != nil {
		return []string{"Task Scheduler task missing or stale"}
	}
	return nil
}

func applyNative(goos string, p paths, status hostreqkit.ItemStatus, opts hostreqkit.EnsureOptions) (hostreqkit.ItemStatus, error) {
	if goos != string(hostreqspec.PlatformWindows) {
		status.ExecutionState = hostreqkit.ExecutionUnsupported
		return status, nil
	}
	if opts.DryRun {
		status.ExecutionState = hostreqkit.ExecutionWouldApply
		status.Notes = append(status.Notes, "dry-run: would create the Windows Task Scheduler watchdog task")
		return status, nil
	}
	command := windowsTaskCommand(p.Binary)
	args := []string{"/Create", "/TN", windowsTaskName, "/SC", "MINUTE", "/MO", "5", "/TR", command, "/F"}
	if _, err := hostreqkit.CombinedOutputFn("schtasks.exe", args...); err != nil {
		status.ExecutionState = hostreqkit.ExecutionFailed
		return status, fmt.Errorf("schtasks create: %w", err)
	}
	status.Applied = true
	status.ExecutionState = hostreqkit.ExecutionApplied
	status.Notes = append(status.Notes, "emergency watchdog installed in Windows Task Scheduler")
	return status, nil
}

func commandAvailable(name string) bool { _, err := exec.LookPath(name); return err == nil }
func windowsTaskCommand(executable string) string {
	return `"` + strings.ReplaceAll(executable, `"`, `\"`) + `" --report-only`
}
