// Package scanner provides the core abstractions for import scanning
package scanner

import "context"

// ImportKind describes the type of import statement
type ImportKind string

const (
	ImportKindStatic     ImportKind = "static"      // Static import (e.g., import x from 'y')
	ImportKindDynamic    ImportKind = "dynamic"     // Dynamic import (e.g., import('x'))
	ImportKindSideEffect ImportKind = "side_effect" // Side-effect import (e.g., import 'x')
)

// Import represents a single import statement
type Import struct {
	Source     string     // Import specifier (e.g., "./utils", "react")
	IsRelative bool       // True for relative imports starting with . or ..
	Line       int        // Source line number (1-indexed)
	Kind       ImportKind // Type of import
}

// Export represents an export statement that re-exports from another module
type Export struct {
	Source     string // Export source (e.g., export * from './utils')
	IsRelative bool   // True for relative exports
	Line       int    // Source line number
}

// ScanResult contains the parsed imports and exports from a file
type ScanResult struct {
	Imports []Import
	Exports []Export
}

// ImportScanner extracts import/export information from file content
type ImportScanner interface {
	// Extensions returns file extensions this scanner handles (e.g., ".ts", ".tsx")
	Extensions() []string

	// Scan extracts import/export information from file content
	// Returns error if content is malformed or unscannable
	Scan(ctx context.Context, content string, filePath string) (*ScanResult, error)
}
