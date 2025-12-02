// Package discovery finds requirement files in a scenario directory.
package discovery

import (
	"context"
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
)

// Reader abstracts file reading for discovery operations.
type Reader interface {
	ReadFile(path string) ([]byte, error)
	ReadDir(path string) ([]fs.DirEntry, error)
	Stat(path string) (fs.FileInfo, error)
	Exists(path string) bool
}

// Sentinel errors for discovery.
var (
	ErrNoRequirementsDir = errors.New("requirements directory not found")
	ErrInvalidJSON       = errors.New("invalid JSON in requirement file")
	ErrMissingReference  = errors.New("validation references non-existent file")
)

// DiscoveredFile represents a found requirement file.
type DiscoveredFile struct {
	AbsolutePath string
	RelativePath string
	IsIndex      bool
	ModuleDir    string // Parent directory name (e.g., "01-internal-orchestrator")
}

// Discoverer finds requirement files in a scenario.
type Discoverer interface {
	// Discover finds all requirement files, respecting index.json imports.
	Discover(ctx context.Context, scenarioRoot string) ([]DiscoveredFile, error)
}

// discoverer implements Discoverer using file system operations.
type discoverer struct {
	reader Reader
}

// osReader implements Reader using the os package.
type osReader struct{}

func (r *osReader) ReadFile(path string) ([]byte, error)       { return os.ReadFile(path) }
func (r *osReader) ReadDir(path string) ([]fs.DirEntry, error) { return os.ReadDir(path) }
func (r *osReader) Stat(path string) (fs.FileInfo, error)      { return os.Stat(path) }
func (r *osReader) Exists(path string) bool                    { _, err := os.Stat(path); return err == nil }

// New creates a Discoverer with the provided Reader.
func New(reader Reader) Discoverer {
	return &discoverer{reader: reader}
}

// NewDefault creates a Discoverer using the real file system.
func NewDefault() Discoverer {
	return &discoverer{reader: &osReader{}}
}

// Discover finds all requirement files in the scenario's requirements directory.
// It follows the import graph starting from index.json.
func (d *discoverer) Discover(ctx context.Context, scenarioRoot string) ([]DiscoveredFile, error) {
	requirementsDir := filepath.Join(scenarioRoot, "requirements")

	// Check if requirements directory exists
	if !d.reader.Exists(requirementsDir) {
		return nil, ErrNoRequirementsDir
	}

	indexPath := filepath.Join(requirementsDir, "index.json")
	if !d.reader.Exists(indexPath) {
		// Fall back to scanning all JSON files if no index exists
		return d.scanAllJSONFiles(ctx, requirementsDir)
	}

	// Parse index.json and follow imports
	return d.discoverFromIndex(ctx, requirementsDir, indexPath)
}

// discoverFromIndex parses index.json and resolves all imported modules.
func (d *discoverer) discoverFromIndex(ctx context.Context, requirementsDir, indexPath string) ([]DiscoveredFile, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	// Read and parse index.json
	data, err := d.reader.ReadFile(indexPath)
	if err != nil {
		return nil, &ParseError{FilePath: indexPath, Err: err}
	}

	var indexFile struct {
		Imports []string `json:"imports"`
	}
	if err := json.Unmarshal(data, &indexFile); err != nil {
		return nil, &ParseError{FilePath: indexPath, Err: ErrInvalidJSON}
	}

	relPath, _ := filepath.Rel(requirementsDir, indexPath)
	files := []DiscoveredFile{
		{
			AbsolutePath: indexPath,
			RelativePath: relPath,
			IsIndex:      true,
			ModuleDir:    "",
		},
	}

	// Resolve each import
	visited := make(map[string]bool)
	visited[indexPath] = true

	for _, importPath := range indexFile.Imports {
		if err := d.resolveImport(ctx, requirementsDir, importPath, visited, &files); err != nil {
			// Log warning but continue - partial discovery is better than none
			continue
		}
	}

	return files, nil
}

