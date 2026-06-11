package aisearch

import "context"

// =============================================================================
// 1. Sources & documents
// =============================================================================

// SourceDoc is one indexable unit as it exists at the source (a CLI command, a
// UI surface, a markdown documentation file). It is the input to chunking.
//
// ContentHash is the source-level drift gate (the first of two levels): when a
// previously-indexed SourceDoc reports an unchanged ContentHash, the
// reconciler skips it wholesale — it is never read, chunked, embedded, or
// compared chunk-by-chunk. This is the single biggest scale lever for the
// large documentation corpus (§4.1 of the plan).
type SourceDoc struct {
	// ID is the stable natural identity of the source unit (e.g. a command
	// full-path, or a documentation file's repo-relative path). Chunk point
	// IDs are derived deterministically from it plus the chunk index.
	ID string
	// Kind names the logical collection/entity the doc belongs to (e.g.
	// "command", "doc"). One Source may emit a single Kind.
	Kind string
	// ContentHash is a stable hash of the raw source content used for the
	// source-level drift skip. Empty means "always re-chunk".
	ContentHash string
	// Body is the raw source text (markdown, help output, etc.) handed to the
	// Chunker.
	Body string
	// Meta carries source metadata propagated into every chunk's payload for
	// filtering and contextual composition (e.g. scenario, relPath, docType,
	// title, description, audience, canonicalFor, maturity, scope).
	Meta map[string]any
}

// Source enumerates everything that should be indexed for one consumer. The
// command/surface consumers return one SourceDoc per command/surface; the KO
// documentation consumer walks docs/manifest.json files and returns one
// SourceDoc per documentation file.
type Source interface {
	LoadAll(ctx context.Context) ([]SourceDoc, error)
}

// =============================================================================
// 2. Chunking
// =============================================================================

// Chunk is one embeddable unit produced from a SourceDoc. For the 1:1
// consumers (commands/surfaces) a SourceDoc yields exactly one Chunk; for
// documentation a SourceDoc fans out into many.
type Chunk struct {
	// ID is the deterministic Qdrant point ID (derived from SourceID + Index).
	ID string
	// SourceID is the parent SourceDoc.ID — the second-level drift and ghost
	// deletion both group chunks by it.
	SourceID string
	// Index is the chunk's ordinal within its SourceDoc (0-based).
	Index int
	// Body is the raw, human-readable chunk text stored in the payload as the
	// retrievable snippet source. (Storing this fixes the legacy KO
	// content:"" defect.)
	Body string
	// Meta is the SourceDoc.Meta plus chunk-derived fields (e.g. headingPath).
	Meta map[string]any
}

// Chunker turns one SourceDoc into one-or-more Chunks. Impls (Phase 1+):
//   - identity:  1 SourceDoc -> 1 Chunk (commands / surfaces).
//   - markdown:  header-aware split with a contextual prefix (KO docs).
type Chunker interface {
	Chunk(doc SourceDoc) ([]Chunk, error)
}

// EmbeddingTextComposer produces the exact text that gets embedded for a
// Chunk. It is separate from Chunk.Body because the embedded text is often
// enriched: the markdown composer prepends a contextual header
// ("{title} — {description}\n{headingPath}\n\n{body}") so each chunk is
// self-contained — the single biggest accuracy lever for documentation.
type EmbeddingTextComposer interface {
	Compose(chunk Chunk) string
}

// =============================================================================
// 3. Embedding (dense) + sparse encoding
// =============================================================================

// Embedder produces the dense vector for a piece of text. The production impl
// shells out to `resource-ollama gateway embed` (lifted verbatim from
// cli-health); tests inject fakes.
type Embedder interface {
	Embed(ctx context.Context, text string) ([]float64, error)
	Available(ctx context.Context) bool
}

