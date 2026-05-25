# Architecture — TypeScript Code Graph

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
                       │   typescript-code-graph/v1/...    │
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
| Contracts (`packages/proto/schemas/typescript-code-graph/`) | Wire shape | Proto messages/services and generated clients | Hand-written route/type mirrors |

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
  `packages/proto/schemas/typescript-code-graph/`.

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
packages/proto/schemas/typescript-code-graph/v1/<domain>/<file>.proto
       │
       ▼
       make generate
       │
       ├──▶ packages/proto/gen/go/typescript-code-graph/v1/...              (api, cli)
       ├──▶ packages/proto/gen/go/typescript-code-graph/v1/...connect       (Connect-Go)
       ├──▶ packages/proto/gen/typescript/js/typescript-code-graph/v1/...   (ui)
       └──▶ packages/proto/gen/python/typescript_code_graph/v1/...    (future tools)
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
   `packages/proto/schemas/typescript-code-graph/v1/<domain>/`.
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

Record deviations from the template or from Vrooli scenario standards when they are deliberate and durable.

| Date | Deviation | Reason | Revisit Trigger |
|---|---|---|---|
| 2026-05-23 | Shared `Graph` proto envelope lives at `packages/proto/schemas/common/v1/code_graph.proto`, not under this scenario's own `v1/`. | The envelope is co-owned with `go-code-graph` (and future Python/Rust siblings). A scenario-private definition would force adapter drift across consumers. | Revisit if the envelope outgrows the shared model. See [`../internal/DECISIONS.md`](../internal/DECISIONS.md). |
| 2026-05-23 | Scenario runs a **Node sidecar process** as a child of the API. The react-vite template does not anticipate language-sidecar scenarios; this scenario hand-authors `sidecar/` with `ts-morph` and a JSON-over-stdio IPC contract. | `ts-morph` is a Node library and the API is Go. Embedding Node via cgo is fragile; a Go-native TS parser doesn't exist with `ts-morph` parity. The sidecar isolates Node's lifecycle from the API. | Revisit only when a Go-native TS parser with `ts-morph` parity becomes viable (OT-P2-006). |
| 2026-05-23 | Leading-comment metadata (`leading_comments: string[]`) on every declaration node is a load-bearing public contract from day one. | `react-component-library` currently uses regex to scrape JSDoc tags. Its migration onto a typed graph requires verbatim leading comments. Removing this field later would be a breaking change for rcl. | Treat as permanent for v1. Revisit only if a structured tag parser is added as a separate field. |
| 2026-05-23 | No data persistence in v1. The optional SQLite Operation Log (P1) is the only planned persisted state. | `Extract` is pure; plans are ephemeral by design (5-min TTL); sidecar state is process-lifetime only. The template's default SQLite-everywhere shape doesn't fit a stateless parser. | Revisit if a real consumer demands persistent caching. |
| 2026-05-23 | Scenario never invokes `git`, `tsc`, or `pnpm`. The operator owns build verification and rollback via git. | Single-responsibility: this scenario is a parser + mechanical mutator, not a build runner or VCS tool. Cartographer's build-green guardrail lives at cartographer's layer. | Revisit only if a consumer demonstrably needs atomic rollback that git can't provide cheaply. |
| 2026-05-23 | Per-path serialization mutex governs both `Extract` and `Rewrite apply` — **at both the Go and sidecar layers**. Calls for different paths run in parallel. | `ts-morph` Project state is not safe to share across parallel invocations against the same project. Two-layer enforcement gives defense in depth. | Revisit if contention becomes a measured bottleneck. |
| 2026-05-23 | Partial graph + `Warnings[]` on parse failures, not hard fail. | Mid-migration scenarios are first-class inputs (both cartographer and rcl care about parsing broken projects). | Revisit if consumers request an opt-in strict mode; we'd add `--strict` rather than flip the default. |
| 2026-05-23 | UI in v1 is intentionally a debug surface (graph explorer + diagnostics including sidecar status panel), not a workbench. Refactor application stays CLI-only. | "Scenarios always have UI" but the most useful human surface for a parser is paste-path-see-graph. The sidecar status panel is prominent because a dead sidecar is the typical failure mode. | Revisit when a consumer scenario asks for a richer UI surface. |

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

## Zone Map

Every top-level directory under the scenario root has exactly one owner and one purpose. The boundary rules in the last column are mechanical (enforced by `no_prod_import_test.go` and `no_external_command_test.go` where applicable) — they are not stylistic preferences.

