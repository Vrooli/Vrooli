package existence

import (
	"io"
	"path/filepath"
	"strings"
)

// StandardDirs defines directories required for a well-formed scenario.
var StandardDirs = []string{
	"api",
	"cli",
	"docs",
	"requirements",
	"test",
	"ui",
}

// StandardFiles defines files required for a well-formed scenario.
var StandardFiles = []string{
	filepath.Join("api", "main.go"),
	filepath.Join("cli", "install.sh"),
	filepath.Join(".vrooli", "service.json"),
	filepath.Join(".vrooli", "testing.json"),
	"README.md",
	"PRD.md",
	"Makefile",
}

// Validator validates that required filesystem entries exist.
type Validator interface {
	// ValidateDirs checks that all required directories exist.
	ValidateDirs(dirs []string) Result

	// ValidateFiles checks that all required files exist.
	ValidateFiles(files []string) Result
}

// validator is the default implementation of Validator.
type validator struct {
	scenarioDir string
	logWriter   io.Writer
}

// New creates a new existence validator for the given scenario directory.
func New(scenarioDir string, logWriter io.Writer) Validator {
	return &validator{
		scenarioDir: scenarioDir,
		logWriter:   logWriter,
	}
}

// ValidateDirs implements Validator.
func (v *validator) ValidateDirs(dirs []string) Result {
	return ValidateDirs(v.scenarioDir, dirs, v.logWriter)
}

// ValidateFiles implements Validator.
func (v *validator) ValidateFiles(files []string) Result {
	return ValidateFiles(v.scenarioDir, files, v.logWriter)
}

// ResolveRequirementsWithOverrides combines standard requirements with additional/excluded paths.
// This is a pure function that doesn't depend on the Expectations type.
func ResolveRequirementsWithOverrides(additionalDirs, additionalFiles, excludedDirs, excludedFiles []string) (dirs []string, files []string) {
	dirs = append([]string{}, StandardDirs...)
	files = append([]string{}, StandardFiles...)

	dirs = append(dirs, additionalDirs...)
	files = append(files, additionalFiles...)
	dirs = filterPaths(dirs, excludedDirs)
	files = filterPaths(files, excludedFiles)

	dirs = deduplicatePaths(dirs)
	files = deduplicatePaths(files)
	return dirs, files
}

// filterPaths removes excluded paths from the input slice.
func filterPaths(paths, excludes []string) []string {
	if len(excludes) == 0 {
		return paths
	}
	excludeSet := make(map[string]struct{}, len(excludes))
	for _, path := range excludes {
		excludeSet[canonicalizePath(path)] = struct{}{}
	}
	var filtered []string
	for _, path := range paths {
		clean := canonicalizePath(path)
		if clean == "" {
			continue
		}
		if _, skip := excludeSet[clean]; skip {
			continue
		}
		filtered = append(filtered, clean)
	}
	return filtered
}

// deduplicatePaths removes duplicate paths from the input slice.
func deduplicatePaths(paths []string) []string {
	seen := make(map[string]struct{}, len(paths))
	var result []string
	for _, path := range paths {
		clean := canonicalizePath(path)
		if clean == "" {
			continue
		}
		if _, exists := seen[clean]; exists {
			continue
		}
		seen[clean] = struct{}{}
		result = append(result, clean)
	}
	return result
}

// canonicalizePath normalizes a path for consistent comparison.
func canonicalizePath(path string) string {
	clean := filepath.Clean(path)
	clean = strings.TrimPrefix(clean, "./")
	if clean == "." {
		return ""
	}
	return filepath.ToSlash(clean)
}

// resolvePath resolves a relative path against a base directory.
func resolvePath(baseDir, rel string) string {
	if rel == "" {
		return baseDir
	}
	osSpecific := filepath.FromSlash(rel)
	if filepath.IsAbs(osSpecific) {
		return osSpecific
	}
	return filepath.Join(baseDir, osSpecific)
}
