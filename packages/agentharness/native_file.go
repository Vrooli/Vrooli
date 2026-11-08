package agentharness

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// ResolveNativeFilePath follows a symlinked native file to its target. A Web
// Console session home links its config to the shared one, and Vrooli manages
// the shared file: the writer locks, rewrites and keeps sidecars beside the
// target rather than failing on, or replacing, the link. A missing file
// resolves to itself.
func ResolveNativeFilePath(path string) (string, error) {
	info, err := os.Lstat(path)
	if errors.Is(err, fs.ErrNotExist) {
		return path, nil
	}
	if err != nil {
		return "", err
	}
	if info.Mode()&fs.ModeSymlink == 0 {
		return path, nil
	}
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil {
		return "", fmt.Errorf("resolve %s: %w", path, err)
	}
	return resolved, nil
}

// BackupNativeFile keeps two private copies of a native file that writer is
// about to rewrite: <file>.vrooli-<writer>.orig is the file as it was before
// that writer first rewrote it (written once, so formatting and comments a
// TOML round trip drops stay recoverable), and <file>.vrooli-<writer>.bak is
// the latest pre-image. It returns the .bak path; a missing file needs none.
func BackupNativeFile(before PermissionFileSnapshot, writer string) (string, error) {
	if !before.Exists {
		return "", nil
	}
	base := before.Path + ".vrooli-" + writer
	if _, err := os.Lstat(base + ".orig"); errors.Is(err, fs.ErrNotExist) {
		if err := writePrivateCopy(base+".orig", before.Data); err != nil {
			return "", fmt.Errorf("write original backup: %w", err)
		}
	} else if err != nil {
		return "", err
	}
	if err := writePrivateCopy(base+".bak", before.Data); err != nil {
		return "", fmt.Errorf("write backup: %w", err)
	}
	return base + ".bak", nil
}

// writePrivateCopy publishes data through a private temp file and a rename, so
// a backup is never observed half-written and never wider than owner-only.
func writePrivateCopy(path string, data []byte) error {
	temp, err := CreatePermissionTemp(filepath.Dir(path), ".vrooli-backup-")
	if err != nil {
		return err
	}
	name := temp.Name()
	defer os.Remove(name)
	if _, err = temp.Write(data); err == nil {
		err = temp.Sync()
	}
	if closeErr := temp.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		return err
	}
	return os.Rename(name, path)
}
