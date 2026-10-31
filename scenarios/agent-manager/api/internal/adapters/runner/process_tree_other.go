//go:build !(aix || darwin || dragonfly || freebsd || linux || netbsd || openbsd || solaris)

package runner

import (
	"os/exec"
	"time"
)

// Windows has no portable process-group signal equivalent in the standard
// library. Process.Kill is the native handle-based termination primitive; the
// launcher never claims descendant cleanup guarantees on this path.
func killProcessTree(cmd *exec.Cmd) {
	if cmd != nil && cmd.Process != nil {
		_ = cmd.Process.Kill()
	}
}

func signalProcessTree(cmd *exec.Cmd, _ time.Duration) {
	killProcessTree(cmd)
}
