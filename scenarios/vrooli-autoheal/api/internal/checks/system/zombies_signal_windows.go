//go:build windows

package system

import "errors"

// signalChild has no Windows equivalent: the platform has no zombie process
// state and no SIGCHLD, so there is nothing for a parent to reap. The zombie
// check itself is Linux-only; this exists so the package builds everywhere.
func signalChild(int) error {
	return errors.New("reaping zombie processes is not supported on Windows")
}
