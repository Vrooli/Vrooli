# Architecture — Vrooli Memory

This document is the scenario's system map. It explains the invariant
shape inherited from the `react-vite` template, then points to the
specialized documents that own product domains, workflows, data,
integrations, deployment, operations, and business strategy.

Keep this file high-signal. Do not turn it into a warehouse for every
domain, endpoint, workflow, or decision. If a concern has a dedicated
document below, update that document and link it here.

## Purpose Of This Document

This document owns:

- the scenario's system shape,
- the role of each surface,
- how contracts and data flow between surfaces,
- the shared infrastructure boundary,
- extension rules for future code,
- architecture maturity and intentional deviations.

This document does not own:

- product capability inventory: [`DOMAINS.md`](DOMAINS.md),
- temporal and user/system workflows: [`FLOWS.md`](FLOWS.md),
- storage details and retention: [`DATA.md`](DATA.md),
- resource and scenario dependencies: [`INTEGRATIONS.md`](INTEGRATIONS.md),
- test seams and fakes: [`../internal/SEAMS.md`](../internal/SEAMS.md),
- test strategy: [`../internal/TESTING.md`](../internal/TESTING.md),
- deployment and operations: [`../operations/DEPLOYMENT.md`](../operations/DEPLOYMENT.md),
- commercial strategy: [`../business/MONETIZATION.md`](../business/MONETIZATION.md).

## Scenario Shape

A scenario is one product expressed through three coordinated surfaces
and one canonical contract layer.

```
                       ┌─────────────────────────────┐
                       │  Generated proto types      │
                       │  packages/proto/schemas/    │
                       │   vrooli-memory/v1/...    │
                       └──────────────┬──────────────┘
                                      │ canonical wire shape
              ┌───────────────────────┼───────────────────────┐
              │                       │                       │
              ▼                       ▼                       ▼
        ┌──────────┐            ┌──────────┐            ┌──────────┐
        │   ui/    │ Connect-JSON│  api/   │ Connect-JSON│  cli/   │
        │ React    │ ◀────────▶ │   Go     │ ◀────────▶ │   Go     │
        │ + Vite   │            │ HTTP     │            │ cli-core │
        └──────────┘            └────┬─────┘            └──────────┘
                                     │
                                     ▼
                                ┌─────────┐
                                │ SQLite  │
                                │ (local) │
                                └─────────┘
```

| Surface | Role | Owns | Does Not Own |
|---|---|---|---|
| API (`api/`) | Scenario core | Business rules, persistence, integrations, transport edge | Browser state, CLI formatting |
| UI (`ui/`) | Browser presentation | Components, i18n, accessibility, browser interaction | Business rules, persistence policy |
| CLI (`cli/`) | Operator/agent wrapper | Argument parsing, output formatting, API invocation | Business rules, duplicated validation |
| Contracts (`packages/proto/schemas/vrooli-memory/`) | Wire shape | Proto messages/services and generated clients | Hand-written route/type mirrors |

The load-bearing principle: the API is the only surface that contains
business logic. UI and CLI translate user/operator intent into API
calls. Proto types flow from one source of truth so wire-shape drift
between surfaces is impossible.

## Memory Layering — the scenario-specific shape

On top of the standard scenario shape, this scenario has one additional
structure that governs everything: **a single direction of truth.**

```
  RECEIPTS  ── vrooli-events. every API call, every scenario. already exists.
     │         Memory does not write here; it is correlated here for free,
     │         because a memory write is itself an API call.
     ▼  (correlation ids only — never copied payloads)
  JOURNAL   ── append-only. every memory an agent deliberately writes.
     │         Never rewritten. Never deleted. Always searchable at full fidelity.
     │         THE SOLE AUTHORITY.
     ▼  (facet tag routes to policy)
  FOREST    ── frontier-agglomerative summaries. episode facet only.
     │         Rebuildable cache. Safe to drop and recompute.
     ▼
  VIEWS     ── recall · wake · zoom · harness projection
```

Three invariants follow, and they are the ones to defend in review:

1. **Compaction is a context-budget device, never a storage device.** Summaries
   are additive. No leaf is ever deleted to reclaim space, and no leaf is ever
   excluded from search because it was compacted.
2. **Derived layers are rebuildable; the journal is not.** `forest` can always be
   recomputed from `journal` + `facets`. The reverse must never become true.
3. **Guaranteed inclusion beats ranking for standing rules.** Pinned entries are
   in `wake` unconditionally. Semantic similarity cannot promise presence, and a
   standing rule that silently vanishes from context is the scenario's
   highest-consequence failure.

### Why not a fixed binary tree over time

