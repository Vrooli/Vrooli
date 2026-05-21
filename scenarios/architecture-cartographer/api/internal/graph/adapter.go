package graph

import "context"

// CodeGraphAdapter is the seam between cartographer and the
// language-specific code-graph scenarios (go-code-graph,
// typescript-code-graph). v0.1 ships the interface + a fake; production
// clients land in Phase 5.
//
// All cartographer logic builds against this interface so the absence
// of the dependency scenarios does not block development. End-to-end
// integration tests that require live language-graph scenarios are
// tagged `e2e_lang_graph` and skipped by default.
type CodeGraphAdapter interface {
	// Name identifies the adapter ("go", "typescript"). Used by the
	// service to pick the right adapter for a language filter.
	Name() string

	// SupportedLanguages declares which Language values this adapter
	// returns nodes for.
	SupportedLanguages() []Language

	// Extract requests a raw graph for the target scenario. The
	// adapter is expected to return a graph that downstream
	// Normalize() can dedupe and content-hash; cartographer does no
	// language-specific work on the result.
	Extract(ctx context.Context, scenario string) (RawGraph, error)
}

// RawGraph is the un-normalized result an adapter returns. Normalize()
// dedupes + hashes into a GraphSnapshot.
type RawGraph struct {
	Languages []Language
	Files     []FileNode
	Packages  []PackageNode
	Symbols   []SymbolNode
	Imports   []ImportEdge
	// ExtractionMS is the adapter-reported elapsed extraction time, in
	// milliseconds. Cartographer also measures its own wall-clock for
	// cross-checking.
	ExtractionMS int64
}
