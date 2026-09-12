# `.vrooli/search.json` — the search SSOT + control-surface dashboard

`search.json` is the **single source of truth** for everything a scenario
exposes to search: how the federated router reaches it (the *descriptor*), the
tuning knobs the optimizer reads and writes (the *tuning* block), and the
labelled corpus its quality is measured against (the *tests* block). One file,
one home, per scenario: `scenarios/<scenario>/.vrooli/search.json`.

This search package owns the file *format* because the file's centre of
gravity is the tuning factors it defines (`tuning.go`, `searchjson.go`). The
scenario owns the file *content*. The `search-hub` router consumes it (it routes
to the descriptor, sweeps the tuning, and runs the tests) but never redefines
the format.

> Nothing search-tunable is a Go literal or an env-var-that-is-the-SSOT anymore.
> The old `<PREFIX>_RERANK_*` / `<PREFIX>_EMBED_TASK_PREFIX` env reads and the
> `NewDenseEngine(...)` code literal were the source of truth before this file;
> they are now demoted to unset-by-default operator *overrides*. See
> [`configuration.md`](configuration.md) §1.

---

## 1. File shape (`SearchFile`)

```jsonc
{
  "$schema": "...",                 // optional editor/validation pointer; ignored at runtime
  "version": "1.0.0",               // required, non-empty
  "providers": [ /* ProviderConfig, ≥1, unique provider_id */ ]
}
```

Parsed by `aisearch.LoadSearchFile(path)` / `ParseSearchFile(bytes)`. Parsing is
**strict**: unknown fields are rejected (`DisallowUnknownFields`), so a typo in a
key fails loudly at boot instead of being silently ignored. `Validate()` enforces
`version` present, ≥1 provider, unique `provider_id`s, and each provider's
`tuning` against the factor taxonomy (§3).

## 2. Provider (`ProviderConfig`)

One entry per searchable corpus a scenario exposes (most scenarios expose one).

| Key | Owner | Purpose |
|---|---|---|
| `provider_id` | scenario | Stable id, `<scenario>.<leaf>` (e.g. `cli-health.commands`). Unique within the file and the registry. |
| `provider_group` | scenario | Grouping for fan-out routing (usually the scenario id). |
| `bucket` | scenario | Search-context bucket (`BUCKET_DO`/`BUCKET_REUSE`/`BUCKET_KNOW`/`BUCKET_STATE`). |
| `type` | scenario | Result kind (`command`, `doc`, …). |
| `description` | scenario | One line the router shows + the classifier reads. |
| `scope` | scenario | Visibility scope (`SCOPE_PROJECT`, …). |
| `class` | scenario | **Operability class** (`local_index` / `local_live` / `external` / `async`). Descriptor policy that drives Search Hub's control/latency posture with no scenario-name special-casing (see below). Required for active providers under maturity certification. |
| `endpoint` | descriptor | How the router calls the provider's **public search** RPC. Opaque registry `Endpoint` shape (mapped to the proto by `searchregister-go`). |
| `status_endpoint` | descriptor | How the router polls index status. |
| `index_timestamp_field` | scenario | Optional status-response field containing the last successful index/reconcile timestamp; defaults to `last_indexed_at` and accepts dotted paths such as `index.last_reconcile_at`. |
| `reindex_endpoint` | descriptor | Token-gated `SearchControlService.Reindex` target. Absent ⇒ provider is routable but **not index-time tunable**. |
| `config_endpoint` | descriptor | Token-gated `SearchControlService.WriteConfig` target — where a sweep writes a winning `tuning` block back. Absent ⇒ provider is **not config-writable** by the sweep. |
| `config_writable` / `config_pinned_reason` | scenario | Explicit non-writable posture for indexed providers whose tuning is intentionally fixed by construction. |
| `tuning` | **this package** | The factor values (§3). Read at boot; written back by the sweep. |
| `tests` | **this package** | The unified evaluation corpus (§4). |

The descriptor sub-objects (`endpoint` / `status_endpoint` / `result_mapping` /
`reindex_endpoint` / `config_endpoint`) are **opaque** to `aisearch-go` — they are
search-hub's registry vocabulary, kept here as raw JSON so this package stays free
of transport/registry types. `result_mapping` projects a provider's native result
fields onto the router's unified result shape.

### Operability `class` — what posture Search Hub expects

`class` is how a provider declares its operational shape so Search Hub can hold it
to the right bar **without hard-coding scenario names**. It is required for active
providers under search maturity certification; a `capability_gap` stub is exempt.

