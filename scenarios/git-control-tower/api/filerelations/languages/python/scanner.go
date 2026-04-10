// Package python provides an import scanner for Python files
package python

import (
	"context"

	"git-control-tower/filerelations/scanner"
)

// Scanner implements scanner.ImportScanner for Python files
type Scanner struct{}

// New creates a new Python scanner
func New() *Scanner {
	return &Scanner{}
}

// Extensions returns the file extensions this scanner handles
func (s *Scanner) Extensions() []string {
	return []string{".py", ".pyi"}
}

// Scan extracts imports from Python file content
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

	// Find 'import module' statements
	for _, match := range MatchImports(content) {
		result.Imports = append(result.Imports, scanner.Import{
			Source:     match.Module,
			IsRelative: IsRelativeImport(match.Module),
			Line:       match.Line,
			Kind:       scanner.ImportKindStatic,
		})
	}

	// Find 'from module import' statements
	for _, match := range MatchFromImports(content) {
		result.Imports = append(result.Imports, scanner.Import{
			Source:     match.Module,
			IsRelative: IsRelativeImport(match.Module),
			Line:       match.Line,
			Kind:       scanner.ImportKindStatic,
		})
	}

	return result, nil
}
