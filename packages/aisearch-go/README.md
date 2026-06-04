# aisearch-go

The shared retrieval engine for Vrooli scenarios that need semantic search.
One place owns embedding, chunking, two-level drift reconciliation, hybrid
(dense + sparse) vector search, and reranking — so a scenario composes the same
primitives instead of re-rolling a ~80%-copied engine.

- **Module:** `github.com/vrooli/aisearch-go`
- **Provenance:** extracted by the Knowledge Observatory search cutover
  (`docs/plans/knowledge-observatory-search-cutover-plan.md`).
- **First adopter / worked example:** `scenarios/cli-health` (commands corpus).
- **Federation:** the `search-hub` router consumes providers built on this
  engine; it never imports the engine. They meet only at the provider contract.

## The shape: everything is an injectable seam

Production wires the concrete; tests inject fakes. Hold each seam as the
interface, never the impl.

| Seam | Interface | Production impl | Notes |
|---|---|---|---|
| Embedding | `Embedder` | `NewEmbedder(model)` (`resource-ollama gateway embed`) | dense vector for text |
| Sparse encode | `SparseEncoder` | `NewBM25SparseEncoder()` (local, model-free) | hybrid only; Qdrant applies IDF |
| Vector store | `VectorStore` | `NewVectorStore(url, key, collection)` (Qdrant) | named dense+sparse, server-side RRF |
| Sources | `Source` | per-consumer adapter | one `SourceDoc` per indexable unit |
| Chunking | `Chunker` | `NewIdentityChunker()` / markdown | 1→1 (commands) or 1→N (docs) |
| Embed text | `EmbeddingTextComposer` | `NewIdentityComposer()` / contextual | nil ⇒ identity (embeds `Body`) |
| Reranking | `Reranker` | `NewCrossEncoderReranker()`, `NewLLMReranker(model)`, `NewRerankerChain(...)` | cross-encoder → llm → fused |

`Reconciler` ties a set of `SourceBinding`s to the store and computes/apply the
drift between each source and its collection.

## Quick start (dense-only, single-chunk — the common case)

```go
store := aisearch.NewVectorStore(cfg.QdrantURL, cfg.QdrantAPIKey, "my-collection")
embedder := aisearch.NewEmbedder(cfg.EmbedModel)

// Collapses the 7-field SourceBinding literal into one call.
binding := aisearch.NewDenseBinding("mykind", "myscenario:", store, mySource)
rec := aisearch.NewReconciler(embedder, []aisearch.SourceBinding{binding}, cfg.ReconcileParallelism)

// Once at startup — creates the collection + records the schema sentinel.
spec := aisearch.CollectionSpec{
    Name: "my-collection", DenseSize: aisearch.DefaultVectorSize,
    DenseDistance: aisearch.DefaultDenseDistance, Model: aisearch.DefaultEmbedModel,
}
if err := store.EnsureCollection(ctx, spec); err != nil { /* degrade */ }
```

A consumer that fans out into many chunks (markdown docs) or needs hybrid sparse
retrieval keeps the full `SourceBinding` literal (set `Chunker`, `Composer`,
`Sparse`).

## Adopting a new scenario

A dense-only, single-chunk scenario (commands, records, short cards) wires the
engine in five steps. Follow the worked example in `scenarios/cli-health`.

1. **Pick an ID prefix + a collection name.** The prefix namespaces your point
   IDs (`"myscenario:"`); the collection name is the Qdrant collection. Define
   both as constants so they are named once.
2. **Implement the `Source` seam.** One `SourceDoc` per indexable unit, each with
   a stable `ContentHash` (the source-level drift gate), the embeddable `Body`,
   and `Meta` for result projection / filtering. (cli-health: `command_index.go`.)
