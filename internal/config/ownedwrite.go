package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/vrooli/vrooli/internal/hostreqkit"
)

const defaultVrooliDirPerm = 0o755

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
	if path == "" {
		return fmt.Errorf("config: empty file path")
	}
	if _, err := EnsureOwnedDir(filepath.Dir(path)); err != nil {
		return err
	}
	if err := os.WriteFile(path, data, perm); err != nil {
		return err
	}
	return chownPathToInvokingUser(path)
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
