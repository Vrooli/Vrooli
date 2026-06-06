# aisearch-go configuration & calibration

The control surface for the shared retrieval engine, and the recipe for
calibrating + validating a new adopter. This is the canonical doc any agent
reading the package should find; `README.md` points here.

The guiding principle: **score-regime calibration is automatic; there is exactly
one genuine per-corpus lever.** A new adopter gets correct weak-labeling and
flooring with zero tuning, because the package detects the active scoring regime
at runtime and picks the right bands. The one decision a human/agent owns —
*whether to rerank at all* — is documented below with a data-driven recipe.

---

## 1. The control surface (`LoadConfig("<PREFIX>")`)

Every knob is read from `<PREFIX>_<NAME>` (an empty prefix reads the bare name).
Malformed values log a warning and fall back to the default.

| Env var | Type | Default | What it trades off |
|---|---|---|---|
| `SYNC_INTERVAL` | duration | `5m` | Reconcile cadence. Lower = fresher index, more embed load. |
| `SYNC_DISABLED` | bool | `false` | Turn the background reconcile loop off entirely. |
| `RECONCILE_PARALLELISM` | int [1,16] | `4` | Embed worker pool size. Higher = faster first index, more Ollama pressure. |
| `MAX_EMBEDS_PER_TICK` | int (0=∞) | `0` | Cap embeds per tick so a first full index never starves Ollama. The 1:1 command corpus leaves it at 0; the large doc corpus sets it. |
| `QDRANT_URL` | string | `http://127.0.0.1:6333` | Qdrant address. |
| `QDRANT_API_KEY` | string | `""` | Qdrant auth. |
| `EMBED_MODEL` | string | `nomic-embed-text` | Dense embedding model. **Changing it is a deliberate re-index** (the schema guard fails loudly on a model mismatch). |
| `RERANK_ENABLED` | bool | `false` | **The one genuine lever.** See §3. |
| `RERANK_MODEL` | string | `llama3.2:3b` | LLM-fallback rerank model. Must be a *non-reasoning* instruct model (see §4). |
| `RERANK_SHORTLIST` | int [1,500] | `50` | Over-fetch depth handed to the reranker: the query pulls this many candidates (or the page size, whichever is larger) so the reranker reorders a meaningful pool before the page is sliced. Higher = better recall into the rerank; negligible latency on the cross-encoder, real cost on the LLM leg. |
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

- **`LabelWeak(leg, score) bool`** — the single home for the "weak vs strong"
  decision. The service computes it **once** and carries a `weak` bool to every
  consumer, so CLI and UI render an identical badge and never re-derive (or
  drift on) a threshold.
- **`FloorForLeg(leg, override) FloorConfig`** — returns the regime-appropriate
  `{MaxGap, HardFloor}` for the active leg. The cross-encoder floor leans on
  `HardFloor` (kill the ~0 gibberish) with a permissive `MaxGap` (don't
  relatively cut a legitimately weak-but-real answer); cosine keeps the legacy
  `{0.15, 0.35}`.

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
> does this; `FloorForLeg(resp.Reranker, override)` picks the matching band.

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

So the engine ships `RERANK_ENABLED` **default-off** (a resource-less consumer
degrades cleanly to dense order), and an adopter decides with evidence:

```bash
# 1. Register a baseline eval suite for your provider (golden queries + soft
#    expectations + score bands).
search-hub evals register --suite @your-suite.json

# 2. Run it both ways. Runs are immutable and snapshot the config that affects
#    results (active reranker leg, embed model, indexed count).
search-hub evals run your-provider.leaf --tag rerank-off
search-hub evals run your-provider.leaf --tag cross-encoder

# 3. Compare and decide.
search-hub evals compare <run_off> <run_cross>
```

Keep the lever count small and intentional: this is the high-value, low-regret
toggle; the thresholds above are not knobs (§2).

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

1. **Adopt.** Wire the engine per `README.md` (`NewDenseEngine` + a `Source`
   adapter). Reranking is off by default — you get correct dense-cosine
   weak-labeling and flooring immediately, no tuning.
2. **Calibrate.** Decide the one lever: register a `search-hub` eval suite, run
   `rerank-off` vs `cross-encoder`, `evals compare`, and set `RERANK_ENABLED`
   from the delta (§3). The regime thresholds need no per-scenario tuning.
3. **Validate.** Store the tagged runs as immutable evidence. Re-run the suite
   after any retrieval change to catch regressions. The thresholds baked into
   the package were themselves chosen from this eval matrix on the live
   cli-health corpus; an adopter whose corpus disagrees overrides via the
   `RELEVANCE_*` env vars with its own run IDs as justification.