// TaskEmbedder is an optional Embedder that distinguishes the two embedding
// roles an asymmetric embedding policy may require: the query and the indexed
// passage are prefixed with different task instructions (for example,
// "search_query:" vs "search_document:"). The read path embeds the query with
// EmbedQuery and the reconciler embeds each passage with EmbedDocument, so an
// asymmetric model lands both sides in the trained space — the single biggest
// dense-retrieval lever for a terse corpus matched by long natural-language
// queries. An Embedder that does NOT implement this interface is embedded
// symmetrically via the embedQueryText / embedDocumentText helpers (they fall
// back to Embed), so existing fakes and symmetric models keep working unchanged.
type TaskEmbedder interface {
	Embedder
	EmbedQuery(ctx context.Context, text string) ([]float64, error)
	EmbedDocument(ctx context.Context, text string) ([]float64, error)
}

// RecipeEmbedder is an optional Embedder that reports a stable identity for its
// embedding "recipe" — the model plus any task prefixes — so the reconciler can
// fold it into the chunk drift hash. When the recipe changes (a model swap, or
// task prefixes being added), every chunk's hash changes and the next reconcile
// re-embeds the whole corpus into the new vector space, instead of silently
// serving queries embedded one way against passages embedded another. An empty
// recipe (the default for symmetric embedders and all existing fakes) folds
// nothing in, so legacy collections keep byte-identical hashes and are NOT
// spuriously re-embedded.
type RecipeEmbedder interface {
	EmbedRecipe() string
}

// embedderRecipe returns the embedder's recipe identity, or "" when it does not
// report one.
func embedderRecipe(e Embedder) string {
	if r, ok := e.(RecipeEmbedder); ok {
		return r.EmbedRecipe()
	}
	return ""
}

// embedQueryText embeds a query, using the role-aware EmbedQuery when the
// embedder supports it (TaskEmbedder) and falling back to the symmetric Embed
// otherwise. The read path calls this for the query vector.
func embedQueryText(ctx context.Context, e Embedder, text string) ([]float64, error) {
	if te, ok := e.(TaskEmbedder); ok {
		return te.EmbedQuery(ctx, text)
	}
	return e.Embed(ctx, text)
}

// embedDocumentText embeds an indexed passage, using EmbedDocument when
// available and falling back to Embed. The reconciler calls this per chunk.
func embedDocumentText(ctx context.Context, e Embedder, text string) ([]float64, error) {
	if te, ok := e.(TaskEmbedder); ok {
		return te.EmbedDocument(ctx, text)
	}
	return e.Embed(ctx, text)
}

// SparseVector is a BM25-style term-weight vector: parallel index/value
// slices. Qdrant applies IDF server-side via the collection's "idf" modifier
// (validated against the live 1.15.3 instance in Phase 0), so the encoder only
// supplies term-frequency-style weights.
type SparseVector struct {
	Indices []uint32
	Values  []float32
}

// SparseEncoder computes a SparseVector locally — no model, no GPU, computed
// inline during reconcile (Phase 0 decision: Qdrant-native sparse + idf, not
// SPLADE). Tokenize -> hash terms to indices -> term-frequency weights.
type SparseEncoder interface {
	Encode(text string) SparseVector
}

// =============================================================================
// 4. Vector store (named vectors + server-side hybrid fusion)
// =============================================================================

// CollectionSpec describes the named-vector collection layout. Model+Dim are
// recorded in collection metadata so a future embedding-model swap is a
// deliberate re-index, not a silent dimension mismatch (the exact failure that
// stranded the legacy 1024-dim KO collections).
type CollectionSpec struct {
	// Name is an optional cross-check; the authoritative collection name is the
	// one passed to NewVectorStore. EnsureCollection errors if a non-empty Name
	// disagrees with the store's collection, so it can never silently mis-target.
	Name                string
	DenseSize           int    // resolved embedding dimension
	DenseDistance       string // e.g. "Cosine"
	Sparse              bool   // create a named "sparse" vector with idf modifier
	SparseModifier      string // e.g. "idf"
	Model               string // embedding model, recorded in metadata
	Role                string // embedding policy role, recorded in metadata
	PolicySchemaVersion string // Ollama policy schema version, recorded in metadata
}

