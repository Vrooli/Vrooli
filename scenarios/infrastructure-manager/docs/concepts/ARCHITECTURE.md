# Architecture — Infrastructure Manager

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
                       │   infrastructure-manager/v1/...    │
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
| Contracts (`packages/proto/schemas/infrastructure-manager/`) | Wire shape | Proto messages/services and generated clients | Hand-written route/type mirrors |

The load-bearing principle: the API is the only surface that contains
business logic. UI and CLI translate user/operator intent into API
calls. Proto types flow from one source of truth so wire-shape drift
between surfaces is impossible.

## System Boundaries

The scenario owns:

- source code under `api/`, `ui/`, and `cli/`,
- generated-scenario docs under `docs/`,
- scenario lifecycle metadata under `.vrooli/`,
- scenario-specific requirements under `requirements/`,
- scenario proto schemas relocated to
  `packages/proto/schemas/infrastructure-manager/`.

The scenario does not own:

- shared package implementation under `packages/`,
- Vrooli resource implementation,
- scenario dependencies it calls,
- generated proto outputs under `packages/proto/gen/`.

### The product boundary — what this scenario deliberately does not do

The defining property is that this is a **thin, read-mostly aggregator**. It
measures the platform's readiness by reading other scenarios' typed outputs,
and it never re-implements a measurement, performs an improvement, or makes a
judgment call. Five exclusions are contractual rather than provisional:

- **Actuation of any kind.** No restart, no shelve or unshelve, no reconcile-and-fix,
  no policy-lever change, no degrade or preempt, no privileged host mutation.
  Operating-model rule 3 is "supervise, don't operate"; live incident response
  belongs to `vrooli-autoheal`, `system-monitor`, `agent-manager` and the operator.
- **Authoring its own denominator, in either half.** The *space* — which cells exist —
  is authored by each control layer in its own `docs/spaces/<projection>-space.md`. The
  *bar* is a checked-in setpoint file with **no API write path at all**, changed only
  through a reviewed commit carrying an approved `reliability-target-update`. An observer
  that writes its own reference model is confirming itself — deviation `D6`.
- **Holding a roster.** Every set is a derivation query executed at read time. This
  is what clears `INSTRUMENTATION_ROADMAP.md` Gap 11's objection to a central
  capability-health scenario, and it is the rule most likely to be eroded by a
  well-meaning caching change.
- **Re-implementing an upstream measurement.** Each source owns its own semantics.
  This scenario reads derived output; it never re-runs a toolchain, a probe, or a scan.
- **Deciding.** It ranks candidates and states confidence. Repair, defer, adopt and
  retire stay with the team and the operator.

Document dependency and resource decisions in
[`INTEGRATIONS.md`](INTEGRATIONS.md), not here.

## Contracts And Data Flow

Wire shapes do not live in TypeScript interfaces, Go structs, or
hand-written JSON schemas. They live in `.proto` files. For
proto-typed API calls, the `.proto` file also declares the service
block that generates Connect handlers and clients.

