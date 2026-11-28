//go:build !windows

package protogen

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"syscall"
)

// handBackOwnership gives a tree built by a privileged process to the owner of
// the directory it is about to be published into. Without it, a root-run
// `vrooli setup` leaves a root-owned compatibility view in the operator's
// checkout, and the operator's next unprivileged materialization cannot delete
// the superseded copy. An unprivileged process, or one that already shares the
// parent's owner, changes nothing.
func handBackOwnership(tree, parent string) error {
	if os.Geteuid() != 0 {
		return nil
	}
	info, err := os.Stat(parent)
	if err != nil {
		return err
	}
	owner, ok := info.Sys().(*syscall.Stat_t)
	if !ok || owner.Uid == 0 {
		return nil
	}
	return filepath.WalkDir(tree, func(path string, _ fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if err := os.Lchown(path, int(owner.Uid), int(owner.Gid)); err != nil {
			return fmt.Errorf("hand back ownership of %s: %w", path, err)
		}
		return nil
	})
}
