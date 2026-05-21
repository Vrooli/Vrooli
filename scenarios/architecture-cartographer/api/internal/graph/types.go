// Package graph is the domain-scoped home for graph snapshot
// extraction, normalization, and caching. Parsing of source code lives
// strictly in the language code-graph scenarios (go-code-graph,
// typescript-code-graph) — graph delegates through CodeGraphAdapter.
//
// Layering mirrors the canonical Vrooli pattern (handler → Service →
// Repository), with mocks/ co-located.
//
// Phase 2 ships types + repository + sqlite + service + the day-one
// seam stubs (adapter.go, normalize.go, cache.go); Phase 5 fills the
// stubs in with real production behavior.
package graph

import (
	"fmt"
	"time"
)

// Language is the source language a node/edge belongs to.
type Language string

const (
	LanguageUnspecified Language = ""
	LanguageGo          Language = "go"
	LanguageTypeScript  Language = "ts"
)

// FileNode is one source file in the target scenario.
type FileNode struct {
	ID        string
	Path      string
	PackageID string
	Language  Language
	Lines     int
	IsTest    bool
}

// PackageNode is one logical package or module.
type PackageNode struct {
	ID         string
	ImportPath string
	Directory  string
	Language   Language
	Internal   bool
}

// SymbolNode is one exported identifier.
type SymbolNode struct {
	ID        string
	Name      string
	PackageID string
	FileID    string
	Kind      string
	Exported  bool
}

// ImportEdge is a directed dependency.
type ImportEdge struct {
	From        string
	ToPackageID string
	SymbolIDs   []string
	TestOnly    bool
}

// Chunk is the unit of scoring. v0.1 chunks correspond 1:1 with a FileNode.
type Chunk struct {
	ID            string
	FileID        string
	Path          string
	CurrentDomain string
}

// GraphSnapshot is the normalized, immutable result of extracting a
// language graph. Cartographer never mutates a snapshot after
// construction.
type GraphSnapshot struct {
	ID           string
	Scenario     string
	ContentHash  string
	Languages    []Language
	ExtractedAt  time.Time
	ExtractionMS int64
	Files        []FileNode
	Packages     []PackageNode
	Symbols      []SymbolNode
	Imports      []ImportEdge
}

// Chunks derives the canonical Chunk list from the snapshot's file
// nodes. v0.1 chunk = file; later versions may sub-chunk.
func (s GraphSnapshot) Chunks() []Chunk {
	out := make([]Chunk, len(s.Files))
	for i, f := range s.Files {
		out[i] = Chunk{
			ID:     "chunk:" + f.ID,
			FileID: f.ID,
			Path:   f.Path,
		}
	}
	return out
}

// ErrSnapshotNotFound is the typed sentinel returned by
// Repository.GetSnapshot when no row matches.
type ErrSnapshotNotFound struct {
	ID string
}

func (e ErrSnapshotNotFound) Error() string {
	return fmt.Sprintf("graph snapshot %q not found", e.ID)
}

// ErrInvalidExtractRequest is the typed sentinel returned by
// Service.ExtractGraph on validation failure.
type ErrInvalidExtractRequest struct {
	Field  string
	Reason string
}

func (e ErrInvalidExtractRequest) Error() string {
	return fmt.Sprintf("invalid extract request: %s: %s", e.Field, e.Reason)
}

// IntegrationError signals that an external dependency (language
// code-graph scenario) was unreachable.
type IntegrationError struct {
	Kind     string // e.g., "scenario_unreachable", "timeout"
	Scenario string // e.g., "go-code-graph"
	Cause    error
}

func (e IntegrationError) Error() string {
	return fmt.Sprintf("integration %s with %s: %v", e.Kind, e.Scenario, e.Cause)
}

func (e IntegrationError) Unwrap() error { return e.Cause }
