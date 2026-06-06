# aisearch-go Graduation — Retrospective

What was hard, unclear, misleading, or inconsistent while turning
`packages/aisearch-go` from an indexing engine into an adoption-ready shared
read-path library — using knowledge-observatory (hybrid/docs) and cli-health
(dense/commands) as the two lockstep adopters. Written so **adopter #3 is
trivial**. Pairs with the package `README.md` (the recipe) and
`docs/reference/configuration.md` (the levers).

## The fixes that landed (mapped to the friction)

| Friction (before) | Fix (after) |
|---|---|
| README recipe stopped at indexing — never produced a query, so adopters copied a 600-line read path | README now has TWO end-to-end recipes (dense + hybrid) that end at `svc.Search(...)` returning hits |
| `SearchHit` (projection) vs `SearchResult` (raw) split, no bridge → every adopter re-implemented rerank reordering | ONE result type: `SearchResult` carries score + `Payload` + projection fields + `Weak`. `ApplyRerank`/`ApplyRelevanceFloor` operate on it directly; `SearchHit` deleted |
| No read-path `Service` — query→floor→rerank→project→fallback + reindex jobs copy-pasted per adopter | Concrete `*Service` owns the pipeline + reindex job control; adopters pass a `Source` + function seams (`Project`/`Filter`/`PostFilter`/`Decorate`/`RerankText`/`TextFallback`) |
| No hybrid assembler; `Spec.Sparse=true` easy to forget → silent dense-only collection | `NewHybridEngine` always sets `Spec.Sparse`; `NewHybridBinding` collapses the 7-field literal; `EnsureCollectionForBinding` makes a hybrid/dense mismatch a boot error |
| Markdown `Chunker` + `ContextualComposer` stuck KO-local | Promoted to the package (`NewMarkdownChunker`/`NewContextualComposer`); KO imports them |
| Rerank hard-wired on for docs (no recall gain) | Config-gated per scenario (`<PREFIX>_RERANK_ENABLED`), justified by a stored search-hub eval A/B; default OFF for fused/doc corpora |
| Reranker read UNPREFIXED globals (two scenarios couldn't target different rerankers) | `Config.RerankerURL`/`RerankerModel`, prefix-aware, env fallback |
| Stale "Phase 0 / Phase 1 stub" docs in `doc.go` + cli-health endpoints | Reworded to production reality |
| `go build ./...` blocked by go.sum drift in KO/search-hub | Added the missing `cli-core` require+replace; all four modules build |

## The non-obvious lessons (read these before adopting)

1. **The floor regime now has a "fusion" band** (resolved post-graduation). The
   original graduation shipped only cross-encoder / llm / **cosine** bands — all
   assuming a 0..1 score — so an RRF-fused leg with rerank OFF (much smaller
   scores) would have been wrongly zeroed by the cosine `HardFloor`, and the floor
   was made opt-in via `ServiceOptions.ApplyFloor` as a stopgap (KO false,
   cli-health true). That stopgap is gone: `regimeFusion` is a real 4th regime,
   and the service classifies the rerank-off hybrid leg by **method**
   (`FloorForMethodLeg` / `LabelWeakForMethod`), not just leg name. The fusion
   band keeps only the relative `MaxGap` and **disables the absolute HardFloor**
   (a fused score is a rank signal, not a 0..1 relevance probability), so
   `ApplyFloor` is safe to leave ON for a fused/doc adopter — KO now does, and
   `TestAccuracyCorpus` confirms recall@5 is unchanged at 0.818. `ApplyFloor`
   survives only as a "no floor at all" escape hatch, not a regime workaround.
   This also fixed a **latent weak-label bug**: the rerank-off hybrid path used to
   weak-label real fused hits against the cosine 0.55 line; it now uses the fusion
   band (0.20).

2. **Over-fetch is not just for rerank.** The shortlist must widen whenever
   *anything* downstream of the query drops/reorders before the page is cut —
   rerank, a `PostFilter` (exact path scope), or a `Decorate` (authority boost).
   The `Service` triggers over-fetch on any of the three, not just rerank.

3. **"Available" means different things per corpus.** The shared default is
   `qdrant OR text-fallback` (doc search degrades to grep). cli-health means
   `ollama AND qdrant` (AI search needs both; text is a degradation, not
   "available"). Decide which you mean and override `Status` if needed.

4. **Typed projection is a boundary generic, not a generic engine** (resolved
   post-graduation). The single result type rides `Payload` for corpus-specific
   fields: doc adopters fill the named projection fields
   (`RelativePath`/`Snippet`/`Path`); command adopters keep
   `Origin`/`Group`/`Binding` in `Payload`. The clean way to hand an adopter its
   own typed hit is **`SearchTyped[H](ctx, svc, query, project)`** — a generic
   *function* that projects the finished page to `[]H` at the boundary — not a
   generic `*Service[H]`. The pipeline stages (rerank/floor/weak/post-filter) all
   operate on the uniform `SearchResult` and must run before the page is cut; only
   the final shape is corpus-specific, and making the whole engine generic would
   thread `H` through `Status`, reindex jobs, and the in-place enrichment
   `Projector` for no gain (and fracture the federation contract's single result
   type). cli-health adopts `SearchTyped` (its `Search` owns no projection loop);
   doc adopters that fill the named fields still pass `Project` and read
   `resp.Results` directly.

5. **One convention for the rerank lever** (resolved post-graduation). Both
   adopters now pass an explicit `RerankEnabled` flag (KO wires it from
   `KO_DOCS_RERANK_ENABLED`; cli-health from its config) rather than inferring it
   from a nil `Reranker`. The flag reads clearly and the search-hub A/B sets it;
   adopter #3 should follow suit.

6. **Eval ids are qdrant point UUIDs, not paths.** A search-hub eval suite that
   wants `expect_ids` must harvest the UUIDs from a live run; you cannot author
   them offline. Lead the suite with `expect_within_top_k` (scale- and
   id-invariant) — especially because the rerank-on/off A/B spans two score
   regimes, so an absolute `expect_min_score` band can't be shared across arms.

## Live verification (2026-06-05, agi)

All three scenarios restarted healthy and verified end-to-end against the live
stack (qdrant/ollama/reranker all 200):

- **cli-health** (dense): "restart a scenario" → `vrooli scenario restart` @ 0.985
  (cross-encoder active); gibberish → 1 result @ 0.0003, `weak=true`; 2253 indexed.
- **knowledge-observatory** (hybrid, rerank-OFF default): federation query →
  `method=hybrid reranker=none`, top `docs/scenarios/DEPLOYMENT.md` @ 0.56.
- **search-hub**: routes the same query to `knowledge-observatory.docs` (parses
  the preserved `docSearchHit` via `providers.MapResults`).

**KO recall gate (the behavior-preservation proof):** `TestAccuracyCorpus`,
N=22, live:

| config | recall@5 | MRR@3 |
|---|---|---|
| hybrid + none (default) | **0.818** | 0.705 |
| hybrid + cross-encoder | 0.864 | 0.689 |
| dense + rerank | 0.682 | 0.545 |

`hybrid+none` = **0.818 / 0.705**, byte-identical to the pre-graduation baseline
— the shared-Service rewire changed nothing about KO's results. **The rerank A/B
verdict:** the cross-encoder buys +0.046 recall but −0.016 MRR on docs —
marginal, and rerank-off already clears the 0.8 gate without a reranker
dependency, so **OFF stays the default for docs**. (cli-health keeps rerank ON:
its commands are a precision/junk-rejection corpus where the cross-encoder
collapses gibberish to ~0, as the live check above shows.)

## Post-graduation follow-up (2026-06-05)

The Tier-1/2/3 polish pass that closed the deferred items (all four modules stay
build/test/vet/gofumpt clean):

- **Tier 2 — fusion regime (the real fix).** `regimeFusion` + `regimeFor(method,
  leg)` + `FloorForMethodLeg`/`LabelWeakForMethod` replaced the `ApplyFloor`
  stopgap. KO flipped to `ApplyFloor: true`; `TestAccuracyCorpus` confirms
  `hybrid+none` recall@5 = **0.818 / MRR 0.705 — byte-identical** to the
  pre-flip baseline, proving the fusion floor (relative MaxGap only, HardFloor
  disabled) trims only the far tail. Also fixes the latent weak-label bug.
- **Tier 3 — ergonomics.** `SearchTyped[H]` (boundary generic) is adopted by
  cli-health; both adopters use the explicit `RerankEnabled` flag; the
  `Available` semantics divergence is documented on `StatusReport`.
- **Tier 1b — stored rerank A/B (search-hub).** Live tagged runs on the graduated
  KO (8419 docs): **rerank-off** `7284f747` (7/7 after gibberish-ceiling recal)
  and `ad5d2170` (5/7, original ceiling); **rerank-on** `39970d88` (7/7). All 5
  real cases stay within top-K in BOTH arms — the starter suite confirms the
  recall corpus verdict (rerank OFF default for docs). The gibberish ceiling was
  recalibrated from the live data: rerank-off RRF cannot separate gibberish
  (~0.56) from weak-real (~0.56) by absolute score — that separation is the
  cross-encoder's job (rerank-on collapses gibberish to ~0.00).
- **Tier 1a — cli-health recall gate: a genuine gap, NOT a calibration.** The
  live `TestCommandRecall` pass (cross-encoder healthy, 2237 cmds) measured
  recall@5 = **0.50** (0.40 before label fixes; 0.35 dense-only) — well below the
  0.8 REQ-P0-004 gate. Three labels were genuinely wrong and corrected; the rest
  are verified-canonical. The residual misses are real retrieval failures
  (canonical commands buried under fleet near-duplicates: `vrooli scenario list`
  ranks #9; vocabulary mismatch leaves `prompt-manager action create`
  unretrieved). Filed as a bug (`knw-1780702659191582434`); REQ-P0-004 stays
  `in_progress`. The corpus labels were NOT rewritten to match the buried engine
  output — the gate is meant to test desired behavior, and it correctly exposed a
  real quality gap (authority boost via the unused cli-health Decorate seam +
  index-time description enrichment is the fix, out of this read-path's scope).

## What adopter #3 does (the whole job)

1. `LoadConfig("MYSCENARIO")`; `NewDenseEngine` (or `NewHybridEngine`).
2. Implement `Source`; bind with `NewDenseBinding`/`NewHybridBinding`;
   `EnsureCollectionForBinding`; start the sync loop.
3. `NewService(ServiceOptions{...})` with a `Project` (doc-style in-place fill)
   *or* read `resp.Results` and project with `SearchTyped[H]` for a typed page
   (lesson 4); a `RerankText`; a `TextFallback`; `RerankEnabled` (explicit flag,
   lesson 5); and `ApplyFloor: true` (safe for every regime now — lesson 1).
4. Expose query/status/reindex (the `Service` already has them).
5. Register a rank-centric eval suite; run the rerank A/B; set the flag.

That is the whole job. No orchestration, no rerank reorder code, no chunker
clone, no silent-sparse cliff.
