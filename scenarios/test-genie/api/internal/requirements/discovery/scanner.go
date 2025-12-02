package discovery

import (
	"context"
	"io/fs"
	"path/filepath"
	"strings"
)

// Scanner performs directory scanning for requirement files.
type Scanner struct {
	reader Reader
}

// NewScanner creates a Scanner with the provided Reader.
func NewScanner(reader Reader) *Scanner {
	return &Scanner{reader: reader}
}

// ScanRecursive finds all JSON files in the directory tree.
func (s *Scanner) ScanRecursive(ctx context.Context, rootDir string) ([]DiscoveredFile, error) {
	var files []DiscoveredFile

	err := s.walkDir(ctx, rootDir, "", &files)
	if err != nil {
		return files, err
	}

	return files, nil
}

// walkDir recursively walks a directory tree.
func (s *Scanner) walkDir(ctx context.Context, baseDir, relDir string, files *[]DiscoveredFile) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	currentDir := baseDir
	if relDir != "" {
		currentDir = filepath.Join(baseDir, relDir)
	}

	entries, err := s.reader.ReadDir(currentDir)
	if err != nil {
		return &DiscoveryError{Path: currentDir, Err: err}
	}

	for _, entry := range entries {
		name := entry.Name()

		// Skip hidden files and directories
		if strings.HasPrefix(name, ".") {
			continue
		}

		// Skip common non-requirement directories
		if entry.IsDir() && isSkippableDir(name) {
			continue
		}

		childRel := name
		if relDir != "" {
			childRel = filepath.Join(relDir, name)
		}

		if entry.IsDir() {
			if err := s.walkDir(ctx, baseDir, childRel, files); err != nil {
				// Continue on error - collect as many files as possible
				continue
			}
		} else if isRequirementFile(name) {
			absPath := filepath.Join(currentDir, name)
			*files = append(*files, DiscoveredFile{
				AbsolutePath: absPath,
				RelativePath: childRel,
				IsIndex:      isIndexFile(name),
				ModuleDir:    extractModuleDir(relDir),
			})
		}
	}

	return nil
}

// ScanImmediate finds requirement files only in the immediate directory.
func (s *Scanner) ScanImmediate(ctx context.Context, dir string) ([]DiscoveredFile, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	entries, err := s.reader.ReadDir(dir)
	if err != nil {
		return nil, &DiscoveryError{Path: dir, Err: err}
	}

	var files []DiscoveredFile
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		if !isRequirementFile(name) {
			continue
		}

		absPath := filepath.Join(dir, name)
		files = append(files, DiscoveredFile{
			AbsolutePath: absPath,
			RelativePath: name,
			IsIndex:      isIndexFile(name),
			ModuleDir:    "",
		})
	}

	return files, nil
}

// FindModules finds all module.json files in subdirectories.
func (s *Scanner) FindModules(ctx context.Context, requirementsDir string) ([]DiscoveredFile, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	entries, err := s.reader.ReadDir(requirementsDir)
	if err != nil {
		return nil, &DiscoveryError{Path: requirementsDir, Err: err}
	}

	var modules []DiscoveredFile
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		name := entry.Name()
		if strings.HasPrefix(name, ".") || isSkippableDir(name) {
			continue
		}

		moduleJSON := filepath.Join(requirementsDir, name, "module.json")
		if s.reader.Exists(moduleJSON) {
			modules = append(modules, DiscoveredFile{
				AbsolutePath: moduleJSON,
				RelativePath: filepath.Join(name, "module.json"),
				IsIndex:      false,
				ModuleDir:    name,
			})
		}
	}

	return modules, nil
}

// FileExists checks if a file exists.
func (s *Scanner) FileExists(path string) bool {
	return s.reader.Exists(path)
}

// GetFileInfo returns file info if the path exists.
func (s *Scanner) GetFileInfo(path string) (fs.FileInfo, error) {
	return s.reader.Stat(path)
}

// isRequirementFile checks if a filename is a requirement JSON file.
func isRequirementFile(name string) bool {
	if !strings.HasSuffix(name, ".json") {
		return false
	}

	// Accept index.json, module.json, and other .json files
	// Exclude known non-requirement JSON files
	lower := strings.ToLower(name)
	excluded := []string{
		"package.json",
		"tsconfig.json",
		"jsconfig.json",
		"package-lock.json",
		"composer.json",
	}

	for _, exc := range excluded {
		if lower == exc {
			return false
		}
	}

	return true
}

// isIndexFile checks if a filename is an index file.
func isIndexFile(name string) bool {
	return strings.ToLower(name) == "index.json"
}

// isSkippableDir checks if a directory should be skipped during scanning.
func isSkippableDir(name string) bool {
	skippable := []string{
		"node_modules",
		"vendor",
		"dist",
		"build",
		"coverage",
		"__pycache__",
		".git",
		".svn",
		".hg",
		"target",
		"bin",
		"obj",
	}

	lower := strings.ToLower(name)
	for _, s := range skippable {
		if lower == s {
			return true
		}
	}

	return false
}

// extractModuleDir extracts the module directory name from a relative path.
func extractModuleDir(relDir string) string {
	if relDir == "" {
		return ""
	}
	// Get the first path component
	parts := strings.Split(filepath.ToSlash(relDir), "/")
	if len(parts) > 0 {
		return parts[0]
	}
	return relDir
}

// SortByPriority sorts discovered files by module priority.
// Files in directories starting with digits (01-, 02-) are sorted numerically.
func SortByPriority(files []DiscoveredFile) {
	// Use a simple bubble sort for stability (small lists)
	n := len(files)
	for i := 0; i < n-1; i++ {
		for j := 0; j < n-i-1; j++ {
			if comparePriority(files[j], files[j+1]) > 0 {
				files[j], files[j+1] = files[j+1], files[j]
			}
		}
	}
}

// comparePriority compares two files by their priority.
// Returns <0 if a comes before b, >0 if a comes after b, 0 if equal.
func comparePriority(a, b DiscoveredFile) int {
	// Index files come first
	if a.IsIndex && !b.IsIndex {
		return -1
	}
	if !a.IsIndex && b.IsIndex {
		return 1
	}

	// Compare module directories
	aPriority := extractPriorityPrefix(a.ModuleDir)
	bPriority := extractPriorityPrefix(b.ModuleDir)

	if aPriority != bPriority {
		return aPriority - bPriority
	}

	// Fall back to alphabetical
	return strings.Compare(a.RelativePath, b.RelativePath)
}

// extractPriorityPrefix extracts a numeric prefix from a directory name.
// Returns 0 for non-numeric prefixes, or the parsed number.
func extractPriorityPrefix(name string) int {
	if name == "" {
		return 0
	}

	// Look for pattern like "01-" or "02-"
	dashIdx := strings.Index(name, "-")
	if dashIdx <= 0 {
		return 1000 // Non-prefixed dirs come last
	}

	prefix := name[:dashIdx]
	var num int
	for _, c := range prefix {
		if c < '0' || c > '9' {
			return 1000
		}
		num = num*10 + int(c-'0')
	}

	return num
}
