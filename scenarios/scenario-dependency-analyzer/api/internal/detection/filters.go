package detection

import (
	"io/fs"
	"path/filepath"
	"strings"
)

// filters.go - Path and directory filtering logic
//
// This file centralizes all logic for determining which files and directories
// should be included or excluded during dependency scanning.

// Directory filtering

// shouldSkipDirectoryEntry returns true if a directory should be skipped during scanning
func shouldSkipDirectoryEntry(entry fs.DirEntry) bool {
	if !entry.IsDir() {
		return false
	}
	name := entry.Name()
	lower := strings.ToLower(name)

	// Check against known skip list
	if _, ok := skipDirectoryNames[lower]; ok {
		return true
	}

	// Skip node_modules and .ignored variants
	if strings.HasPrefix(lower, "node_modules") {
		return true
	}
	if strings.HasPrefix(lower, ".ignored") {
		return true
	}

	// Skip hidden directories except .vrooli
	if strings.HasPrefix(name, ".") && name != ".vrooli" {
		return true
	}

	return false
}

// skipDirectoryNames lists directories that should always be excluded from scanning
var skipDirectoryNames = map[string]struct{}{
	"node_modules":     {},
	"dist":             {},
	"build":            {},
	"coverage":         {},
	"logs":             {},
	"tmp":              {},
	"temp":             {},
	"vendor":           {},
	"__pycache__":      {},
	".pytest_cache":    {},
	".nyc_output":      {},
	"storybook-static": {},
	".next":            {},
	".nuxt":            {},
	".svelte-kit":      {},
	".vercel":          {},
	".parcel-cache":    {},
	".turbo":           {},
	".git":             {},
	".hg":              {},
	".svn":             {},
	".idea":            {},
	".vscode":          {},
	".cache":           {},
	".output":          {},
	".yalc":            {},
	".yarn":            {},
	".pnpm":            {},
}

// File filtering

// shouldIgnoreDetectionFile returns true if a file should be ignored during dependency detection
func shouldIgnoreDetectionFile(relPath string) bool {
	if relPath == "" {
		return false
	}

	lower := strings.ToLower(relPath)
	base := strings.ToLower(filepath.Base(lower))

	// Check if filename is in the doc files list
	if _, ok := docFileNames[base]; ok {
		return true
	}

	// Check if extension is a documentation extension
	if ext := strings.ToLower(filepath.Ext(base)); ext != "" {
		if _, ok := docExtensions[ext]; ok {
			return true
		}
	}

	// Skip README files
	if strings.HasPrefix(base, "readme") {
		return true
	}

	// Check if any path segment should be ignored
	segments := strings.Split(lower, string(filepath.Separator))
	for _, segment := range segments {
		if _, ok := analysisIgnoreSegments[segment]; ok {
			return true
		}
	}

	return false
}

// analysisIgnoreSegments lists path segments that should be excluded from analysis
var analysisIgnoreSegments = map[string]struct{}{
	"docs":          {},
	"doc":           {},
	"documentation": {},
	"readme":        {},
	"test":          {},
	"tests":         {},
	"testdata":      {},
	"__tests__":     {},
	"spec":          {},
	"specs":         {},
	"coverage":      {},
	"examples":      {},
	"playbooks":     {},
	"data":          {},
	"draft":         {},
	"drafts":        {},
	"prd-drafts":    {},
	"dist":          {},
	"build":         {},
	"out":           {},
	"outputs":       {},
}

// docExtensions lists file extensions that are considered documentation
var docExtensions = map[string]struct{}{
	".md":  {},
	".mdx": {},
	".rst": {},
	".txt": {},
}

// docFileNames lists specific filenames that should be ignored as documentation
var docFileNames = map[string]struct{}{
	"readme":           {},
	"readme.md":        {},
	"readme.mdx":       {},
	"prd.md":           {},
	"prd.mdx":          {},
	"problems.md":      {},
	"requirements.md":  {},
	"requirements.mdx": {},
}

// Resource CLI path filtering

// isAllowedResourceCLIPath returns true if resource CLI commands found in this path should be counted
// This prevents false positives from documentation, tests, etc.
func isAllowedResourceCLIPath(relPath string) bool {
	if relPath == "" {
		return true
	}

	segments := strings.Split(relPath, string(filepath.Separator))
	if len(segments) == 0 {
		return true
	}

	root := strings.ToLower(strings.TrimSpace(segments[0]))
	if root == "." {
		root = ""
	}

	_, ok := resourceCLIDirectoryAllowList[root]
	return ok
}

// resourceCLIDirectoryAllowList defines which top-level directories are allowed to contain
// resource CLI commands that should be detected as dependencies
var resourceCLIDirectoryAllowList = map[string]struct{}{
	"":               {}, // Root level
	"api":            {},
	"cli":            {},
	"cmd":            {},
	"scripts":        {},
	"script":         {},
	"ui":             {},
	"src":            {},
	"server":         {},
	"services":       {},
	"service":        {},
	"lib":            {},
	"pkg":            {},
	"internal":       {},
	"tools":          {},
	"initialization": {},
	"automation":     {},
	"test":           {},
	"tests":          {},
	"integration":    {},
	"config":         {},
}
