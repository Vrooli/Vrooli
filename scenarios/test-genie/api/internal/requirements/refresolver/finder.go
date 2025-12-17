package refresolver

import (
	"os"
	"path/filepath"
	"strings"
)

// Finder searches for files within a scenario directory.
type Finder struct {
	scenarioDir string
	reader      Reader
	// cache stores discovered files by basename for faster lookups
	cache map[string][]string
}

// NewFinder creates a new Finder for the given scenario directory.
func NewFinder(scenarioDir string) *Finder {
	return &Finder{
		scenarioDir: scenarioDir,
		reader:      OSReader{},
		cache:       make(map[string][]string),
	}
}

// NewFinderWithReader creates a Finder with a custom Reader (for testing).
func NewFinderWithReader(scenarioDir string, reader Reader) *Finder {
	return &Finder{
		scenarioDir: scenarioDir,
		reader:      reader,
		cache:       make(map[string][]string),
	}
}

// AbsolutePath returns the absolute path for a relative ref path.
func (f *Finder) AbsolutePath(relPath string) string {
	return filepath.Join(f.scenarioDir, filepath.FromSlash(relPath))
}

// FileExists checks if a file exists at the given relative path.
func (f *Finder) FileExists(relPath string) bool {
	absPath := f.AbsolutePath(relPath)
	info, err := f.reader.Stat(absPath)
	if err != nil {
		return false
	}
	return !info.IsDir()
}

// FindByBasename searches the scenario directory for files matching the given basename.
// Results are returned as relative paths from the scenario root.
// Search is limited to common source directories for performance.
func (f *Finder) FindByBasename(basename string) []string {
	// Check cache first
	if cached, ok := f.cache[basename]; ok {
		return cached
	}

	var matches []string

	// Common source directories to search (for performance)
	searchDirs := []string{
		"api",
		"cli",
		"ui/src",
		"lib",
		"src",
		"internal",
		"pkg",
		"test",
		"tests",
		"bas",
		"__tests__",
	}

	// Also check root level
	rootPath := filepath.Join(f.scenarioDir, basename)
	if info, err := f.reader.Stat(rootPath); err == nil && !info.IsDir() {
		matches = append(matches, basename)
	}

	// Search common directories
	for _, dir := range searchDirs {
		dirPath := filepath.Join(f.scenarioDir, dir)
		if info, err := f.reader.Stat(dirPath); err != nil || !info.IsDir() {
			continue
		}
		f.walkDir(dirPath, basename, &matches)
	}

	f.cache[basename] = matches
	return matches
}

// walkDir recursively searches for files matching basename.
// Limits depth to avoid searching too deeply.
func (f *Finder) walkDir(dir, basename string, matches *[]string) {
	// Limit search to avoid performance issues
	maxDepth := 10
	f.walkDirWithDepth(dir, basename, matches, 0, maxDepth)
}

func (f *Finder) walkDirWithDepth(dir, basename string, matches *[]string, depth, maxDepth int) {
	if depth > maxDepth {
		return
	}

	// Skip common non-source directories
	dirBase := filepath.Base(dir)
	skipDirs := map[string]bool{
		"node_modules": true,
		".git":         true,
		"vendor":       true,
		"dist":         true,
		"build":        true,
		".cache":       true,
		"coverage":     true,
		"__pycache__":  true,
	}
	if skipDirs[dirBase] {
		return
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}

	for _, entry := range entries {
		path := filepath.Join(dir, entry.Name())
		if entry.IsDir() {
			f.walkDirWithDepth(path, basename, matches, depth+1, maxDepth)
		} else if entry.Name() == basename {
			// Convert to relative path
			relPath, err := filepath.Rel(f.scenarioDir, path)
			if err == nil {
				*matches = append(*matches, filepath.ToSlash(relPath))
			}
		}
	}
}

// BestMatch finds the match most similar to the expected path.
// Uses path component similarity scoring.
func (f *Finder) BestMatch(expectedPath string, matches []string) string {
	if len(matches) == 0 {
		return ""
	}
	if len(matches) == 1 {
		return matches[0]
	}

	expectedParts := strings.Split(filepath.ToSlash(expectedPath), "/")
	var bestMatch string
	bestScore := -1

	for _, match := range matches {
		matchParts := strings.Split(match, "/")
		score := pathSimilarity(expectedParts, matchParts)
		if score > bestScore {
			bestScore = score
			bestMatch = match
		}
	}

	return bestMatch
}

// pathSimilarity scores how similar two paths are based on common components.
func pathSimilarity(expected, actual []string) int {
	score := 0

	// Reward matching directory components
	expectedDirs := expected[:len(expected)-1] // Exclude filename
	actualDirs := actual[:len(actual)-1]

	// Check for common directory names
	expectedSet := make(map[string]bool)
	for _, dir := range expectedDirs {
		expectedSet[dir] = true
	}
	for _, dir := range actualDirs {
		if expectedSet[dir] {
			score += 10
		}
	}

	// Bonus for matching depth
	if len(expectedDirs) == len(actualDirs) {
		score += 5
	}

	// Bonus for matching path suffix (e.g., "internal/handlers/handlers_test.go")
	minLen := len(expected)
	if len(actual) < minLen {
		minLen = len(actual)
	}
	for i := 1; i <= minLen; i++ {
		if expected[len(expected)-i] == actual[len(actual)-i] {
			score += 15 // Path suffix matches are very valuable
		} else {
			break
		}
	}

	return score
}

// ClearCache clears the file search cache.
func (f *Finder) ClearCache() {
	f.cache = make(map[string][]string)
}