The obvious prior art (OptMem) builds a complete binary tree over the log and
merges *time-adjacent* memories. That works for a single user in a single
session, where adjacent-in-time correlates with same-topic. It does not work
here: this fleet writes memories from parallel agents across ~100 scenarios plus
the operator's personal domains, so time-adjacent entries are unrelated and
their summaries would be incoherent. Compaction therefore groups by **semantic
proximity over the frontier**, not by position.

What is kept from that design is the budget idea — a fixed-size ambient view
whose granularity is finest near the present — applied to the frontier rather
than to a positional tree.

## System Boundaries

The scenario owns:

- source code under `api/`, `ui/`, and `cli/`,
- generated-scenario docs under `docs/`,
- scenario lifecycle metadata under `.vrooli/`,
- scenario-specific requirements under `requirements/`,
- scenario proto schemas relocated to
  `packages/proto/schemas/vrooli-memory/`.

The scenario does not own:

- shared package implementation under `packages/`,
- Vrooli resource implementation,
- scenario dependencies it calls,
- generated proto outputs under `packages/proto/gen/`.

Document dependency and resource decisions in
[`INTEGRATIONS.md`](INTEGRATIONS.md), not here.

## Contracts And Data Flow

Wire shapes do not live in TypeScript interfaces, Go structs, or
hand-written JSON schemas. They live in `.proto` files. For
proto-typed API calls, the `.proto` file also declares the service
block that generates Connect handlers and clients.

```
packages/proto/schemas/vrooli-memory/v1/<domain>/<file>.proto
       │
       ▼
       make generate
       │
       ├──▶ packages/proto/gen/go/vrooli-memory/v1/...              (api, cli)
       ├──▶ packages/proto/gen/go/vrooli-memory/v1/...connect       (Connect-Go)
       ├──▶ packages/proto/gen/typescript/vrooli-memory/v1/...   (ui)
       └──▶ packages/proto/gen/python/vrooli_memory/v1/...    (future tools)
```

Use Connect-RPC by default:

- UI to API for proto-typed payloads,
- CLI to API for proto-typed payloads,
- API to API / inter-scenario calls with Vrooli-owned protos.

REST is allowed only for four enumerated reasons, defined as
`RESTReason` constants in `api/internal/module/module.go`:

| Reason | When it applies |
|---|---|
| `RESTReasonMultipartUpload` | Opaque file bytes via `multipart/form-data`. A binary/blob attachment-upload endpoint is the canonical case. |
| `RESTReasonWebhookReceiver` | Endpoint shape is dictated by a third-party system (Stripe, GitHub, etc.) we do not own. |
| `RESTReasonThirdPartyShape` | Request or response is an externally-defined contract (OAuth callbacks, OpenAPI passthrough). |
| `RESTReasonOpsProbe` | Lifecycle systems, load balancers, and `curl` must reach the endpoint without a generated client (plain `GET /health`, static iframe-facing HTML wrappers). |

Mechanical enforcement: `cmd/gen-endpoints` rejects any
`EndpointDescriptor.Path` that is not a generated Connect procedure
constant (i.e. does not start with `/vrooli.`) unless the descriptor
carries a `RESTException` with one of the four reasons. A REST
endpoint without that tag fails `make endpoints`, which fails
`make test`, which fails CI. The fix is either to author a proto
service method (the preferred path) or to tag the exception
explicitly. There is no "internal endpoint, REST is fine" path —
that rationalization is exactly what the validation pass prevents.

Note: even for REST exceptions, the **payload shape** stays
proto-typed wherever possible. A multipart attachment-upload handler
should return a proto-typed metadata message (e.g.
`UploadAttachmentResponse`); only the request transport is multipart.
Drift between API/UI/CLI is eliminated as long as the wire payload type
is shared.

## Shared Infrastructure

Shared infrastructure is allowed only when the code is
business-vocabulary-free and used by unrelated domains or surfaces.

| Package/Folder | Purpose | Why Not Domain-Owned | Consumers |
|---|---|---|---|
| `api/internal/server/` | Compose modules and middleware into one HTTP server. | Server lifecycle is not a product capability. | API entrypoint and handler modules. |
| `api/internal/module/` | Shared module and endpoint descriptor types. | Domain modules return this common shape. | Handler packages, server, endpoint codegen. |
| `api/internal/modules/` | Thin registry for schemas and endpoints. | Boot/codegen need central lists; logic stays domain-owned. | `main.go`, `gen-endpoints`. |
| `api/internal/database/` | System schema and DB reachability seam. | Cross-cutting DB infrastructure, not one domain's data. | API boot, health. |
| `api/internal/clock/` | Deterministic time seam. | Time is cross-cutting and test-substitutable. | Middleware, repositories. |
| `api/internal/testutil/` | Cross-domain test harnesses and fakes. | Used by unrelated domains; domain fakes stay domain-local. | API tests. |
| `ui/src/test-utils/` | Cross-feature render helpers, a11y helpers, and model tests. | Used by unrelated UI features. | UI tests. |

