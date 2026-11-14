//go:build aix || darwin || dragonfly || freebsd || linux || netbsd || openbsd || solaris

package resources

import (
	"os"
	"syscall"
)

func openOpenRGBSourceFileNative(path string) (*os.File, error) {
	// A source swapped to a FIFO must not block an elevated setup before its
	// opened-object type and identity can be rejected.
	return os.OpenFile(path, os.O_RDONLY|syscall.O_NONBLOCK, 0)
}
