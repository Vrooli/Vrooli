package aisearch

import (
	"context"
	"fmt"

	pkg "github.com/vrooli/aisearch-go"
)

// Options configure a documentation Indexer. Embedder and VectorStore are
// injected so tests can fake them; ScenariosRoot locates the corpus.
type Options struct {
	Embedder      pkg.Embedder
	VectorStore   pkg.VectorStore
	ScenariosRoot string
	// Parallelism bounds concurrent embeds (defaults applied by the engine).
	Parallelism int
	// MaxEmbedsPerTick caps embeds per reconcile pass; 0 = unlimited. The first
	// full-repo index can run uncapped via a one-shot Reindex; the background
	// loop (Phase 6) sets a budget so it never starves Ollama (plan §4.2).
	MaxEmbedsPerTick int
}

// Indexer owns the documentation indexing half of KO search: the vrooli-docs
// collection spec, the manifest-driven hybrid (dense+sparse) source binding,
// and the shared reconciler. The hybrid search read path (Phase 5) and the
// Connect/CLI surface + background sync loop (Phase 6) build on top of it.
type Indexer struct {
	embedder    pkg.Embedder
	vectorStore pkg.VectorStore
	source      *DocSource
	reconciler  *pkg.Reconciler
	spec        pkg.CollectionSpec
}

// NewIndexer wires the documentation indexer over the shared engine. KO indexes
// docs as multi-chunk sources (markdown chunker + contextual composer) with a
// local BM25 sparse vector for hybrid retrieval — the configuration that
// distinguishes the documentation consumer from cli-health's dense-only,
// single-chunk command index.
func NewIndexer(opts Options) (*Indexer, error) {
	if opts.VectorStore == nil {
		return nil, fmt.Errorf("aisearch: vector store is required")
	}
	source, err := NewDocSource(opts.ScenariosRoot)
	if err != nil {
		return nil, err
	}

	binding := pkg.SourceBinding{
		Kind:     DocKind,
		Store:    opts.VectorStore,
		Source:   source,
		Chunker:  NewMarkdownChunker(),
		Composer: NewContextualComposer(),
		Sparse:   pkg.NewBM25SparseEncoder(),
		IDPrefix: IDPrefix,
	}
	rec := pkg.NewReconciler(opts.Embedder, []pkg.SourceBinding{binding}, opts.Parallelism)
	rec.MaxEmbedsPerTick = opts.MaxEmbedsPerTick

	return &Indexer{
		embedder:    opts.Embedder,
		vectorStore: opts.VectorStore,
		source:      source,
		reconciler:  rec,
		spec: pkg.CollectionSpec{
			Name:           DefaultCollection,
			DenseSize:      pkg.DefaultVectorSize,
			DenseDistance:  pkg.DefaultDenseDistance,
			Sparse:         true,
			SparseModifier: pkg.DefaultSparseModifier,
			Model:          pkg.DefaultEmbedModel,
		},
	}, nil
}

// CollectionSpec exposes the named-vector layout (dense + idf sparse) so the
// startup path and tests can assert/recreate it.
func (i *Indexer) CollectionSpec() pkg.CollectionSpec { return i.spec }

// Reconciler exposes the underlying reconciler for the Phase-6 sync loop.
func (i *Indexer) Reconciler() *pkg.Reconciler { return i.reconciler }

// EnsureCollection creates the vrooli-docs collection (named dense+sparse
// vectors with the idf modifier) if absent. Idempotent. The legacy 1024-dim KO
// collections are dropped out-of-band before first start (plan §3.2 / Phase 3).
func (i *Indexer) EnsureCollection(ctx context.Context) error {
	return i.vectorStore.EnsureCollection(ctx, i.spec)
}

// ReindexResult summarizes one reconcile pass.
type ReindexResult struct {
	DryRun    bool
	Planned   int      // chunks to upsert
	Deletes   int      // ghost chunks to delete
	Upserted  int      // chunks embedded+written
	Deleted   int      // chunks removed
	Refreshed int      // payloads refreshed without re-embedding
	Deferred  int      // upserts deferred past the per-tick budget
	Errors    []string // per-item errors (non-fatal)
}

// Reindex runs one synchronous reconcile pass over the documentation corpus
// (Plan, then Apply unless dryRun). This is the Phase-3 indexing entrypoint;
// the Phase-6 surface adds async job control and the background loop on top of
// the same reconciler. EnsureCollection must have run first.
func (i *Indexer) Reindex(ctx context.Context, dryRun bool) (*ReindexResult, error) {
	plan, err := i.reconciler.Plan(ctx)
	if err != nil {
		return nil, fmt.Errorf("plan: %w", err)
	}
	res := &ReindexResult{DryRun: dryRun}
	for _, c := range plan.Collections {
		res.Planned += len(c.ToUpsert)
		res.Deletes += len(c.ToDelete)
		res.Refreshed += len(c.ToRefresh)
	}
	if dryRun {
		return res, nil
	}

	apply, err := i.reconciler.Apply(ctx, plan)
	if err != nil {
		return nil, fmt.Errorf("apply: %w", err)
	}
	for _, c := range apply.Collections {
		res.Upserted += c.Upserted
		res.Deleted += c.Deleted
	}
	res.Deferred = apply.Deferred
	for _, e := range apply.Errors {
		res.Errors = append(res.Errors, fmt.Sprintf("%s/%s %s: %s", e.Kind, e.Name, e.Op, e.Err))
	}
	return res, nil
}
