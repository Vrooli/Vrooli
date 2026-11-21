//go:build linux

package main

import "golang.org/x/sys/unix"

func publishRendererDirectory(stage, destination string) error {
	return unix.Renameat2(unix.AT_FDCWD, stage, unix.AT_FDCWD, destination, unix.RENAME_NOREPLACE)
}