3. **Assemble the engine with `NewDenseEngine`.** It returns the embedder, store,
   reranker chain, and a `CollectionSpec` derived from `cfg`/defaults:
   ```go
   cfg := aisearch.LoadConfig("MYSCENARIO")
   engine := aisearch.NewDenseEngine(cfg, "myscenario-collection") // name appears ONCE
   binding := aisearch.NewDenseBinding("mykind", "myscenario:", engine.VectorStore, mySource)
   rec := aisearch.NewReconciler(engine.Embedder, []aisearch.SourceBinding{binding}, cfg.ReconcileParallelism)
   ```
4. **`EnsureCollection` at boot (best-effort).** Pass `engine.Spec`; on a fresh
   collection it creates the layout and arms the schema sentinel, on an existing
   one it validates the layout (and backfills a missing sentinel):
   ```go
   if err := engine.VectorStore.EnsureCollection(ctx, engine.Spec); err != nil { /* degrade */ }
   ```
5. **Drive the sync loop.** `go aisearch.NewSyncLoop("myscenario", rec, cfg).Start(ctx)`.

### Foot-guns this engine closes for you

- **One collection name.** `NewDenseEngine(cfg, name)` is the single place the
  name appears; `engine.Spec.Name` is derived from it and cross-checked against
  the store by `EnsureCollection` (a disagreeing `CollectionSpec.Name` is a loud
  error, never a silent mis-target).
- **Model guard auto-arms.** A collection migrated in before the schema guard
  shipped has no meta sentinel, so a model swap could once corrupt it silently.
  `EnsureCollection` now backfills the sentinel (recording `spec.Model`) on the
  next boot for any layout-compatible collection, so a later swap fails loudly.
  ⚠ If you fork a *foreign* collection embedded with a different model at the
  same dimension, drop+recreate instead of relying on backfill — the backfill
  presumes the collection was embedded with `spec.Model`.

## On-disk contract (do not break silently)

These are byte-stable so a live collection is never silently re-embedded:

- **Point IDs** — deterministic UUIDv5 from `PointIDFor(prefix, sourceID, index, total)`.
  Single-chunk sources keep the un-suffixed ID.
- **Payload keys** — `payload_hash` (chunk drift), `source_hash` (source drift),
  `source_id`, `chunk_index`, `chunk_total`, `body` (retrievable text).
- **Vectors** — a named `dense` vector (even dense-only consumers); optional
  named `sparse` vector with the `idf` modifier.

### Schema-mismatch guard + remediation

`EnsureCollection` inspects a pre-existing collection and fails loudly with
`*CollectionSchemaMismatchError` (`errors.Is(err, ErrCollectionSchemaMismatch)`)
when the on-disk layout disagrees with the requested `CollectionSpec`: a legacy
unnamed vector, wrong dense size/distance, sparse presence mismatch, or a
recorded embedding **model** that differs from `spec.Model`.

The model + layout are recorded on a per-collection **meta sentinel** point
(marked by the reserved `__aisearch_meta__` payload key, excluded from both
search results and reconcile drift). A collection created before this guard has
no sentinel, so the model check is skipped (the vector-layout checks still run).

The guard **never auto-drops** — data loss is operator-initiated. To remediate a
mismatch:

```bash
# A model/dimension swap is a deliberate, data-losing re-index.
curl -X DELETE "$QDRANT_URL/collections/<name>"
# then restart the scenario so EnsureCollection recreates + reindexes.
```

## Relevance floor (WS2)

`ApplyRelevanceFloor(hits, FloorConfig{MaxGap, HardFloor})` drops weak/garbage
hits without hiding correct answers to legitimately-sparse queries: it always
keeps the top hit, then drops any hit below `max(topScore-MaxGap, HardFloor)`.
The gap is query-adaptive (a fixed floor would hide correct weak-but-real
answers, which overlap the gibberish band). Defaults:
`DefaultRelevanceMaxGap` (0.15), `DefaultRelevanceHardFloor` (0.35). Consumers
tune them via `LoadConfig` (`<prefix>_RELEVANCE_MAX_GAP` / `_RELEVANCE_HARD_FLOOR`).

## Reranking (WS4)

