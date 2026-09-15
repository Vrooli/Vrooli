//go:build linux

package resources

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

func managedServiceExecutableMatchesArtifact(state ManagedServiceState) bool {
	if state.PID <= 0 || strings.TrimSpace(state.ArtifactPath) == "" {
		return false
	}
	executable, err := os.Readlink(filepath.Join("/proc", strconv.Itoa(state.PID), "exe"))
	if err != nil {
		return false
	}
	executable = filepath.Clean(executable)
	artifact := filepath.Clean(state.ArtifactPath)
	if executable == artifact {
		return true
	}
	relative, err := filepath.Rel(artifact, executable)
	return err == nil && relative != "." && relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator)) && !filepath.IsAbs(relative)
}
