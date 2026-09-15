package resources

import (
	"path/filepath"
	"strings"

	platform "github.com/vrooli/platform-go"
	"github.com/vrooli/vrooli/internal/process"
)

// Seams for the two process facts ownership is proved from. Tests replace
// them to exercise a host that cannot observe a process environment.
var (
	readManagedServiceEnvironment = process.ReadEnvironment
	managedServiceExecutablePath  = platform.ProcessExecutablePath
)

// managedServiceExecutableMatchesArtifact is the process-identity ownership
// proof: the live executable must be the recorded artifact or lie inside the
// recorded artifact bundle. It never accepts a PID, port, or process name on
// its own. Hosts that cannot name a process executable never match.
func managedServiceExecutableMatchesArtifact(state ManagedServiceState) bool {
	if state.PID <= 0 || strings.TrimSpace(state.ArtifactPath) == "" {
		return false
	}
	executable, err := managedServiceExecutablePath(state.PID)
	if err != nil || strings.TrimSpace(executable) == "" {
		return false
	}
	executable, artifact := filepath.Clean(executable), filepath.Clean(state.ArtifactPath)
	if executableWithinArtifact(executable, artifact) {
		return true
	}
	// Linux reports a resolved executable while darwin reports the path the
	// process was started with, so a symlinked artifact (a Homebrew prefix
	// link) is compared in resolved form as well.
	resolvedExecutable, execErr := filepath.EvalSymlinks(executable)
	resolvedArtifact, artifactErr := filepath.EvalSymlinks(artifact)
	return execErr == nil && artifactErr == nil && executableWithinArtifact(resolvedExecutable, resolvedArtifact)
}

func executableWithinArtifact(executable, artifact string) bool {
	if executable == artifact {
		return true
	}
	relative, err := filepath.Rel(artifact, executable)
	return err == nil && relative != "." && relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator)) && !filepath.IsAbs(relative)
}