If shared infrastructure starts using product vocabulary, move that
piece back into the owning domain or split a new domain first.

## Extension Rules

Add product behavior by adding or updating the owning domain, not by
growing generic buckets.

For a normal proto-backed domain:

1. Add proto messages and service methods under
   `packages/proto/schemas/vrooli-memory/v1/<domain>/`.
2. Add API domain code under `api/internal/<domain>/`.
3. Add transport code under `api/handlers/<domain>/`.
4. Register schemas/endpoints in `api/internal/modules/registry.go`
   and mount the module in `api/main.go`.
5. Add CLI commands under `cli/domains/<domain>/`.
6. Add UI API wrappers under `ui/src/api/<domain>.ts` and UI feature
   code under `ui/src/features/<domain>/`.
7. Update selectors, strings, endpoints, tests, and the docs contract
   in `docs/manifest.json`.

For detailed product ownership, update [`DOMAINS.md`](DOMAINS.md).
For persistence and retention, update [`DATA.md`](DATA.md). For
temporal behavior, update [`FLOWS.md`](FLOWS.md).

## Deliberately Not Built — the general source ledger

**Status: recorded idea, not design. Nothing below is scheduled, and no P0/P1
requirement depends on it.** It is written down so the seam is deliberate rather
than rediscovered, and so a future agent does not mistake the absence of
generalization for an oversight.

### The observation

The bottom of this scenario is not memory-specific. Strip the vocabulary away
and what remains is an **append-only journal with provenance, a multi-vector
semantic index, frontier-agglomerative compaction, and a fixed-budget ambient
view** — plus content-addressed idempotent import. That combination is what any
*source ledger* needs underneath: a durable, auditable, semantically searchable
store of accumulated findings that stays navigable as it grows without bound.

Several plausible applications want exactly that substrate and differ only above
it:

| Application | Ledger holds | Layer built on top |
|---|---|---|
| Fact-checking | Claims, evidence, sources, refutations | Belief state and prior updating; research directions that challenge current beliefs |
| Mathematical research | Conjecture-scoped ideas, lemmas, proof attempts, counterexamples | Formalization via Lean; open/proven/refuted state |
| Scientific research | Papers, datasets, experiment results, replications | Hypothesis tracking, confidence, contradiction surfacing |
| Personal health protocols | Interventions, measurements, outcomes over time | Protocol state, adherence, effect attribution |

### Where the seam actually is

What generalizes is everything from the journal down and the compaction
mechanics up. What does **not** generalize is the vocabulary:
`DOMAINS.md` makes the facet set closed, and the six facets in it
(standing-rule, environment-fact, gotcha, episode, thread, entity-record) are
*agent-memory* facets. A fact-checking ledger wants claim / evidence / source /
refutation; a conjecture ledger wants lemma / proof-attempt / counterexample.

So the generalization is narrow and precise: **the facet vocabulary and its
policy table become per-ledger; the engine below them does not change.** That is
in the grain of this scenario, which already uses "a descriptor, not a build"
twice — `.vrooli/search.json` for search providers and the declarative harness
adapters in [`DATA.md`](DATA.md).

Two things a future attempt must not assume are already solved:

- **Belief state is not summarization.** A claim whose confidence updates as
  evidence arrives, or a conjecture that is open/proven/refuted, is a computed
  projection with domain logic. Supersession marks are a primitive version and
  will not stretch that far. That logic belongs in a scenario *on top of* the
  ledger, not inside it.
- **Entry shape is the real fork.** See D-020. Compaction scores clusters by
  node count, so chunking long artifacts into entries would distort the frontier
  as well as flood it.

### If it is ever taken: two shapes

**Extract the engine as a package** — `journal + forest + cover()` as shared Go,
with `vrooli-memory` as one consumer and a `source-ledger` scenario as another.
Each keeps its own facet vocabulary, storage, UI, and search-hub leaves.
Cross-ledger retrieval is already solved: a second scenario that registers its
own provider leaf is federated-queryable through `search-hub query` with zero
work on this side.

**Or multi-store inside this scenario** — a `store_id` on every table and a
per-store descriptor. Simpler to reach, but it puts multi-tenancy in every query
path, gives one scenario the blast radius of all ledgers, and forces a UI
generic enough to serve both agent memory and mathematical research, which
usually means it serves neither well.

**The package shape is preferred** on current information: it matches how the
fleet already shares code (`aisearch-go`, `api-core`) and keeps scope discipline
intact.

### Why it is not built now

The design is unvalidated in exactly the places generalization would multiply.
Only `health` exists in code. D-007 records the compaction scoring shape as
unvalidated; D-018 records the pin budget as unvalidated; the facet taxonomy has
never met a real corpus. If the facet model is wrong, fixing it across one store
is a correction and fixing it across several is a migration.