// Point is one upsert: a deterministic ID, a dense vector, an optional sparse
// vector (when hybrid is enabled), and the payload (which must carry
// payload_hash for chunk-level drift and the retrievable body+metadata).
type Point struct {
	ID      string
	Dense   []float64
	Sparse  *SparseVector
	Payload map[string]any
}

// FieldMatch is one payload filter clause. Value matches a scalar; AnyOf
// matches set membership (for multi-value facets like audience / canonicalFor).
type FieldMatch struct {
	Key   string
	Value any
	AnyOf []any
}

// QueryFilter is an AND of FieldMatch clauses applied to every prefetch leg
// (validated in Phase 0: per-prefetch filters correctly scope hybrid results).
type QueryFilter struct {
	Must []FieldMatch
}

// HybridQuery drives the Qdrant Query API. With both Dense and Sparse set and
// Fusion="rrf" it runs the two prefetch legs and fuses server-side; with only
// Dense it degrades to a dense-only query (the fallback leg). Filter, when
// non-nil, scopes every leg.
type HybridQuery struct {
	Dense          []float64
	Sparse         *SparseVector
	Filter         *QueryFilter
	Limit          int
	PrefetchLimit  int     // per-leg shortlist before fusion (e.g. 50)
	ScoreThreshold float64 // dense-only threshold; ignored for fused queries
	Fusion         string  // "rrf" (empty => dense-only)
}

// SearchResult is THE result type for the whole read path — there is no
// separate "hit" type. It starts life as a raw vector-store hit (only ID,
// Score, Payload set), is reordered by ApplyRerank, filtered by
// ApplyRelevanceFloor, weak-labeled by LabelWeakForMethod (which sets Weak), and finally
// projected in place by an adopter's Projector (which fills the projection
// fields below from Payload). Carrying one type end-to-end is deliberate: it is
// why no adopter re-implements rerank reordering against a second "hit" type.
//
// The projection fields RelativePath/Score/Snippet/Path (plus ID) are the
// federation contract the search-hub registry maps for the KO `doc` leaf — keep
// their names/JSON shape stable or update the provider descriptor in lockstep.
// A non-doc adopter (e.g. cli-health commands) leaves the doc-flavored fields
// empty and reads its corpus-specific fields from Payload, which is always the
// full vector-store payload.
type SearchResult struct {
	ID      string
	Score   float64
	Payload map[string]any
	// Projection fields, filled by a Projector after rerank + floor + weak-label.
	RelativePath string
	Snippet      string
	Path         string
	SourceID     string
	// Weak is set by LabelWeakForMethod from the active regime; carried so
	// CLI and UI never re-derive (and never drift on) the weak/strong threshold.
	Weak bool
}

// ScrollItem is the per-point projection used by the reconciler to detect
// drift and ghosts without re-reading vectors. It carries both drift levels:
// SourceHash gates a whole unchanged file (§4.1), PayloadHash gates an
// individual unchanged chunk; SourceID groups a source's chunks for the
// source-level skip and for ghost deletion.
type ScrollItem struct {
	PayloadHash string
	SourceID    string
	SourceHash  string
	// ChunkTotal is how many chunks the source fanned out into when last
	// indexed; the source-level skip requires every chunk present.
	ChunkTotal int
}

// VectorStore is the Qdrant-shaped interface, generalized from the cli-health
// dense-only store to named dense+sparse vectors and the server-side hybrid
// Query API with RRF fusion. EnsureCollection is idempotent; Query dispatches
// hybrid vs dense by whether Sparse/Fusion are set.
type VectorStore interface {
	EnsureCollection(ctx context.Context, spec CollectionSpec) error
	Upsert(ctx context.Context, point Point) error
	// SetPayload refreshes a point's payload without re-embedding it. The
	// reconciler uses it to converge the source-level hash on chunks whose
	// body is unchanged but whose parent file changed elsewhere — the
	// minimal-embed path for the large documentation corpus (§4.1).
	SetPayload(ctx context.Context, id string, payload map[string]any) error
	BatchDelete(ctx context.Context, ids []string) error
	Query(ctx context.Context, q HybridQuery) ([]SearchResult, error)
	CountPoints(ctx context.Context) (int, error)
	ScrollIDs(ctx context.Context) (map[string]ScrollItem, error)
	Available(ctx context.Context) bool
}

