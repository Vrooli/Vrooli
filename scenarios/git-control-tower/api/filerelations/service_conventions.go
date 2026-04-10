// Package filerelations provides file relationship discovery services
package filerelations

import (
	"context"
	"os"
	"path/filepath"
	"strings"
)

// isTestFileName returns true if the name (without extension) represents a test file.
func isTestFileName(nameWithoutExt string) bool {
	return strings.HasSuffix(nameWithoutExt, ".test") ||
		strings.HasSuffix(nameWithoutExt, ".spec") ||
		strings.HasSuffix(nameWithoutExt, "_test")
}

// buildTestPatterns builds test file name patterns for the given base name and extension.
func buildTestPatterns(nameWithoutExt, ext string) []string {
	patterns := []string{
		nameWithoutExt + ".test" + ext,
		nameWithoutExt + ".spec" + ext,
		nameWithoutExt + "_test" + ext,
		nameWithoutExt + "_test.go",
	}

	if isJSTSFile(ext) {
		for _, altExt := range jsTSExtensions() {
			if altExt != ext {
				patterns = append(patterns,
					nameWithoutExt+".test"+altExt,
					nameWithoutExt+".spec"+altExt,
					nameWithoutExt+"_test"+altExt,
				)
			}
		}
	}

	return patterns
}

// checkPatternsInDirs checks test patterns in the given directories and appends matches.
func (s *Service) checkPatternsInDirs(dirs []string, patterns []string, repoRoot string, found map[string]bool) []RelatedFile {
	var related []RelatedFile
	for _, dir := range dirs {
		for _, pattern := range patterns {
			testPath := filepath.Join(dir, pattern)
			if !found[testPath] && s.fileExists(repoRoot, testPath) {
				found[testPath] = true
				related = append(related, RelatedFile{Path: testPath, RelationType: RelationTest})
			}
		}
	}
	return related
}

// findTestFilesForSource finds test files for a source file.
func (s *Service) findTestFilesForSource(dir, nameWithoutExt, ext, repoRoot string) []RelatedFile {
	patterns := buildTestPatterns(nameWithoutExt, ext)
	found := make(map[string]bool)
	var related []RelatedFile

	// 1. Same directory
	related = append(related, s.checkPatternsInDirs([]string{dir}, patterns, repoRoot, found)...)

	// 2. __tests__ and __test__ subdirectories
	subDirs := []string{
		filepath.Join(dir, "__tests__"),
		filepath.Join(dir, "__test__"),
	}
	related = append(related, s.checkPatternsInDirs(subDirs, patterns, repoRoot, found)...)

	// 3. Top-level test directories mirroring source structure
	related = append(related, s.findTestsInTopLevelDirs(dir, patterns, repoRoot, found)...)

	return related
}

// findTestsInTopLevelDirs searches top-level test directories for test files.
func (s *Service) findTestsInTopLevelDirs(dir string, patterns []string, repoRoot string, found map[string]bool) []RelatedFile {
	topLevelDirs := []string{"tests", "test", "__tests__", "__test__"}
	var dirs []string

	for _, testDir := range topLevelDirs {
		dirs = append(dirs, filepath.Join(testDir, dir))

		// Also try with "src/" prefix stripped
		if strings.HasPrefix(dir, "src/") || strings.HasPrefix(dir, "src\\") {
			strippedDir := strings.TrimPrefix(strings.TrimPrefix(dir, "src/"), "src\\")
			dirs = append(dirs, filepath.Join(testDir, strippedDir))
		}
	}

	return s.checkPatternsInDirs(dirs, patterns, repoRoot, found)
}

// extractSourceBase extracts the source file base name from a test file name.
func extractSourceBase(nameWithoutExt string) string {
	switch {
	case strings.HasSuffix(nameWithoutExt, ".test"):
		return strings.TrimSuffix(nameWithoutExt, ".test")
	case strings.HasSuffix(nameWithoutExt, ".spec"):
		return strings.TrimSuffix(nameWithoutExt, ".spec")
	case strings.HasSuffix(nameWithoutExt, "_test"):
		return strings.TrimSuffix(nameWithoutExt, "_test")
	}
	return ""
}

// findSourceForTestFile finds the source file for a test file.
func (s *Service) findSourceForTestFile(dir, base, nameWithoutExt, ext, repoRoot string) []RelatedFile {
	var related []RelatedFile

	sourceBase := extractSourceBase(nameWithoutExt)
	if sourceBase != "" {
		related = append(related, s.findSourceInDir(dir, sourceBase, ext, repoRoot)...)
		related = append(related, s.findSourceInParentTestDir(dir, sourceBase, ext, repoRoot)...)
	}

	// Handle Go _test.go -> .go specifically
	if ext == ".go" && strings.HasSuffix(base, "_test.go") {
		goSourceBase := strings.TrimSuffix(base, "_test.go")
		goSourcePath := filepath.Join(dir, goSourceBase+".go")
		if s.fileExists(repoRoot, goSourcePath) {
			related = append(related, RelatedFile{Path: goSourcePath, RelationType: RelationTest})
		}
	}

	return related
}

