// Package filerelations provides file relationship discovery services
package filerelations

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	"git-control-tower/filerelations/languages/golang"
	"git-control-tower/filerelations/languages/python"
	"git-control-tower/filerelations/languages/typescript"
	"git-control-tower/filerelations/resolver"
	"git-control-tower/filerelations/scanner"
)

// RelationType describes how files are related
type RelationType string

const (
	RelationImports    RelationType = "imports"     // Files this file imports
	RelationImportedBy RelationType = "imported_by" // Files that import this file
	RelationTest       RelationType = "test"        // Test file for this file
	RelationIndex      RelationType = "index"       // Index file that exports this file
	RelationTypes      RelationType = "types"       // Type definition file
)

// RelatedFile represents a file related to another file
type RelatedFile struct {
	Path         string       `json:"path"`
	RelationType RelationType `json:"relation_type"`
}

// Service provides file relationship discovery
type Service struct {
	registry         *scanner.Registry
	resolver         *resolver.RelativeResolver
	goModResolver    *resolver.GoModResolver
	tsConfigResolver *resolver.TSConfigResolver
}

// NewService creates a new file relations service
func NewService() *Service {
	reg := scanner.NewRegistry()

	// Register language scanners
	reg.Register(typescript.New())
	reg.Register(golang.New())
	reg.Register(python.New())

	return &Service{
		registry:         reg,
		resolver:         resolver.NewRelativeResolver(),
		goModResolver:    resolver.NewGoModResolver(),
		tsConfigResolver: resolver.NewTSConfigResolver(),
	}
}

// GetRelatedFiles finds all files related to the given file
func (s *Service) GetRelatedFiles(ctx context.Context, filePath string, repoRoot string) ([]RelatedFile, error) {
	var related []RelatedFile

	// Find imports (files this file imports)
	imports, err := s.findImports(ctx, filePath, repoRoot)
	if err == nil {
		related = append(related, imports...)
	}

	// Find convention-based related files (tests, index, types)
	conventionFiles := s.findConventionFiles(ctx, filePath, repoRoot)
	related = append(related, conventionFiles...)

	// Note: Finding "imported by" requires scanning all files in the repo
	// which is expensive. We'll skip this for now and can add it later
	// with caching if needed.

	return related, nil
}

// findImports finds files that the given file imports
func (s *Service) findImports(ctx context.Context, filePath string, repoRoot string) ([]RelatedFile, error) {
	var related []RelatedFile

	// Get the appropriate scanner for this file type
	sc := s.registry.Get(filePath)
	if sc == nil {
		return related, nil // No scanner for this file type
	}

	// Read the file content
	absPath := filepath.Join(repoRoot, filePath)
	content, err := os.ReadFile(absPath)
	if err != nil {
		return nil, err
	}

	// Scan for imports
	result, err := sc.Scan(ctx, string(content), filePath)
	if err != nil {
		return nil, err
	}

	// Resolve each import to a file path
	seen := make(map[string]bool)
	ext := filepath.Ext(filePath)
	isGoFile := ext == ".go"
	isTSFile := ext == ".ts" || ext == ".tsx" || ext == ".js" || ext == ".jsx"

	for _, imp := range result.Imports {
		var resolved string

		if imp.IsRelative {
			// Use relative resolver for relative imports
			resolved = s.resolver.Resolve(imp.Source, filePath, repoRoot)
		} else {
			// Try non-relative resolvers based on file type
			if isGoFile {
				resolved = s.goModResolver.Resolve(imp.Source, filePath, repoRoot)
			} else if isTSFile {
				resolved = s.tsConfigResolver.Resolve(imp.Source, filePath, repoRoot)
			}
		}

		if resolved != "" && !seen[resolved] && resolved != filePath {
			seen[resolved] = true
			related = append(related, RelatedFile{
				Path:         resolved,
				RelationType: RelationImports,
			})
		}
	}

	// Also resolve re-exports
	for _, exp := range result.Exports {
		var resolved string

		if exp.IsRelative {
			resolved = s.resolver.Resolve(exp.Source, filePath, repoRoot)
		} else {
			// Try non-relative resolvers based on file type
			if isGoFile {
				resolved = s.goModResolver.Resolve(exp.Source, filePath, repoRoot)
			} else if isTSFile {
				resolved = s.tsConfigResolver.Resolve(exp.Source, filePath, repoRoot)
			}
		}

		if resolved != "" && !seen[resolved] && resolved != filePath {
			seen[resolved] = true
			related = append(related, RelatedFile{
				Path:         resolved,
				RelationType: RelationImports,
			})
		}
	}

	return related, nil
}

