package runtime

import (
	"strings"

	"github.com/vrooli/vrooli/internal/artifactlock"
)

var toolInstallLockPathFn = artifactlock.LockPath

// acquireToolInstallLock serializes a managed host-tool convergence across
// concurrent vrooli processes. The process-local mutex avoids needless flock
// contention; the advisory file lock covers separate processes.
func acquireToolInstallLock(tool string) (func(), error) {
	key := strings.TrimSpace(tool)
	return artifactlock.AcquireWithPath(key, toolInstallLockPathFn)
}