// findSourceInDir finds source files in the same directory as the test file.
func (s *Service) findSourceInDir(dir, sourceBase, ext, repoRoot string) []RelatedFile {
	var related []RelatedFile

	sourcePath := filepath.Join(dir, sourceBase+ext)
	if s.fileExists(repoRoot, sourcePath) {
		related = append(related, RelatedFile{Path: sourcePath, RelationType: RelationTest})
	}

	if isJSTSFile(ext) {
		for _, altExt := range jsTSExtensions() {
			if altExt != ext {
				altPath := filepath.Join(dir, sourceBase+altExt)
				if s.fileExists(repoRoot, altPath) {
					related = append(related, RelatedFile{Path: altPath, RelationType: RelationTest})
				}
			}
		}
	}

	return related
}

// findSourceInParentTestDir finds source files in the parent of __tests__/__test__ dirs.
func (s *Service) findSourceInParentTestDir(dir, sourceBase, ext, repoRoot string) []RelatedFile {
	parentDir := filepath.Dir(dir)
	baseDirName := filepath.Base(dir)
	if baseDirName != "__tests__" && baseDirName != "__test__" {
		return nil
	}

	var related []RelatedFile

	parentSourcePath := filepath.Join(parentDir, sourceBase+ext)
	if s.fileExists(repoRoot, parentSourcePath) {
		related = append(related, RelatedFile{Path: parentSourcePath, RelationType: RelationTest})
	}

	if isJSTSFile(ext) {
		for _, altExt := range jsTSExtensions() {
			if altExt != ext {
				altPath := filepath.Join(parentDir, sourceBase+altExt)
				if s.fileExists(repoRoot, altPath) {
					related = append(related, RelatedFile{Path: altPath, RelationType: RelationTest})
				}
			}
		}
	}

	return related
}

// findIndexFiles finds index files in the same directory.
func (s *Service) findIndexFiles(dir, base, repoRoot string) []RelatedFile {
	if strings.HasPrefix(base, "index.") || base == "__init__.py" {
		return nil
	}

	indexPatterns := []string{
		"index.ts", "index.tsx", "index.js", "index.jsx",
		"index.go", "__init__.py", "mod.rs",
	}

	var related []RelatedFile
	for _, pattern := range indexPatterns {
		indexPath := filepath.Join(dir, pattern)
		if s.fileExists(repoRoot, indexPath) {
			related = append(related, RelatedFile{Path: indexPath, RelationType: RelationIndex})
		}
	}
	return related
}

// findTypeFiles finds type definition files for TypeScript/JavaScript files.
func (s *Service) findTypeFiles(dir, nameWithoutExt, ext, filePath, repoRoot string) []RelatedFile {
	if !isJSTSFile(ext) {
		return nil
	}

	typePatterns := []string{
		nameWithoutExt + ".types.ts",
		nameWithoutExt + ".d.ts",
		"types.ts",
		"types.d.ts",
	}

	var related []RelatedFile
	for _, pattern := range typePatterns {
		typePath := filepath.Join(dir, pattern)
		if s.fileExists(repoRoot, typePath) && typePath != filePath {
			related = append(related, RelatedFile{Path: typePath, RelationType: RelationTypes})
		}
	}
	return related
}

// findConventionFiles finds related files based on naming conventions
func (s *Service) findConventionFiles(ctx context.Context, filePath string, repoRoot string) []RelatedFile {
	dir := filepath.Dir(filePath)
	base := filepath.Base(filePath)
	ext := filepath.Ext(base)
	nameWithoutExt := strings.TrimSuffix(base, ext)

	var related []RelatedFile

	if isTestFileName(nameWithoutExt) {
		related = append(related, s.findSourceForTestFile(dir, base, nameWithoutExt, ext, repoRoot)...)
	} else {
		related = append(related, s.findTestFilesForSource(dir, nameWithoutExt, ext, repoRoot)...)
	}

	related = append(related, s.findIndexFiles(dir, base, repoRoot)...)
	related = append(related, s.findTypeFiles(dir, nameWithoutExt, ext, filePath, repoRoot)...)

	return related
}

// fileExists checks if a file exists
func (s *Service) fileExists(repoRoot, relPath string) bool {
	absPath := filepath.Join(repoRoot, relPath)
	info, err := os.Stat(absPath)
	if err != nil {
		return false
	}
	return !info.IsDir()
}
