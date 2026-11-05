package agentharness

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// PermissionFileSnapshot binds preview to exact bytes, existence and mode.
// Permission writers refuse symlinks rather than replacing their targets.
type PermissionFileSnapshot struct {
	Path     string
	Data     []byte
	Mode     fs.FileMode
	Security string
	Exists   bool
}

func ReadPermissionFile(path string) (PermissionFileSnapshot, error) {
	s := PermissionFileSnapshot{Path: path, Mode: 0o600}
	info, err := os.Lstat(path)
	if errors.Is(err, fs.ErrNotExist) {
		return s, nil
	}
	if err != nil {
		return s, err
	}
	if !info.Mode().IsRegular() {
		return s, fmt.Errorf("permission file %s must be a regular file (symlinks are unsupported)", path)
	}
	s.Exists = true
	s.Mode = info.Mode().Perm()
	s.Security, err = permissionFileSecurity(path)
	if err != nil {
		return s, err
	}
	s.Data, err = os.ReadFile(path)
	return s, err
}

func PermissionFilesDigest(snapshots ...PermissionFileSnapshot) string {
	hash := sha256.New()
	for _, s := range snapshots {
		fmt.Fprintf(hash, "%d:%s:%t:%o:%d:%d:", len(s.Path), s.Path, s.Exists, s.Mode, len(s.Security), len(s.Data))
		hash.Write([]byte(s.Security))
		hash.Write(s.Data)
	}
	return hex.EncodeToString(hash.Sum(nil))
}

// PublishPermissionFile preserves existing modes, uses a private unique temp
// file, and detects edits since the supplied snapshot. Call under WithLock.
func PublishPermissionFile(before PermissionFileSnapshot, data []byte) error {
	current, err := ReadPermissionFile(before.Path)
	if err != nil {
		return err
	}
	if PermissionFilesDigest(current) != PermissionFilesDigest(before) {
		return fmt.Errorf("permission file changed since preview: %s", before.Path)
	}
	if before.Exists && bytes.Equal(before.Data, data) {
		return nil
	}
	if err = os.MkdirAll(filepath.Dir(before.Path), 0o700); err != nil {
		return err
	}
	temp, err := CreatePermissionTemp(filepath.Dir(before.Path), ".vrooli-permissions-")
	if err != nil {
		return err
	}
	name := temp.Name()
	defer os.Remove(name)
	if err = temp.Chmod(before.Mode); err == nil {
		err = SecurePermissionFile(name, before.Security)
	}
	if err == nil {
		_, err = temp.Write(data)
	}
	if err == nil {
		err = temp.Sync()
	}
	closeErr := temp.Close()
	if err == nil {
		err = closeErr
	}
	if err != nil {
		return err
	}
	current, err = ReadPermissionFile(before.Path)
	if err != nil {
		return err
	}
	if PermissionFilesDigest(current) != PermissionFilesDigest(before) {
		return fmt.Errorf("permission file changed during write: %s", before.Path)
	}
	if err = os.Rename(name, before.Path); err != nil {
		return err
	}
	return nil
}