| `class` | Means | Control-plane posture Search Hub enforces |
|---|---|---|
| `local_index` | Tunable, locally-owned indexed corpus | `reindex_endpoint` **required** (the corpus must be reconcilable); `config_endpoint` required unless `config_writable:false` and `config_pinned_reason` explain a fixed/hybrid-by-construction provider; `status_endpoint` advisory. |
| `async` | Locally-owned corpus rebuilt asynchronously | Same as `local_index` — `reindex_endpoint` **required**. |
| `local_live` | Computed live, no tunable index | No `reindex_endpoint` / `config_endpoint` expected; `status_endpoint` advisory. |
| `external` | Third-party / external provider (e.g. web search) | No local `status_endpoint` / `reindex_endpoint` / `config_endpoint` expected; latency/reliability budget is looser (see the performance policy). |

A missing or unknown class on an active provider fails certification with
`SEARCH_PROVIDER_CLASS_MISSING`; an indexed class without a reindex endpoint fails
with `SEARCH_REINDEX_ENDPOINT_MISSING`. Model web/external behaviour by declaring
`class: external`, never by scenario name.

### Performance evidence policy

`performance` records the provider-owned SLO that Search Hub evaluates from the
latest fresh eval run:

```jsonc
"performance": {
  "p95_ms": 1500,
  "degraded_rate_max": 0.05,
  "telemetry_required": true,
  "minimum_samples": 8
}
```

`minimum_samples` prevents a p95 or degradation rate from masquerading as a
production result when it was calculated from a smoke-sized run. Local production
providers normally declare at least 8 evaluated cases; external smoke-only
providers declare 1. A run below this explicit floor produces the required
`SEARCH_PERF_SAMPLES_UNPROVEN` finding.

## 3. `tuning` — the control-surface dashboard

The `tuning` block is the typed `TuningConfig`. **Every field is one row of the
factor taxonomy** (`aisearch.Factors`, the SSOT in `tuning.go`); a test asserts
the table and the struct can never drift. The taxonomy says, for each knob, its
**cost tier** (the central distinction), its value domain, default, and the
one-line decision rule:

| `tuning` key | Tier | Kind | Default | Decision rule (when to move it) |
|---|---|---|---|---|
| `engine` | **index-time** | enum `dense`\|`hybrid` | `dense` | hybrid adds a BM25 sparse leg (recall on keyword/long-prose corpora) at index + query cost; dense is simpler and faster for terse corpora that embed well. |
| `embed_role` | **index-time** | enum | `embedding.default` | the Ollama embedding role; changing the resolved role/model re-embeds the whole corpus and may require collection recreation if dimensions change. |
| `embed_task_prefix` | **index-time** | bool | `false` | asymmetric `search_query:` / `search_document:` prefixes for nomic (+0.20 recall on terse command corpora); changes the embedding space → reindex. Leave off for symmetric / already-tuned corpora. |
| `rerank_enabled` | query-time | bool | `false` | cross-encoder / LLM rerank lifts precision + junk rejection; buys no recall on prose corpora and adds latency + a reranker resource dependency. Off degrades cleanly to retrieval order. |
| `rerank_blend` | query-time | bool | `false` | fuse the rerank order with retrieval via RRF instead of reordering outright; keeps junk rejection without burying strongly-retrieved canonical hits (+0.20 recall on the command corpus). Requires `rerank_enabled`. |
| `rerank_shortlist` | query-time | int [1,500] | `50` | over-fetch depth into the reranker; higher = more recall into the rerank but more candidates to score (LLM-leg latency; negligible on the cross-encoder). |
| `rerank_preference` | query-time | `cross_encoder` / `llm_fallback` / `auto` | `auto` | Declares the preferred available reranker leg. A preference is advisory when that leg is unavailable; the response and telemetry still name the active leg and degradation reason. |
| `floor.max_gap` | query-time | float [0,1] | `0` | relative cutoff below the query's top hit; `0` = let the package pick the regime-appropriate band. Raise to cut more of the weak tail. |
| `floor.hard_floor` | query-time | float [0,1] | `0` | absolute garbage floor; `0` = let the package pick the regime default. Non-zero overrides it (cosine regimes want a real floor; fused regimes want 0). |

The **tier is load-bearing** — it is the split the whole self-tuning loop turns on:

- **Query-time** factors vary per request and never touch the stored vectors. The
  router's override channel (token-gated) may set them per request, and the sweep
  explores them **full-factorial** (cheap — no reindex).
- **Index-time** factors change the embedded/indexed representation, so moving one
  needs a **reindex** (the recipe-aware drift hash triggers the re-embed). The
  sweep explores them by **coordinate-ascent** (one at a time) to bound reindex
  cost; the override channel rejects them by construction.

`TuningConfig.IndexTimeChanged(other)` is the predicate `WriteConfig` uses to
decide whether persisting a new `tuning` block also requires a reindex.

