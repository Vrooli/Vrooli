//go:build aix || darwin || dragonfly || freebsd || linux || netbsd || openbsd || solaris

package runs

import (
	"os"
	"syscall"
)

func lockFile(file *os.File) (func(), error) {
	if err := syscall.Flock(int(file.Fd()), syscall.LOCK_EX); err != nil {
		return nil, err
	}
	return func() { _ = syscall.Flock(int(file.Fd()), syscall.LOCK_UN) }, nil
}
