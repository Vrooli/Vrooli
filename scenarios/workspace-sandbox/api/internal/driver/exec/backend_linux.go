//go:build linux

package exec

import (
	"fmt"

	"workspace-sandbox/internal/process"
	"workspace-sandbox/internal/types"
)

// platformContainmentBackend returns the containment backend for this OS.
// On Linux that is bubblewrap.
func platformContainmentBackend() containmentBackend {
	return bwrapBackend{}
}

// bwrapBackend contains processes with bubblewrap: user/ipc/uts/cgroup
// (and optionally pid/net) namespace unsharing plus bind-mounted paths.
// Resource limits are applied by prepending prlimit (see BuildExecCommand).
type bwrapBackend struct{}

func (bwrapBackend) id() string { return "bwrap" }

func (bwrapBackend) available(starter process.Starter) error {
	if _, err := starter.LookPath("bwrap"); err != nil {
		return fmt.Errorf("bubblewrap (bwrap) not found: %w. Install with: apt-get install bubblewrap", err)
	}
	return nil
}

func (bwrapBackend) buildStartOpts(starter process.Starter, s *types.Sandbox, cfg BwrapConfig, cmd string, args ...string) (process.StartOpts, error) {
	executable, execArgs := BuildExecCommand(s, cfg, cmd, args...)
	execPath, err := starter.LookPath(executable)
	if err != nil {
		if executable == "prlimit" {
			return process.StartOpts{}, fmt.Errorf("prlimit not found: %w. Resource limits require prlimit (part of util-linux)", err)
		}
		return process.StartOpts{}, fmt.Errorf("bubblewrap (bwrap) not found: %w", err)
	}
	return process.StartOpts{
		Path: execPath,
		Args: append([]string(nil), execArgs...),
	}, nil
}
