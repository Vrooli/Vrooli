# Architecture — Architecture Cartographer

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
                       │   architecture-cartographer/v1/...    │
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
| Contracts (`packages/proto/schemas/architecture-cartographer/`) | Wire shape | Proto messages/services and generated clients | Hand-written route/type mirrors |

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
  `packages/proto/schemas/architecture-cartographer/`.

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
packages/proto/schemas/architecture-cartographer/v1/<domain>/<file>.proto
       │
       ▼
       make generate
       │
       ├──▶ packages/proto/gen/go/architecture-cartographer/v1/...              (api, cli)
       ├──▶ packages/proto/gen/go/architecture-cartographer/v1/...connect       (Connect-Go)
       ├──▶ packages/proto/gen/typescript/js/architecture-cartographer/v1/...   (ui)
       └──▶ packages/proto/gen/python/architecture_cartographer/v1/...    (future tools)
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

## Code-Facts Substrate

Cartographer consumes `code-facts` for scenario surface and parse-unit
inventory. Code-facts answers "what execution surfaces and analyzer units
exist?" Cartographer answers "which domain owns this code and is the
architecture healthy?" The production API wires this through
`domains.SurfaceProvider`; if code-facts is unavailable, cartographer uses a
local filesystem fallback and emits a `code_facts.unavailable` extraction
warning so the audit can be treated as degraded rather than silently
authoritative.

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
   `packages/proto/schemas/architecture-cartographer/v1/<domain>/`.
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

Architecture Cartographer is the L5 "programmatic drift checks" tool
called out by the screaming-architecture audit. Its own architecture
is bound by exactly the rules it enforces on other scenarios.

| Area | Maturity | Evidence | Remaining Drift |
|---|---|---|---|
| API | Charter-defined | PRD published with 10 P0 targets; proto-first contract mandated; module registry pattern inherited from template. | Product domains (`graph`, `domains`, `conflicts`, `signals`, `apply`, `analytics`) implemented; the per-scenario architecture manifest was deleted in favor of zero-config derivation (the `domains` domain). |
| UI | Charter-defined | UX direction set to dense operational workbench; design tokens inherited from vrooli-default. | Conflict workbench, graph view, history dashboards all to be built. |
| CLI | Charter-defined | Human-friendly CLI contract specified (classified pattern + ranked fixes + evidence + caveats per conflict). | All `arch-cart` subcommands to be implemented. |
| Docs | Active | Manifest v2 registers all docs; required headings present in every concept doc; signal ladder and conflict model documented before implementation. | Maturity values flip to `draft` → `active` as each domain ships. |
| Contracts | Pre-implementation | Conflict envelope shape pinned in PRD and SIGNAL_LADDER doc; pluggable Detector / Resolver / Signal / Recipe interfaces defined as durable seams. | Proto schemas to be authored before any handler code. |

The cartographer dogfoods itself: its own scenario must pass cartographer
health checks against its own **derived** domain map (from
`docs/concepts/DOMAINS.md` via the extraction ladder — there is no per-scenario
manifest). This is the closure that proves the tool works.

Core structural detectors currently include `cycle`, `layering`, `naming`,
`glossary_drift`, `mislocated_file`, `convergence_drift`, `coupling_smell`,
`surface_coherence`, `file_cohesion`, `cross_scenario`, and
`domains_doc_parse_warning`. `layering` enforces wrong-direction dependency
rules using surface zones plus domain archetypes;
`naming` flags generic package/domain vocabulary such as `utils` or `helpers`
before those buckets can hide product ownership. `glossary_drift` uses the
curated DOMAINS.md glossary as evidence and flags exported symbols that carry
another domain's vocabulary. `surface_coherence` checks declared and
archetype-implied API/CLI/UI domain surfaces against actual implementation
evidence. `cross_scenario` blocks direct imports into another scenario's
private `api/internal` packages, keeping reuse on public API, CLI, or
shared-package contracts.

The detector registry applies per-surface profiles. API/Go receives coupling,
layering, convergence, naming, placement, cycles, cross-scenario boundary
checks, file cohesion, and surface coherence; CLI/Go receives layering,
convergence, naming, placement, cross-scenario boundary checks, and cycles;
UI/TypeScript receives naming, placement, cycles, and surface coherence. The
universal floor remains `cycle`, `naming`, and `mislocated_file`.

## Audit Contract (L5-Readiness)

The `audit` domain is the CI-shaped surface over the cartographer. The
contract below is pinned by the proto schema; humans and CI tools
depend on it.

- **Stable identity.** Every finding carries a deterministic
  `stable_id` (prefix `csid:`) derived from
  `(scenario, detector, type, subtype, sorted locations, sorted domains)`.
  Two runs that detect the same underlying drift collapse onto the
  same row. The per-run `instance_id` is preserved on the wire for log
  correlation.
