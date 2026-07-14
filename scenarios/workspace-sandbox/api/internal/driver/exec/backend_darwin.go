//go:build darwin

package exec

import (
	"fmt"
	"os"

	"workspace-sandbox/internal/process"
	"workspace-sandbox/internal/types"
)

// platformContainmentBackend returns the containment backend for this OS.
// On macOS that is Seatbelt (sandbox-exec).
func platformContainmentBackend() containmentBackend {
	return seatbeltBackend{}
}

// seatbeltBackend contains processes with macOS Seatbelt via the system
// sandbox-exec binary and a generated profile (see seatbelt.go). It is an
// honestly-partial backend: it enforces filesystem write-containment and
// network denial, but provides no path illusion and no pid namespace.
// Resource limits are applied by prepending the api binary's rlimit self-exec
// shim, replacing the Linux-only prlimit.
type seatbeltBackend struct{}

func (seatbeltBackend) id() string { return "seatbelt" }

func (seatbeltBackend) available(starter process.Starter) error {
	if _, err := starter.LookPath("sandbox-exec"); err != nil {
		return fmt.Errorf("sandbox-exec not found: %w. macOS Seatbelt containment requires the system sandbox-exec binary", err)
	}
	return nil
}

func (seatbeltBackend) buildStartOpts(starter process.Starter, s *types.Sandbox, cfg BwrapConfig, cmd string, args ...string) (process.StartOpts, error) {
	sandboxExec, err := starter.LookPath("sandbox-exec")
	if err != nil {
		return process.StartOpts{}, fmt.Errorf("sandbox-exec not found: %w", err)
	}

	// The rlimit shim is the api binary re-invoked with the rlimit-exec
	// subcommand; resolve it only when resource limits are configured.
	shimPath := ""
	if cfg.ResourceLimits.HasLimits() {
		shimPath, err = shimExecutablePath()
		if err != nil {
			return process.StartOpts{}, fmt.Errorf("resolve rlimit shim path: %w", err)
		}
	}

	_, execArgs := BuildSeatbeltCommand(shimPath, s, cfg, cmd, args...)
	return process.StartOpts{
		Path: sandboxExec,
		Args: append([]string(nil), execArgs...),
	}, nil
}

// shimExecutablePath returns the path to the running api binary, used as the
// rlimit self-exec shim. os.Executable resolves /proc/self/exe-equivalents;
// it is not a subprocess spawn, so it stays outside the process.Starter seam.
func shimExecutablePath() (string, error) {
	return os.Executable()
}
