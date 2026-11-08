// Package authority decides what this storage-manager instance may mutate.
//
// storage-manager is the host's storage authority only when its own storage is
// not redirected. A test-genie isolated run or a desktop bundle starts it with
// VROOLI_STORAGE_ROOT pointing at a private tree: scenario owners then resolve
// inside that tree, but resources and safeguards still resolve to the host's
// native directories, and owner reclaim, Docker and Ollama act on live
// services. Such an instance may observe the host, but every destructive path
// (retention pruning, cleanup removal and renames, placement writes, owner
// reclaim, host commands) consults Scope first and stays inside the tree.
package authority

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// StorageRootEnv is api-core storage's global class-root override.
const StorageRootEnv = "VROOLI_STORAGE_ROOT"

// ErrOutOfScope marks a mutation this instance is not the authority for.
var ErrOutOfScope = errors.New("outside this storage-manager instance's storage authority")

// Scope is the storage this instance may mutate. The zero value follows the
// process environment at each check, so every destructive path is guarded
// without wiring; tests pin it with Host or Redirected.
type Scope struct {
	pinned bool
	root   string
}

// Host is unrestricted authority over the host's storage.
func Host() Scope { return Scope{pinned: true} }

// Redirected limits authority to root. An empty root is Host.
func Redirected(root string) Scope {
	return Scope{pinned: true, root: cleanRoot(root)}
}

func cleanRoot(root string) string {
	root = strings.TrimSpace(root)
	if root == "" {
		return ""
	}
	return filepath.Clean(root)
}

// Root is the redirected storage root, or "" for host authority.
func (s Scope) Root() string {
	if s.pinned {
		return s.root
	}
	return cleanRoot(os.Getenv(StorageRootEnv))
}

// IsHost reports whether this instance is the host's storage authority.
func (s Scope) IsHost() bool { return s.Root() == "" }

// CheckPath allows a mutation of path only beneath the redirected root.
func (s Scope) CheckPath(path string) error {
	root := s.Root()
	if root == "" {
		return nil
	}
	clean := filepath.Clean(path)
	rel, err := filepath.Rel(root, clean)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || filepath.IsAbs(rel) {
		return fmt.Errorf("%s is %w (storage redirected to %s)", clean, ErrOutOfScope, root)
	}
	return nil
}

// CheckHostAction allows an action on live host services (owner reclaim,
// container or model removal, privileged repair) only with host authority.
func (s Scope) CheckHostAction(action string) error {
	if root := s.Root(); root != "" {
		return fmt.Errorf("%s is %w (storage redirected to %s)", action, ErrOutOfScope, root)
	}
	return nil
}
