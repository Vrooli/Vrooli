// Package golang provides an import scanner for Go files
package golang

import (
	"context"

	"git-control-tower/filerelations/scanner"
)

// Scanner implements scanner.ImportScanner for Go files
type Scanner struct{}

// New creates a new Go scanner
func New() *Scanner {
	return &Scanner{}
}

// Extensions returns the file extensions this scanner handles
func (s *Scanner) Extensions() []string {
	return []string{".go"}
}

// Scan extracts imports from Go file content
func (s *Scanner) Scan(ctx context.Context, content string, filePath string) (*scanner.ScanResult, error) {
	result := &scanner.ScanResult{
		Imports: []scanner.Import{},
		Exports: []scanner.Export{},
	}

	// Check for cancellation
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	// Find single-line imports
	for _, match := range MatchSingleImports(content) {
		result.Imports = append(result.Imports, scanner.Import{
			Source:     match.Path,
			IsRelative: IsRelativeImport(match.Path),
			Line:       match.Line,
			Kind:       scanner.ImportKindStatic,
		})
	}

	// Find imports from import blocks
	for _, match := range MatchImportBlocks(content) {
		result.Imports = append(result.Imports, scanner.Import{
			Source:     match.Path,
			IsRelative: IsRelativeImport(match.Path),
			Line:       match.Line,
			Kind:       scanner.ImportKindStatic,
		})
	}

	// Go doesn't have re-exports in the same sense as JavaScript

	return result, nil
}
