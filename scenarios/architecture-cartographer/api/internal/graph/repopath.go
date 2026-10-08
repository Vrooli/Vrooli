// repopath.go — shared helper that fills PackageNode.RepoPath from
// the package's file nodes. Both language adapters call it after
// translating the producer's proto envelope, because no language
// code-graph scenario emits a per-package repo-relative directory.
//
// The package's RepoPath is the longest common directory prefix of its
// files' repo-relative paths (slash-separated). When a package has zero
// files, RepoPath stays empty and the package will not map to a domain
// — that's the correct behavior for synthetic / external packages.

package graph

import (
	"strings"
)

// AssignPackageRepoPaths fills PackageNode.RepoPath in place from the
// longest common directory prefix of every file's repo-relative path.
// Exported so the language adapters in subpackages can call it.
func AssignPackageRepoPaths(pkgs []PackageNode, files []FileNode) {
	filesByPkg := make(map[string][]string, len(pkgs))
	for _, f := range files {
		if f.PackageID == "" || f.Path == "" {
			continue
		}
		filesByPkg[f.PackageID] = append(filesByPkg[f.PackageID], f.Path)
	}
	for i := range pkgs {
		paths := filesByPkg[pkgs[i].ID]
		if len(paths) == 0 {
			continue
		}
		pkgs[i].RepoPath = commonDir(paths)
	}
}

// commonDir returns the longest common directory prefix of slash-separated
// repo-relative paths. A single path returns its directory.
func commonDir(paths []string) string {
	if len(paths) == 0 {
		return ""
	}
	dirs := make([][]string, len(paths))
	for i, p := range paths {
		dirs[i] = dirParts(p)
	}
	short := dirs[0]
	for _, d := range dirs[1:] {
		if len(d) < len(short) {
			short = d
		}
	}
	prefix := 0
	for prefix < len(short) {
		seg := short[prefix]
		match := true
		for _, d := range dirs {
			if d[prefix] != seg {
				match = false
				break
			}
		}
		if !match {
			break
		}
		prefix++
	}
	return strings.Join(short[:prefix], "/")
}

// dirParts splits a file path into its directory segments (drops the
// basename). "api/internal/graph/types.go" → ["api","internal","graph"].
func dirParts(path string) []string {
	path = strings.TrimPrefix(path, "/")
	idx := strings.LastIndex(path, "/")
	if idx < 0 {
		return nil
	}
	return strings.Split(path[:idx], "/")
}
