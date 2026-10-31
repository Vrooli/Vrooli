//go:build !windows

package config

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/vrooli/vrooli/internal/hostreqkit"
)

// reconcileHomeOwnership walks home and Lchowns every entry currently owned by
// root (uid 0) to uid:gid. Entries owned by any non-root user are left
// untouched (CD-7). WalkDir does not descend into symlinked directories and
// Lchown does not follow symlinks, so ownership operations never escape home.
// Unreadable entries are skipped, not fatal.
func reconcileHomeOwnership(home string, uid, gid int) (int, error) {
	changed := 0
	walkErr := filepath.WalkDir(home, func(path string, _ fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		info, lerr := os.Lstat(path)
		if lerr != nil {
			return nil
		}
		st, ok := info.Sys().(*syscall.Stat_t)
		if !ok {
			return nil
		}
		if st.Uid != 0 {
			return nil // only reclaim root-owned entries
		}
		if err := os.Lchown(path, uid, gid); err == nil {
			changed++
		}
		return nil
	})
	return changed, walkErr
}

// chownCreatedToInvokingUser chowns each freshly-created component back to the
// invoking (sudo) user, bounded to within the resolved runtime home. It is a
// no-op when the process is not running root-via-sudo. Lchown is used so a
// symlink is never followed.
func chownCreatedToInvokingUser(created []string) error {
	uid, gid, ok := hostreqkit.InvokingUserIDs()
	if !ok || len(created) == 0 {
		return nil
	}
	boundary, err := VrooliHome()
	if err != nil {
		return err
	}
	for _, path := range created {
		if !withinHome(path, boundary) {
			continue
		}
		if err := os.Lchown(path, uid, gid); err != nil {
			return err
		}
	}
	return nil
}

// chownPathToInvokingUser chowns a single path to the invoking (sudo) user when
// it lies within the resolved runtime home. No-op otherwise.
func chownPathToInvokingUser(path string) error {
	uid, gid, ok := hostreqkit.InvokingUserIDs()
	if !ok {
		return nil
	}
	boundary, err := VrooliHome()
	if err != nil {
		return err
	}
	if !withinHome(path, boundary) {
		return nil
	}
	return os.Lchown(path, uid, gid)
}

// withinHome reports whether path is the home root or sits beneath it. The
// chown boundary: ownership operations never escape the resolved runtime home.
func withinHome(path, home string) bool {
	path = filepath.Clean(path)
	home = filepath.Clean(home)
	return path == home || strings.HasPrefix(path, home+string(filepath.Separator))
}
