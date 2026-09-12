//go:build windows

package cliutil

import (
	"os"
	"path/filepath"
	"strings"
)

// defaultPathExt mirrors the Windows default when PATHEXT is unset or empty.
const defaultPathExt = ".COM;.EXE;.BAT;.CMD"

// pathExtensions returns the executable suffixes to try, lowercased and
// normalised to a leading dot.
func pathExtensions() []string {
	raw := strings.TrimSpace(os.Getenv("PATHEXT"))
	if raw == "" {
		raw = defaultPathExt
	}
	extensions := make([]string, 0, 8)
	for _, ext := range strings.Split(raw, ";") {
		ext = strings.ToLower(strings.TrimSpace(ext))
		if ext == "" {
			continue
		}
		if !strings.HasPrefix(ext, ".") {
			ext = "." + ext
		}
		extensions = append(extensions, ext)
	}
	return extensions
}

// executableCandidates expands a bare path into the PATHEXT variants Windows
// would consider, keeping an already-suffixed path first so an explicit
// `codex.exe` is honoured before `codex.com`.
func executableCandidates(path string) []string {
	candidates := make([]string, 0, 8)
	if ext := strings.ToLower(filepath.Ext(path)); ext != "" && isExecutableExtension(ext) {
		candidates = append(candidates, path)
	}
	for _, ext := range pathExtensions() {
		candidates = append(candidates, path+ext)
	}
	return candidates
}

// isExecutableExtension reports whether ext is one Windows treats as directly
// runnable, so [ShimAliasFromArgv0] can strip it before matching an alias.
func isExecutableExtension(ext string) bool {
	ext = strings.ToLower(strings.TrimSpace(ext))
	for _, candidate := range pathExtensions() {
		if candidate == ext {
			return true
		}
	}
	return false
}

// isExecutableFile reports whether path is a regular file. Windows carries no
// execute bit; runnability is decided by the extension, which
// [executableCandidates] has already applied.
func isExecutableFile(path string) bool {
	info, err := os.Stat(path)
	if err != nil || info.IsDir() || !info.Mode().IsRegular() {
		return false
	}
	return true
}