### Defaults, validation, presets

- `WithDefaults()` fills absent/zero fields from the taxonomy, so a partial
  `tuning` block is always meaningful (a missing `engine` resolves to `dense`).
- `Validate()` rejects an unknown engine, an out-of-range shortlist, floors
  outside `[0,1]`, and `rerank_blend` without `rerank_enabled`.
- Two measured-best presets ship as constructors (use them as the starting point,
  then let the sweep refine):
  - `aisearch.CommandCorpusTuning()` — terse command corpora (cli-health): dense,
    `embed_task_prefix:true`, rerank on + blend. Lifts recall@5 `0.50 → 0.70`.
  - `aisearch.DocCorpusTuning()` — large prose/doc corpora (knowledge-observatory):
    hybrid, symmetric embeddings (`embed_task_prefix:false`), rerank off.
    Reproduces KO's guarded recall@5 = 0.818 baseline exactly.

> **Why these aren't more knobs.** Score-regime calibration (the weak-label band
> and the relevance floor) is **automatic** — the package detects the active
> scoring regime at runtime and picks the right band, so an adopter does not tune
> it. The `floor.*` fields exist only as overrides for an operator with eval
> evidence. See [`configuration.md`](configuration.md) §2.

## 4. `tests` — the evaluation corpus (`TestSuite`)

One labelled corpus per provider, in **one canonical rank-centric shape** that is a
1:1 structural match (modulo store-assigned fields) of search-hub's eval proto
`EvalSuite` — so it converts losslessly and a scenario self-registers it at boot
(§5). There is one case list; **negatives are cases** (a case with
`expect_no_strong_hit`), not a separate array.

```jsonc
"tests": {
  "suite_id": "cli-health.commands.primary",  // optional; default "<provider_id>.primary"
  "name": "cli-health commands — primary",     // optional suite metadata (mirrored)
  "description": "rank-centric golden corpus",  // optional
  "coverage": {
    "required_tags": ["lifecycle"],
    "required_tag_groups": [
      {
        "id": "operator-workflows",
        "description": "Common operator tasks this provider promises to answer.",
        "tags": ["lifecycle", "diagnostics"],
        "min_reviewed_positive": 2
      }
    ]
  },
  "cases": [
    {
      "id": "restart-scenario",
      "query": "restart a scenario",
      "scope": "global",                         // optional: global|scenario:<id>|path:<prefix>
      "tags": ["strong", "lifecycle"],          // difficulty/category/provenance band
      "expect_ids": ["restart"],                // POSITIVE label: leaf ids (per id_field)
      "expect_within_top_k": 3                  // the expected hit must land within K
    },
    {
      "id": "gibberish-1",                       // a NEGATIVE is just a case…
      "query": "asdf qwer zxcv",
      "tags": ["gibberish"],
      "expect_no_strong_hit": true,              // …with expect_no_strong_hit set
      "expect_max_score": 0.3
    }
  ]
}
```

