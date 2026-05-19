# Architecture — Development Toolchain Validator

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
                       │   development-toolchain-validator/v1/...    │
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
| Contracts (`packages/proto/schemas/development-toolchain-validator/`) | Wire shape | Proto messages/services and generated clients | Hand-written route/type mirrors |

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
  `packages/proto/schemas/development-toolchain-validator/`.

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
packages/proto/schemas/development-toolchain-validator/v1/<domain>/<file>.proto
       │
       ▼
       make generate
       │
       ├──▶ packages/proto/gen/go/development-toolchain-validator/v1/...              (api, cli)
       ├──▶ packages/proto/gen/go/development-toolchain-validator/v1/...connect       (Connect-Go)
       ├──▶ packages/proto/gen/typescript/js/development-toolchain-validator/v1/...   (ui)
       └──▶ packages/proto/gen/python/development_toolchain_validator/v1/...    (future tools)
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
   `packages/proto/schemas/development-toolchain-validator/v1/<domain>/`.
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
| API | Reference-ready | Domain-owned notes stack, module registry, per-domain schema, documented seams. | Starter domains must be replaced with scenario-specific capabilities. |
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
| 2026-05-18 | None yet. | Generated from `react-vite`. | Update when the scenario intentionally diverges. |

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

## DTV-Specific Architecture

This scenario layers a validation pipeline on top of the template's standard Connect-RPC + SQLite module-per-domain shape. The pipeline is:

```
golden (template-pristine scenario)  ─┐
                                       ├─► sandboxed agent run via agent-manager ─► diff
skill (steer skill from               ─┘                                            │
       prompt-manager)                                                              ▼
                                                                      compare to expected-diff manifest
                                                                                    │
                                                                                    ▼
                                                                             verdict + record
                                                                       (pass / unexpected-mutation /
                                                                        run-failure / stale)
```

Each PRD operational target maps to a planned API module (one module per bounded context, all under `api/internal/<domain>/`):

| PRD Ref | Domain | Status (2026-05-18) |
|---|---|---|
| OT-P0-001 | `golden` | **Shipped.** Reference vertical. |
| OT-P0-002 | `skill_catalog` | **Shipped.** Local mirror of prompt-manager's skill catalog. RPCs: Sync / ListSkills / GetSkill. Outbound seam: `SkillCatalogSource` (REST adapter to prompt-manager today). |
| OT-P0-003 | `manifest` | **Shipped.** Per-(skill_id, golden_slug) expected-diff manifest with pure-policy `Evaluate(manifest, diff) → Verdict` decision boundary (≥15 table-driven cases). RPCs: ListManifests / GetManifest / UpsertManifest / ClearStale. |
| OT-P0-004 + 005 | `validation_run` | **Shipped (async lifecycle).** RPCs: Start / Get / ListActive. In-process worker advances queued → running → evaluating → terminal. Outbound seams: AgentManagerClient (stub; see PROBLEMS), ToolRunner (dev-tool CLI shell-out), WorkspaceSandboxClient (stub; optional). |
| OT-P0-006 | `validation_record` | **Shipped.** Append-only history. RPCs: ListRecords (cursor-paginated, filters) / GetRecord. |
| OT-P0-007 | `staleness` | **Shipped.** Pure derivation over manifest pinning vs current template + skill versions, suppressed by manual ClearStale overrides. RPC: ListStale. |
| OT-P0-008 | `report` | **Shipped.** Pure composition. RPCs: GetGoldenSummary / GetTupleHistory / GetCoverage. No storage. |
| OT-P0-009 | CLI parity | **Shipped.** Every Connect RPC has a matching `development-toolchain-validator <domain-kebab> <verb>` command. |

Every domain follows the same shape (per the template):
- `api/internal/<domain>/{model,repository,sqlite,schema}.{go,sql}` + `module.go` + `service.go`
- `api/handlers/<domain>/connect_handler.go` + `module.go`
- `cli/domains/<domain>/`
- `ui/src/features/<domain>/`
- `packages/proto/schemas/development-toolchain-validator/v1/<domain>/<domain>.proto`

### External Dependencies

DTV declares its scenario dependencies in `.vrooli/service.json`
(`dependencies.scenarios`):

- **agent-manager** (Connect-RPC, **required**) — submit sandboxed
  (skill, golden) runs and receive run summary + diff. Consumed by the
  `validation_run` module via `integrations/agent_manager`. Current
  wiring is a stub returning `ErrDependencyUnavailable`; see
  PROBLEMS.md for the followup.
- **prompt-manager** (REST today, **required**) — read steer skill
  catalog with versions + content hashes. Consumed by the
  `skill_catalog` module via `integrations/prompt_manager`
  (REST adapter behind the `SkillCatalogSource` seam; migrates to
  Connect when prompt-manager exposes proto).
- **workspace-sandbox** (Connect-RPC, **optional**) — read per-path
  file content when manifest content-rule evaluation requires body
  inspection. Consumed by the `validation_run` worker via
  `integrations/workspace_sandbox`. Optional: the worker degrades to
  path-only verdicts when unreachable.
- **`scenario-auditor`, `test-genie`, `scenario-completeness-scoring`**
  — dev-tool CLIs invoked behind the `ToolRunner` seam in
  `integrations/dev_tools`. Used by `TUPLE_KIND_TOOL` validation runs.
- **`templates/scenarios/*`** — read filesystem source for golden
  regeneration. Consumed by the `golden` module via
  `vrooli scenario generate` subprocess.

All outbound URLs come from `package:api-core/discovery` resolution;
nothing in DTV hardcodes a scenario port.

Nothing in the broader ecosystem depends on DTV; it is a verification
layer.

### Testing Infrastructure (L3+ architecture)

Every new domain ships at L≥3 from day one (per
`unit-testing-architecture-steer`):

- No `time.Now()`, `os.Getenv`, `http.DefaultClient`, or
  `log.Default()` in `internal/<dom>/` outside the ambient seam
  packages (`internal/clock/`, `internal/httpc/`).
- Repository tests use the real SQLite handle via
  `internal/testutil/db.NewSQLite` paired with `apidb.EnsureSchemas`.
- Service tests use fakes from `internal/<dom>/mocks/` and
  `internal/testutil/mocks/`. Each fake carries a `var _ Iface = ...`
  compile-time assertion.
- Handler tests use `connectxtest.StartTestServer` against the real
  Connect handler.
- CLI tests use `cli-core/cliapptest.NewCapturedRunContext` against a
  stubbed Connect mux.
- The validation_run worker is exercised by an integration test that
  drives the full async pipeline through fakes (`worker_test.go`).

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
