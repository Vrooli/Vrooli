//go:build linux

package desktophelper

import (
	"golang.org/x/sys/unix"
	"io"
	"os"
	"path/filepath"
)

func readPrivate(path string, limit int64) ([]byte, error) {
	if !filepath.IsAbs(path) {
		return nil, ErrBootstrap
	}
	fd, err := unix.Open(path, unix.O_RDONLY|unix.O_NOFOLLOW|unix.O_CLOEXEC|unix.O_NONBLOCK, 0)
	if err != nil {
		return nil, ErrBootstrap
	}
	file := os.NewFile(uintptr(fd), path)
	defer file.Close()
	var stat unix.Stat_t
	if unix.Fstat(fd, &stat) != nil || stat.Uid != uint32(os.Getuid()) || stat.Mode&unix.S_IFMT != unix.S_IFREG || stat.Mode&0077 != 0 || stat.Size > limit {
		return nil, ErrBootstrap
	}
	data, err := io.ReadAll(io.LimitReader(file, limit+1))
	if err != nil || int64(len(data)) > limit {
		return nil, ErrBootstrap
	}
	return data, nil
}
func privateDirectory(path string) error {
	if !filepath.IsAbs(path) {
		return ErrBootstrap
	}
	if err := os.MkdirAll(path, 0700); err != nil {
		return err
	}
	canonical, err := filepath.EvalSymlinks(path)
	if err != nil || canonical != filepath.Clean(path) {
		return ErrBootstrap
	}
	var stat unix.Stat_t
	if unix.Lstat(path, &stat) != nil || stat.Uid != uint32(os.Getuid()) || stat.Mode&unix.S_IFMT != unix.S_IFDIR || stat.Mode&0077 != 0 {
		return ErrBootstrap
	}
	return nil
}

func checkPrivateDatabase(path string) error {
	var stat unix.Stat_t
	if !filepath.IsAbs(path) || unix.Lstat(path, &stat) != nil || stat.Uid != uint32(os.Getuid()) || stat.Mode&unix.S_IFMT != unix.S_IFREG || stat.Mode&0077 != 0 {
		return ErrBootstrap
	}
	return nil
}

func lockDirectory(path string) (func(), error) {
	return lockPrivateFile(filepath.Join(path, "helper.lock"))
}

func lockPrivateFile(path string) (func(), error) {
	fd, err := unix.Open(path, unix.O_RDWR|unix.O_CREAT|unix.O_NOFOLLOW|unix.O_CLOEXEC|unix.O_NONBLOCK, 0600)
	if err != nil {
		return nil, ErrBootstrap
	}
	var stat unix.Stat_t
	if unix.Fstat(fd, &stat) != nil || stat.Uid != uint32(os.Getuid()) || stat.Mode&unix.S_IFMT != unix.S_IFREG || stat.Mode&0077 != 0 {
		unix.Close(fd)
		return nil, ErrBootstrap
	}
	if unix.Flock(fd, unix.LOCK_EX|unix.LOCK_NB) != nil {
		unix.Close(fd)
		return nil, ErrBootstrap
	}
	return func() { _ = unix.Flock(fd, unix.LOCK_UN); _ = unix.Close(fd) }, nil
}
