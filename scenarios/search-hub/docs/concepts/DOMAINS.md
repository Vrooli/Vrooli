# Domains — Search Hub

This document is the canonical map of product capabilities, bounded
contexts, and ownership for this scenario. Keep it current whenever a
domain is added, renamed, split, merged, or removed.

Search Hub is a **thin federated search router**. It owns only the
registry, classifier, fan-out, reranker, and metrics — it stores **no
vectors and no corpus content**. Each searchable corpus stays
authoritative in its own scenario (a provider) and is reached only
through a registered, declarative descriptor. This is the load-bearing
invariant behind the domain boundaries below: a new provider is a
registry row, never router code. See
[`ARCHITECTURE.md`](ARCHITECTURE.md) §Thin-router boundary and the plan
`docs/plans/unified-search-hub-plan.md` §3.

> **Domain status (updated 2026-06-03, Phase 7).** The product
> domains below (`registry`, `routing`, `rerank`, `metrics`,
> `providers`) are the **target map**. `registry` + `providers` are live
> (Phase 3, first vertical slice), `routing` fan-out is live (Phase 4)
> with the classifier auto-routing layer (Phase 5), `rerank` is live
> (Phase 6) — implemented as a seam *within* the `routing` package (see
> the rerank section) — and `metrics` is now **live (Phase 7)**: the
> `query_telemetry` store, `MetricsService.Insights`, and the implemented
> `RoutingService.Status` (per-provider health + classifier/reranker
> availability). The template's `notes` worked example has been
> **removed**, and `health` is kept. All five product domains are now
> implemented. The "Build Phase" column records the plan phase that lands
> each domain.

## Purpose Of This Document

Use this document to answer:

- What product capabilities does this scenario expose?
- Which domain owns each concept, table, proto, endpoint, UI feature,
  CLI command, and test surface?
- Which concepts are shared, deferred, or deliberately not domains?

System-level architecture belongs in [`ARCHITECTURE.md`](ARCHITECTURE.md).
Workflow details belong in [`FLOWS.md`](FLOWS.md). Storage details
belong in [`DATA.md`](DATA.md).

## The router pipeline (how the domains compose)

```
query ─▶ [routing: classifier picks provider types]  (or explicit --type / --all)
      ─▶ [routing: bounded fan-out to matching registered providers]
             each call resolved via [registry] descriptor + [providers] adapter
             (provider runs ITS own retrieval over ITS index)
      ─▶ collect heterogeneous candidates (+ per-provider score, provenance)
      ─▶ [rerank: fuse into one comparable ranked list]  (else honest by-provider groups)
      ─▶ operator-friendly result (corpora searched, expand hints, provenance)
      ─▶ [metrics: persist per-query telemetry for measurement]
```

- **registry** owns *what providers exist* (the declarative descriptor + its storage/CRUD).
- **providers** owns *how a descriptor becomes results* (the adapter runtime: call endpoint, apply mapping, normalize score) and the concrete registration rows for live leaves + gap stubs.
- **routing** owns *which providers to call and how to combine them* (classifier, fan-out, grouping, operator output).
- **rerank** owns *one comparable ranking* across providers, with graceful degradation.
- **metrics** owns *measurement* — per-query telemetry and the insights/status aggregates that tell us whether federation is working.

## Domain Inventory

| Domain | Purpose | Primary Archetype | Owns Data | Surfaces | Requirements | Build Phase | Source Paths (target) |
|---|---|---|---|---|---|---|---|
| registry | Persist provider descriptors and serve registration CRUD. The contract every provider self-registers against. | Registry / entity | `providers` table (descriptors only — no corpus data). | API, CLI, UI | MOD-P0-001, MOD-P1-010 | 3 | `api/internal/registry/`, `api/handlers/registry/`, `cli/domains/providers/`, `ui/src/features/providers/`, `packages/proto/schemas/search-hub/v1/registry/` |
| providers | Turn a descriptor into unified `SearchHit`s: call the leaf endpoint (HTTP+JSON or CLI), apply the declarative `ResultMapping`, normalize the score. Holds the live-provider registration rows + gap stubs. | Adapter / integration | None (stateless adapters; descriptor rows owned by `registry`). | API (internal), CLI (register hooks) | MOD-P1-009, MOD-P1-010 | 3 (first leaf), 4/8 (rest) | `api/internal/providers/`, `api/internal/providers/adapters/`, registration descriptors |
| routing | Classify a free query to provider types, fan out with bounded concurrency + per-provider timeout, collect candidates with provenance, group/return them, render operator-friendly output. | Orchestration / query | None (stateless). | API, CLI, UI | MOD-P0-002, MOD-P0-003, MOD-P0-004, MOD-P0-005, MOD-P0-006 | 4 (fan-out), 5 (classifier) | `api/internal/routing/`, `api/internal/routing/classifier/`, `api/handlers/routing/`, `cli/domains/search/`, `ui/src/features/search/`, `packages/proto/schemas/search-hub/v1/routing/` |
| rerank | Fuse the per-provider shortlists into one comparable, ranked list; degrade gracefully to by-provider grouping when the reranker is unavailable. | Transform / ranking | None (calls Ollama). | API (internal) | MOD-P1-007 | 6 ✅ | `api/internal/routing/reranker.go`, `api/internal/routing/reranker_ollama.go` (a seam *within* the routing package — it post-processes the same fan-out, so co-location keeps the merge step beside what it merges; not a separate Go package) |
| metrics | Persist per-query telemetry and compute insights: provider utilization, under-used providers, zero-result rate, latency percentiles. The validation backbone. | Telemetry / reporting | `query_telemetry` table (+ derived aggregates). | API, CLI, UI | MOD-P1-008 | 7 | `api/internal/metrics/`, `api/handlers/metrics/`, `ui/src/features/insights/` |
| health | Report runtime readiness and dependency reachability (template-provided, kept). | Reporting / query | No product data. | API, UI | Starter scaffold health. | (scaffold) | `api/handlers/health/`, `ui/src/features/health/`, `packages/proto/schemas/search-hub/v1/health/` |

