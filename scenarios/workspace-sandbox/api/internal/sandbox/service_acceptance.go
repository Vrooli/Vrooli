package sandbox

import (
	"path"
	"path/filepath"
	"strings"

	"workspace-sandbox/internal/diff"
	"workspace-sandbox/internal/types"
)

// service_acceptance.go: per-file acceptance evaluation.
//
// "Acceptance" is the set of allow/deny rules a sandbox declares at
// creation time. Each FileChange returned by GetDiff (or used in
// Approve) is annotated with an AcceptanceInfo telling reviewers (and
// the apply path) whether the file is eligible for approval. The
// glob/criteria machinery here is purely functional and side-effect
// free.

func normalizeCriteria(c types.FileCriteria) types.FileCriteria {
	paths := make([]string, 0, len(c.PathGlobs))
	seenPaths := make(map[string]bool)
	for _, p := range c.PathGlobs {
		p = strings.TrimSpace(p)
		if p == "" || seenPaths[p] {
			continue
		}
		seenPaths[p] = true
		paths = append(paths, p)
	}

	exts := make([]string, 0, len(c.Extensions))
	seenExts := make(map[string]bool)
	for _, ext := range c.Extensions {
		ext = strings.TrimSpace(ext)
		if ext == "" {
			continue
		}
		if !strings.HasPrefix(ext, ".") {
			ext = "." + ext
		}
		ext = strings.ToLower(ext)
		if seenExts[ext] {
			continue
		}
		seenExts[ext] = true
		exts = append(exts, ext)
	}

	c.PathGlobs = paths
	c.Extensions = exts
	return c
}

func applyAcceptanceInfo(sandbox *types.Sandbox, changes []*types.FileChange) {
	if len(changes) == 0 {
		return
	}
	for _, change := range changes {
		if change == nil {
			continue
		}
		change.Acceptance = evaluateAcceptance(sandbox, change)
	}
}

// evaluateAcceptance returns the acceptance status for a single change.
// noLock sandboxes accept everything; otherwise the deny rules are
// evaluated first (with a guard against the empty-criteria default-true
// trap), then the allow rules.
func evaluateAcceptance(sandbox *types.Sandbox, change *types.FileChange) *types.AcceptanceInfo {
	if sandbox.NoLock {
		return &types.AcceptanceInfo{
			Status: types.AcceptanceStatusAccepted,
			Reason: "no acceptance rules (noLock sandbox)",
		}
	}

	behavior := normalizeBehavior(sandbox.Behavior)
	acceptance := behavior.Acceptance

	relPath := projectRelativePath(sandbox, change.FilePath)
	ext := strings.ToLower(filepath.Ext(relPath))

	if acceptance.IgnoreBinary && isBinaryChange(sandbox, change) {
		return &types.AcceptanceInfo{
			Status: types.AcceptanceStatusBinaryIgnored,
			Reason: "binary file ignored",
		}
	}

	// IMPORTANT: Empty deny criteria must NOT match. matchesCriteria()
	// defaults to true when both PathGlobs and Extensions are empty
	// (since no conditions fail). Without this guard, `"deny": {}` would
	// deny ALL files. Mirrors the isCriteriaEmpty guard on allow below.
	if !isCriteriaEmpty(acceptance.Deny) && matchesCriteria(relPath, ext, acceptance.Deny) {
		return &types.AcceptanceInfo{
			Status: types.AcceptanceStatusDenied,
			Reason: "matched deny rules",
		}
	}

	if acceptance.Mode == "" || acceptance.Mode == "allowlist" {
		if isCriteriaEmpty(acceptance.Allow) || matchesCriteria(relPath, ext, acceptance.Allow) {
			return &types.AcceptanceInfo{
				Status: types.AcceptanceStatusAccepted,
				Reason: "matched allow rules",
			}
		}
		return &types.AcceptanceInfo{
			Status: types.AcceptanceStatusIgnored,
			Reason: "not matched by allow rules",
		}
	}

	return &types.AcceptanceInfo{
		Status: types.AcceptanceStatusAccepted,
		Reason: "acceptance mode default",
	}
}

func isCriteriaEmpty(c types.FileCriteria) bool {
	return len(c.PathGlobs) == 0 && len(c.Extensions) == 0
}

