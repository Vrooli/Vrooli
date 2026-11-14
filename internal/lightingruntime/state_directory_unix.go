//go:build !windows

package lightingruntime

import (
	"errors"
	"os"

	"github.com/vrooli/vrooli/internal/config"
	"golang.org/x/sys/unix"
)

func verifyLightingStateDirectory(path string, write bool) error {
	uid, _ := config.RepairIdentity()
	return verifyLightingStateDirectoryForOwner(path, write, uid)
}

func verifyLightingStateDirectoryForOwner(path string, write bool, uid uint32) error {
	info, err := os.Lstat(path)
	if err != nil {
		return err
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return errors.New("lighting state must be a real directory")
	}
	fd, err := unix.Open(path, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0)
	if err != nil {
		return err
	}
	file := os.NewFile(uintptr(fd), path)
	defer file.Close()
	opened, err := file.Stat()
	if err != nil || !os.SameFile(info, opened) {
		return errors.Join(errors.New("lighting state directory changed while opening"), err)
	}
	var st unix.Stat_t
	if err = unix.Fstat(fd, &st); err != nil {
		return err
	}
	if st.Uid != uid {
		return errors.New("lighting state directory belongs to another owner")
	}
	if opened.Mode().Perm()&0o022 != 0 {
		return errors.New("lighting state directory permits foreign writes")
	}
	if write && opened.Mode().Perm() != 0o700 {
		if err = file.Chmod(0o700); err != nil {
			return err
		}
		if err = file.Sync(); err != nil {
			return err
		}
	}
	after, err := file.Stat()
	if err != nil || after.Mode().Perm()&0o077 != 0 {
		return errors.Join(errors.New("lighting state directory must be private"), err)
	}
	current, err := os.Lstat(path)
	if err != nil || !current.IsDir() || current.Mode()&os.ModeSymlink != 0 || !os.SameFile(after, current) {
		return errors.Join(errors.New("lighting state directory changed during verification"), err)
	}
	return nil
}
