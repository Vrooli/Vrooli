//go:build !windows

package cliutil

import "os"

// executableCandidates is the identity on Unix: an executable is named exactly
// what it is called, with no implicit extension.
func executableCandidates(path string) []string { return []string{path} }

// isExecutableExtension is always false on Unix, so a shim installed as
// `some.name` keeps its full basename when the alias is read back.
func isExecutableExtension(string) bool { return false }

// isExecutableFile reports whether path is a regular file carrying an execute
// bit. Directories and non-regular files are rejected so a directory named like
// the agent cannot shadow it.
func isExecutableFile(path string) bool {
	info, err := os.Stat(path)
	if err != nil || info.IsDir() || !info.Mode().IsRegular() {
		return false
	}
	return info.Mode().Perm()&0o111 != 0
}
