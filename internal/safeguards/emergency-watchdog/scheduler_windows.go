//go:build windows

package emergencywatchdog

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	platformgo "github.com/vrooli/platform-go"
	"github.com/vrooli/vrooli/internal/hostreqkit"
	"github.com/vrooli/vrooli/internal/hostreqspec"
	"github.com/vrooli/vrooli/internal/tuning"
)

// windowsTaskName is the Task Scheduler identity from the core unit table.
func windowsTaskName() string {
	unit, _ := platformgo.CoreUnitByID(platformgo.CoreUnitEmergencyWatchdog)
	return unit.Windows
}

func nativeSchedulerAvailable(goos string) bool {
	return goos == string(hostreqspec.PlatformWindows) && commandAvailable("schtasks.exe")
}

func guiLaunchdAvailable() bool { return false }

func nativePending(goos string, _ paths) []string {
	if goos != string(hostreqspec.PlatformWindows) {
		return []string{"unsupported scheduler"}
	}
	if _, err := hostreqkit.CombinedOutputFn("schtasks.exe", "/Query", "/TN", windowsTaskName()); err != nil {
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
		status.Notes = append(status.Notes, "dry-run: would register the Windows Task Scheduler watchdog task")
		return status, nil
	}
	artifact, err := renderedArtifact("windows", p)
	if err != nil {
		status.ExecutionState = hostreqkit.ExecutionFailed
		return status, err
	}
	verdict := platformgo.ValidateArtifact(artifact, platformgo.ScopeUser)
	recordVerdict(&status, verdict)
	if verdict.Rejected() {
		status.ExecutionState = hostreqkit.ExecutionFailed
		status.Notes = append(status.Notes, "the task document was rejected; nothing was registered: "+verdict.Output)
		return status, nil
	}
	xmlPath := filepath.Join(os.TempDir(), artifact.Primary().Name)
	if err := os.WriteFile(xmlPath, []byte(artifact.Primary().Content), tuning.PermFile); err != nil {
		status.ExecutionState = hostreqkit.ExecutionFailed
		return status, err
	}
	defer os.Remove(xmlPath)
	if _, err := hostreqkit.CombinedOutputFn("schtasks.exe", "/Create", "/TN", windowsTaskName(), "/XML", xmlPath, "/F"); err != nil {
		status.ExecutionState = hostreqkit.ExecutionFailed
		return status, fmt.Errorf("schtasks create: %w", err)
	}
	status.Applied = true
	status.ExecutionState = hostreqkit.ExecutionApplied
	status.Notes = append(status.Notes, "emergency watchdog installed in Windows Task Scheduler")
	return status, nil
}

func commandAvailable(name string) bool { _, err := exec.LookPath(name); return err == nil }
