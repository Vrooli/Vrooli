// Package executioninventory defines the BAS-owned files that may live under
// one canonical execution directory. It is shared by the writer and purge
// verifier so inventory refresh cannot bless unrelated neighboring files.
package executioninventory

import (
	"path/filepath"
	"strings"

	"github.com/google/uuid"
)

var topLevelFiles = map[string]struct{}{
	"README.md":            {},
	"checkpoint.json":      {},
	"evidence.proto.json":  {},
	"execution.proto.json": {},
	"result.json":          {},
	"timeline.proto.json":  {},
}

var artifactRoots = map[string]struct{}{
	"accessibility": {},
	"console":       {},
	"dom":           {},
	"har":           {},
	"network":       {},
	"performance":   {},
	"screenshots":   {},
	"traces":        {},
	"videos":        {},
}

// IsOwnedPath accepts only known top-level files and known execution artifact
// subtrees. The inventory file is generated after this check and is excluded.
func IsOwnedPath(path string, executionIDs ...string) bool {
	if !isCanonicalRelativePath(path) {
		return false
	}
	if _, ok := topLevelFiles[path]; ok {
		return true
	}
	if !strings.HasPrefix(path, "artifacts/") {
		return false
	}
	parts := strings.Split(path, "/")
	if len(parts) < 3 {
		return false
	}
	if len(parts) == 3 && len(executionIDs) == 1 && executionIDs[0] != "" {
		if _, err := uuid.Parse(executionIDs[0]); err == nil && parts[1] == executionIDs[0] {
			return true
		}
	}
	_, ok := artifactRoots[parts[1]]
	return ok
}

// IsOwnedDirectory accepts only directories BAS may create under an execution.
func IsOwnedDirectory(path string, executionIDs ...string) bool {
	if !isCanonicalRelativePath(path) {
		return false
	}
	if path == "artifacts" {
		return true
	}
	if !strings.HasPrefix(path, "artifacts/") {
		return false
	}
	parts := strings.Split(path, "/")
	if len(parts) < 2 {
		return false
	}
	if len(parts) == 2 && len(executionIDs) == 1 && executionIDs[0] != "" {
		if _, err := uuid.Parse(executionIDs[0]); err == nil && parts[1] == executionIDs[0] {
			return true
		}
	}
	_, ok := artifactRoots[parts[1]]
	return ok
}

// IsExecutionObjectKey accepts only canonical object keys generated for one
// execution by BAS's screenshot/telemetry or generic artifact storage paths.
// It is for private purge ownership receipts; public evidence manifests must
// continue to omit object keys entirely.
func IsExecutionObjectKey(key, executionID string) bool {
	if _, err := uuid.Parse(executionID); err != nil || executionID == "" || !isCanonicalRelativePath(key) {
		return false
	}
	parts := strings.Split(key, "/")
	if len(parts) == 3 && parts[0] == "artifacts" && parts[1] == executionID {
		return parts[2] != "" && parts[2] != "." && parts[2] != ".."
	}
	if len(parts) != 4 || parts[0] != executionID || parts[1] != "artifacts" || parts[3] == "" {
		return false
	}
	_, ok := artifactRoots[parts[2]]
	return ok
}

func isCanonicalRelativePath(path string) bool {
	if path == "" || filepath.IsAbs(path) || strings.Contains(path, "\\") {
		return false
	}
	clean := filepath.ToSlash(filepath.Clean(filepath.FromSlash(path)))
	return clean == path && clean != "." && clean != ".." && !strings.HasPrefix(clean, "../")
}
