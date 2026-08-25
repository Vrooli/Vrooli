package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/vrooli/vrooli/internal/hostreqkit"
)

const defaultVrooliDirPerm = 0o755

// RepairIdentity returns the only identity that a managed runtime-home repair
// may target: the sudo invoking user when present, otherwise the current
// process identity.
func RepairIdentity() (uint32, uint32) {
	if uid, gid, ok := invokingRepairIdentity(); ok {
		return uid, gid
	}
	return currentRepairIdentity()
}

// EnsureOwnedDir creates dir (and any missing ancestors) and — when the process
// is root via sudo — chowns exactly the components this call created back to the
// invoking user. Components that already existed are left untouched, so a
// sudo'd vrooli never leaves root-owned directories in the operator's home and
// never disturbs pre-existing ownership. Returns the directory path.
func EnsureOwnedDir(dir string) (string, error) {
	if dir == "" {
		return "", fmt.Errorf("config: empty directory path")
	}
	created := missingAncestors(dir)
	if err := os.MkdirAll(dir, defaultVrooliDirPerm); err != nil {
		return "", err
	}
	if err := chownCreatedToInvokingUser(created); err != nil {
		return "", err
	}
	return dir, nil
}

// WriteOwnedFile ensures the parent directory exists (owned), writes the file,
// then chowns the file to the invoking user under sudo.
func WriteOwnedFile(path string, data []byte, perm os.FileMode) error {
	return WriteOwnedFileAtomic(path, data, perm)
}

// WriteOwnedFileAtomic replaces a managed file through a same-directory
// temporary file. The temporary file is explicitly chmod/chowned before the
// rename, and the final path is rechecked after replacement. This keeps setup
// sudo from leaving either a root-owned temporary or a partially-written file.
func WriteOwnedFileAtomic(path string, data []byte, perm os.FileMode) error {
	if path == "" {
		return fmt.Errorf("config: empty file path")
	}
	if _, err := EnsureOwnedDir(filepath.Dir(path)); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".vrooli-owned-*")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)
	if err := tmp.Chmod(perm); err != nil {
		_ = tmp.Close()
		return err
	}
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := chownPathToInvokingUser(tmpPath); err != nil {
		return err
	}
	if err := os.Rename(tmpPath, path); err != nil {
		return err
	}
	return chownPathToInvokingUser(path)
}

// OpenOwnedFile opens a managed runtime file with an explicit mode and repairs
// ownership when setup is running for an invoking user. It is the append/lock
// counterpart to WriteOwnedFileAtomic for lifecycle records and logs.
func OpenOwnedFile(path string, flags int, perm os.FileMode) (*os.File, error) {
	if path == "" {
		return nil, fmt.Errorf("config: empty file path")
	}
	if _, err := EnsureOwnedDir(filepath.Dir(path)); err != nil {
		return nil, err
	}
	file, err := os.OpenFile(path, flags, perm)
	if err != nil {
		return nil, err
	}
	if err := chownPathToInvokingUser(path); err != nil {
		_ = file.Close()
		return nil, err
	}
	return file, nil
}

// EnsureVrooliDir resolves a runtime-home entry (repocontract.HomeKey*, plus any
// consumer sub-components) and ensures it exists with correct ownership.
func EnsureVrooliDir(key string, sub ...string) (string, error) {
	dir, err := VrooliPath(key, sub...)
	if err != nil {
		return "", err
	}
	return EnsureOwnedDir(dir)
}

// WriteVrooliFile resolves a runtime-home entry path (repocontract.HomeKey*,
// plus any consumer sub-components) and writes it with correct ownership.
func WriteVrooliFile(data []byte, perm os.FileMode, key string, sub ...string) error {
	path, err := VrooliPath(key, sub...)
	if err != nil {
		return err
	}
	return WriteOwnedFile(path, data, perm)
}

// ReconcileVrooliOwnership heals pre-existing strays: when the process is root
// via sudo, it walks the resolved runtime home and reclaims every entry still
// owned by root to the invoking user. It only ever touches root-owned entries
// (never reassigns a non-root owner, so multi-user hosts are safe), never
// follows symlinks, and never escapes the home root. Returns the number of
// entries re-owned. A no-op (0, nil) when not running root-via-sudo or when the
// home does not exist. This is what cleans up the root-owned files a sudo'd
// vrooli left behind before the owned-write seam existed.
func ReconcileVrooliOwnership() (int, error) {
	uid, gid, ok := hostreqkit.InvokingUserIDs()
	if !ok {
		return 0, nil
	}
	home, err := VrooliHome()
	if err != nil {
		return 0, err
	}
	if _, statErr := os.Lstat(home); statErr != nil {
		if os.IsNotExist(statErr) {
			return 0, nil
		}
		return 0, statErr
	}
	return reconcileHomeOwnership(home, uid, gid)
}

// ChownToInvokingUser chowns a single existing path to the invoking (sudo) user
// when it lies within the resolved runtime home. It is a no-op when the process
// is not root-via-sudo. Use this for home writes that don't flow through
// WriteOwnedFile — e.g. atomic temp+rename or O_EXCL lock files — so a sudo'd
// vrooli never leaves them root-owned.
func ChownToInvokingUser(path string) error {
	return chownPathToInvokingUser(path)
}

// missingAncestors returns the path and each ancestor that does not yet exist,
// deepest-first, stopping at the first component that already exists. This is
// exactly the set a subsequent MkdirAll will create, and therefore the set we
// must chown. Pure (no syscalls beyond Lstat, no chown) so the
// "which-ancestors-were-created" logic is unit-testable without root.
func missingAncestors(dir string) []string {
	dir = filepath.Clean(dir)
	var missing []string
	for p := dir; ; {
		if _, err := os.Lstat(p); err == nil {
			break
		}
		missing = append(missing, p)
		parent := filepath.Dir(p)
		if parent == p {
			break
		}
		p = parent
	}
	return missing
}