- **Positives** assert `expect_ids` (leaf ids per the provider's `id_field`) landing
  within `expect_within_top_k`. By design a positive carries **no absolute
  `expect_min_score`**: the sweep compares arms across score regimes (dense cosine,
  cross-encoder, RRF-blend) where a shared absolute band would mislabel a
  rank-correct hit. `expect_min_score`/`expect_max_score` remain available for a
  single-regime corpus but are not the default.
- **Scope is optional and generic.** Empty/`global` searches the whole provider;
  `scenario:<id>` and `path:<prefix>` are passed through `GradeSuite` to the
  shared `SearchQuery.Scope`. Provider endpoints that want search-hub eval runs
  to honor scope declare `{{scope_kind}}` / `{{scope_value}}` placeholders in
  their `body_template`.
- **Review state is explicit.** Empty/`reviewed` cases count in the acceptance
  denominator; `candidate` cases are excluded until `search-hub evals promote`
  writes the reviewed state back to the provider's `search.json`.
- **Negatives** are cases with `expect_no_strong_hit` (+ `expect_max_score`) — the
  junk-rejection guards the sweep validates stay rejected, never optimizes against.
- **Provenance rides `tags`.** `"generated"` is the load-bearing marker the sweep
  **always holds out of the tuning fold** (overfit guard #2): a tuning can never be
  selected on cases a machine wrote for it. There is no separate `source` field.
- **Coverage is provider-owned.** `coverage.required_tags` and
  `coverage.required_tag_groups` declare the question families this provider must
  prove for production maturity. Only reviewed positive cases satisfy coverage;
  candidate/generated positives and negatives do not count.
- **Deleted vs. the old shape:** `expected_paths` (one label shape now — `expect_ids`),
  the separate `negatives[]` array, and `source` (now a tag) were removed
  (greenfield). Gate policy moved to the provider `scoring` block.

**Adequacy, not just well-formedness.** Parsing only checks the suite is
*well-formed*. Search-hub additionally **grades** the corpus: no reviewed
positives and no negatives are hard failures; duplicate queries, single-band
difficulty, and missing declared coverage groups are required production-maturity
debt. Thin or stale corpora are the central overfit risk, so the findings fire on
`evals show` / `evals run` / `evals validate` / `evals generate` — see the
[search-hub tuning recipe](../../../../scenarios/search-hub/docs/reference/configuration.md#search-tuning-control-surface).

## 5. Lifecycle — who reads, who writes

```
scenario boot ──► LoadSearchFile(.vrooli/search.json)
                    ├─ tuning  ──► NewServiceForTuning(tuning, deps)   (engine: dense|hybrid, by DATA)
                    └─ self-register ──► RegisterProvider(descriptor + tuning) ──► registry store (cache)
                                          ├─ search-hub mints a control TOKEN, returns it to the scenario
                                          └─ RegisterSuite(convert(tests))     ──► eval store (cache)   ← corpusStoreMirrorsFile

search-hub optimization (holds the control token):
  evals sweep --apply     ─► WriteConfig(token)  ─► scenario rewrites its OWN search.json `tuning`
                                                     ├─ reindex iff IndexTimeChanged
                                                     └─ registry cache refreshed (no reboot)
  evals generate --apply / evals promote
                           ─► WriteCorpus(token)  ─► scenario rewrites its OWN search.json `tests`
                                                     └─ eval cache re-registered from the file
```

The scenario's file is **authoritative** for both `tuning` and `tests`: every
machine write-back goes through the scenario's own token-gated `WriteConfig` /
`WriteCorpus` RPC, which rewrites the file. search-hub's registry tuning and eval
suite are **verified mirrors** of the file (`corpusStoreMirrorsFile`); the only way
they diverge is a manual file edit, which re-registration heals on the scenario's
next boot. A provider that declares no `config_endpoint` can still self-register and
be swept by overrides, but cannot accept a `tuning`/`corpus` write-back.

## 6. Minimal worked example

A dense command corpus that self-registers, exposes the control plane, and ships a
small golden corpus:

```jsonc
{
  "$schema": "https://vrooli.dev/schemas/search.json",
  "version": "1.0.0",
  "providers": [
    {
      "provider_id": "cli-health.commands",
      "provider_group": "cli-health",
      "bucket": "BUCKET_DO",
      "type": "command",
      "description": "Vrooli CLI commands across all scenarios",
      "scope": "SCOPE_PROJECT",
      "endpoint":        { "scenario_id": "cli-health", "rpc": "SearchService.Search" },
      "status_endpoint": { "scenario_id": "cli-health", "rpc": "SearchService.Status" },
      "reindex_endpoint":{ "scenario_id": "cli-health", "rpc": "SearchControlService.Reindex" },
      "config_endpoint": { "scenario_id": "cli-health", "rpc": "SearchControlService.WriteConfig" },
      "result_mapping":  { "id": "name", "title": "name", "path": "name", "snippet": "description" },
      "tuning": {
        "engine": "dense",
        "embed_role": "embedding.default",
        "embed_task_prefix": true,
        "rerank_enabled": true,
        "rerank_blend": true,
        "rerank_shortlist": 50,
        "floor": { "max_gap": 0, "hard_floor": 0 }
      },
      "tests": {
        "cases": [
          { "id": "restart-scenario", "query": "restart a scenario", "expect_ids": ["restart"], "tags": ["strong"], "expect_within_top_k": 3 },
          { "id": "gibberish-1", "query": "asdf qwer zxcv", "expect_no_strong_hit": true, "expect_max_score": 0.3, "tags": ["gibberish"] }
        ]
      }
    }
  ]
}
```

## 7. Cross-references

- [`configuration.md`](configuration.md) — the full control surface, the
  automatic score-regime calibration (why the floor/weak bands are not levers),
  the reranker chain, and the adopt → calibrate → validate flow.
- [`../../README.md`](../../README.md) — how to wire the engine
  (`NewServiceForTuning`) from a parsed `tuning` block.
- search-hub `docs/reference/configuration.md` → **Search tuning control surface**
  — the `evals sweep` / `evals generate` operator recipe (the loop that writes
  this file back).
- Code SSOTs: `tuning.go` (`Factors`, `TuningConfig`), `searchjson.go`
  (`SearchFile` / `ProviderConfig` / `TestSuite` parsing).
