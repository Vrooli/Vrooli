package storage

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/vrooli/vrooli/internal/config"
)

const (
	DefaultDirPerm  = 0o755
	DefaultFilePerm = 0o644
)

func EnsureAllDirs(r *Resolver, opts Options, perm os.FileMode) (Paths, error) {
	if perm == 0 {
		perm = DefaultDirPerm
	}
	paths, err := r.Resolve(opts)
	if err != nil {
		return Paths{}, err
	}
	for _, dir := range []string{paths.ConfigDir, paths.DataDir, paths.CacheDir, paths.LogsDir, paths.StateDir} {
		if err := requireExistingAncestorOwnedByCurrentUser(dir); err != nil {
			return Paths{}, err
		}
		created := missingStorageAncestors(dir)
		if err := os.MkdirAll(dir, perm); err != nil {
			return Paths{}, &Error{Kind: ErrResolve, Message: "ensure storage dir", Details: dir, Err: err}
		}
		if err := restoreCreatedStorageOwnership(created); err != nil {
			return Paths{}, err
		}
		if err := requirePathOwnedByCurrentUser(dir); err != nil {
			return Paths{}, err
		}
	}
	return paths, nil
}

func missingStorageAncestors(dir string) []string {
	dir = filepath.Clean(dir)
	var missing []string
	for candidate := dir; ; candidate = filepath.Dir(candidate) {
		if _, err := os.Lstat(candidate); err == nil {
			return missing
		}
		missing = append(missing, candidate)
		parent := filepath.Dir(candidate)
		if parent == candidate {
			return missing
		}
	}
}

// requireExistingAncestorOwnedByCurrentUser prevents an alternate account (or
// an elevated helper) from creating per-user resource state beneath another
// user's XDG tree. MkdirAll otherwise succeeds whenever an ancestor is
// traversable, permanently stranding the intended user behind foreign-owned
// directories.
func requireExistingAncestorOwnedByCurrentUser(path string) error {
	for candidate := filepath.Clean(path); ; candidate = filepath.Dir(candidate) {
		info, err := os.Lstat(candidate)
		if err == nil {
			return requireFileInfoOwnedByCurrentUser(candidate, info)
		}
		if !os.IsNotExist(err) {
			return &Error{Kind: ErrResolve, Message: "inspect storage parent", Details: candidate, Err: err}
		}
		parent := filepath.Dir(candidate)
		if parent == candidate {
			return &Error{Kind: ErrResolve, Message: "find existing storage parent", Details: path, Err: fmt.Errorf("no existing ancestor")}
		}
	}
}

func requirePathOwnedByCurrentUser(path string) error {
	info, err := os.Lstat(path)
	if err != nil {
		return &Error{Kind: ErrResolve, Message: "inspect storage dir", Details: path, Err: err}
	}
	return requireFileInfoOwnedByCurrentUser(path, info)
}

func WriteFileAtomic(path string, data []byte, perm os.FileMode) error {
	if perm == 0 {
		perm = DefaultFilePerm
	}
	// Route managed runtime-home writes through the control-plane seam. It
	// preserves explicit modes, same-directory atomic replacement, and
	// sudo-aware ownership without making this shared storage package a host
	// remediation implementation.
	if isOutsideVrooliRuntimeHome(path) {
		// Resource storage may legitimately live in an XDG path outside the
		// Vrooli runtime home. Preserve that existing portable atomic contract.
		return writeFileAtomicUnmanaged(path, data, perm)
	}
	return config.WriteOwnedFileAtomic(path, data, perm)
}

func writeFileAtomicUnmanaged(path string, data []byte, perm os.FileMode) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, DefaultDirPerm); err != nil {
		return &Error{Kind: ErrResolve, Message: "ensure parent dir", Details: dir, Err: err}
	}
	tmp, err := os.CreateTemp(dir, ".tmp-*")
	if err != nil {
		return &Error{Kind: ErrResolve, Message: "create temp file", Details: dir, Err: err}
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return &Error{Kind: ErrResolve, Message: "write temp file", Details: tmpName, Err: err}
	}
	if err := tmp.Chmod(perm); err != nil {
		tmp.Close()
		return &Error{Kind: ErrResolve, Message: "chmod temp file", Details: tmpName, Err: err}
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return &Error{Kind: ErrResolve, Message: "sync temp file", Details: tmpName, Err: err}
	}
	if err := tmp.Close(); err != nil {
		return &Error{Kind: ErrResolve, Message: "close temp file", Details: tmpName, Err: err}
	}
	if err := os.Rename(tmpName, path); err != nil {
		return &Error{Kind: ErrResolve, Message: "rename temp file", Details: path, Err: err}
	}
	return nil
}

func isOutsideVrooliRuntimeHome(path string) bool {
	home, err := config.VrooliHome()
	if err != nil {
		return true
	}
	path, home = filepath.Clean(path), filepath.Clean(home)
	return path != home && !strings.HasPrefix(path, home+string(filepath.Separator))
}
