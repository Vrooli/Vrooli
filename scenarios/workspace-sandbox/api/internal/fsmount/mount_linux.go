//go:build linux

package fsmount

import "syscall"

// sysMountOverlay is the raw kernel-overlay fast path: syscall.Mount with
// the overlay filesystem type. Callers fall back to the `mount -t overlay`
// command when this returns a non-nil, non-ErrBackendUnsupported error.
func sysMountOverlay(target, optsString string) error {
	return syscall.Mount("overlay", target, "overlay", 0, optsString)
}

// sysUnmount is the raw kernel unmount fast path. lazy selects a
// MNT_DETACH-style detach that succeeds even with open files.
func sysUnmount(target string, lazy bool) error {
	flags := 0
	if lazy {
		flags = syscall.MNT_DETACH
	}
	return syscall.Unmount(target, flags)
}
