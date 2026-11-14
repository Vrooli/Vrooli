//go:build windows

package runs

import (
	"os"
	"unsafe"

	"golang.org/x/sys/windows"
)

func lockFile(file *os.File) (func(), error) {
	overlap := windows.Overlapped{}
	if err := windows.LockFileEx(
		windows.Handle(file.Fd()),
		windows.LOCKFILE_EXCLUSIVE_LOCK,
		0,
		1,
		0,
		&overlap,
	); err != nil {
		return nil, err
	}
	return func() {
		_ = windows.UnlockFileEx(windows.Handle(file.Fd()), 0, 1, 0, (*windows.Overlapped)(unsafe.Pointer(&overlap)))
	}, nil
}
