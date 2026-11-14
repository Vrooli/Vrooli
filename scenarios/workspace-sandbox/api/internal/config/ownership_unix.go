//go:build aix || darwin || dragonfly || freebsd || linux || netbsd || openbsd || solaris

package config

import (
	"fmt"
	"os"
	"syscall"
)

func validateAuthoritativeDirectory(path string, info os.FileInfo) error {
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return fmt.Errorf("could not inspect ownership metadata")
	}
	if uint32(stat.Uid) != uint32(os.Geteuid()) {
		return fmt.Errorf("directory owner uid %d does not match process uid %d", stat.Uid, os.Geteuid())
	}
	if info.Mode().Perm()&0o077 != 0 {
		return fmt.Errorf("directory permissions %04o expose group or other access; expected owner-only access", info.Mode().Perm())
	}
	return nil
}
