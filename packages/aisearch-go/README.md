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
retrieval uses `NewHybridEngine` + `NewHybridBinding` instead (see *Adopting a
new scenario → Hybrid* below) — symmetric with the dense path, and `Spec.Sparse`
can't be silently wrong.

## Adopting a new scenario

The recipe below ends at a **query returning hits** — not at indexing. There are
two shapes; pick by corpus. Worked examples: `scenarios/cli-health` (dense,
commands) and `scenarios/knowledge-observatory` (hybrid, markdown docs).

### Dense, single-chunk (commands, records, short cards)

```go
cfg := aisearch.LoadConfig("MYSCENARIO")

// 1. Assemble the engine — the collection name appears ONCE.
engine := aisearch.NewDenseEngine(cfg, "myscenario-collection")

// 2. Implement the Source seam (one SourceDoc per unit: stable ContentHash, the
//    embeddable Body, and Meta for projection/filtering) and bind it.
binding := aisearch.NewDenseBinding("mykind", "myscenario:", engine.VectorStore, mySource)
rec := aisearch.NewReconciler(engine.Embedder, []aisearch.SourceBinding{binding}, cfg.ReconcileParallelism)

// 3. EnsureCollection at boot — use EnsureCollectionForBinding so a dense/hybrid
//    mismatch is a loud error, never a silent dense-only collection.
if err := aisearch.EnsureCollectionForBinding(ctx, engine.VectorStore, binding, engine.Spec); err != nil { /* degrade */ }

// 4. Drive the sync loop (keeps the index converged with the source).
go aisearch.NewSyncLoop("myscenario", rec, cfg).Start(ctx)

// 5. Construct the shared read-path Service. Project maps a payload to your hit
//    shape; RerankText is the passage the reranker scores; TextFallback is your
//    offline keyword leg. Rerank is ON iff a chain is wired AND RerankEnabled.
svc := aisearch.NewService(aisearch.ServiceOptions{
    Embedder:      engine.Embedder,
    VectorStore:   engine.VectorStore,
    Reranker:      engine.Reranker,
    Reconciler:    rec,
    RerankEnabled: cfg.RerankEnabled, // set from the stored eval A/B (see below)
    ApplyFloor:    true,              // dense scores are 0..1 → the regime floor fits
    Threshold:     cfg.RelevanceHardFloor,
    RerankText:    func(r aisearch.SearchResult) string { return myCandidateText(r.Payload) },
    TextFallback:  myGrepLeg, // func(ctx, SearchQuery) ([]SearchResult, error)
})

// 6. Query — this returns ranked, weak-labeled, floored, projected hits.
resp, err := svc.Search(ctx, aisearch.SearchQuery{Query: "restart a scenario", Limit: 10})
//    resp.Results[i].Score / .Weak / .Payload  → project to your wire type.
//    svc.Status(ctx) and svc.Reindex(ctx, "", false) expose status + reindex jobs.
```

### Hybrid, fan-out (markdown docs, papers, records with sections)

Same shape, but `NewHybridEngine` (sparse + RRF fusion, `Spec.Sparse` set for
you) and `NewHybridBinding` (wires the shared `NewMarkdownChunker` +
`NewContextualComposer`):

```go
engine := aisearch.NewHybridEngine(cfg, "myscenario-docs")
binding := aisearch.NewHybridBinding("doc", "myscenario:", engine.VectorStore, mySource,
    aisearch.NewMarkdownChunker(), aisearch.NewContextualComposer(), engine.SparseEncoder)
rec := aisearch.NewReconciler(engine.Embedder, []aisearch.SourceBinding{binding}, cfg.ReconcileParallelism)
if err := aisearch.EnsureCollectionForBinding(ctx, engine.VectorStore, binding, engine.Spec); err != nil { /* degrade */ }
go aisearch.NewSyncLoop("myscenario", rec, cfg).Start(ctx)

svc := aisearch.NewService(aisearch.ServiceOptions{
    Embedder: engine.Embedder, SparseEncoder: engine.SparseEncoder, VectorStore: engine.VectorStore,
    Reranker: engine.Reranker, Reconciler: rec,
    RerankEnabled: cfg.RerankEnabled,
    ApplyFloor:    true, // safe: a rerank-off hybrid leg is classified into the fusion floor band
                         // (relative MaxGap only, no absolute HardFloor) by retrieval method
    Project:       myDocProjector, // fills RelativePath/Snippet/Path from payload
    Filter:        func(q aisearch.SearchQuery) *aisearch.QueryFilter { return myScopeFilter(q) },
    PostFilter:    myPathScopeTrim,  // optional client-side scope (e.g. exact path prefix)
    Decorate:      myAuthorityBoost, // optional late score nudge
    RerankText:    myDocRerankText,
    TextFallback:  myGrepLeg,
})
resp, _ := svc.Search(ctx, aisearch.SearchQuery{Query: "how do I deploy a scenario", Mode: aisearch.ModeAuto, Limit: 10})
```

