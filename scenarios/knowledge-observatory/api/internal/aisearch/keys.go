// Package aisearch is knowledge-observatory's documentation consumer of the
// shared retrieval engine (github.com/vrooli/aisearch-go). It supplies the
// KO-local pieces the generic engine is parameterized over: a manifest-driven
// documentation Source, a markdown-header-aware Chunker, and a contextual
// EmbeddingTextComposer. The embed/sparse/vectorstore/reconcile core lives in
// the shared package; this package only shapes documentation into it.
//
// Phase 3 of the KO search cutover delivers indexing (Source + Chunker +
// Composer + the reconciler wiring). The hybrid search read path + rerank
// (Phase 5) and the Connect/CLI surface + sync loop (Phase 6) build on top of
// the Indexer exposed here.
package aisearch

import pkg "github.com/vrooli/aisearch-go"

// DocKind is the logical collection/entity these sources belong to. One Source
// emits a single Kind (the shared SourceBinding.Kind).
const DocKind = "doc"

// DefaultCollection is the single documentation collection. Per Phase-0
// decision D1 the corpus lives in one collection with a scope/scenario payload
// filter rather than per-scenario collections, so a query can span the whole
// corpus or be confined to one scenario / path prefix with an in-leaf filter.
const DefaultCollection = "vrooli-docs"

// IDPrefix namespaces KO documentation point IDs inside the shared engine so a
// doc's natural key (its repo-relative path) never collides with another
// consumer's in a shared collection.
const IDPrefix = "ko-docs:"

// Payload metadata keys. kodocsource writes these into SourceDoc.Meta; the
// shared engine copies them verbatim into every chunk's Qdrant payload, where
// the contextual composer reads title/description/heading_path for the
// embedding prefix and (Phase 5) the search service filters on scope/scenario/
// doc_type/audience/maturity/canonical_for and projects results back to hits.
//
// The first results-projection fields (id, relative_path, score, snippet, path)
// are the search-hub federation contract for the KO `doc` leaf — keep
// relative_path and path stable or update the registry descriptor in lockstep.
const (
	MetaScenario     = "scenario"      // owning scenario name ("" for project-level docs)
	MetaRelativePath = "relative_path" // repo-relative path, forward slashes
	MetaPath         = "path"          // repo-relative path (federation contract field)
	MetaDocType      = "doc_type"      // manifest docType, or inferred (readme/prd/doc)
	MetaAudience     = "audience"      // []string facet
	MetaCanonicalFor = "canonical_for" // []string facet
	MetaMaturity     = "maturity"      // manifest maturity (active/draft/stub/...)
	MetaScope        = "scope"         // derived: project | scenario
	MetaSource       = "source"        // discovery origin: manifest | readme | prd | sweep
	MetaPathPrefixes = "path_prefixes" // []string ancestor dir prefixes, for server-side path scope

	// MetaTitle/MetaDescription/MetaHeadingPath are the contextual-composition
	// keys OWNED by the shared package (its MarkdownChunker writes heading_path;
	// its ContextualComposer reads all three). Aliased here so KO's source/search
	// code keeps using the short names while the string values stay single-homed
	// in the package — no drift, and the Qdrant payload keys are unchanged.
	MetaTitle       = pkg.MetaTitle
	MetaDescription = pkg.MetaDescription
	MetaHeadingPath = pkg.MetaHeadingPath
)

// Scope values stored in the MetaScope payload field.
const (
	ScopeProject  = "project"  // a repo-root /docs document
	ScopeScenario = "scenario" // a scenario (or template scenario) document
)

// Discovery-origin values stored in the MetaSource payload field.
const (
	SourceManifest = "manifest"
	SourceReadme   = "readme"
	SourcePRD      = "prd"
	// SourceSweep marks a doc discovered by the filesystem sweep of a docs/
	// tree rather than by a manifest entry — so search coverage never silently
	// depends on a manifest listing every file (it gets inferred metadata).
	SourceSweep = "sweep"
)
