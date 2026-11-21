//go:build !windows

package agentinstall

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"syscall"
)

func ownedByRoot(info os.FileInfo) bool {
	stat, ok := info.Sys().(*syscall.Stat_t)
	return ok && stat.Uid == 0
}

// writableByCaller asks the kernel, so group membership and ACLs count. A mode
// comparison would misjudge both.
func writableByCaller(path string) bool {
	const writeAccess = 2
	return syscall.Access(path, writeAccess) == nil
}

// handBackToBinDirOwner gives files a privileged install created to the owner
// of the install directory. `sudo vrooli setup` installs optional harnesses as
// root; without this the new executable is root-owned inside the operator's
// home, the managed update then reports needs_setup, and the next setup repeats
// the cycle. An unprivileged caller, or a root-owned BinDir, changes nothing.
func handBackToBinDirOwner(binDir string, paths ...string) error {
	if os.Geteuid() != 0 {
		return nil
	}
	info, err := os.Stat(binDir)
	if err != nil {
		return err
	}
	owner, ok := info.Sys().(*syscall.Stat_t)
	if !ok || owner.Uid == 0 {
		return nil
	}
	var failures []error
	for _, root := range paths {
		if root == "" {
			continue
		}
		walkErr := filepath.WalkDir(root, func(path string, _ fs.DirEntry, err error) error {
			if err != nil {
				if errors.Is(err, fs.ErrNotExist) {
					return nil
				}
				return err
			}
			// Lchown never follows a link, so a link cannot redirect the change.
			return os.Lchown(path, int(owner.Uid), int(owner.Gid))
		})
		if walkErr != nil {
			failures = append(failures, fmt.Errorf("hand back %s: %w", root, walkErr))
		}
	}
	return errors.Join(failures...)
}
