# AI Go Search configuration & calibration

The control surface for the shared retrieval engine, and the recipe for
calibrating + validating a new adopter. This is the canonical doc any agent
reading the package should find; `README.md` points here.

The guiding principle: **score-regime calibration is automatic; there is exactly
one genuine per-corpus lever.** A new adopter gets correct weak-labeling and
flooring with zero tuning, because the package detects the active scoring regime
at runtime and picks the right bands. The one decision a human/agent owns —
*whether to rerank at all* — is documented below with a data-driven recipe.

> **The tuning SSOT is `.vrooli/search.json`, not the environment.** The genuine
> search *factors* (engine shape, embedding recipe, rerank policy, floor band)
> live in the scenario-owned `tuning` block — see
> [`search-json.md`](search-json.md) for the schema + the full factor dashboard
> (tier, default, decision rule per knob). A migrated adopter (cli-health, KO)
> reads them via `NewServiceForTuning(tuning, deps)` and the `search-hub` sweep
> writes them back; nothing search-tunable is a Go literal or an env-var-that-is-
> the-SSOT. The env vars in §1 are now **wiring/operational** config plus
> unset-by-default operator *overrides* — not the source of truth.

---

## 1. The operational control surface (`LoadConfig("<PREFIX>")`)

These are the **wiring/operational** knobs ONLY: sync cadence, Qdrant address,
the deployed embed model, and the reranker resource endpoints + fallback-leg
model. Each is read from `<PREFIX>_<NAME>` (an empty prefix reads the bare name);
malformed values log a warning and fall back to the default. The search *factors*
(engine, `embed_task_prefix`, `rerank_enabled`/`rerank_blend`/`rerank_shortlist`,
the floor band) are **no longer env vars** — they are owned by `search.json` /
`TuningConfig` (above) and read via `NewServiceForTuning`; `LoadConfig` does not
read them. (`EmbedTaskPrefix` remains a `Config` *field* — the
`NewEmbedderForConfig` input — but the adopter fills it from the SSOT, not env.)

| Env var | Type | Default | What it trades off |
|---|---|---|---|
| `SYNC_INTERVAL` | duration | `5m` | Reconcile cadence. Lower = fresher index, more embed load. |
| `SYNC_DISABLED` | bool | `false` | Turn the background reconcile loop off entirely. |
| `RECONCILE_PARALLELISM` | int [1,16] | `4` | Embed worker pool size. Higher = faster first index, more Ollama pressure. |
| `MAX_EMBEDS_PER_TICK` | int (0=∞) | `0` | Cap embeds per tick so a first full index never starves Ollama. The 1:1 command corpus leaves it at 0; the large doc corpus sets it. |
| `QDRANT_URL` | string | `http://127.0.0.1:6333` | Qdrant address. |
| `QDRANT_API_KEY` | string | `""` | Qdrant auth. |
| `EMBED_ROLE` | string | `embedding.default` | Ollama policy role used for runtime embedding calls. |
| `RERANK_ROLE` | string | `rerank.llm_fallback` | Ollama policy role used for the LLM fallback reranker. |
| `RERANKER_URL` | string | `""` (falls back to resource env) | Cross-encoder reranker endpoint. When empty, the reranker resource's own unprefixed env (`RERANKER_BASE_URL`/`RERANKER_URL`/`RERANKER_HOST+PORT`) is used — preserving zero-config local use. Lets two scenarios on one host point at different reranker instances. |
| `RERANKER_MODEL` | string | `""` (falls back to resource env) | Cross-encoder model identifier read by the reranker resource. Distinct from `RERANK_ROLE` (which selects the LLM *fallback* leg). When empty, the resource's own env applies. |

### Qdrant storage profile

Set `CollectionSpec.Storage` from the scenario's governed configuration when a
corpus needs a non-default Qdrant layout. The profile exposes on-disk dense
vectors, sparse indexes, payload, and HNSW; HNSW `m`, construction effort, and
full-scan threshold; optimizer indexing threshold and worker count; scalar
quantization; and bounded upsert batch size. Zero values preserve Qdrant
defaults. Do not hide these values in adopter-specific request JSON.