- **Snapshot freshness.** The audit always invokes `ExtractGraph`,
  which is hash-aware: the response's `snapshot_freshness` reports
  `cached` (hash matched a persisted snapshot), `re-extracted` (prior
  snapshot existed with a different hash), or `fresh` (no prior).
  Stale-snapshot reuse is impossible.
- **Authority confidence is an outcome axis.** A scenario with
  `authority_confidence=low` (no curated DOMAINS.md or API manifest)
  flips the outcome to `FINDINGS` with `outcome_reason` set; callers
  opt back to `CLEAN` with `--allow-low-authority`.
- **Test Genie gates only trusted blockers.** The `ScenarioValidationService`
  packs the native `AuditRunResponse` into `native_detail`; Test Genie's
  architecture phase reads that authority field and only hard-fails blocker
  findings by default when `authority_confidence=high`.
- **Coverage is explicit.** Every audit response includes a `coverage`
  block with mutually-exclusive file buckets: `auto_place`, `suggest`,
  `conflict`, and `all_abstained`. `all_abstained` separates "the
  signals had no evidence" from a confident conflict, and the block
  repeats authority confidence for scripts that read only coverage.
- **Suppression is reported, never silent.** Findings sanctioned by
  active `// arch:allow` markers stay in `findings` and contribute to
  `suppressed_findings`; they do not flip the outcome.
- **Rollups.** `by_severity`, `by_type`, and `by_domain` are populated
  on every response; the human renderer surfaces all three.
- **Sweep.** `AuditService.RunAll` walks every directory under
  `<repo>/scenarios/` and aggregates per-scenario reports plus
  totals. CLI: `architecture-cartographer audit run-all`.

The domains surface also provides `domains draft <scenario>`, which
prints a proposed `docs/concepts/DOMAINS.md` inventory from the same
ladder evidence used by derivation. It is intentionally read-only:
humans or review agents must ratify purpose, ownership, archetype, and
glossary before committing the draft as authority.

`DomainsDocExtractor` now reports non-fatal parse warnings (e.g. rows
with the wrong column count) via the
[`domains_doc_parse_warning`](../../api/internal/conflicts/detectors/domainsparsewarning/)
detector. Silently-skipped rows are no longer a mystery.

## Intentional Deviations

Record deviations from the template or from Vrooli scenario standards
when they are deliberate and durable.

| Date | Deviation | Reason | Revisit Trigger |
|---|---|---|---|
| 2026-05-21 | Layered scenario design: cartographer depends on two new scenarios (`go-code-graph`, `typescript-code-graph`). Both initialized 2026-05-23 (PRD + requirements + docs in place at `scenarios/go-code-graph/` and `scenarios/typescript-code-graph/`); neither implemented. | Cartographer must not parse source code itself — language parsing is a separate concern that belongs in language-specific scenarios so multiple consumers (cartographer, react-component-library) can share it. See [`INTEGRATIONS.md`](INTEGRATIONS.md) for the dependency contract. | Revisit if the layering proves too granular in practice; consolidate only when a measured friction point emerges. |
| 2026-05-21 | Analytics (SQLite-backed) shipped in v0.1, not deferred to P1 like most scenarios. | Analytics is a precondition for both ladder calibration and recipe-emergence detection. Ranked CLI suggestions depend on having a minimum N=5 historical sample before showing success rates; that requires capturing data from the first conflict resolution onward. | None — this is permanent. |
| 2026-05-21 | No embedding/Ollama dependency in v1, even though `scenario-dependency-analyzer` already proves the pattern. | Auto-placement requires deterministic signals; embeddings introduce silently-wrong placements. Deterministic ladder (path tokens, import clusters, importer voting, test coupling, symbol glossary, git co-edit) is expected to cover the vast majority of placements. Embeddings are a P2 *suggestion* mechanism, never auto-applied. | Promote to v2 only if measured residual conflicts the deterministic ladder cannot answer are high *and* embedding ranking demonstrably reduces them in offline evaluation. |

## Signal Ladder And Conflict Model

Two concepts have their own canonical homes because they are
load-bearing for the entire cartographer workflow:

- [`SIGNAL_LADDER.md`](SIGNAL_LADDER.md) — pluggable scoring signals
  for chunk-to-domain auto-placement, weights, thresholds, and
  explainability contract.
- [`DOMAINS.md`](DOMAINS.md) (see `signals` and `conflicts` domains)
  — the Detector / Resolver / Signal / Recipe interface seams.

These must be read together to understand how the cartographer arrives
at "auto-place," "suggest with evidence," or "conflict, agent
decides."

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
