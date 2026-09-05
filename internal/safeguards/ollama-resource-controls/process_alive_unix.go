//go:build !windows

package ollamaresourcecontrols

import (
	"errors"

	"golang.org/x/sys/unix"
)

// processAlivePID uses the platform's non-destructive signal probe. It is
// kept out of handler.go so the shared safeguard package also cross-compiles
// for Windows, where golang.org/x/sys/unix is unavailable.
func processAlivePID(pid int) bool {
	err := unix.Kill(pid, 0)
	return err == nil || errors.Is(err, unix.EPERM)
}
