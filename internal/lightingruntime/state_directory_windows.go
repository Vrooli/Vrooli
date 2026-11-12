//go:build windows

package lightingruntime

import (
	"errors"
	"os"
)

// The ordinary intent retains the user-profile ACL inheritance used by its
// existing writer. Native resource private ACLs are a separate platform owner;
// Unix mode bits are not presented as a Windows DACL qualification.
func verifyLightingStateDirectory(path string, _ bool) error {
	info, err := os.Lstat(path)
	if err != nil {
		return err
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return errors.New("lighting state must be a real directory")
	}
	return nil
}