Use `BatchVectorStore.UpsertBatch` for bounded writes. The Qdrant adapter clamps
the requested batch size to `MaxSourcePageSize`. Use `ReconcileParallelism` and
a shared `WeightedAdmission` budget to bound workers and total expensive work.

Large sources must implement `PagedSource`; event-driven sources should also
implement `ChangeSetSource`. Use `StreamingReconciler` with a
`GenerationStore`, so cancellation or validation failure rolls back a shadow
generation and never changes the serving alias. `NewPagedSourceAdapter` is for
small legacy sources because it intentionally materializes `LoadAll` once.

The factors that used to appear here (`EMBED_TASK_PREFIX`, `RERANK_ENABLED`,
`RERANK_BLEND`, `RERANK_SHORTLIST`, `RELEVANCE_MAX_GAP`, `RELEVANCE_HARD_FLOOR`)
now live in the `tuning` block of `search.json` — see [`search-json.md`](search-json.md)
for the per-knob dashboard, and §2/§3 below for how the floor band and the rerank
lever behave.

---

## 2. Score-regime calibration — automatic, NOT a lever

A fixed weak threshold (or floor) is fragile because the numeric meaning of a
score depends on *which scoring regime produced it*:

| Regime | Active leg (`SearchResponse.Reranker`) | Score shape | Junk lands at | Weak threshold |
|---|---|---|---|---|
| Cross-encoder | `cross-encoder:…` | sigmoid relevance 0..1 | ~0 | `< 0.30` |
| LLM | `llm:…` | listwise judge 0..1 | low (0–0.1) | `< 0.50` |
| Cosine (rerank-off) | `none` / `""` / text | dense cosine | 0.50–0.55 (overlaps weak-real!) | `< 0.55` |

The active regime is **detectable at runtime** — it is the reranker leg the
response already reports. So the thresholds are a **regime → band table that
lives once in the package** (`relevance.go`, `floor.go`); an adopter inherits
correct behavior with no configuration:

- **`LabelWeakForMethod(method, leg string, score float64) bool`** — the single
  home for the "weak vs strong" decision. The service computes it **once** and
  carries a `weak` bool to every consumer, so CLI and UI render an identical badge
  and never re-derive (or drift on) a threshold. Being method-aware, it classifies
  a rerank-off hybrid leg on the fusion band instead of the cosine band.
- **`FloorForMethodLeg(method, leg string, override FloorConfig) FloorConfig`** —
  returns the regime-appropriate `{MaxGap, HardFloor}` for the retrieval method +
  active reranker leg. The cross-encoder floor leans on `HardFloor` (kill the ~0
  gibberish) with a permissive `MaxGap` (don't relatively cut a legitimately
  weak-but-real answer); cosine keeps `{0.15, 0.35}`; the fusion band (rerank-off
  hybrid, distinguished by `method == "hybrid"`) disables `HardFloor` entirely and
  keeps only the relative `MaxGap`.

**Why these aren't levers:** the regime is auto-detected, so there is nothing
for an operator to tune per corpus — tuning would just risk mislabeling. The
`tuning.floor.{max_gap,hard_floor}` fields in `search.json` exist **only as
overrides** for an operator who has eval evidence that a specific corpus wants a
different band; they default to `0` ("unset"), in which case `FloorForMethodLeg`
supplies the regime default. A non-zero `MaxGap` or non-zero `HardFloor` wins
per-field (set `HardFloor` negative to deliberately disable the garbage floor).
They reach the read path as `ServiceOptions.Floor` (the adopter threads
`tuning.Floor.Config()`); the package no longer reads a `RELEVANCE_*` env var.

> Contract reminder: **floor *after* rerank.** A consumer that reranks must
> apply `ApplyRelevanceFloor` on the *reranked* scores (the reranker drives junk
> to ~0; flooring before rerank would let it still fill the page). cli-health
> does this; `FloorForMethodLeg(method, resp.Reranker, override)` picks the
> matching band.

---

## 3. The one real lever — `tuning.rerank_enabled` + its decision recipe

Whether to rerank **can't** be auto-derived; it is a per-corpus call, because
what reranking buys depends on the corpus:

- **Precision / junk-rejection corpora** (e.g. cli-health commands): big win —
  the cross-encoder collapses gibberish from a confident-looking ~0.5 page down
  to ~0 while leaving strong queries intact, at negligible latency.
- **Recall corpora** (e.g. knowledge-observatory docs): no measurable gain —
  hybrid RRF + authority boosting already ties the cross-encoder on recall, so
  the doc leaf runs rerank-off.

So the engine ships rerank **default-off** in the taxonomy (a resource-less
consumer degrades cleanly to dense order), the decision is persisted as
`tuning.rerank_enabled` (+ `rerank_blend`) in `search.json`, and an adopter
decides with evidence. The `search-hub` **sweep** automates the A/B and the
write-back:

```bash
# One command: enumerate the query-time arms (rerank on/off × blend on/off),
# run the provider's golden suite under each via per-request overrides, and
# (with --apply) write the winning tuning back — but ONLY if it clears the four
# overfit guards (significance, held-out, constraints, complexity tie-break).
search-hub evals sweep cli-health.commands.primary --query-time-only --apply
```

The manual A/B still exists when you want to inspect a single pair
(`evals register` / `evals run --tag` / `evals compare`), but the sweep is the
loop: it never promotes a within-noise win and it writes the result into the
tuning SSOT for you. Keep the lever count small and intentional: this is the
high-value, low-regret toggle; the thresholds above are not knobs (§2).

---

## 4. Reranker chain — legs, resilience & the LLM model

`NewRerankerChain(crossEncoder, llm)` routes a rerank call to the first
*available* leg (cross-encoder → LLM → fused order). It is the resilience
boundary:

- **`Active()` is TTL-cached behind an injected clock** (`DefaultRerankerProbeTTL`
  = 20 s). Without the cache, every query re-ran the per-leg availability probe —
  a single down leg imposed a probe on *every* query. The cache caps that to one
  probe per window and reflects an outage or recovery within one TTL. A live
  status readout uses **`ActiveUncached()`** to bypass the cache.
- **Probe timeouts are bounded** (cross-encoder `/health` and the LLM one-token
  generate both ≤ 3 s) so a down leg can never reintroduce a per-query latency
  cliff.
- **Cross-encoder requests are auto-chunked** to ≤ 32 candidates per TEI
  `/rerank` call (the server's `--max-client-batch-size`; a larger batch is
  answered with HTTP 413) and the scores merged. This decouples
  `tuning.rerank_shortlist` from the server limit — a shortlist of 50 reranks
  correctly as two chunks.
  Without this, a shortlist above 32 silently fell back to dense order while
  still *reporting* the cross-encoder as active — a regime mislabel.
- **`RERANK_ROLE` owns the LLM fallback.** Runtime model selection is resolved by
  `resource-ollama` policy instead of repeated in adopters. The score parser
  still strips `<think>` blocks defensively in case policy points the role at a
  reasoning model.

---

## 5. End-to-end: adopt → calibrate → validate

1. **Adopt.** Author `.vrooli/search.json` (descriptor + a starting `tuning`
   block + a small golden `tests` corpus — start from `CommandCorpusTuning()` or
   `DocCorpusTuning()`), then wire the engine with
   `NewServiceForTuning(provider.ResolvedTuning(), deps)` per `README.md`. The
   scenario self-registers the file with `search-hub` at boot. Reranking is off
   by default — you get correct dense-cosine weak-labeling and flooring
   immediately, no tuning.
2. **Calibrate.** Run `search-hub evals sweep <suite_id> --apply`.
   The sweep enumerates the arms, runs the golden suite under each, clears the
   four overfit guards, and writes the winning `tuning` back into `search.json`
   (reindexing if an index-time factor moved). The regime thresholds need no
   per-scenario tuning. Grow a thin corpus first with `evals generate` (adequacy
   warnings tell you when it is too thin to trust a sweep).
3. **Validate.** Each arm is a stored, immutable, tagged run snapshotting the
   config that affects results. Re-run the suite (or the scenario's thin recall
   gate) after any retrieval change to catch regressions. The thresholds baked
   into the package were themselves chosen from this eval matrix on the live
   cli-health corpus; an adopter whose corpus disagrees overrides via the
   `tuning.floor.*` fields in `search.json` with its own run IDs as justification.
