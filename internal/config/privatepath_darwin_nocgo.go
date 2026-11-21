//go:build darwin && !cgo

package config

import (
	"fmt"
	platform "github.com/vrooli/platform-go"
	"os"
	"path/filepath"
)

func verifyNativePathAccess(string, os.FileInfo, PrivatePathPolicy) error {
	return fmt.Errorf("native Darwin ACL inspection requires CGO: %w", platform.ErrUnsupported)
}
func configureNativeReadOnlyPathAccess(string, os.FileInfo, PrivatePathPolicy) error {
	return fmt.Errorf("native Darwin ACL configuration requires CGO: %w", platform.ErrUnsupported)
}
func verifyNativePrivatePath(_ string, info os.FileInfo, p PrivatePathPolicy) error {
	if p.OwnerPrincipal != "" || p.RuntimePrincipal != "" {
		return fmt.Errorf("native Darwin ACL inspection requires CGO: %w", platform.ErrUnsupported)
	}
	if info.Mode().Perm()&0o077 != 0 {
		return fmt.Errorf("private path has group or other permissions")
	}
	return nil
}
func ensureNativePrivateDirectory(path string, p PrivatePathPolicy) error {
	if p.OwnerPrincipal != "" || p.RuntimePrincipal != "" {
		return fmt.Errorf("native Darwin ACL provisioning requires CGO: %w", platform.ErrUnsupported)
	}
	return os.MkdirAll(path, 0o700)
}
func ConfigurePrivateDirectoryAccess(path string, p PrivatePathPolicy) error {
	return ensureNativePrivateDirectory(path, p)
}
func assignPrivateWriterOwner(*os.File, PrivatePathPolicy) error     { return nil }
func configurePrivateWriterAccess(*os.File, PrivatePathPolicy) error { return nil }
func publishNativePrivateFile(from, to string) error {
	if err := os.Rename(from, to); err != nil {
		return err
	}
	return syncOwnedDirectory(filepath.Dir(to))
}
func removeNativePrivateFile(path string) error {
	if err := os.Remove(path); err != nil {
		return err
	}
	return syncOwnedDirectory(filepath.Dir(path))
}