// findConventionFiles finds related files based on naming conventions
func (s *Service) findConventionFiles(ctx context.Context, filePath string, repoRoot string) []RelatedFile {
	var related []RelatedFile

	dir := filepath.Dir(filePath)
	base := filepath.Base(filePath)
	ext := filepath.Ext(base)
	nameWithoutExt := strings.TrimSuffix(base, ext)

	// Skip if this is already a test/spec file
	isTestFile := strings.HasSuffix(nameWithoutExt, ".test") ||
		strings.HasSuffix(nameWithoutExt, ".spec") ||
		strings.HasSuffix(nameWithoutExt, "_test")

	// Find test files
	if !isTestFile {
		// Build test file patterns for the base name
		// Include common test extensions for JS/TS projects
		testPatterns := []string{
			nameWithoutExt + ".test" + ext,
			nameWithoutExt + ".spec" + ext,
			nameWithoutExt + "_test" + ext,
			nameWithoutExt + "_test.go",
		}

		// Also check alternative extensions for JS/TS files
		if ext == ".js" || ext == ".jsx" || ext == ".ts" || ext == ".tsx" {
			altExts := []string{".ts", ".tsx", ".js", ".jsx"}
			for _, altExt := range altExts {
				if altExt != ext {
					testPatterns = append(testPatterns,
						nameWithoutExt+".test"+altExt,
						nameWithoutExt+".spec"+altExt,
						nameWithoutExt+"_test"+altExt,
					)
				}
			}
		}

		// Track found paths to avoid duplicates
		foundPaths := make(map[string]bool)

		// 1. Check same directory
		for _, pattern := range testPatterns {
			testPath := filepath.Join(dir, pattern)
			if !foundPaths[testPath] && s.fileExists(repoRoot, testPath) {
				foundPaths[testPath] = true
				related = append(related, RelatedFile{
					Path:         testPath,
					RelationType: RelationTest,
				})
			}
		}

		// 2. Check __tests__ and __test__ subdirectories (both common conventions)
		testsSubDirs := []string{"__tests__", "__test__"}
		for _, subDir := range testsSubDirs {
			testsSubDir := filepath.Join(dir, subDir)
			for _, pattern := range testPatterns {
				testPath := filepath.Join(testsSubDir, pattern)
				if !foundPaths[testPath] && s.fileExists(repoRoot, testPath) {
					foundPaths[testPath] = true
					related = append(related, RelatedFile{
						Path:         testPath,
						RelationType: RelationTest,
					})
				}
			}
		}

		// 3. Check top-level test directories mirroring the source structure
		// e.g., src/components/Button.tsx -> tests/components/Button.test.tsx
		// or src/components/Button.tsx -> test/src/components/Button.test.tsx
		topLevelTestDirs := []string{"tests", "test", "__tests__", "__test__"}
		for _, testDir := range topLevelTestDirs {
			// Try direct mirror (tests/components/Button.test.tsx for src/components/Button.tsx)
			testDirPath := filepath.Join(testDir, dir)
			for _, pattern := range testPatterns {
				testPath := filepath.Join(testDirPath, pattern)
				if !foundPaths[testPath] && s.fileExists(repoRoot, testPath) {
					foundPaths[testPath] = true
					related = append(related, RelatedFile{
						Path:         testPath,
						RelationType: RelationTest,
					})
				}
			}

			// Try with source prefix stripped (tests/components/Button.test.tsx for src/components/Button.tsx)
			// Common pattern: strip "src/" prefix
			if strings.HasPrefix(dir, "src/") || strings.HasPrefix(dir, "src\\") {
				strippedDir := strings.TrimPrefix(strings.TrimPrefix(dir, "src/"), "src\\")
				testDirPath := filepath.Join(testDir, strippedDir)
				for _, pattern := range testPatterns {
					testPath := filepath.Join(testDirPath, pattern)
					if !foundPaths[testPath] && s.fileExists(repoRoot, testPath) {
						foundPaths[testPath] = true
						related = append(related, RelatedFile{
							Path:         testPath,
							RelationType: RelationTest,
						})
					}
				}
			}
		}
	} else {
		// This IS a test file - find the source file
		var sourceBase string
		switch {
		case strings.HasSuffix(nameWithoutExt, ".test"):
			sourceBase = strings.TrimSuffix(nameWithoutExt, ".test")
		case strings.HasSuffix(nameWithoutExt, ".spec"):
			sourceBase = strings.TrimSuffix(nameWithoutExt, ".spec")
		case strings.HasSuffix(nameWithoutExt, "_test"):
			sourceBase = strings.TrimSuffix(nameWithoutExt, "_test")
		}

		if sourceBase != "" {
			// Check same directory for source file
			sourcePath := filepath.Join(dir, sourceBase+ext)
			if s.fileExists(repoRoot, sourcePath) {
				related = append(related, RelatedFile{Path: sourcePath, RelationType: RelationTest})
			}

			// Check alternative extensions for JS/TS files
			if ext == ".js" || ext == ".jsx" || ext == ".ts" || ext == ".tsx" {
				altExts := []string{".ts", ".tsx", ".js", ".jsx"}
				for _, altExt := range altExts {
					if altExt != ext {
						altSourcePath := filepath.Join(dir, sourceBase+altExt)
						if s.fileExists(repoRoot, altSourcePath) {
							related = append(related, RelatedFile{Path: altSourcePath, RelationType: RelationTest})
						}
					}
				}
			}

			// Check parent directory (for __tests__ or __test__ subfolders)
			parentDir := filepath.Dir(dir)
			baseDirName := filepath.Base(dir)
			if baseDirName == "__tests__" || baseDirName == "__test__" {
				parentSourcePath := filepath.Join(parentDir, sourceBase+ext)
				if s.fileExists(repoRoot, parentSourcePath) {
					related = append(related, RelatedFile{Path: parentSourcePath, RelationType: RelationTest})
				}

				// Also check alternative extensions in parent
				if ext == ".js" || ext == ".jsx" || ext == ".ts" || ext == ".tsx" {
					altExts := []string{".ts", ".tsx", ".js", ".jsx"}
					for _, altExt := range altExts {
						if altExt != ext {
							altSourcePath := filepath.Join(parentDir, sourceBase+altExt)
							if s.fileExists(repoRoot, altSourcePath) {
								related = append(related, RelatedFile{Path: altSourcePath, RelationType: RelationTest})
							}
						}
					}
				}
			}
		}

		// Handle Go _test.go -> .go specifically
		if ext == ".go" && strings.HasSuffix(base, "_test.go") {
			goSourceBase := strings.TrimSuffix(base, "_test.go")
			goSourcePath := filepath.Join(dir, goSourceBase+".go")
			if s.fileExists(repoRoot, goSourcePath) {
				related = append(related, RelatedFile{Path: goSourcePath, RelationType: RelationTest})
			}
		}
	}

	// Find index files (if not already an index file)
	if !strings.HasPrefix(base, "index.") && base != "__init__.py" {
		indexPatterns := []string{
			"index.ts",
			"index.tsx",
			"index.js",
			"index.jsx",
			"index.go",
			"__init__.py",
			"mod.rs",
		}

		for _, pattern := range indexPatterns {
			indexPath := filepath.Join(dir, pattern)
			if s.fileExists(repoRoot, indexPath) {
				related = append(related, RelatedFile{
					Path:         indexPath,
					RelationType: RelationIndex,
				})
			}
		}
	}

	// Find type files for TypeScript
	if ext == ".ts" || ext == ".tsx" || ext == ".js" || ext == ".jsx" {
		typePatterns := []string{
			nameWithoutExt + ".types.ts",
			nameWithoutExt + ".d.ts",
			"types.ts",
			"types.d.ts",
		}

		for _, pattern := range typePatterns {
			typePath := filepath.Join(dir, pattern)
			if s.fileExists(repoRoot, typePath) && typePath != filePath {
				related = append(related, RelatedFile{
					Path:         typePath,
					RelationType: RelationTypes,
				})
			}
		}
	}

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