| Path | Owner | Purpose | Boundary rules |
|---|---|---|---|
| `api/` | API service | Go HTTP/Connect-RPC server; business logic; per-path serialization; sidecar supervision. | Module root. Subdirectory rules below. |
| `api/cmd/gen-endpoints/` | Codegen | Generates `.vrooli/endpoints.json` from the modules registry. | Pure tool; not imported by production code. |
| `api/handlers/{graph,health,rewrite}/` | Transport edge | Connect handler factories, proto<->domain translation, error-code mapping. | Imports `internal/<domain>` for the service interface; never the repository or persistence directly. |
| `api/internal/graph/` | Graph domain | Extract orchestration, NodeID/EdgeID derivation, hashing, normalization, error mapping, leading-comment passthrough. | Forbidden imports: `os/exec` (sidecar only), `time` for wall-clock (use `clock.Clock`), `net/http` (use `httpc.Doer`), sibling product domains. |
| `api/internal/rewrite/` | Rewrite domain | RewritePlan/Apply orchestration, PlanID stability, in-memory PlanStore, dry-run short-circuit. | Same import-quarantine as `graph/`. `no_external_command_test.go` asserts no `exec.Command` to git/tsc/pnpm/node from this package. |
| `api/internal/sidecar/` | Sidecar substrate | **Only** package allowed to `import "os/exec"`. Owns spawn, framing, handshake, heartbeat, restart-with-backoff, cancellation, and the `SidecarClient`/`Supervisor` types. | Importable by `main.go` (wiring) and test packages. Not importable by `graph`/`rewrite` (they take `SidecarClient` via constructor). |
| `api/internal/{clock,httpc,httpx,middleware,module,modules,server,database,testutil}/` | Template substrate | Cross-cutting infrastructure inherited from the react-vite template. | See react-vite template docs. |
| `api/internal/{graph,rewrite,sidecar}/mocks/` | Per-domain fakes | `FakeSidecarClient`, `FakePlanStore`. Co-located with the domain. | Test-only via `testutil/no_prod_import_test.go`. |
| `api/internal/sidecar/testdata/fake_sidecar.js` | Test fixture | Minimal Node IPC peer for supervisor tests (avoids the real `dist/index.js`). | Test-only. |
| `cli/` | CLI binary | Thin wrapper over the Connect-RPC API. `cli-core` argument parsing + manifest-driven dispatch. | No business logic. Forbidden: hand-rolled HTTP/JSON; must use generated Connect-Go client. |
| `cli/domains/{graph,rewrite}/` | Per-domain commands | `extract`, `rewrite plan`, `rewrite apply` handlers. | Each `Register(core, manifest)` binds proto methods to handlers. No HTTP code. |
| `cli/internal/testutil/` | CLI test helpers | RunContext factories per `cli-core/cliapptest`. | Test-only. |
| `cli/{install.sh,install.ps1,manifest.json,manifest_embed.go}` | CLI install + surface | `manifest.json` is the declarative command surface (cli-manifest/v1). | `RequireProtoServiceCoverage` test enforces every RPC has a binding or `omitted` entry. |
| `sidecar/` | Node sidecar | `ts-morph`-based extract + rewrite peer. JSON-over-stdio IPC. Self-contained Node project (pinned `ts-morph`, `--frozen-lockfile`). | Owns the IPC protocol definition (`src/protocol.ts`) as source of truth. |
| `sidecar/src/` | Sidecar runtime | `index.ts` (stdio loop), `protocol.ts` (message envelopes), `extract.ts`, `rewrite.ts`, `framing.ts`, `handshake.ts`, `lock.ts` (per-path serialization), `logger.ts` (stderr only). | **`console.log` to stdout is forbidden** — would corrupt the framer. All logs go to stderr via `logger.ts`. `child_process.spawn`/`exec`/`execSync` are intercepted in tests and must never be called in production code. |
| `sidecar/scripts/build.mjs` | Build script | esbuild bundle → `dist/index.js`. | Single-file bundle; only runtime dep is `ts-morph` (devDeps stripped). |
| `sidecar/tests/` | Sidecar tests | vitest suite (`protocol`, `extract`, `lock`, `rewrite`, `no-external-command`). | Run via `pnpm test`. |
| `sidecar/dist/index.js` | Build output | Bundled sidecar entrypoint. | Gated by `.vrooli/service.json` lifecycle setup; checked into VCS for `vrooli scenario start` zero-network startup. |
| `bas/` | Browser-automation fixtures | UI demo flows + extraction fixtures for the debug UI. | BAS-owned shape; see `bas/registry.json`. |
| `bas/{actions,cases,fixtures,flows,seeds}/` | BAS sub-zones | Standard BAS layout. | Per BAS scenario conventions. |

