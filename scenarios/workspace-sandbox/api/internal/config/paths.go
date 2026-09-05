package config

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// StoragePaths is the complete filesystem contract owned by workspace-sandbox.
// Every path is selected by the platform resolver; callers must not replace an
// individual path with an environment-derived value.
type StoragePaths struct {
	PersistentData string
	Transient      string
	Runtime        string
}

// PathFailure is an actionable startup failure for one authoritative path.
// Code is stable enough for health and diagnostics consumers; Path is included
// so an operator can repair the exact location without guessing.
type PathFailure struct {
	Code    string
	Purpose string
	Path    string
	Cause   error
}

func (e *PathFailure) Error() string {
	if e == nil {
		return "workspace-sandbox storage path failure"
	}
	return fmt.Sprintf("workspace-sandbox storage preflight failed: purpose=%s code=%s path=%q: %v", e.Purpose, e.Code, e.Path, e.Cause)
}

func (e *PathFailure) Unwrap() error { return e.Cause }

// ResolveStoragePaths chooses the one platform-native location for each
// storage purpose. It intentionally does not read XDG variables, scenario
// overrides, or fallback candidates.
func ResolveStoragePaths() (StoragePaths, error) {
	home, err := os.UserHomeDir()
	if err != nil || strings.TrimSpace(home) == "" {
		if err == nil {
			err = errors.New("user home directory is empty")
		}
		return StoragePaths{}, &PathFailure{Code: "HOME_UNAVAILABLE", Purpose: "persistent-data", Cause: err}
	}
	temp := strings.TrimSpace(os.TempDir())
	if temp == "" {
		return StoragePaths{}, &PathFailure{Code: "TEMP_UNAVAILABLE", Purpose: "transient", Cause: errors.New("system temporary directory is empty")}
	}
	return platformStoragePaths(filepath.Clean(home), filepath.Clean(temp)), nil
}

// StorageNamespace is deterministic and does not depend on USER, UID, or
// platform-specific account formats. It keeps transient state separate when a
// shared temporary directory is used by multiple users.
func StorageNamespace(home string) string {
	sum := sha256.Sum256([]byte(filepath.Clean(home)))
	return hex.EncodeToString(sum[:8])
}

// PrepareStoragePaths validates and creates every authoritative directory.
// It intentionally stops at the first failure and never tries another path.
func PrepareStoragePaths(paths StoragePaths) error {
	checks := []struct {
		purpose string
		path    string
	}{
		{"persistent-data", paths.PersistentData},
		{"transient", paths.Transient},
		{"runtime", paths.Runtime},
	}
	for _, check := range checks {
		if err := prepareAuthoritativeDirectory(check.path, check.purpose); err != nil {
			return err
		}
	}
	return nil
}

func prepareAuthoritativeDirectory(path, purpose string) error {
	path = filepath.Clean(strings.TrimSpace(path))
	if path == "." || path == "" {
		return &PathFailure{Code: "PATH_EMPTY", Purpose: purpose, Path: path, Cause: errors.New("authoritative path is empty")}
	}
	if err := os.MkdirAll(path, 0o700); err != nil {
		return &PathFailure{Code: "PATH_CREATE_FAILED", Purpose: purpose, Path: path, Cause: err}
	}
	info, err := os.Stat(path)
	if err != nil {
		return &PathFailure{Code: "PATH_STAT_FAILED", Purpose: purpose, Path: path, Cause: err}
	}
	if !info.IsDir() {
		return &PathFailure{Code: "PATH_NOT_DIRECTORY", Purpose: purpose, Path: path, Cause: errors.New("authoritative path is not a directory")}
	}
	// Tighten older installations in place. We never broaden access and never
	// select another directory when the authoritative one needs repair.
	if err := os.Chmod(path, 0o700); err != nil {
		return &PathFailure{Code: "PATH_MODE_REPAIR_FAILED", Purpose: purpose, Path: path, Cause: err}
	}
	info, err = os.Stat(path)
	if err != nil {
		return &PathFailure{Code: "PATH_STAT_FAILED", Purpose: purpose, Path: path, Cause: err}
	}
	if err := validateAuthoritativeDirectory(path, info); err != nil {
		return &PathFailure{Code: "PATH_OWNERSHIP_OR_MODE_INVALID", Purpose: purpose, Path: path, Cause: err}
	}
	probe, err := os.CreateTemp(path, ".workspace-sandbox-preflight-")
	if err != nil {
		return &PathFailure{Code: "PATH_NOT_WRITABLE", Purpose: purpose, Path: path, Cause: err}
	}
	probeName := probe.Name()
	if err := probe.Close(); err != nil {
		_ = os.Remove(probeName)
		return &PathFailure{Code: "PATH_PROBE_CLOSE_FAILED", Purpose: purpose, Path: path, Cause: err}
	}
	if err := os.Remove(probeName); err != nil {
		return &PathFailure{Code: "PATH_PROBE_CLEANUP_FAILED", Purpose: purpose, Path: path, Cause: err}
	}
	return nil
}