`NewRerankerChain(crossEncoder, llm)` routes a rerank call to the first available
leg (`chain.Active(ctx)`); when none is reachable it returns `(nil, nil)` so the
caller keeps the upstream fused/dense order — reranking is a pure addition.
Surface the active leg via `chain.ActiveName(ctx)`. `<prefix>_RERANK_MODEL`
selects the LLM-fallback model (`DefaultRerankModel`); the cross-encoder URL is
resolved from the reranker resource's own `RERANKER_URL` env.

### Default off, opt in per consumer — reranking is corpus-dependent

`<prefix>_RERANK_ENABLED` stays **default-off engine-wide**; whether a consumer
flips it on is a per-corpus call, because what reranking buys depends on the
corpus and the *axis* you measure:

- **Precision / junk-rejection corpora (e.g. cli-health commands):** big win.
  In a live A/B (`rec-2f91e95c6f9ed648`) the cross-encoder
  (`bge-reranker-v2-m3`) drove gibberish queries from cosine ~0.50–0.55 (a
  confident-looking full page) down to ~0.0, while strong queries were
  unchanged — at **no measurable latency cost** (embedding dominates the
  request). Floor-only tuning can't replace it here: gibberish (0.50–0.55)
  *overlaps* weak-but-real (0.54–0.70) in the dense band, so raising `HardFloor`
  would also hide legitimate sparse answers.
- **Recall corpora (e.g. knowledge-observatory docs):** no measurable gain.
  Hybrid RRF + authority boosting tied the cross-encoder on recall@5 and beat
  the LLM reranker (KO cutover §6.7), so KO docs run rerank-off for the doc leaf
  and keep the chain only for harder/federated corpora.

So: enable it where junk-rejection / precision matters, leave it off where a
strong hybrid retrieval already maximizes recall, and keep the engine default
off so a resource-less consumer degrades cleanly to dense order.

> **Contract — floor *after* rerank.** `ApplyRelevanceFloor` operates on
> whatever scores you hand it. Flooring the dense/fused scores *before*
> reranking lets gibberish (which lands at ~0.0 after rerank) still fill the
> page — the reranker rescores the survivors but the result **count** never
> collapses. So a consumer that reranks **must** apply the floor *after* the
> rerank step, on the reranked scores. cli-health now does this (the WS4 rerank
> runs before the WS2 floor); `cap-fabecce56b518120` is closed.

## Config knobs (`LoadConfig("<PREFIX>")`)

`<PREFIX>_` + `SYNC_INTERVAL`, `SYNC_DISABLED`, `RECONCILE_PARALLELISM`,
`MAX_EMBEDS_PER_TICK`, `QDRANT_URL`, `QDRANT_API_KEY`, `EMBED_MODEL`,
`RELEVANCE_MAX_GAP`, `RELEVANCE_HARD_FLOOR`, `RERANK_ENABLED`, `RERANK_MODEL`.

## Measuring adoption quality (search-hub evals)

Adopting this engine is step one; knowing whether retrieval is actually *good*
for your corpus is step two. Register a per-provider **baseline eval suite** in
`search-hub` (golden queries + soft expectations + score bands), run it with an
experiment tag, and compare runs over time:

```bash
search-hub evals register --suite @your-suite.json
search-hub evals run your-provider.leaf --tag rerank-off
search-hub evals run your-provider.leaf --tag cross-encoder
search-hub evals compare <run_a> <run_b>
```

Runs are immutable and snapshot the config that affects results (active reranker
leg, embed model, indexed count), so a before/after — e.g. proving the
cross-encoder collapses gibberish leakage while leaving strong queries intact —
is a stored, tagged artifact, not a one-off. See
`scenarios/search-hub/README.md` (the eval domain) for the full recipe.

## Canonical worked example

`scenarios/cli-health/api/internal/aisearch/` binds this engine to the
cross-scenario CLI-command corpus: `command_index.go` is the `Source` adapter +
result projection, `service.go` is the search/reindex surface, `main.go` is the
production wiring. It is the reference an adopting scenario should copy.
