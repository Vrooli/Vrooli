//go:build linux

package platform

import "golang.org/x/sys/unix"

// The Darwin engine's hermetic Linux fixtures exercise the same atomic
// no-replace/single-link commit semantics; this does not install a Darwin unit.
func darwinSystemBrokerPublishExclusive(source, destination string) error {
	return unix.Renameat2(unix.AT_FDCWD, source, unix.AT_FDCWD, destination, unix.RENAME_NOREPLACE)
}