The two cliffs the assemblers close: `NewHybridEngine` can't forget
`Spec.Sparse=true`, and `EnsureCollectionForBinding` turns a hybrid-binding /
dense-spec mismatch into a boot error instead of silently dropping the sparse leg.

### 7. Ship `search.json` + let the sweep set the tuning

Author the provider's `.vrooli/search.json` (descriptor + `tuning` + a small
golden `tests` corpus — start from `CommandCorpusTuning()` / `DocCorpusTuning()`)
and wire the engine from it with `NewServiceForTuning(provider.ResolvedTuning(),
deps)` instead of hand-picking `NewDenseEngine` vs `NewHybridEngine` (the
`engine` field decides, by data). The scenario self-registers the file with
search-hub at boot; the search-hub **sweep** then runs the rerank-on/off (and
other) arms and writes the winning `tuning` back — see
[`docs/reference/search-json.md`](docs/reference/search-json.md) for the schema +
factor dashboard and *Measuring adoption quality* below. Until you sweep, default
rerank OFF for fused/doc corpora and ON for dense precision corpora.

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
answers, which overlap the gibberish band).

The *band* is **regime-aware**: don't hand a hard-coded `FloorConfig` — call
`FloorForMethodLeg(method, activeLeg, override)` (the `Service` does this for
you), which returns the cross-encoder / llm / **fusion** / cosine defaults for
the retrieval method + active reranker leg. The cross-encoder floor leans on
`HardFloor` to kill ~0 gibberish with a permissive `MaxGap`; cosine keeps
`{0.15, 0.35}`; the **fusion** band (a rerank-off RRF hybrid leg — distinguished
by method, since both it and dense report `Reranker=="none"`) keeps only the
relative `MaxGap` and **disables the absolute HardFloor**, because a fused score
is an uncalibrated rank signal, not a 0..1 relevance probability. This is why
`ServiceOptions.ApplyFloor` is safe to leave ON even for a fused/doc corpus — it
now means "run the floor at all", not "which band". `override` is the consumer's
`LoadConfig`-derived `{RelevanceMaxGap, RelevanceHardFloor}`, which default to `0`
("unset" → use the regime default) and only override when an operator sets the
env var. See [`docs/reference/configuration.md`](docs/reference/configuration.md) §2.

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

## Config knobs

> **The tuning SSOT is `.vrooli/search.json`**, not the environment. The search
> *factors* — `engine`, `embed_model`, `embed_task_prefix`, `rerank_enabled`,
> `rerank_blend`, `rerank_shortlist`, `floor.{max_gap,hard_floor}` — live in the
> scenario-owned `tuning` block (schema + dashboard:
> [`docs/reference/search-json.md`](docs/reference/search-json.md)) and are read
> via `NewServiceForTuning`. The env vars below are **wiring/operational** config
> plus unset-by-default operator *overrides* of those factors.

`LoadConfig("<PREFIX>")` reads `<PREFIX>_` + `SYNC_INTERVAL`, `SYNC_DISABLED`,
`RECONCILE_PARALLELISM`, `MAX_EMBEDS_PER_TICK`, `QDRANT_URL`, `QDRANT_API_KEY`,
`EMBED_MODEL`, `EMBED_TASK_PREFIX`, `RERANK_ENABLED`, `RERANK_BLEND`,
`RERANK_MODEL`, `RERANK_SHORTLIST`, `RELEVANCE_MAX_GAP`, `RELEVANCE_HARD_FLOOR`.

**Two recall levers added 2026-06-07 (measured +0.20 recall@5 on cli-health
commands, no precision loss — see the retrospective):**
- `EMBED_TASK_PREFIX` (default off) — embed queries with `search_query:` and
  passages with `search_document:` for an asymmetric model (nomic-embed-text).
  A large retrieval win for terse corpora matched by natural-language queries.
  Flipping it on changes the embedding space; the drift hash is recipe-aware, so
  the next reconcile **auto-re-embeds** the corpus (a one-time cost). Leave it off
  for a guarded/symmetric baseline until you have re-measured it.