> The template's `notes` worked example was **removed in Phase 3** (it
> was never product scope); `registry`/`providers` replaced it. See
> **Non-Domains** for the durable seams kept from it.

## Domain Details

### registry

- Purpose: be the single source of truth for *what corpora are
  searchable and how the router reaches each one*. A provider scenario
  self-registers a `ProviderDescriptor` (proto, §3.2 of the plan); the
  registry persists it and serves CRUD.
- Primary archetype: registry / entity.
- Owns: the `providers` SQLite table (descriptors only), descriptor
  validation, and the `RegisterProvider` / `ListProviders` /
  `DeregisterProvider` RPCs + `providers register|list|remove` CLI.
- Owns the contract, not the routing: the registry never calls a
  provider and never interprets a query. It hands descriptors to
  `routing`/`providers`.
- Does not own: corpus data, vectors, retrieval, or scoring. The
  descriptor is *declarative* (endpoint + `ResultMapping`) precisely so
  the router needs zero provider-specific code (no-conditional-monolith
  invariant, plan §4 / Validation #5).
- Special rows: `CAPABILITY_GAP` stubs — corpora with no search yet
  (description + `intended_home`, no endpoint). They surface in
  `providers list` / `status` as the live Track-A adoption checklist
  (MOD-P1-010, gap requirements REQ-P1-006..015).
- Storage: embedded SQLite (`SQLITE_PATH`). See [`DATA.md`](DATA.md).
- Requirements: MOD-P0-001 (registry + self-registration), MOD-P1-010
  (gap-corpus stubs).
- Tests: descriptor validation, registry CRUD, gap-stub listing.

### providers

- Purpose: the adapter runtime that makes the registry's descriptors
  executable. Given a descriptor, it resolves the live base URL (via the
  backend cross-scenario resolver — never client-computed), calls the
  leaf endpoint, applies the declarative `ResultMapping` (generic
  JSON-path extraction), normalizes the provider's score scale to a
  common [0,1] band, and emits unified `SearchHit`s.
- Primary archetype: adapter / integration.
- Owns: the generic descriptor→`SearchHit` execution path and the
  concrete registration rows for live leaves (`cli-health.commands`,
  `ui-health.surfaces`/`.widgets`, `swarm-manager.records`/`.backlog`/
  `.initiative`, `knowledge-observatory.docs`, `prompt-manager.skill`/
  `.action`) plus the gap stubs.
- Does not own: per-provider parsing code. There is exactly **one**
  generic adapter driven by the descriptor — adding a provider is a
  registry row, not a new adapter. (This is the boundary that keeps the
  router thin; the first leaf, `cli-health.commands`, proves the
  result-mapping path in Phase 3.)
- Storage: none (stateless; descriptor rows live in `registry`).
- Requirements: MOD-P1-009 (live providers federated), MOD-P1-010 (gap
  stubs).
- Tests: result-mapping against captured provider fixtures, score
  normalization, base-URL resolution seam.

### routing

- Purpose: turn one query into the right fan-out and a coherent result.
  Reads provider `description`s from the registry to classify a free
  query to provider `type`s (Ollama, `qwen3:1.7b`) — or honors explicit
  `--type` / `--all`. Fans out to matching providers with bounded
  concurrency and per-provider timeouts, returns partial results rather
  than blocking on the slowest, and renders operator-friendly output
  (corpora searched, counts, expand hints, provenance) with `--json` for
  scripting.
- Primary archetype: orchestration / query.
- Owns: classifier (behind a `Classifier` interface — swappable),
  fan-out orchestration, by-provider grouping (the honest pre-rerank
  shape), graceful skip-with-warning for down/stale providers, and the
  `Query` / `Status` RPCs + `search` CLI.
- Does not own: corpus retrieval (providers), final ranking (rerank),
  or descriptor persistence (registry). Classification is **widen-on-
  uncertainty** — over-fetch and let rerank kill noise (plan §4).
- Storage: none (stateless).
- Requirements: MOD-P0-002 (explicit-type federation), MOD-P0-003
  (graceful degradation), MOD-P0-004 (operator-friendly output),
  MOD-P0-005 (automatic routing), MOD-P0-006 (thin-router boundary).
- Tests: routing recall ≥0.85 against `testdata/routing_queries.json`,
  uncertain-widens-not-drops, partial-results-on-timeout, operator
  output shape.

### rerank ✅ (Phase 6)

- Purpose: merge the heterogeneous per-provider shortlists into one
  comparable, ranked list. Raw provider scores are not comparable across
  corpora, so rerank is the merge step. When no reranker is wired (or it
  is unavailable), results are grouped by provider rather than falsely
  interleaved.
- Primary archetype: transform / ranking.
- Implementation note: rerank is a **seam within the `routing` package**
  (`api/internal/routing/reranker*.go`), not a standalone Go package — it
  post-processes the same fan-out it ranks, so co-locating the merge step
  beside what it merges keeps the cohesion the plan's F.6 handoff
  prescribes ("a `Reranker` interface … in the `routing` domain"). The
  earlier `api/internal/rerank/` target path was a Phase-2 sketch,
  superseded.
- Owns: the `Reranker` interface and its default implementation
  (LLM-as-reranker, pointwise 0–10 relevance), plus the degradation path:
  reranker unavailable/erroring ⇒ fall back to by-provider grouping +
  `reranked=false` + a `degraded` flag. Swap in a true cross-encoder
  (`bge-reranker-v2-m3`) behind the same interface when the KO cutover
  plan lands one — the router contract does not change.
- Model note: the default model is **`qwen3:1.7b`**, not the Phase-0
  nominee `qwen3:4b`. qwen3:4b does not honor `/no_think` through the
  resource-ollama gateway and reasons past the gateway's ~60s deadline
  (every rerank would time out and degrade); qwen3:1.7b honors it, emits
  the scores JSON directly, and reranks in ~6s. The order also breaks
  rerank-score ties by the original per-provider score, so a hedging
  small model (all-5/10) preserves the providers' retrieval signal rather
  than collapsing to fan-out order. Override with
  `SEARCH_HUB_RERANKER_MODEL`.
- Does not own: candidate generation (providers) or fan-out (routing).
- Storage: none (calls Ollama).
- Requirements: MOD-P1-007 (unified cross-provider ranking).
- Tests: deterministic parse/ordering/fuse unit tests (always-on) + the
  real-model rerank-ordering/MRR gate (`reranker_mrr_test.go`, skips when
  Ollama is down) + the degradation fallback path.

### metrics

- Purpose: the validation backbone. Persist per-query telemetry and
  derive the insights that tell us, over time, whether federation is
  actually working and where it is under-used. This is first-class, not
  an afterthought — it is how the classifier/rerank earn their cost.
- Primary archetype: telemetry / reporting.
- Owns: the `query_telemetry` + `query_telemetry_provider` tables
  (routed types, per-provider hit counts, total result count, latency,
  degraded/zero-result/reranked flags) and the aggregates surfaced via
  `MetricsService.Insights` (CLI `search-hub insights`): per-provider
  utilization, registered-but-never-routed-to providers (`under_utilized`,
  signals bad descriptions), zero-result rate, p50/p95 latency. Live
  federation health (per-provider reachability + classifier/reranker
  availability) is surfaced via `RoutingService.Status` (CLI
  `search-hub federation`) — implemented in the `routing` package since it
  reads the registry + the model seams, not the telemetry tables.
- Boundary: the router owns the `TelemetrySample` write seam
  (`internal/routing/telemetry.go`); the metrics store implements it via a
  bridge at the wiring edge (`handlers/metrics/recorder.go`), so
  `internal/metrics` never imports `internal/routing` and vice versa.
- Does not own: query orchestration (routing) or descriptors (registry).
  Query text is **hashed** (SHA-256) before it reaches telemetry — the
  tables carry no recoverable user input.
- Storage: embedded SQLite (`SQLITE_PATH`). See [`DATA.md`](DATA.md).
- Requirements: MOD-P1-008 (measurement backbone).
- Tests: `internal/metrics/store_test.go` (Record + p50/p95 + window +
  per-provider aggregates), `handlers/metrics/connect_handler_test.go`
  (under-utilized reconciliation + zero-divide guard + opaque errors),
  `internal/routing/telemetry_status_test.go` (a query records exactly one
  sample with hashed text; `Status` reports reachability + model
  availability).

### health

- Purpose: expose API/database readiness and show the UI can read live
  backend state (template-provided seam, kept).
- Primary archetype: reporting / query.
- Owns: health response construction and dependency status mapping.
- Does not own: product data or scenario behavior.
- API: `api/handlers/health/`. UI: `ui/src/features/health/`. CLI:
  built-in `status` via cli-core.
- Requirements: starter scaffold health only.
- Tests: handler, module, UI feature, accessibility.

## Shared Concepts

| Concept | Meaning | Owner |
|---|---|---|
| Provider (leaf) | One indexed corpus of one type (≈ one qdrant collection), reached via one descriptor. | `registry` defines the descriptor; the owning scenario owns the corpus. |
| Provider group | A scenario that registers N leaf providers (e.g. `swarm-manager` → `.records`/`.backlog`/`.initiative`). Provenance + optional `--group` scoping. | `registry` (the `provider_group` field). |
| Descriptor | The declarative `(endpoint, ResultMapping, bucket, type, description)` row that makes "new provider = a row" true. | `registry`. |
| ResultMapping | JSON-path field selectors + score scale that map any provider's response onto `SearchHit` — the reason the router has zero provider-specific code. | `registry` (data) / `providers` (executes). |
| Bucket | `DO` / `REUSE` / `KNOW` / `STATE` routing facet from `ai-search-routing.md`. | `registry`. |
| Capability gap | A tracked corpus with no search yet (stub row, no endpoint). The Track-A adoption checklist. | `registry`. |
| Surface | API, UI, or CLI layer exposing the same capability. | [`ARCHITECTURE.md`](ARCHITECTURE.md). |
| Seam | Test-substitutable boundary wired once in production. | [`../internal/SEAMS.md`](../internal/SEAMS.md). |
| Requirement | Implementation-facing measurement tied back to the PRD. | `requirements/`. |

## Deferred Domains

| Candidate Domain | Why Deferred | Revisit Trigger |
|---|---|---|
| External providers (web / papers / inventory) | Contract carries `scope = EXTERNAL` from day one; none registered in v1. | A paid/external corpus wants to federate. |
| Hub-assisted provider indexing | Out of scope for v1 to protect the thin-router boundary (router indexes nothing on a provider's behalf). | Re-open only if a provider has no search of its own *and* `packages/aisearch-go` adoption (Track B) does not cover it. |
| `--group` unified search through the hub | Optional; each scenario keeps its own group-local unified search regardless (non-destructive invariant). | Demand for reproducing a group's `--entity both` through the hub. |

## Non-Domains

These are important but should not become product domains:

- `api/internal/server/` — HTTP composition substrate.
- `api/internal/module/` — shared module descriptor type.
- `api/internal/modules/` — thin registry for boot/codegen.
- `api/internal/database/` — cross-cutting database infrastructure.
- `api/internal/httpc/` — outbound HTTP seam (the `providers` adapter's
  transport; the seam is generic, the adapter is the domain).
- `api/internal/clock/` — deterministic time seam.
- `api/internal/testutil/` — cross-domain test harnesses.
- `ui/src/components/` — shared presentation primitives.
- `ui/src/test-utils/` — cross-feature testing support.

**Scaffold removed (Phase 3):** the template's `notes` worked example
(`api/internal/notes/`, `api/handlers/notes/`, `cli/domains/notes/`,
`ui/src/features/notes/`, `ui/src/api/notes.ts`,
`packages/proto/schemas/search-hub/v1/notes/`) is **gone** — `registry`
replaced it as the first real green domain. The durable UI seams it
demonstrated (i18n, a11y, design tokens, feature-folder pattern) are
kept.

If one of the infrastructure pieces starts using product vocabulary,
split the product piece into an owning domain instead of growing
infrastructure.

## Cross-References

- [`ARCHITECTURE.md`](ARCHITECTURE.md) — system shape and extension rules
- [`FLOWS.md`](FLOWS.md) — workflows and state transitions
- [`DATA.md`](DATA.md) — data ownership and storage
- [`INTEGRATIONS.md`](INTEGRATIONS.md) — dependency contracts
- [`../internal/SEAMS.md`](../internal/SEAMS.md) — boundary registry
- [`../internal/TESTING.md`](../internal/TESTING.md) — test strategy
- `docs/plans/unified-search-hub-plan.md` (repo root) — the full plan
  this scenario implements (§3 architecture, Appendix A contracts)
