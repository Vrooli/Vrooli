package scanner

import (
	"path/filepath"
	"strings"
)

// Registry maps file extensions to their appropriate scanner
type Registry struct {
	scanners map[string]ImportScanner
}

// NewRegistry creates a new scanner registry
func NewRegistry() *Registry {
	return &Registry{
		scanners: make(map[string]ImportScanner),
	}
}

// Register adds a scanner to the registry for all its extensions
func (r *Registry) Register(scanner ImportScanner) {
	for _, ext := range scanner.Extensions() {
		r.scanners[ext] = scanner
	}
}

// Get returns the scanner for a given file path based on its extension
// Returns nil if no scanner is registered for the extension
func (r *Registry) Get(filePath string) ImportScanner {
	ext := strings.ToLower(filepath.Ext(filePath))
	return r.scanners[ext]
}

// HasScanner returns true if a scanner exists for the given file path
func (r *Registry) HasScanner(filePath string) bool {
	return r.Get(filePath) != nil
}
