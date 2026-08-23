# Architecture — Asset Studio

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
                       │   asset-studio/v1/...    │
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
| Contracts (`packages/proto/schemas/asset-studio/`) | Wire shape | Proto messages/services and generated clients | Hand-written route/type mirrors |

The load-bearing principle: the API is the only surface that contains
business logic. UI and CLI translate user/operator intent into API
calls. Proto types flow from one source of truth so wire-shape drift
between surfaces is impossible.

## Production Layering — the scenario-specific shape

On top of the standard scenario shape, this scenario has one structure that
governs everything: **declarations are versioned and immutable once used;
production is disposable and always re-derivable.**

```
  CANON     ── docs/marketing/catalogs/rich-media/. Operator-curated, decision-gated.
     │         Who the personas are and why. This scenario reads it, never writes it.
     │         Currently EMPTY — templates only, zero authored records (D-021).
     ▼  (idempotent import, content-addressed)      ┌── operator authoring ──┐
  IDENTITIES ── validated, versioned records. A block referenced by an accepted
     │          asset is frozen forever; a change is a NEW VERSION.
     │          Two ingresses: authoring (primary today) and import (migration).
     ▼  (bind by VERSION, never by name)
  SPECS     ── declarative, resolves as a pure function. Same inputs, same payload.
     │
     ▼  (submit through ai-gateway)
  RENDERS   ── the job. Captures PROVENANCE and COST. Disposable: an artifact
     │         can be thrown away and re-made from its provenance alone.
     ▼
  ASSETS    ── bytes + metadata + disclosure. One copy, cited by identifier.
     │
     ▼  (gate)
  CONFORMANCE ── did the render honour the identity? Fails closed.
```

Four invariants follow, and they are the ones to defend in review:

1. **A declaration that something depends on is immutable.** Editing an identity
   block that an accepted asset binds would silently change what that asset's
   provenance means. The change is a new version; the old one stays resolvable
   forever.
2. **Provenance is captured or the job fails.** It cannot be backfilled. An
   artifact produced without a complete provenance record can never acquire one,
   so a partial record is not a degraded success — it is a failure.
3. **An unchecked frame is not a passing frame.** Release blocks on an
   *unresolved* conformance verdict, not only a failing one. The gate fails
   closed, because the failure it prevents — a drifted identity entering the
   published record — compounds across every artifact that anchors to it.
4. **Production never becomes authoritative for a declaration.** Nothing flows
   from `assets` back into `specs` or `identities`, and nothing flows from this
   scenario back into the catalogue. If a change would let produced output
   redefine its own inputs, it is a layering defect.

### What P0 pays forward, and what it does not

Two P0 requirements buy schema shape rather than behaviour, and a reader
budgeting the first slice should know which. This is the same "cheap insurance,
taken deliberately" posture `vrooli-memory` records in its D-019 and D-020.

| Requirement | What P0 buys | What it does not buy | Why it cannot wait |
|---|---|---|---|
| `ASSET-P0-016` conditioning reference | The field on the identity block, and its capture in every provenance record. | Rendering from it — that is `ASSET-P1-012`. | Provenance cannot be backfilled. An artifact released without the reference can never acquire it, so the column has to exist before the first release, not before the first render. |
| `ASSET-P0-017` candidate set | Job-to-artifact cardinality of one-to-many, and cost attribution across the set. | A rich selection surface. | Changing cardinality later rewrites provenance, cost attribution, and the asset lifecycle at once. The metric most likely to reveal a problem — spend per *released* artifact — is wrong by the candidate count until it is right. |

Everything else in P0 is slice work: it is exercised by walking one product
identity from authoring through release. Nothing else is paid forward, and in
particular no multi-store, multi-tenant, or plugin seam exists anywhere in this
scenario — see § Intentional Deviations before adding one.

### Why versioning rather than a mutable library

The obvious design is a library of characters you edit as they evolve, which is
how the catalogue works today. That is correct for canon and wrong here, for one
reason: this scenario promises that an artifact can be explained and re-made.
A mutable identity makes both promises false the first time someone improves a
character description — every prior artifact silently claims to have been made
from a definition it never saw.

What is kept from the mutable model is that *unreferenced* records stay freely
editable. The constraint binds only once something depends on it, so authoring a
new persona has no versioning overhead at all.

### Why generation and capture share one pipeline

