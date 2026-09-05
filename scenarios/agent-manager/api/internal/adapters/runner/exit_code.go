package runner

import (
	"errors"
	"os/exec"
)

// exitCoder is the interface implemented by both *exec.ExitError (host
// launches) and *sandbox.remoteExitError (sandbox launches). Defining it
// here keeps the runner package free of a sandbox-package dependency
// while letting both error types flow through the same extraction logic.
type exitCoder interface{ ExitCode() int }

// ExtractExitCode returns the process exit code from a launcher Wait error.
//
//   - nil err → (0, true).
//   - err implementing ExitCoder (host *exec.ExitError or sandbox
//     *remoteExitError) → (its ExitCode, true).
//   - any other error (typically a transport / launch failure with no
//     captured exit code) → (-1, false). Callers treat this as a generic
//     execution failure and surface err.Error() as the error message.
//
// Use this helper at the wait-error type-switch in every runner so
// protected-mode runs (which return *remoteExitError) report exit codes
// the same way tracking-mode runs (*exec.ExitError) do.
func ExtractExitCode(err error) (int, bool) {
	if err == nil {
		return 0, true
	}
	var ec exitCoder
	if errors.As(err, &ec) {
		return ec.ExitCode(), true
	}
	// Belt-and-suspenders: errors.As walks the wrap chain, but if the
	// caller passes a top-level *exec.ExitError on a Go version where
	// the As implementation has a quirk, the direct type assertion still
	// catches it. Cheap; keeps the helper resilient.
	if ee, ok := err.(*exec.ExitError); ok {
		return ee.ExitCode(), true
	}
	return -1, false
}
