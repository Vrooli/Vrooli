# aisearch-go configuration & calibration

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

These are the **wiring/operational** knobs (sync cadence, Qdrant address,
reranker resource endpoints) plus the override forms of the search factors. Each
is read from `<PREFIX>_<NAME>` (an empty prefix reads the bare name); malformed
values log a warning and fall back to the default. The factor values themselves
are owned by `search.json` (above) — a migrated adopter ignores the factor env
reads below.

| Env var | Type | Default | What it trades off |
|---|---|---|---|
| `SYNC_INTERVAL` | duration | `5m` | Reconcile cadence. Lower = fresher index, more embed load. |
| `SYNC_DISABLED` | bool | `false` | Turn the background reconcile loop off entirely. |
| `RECONCILE_PARALLELISM` | int [1,16] | `4` | Embed worker pool size. Higher = faster first index, more Ollama pressure. |
| `MAX_EMBEDS_PER_TICK` | int (0=∞) | `0` | Cap embeds per tick so a first full index never starves Ollama. The 1:1 command corpus leaves it at 0; the large doc corpus sets it. |
| `QDRANT_URL` | string | `http://127.0.0.1:6333` | Qdrant address. |
| `QDRANT_API_KEY` | string | `""` | Qdrant auth. |
| `EMBED_MODEL` | string | `nomic-embed-text` | Dense embedding model. **Changing it is a deliberate re-index** (the schema guard fails loudly on a model mismatch). |
| `EMBED_TASK_PREFIX` | bool | `false` | Opt into asymmetric task-instruction prefixes (`search_query:`/`search_document:`) for models like nomic-embed-text. Measured +0.20 recall@5. Flipping it changes the embedding space and triggers an automatic full re-index on the next reconcile tick. Leave off for a guarded/symmetric baseline until you have measured it. |
| `RERANK_ENABLED` | bool | `false` | **The one genuine lever.** See §3. |
| `RERANK_BLEND` | bool | `false` | Fuse the reranker order with the retrieval order via `ApplyRerankRRF` instead of letting the reranker order win outright. Prevents a strongly-retrieved canonical result from being buried by literal-token lookalikes (measured +0.20 recall on the cli-health command corpus, no precision loss). Requires `RERANK_ENABLED`. The RRF fusion constant is `ServiceOptions.RerankRRFK` (≤0 → `DefaultRRFK = 60`). |
| `RERANK_MODEL` | string | `llama3.2:3b` | LLM-fallback rerank model. Must be a *non-reasoning* instruct model (see §4). |
| `RERANK_SHORTLIST` | int [1,500] | `50` | Over-fetch depth handed to the reranker: the query pulls this many candidates (or the page size, whichever is larger) so the reranker reorders a meaningful pool before the page is sliced. Higher = better recall into the rerank; negligible latency on the cross-encoder, real cost on the LLM leg. |
| `RERANKER_URL` | string | `""` (falls back to resource env) | Cross-encoder reranker endpoint. When empty, the reranker resource's own unprefixed env (`RERANKER_BASE_URL`/`RERANKER_URL`/`RERANKER_HOST+PORT`) is used — preserving zero-config local use. Lets two scenarios on one host point at different reranker instances. |
| `RERANKER_MODEL` | string | `""` (falls back to resource env) | Cross-encoder model identifier read by the reranker resource. Distinct from `RERANK_MODEL` (which selects the LLM *fallback* leg). When empty, the resource's own env applies. |
| `RELEVANCE_MAX_GAP` | float | `0` (unset → regime) | **Override only.** See §2. |
| `RELEVANCE_HARD_FLOOR` | float | `0` (unset → regime) | **Override only.** See §2. |

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
`RELEVANCE_*` env vars exist **only as overrides** for an operator who has eval
evidence that a specific corpus wants a different band; they default to `0`
("unset"), in which case `FloorForLeg` supplies the regime default. A non-zero
`MaxGap` or non-zero `HardFloor` wins per-field (set `HardFloor` negative to
deliberately disable the garbage floor).

> Contract reminder: **floor *after* rerank.** A consumer that reranks must
> apply `ApplyRelevanceFloor` on the *reranked* scores (the reranker drives junk
> to ~0; flooring before rerank would let it still fill the page). cli-health
> does this; `FloorForMethodLeg(method, resp.Reranker, override)` picks the
> matching band.

---

## 3. The one real lever — `RERANK_ENABLED` + its decision recipe

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
  answered with HTTP 413) and the scores merged. This decouples `RERANK_SHORTLIST`
  from the server limit — a shortlist of 50 reranks correctly as two chunks.
  Without this, a shortlist above 32 silently fell back to dense order while
  still *reporting* the cross-encoder as active — a regime mislabel.
- **`RERANK_MODEL` must be a non-reasoning instruct model.** Measured 2026-06-05:
  `qwen3:4b` burns its whole token budget on a `<think>` preamble (~38 s/query)
  even with the `/no_think` hack; `llama3.2:3b` emits clean, complete,
  correctly-ordered listwise JSON at ~2.8 s warm. The default is therefore
  `llama3.2:3b`, making the LLM leg a viable CPU-only fallback for hosts without
  the cross-encoder GPU resource. The score parser still strips `<think>` blocks
  defensively in case a consumer points `RERANK_MODEL` at a reasoning model.

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
   `tuning.floor.*` fields (or the `RELEVANCE_*` env override) with its own run
   IDs as justification.
