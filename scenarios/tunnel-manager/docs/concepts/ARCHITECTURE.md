# Architecture — Tunnel Manager

This document is the scenario's system map. It explains the invariant
shape inherited from the `react-vite` template, then points to the
specialized documents that own product domains, workflows, data,
integrations, deployment, operations, and business strategy.

> Status: implemented and validation-green for the production-readiness
> redesign. The proto/API/CLI/UI surfaces described here are built, with
> `vrooli scenario test tunnel-manager` last recorded at 18/18 phases
> green. Deferred items are called out explicitly instead of described as
> scaffold gaps.

## What Tunnel Manager Is

Tunnel Manager is Vrooli's external-access control plane — an **exposure
broker** and **self-healing tunnel manager**. It owns which scenarios are
reachable from the public internet through the Cloudflare tunnel, keeps a
route/exposure manifest as the single source of truth, enforces fixed-port
contracts, and auto-recovers the tunnel from failure. Architecturally it is
a normal three-surface scenario (API/UI/CLI over proto contracts on SQLite)
whose distinguishing trait is that its API reaches **outward** to real
infrastructure: `cloudflared` (systemd + Prometheus + `/ready`), the
Cloudflare API v4, scenario `service.json` files, the `api-core/coreset`
seam, and the `internal/lifecycle` ensure-running seam. Those outward
contracts are documented in [`INTEGRATIONS.md`](INTEGRATIONS.md); this file
covers the in-scenario shape.

The seven product domains (`routes`, `exposure`, `config`, `audit`,
`tunnel`, `probes`, `recovery`) plus the scaffold `health` domain follow
the screaming-architecture layout below; their boundaries and ownership
live in [`DOMAINS.md`](DOMAINS.md).

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
                       │   tunnel-manager/v1/...    │
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

Tunnel Manager's API additionally talks to infrastructure outside the
scenario. These are seams (test-substitutable boundaries wired once in
production), not new surfaces:

```
        ┌──────────┐  api/internal/<domain> seams
        │  api/Go  │
        └────┬─────┘
             ├──▶ cloudflared        systemd status + /ready + Prometheus (tunnel)
             ├──▶ Cloudflare API v4  remote ingress push/sync           (config)
             ├──▶ ~/.cloudflared/config.yml  local-mode generation      (config)
             ├──▶ scenario service.json      fixed-UI-port audit         (audit)
             ├──▶ api-core/coreset           core-tier membership        (exposure)
             ├──▶ internal/lifecycle         ensure-running delegation   (exposure)
             └──▶ public route URLs          external liveness probes    (probes)
```

| Surface | Role | Owns | Does Not Own |
|---|---|---|---|
| API (`api/`) | Scenario core | Business rules, persistence, integrations, transport edge | Browser state, CLI formatting |
| UI (`ui/`) | Browser presentation | Components, i18n, accessibility, browser interaction | Business rules, persistence policy |
| CLI (`cli/`) | Operator/agent wrapper | Argument parsing, output formatting, API invocation | Business rules, duplicated validation |
| Contracts (`packages/proto/schemas/tunnel-manager/`) | Wire shape | Proto messages/services and generated clients | Hand-written route/type mirrors |

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
  `packages/proto/schemas/tunnel-manager/`.

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
packages/proto/schemas/tunnel-manager/v1/<domain>/<file>.proto
       │
       ▼
       make generate
       │
       ├──▶ packages/proto/gen/go/tunnel-manager/v1/...              (api, cli)
       ├──▶ packages/proto/gen/go/tunnel-manager/v1/...connect       (Connect-Go)
       ├──▶ packages/proto/gen/typescript/js/tunnel-manager/v1/...   (ui)
       └──▶ packages/proto/gen/python/tunnel_manager/v1/...    (future tools)
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
   `packages/proto/schemas/tunnel-manager/v1/<domain>/`.
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

| Area | Maturity | Evidence | Remaining Drift |
|---|---|---|---|
| API | Implemented | Seven product domains (`routes`/`exposure`/`config`/`audit`/`tunnel`/`probes`/`recovery`) expose Connect-RPC services backed by SQLite repositories, production seams, generated endpoint metadata, scheduler wiring where required, and optional service-layer operator-token authz for privileged mutation RPCs. | Scenario-authenticator aud-scoped tokens remain deferred before non-operator cross-scenario callers get direct privileged mutation access. Richer DNS/Cloudflare outage classification is deferred until additional signals exist. |
| UI | Implemented | Overview, Settings/Setup, Exposure, Metrics/Diagnostics, Audit, and Recovery surfaces call generated clients and have component coverage for readiness, filtering, reconciliation, diagnostics, audit remediation, and recovery guidance. | BAS/e2e journey coverage and richer visual regression evidence can still improve confidence; P2 analytics/dashboard export remain future work. |
| CLI | Implemented | `tunnel`, `routes`, `exposure`, `probes`, `audit`, `recovery`, and `config` command groups mirror the API and keep proto-shaped `--json` output. | Operator ergonomics can keep improving around hints and empty states, but command groups are no longer scaffold placeholders. |
| Docs | Active | Docs now track implementation state, validation history, deferred work, and advisory gaps through `PROGRESS.md`/`PROBLEMS.md`; `docs/manifest.json` remains the validation contract. | Some docs-health warnings are advisory and remain tracked as cleanup, not blocking product readiness. |

Use `docs/manifest.json` as the documentation contract. The declared
`maturity` values are expected to be maintained by agents and later
grounded by Knowledge Observatory validation.

## Intentional Deviations

Record deviations from the template or from Vrooli scenario standards
when they are deliberate and durable.

| Date | Deviation | Reason | Revisit Trigger |
|---|---|---|---|
| 2026-06-18 | API reaches outward to host infrastructure (`cloudflared`, Cloudflare API, `service.json`, `systemctl`) via per-domain seams. | This scenario *is* the external-access control plane; outward I/O is its product, not an accident. | Revisit if any outward call grows business logic that should be its own domain. |
| 2026-06-18 | `recovery` acts LIVE on foundational infra (restarts cloudflared) from day one. | Operator chose immediate self-healing over a monitor-only soak (see [`../internal/DECISIONS.md`](../internal/DECISIONS.md)). | Tighten circuit breaker if false-positive restarts occur. |
| 2026-06-18 | Single-owner restart contract: TM is the only authority that restarts cloudflared; `vrooli-autoheal` downgrades to alert-only. | Avoid dueling restarters fighting over the same daemon. | Revisit if autoheal ownership model changes. |
| 2026-06-18 | UI organized as exactly 5 surfaces, not score-chasing pages. | PRD mandates a minimal, glanceable operator dashboard. | Revisit per operator feedback. |

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
