package cliutil

import (
	"path/filepath"
	"strings"

	"github.com/vrooli/cli-core/buildinfo"
)

func expandDeclaredInputPaths(root, input string) ([]string, error) {
	if hasGlobPattern(input) {
		return filepath.Glob(filepath.Join(root, filepath.FromSlash(input)))
	}
	return []string{filepath.Join(root, filepath.FromSlash(input))}, nil
}

func hasGlobPattern(value string) bool {
	return strings.ContainsAny(value, "*?[")
}

func buildinfoSkipDir(root, path string) bool {
	path = strings.ReplaceAll(filepath.ToSlash(path), "\\", "/")
	for _, skip := range []string{".git", ".vscode", ".idea", "dist", "build", "tmp", "data", "node_modules"} {
		if buildinfo.PathHasComponent(path, skip) {
			return true
		}
	}
	return isCoverageOutputDir(root, path)
}

// isCoverageOutputDir matches the repository's scoped coverage report roots.
// A package directory named coverage is source unless its full path is one of
// these output locations; matching the basename alone drops real Go inputs.
func isCoverageOutputDir(root, path string) bool {
	return buildinfo.IsCoverageOutputDir(root, path)
}

func buildinfoSkipFile(path string, extra []string) bool {
	path = strings.ReplaceAll(filepath.ToSlash(path), "\\", "/")
	if isBuildOutput(path) {
		return true
	}
	for _, skip := range append([]string{"build.meta"}, extra...) {
		if buildinfo.PathHasComponent(path, skip) {
			return true
		}
	}
	return false
}