- `RERANK_BLEND` (default off; requires `RERANK_ENABLED`) — fuse the reranker
  order with the retrieval order via RRF instead of letting the reranker reorder
  outright. A pure-reorder cross-encoder can *bury* a strongly-retrieved canonical
  result beneath literal-token lookalikes (measured −0.20 recall); blending keeps
  the reranker's junk rejection while preserving retrieval recall. If you run a
  rerank on a corpus where recall matters, prefer blend. The blend uses
  `ApplyRerankRRF` with the `ServiceOptions.RerankRRFK` fusion constant (≤0 →
  `DefaultRRFK = 60`, the value from the original RRF paper). Raise it to flatten
  rank-position contribution; lower it to sharpen it. In practice the default is
  the right starting point. Weak labeling on this path is judged from the
  reranker's RAW calibrated scores (in the leg's own regime), not the blended
  magnitude: a blended score is a rank-fusion signal (~2/(K+1) at the top), so
  comparing it against any absolute band mislabels everything — the 2026-06
  web-search regression where every hit, near-exact matches included, rendered
  "(weak)". Hits the reranker did not score are labeled weak.

**The full control surface — every knob, its range, default, and tradeoff, plus
the regime-keyed weak/floor calibration (and why the thresholds are NOT levers),
the one real lever `RERANK_ENABLED` + its decision recipe, and the
adopt → calibrate → validate flow — lives in
[`docs/reference/configuration.md`](docs/reference/configuration.md). Read that
before tuning anything.** `RELEVANCE_MAX_GAP` / `RELEVANCE_HARD_FLOOR` are
*overrides* (default 0 → the package picks the regime-appropriate band via
`FloorForMethodLeg(method, leg, override)`); the weak-match decision is owned by
the regime-aware `LabelWeakForMethod(method, leg, score)`.

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

### Worked example: the rerank-on-docs decision (the tuning loop)

The rerank on/off choice is per-corpus and must be *measured*, not assumed — and
it is now a `tuning.rerank_enabled` value in `search.json`, swept and written
back, not an env flag flipped across restarts. The loop, end to end:

```bash
# Query-time arms (rerank on/off × blend on/off) are explored full-factorial via
# per-request overrides — NO reindex, no restart. The sweep runs the provider's
# golden suite under each, stores one immutable tagged run per arm, and (with
# --apply) writes the winning tuning back into search.json — but only if it
# clears the four overfit guards (significance, held-out, constraints,
# complexity tie-break). A within-noise result changes nothing.
search-hub evals sweep knowledge-observatory.docs.starter --query-time-only --apply
```

For KO docs the measured finding is that rerank buys ordering parity, not recall,
so the winner is rerank-OFF and the sweep leaves `search.json` unchanged. The
decision is justified by the stored, tagged arms — never hard-coded in a
constructor. (Inspect a single pair by hand with `evals register` / `evals run
--tag` / `evals compare` when you want to eyeball the score-regime difference —
RRF ~0.01 vs cross-encoder ~0..1 — that the sweep handles rank-wise for you.)

**One SSOT, two surfaces — do not build a third.** The search-hub `eval` domain
is the single source of truth for *A/B and tuning*: it is where you justify a
config decision (rerank on vs off, a floor band, an embed-model swap) with a
stored, tagged, comparable run. A scenario *may* also keep a thin **recall@k
per-build gate** (e.g. KO's `TestAccuracyCorpus`) — but that gate is a smoke
check that the corpus still resolves its own goldens, NOT a second eval system:
it must not re-implement A/B comparison, run storage, or experiment tagging.
When you want to *change* a knob, run `evals sweep` (or register/compare by hand)
in search-hub and let it write the winning `tuning` back into `search.json`. When
you want to *guard* a build, the local recall gate is enough. The tuning loop is:
golden suite in `search.json` → `evals sweep --apply` → the four overfit guards →
winning `tuning` written back into `search.json` (the SSOT).

## Canonical worked example

`scenarios/cli-health/api/internal/aisearch/` binds this engine to the
cross-scenario CLI-command corpus: `command_index.go` is the `Source` adapter +
result projection, `service.go` is the search/reindex surface, `main.go` is the
production wiring. It is the reference an adopting scenario should copy.
