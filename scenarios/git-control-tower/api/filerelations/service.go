// Package filerelations provides file relationship discovery services
package filerelations

import (
	"context"
	"os"
	"path/filepath"

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

// isJSTSFile returns true if the extension is a JavaScript/TypeScript extension.
func isJSTSFile(ext string) bool {
	return ext == ".ts" || ext == ".tsx" || ext == ".js" || ext == ".jsx"
}

// jsTSExtensions returns the set of JS/TS file extensions.
func jsTSExtensions() []string {
	return []string{".ts", ".tsx", ".js", ".jsx"}
}

// resolveImportSource resolves an import/export source to a file path.
func (s *Service) resolveImportSource(source string, isRelative, isGoFile, isTSFile bool, filePath, repoRoot string) string {
	if isRelative {
		return s.resolver.Resolve(source, filePath, repoRoot)
	}
	if isGoFile {
		return s.goModResolver.Resolve(source, filePath, repoRoot)
	}
	if isTSFile {
		return s.tsConfigResolver.Resolve(source, filePath, repoRoot)
	}
	return ""
}

// importRef represents a source reference that needs resolution (import or re-export).
type importRef struct {
	Source     string
	IsRelative bool
}

// collectImportRefs gathers all import and re-export references from a scan result.
func collectImportRefs(result *scanner.ScanResult) []importRef {
	refs := make([]importRef, 0, len(result.Imports)+len(result.Exports))
	for _, imp := range result.Imports {
		refs = append(refs, importRef{Source: imp.Source, IsRelative: imp.IsRelative})
	}
	for _, exp := range result.Exports {
		refs = append(refs, importRef{Source: exp.Source, IsRelative: exp.IsRelative})
	}
	return refs
}

// findImports finds files that the given file imports
func (s *Service) findImports(ctx context.Context, filePath string, repoRoot string) ([]RelatedFile, error) {
	sc := s.registry.Get(filePath)
	if sc == nil {
		return nil, nil
	}

	absPath := filepath.Join(repoRoot, filePath)
	content, err := os.ReadFile(absPath)
	if err != nil {
		return nil, err
	}

	result, err := sc.Scan(ctx, string(content), filePath)
	if err != nil {
		return nil, err
	}

	ext := filepath.Ext(filePath)
	isGoFile := ext == ".go"
	isTSFile := isJSTSFile(ext)
	seen := make(map[string]bool)
	var related []RelatedFile

	for _, ref := range collectImportRefs(result) {
		resolved := s.resolveImportSource(ref.Source, ref.IsRelative, isGoFile, isTSFile, filePath, repoRoot)
		if resolved != "" && !seen[resolved] && resolved != filePath {
			seen[resolved] = true
			related = append(related, RelatedFile{Path: resolved, RelationType: RelationImports})
		}
	}

	return related, nil
}

// Convention-based test matching functions are in service_conventions.go
