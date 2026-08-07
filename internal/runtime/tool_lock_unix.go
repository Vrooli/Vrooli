//go:build !windows

package runtime

import (
	"os"
	"syscall"
)

func lockToolInstallFile(file *os.File) error {
	return syscall.Flock(int(file.Fd()), syscall.LOCK_EX)
}

func unlockToolInstallFile(file *os.File) error {
	return syscall.Flock(int(file.Fd()), syscall.LOCK_UN)
}