func matchesCriteria(relPath, ext string, criteria types.FileCriteria) bool {
	pathOK := true
	if len(criteria.PathGlobs) > 0 {
		pathOK = matchAnyGlob(criteria.PathGlobs, relPath)
	}
	extOK := true
	if len(criteria.Extensions) > 0 {
		extOK = containsString(criteria.Extensions, ext)
	}
	return pathOK && extOK
}

func matchAnyGlob(globs []string, relPath string) bool {
	for _, pattern := range globs {
		if matchGlob(pattern, relPath) {
			return true
		}
	}
	return false
}

func matchGlob(pattern, relPath string) bool {
	pattern = normalizeGlobPath(pattern)
	relPath = normalizeGlobPath(relPath)
	if pattern == "" {
		return relPath == ""
	}
	patParts := strings.Split(pattern, "/")
	pathParts := strings.Split(relPath, "/")
	return matchGlobParts(patParts, pathParts)
}

func normalizeGlobPath(value string) string {
	value = filepath.ToSlash(strings.TrimSpace(value))
	value = strings.TrimPrefix(value, "/")
	value = path.Clean(value)
	if value == "." {
		return ""
	}
	return value
}

func matchGlobParts(patternParts, pathParts []string) bool {
	if len(patternParts) == 0 {
		return len(pathParts) == 0
	}
	if patternParts[0] == "**" {
		for i := 0; i <= len(pathParts); i++ {
			if matchGlobParts(patternParts[1:], pathParts[i:]) {
				return true
			}
		}
		return false
	}
	if len(pathParts) == 0 {
		return false
	}
	if ok, _ := path.Match(patternParts[0], pathParts[0]); !ok {
		return false
	}
	return matchGlobParts(patternParts[1:], pathParts[1:])
}

func containsString(list []string, value string) bool {
	for _, item := range list {
		if item == value {
			return true
		}
	}
	return false
}

func projectRelativePath(sandbox *types.Sandbox, relScopePath string) string {
	if relScopePath == "" {
		return relScopePath
	}
	abs := filepath.Join(sandbox.ScopePath, relScopePath)
	rel, err := filepath.Rel(sandbox.ProjectRoot, abs)
	if err != nil {
		return filepath.ToSlash(relScopePath)
	}
	return filepath.ToSlash(rel)
}

// scopePathPrefix returns the relative path from ProjectRoot to ScopePath.
// This prefix is prepended to scope-relative file paths to make them
// project-relative for git apply. Empty when ScopePath == ProjectRoot.
func scopePathPrefix(sandbox *types.Sandbox) string {
	if sandbox.ScopePath == "" || sandbox.ProjectRoot == "" {
		return ""
	}
	cleanScope := filepath.Clean(sandbox.ScopePath)
	cleanProject := filepath.Clean(sandbox.ProjectRoot)
	if cleanScope == cleanProject {
		return ""
	}
	rel, err := filepath.Rel(cleanProject, cleanScope)
	if err != nil {
		return ""
	}
	return filepath.ToSlash(rel)
}

func isBinaryChange(sandbox *types.Sandbox, change *types.FileChange) bool {
	p := ""
	switch change.ChangeType {
	case types.ChangeTypeDeleted:
		if sandbox.LowerDir != "" {
			p = filepath.Join(sandbox.LowerDir, change.FilePath)
		}
	default:
		if sandbox.UpperDir != "" {
			p = filepath.Join(sandbox.UpperDir, change.FilePath)
		}
	}
	if p == "" {
		return false
	}
	ok, err := diff.IsBinaryFile(p)
	if err != nil {
		return false
	}
	return ok
}

// filterChangesByAcceptance partitions changes into (accepted, rejected)
// based on each change's acceptance evaluation. When override is true,
// every change is accepted regardless of rules.
func filterChangesByAcceptance(sandbox *types.Sandbox, changes []*types.FileChange, override bool) ([]*types.FileChange, []*types.FileChange) {
	if override {
		return changes, nil
	}
	accepted := make([]*types.FileChange, 0, len(changes))
	rejected := make([]*types.FileChange, 0)
	for _, change := range changes {
		if change == nil {
			continue
		}
		info := evaluateAcceptance(sandbox, change)
		change.Acceptance = info
		if info != nil && info.Status == types.AcceptanceStatusAccepted {
			accepted = append(accepted, change)
		} else {
			rejected = append(rejected, change)
		}
	}
	return accepted, rejected
}
