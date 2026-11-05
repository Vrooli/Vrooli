//go:build darwin

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

const launchAgentLabel = "com.vrooli.emergency-watchdog"

func nativeSchedulerAvailable(goos string) bool {
	return goos == string(hostreqspec.PlatformDarwin) && commandAvailable("launchctl")
}

func nativePending(goos string, p paths) []string {
	if goos != string(hostreqspec.PlatformDarwin) {
		return []string{"unsupported scheduler"}
	}
	var pending []string
	artifact, err := renderedArtifact("darwin", p)
	if err != nil {
		return []string{"render launchd agent: " + err.Error()}
	}
	if !hostreqkit.FileContentMatches(p.LaunchAgent, artifact.Primary().Content) {
		pending = append(pending, p.LaunchAgent+" missing or stale")
	}
	if _, err := hostreqkit.CombinedOutputFn("launchctl", "print", launchdTarget()); err != nil {
		pending = append(pending, "launchd agent not loaded")
	}
	return pending
}

// guiLaunchdAvailable reports whether the invoking user's GUI launchd domain
// exists. SSH-only sessions can still have launchctl installed while lacking
// that domain; attempting to load a user LaunchAgent there returns the opaque
// exit status 5 and must not make unrelated project setup fail.
func guiLaunchdAvailable() bool {
	_, err := hostreqkit.CombinedOutputFn("launchctl", "print", fmt.Sprintf("gui/%d", os.Getuid()))
	return err == nil
}

func applyNative(goos string, p paths, status hostreqkit.ItemStatus, opts hostreqkit.EnsureOptions) (hostreqkit.ItemStatus, error) {
	if goos != string(hostreqspec.PlatformDarwin) {
		status.ExecutionState = hostreqkit.ExecutionUnsupported
		return status, nil
	}
	if opts.DryRun {
		status.ExecutionState = hostreqkit.ExecutionWouldApply
		status.Notes = append(status.Notes, "dry-run: would install "+p.LaunchAgent+" and load the launchd agent")
		return status, nil
	}
	if !guiLaunchdAvailable() {
		status.SupportClass = hostreqkit.SupportNotApplicable
		status.ExecutionState = hostreqkit.ExecutionNotApplicable
		status.Notes = append(status.Notes, "the invoking user's GUI launchd domain is unavailable; this user LaunchAgent is not applicable to the current SSH/headless session")
		return status, nil
	}
	artifact, err := renderedArtifact("darwin", p)
	if err != nil {
		status.ExecutionState = hostreqkit.ExecutionFailed
		return status, err
	}
	verdict := platformgo.ValidateArtifact(artifact, platformgo.ScopeUser)
	recordVerdict(&status, verdict)
	if verdict.Rejected() {
		status.ExecutionState = hostreqkit.ExecutionFailed
		status.Notes = append(status.Notes, "plutil rejected the rendered LaunchAgent; nothing was installed: "+verdict.Output)
		return status, nil
	}
	if err := os.MkdirAll(filepath.Dir(p.LaunchAgent), tuning.PermDir); err != nil {
		status.ExecutionState = hostreqkit.ExecutionFailed
		return status, err
	}
	if err := os.WriteFile(p.LaunchAgent, []byte(artifact.Primary().Content), tuning.PermFile); err != nil {
		status.ExecutionState = hostreqkit.ExecutionFailed
		return status, err
	}
	_, _ = hostreqkit.CombinedOutputFn("launchctl", "bootout", launchdTarget())
	if _, err := hostreqkit.CombinedOutputFn("launchctl", "bootstrap", launchdDomain(), p.LaunchAgent); err != nil {
		status.ExecutionState = hostreqkit.ExecutionFailed
		return status, fmt.Errorf("launchctl bootstrap: %w", err)
	}
	if _, err := hostreqkit.CombinedOutputFn("launchctl", "enable", launchdTarget()); err != nil {
		status.ExecutionState = hostreqkit.ExecutionFailed
		return status, fmt.Errorf("launchctl enable: %w", err)
	}
	status.Applied = true
	status.ExecutionState = hostreqkit.ExecutionApplied
	status.Notes = append(status.Notes, "emergency watchdog installed and loaded by launchd")
	return status, nil
}

func commandAvailable(name string) bool { _, err := exec.LookPath(name); return err == nil }
func launchdDomain() string {
	uid := os.Getuid()
	gui := fmt.Sprintf("gui/%d", uid)
	if guiLaunchdAvailable() {
		return gui
	}
	return fmt.Sprintf("user/%d", uid)
}
func launchdTarget() string { return launchdDomain() + "/" + launchAgentLabel }
