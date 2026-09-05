//go:build !windows

package cliutil

import "syscall"

// execReplaceSupported reports whether this platform can replace the running
// process image in place. Every Unix can; Windows cannot, and the launcher
// degrades to spawn-and-wait there.
const execReplaceSupported = true

// execReplace replaces the current process image with path, keeping the same
// pid, process group, controlling terminal, and open file descriptors.
//
// It returns only on failure. On success the calling process *is* the coding
// agent, which is the whole point: the launcher does its attribution work and
// then stops existing, so it can never interpose on the agent's terminal,
// signals, or exit status.
func execReplace(path string, argv []string, environment []string) error {
	return syscall.Exec(path, argv, environment)
}