```
packages/proto/schemas/infrastructure-manager/v1/<domain>/<file>.proto
       │
       ▼
       make generate
       │
       ├──▶ packages/proto/gen/go/infrastructure-manager/v1/...              (api, cli)
       ├──▶ packages/proto/gen/go/infrastructure-manager/v1/...connect       (Connect-Go)
       ├──▶ packages/proto/gen/typescript/js/infrastructure-manager/v1/...   (ui)
       └──▶ packages/proto/gen/python/infrastructure_manager/v1/...    (future tools)
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
   `packages/proto/schemas/infrastructure-manager/v1/<domain>/`.
2. Add API domain code under `api/internal/<domain>/`.
3. Add transport code under `api/handlers/<domain>/`.
4. Register schemas/endpoints in `api/internal/modules/registry.go`
   and mount the module in `api/main.go`.
5. Add CLI commands under `cli/domains/<domain>/`.
6. Add UI API wrappers under `ui/src/api/<domain>.ts` and UI feature
   code under `ui/src/features/<domain>/`.
7. Update selectors, strings, endpoints, tests, and the docs contract
   in `docs/manifest.json`.

### Scenario-specific extension rules

These six bind every change here and exist because each protects an invariant
that is easy to break by accident and expensive to notice afterwards:

1. **A new domain measures by reading.** It never re-implements an owner's
   measurement, and never calls a mutating verb on a dependency.
2. **A new read source is a typed client with graceful degradation** — never a
   hard dependency that can fail the whole board. An unreadable source produces a
   visible availability entry with a stated reason, never a zero and never a
   dropped row.
3. **Never persist a band verdict.** Store readings; recompute in-band / out-of-band at
   query time against the current deadband. A stored band verdict silently outlives the
   target that produced it. **Trust verdicts are the deliberate exception** and are stored
   on the reading row: a band verdict is a statement about the target and is recomputable,
   while a trust verdict is a statement about the observation and is not — nothing can
   reconstruct whether a check was saturated last Tuesday.
4. **Never cache a space, the setpoint, or a derived set.** All three are read fresh. A
   `coverage` table, a stored cell grid, or a persisted leg population is the signal that
   this rule has been broken.
5. **Always pair a ratio with its denominator-confidence**, and never report an
   uninstrumented target or an untrusted reading as healthy. The board must be
   structurally unable to imply completeness it has not earned.
6. **No actuation path, ever.** If a change introduces a call that alters a
   dependency's state, it belongs in a different scenario. This is the boundary that
   separates an instrument from a controller.
7. **A new projection is data, never code.** Adding an eleventh projection must require
   zero new domains, zero new tables and zero new handlers — an owner authors a space doc,
   the setpoint gains bars, and the grid grows a column. If adding one requires a code
   change here, the grid has been special-cased and the change should be reverted rather
   than extended.
8. **No write path to the setpoint.** Not an endpoint, not a table, not a migration.
   The absence is the D6 defence; a "convenient" tuning screen is the specific way this
   invariant will be proposed away.

For detailed product ownership, update [`DOMAINS.md`](DOMAINS.md).
For persistence and retention, update [`DATA.md`](DATA.md). For
temporal behavior, update [`FLOWS.md`](FLOWS.md). For the four model
documents — projections and cells, banding, trust, and the bar — update
[`COVERAGE-MODEL.md`](COVERAGE-MODEL.md), [`CONDITION-MODEL.md`](CONDITION-MODEL.md),
[`TRUST-MODEL.md`](TRUST-MODEL.md) and [`SETPOINT-MODEL.md`](SETPOINT-MODEL.md).

## Architecture Maturity

**Current state: draft, documentation-first.** The charter (`PRD.md`), the
requirements registry, the domain map, the four model documents, and this
architecture are authored. **No domain code exists yet.** This mirrors how
`meta-optimization-manager` was built and is deliberate: the models are the
design decisions worth getting right before code hardens around them — and the
2026-08-19 reframe onto projections is exactly the payoff, since it would have
been a rewrite rather than an edit had the flat sensor map already been coded.

The first real vertical slice is `coverage` — API, CLI and its board page —
preceded by a design pass and by the `vrooli-autoheal` typed surface that the
`condition` vertical depends on.

| Area | Maturity | Evidence | Remaining Drift |
|---|---|---|---|
| Concept model | **Authored** | `COVERAGE-MODEL.md`, `CONDITION-MODEL.md`, `TRUST-MODEL.md` and `SETPOINT-MODEL.md` define ten projections, the cell grid, banding, the closed trust vocabulary, the bar, and the honesty invariants. `DOMAINS.md` names three domains and their build order. | No control layer has authored a space doc, so every denominator starts at `SKETCH`. The obligation list remains judgment owned by the team, not derivable here. |
| API | Template-ready, no product code | Domain-owned vertical-slice stack, module registry, per-domain schema, documented seams — all from the template. | No `coverage`, `condition`, `focus`, or `sources` package exists. |
| CLI | Template-ready, no product code | Domain command groups wrap API calls and render reports. | Command surface is designed in `DOMAINS.md` but unimplemented. |
| UI | Template-ready, in scope | Feature folders, typed API clients, selector/i18n registries. | `OT-P1-005`; a polished board is first-class and ships vertical-by-vertical with each domain, after one design pass. |
| Docs | **Contract-complete for a draft** | Manifest v2 registers every document with a maturity value; concept docs are authored rather than stubbed. | Reference and operations docs describe the template, not the product, until the first slice lands. |

Use `docs/manifest.json` as the documentation contract. The declared
`maturity` values are expected to be maintained by agents and later
grounded by Knowledge Observatory validation.

## Intentional Deviations

Record deviations from the template or from Vrooli scenario standards
when they are deliberate and durable.

| Date | Deviation | Reason | Revisit Trigger |
|---|---|---|---|
| 2026-08-17 | **Readings are persisted.** `meta-optimization-manager`, the sibling instrument, never stores a numerator so that a stale board is structurally impossible. This scenario stores readings. | Reliability targets are inherently historical — "99.5% uptime over 30 days" cannot be computed from a live probe. `INSTRUMENTATION_ROADMAP.md` Gap 11 names the cost of not storing them: an outage becomes indistinguishable from missing data after the fact. The invariant's *intent* is preserved by storing readings but never verdicts, so every conclusion is still computed at query time. | If reading history is ever consumed as a conclusion rather than re-evaluated against the current deadband, this deviation has failed and the storage must be reconsidered. |
| 2026-08-17 | **Four concept documents** — `COVERAGE-MODEL.md`, `CONDITION-MODEL.md`, `TRUST-MODEL.md` and `SETPOINT-MODEL.md` — beyond the template's concept set. | Two mirror `meta-optimization-manager`'s axes. The Trust axis has no analogue anywhere in the fleet: this instrument's inputs are alarms on live processes, which lie in four named ways, rather than registry joins, which do not. The setpoint earns its own document because it is the one artifact whose write path must not exist. | If the fleet grows a shared trust/sensor-integrity contract, `TRUST-MODEL.md` collapses into a pointer to it. |
| 2026-08-19 | **The denominator is split across two owners.** `meta-optimization-manager` puts the whole denominator with the capability owner; here the space lives with the owner and the bar lives with the instrument. | Reliability bars are operator judgment, not owner knowledge. An owner that sets the bar it is graded against is `D6` one layer down; an instrument that asserts the owner's cell set is a roster in all but name. Splitting on that seam is the only shape where neither party can grade itself. | If reliability bars ever become genuinely owner-knowledge, the split collapses back to the MoM shape. |
| 2026-08-19 | **One CLI read survives: `vrooli capacity`.** Every other source is a typed Connect call through `api-core/discovery`. | `vrooli capacity` is control-plane `internal/`, so a separate Go module cannot import it and `discovery` cannot resolve it. This is a construction constraint, not a shortcut, and it is fenced to one named source so it cannot spread. | If the control plane exposes a typed local surface, this read moves to it and the exception is deleted. |
| 2026-08-17 | **Generated from a template whose registry status is `quarantined`.** | `template-manager registry list --kind scenario` reports both scenario templates as quarantined, while Template Manager's own progress log records react-vite 1.6.5 passing deep validation on 2026-07-12 with the registry active. Generation succeeded and the scaffold validates. The status discrepancy is upstream, not a property of this scenario. | Resolve with Template Manager before the first vertical slice; if the quarantine is real rather than stale, re-evaluate the template choice. |

## Documentation Architecture

Scenario docs follow the same ownership rule as code: one durable
question, one canonical home.

| Concern | Canonical Document |
|---|---|
| System map and extension rules | `docs/concepts/ARCHITECTURE.md` |
| Projections, cells, denominators, confidence | `docs/concepts/COVERAGE-MODEL.md` |
| Banding, reading history, actuation efficacy | `docs/concepts/CONDITION-MODEL.md` |
| The closed trust vocabulary | `docs/concepts/TRUST-MODEL.md` |
| The bar, deadband discipline, update protocol | `docs/concepts/SETPOINT-MODEL.md` |
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
