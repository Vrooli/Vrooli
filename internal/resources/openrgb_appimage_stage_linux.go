//go:build linux

package resources

import (
	"errors"
	"fmt"
	"os"

	"golang.org/x/sys/unix"
)

func verifyOpenRGBFileCapabilities(path string) error {
	if _, err := unix.Lgetxattr(path, "security.capability", nil); !errors.Is(err, unix.ENODATA) && !errors.Is(err, unix.EOPNOTSUPP) {
		return errors.Join(fmt.Errorf("OpenRGB entry has capabilities or could not be verified: %s", path), err)
	}
	return nil
}

func openRGBArtifactOwnerUID() (uint32, error) { return uint32(os.Geteuid()), nil }

func openRGBArtifactNativeACLVerifier() (func(string, os.FileInfo, bool) error, error) {
	return func(string, os.FileInfo, bool) error { return nil }, nil
}

func openRGBArtifactNativeACLSealer() (func(string, os.FileInfo) error, error) {
	return func(string, os.FileInfo) error { return nil }, nil
}
