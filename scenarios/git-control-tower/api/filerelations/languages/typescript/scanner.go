// Package typescript provides an import scanner for TypeScript and JavaScript files
package typescript

import (
	"context"

	"git-control-tower/filerelations/scanner"
)

// Scanner implements scanner.ImportScanner for TypeScript/JavaScript files
type Scanner struct{}

// New creates a new TypeScript scanner
func New() *Scanner {
	return &Scanner{}
}

// Extensions returns the file extensions this scanner handles
func (s *Scanner) Extensions() []string {
	return []string{".ts", ".tsx", ".js", ".jsx", ".mjs", ".cjs"}
}

// Scan extracts imports and exports from TypeScript/JavaScript content
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

	// Find static imports
	for _, match := range MatchStaticImports(content) {
		result.Imports = append(result.Imports, scanner.Import{
			Source:     match.Source,
			IsRelative: IsRelativeImport(match.Source),
			Line:       match.Line,
			Kind:       scanner.ImportKindStatic,
		})
	}

	// Find dynamic imports
	for _, match := range MatchDynamicImports(content) {
		result.Imports = append(result.Imports, scanner.Import{
			Source:     match.Source,
			IsRelative: IsRelativeImport(match.Source),
			Line:       match.Line,
			Kind:       scanner.ImportKindDynamic,
		})
	}

	// Find side-effect imports (but avoid duplicating static imports)
	staticSources := make(map[string]bool)
	for _, imp := range result.Imports {
		staticSources[imp.Source] = true
	}
	for _, match := range MatchSideEffectImports(content) {
		if !staticSources[match.Source] {
			result.Imports = append(result.Imports, scanner.Import{
				Source:     match.Source,
				IsRelative: IsRelativeImport(match.Source),
				Line:       match.Line,
				Kind:       scanner.ImportKindSideEffect,
			})
		}
	}

	// Find re-exports
	for _, match := range MatchReExports(content) {
		result.Exports = append(result.Exports, scanner.Export{
			Source:     match.Source,
			IsRelative: IsRelativeImport(match.Source),
			Line:       match.Line,
		})
	}

	return result, nil
}