// resolveImport resolves a single import path and any nested imports.
func (d *discoverer) resolveImport(ctx context.Context, baseDir, importPath string, visited map[string]bool, files *[]DiscoveredFile) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	absPath := filepath.Join(baseDir, importPath)
	if visited[absPath] {
		return nil // Already processed
	}
	visited[absPath] = true

	if !d.reader.Exists(absPath) {
		return &DiscoveryError{Path: absPath, Err: ErrMissingReference}
	}

	// Determine if this is an index or module file
	isIndex := filepath.Base(absPath) == "index.json"
	moduleDir := ""
	if !isIndex {
		moduleDir = filepath.Base(filepath.Dir(absPath))
	}

	discovered := DiscoveredFile{
		AbsolutePath: absPath,
		RelativePath: importPath,
		IsIndex:      isIndex,
		ModuleDir:    moduleDir,
	}
	*files = append(*files, discovered)

	// If it's a JSON file, check for nested imports
	data, err := d.reader.ReadFile(absPath)
	if err != nil {
		return nil // File exists but can't read - continue anyway
	}

	var nested struct {
		Imports []string `json:"imports"`
	}
	if err := json.Unmarshal(data, &nested); err != nil {
		return nil // Not valid JSON or no imports field - that's fine
	}

	// Recursively resolve nested imports
	nestedBaseDir := filepath.Dir(absPath)
	for _, nestedImport := range nested.Imports {
		// Imports are relative to the current file's directory
		d.resolveImport(ctx, nestedBaseDir, nestedImport, visited, files)
	}

	return nil
}

// scanAllJSONFiles finds all .json files in the requirements directory.
// Used as fallback when no index.json exists.
func (d *discoverer) scanAllJSONFiles(ctx context.Context, requirementsDir string) ([]DiscoveredFile, error) {
	var files []DiscoveredFile

	entries, err := d.reader.ReadDir(requirementsDir)
	if err != nil {
		return nil, &DiscoveryError{Path: requirementsDir, Err: err}
	}

	for _, entry := range entries {
		select {
		case <-ctx.Done():
			return files, ctx.Err()
		default:
		}

		name := entry.Name()
		absPath := filepath.Join(requirementsDir, name)

		if entry.IsDir() {
			// Look for module.json in subdirectories
			moduleJSON := filepath.Join(absPath, "module.json")
			if d.reader.Exists(moduleJSON) {
				relPath := filepath.Join(name, "module.json")
				files = append(files, DiscoveredFile{
					AbsolutePath: moduleJSON,
					RelativePath: relPath,
					IsIndex:      false,
					ModuleDir:    name,
				})
			}
		} else if filepath.Ext(name) == ".json" {
			isIndex := name == "index.json"
			files = append(files, DiscoveredFile{
				AbsolutePath: absPath,
				RelativePath: name,
				IsIndex:      isIndex,
				ModuleDir:    "",
			})
		}
	}

	return files, nil
}

// DiscoverWithScanner performs a full recursive scan without following imports.
// Useful for finding orphaned files or doing a complete audit.
func DiscoverWithScanner(ctx context.Context, reader Reader, requirementsDir string) ([]DiscoveredFile, error) {
	scanner := NewScanner(reader)
	return scanner.ScanRecursive(ctx, requirementsDir)
}

// ParseError wraps parsing errors with file context.
type ParseError struct {
	FilePath string
	Err      error
}

func (e *ParseError) Error() string {
	return e.FilePath + ": " + e.Err.Error()
}

func (e *ParseError) Unwrap() error {
	return e.Err
}

// DiscoveryError represents an error during file discovery.
type DiscoveryError struct {
	Path string
	Err  error
}

func (e *DiscoveryError) Error() string {
	return "discovery error at " + e.Path + ": " + e.Err.Error()
}

func (e *DiscoveryError) Unwrap() error {
	return e.Err
}
