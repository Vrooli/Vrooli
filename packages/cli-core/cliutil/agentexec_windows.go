//go:build windows

package cliutil

import "errors"

// execReplaceSupported is false on Windows: the platform has no execve-style
// image replacement, so the launcher necessarily remains in the process tree
// and falls back to spawn-and-wait with exit-status propagation.
const execReplaceSupported = false

// execReplace always fails on Windows. Callers must treat a non-nil return as
// "fall back to spawning", never as a launch failure.
func execReplace(string, []string, []string) error {
	return errors.New("process image replacement is not supported on windows")
}
