// Package resolver handles resolution of import paths to actual file paths
package resolver

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
)

// GoModResolver resolves Go module imports to file paths
type GoModResolver struct {
	modules   map[string]string // module path -> local directory (relative to repoRoot)
	repoRoot  string
	initOnce  bool
	initError error
}

// NewGoModResolver creates a new Go module resolver
func NewGoModResolver() *GoModResolver {
	return &GoModResolver{
		modules: make(map[string]string),
	}
}

// Init initializes the resolver by scanning for go.mod and go.work files
func (r *GoModResolver) Init(repoRoot string) error {
	if r.initOnce && r.repoRoot == repoRoot {
		return r.initError
	}
	r.initOnce = true
	r.repoRoot = repoRoot
	r.modules = make(map[string]string)

	// First check for go.work (workspace mode)
	workPath := filepath.Join(repoRoot, "go.work")
	if _, err := os.Stat(workPath); err == nil {
		if err := r.parseGoWork(workPath, repoRoot); err != nil {
			// Continue even if go.work parsing fails
			r.initError = err
		}
	}

	// Also scan for all go.mod files in the repo
	if err := r.scanGoModFiles(repoRoot); err != nil {
		r.initError = err
	}

	return r.initError
}

// parseGoWorkLine processes a single line from a use block or single-line use directive.
func (r *GoModResolver) parseGoWorkLine(line, repoRoot string) {
	modDir := strings.Trim(line, "\t \"")
	if modDir != "" && !strings.HasPrefix(modDir, "//") {
		r.addModuleFromDir(repoRoot, modDir)
	}
}

// parseGoWork parses a go.work file to find workspace modules
func (r *GoModResolver) parseGoWork(workPath, repoRoot string) error {
	file, err := os.Open(workPath)
	if err != nil {
		return err
	}
	defer file.Close()

	sc := bufio.NewScanner(file)
	inUseBlock := false

	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())

		if strings.HasPrefix(line, "use (") {
			inUseBlock = true
			continue
		}
		if inUseBlock && line == ")" {
			inUseBlock = false
			continue
		}
		if inUseBlock {
			r.parseGoWorkLine(line, repoRoot)
			continue
		}

		if strings.HasPrefix(line, "use ") {
			modDir := strings.TrimSpace(strings.TrimPrefix(line, "use "))
			modDir = strings.Trim(modDir, "\"")
			if modDir != "" {
				r.addModuleFromDir(repoRoot, modDir)
			}
		}
	}

	return sc.Err()
}

// scanGoModFiles walks the directory tree to find go.mod files
func (r *GoModResolver) scanGoModFiles(repoRoot string) error {
	return filepath.Walk(repoRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip errors
		}

		// Skip hidden directories and common non-source directories
		if info.IsDir() {
			base := filepath.Base(path)
			if strings.HasPrefix(base, ".") || base == "node_modules" || base == "vendor" {
				return filepath.SkipDir
			}
			return nil
		}

		if info.Name() == "go.mod" {
			modDir, _ := filepath.Rel(repoRoot, filepath.Dir(path))
			if modDir == "" {
				modDir = "."
			}
			r.addModuleFromGoMod(path, modDir)
		}

		return nil
	})
}

// addModuleFromDir reads the go.mod in a directory and adds the module
func (r *GoModResolver) addModuleFromDir(repoRoot, modDir string) {
	goModPath := filepath.Join(repoRoot, modDir, "go.mod")
	r.addModuleFromGoMod(goModPath, modDir)
}

// addModuleFromGoMod reads a go.mod file and extracts the module path
func (r *GoModResolver) addModuleFromGoMod(goModPath, modDir string) {
	file, err := os.Open(goModPath)
	if err != nil {
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "module ") {
			modulePath := strings.TrimSpace(strings.TrimPrefix(line, "module "))
			modulePath = strings.Trim(modulePath, "\"")
			if modulePath != "" {
				// Normalize the module directory
				if modDir == "." {
					modDir = ""
				}
				r.modules[modulePath] = modDir
			}
			break
		}
	}
}

// isStdLibImport returns true if the import source looks like a Go standard library import.
func isStdLibImport(importSource string) bool {
	firstSlash := strings.Index(importSource, "/")
	var firstSegment string
	if firstSlash == -1 {
		firstSegment = importSource
	} else {
		firstSegment = importSource[:firstSlash]
	}
	return !strings.Contains(firstSegment, ".")
}

// findLongestModuleMatch finds the longest matching module prefix for an import.
func (r *GoModResolver) findLongestModuleMatch(importSource string) (modulePath, moduleDir string, found bool) {
	for mp, md := range r.modules {
		if importSource == mp || strings.HasPrefix(importSource, mp+"/") {
			if len(mp) > len(modulePath) {
				modulePath = mp
				moduleDir = md
				found = true
			}
		}
	}
	return
}

// buildLocalPath computes the local filesystem path for an import within a matched module.
func buildLocalPath(importSource, longestMatch, matchedDir string) string {
	subPath := strings.TrimPrefix(importSource, longestMatch)
	subPath = strings.TrimPrefix(subPath, "/")

	if matchedDir == "" {
		return subPath
	}
	return filepath.Join(matchedDir, subPath)
}

// resolveGoPackageDir tries to find a non-test .go file in a package directory.
func resolveGoPackageDir(absLocalPath, localPath string) string {
	entries, err := os.ReadDir(absLocalPath)
	if err != nil {
		return ""
	}
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".go") &&
			!strings.HasSuffix(entry.Name(), "_test.go") {
			return filepath.Join(localPath, entry.Name())
		}
	}
	return ""
}

// resolveLocalFile tries various strategies to resolve a local path to an actual file.
func resolveLocalFile(localPath, repoRoot string) string {
	absLocalPath := filepath.Join(repoRoot, localPath)

	// Check if it's a directory (package) - find a .go file
	if info, err := os.Stat(absLocalPath); err == nil && info.IsDir() {
		if result := resolveGoPackageDir(absLocalPath, localPath); result != "" {
			return result
		}
	}

	// Check if adding .go extension works
	goPath := localPath + ".go"
	if info, err := os.Stat(filepath.Join(repoRoot, goPath)); err == nil && !info.IsDir() {
		return goPath
	}

	// If local path itself exists as a directory, return the path
	if info, err := os.Stat(absLocalPath); err == nil && info.IsDir() {
		return localPath
	}

	return ""
}

// Resolve resolves a Go module import to a file path
func (r *GoModResolver) Resolve(importSource string, fromFile string, repoRoot string) string {
	_ = r.Init(repoRoot)

	if strings.HasPrefix(importSource, ".") {
		return ""
	}
	if isStdLibImport(importSource) {
		return ""
	}

	longestMatch, matchedDir, found := r.findLongestModuleMatch(importSource)
	if !found {
		return ""
	}

	localPath := buildLocalPath(importSource, longestMatch, matchedDir)
	return resolveLocalFile(localPath, repoRoot)
}

// GetModules returns the discovered modules (for testing/debugging)
func (r *GoModResolver) GetModules() map[string]string {
	result := make(map[string]string, len(r.modules))
	for k, v := range r.modules {
		result[k] = v
	}
	return result
}
