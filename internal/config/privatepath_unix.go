//go:build !windows && !darwin

package config

import (
	"errors"
	"os"
	"path/filepath"
	"syscall"
)

func verifyNativePathAccess(string, os.FileInfo, PrivatePathPolicy) error {
	return errors.New("native artifact ACL qualification is unsupported on this platform")
}
func configureNativeReadOnlyPathAccess(string, os.FileInfo, PrivatePathPolicy) error {
	return errors.New("native artifact ACL qualification is unsupported on this platform")
}
func verifyNativePrivatePath(_ string, info os.FileInfo, p PrivatePathPolicy) error {
	if p.OwnerPrincipal != "" || p.RuntimePrincipal != "" {
		return errors.New("explicit native account ACLs are unsupported on this platform")
	}
	if info.Mode().Perm()&0o077 != 0 {
		return errors.New("private path has group or other permissions")
	}
	return nil
}
func ensureNativePrivateDirectory(path string, p PrivatePathPolicy) error {
	if p.OwnerPrincipal != "" || p.RuntimePrincipal != "" {
		return errors.New("explicit native account ACLs are unsupported on this platform")
	}
	if err := os.MkdirAll(path, 0o700); err != nil {
		return err
	}
	return nil
}
func publishNativePrivateFile(from, to string) error {
	if err := os.Rename(from, to); err != nil {
		return err
	}
	return syncOwnedDirectory(filepath.Dir(to))
}
func ConfigurePrivateDirectoryAccess(path string, p PrivatePathPolicy) error {
	return EnsurePrivateDirectory(path, p)
}
func assignPrivateWriterOwner(*os.File, PrivatePathPolicy) error     { return nil }
func configurePrivateWriterAccess(*os.File, PrivatePathPolicy) error { return nil }
func createNativePrivateTemp(parent string, _ PrivatePathPolicy) (privateFileTemporary, error) {
	file, err := os.CreateTemp(parent, ".vrooli-private-*")
	if err != nil {
		return privateFileTemporary{}, err
	}
	return privateFileTemporary{File: file, Name: file.Name(), Publish: func(to string) error { return publishNativePrivateFile(file.Name(), to) }, Cleanup: func() { file.Close(); os.Remove(file.Name()) }}, nil
}
func openNativePrivateLock(path string, _ PrivatePathPolicy) (*os.File, func(), error) {
	fd, err := syscall.Open(path, syscall.O_RDWR|syscall.O_CREAT|syscall.O_CLOEXEC|syscall.O_NOFOLLOW, 0o600)
	if err != nil {
		return nil, nil, err
	}
	file := os.NewFile(uintptr(fd), path)
	info, err := file.Stat()
	if err != nil {
		file.Close()
		return nil, nil, err
	}
	st, ok := info.Sys().(*syscall.Stat_t)
	if !info.Mode().IsRegular() || !ok || st.Nlink != 1 {
		file.Close()
		return nil, nil, errors.New("private lock must be a regular single-link file")
	}
	return file, func() { file.Close() }, nil
}
func removeNativePrivateFile(path string) error {
	if err := os.Remove(path); err != nil {
		return err
	}
	return syncOwnedDirectory(filepath.Dir(path))
}
