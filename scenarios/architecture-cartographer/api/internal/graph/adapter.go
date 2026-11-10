package graph

import "context"

// CodeGraphAdapter projects Code Facts language graph envelopes into immutable
// architectural snapshots. Code Facts owns source discovery, extraction and cache.
// The live shared-owner regression is tagged e2e_lang_graph.
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
	// SourceFingerprint is supplied by Code Facts, the source identity owner.
	SourceFingerprint string
	SkippedAdapters   []string
	Languages         []Language
	Files             []FileNode
	Packages          []PackageNode
	Symbols           []SymbolNode
	Imports           []ImportEdge
	// ExtractionMS is the adapter-reported elapsed extraction time, in
	// milliseconds. Cartographer also measures its own wall-clock for
	// cross-checking.
	ExtractionMS int64
	// ExtractionProfiles are the effective producer profiles. They are
	// preserved for explainability when lighter profiles intentionally omit
	// facts.
	ExtractionProfiles []string
	OmittedInformation []InformationOmission
}

// InformationOmission explains a capability intentionally excluded by an
// adapter's extraction profile. It is informational, not an error.
type InformationOmission struct {
	Capability string
	Reason     string
}
