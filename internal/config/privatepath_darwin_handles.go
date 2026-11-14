//go:build darwin

package config

import (
	"errors"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/sys/unix"
)

// Resolve each component relative to a pinned directory, never following a
// link during privileged ownership or ACL provisioning.
func openDarwinPrivateObject(path string, expected os.FileInfo) (*os.File, func(), error) {
	directory, _, err := openDarwinPrivateDirectory(filepath.Dir(path), false)
	if err != nil {
		return nil, nil, err
	}
	fd, err := unix.Openat(int(directory.Fd()), filepath.Base(path), unix.O_RDONLY|unix.O_CLOEXEC|unix.O_NOFOLLOW, 0)
	if err != nil {
		directory.Close()
		return nil, nil, err
	}
	file := os.NewFile(uintptr(fd), path)
	closeAll := func() { file.Close(); directory.Close() }
	actual, err := file.Stat()
	if err != nil || !os.SameFile(expected, actual) {
		closeAll()
		return nil, nil, errors.Join(errors.New("native private object identity changed"), err)
	}
	return file, closeAll, nil
}
func openDarwinPrivateDirectory(path string, create bool) (*os.File, bool, error) {
	return openDarwinDirectoryWithMode(path, create, 0o700)
}
func openDarwinDirectoryWithMode(path string, create bool, mode uint32) (*os.File, bool, error) {
	if !filepath.IsAbs(path) || filepath.Clean(path) != path {
		return nil, false, errors.New("native private directory requires a canonical absolute path")
	}
	fd, err := unix.Open("/", unix.O_RDONLY|unix.O_DIRECTORY|unix.O_CLOEXEC, 0)
	if err != nil {
		return nil, false, err
	}
	parts := strings.Split(strings.TrimPrefix(path, "/"), "/")
	createdFinal := false
	for i, part := range parts {
		if part == "" {
			continue
		}
		next, e := unix.Openat(fd, part, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_CLOEXEC|unix.O_NOFOLLOW, 0)
		if errors.Is(e, unix.ENOENT) && create {
			e = unix.Mkdirat(fd, part, mode)
			if e == nil && i == len(parts)-1 {
				createdFinal = true
			}
			if e != nil && !errors.Is(e, unix.EEXIST) {
				unix.Close(fd)
				return nil, false, e
			}
			next, e = unix.Openat(fd, part, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_CLOEXEC|unix.O_NOFOLLOW, 0)
		}
		unix.Close(fd)
		if e != nil {
			return nil, false, e
		}
		fd = next
	}
	return os.NewFile(uintptr(fd), path), createdFinal, nil
}

// EnsureTraversableServiceNamespace creates public traversal namespaces, not
// private data directories. Existing directory permissions are never widened.
func EnsureTraversableServiceNamespace(path string) error {
	file, _, err := openDarwinDirectoryWithMode(path, true, 0o755)
	if err != nil {
		return err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return err
	}
	if info.Mode().Perm()&0o022 != 0 || info.Mode().Perm()&0o111 != 0o111 {
		return errors.New("native service namespace must be non-writable and traversable; existing permissions were preserved")
	}
	return nil
}

func createNativePrivateTemp(parent string, _ PrivatePathPolicy) (privateFileTemporary, error) {
	directory, _, err := openDarwinPrivateDirectory(parent, false)
	if err != nil {
		return privateFileTemporary{}, err
	}
	name, err := privateTemporaryName()
	if err != nil {
		directory.Close()
		return privateFileTemporary{}, err
	}
	fd, err := unix.Openat(int(directory.Fd()), name, unix.O_RDWR|unix.O_CREAT|unix.O_EXCL|unix.O_CLOEXEC|unix.O_NOFOLLOW, 0o600)
	if err != nil {
		directory.Close()
		return privateFileTemporary{}, err
	}
	file := os.NewFile(uintptr(fd), filepath.Join(parent, name))
	return privateFileTemporary{File: file, Name: file.Name(), Publish: func(to string) error {
		if filepath.Dir(to) != parent {
			return errors.New("private publication target changed its pinned directory")
		}
		if err := unix.Renameat(int(directory.Fd()), name, int(directory.Fd()), filepath.Base(to)); err != nil {
			return err
		}
		return directory.Sync()
	}, Cleanup: func() { file.Close(); unix.Unlinkat(int(directory.Fd()), name, 0); directory.Close() }}, nil
}

func openNativePrivateLock(path string, p PrivatePathPolicy) (*os.File, func(), error) {
	directory, _, err := openDarwinPrivateDirectory(filepath.Dir(path), false)
	if err != nil {
		return nil, nil, err
	}
	fd, err := unix.Openat(int(directory.Fd()), filepath.Base(path), unix.O_RDWR|unix.O_CREAT|unix.O_EXCL|unix.O_CLOEXEC|unix.O_NOFOLLOW, 0o600)
	created := err == nil
	if errors.Is(err, unix.EEXIST) {
		fd, err = unix.Openat(int(directory.Fd()), filepath.Base(path), unix.O_RDWR|unix.O_CLOEXEC|unix.O_NOFOLLOW, 0)
	}
	if err != nil {
		directory.Close()
		return nil, nil, err
	}
	file := os.NewFile(uintptr(fd), path)
	closeAll := func() { file.Close(); directory.Close() }
	var st unix.Stat_t
	if err = unix.Fstat(fd, &st); err != nil || st.Mode&unix.S_IFMT != unix.S_IFREG || st.Nlink != 1 {
		closeAll()
		return nil, nil, errors.Join(errors.New("private lock must be a regular single-link file"), err)
	}
	if created {
		if err = configurePrivateWriterAccess(file, p); err == nil {
			err = assignPrivateWriterOwner(file, p)
		}
		if err != nil {
			closeAll()
			return nil, nil, err
		}
	}
	return file, closeAll, nil
}
