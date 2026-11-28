//go:build linux

package cliutil

import (
	"io/fs"
	"os"
	"syscall"
)

// statChangeToken returns the inode change time and inode number. Unlike
// mtime, ctime cannot be set from user space, so a content rewrite that
// restores size and mtime still moves it.
func statChangeToken(info fs.FileInfo) (int64, uint64) {
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return 0, 0
	}
	return stat.Ctim.Nano(), stat.Ino
}

// runningExecutableIs reports whether path is the file this process runs. A
// reinstall replaces the file at path, after which they differ.
func runningExecutableIs(path string) bool {
	installed, err := os.Stat(path)
	if err != nil {
		return false
	}
	running, err := os.Stat("/proc/self/exe")
	return err == nil && os.SameFile(installed, running)
}