// =============================================================================
// 5. Reranking (swappable; two impls ship from the start)
// =============================================================================

// RerankCandidate is one shortlist entry handed to a Reranker.
type RerankCandidate struct {
	ID   string
	Text string
}

// RerankScore is a Reranker's relevance score for one candidate; the service
// reorders the shortlist by descending Score.
type RerankScore struct {
	ID    string
	Score float64
}

// Reranker reorders the fused top-K shortlist. Two impls ship from the start
// (user-approved 2026-06-03), behind this one interface:
//   - LLMReranker:          DefaultRerankRole via `resource-ollama gateway
//     generate` (dependency-free, always-available fallback).
//   - CrossEncoderReranker: BAAI/bge-reranker-v2-m3 served by the dedicated
//     `reranker` TEI resource (Phase 4) via RERANKER_URL /rerank.
//
// Degradation chain (the service composes it): cross-encoder resource healthy
// -> CrossEncoderReranker; else -> LLMReranker; else -> fused (RRF) order.
// Name() feeds StatusReport.Reranker so callers can see which leg is active.
type Reranker interface {
	Name() string
	Available(ctx context.Context) bool
	Rerank(ctx context.Context, query string, candidates []RerankCandidate) ([]RerankScore, error)
}

// =============================================================================
// 6. Search service surface
// =============================================================================

// SearchMode selects the retrieval path. ModeAuto walks the fallback chain
// hybrid -> dense -> text (grep) so search never hard-fails.
type SearchMode string

const (
	ModeAuto   SearchMode = "auto"
	ModeHybrid SearchMode = "hybrid"
	ModeDense  SearchMode = "dense"
	ModeText   SearchMode = "text"
)

// ScopeKind constrains a query to the whole corpus, one scenario, or a path
// prefix (mirrors today's docsearch scopes).
type ScopeKind string

const (
	ScopeGlobal   ScopeKind = "global"
	ScopeScenario ScopeKind = "scenario"
	ScopePath     ScopeKind = "path"
)

// Scope is a scope kind plus its value (scenario name or path prefix; ignored
// for global).
type Scope struct {
	Kind  ScopeKind
	Value string
}

// Facets are the optional payload facet filters. Authority boost prefers
// maturity:active + canonicalFor matches at ranking time.
type Facets struct {
	DocType      []string
	Audience     []string
	Maturity     []string
	CanonicalFor []string
}

// SearchQuery is one search request.
type SearchQuery struct {
	Query  string
	Mode   SearchMode
	Scope  Scope
	Facets Facets
	Limit  int
}

// SearchResponse wraps results with the leg that answered and the active
// reranker, for observability and the degraded-state UI.
type SearchResponse struct {
	Results  []SearchResult
	Total    int
	Query    string
	Method   string // which leg answered: "hybrid" | "dense" | "text"
	Reranker string // active reranker Name() or "none"
}

// StatusReport describes backend availability for `search status`.
//
// Available carries a deliberate default that adopters sometimes override:
// Service.Status sets it to (Qdrant OR a text fallback exists) — "search can
// answer at all", which suits a doc corpus that degrades to grep. A corpus where
// AI search is the product (e.g. cli-health) means something stricter —
// (Ollama AND Qdrant) — and recomputes Available in its own Status wrapper,
// treating the text leg as a degradation, not "available". Decide which you mean
// and override if the default does not fit.
type StatusReport struct {
	Available            bool
	Ollama               bool
	Qdrant               bool
	Reranker             string // active reranker or "none"/"degraded"
	IndexedCount         int
	LastReconcileAt      string
	LastReconcileOutcome string
}

// The consumer-facing search surface is the concrete *Service in service.go
// (Search + Status + reindex job control). Adopters that want a test seam define
// their own narrow interface over it (KO's docSearchEngine does exactly this).
