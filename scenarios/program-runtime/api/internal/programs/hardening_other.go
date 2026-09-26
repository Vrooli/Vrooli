//go:build !linux

package programs

import "os/exec"

// Other hosts retain the portable environment, workdir, and wall-clock
// controls. Resource limits require host-specific APIs and remain a no-op
// until a native implementation is added for that platform.
func configureProcessGroup(_ *exec.Cmd) error           { return nil }
func applyProcessLimits(_ int, _ ExecutionLimits) error { return nil }
func killProcessGroup(_ int)                            {}
