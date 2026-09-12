package retention

import (
	"fmt"
	"path/filepath"
	"strings"
)

// This file is the single authority for "may retention touch this path".
//
// It used to exist twice: once here, guarding the deletion boundary in
// DirectoryPruner, and once in storage-manager's enforcer adapter, guarding the
// entry boundary before a pruner was ever constructed. The two copies drifted.
// The copy guarding os.RemoveAll carried an extra `len(rel) >= 3` term, which
// silently exempted every child whose name was one or two characters long --
// ~/.vrooli/bin/go among them -- from a protected root. A predicate that
// decides whether a directory tree is destroyed is the last place a codebase
// should hold two implementations of, so there is now one, exported, and the
// adapter consumes it.

// NormalizeProtectedRoots validates and cleans a protected-root set.
//
// Every root must be absolute: a relative root would be resolved against the
// working directory of whichever process happened to run retention, which is
// exactly the kind of ambient-state dependency that makes a protection silently
// stop protecting.
func NormalizeProtectedRoots(roots []string) ([]string, error) {
	out := make([]string, 0, len(roots))
	for _, root := range roots {
		if strings.TrimSpace(root) == "" {
			continue
		}
		if !filepath.IsAbs(root) {
			return nil, fmt.Errorf("protected root %q must be absolute", root)
		}
		out = append(out, filepath.Clean(root))
	}
	return out, nil
}

// ProtectedPathOverlap reports whether acting on candidate would act on a
// protected root.
//
// Both directions of containment count, and for different reasons:
//
//   - candidate below a root: deleting ~/.vrooli/bin/vrooli deletes part of a
//     protected tree.
//   - candidate above a root: a broad retention target such as ~/.vrooli would
//     remove its protected ~/.vrooli/bin child as one top-level entry, so the
//     ancestor has to be refused before it is ever walked.
func ProtectedPathOverlap(candidate string, protectedRoots []string) bool {
	candidate = filepath.Clean(candidate)
	for _, root := range protectedRoots {
		root = filepath.Clean(root)
		if candidate == root || PathContains(root, candidate) || PathContains(candidate, root) {
			return true
		}
	}
	return false
}

// PathContains reports whether child lies strictly beneath parent.
//
// Equality is not containment: callers test that separately, because "the same
// path" and "inside that path" justify different messages.
func PathContains(parent, child string) bool {
	rel, err := filepath.Rel(parent, child)
	if err != nil {
		// Rel fails when the two paths cannot be related at all -- different
		// Windows volumes, for instance. Unrelated is not contained.
		return false
	}
	if rel == "." || rel == ".." {
		return false
	}
	return !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}
