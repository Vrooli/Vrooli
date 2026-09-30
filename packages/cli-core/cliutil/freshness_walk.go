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

func buildinfoSkipDir(path string) bool {
	path = strings.ReplaceAll(filepath.ToSlash(path), "\\", "/")
	for _, skip := range []string{".git", ".vscode", ".idea", "coverage", "dist", "build", "tmp", "data", "node_modules"} {
		if buildinfo.PathHasComponent(path, skip) {
			return true
		}
	}
	return false
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
