package pathhygiene

import (
	"os"
	"path/filepath"
	"runtime"
	"sort"
)

// DuplicateEntry is one PATH directory that appears more than once.
type DuplicateEntry struct {
	Dir   string
	Count int
}

// DuplicateEntries reports every PATH directory listed more than once,
// ordered worst-first. Reported for the whole PATH, not just the Vrooli
// entry: duplicates from any source levy the same per-lookup scan cost, and
// an operator seeing "17 unique of 236" learns more than one that only
// counts its own.
func DuplicateEntries(pathEnv string) []DuplicateEntry {
	counts := map[string]int{}
	for _, dir := range filepath.SplitList(pathEnv) {
		if dir == "" {
			continue
		}
		counts[filepath.Clean(dir)]++
	}
	dups := make([]DuplicateEntry, 0)
	for dir, n := range counts {
		if n > 1 {
			dups = append(dups, DuplicateEntry{Dir: dir, Count: n})
		}
	}
	sort.Slice(dups, func(i, j int) bool {
		if dups[i].Count != dups[j].Count {
			return dups[i].Count > dups[j].Count
		}
		return dups[i].Dir < dups[j].Dir
	})
	return dups
}

// UniqueEntryCount reports how many distinct directories a PATH holds.
func UniqueEntryCount(pathEnv string) int {
	seen := map[string]struct{}{}
	for _, dir := range filepath.SplitList(pathEnv) {
		if dir != "" {
			seen[filepath.Clean(dir)] = struct{}{}
		}
	}
	return len(seen)
}

// EntryCount reports the total number of PATH entries.
func EntryCount(pathEnv string) int {
	n := 0
	for _, dir := range filepath.SplitList(pathEnv) {
		if dir != "" {
			n++
		}
	}
	return n
}

// isExecutableFileFn is a test seam so shadow detection can be exercised
// without laying down real executables.
var isExecutableFileFn = isExecutableFile

func isExecutableFile(path string) bool {
	info, err := os.Stat(path)
	if err != nil || info.IsDir() {
		return false
	}
	if runtime.GOOS == "windows" {
		return true
	}
	return info.Mode().Perm()&0o111 != 0
}

// ShadowingBinaries returns directories that hold a binary named `name` and
// sit ahead of canonicalDir on PATH — that is, copies that would win a bare
// invocation.
//
// These are reported, never removed. A second `vrooli` in ~/go/bin is
// somebody's `go install` output; deleting another tool's artifact is the
// operator's call, not a safeguard's. Naming it is what turns a silent
// wrong-binary failure into a decision.
func ShadowingBinaries(pathEnv, canonicalDir, name string) []string {
	canonical := filepath.Clean(canonicalDir)
	shadows := make([]string, 0)
	for _, dir := range filepath.SplitList(pathEnv) {
		if dir == "" {
			continue
		}
		clean := filepath.Clean(dir)
		if clean == canonical {
			// Everything from here on is behind the canonical entry.
			return shadows
		}
		candidate := filepath.Join(clean, name)
		if runtime.GOOS == "windows" {
			candidate += ".exe"
		}
		if isExecutableFileFn(candidate) {
			shadows = append(shadows, candidate)
		}
	}
	return shadows
}
