//go:build !linux

package programs

import (
	"os"
	"os/exec"
)

// Other hosts retain the portable environment, workdir, and wall-clock
// controls. Resource limits require host-specific APIs and remain a no-op
// until a native implementation is added for that platform.
func configureProcessGroup(_ *exec.Cmd) error           { return nil }
func applyProcessLimits(_ int, _ ExecutionLimits) error { return nil }

// terminateKernelProcess can only kill the direct child on platforms without
// process-group support here. Descendants may survive and require a native
// platform-specific process-tree implementation.
func terminateKernelProcess(pid int) {
	process, err := os.FindProcess(pid)
	if err != nil {
		return
	}
	_ = process.Kill()
}