A generated persona frame and a recorded product demo feel like different
products. They differ only in their *source*: everything downstream — the job
lifecycle, provenance, cost, the asset library, variants, alt text, release —
is identical. Modelling capture as a second pipeline would duplicate all of it
to avoid one discriminator field. Capture is therefore a spec kind and a
backend (`ASSET-P1-003`), which is also where the orphaned `video-studio` skill's
scope lands.

## System Boundaries

The scenario owns:

- source code under `api/`, `ui/`, and `cli/`,
- generated-scenario docs under `docs/`,
- scenario lifecycle metadata under `.vrooli/`,
- scenario-specific requirements under `requirements/`,
- scenario proto schemas relocated to
  `packages/proto/schemas/asset-studio/`.

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
packages/proto/schemas/asset-studio/v1/<domain>/<file>.proto
       │
       ▼
       make generate
       │
       ├──▶ packages/proto/gen/go/asset-studio/v1/...              (api, cli)
       ├──▶ packages/proto/gen/go/asset-studio/v1/...connect       (Connect-Go)
       ├──▶ packages/proto/gen/typescript/asset-studio/v1/...   (ui)
       └──▶ packages/proto/gen/python/asset_studio/v1/...    (future tools)
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
   `packages/proto/schemas/asset-studio/v1/<domain>/`.
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

## Architecture Maturity

Generated scenarios start with a mature template shape and starter
reference domains. Replace this table as the scenario becomes real.

| Area | Maturity | Evidence | Remaining Drift |
|---|---|---|---|
| API | Scaffold | Template vertical-slice stack and module registry only. No product domain exists in code. | All five product domains are designed and unbuilt. Build order is `identities → specs → renders → assets → conformance` (see `DOMAINS.md` § Build Order). |
| UI | Scaffold | Template feature folders, typed clients, selector/i18n registries. | The workbench is specified in `UI-ARCHITECTURE.md` and unbuilt. |
| CLI | Scaffold | Template command groups only. | Compose, render, and query verbs are unbuilt. |
| Docs | Contract-ready | PRD, requirements registry (30 requirements across P0/P1/P2, all `planned`), and the concept and internal sets are authored and scenario-specific. | Reference docs (`api-endpoints`, `cli-commands`, `configuration`) stay template-shaped until surfaces exist to describe. |
| Contracts | Not started | No proto schemas authored. | `packages/proto/schemas/asset-studio/v1/<domain>/` is empty; the first slice creates it. |
| Formal flows | Not started | The render job lifecycle is specified in `FLOWS.md` § State Machines. | `api/internal/renders/flow/` must be scaffolded through `flow-verifier` before the domain is built, not after. |

Use `docs/manifest.json` as the documentation contract. The declared
`maturity` values are expected to be maintained by agents and later
grounded by Knowledge Observatory validation.

## Intentional Deviations

Record deviations from the template or from Vrooli scenario standards
when they are deliberate and durable.

| Date | Deviation | Reason | Revisit Trigger |
|---|---|---|---|
| 2026-07-28 | `identities` exposes no in-place update for a referenced identity block, unlike every other CRUD domain in the fleet. | Immutability-once-referenced is the scenario's central invariant. A change is a new version, not an edit. | Never — this is the scenario. |
| 2026-07-28 | This scenario keeps a BlobStore seam while its sibling `content-desk` deliberately has none. | Bytes live where they are produced. Exactly one copy of every artifact exists, in the scenario that made it; consumers hold references. | Never. A change putting artifact bytes in a consumer is a layering defect. |
| 2026-07-28 | Cost accounting is a P0 requirement rather than an operational concern added later. | Generation spend is unbounded in a way editorial work is not — a mis-specified multi-frame video spec can cost real money before a human sees a frame. A pipeline that cannot answer what it spent cannot be given a budget later. | Never expected. |
| 2026-07-28 | The release gate requires a human verdict even when an automated score is available. | The scoring model is unvalidated. An automated pass on a mis-scored frame is the silent failure the gate exists to prevent. | If scoring is ever validated against a corpus of operator judgements large enough to measure its false-pass rate. |
| 2026-07-28 | A `campaign_ref` on a spec is free text this scenario never resolves. | It is a label for cost reporting, not a foreign key. Resolving it would create a dependency on `content-desk` and invert the intended direction. | Never. |

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
