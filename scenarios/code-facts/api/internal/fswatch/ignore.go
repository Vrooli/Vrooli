package fswatch

import (
	"path/filepath"
	"strings"
)

// ignoredDirNames are directory basenames that can never hold a code fact and
// are write-hot enough that watching them turns the watcher into a busy loop:
// dependency stores, build output, caches, and scratch space. The list mirrors
// the transient segments catalog.Classify rejects plus the package manager
// stores that live beside node_modules.
var ignoredDirNames = map[string]struct{}{
	".git":         {},
	"node_modules": {},
	".pnpm":        {},
	".pnpm-store":  {},
	"vendor":       {},
	"dist":         {},
	"build":        {},
	"coverage":     {},
	"tmp":          {},
	"temp":         {},
	"logs":         {},
	".cache":       {},
	".next":        {},
	".turbo":       {},
	".mypy_cache":  {},
	"__pycache__":  {},
	".venv":        {},
	"venv":         {},
}

// ignoredFileSuffixes are file basename suffixes for live databases, journals
// and logs. Other daemons write these continuously; none of them is source.
var ignoredFileSuffixes = []string{
	".db", ".db-wal", ".db-shm", ".db-journal",
	".sqlite", ".sqlite3", ".sqlite-wal", ".sqlite-shm", ".sqlite-journal",
	".log",
}

// Ignored reports whether a repository-relative path (any separator, e.g.
// "scenarios/x/data/state.db-wal") is outside the code-facts corpus by
// construction and must be neither watched nor allowed to wake a refresh.
// Callers must pass paths relative to the repository, never absolute ones:
// the rule looks at every segment, and a checkout living under /tmp or a
// home directory named build must not be ignored wholesale. The rule is
// shared by the watch-tree walk and the event loop so an ignored directory
// can never be half-watched.
//
// A scenario-level or resource-level data directory (scenarios/<slug>/data,
// resources/<slug>/data) is the durable state home of a running service and
// holds SQLite databases with live WAL traffic; it is ignored as a whole.
func Ignored(path string) bool {
	normalized := filepath.ToSlash(filepath.Clean(path))
	segments := strings.Split(strings.Trim(normalized, "/"), "/")
	for i, segment := range segments {
		if _, ok := ignoredDirNames[segment]; ok {
			return true
		}
		if segment == "data" && i >= 2 && (segments[i-2] == "scenarios" || segments[i-2] == "resources") {
			return true
		}
		if strings.HasPrefix(segment, ".proto-gen-stage-") || strings.HasPrefix(segment, ".verify-") || strings.HasPrefix(segment, ".tmp-") {
			return true
		}
	}
	base := strings.ToLower(segments[len(segments)-1])
	for _, suffix := range ignoredFileSuffixes {
		if strings.HasSuffix(base, suffix) {
			return true
		}
	}
	if strings.Contains(base, ".sqlite") {
		return true
	}
	return false
}
