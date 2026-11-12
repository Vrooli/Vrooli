//go:build darwin

package platform

import "golang.org/x/sys/unix"

func darwinSystemBrokerPublishExclusive(source, destination string) error {
	return unix.RenameatxNp(unix.AT_FDCWD, source, unix.AT_FDCWD, destination, unix.RENAME_EXCL)
}
