//go:build linux || darwin || freebsd || openbsd || netbsd || dragonfly

package runtimediagnostics

import (
	"fmt"
	"os"
	"syscall"
)

func ensurePrivateDirectory(directory string) error {
	if err := os.MkdirAll(directory, 0o700); err != nil {
		return err
	}
	info, err := os.Lstat(directory)
	if err != nil {
		return err
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !info.IsDir() || info.Mode().Perm()&0o077 != 0 || !ok || stat.Uid != uint32(os.Geteuid()) {
		return fmt.Errorf("runtime diagnostics requires an owner-only directory owned by the current user: %s", directory)
	}
	return nil
}
