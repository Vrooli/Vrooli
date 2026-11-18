//go:build linux

package setupprivilege

import (
	"fmt"
	"os"
	"syscall"
)

func ensureSudoersDirectory() error {
	for _, path := range []string{"/etc", "/etc/sudoers.d"} {
		info, err := os.Lstat(path)
		if os.IsNotExist(err) && path == "/etc/sudoers.d" {
			if err := os.Mkdir(path, 0o755); err != nil && !os.IsExist(err) {
				return err
			}
			info, err = os.Lstat(path)
		}
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() || info.Mode().Perm()&0o022 != 0 {
			return fmt.Errorf("%s is not a protected real directory", path)
		}
		stat, ok := info.Sys().(*syscall.Stat_t)
		if !ok || stat.Uid != 0 {
			return fmt.Errorf("%s is not owned by root", path)
		}
	}
	return nil
}
