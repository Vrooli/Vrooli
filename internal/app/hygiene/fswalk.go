package hygiene

import (
	"errors"
	"io/fs"
	"path/filepath"
)

// A hygiene run walks real contributor checkouts, which routinely contain paths
// the current user cannot read: root-owned artifact directories left by other
// agents, caches written under a different uid, mount points. A single
// unreadable entry must degrade that one path, not abort every check in the run.

// skipUnreadable converts a permission error surfaced to a filepath.WalkDir
// callback into a skip -- of the whole subtree for a directory, or just the
// entry for a file -- so the walk continues. Any other walk error is returned
// unchanged so genuine faults still fail loudly. entry may be nil when WalkDir
// could not descend into the path it is reporting.
func skipUnreadable(walkErr error, entry fs.DirEntry) error {
	if walkErr == nil {
		return nil
	}
	if isUnreadable(walkErr) {
		if entry != nil && entry.IsDir() {
			return filepath.SkipDir
		}
		return nil
	}
	return walkErr
}

// isUnreadable reports whether err is a permission error, for the in-callback
// os.ReadDir/os.ReadFile/os.Stat calls that WalkDir does not route through its
// own error parameter.
func isUnreadable(err error) bool {
	return errors.Is(err, fs.ErrPermission)
}
