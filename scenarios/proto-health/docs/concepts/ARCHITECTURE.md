# Architecture — Proto Health

This document is the scenario's system map. It explains the invariant
shape inherited from the `react-vite` template, then names the real
`proto-health` domains that will replace the generated notes reference:
`validation` and `protosurface`.

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

`proto-health` is a meta / interface-enabler scenario. It validates one
scenario's proto contracts at a time and publishes a structured
proto-surface fact for downstream tools.

```
                       ┌─────────────────────────────┐
                       │  Generated proto types      │
                       │  packages/proto/schemas/    │
                       │   proto-health/v1/...    │
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
                                │ Repo +  │
                                │ descriptor
                                └─────────┘
```

| Surface | Role | Owns | Does Not Own |
|---|---|---|---|
| API (`api/`) | Scenario core | Descriptor reading, per-scenario validation, proto-surface facts, transport edge | Fleet graph computation, dependency drift analysis |
| UI (`ui/`) | Browser presentation | Fleet/status inspection, findings and surface rendering, loading/error/empty states | Validation rules, descriptor parsing |
| CLI (`cli/`) | Operator/agent wrapper | `validate scenario` and `describe scenario` commands, output formatting, API invocation | Duplicated validation logic |
| Contracts (`packages/proto/schemas/proto-health/`) | Wire shape | `ProtoHealthService` and generated clients | Hand-written route/type mirrors |

The load-bearing principle: the API is the only surface that contains
validation and descriptor logic. UI, CLI, test-genie, and later
dependency-analyzer call the API instead of re-implementing proto
inspection.

## System Boundaries

The scenario owns:

- source code under `api/`, `ui/`, and `cli/`,
- generated-scenario docs under `docs/`,
- scenario lifecycle metadata under `.vrooli/`,
- scenario-specific requirements under `requirements/`,
- scenario proto schemas relocated to
  `packages/proto/schemas/proto-health/`,
- the stable `DescribeScenarioProtos` fact contract.

The scenario does not own:

- shared package implementation under `packages/`,
- Vrooli resource implementation,
- scenario dependencies it calls,
- generated proto outputs under `packages/proto/gen/`,
- cross-scenario graph computation, service dependency drift, or
  fleet-aware dead-proto detection.

Document dependency and resource decisions in
[`INTEGRATIONS.md`](INTEGRATIONS.md), not here.

## Contracts And Data Flow

Wire shapes do not live in TypeScript interfaces, Go structs, or
hand-written JSON schemas. They live in `.proto` files. For v1,
`ProtoHealthService` owns two methods:

- `ValidateScenario` - validate one scenario and return findings.
- `DescribeScenarioProtos` - return the reusable proto-surface fact.

```
packages/proto/schemas/proto-health/v1/<domain>/<file>.proto
       │
       ▼
       make generate
       │
       ├──▶ packages/proto/gen/go/proto-health/v1/...              (api, cli)
       ├──▶ packages/proto/gen/go/proto-health/v1/...connect       (Connect-Go)
       ├──▶ packages/proto/gen/typescript/js/proto-health/v1/...   (ui)
       └──▶ packages/proto/gen/python/proto_health/v1/...    (future tools)
```

Use Connect-RPC by default:

- UI to API for proto-typed payloads,
- CLI to API for proto-typed payloads,
- API to API / inter-scenario calls with Vrooli-owned protos.

REST is allowed only for four enumerated reasons, defined as
`RESTReason` constants in `api/internal/module/module.go`:

| Reason | When it applies |
|---|---|
| `RESTReasonMultipartUpload` | Opaque file bytes via `multipart/form-data`. The notes attachments endpoint is the worked example. |
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
proto-typed wherever possible. The notes attachments handler returns
the proto `UploadAttachmentResponse` message; only the request
transport is multipart. Drift between API/UI/CLI is eliminated as
long as the wire payload type is shared.

`proto-health` validates contracts in tiers:

1. **Declaration completeness**: endpoint metadata must explicitly
   declare REST exception request, response, and error payload intent.
2. **Static contract consistency**: declared payloads, imports,
   stability, and shared-type placement must match descriptor facts.
3. **Implementation proof**: `proto-health` consumes `code-facts`
   proof reports for generated proto adoption and REST exception
   handler conformance. Missing or contradicted proof becomes a
   proto-health finding; unsupported or unavailable analyzers degrade
   to warnings. `proto-health` does not parse Go or TypeScript source.

The generated `notes` domain remains in the scaffold as a reference
vertical until `validation` and `protosurface` are implemented. It is
not product scope.

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

For `proto-health`, real domain implementation should follow this order:

1. Add `validation` and `protosurface` proto contracts under
   `packages/proto/schemas/proto-health/v1/`.
2. Build a descriptor-reading seam modeled on `packages/measures-go`.
3. Implement `protosurface` as the read model over descriptor and repo
   facts.
4. Implement `validation` as checks over that model.
5. Add CLI commands and the UI inspection surface.
6. Wire test-genie, ecosystem-manager, and the audit skill only after
   the local API/CLI contract is stable.

For a normal proto-backed domain:

1. Add proto messages and service methods under
   `packages/proto/schemas/proto-health/v1/<domain>/`.
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
reference domains. `proto-health` has a real target map but has not yet
replaced the notes reference in code.

| Area | Maturity | Evidence | Remaining Drift |
|---|---|---|---|
| API | Planned | Health + notes reference compile; PRD/requirements name validation and protosurface. | Implement descriptor reader, validation service, and surface fact RPC. |
| UI | Planned | Template shell and typed API patterns exist. | Replace template dashboard/notes emphasis with proto-health fleet inspection. |
| CLI | Planned | Template CLI manifest and command wiring exist. | Add `validate scenario` and `describe scenario`; keep notes as reference until replaced. |
| Docs | Active | PRD, requirements, architecture, domains, seams, problems, and testing docs describe target state. | Keep docs synced as real domains land. |

Use `docs/manifest.json` as the documentation contract. The declared
`maturity` values are expected to be maintained by agents and later
grounded by Knowledge Observatory validation.

## Intentional Deviations

Record deviations from the template or from Vrooli scenario standards
when they are deliberate and durable.

| Date | Deviation | Reason | Revisit Trigger |
|---|---|---|---|
| 2026-06-11 | Keep generated `notes` slice temporarily. | It remains the reference vertical until the first real domain is implemented. | Remove when `validation` and `protosurface` have their own API/CLI/UI/test examples. |

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