## API Surface

The full proto-typed product surface (excluding template `/health` REST probe). Every row is wire-traceable: the request/response messages live at `packages/proto/schemas/typescript-code-graph/v1/{graph,rewrite}/*.proto`; the CLI binding is declared in `cli/manifest.json` and dispatched by `cli/domains/<group>/`.

| RPC | Request | Response | Errors | CLI binding |
|---|---|---|---|---|
| `TypeScriptCodeGraphService.Extract` | `ExtractRequest{scenario_path}` | `ExtractResponse{graph, warnings, extraction_ms, graph_hash, sidecar_request_id}` | `InvalidArgument` (no tsconfig, multiple tsconfig), `NotFound` (path unreadable), `Unimplemented` (pnpm workspace), `Unavailable` (sidecar down), `DeadlineExceeded`, `Internal` | `typescript-code-graph graph extract <path>` |
| `TypeScriptCodeGraphService.RewritePlan` | `RewritePlanRequest{scenario_path, operations[]}` | `RewritePlanResponse{plan_id, normalized_operations}` | `InvalidArgument` (no ops, conflicting ops), `Internal` | `typescript-code-graph rewrite plan <ops.json>` |
| `TypeScriptCodeGraphService.RewriteApply` | `RewriteApplyRequest{scenario_path, plan_id, apply}` | `RewriteApplyResponse{plan_id, results[], dry_run}` | `InvalidArgument`, `FailedPrecondition` (plan unknown / expired), `Unavailable`, `Internal` | `typescript-code-graph rewrite apply <plan_id>` (honours `X-Dry-Run: true`) |
| `HealthService.Check` | `CheckRequest{}` | `CheckResponse{status, sidecar_status, ...}` | `Internal` | `typescript-code-graph status` |

Template-inherited REST exception: `GET /health` carries `RESTReasonOpsProbe`. There are no other REST endpoints.

## Sidecar Architecture

The scenario is the first in the template to host a long-lived non-Go child process. The Go side never calls `ts-morph` directly; it asks the Node sidecar over a line-delimited JSON stream on stdio.

```
+----------------------------------+        +------------------------------+
|              api/                |        |          sidecar/            |
|                                  |        |                              |
|  handlers/graph                  |        |   src/index.ts               |
|  handlers/rewrite                |        |     (stdio loop, framing,    |
|        |                         |        |      per-path lock,          |
|        v                         |        |      extract/rewrite ts-morph)|
|  internal/graph.Service          |        |                              |
|  internal/rewrite.Service        |        |              ^               |
|        |                         |        |              |               |
|        v                         |        |     line-delimited JSON      |
|  internal/sidecar.SidecarClient  |        |        on stdin/stdout       |
|        |                         |        |     stderr -> Go logger      |
|        v                         |        |              |               |
|  internal/sidecar.Supervisor     |  spawn |              |               |
|     (exec.CommandContext) -------+------> | node dist/index.js           |
|     handshake / heartbeat /      |        |                              |
|     restart-with-backoff         |        |                              |
+----------------------------------+        +------------------------------+
```

Boundary contracts:

- **One spawn point.** `internal/sidecar.Supervisor` is the only place that imports `os/exec`. Drift is caught by `internal/{graph,rewrite}/no_prod_import_test.go` and `internal/rewrite/no_external_command_test.go`.
- **One IPC contract.** `sidecar/src/protocol.ts` defines every message type; every request carries a UUID-v4 `request_id`; every response echoes it. Cancellation is best-effort: `{type:"cancel", request_id}` resolves the local future immediately, sidecar discards any late completion.
- **Two-layer per-path serialization.** Go-side `graph.PathMutex` and sidecar-side `Map<scenarioPath, Promise<void>>` chain both serialize. Defense in depth; a future caller that bypasses the Go side still sees correct semantics.
- **Stderr-only logging.** `console.log` to stdout would corrupt the framer. `sidecar/src/logger.ts` redirects all sidecar logs to stderr; the Go side reads stderr on a separate goroutine.
- **Shutdown order.** `Supervisor.Shutdown(ctx)` sends `{type:"shutdown"}`, waits up to the context deadline, then `cmd.Process.Kill()` if the child hasn't exited. **Footgun** (see `../internal/PROBLEMS.md`): the context passed to `Supervisor.Start` must not be cancelled before `Shutdown` runs — `exec.CommandContext` will kill the child as soon as the start context is cancelled, racing the orderly shutdown path.

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