What was paid forward instead is the cheap insurance only — D-019 (facet policy
in data rather than Go constants) and D-020 (entry shape decided rather than
discovered). Together they cost close to nothing now and keep the door open. See
D-021.

## Architecture Maturity

Generated scenarios start with a mature template shape and starter
reference domains. Replace this table as the scenario becomes real.

| Area | Maturity | Evidence | Remaining Drift |
|---|---|---|---|
| API | Reference-ready | Domain-owned vertical-slice stack, module registry, per-domain schema, documented seams. | Starter domains must be replaced with scenario-specific capabilities. |
| UI | Reference-ready | Feature folders, typed API clients, selector/i18n registries, modeltest helpers. | Real scenarios may need routing/state patterns once multiple screens exist. |
| CLI | Reference-ready | Domain command groups wrap API calls and render reports. | New domains must add commands intentionally; CLI should remain thin. |
| Docs | Contract-ready | Manifest v2 registers docs, maturity, stages, and validation hints. | Scenario-specific stubs must be filled or marked not-applicable. |

Use `docs/manifest.json` as the documentation contract. The declared
`maturity` values are expected to be maintained by agents and later
grounded by Knowledge Observatory validation.

## Intentional Deviations

Record deviations from the template or from Vrooli scenario standards
when they are deliberate and durable.

| Date | Deviation | Reason | Revisit Trigger |
|---|---|---|---|
| 2026-07-27 | `journal` exposes no update or delete operation, unlike every other CRUD domain in the fleet. | Append-only is the scenario's central invariant. A correction is a new entry plus a supersession mark. | Never — this is the scenario. |
| 2026-07-27 | Compaction is a scheduled background sweep rather than an inline agent action. | The prior art makes compaction agent-labour, which forces strict ordering and forbids subagents from writing. A sweep removes both constraints, so a swarm can write concurrently. | If sweep latency ever makes the frontier unbounded in practice. |
| 2026-07-27 | Memory stores run correlation ids but never run payloads. | `vrooli-events` is the one truth about a run. Copying would create a second, drifting truth. | Never. |
| 2026-07-27 | No access-control partitioning of memory content. | Unified read across all scenarios is the product. See `DECISIONS.md` D-005. | If this scenario is ever deployed multi-tenant, which is out of scope today. |

## Documentation Architecture

Scenario docs follow the same ownership rule as code: one durable
question, one canonical home.

| Concern | Canonical Document |
|---|---|
| System map and extension rules | `docs/concepts/ARCHITECTURE.md` |
| Product capabilities and bounded contexts | `docs/concepts/DOMAINS.md` |
| Workflows and state transitions | `docs/concepts/FLOWS.md` |
| Data ownership, retention, and migrations | `docs/concepts/DATA.md` |
| Resources, scenarios, and external services | `docs/concepts/INTEGRATIONS.md` |
| Monetization and packaging | `docs/business/MONETIZATION.md` |
| Go-to-market strategy | `docs/business/GO-TO-MARKET.md` |
| Deployment tiers and readiness | `docs/operations/DEPLOYMENT.md` |
| Operator procedures | `docs/operations/RUNBOOK.md` |
| Telemetry, metrics, and alerts | `docs/operations/OBSERVABILITY.md` |
| Seams and test doubles | `docs/internal/SEAMS.md` |
| Testing strategy | `docs/internal/TESTING.md` |
| Known drift and deferred work | `docs/internal/PROBLEMS.md` |
| Change history | `docs/internal/PROGRESS.md` |

Every durable scenario document should be registered in
`docs/manifest.json`. Put deep domain-specific documentation under
`docs/domains/<domain>/` when `DOMAINS.md` would become noisy.

## Cross-References

- [`START-HERE.md`](../START-HERE.md) — first implementation workflow
- [`QUICKSTART.md`](../QUICKSTART.md) — clone-to-running flow
- [`DOMAINS.md`](DOMAINS.md) — bounded contexts and ownership
- [`FLOWS.md`](FLOWS.md) — workflow and state-transition map
- [`DATA.md`](DATA.md) — data ownership and storage
- [`INTEGRATIONS.md`](INTEGRATIONS.md) — dependency contracts
- [`../business/MONETIZATION.md`](../business/MONETIZATION.md) — commercial story
- [`../operations/DEPLOYMENT.md`](../operations/DEPLOYMENT.md) — deployment readiness
- [`../internal/SEAMS.md`](../internal/SEAMS.md) — seam registry
- [`../internal/TESTING.md`](../internal/TESTING.md) — test patterns
- [`../internal/ERROR-HANDLING.md`](../internal/ERROR-HANDLING.md) — error semantics
- [`../internal/PROBLEMS.md`](../internal/PROBLEMS.md) — known issues / tech debt
- [`../internal/PROGRESS.md`](../internal/PROGRESS.md) — lifecycle log
