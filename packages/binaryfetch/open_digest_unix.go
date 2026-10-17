//go:build aix || darwin || dragonfly || freebsd || linux || netbsd || openbsd || solaris

package binaryfetch

import (
	"os"
	"syscall"
)

// Nonblocking open prevents a replaced FIFO from blocking before fstat rejects
// it. Follow final symlinks to preserve legacy file artifact semantics.
func openDigestFile(path string) (*os.File, error) {
	return os.OpenFile(path, os.O_RDONLY|syscall.O_NONBLOCK, 0)
}
