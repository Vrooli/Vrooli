# Architecture — Data Backup Manager

This document is the scenario's system map. It explains the invariant
shape inherited from the `react-vite` template, then points to the
specialized documents that own product domains, workflows, data,
integrations, deployment, operations, and business strategy.

Data Backup Manager provides Vrooli a dependable, engine-backed backup
and verified-restore capability. Owning scenarios self-register the
runtime state they own; the manager snapshots that state on a schedule
to one or more encrypted destinations and proves it can restore. It
exists to make runtime state safe to keep **out of git** — removing
"we commit it so we don't lose it" as a justification. It **wraps** the
`kopia` resource for all repository, snapshot, restore, dedup,
encryption, compression, retention, and stats work; it does not
hand-roll crypto, dedup, or an external orchestrator.

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
                       │   data-backup-manager/v1/...    │
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
| Contracts (`packages/proto/schemas/data-backup-manager/`) | Wire shape | Proto messages/services and generated clients | Hand-written route/type mirrors |

The load-bearing principle: the API is the only surface that contains
business logic. UI and CLI translate user/operator intent into API
calls. Proto types flow from one source of truth so wire-shape drift
between surfaces is impossible.

## Domain Model

The product vocabulary is fixed and decouples *what is backed up* from
*where it lands*. The full ownership map lives in
[`DOMAINS.md`](DOMAINS.md); the core relationships are:

```
   Owning scenario
        │ self-registers (owner+name)
        ▼
     TARGET ──────┐                ┌────── DESTINATION (= kopia repository)
   (a source:     │  many-to-many  │       (local fs or S3/MinIO,
    fs / sqlite /  ├──── PLAN ──────┤        encrypted by default,
    postgres /     │  + schedule    │        per-destination cap)
    redis /        │  + retention   │
    qdrant /       └────────────────┘
    object-store)
        │
        ▼
       RUN  ──snapshot──▶  DESTINATION repository (via resource-kopia)
        │
        ▼
     RESTORE  (restore to a location, or VERIFY to scratch + checksum)
```

- **Target** — a source registered by an owning scenario. Unique key is
  `owner + name`. Carries one of six **source kinds** (filesystem,
  SQLite, Postgres, Redis, Qdrant, object-storage), a locator, and
  optional pre/post quiesce hooks (P1). Source secrets come from vault.
- **Destination** — where artifacts land: one kopia repository, backed
  by a local filesystem path or S3/MinIO. Encrypted by default;
  credentials/passphrases from `vault`; **must not** point under the
  storage root it protects (separate-root rule); carries a configurable
  storage cap defaulting to **alert + block**.
- **Plan** — binds targets to destinations (many-to-many) with a
  schedule and retention. One target may be in several plans
  (daily-to-local *and* weekly-to-offsite).
- **Run** — one execution of a plan; records run history and
  last-success-per-target.
- **Restore** — restores a target to a location; **verify** mode
  test-restores to scratch and checksums (records last-verified).
  A verified restore is the gate before any committed runtime data is
  removed from git.
- **Discovery** (onboarding helper, not a noun in the core model) — a
  read-only domain that scans the local environment and *suggests*
  targets to protect (well-known `~/.vrooli` runtime state) and
  destinations to back up to (mounted volumes, removable drives first).
  Suggestions are derived (only dismissals persist); accepting one calls
  the existing `RegisterTarget` / `CreateDestination`, so it never
  becomes a second write path. The OS volume scan is confined behind the
  `internal/sysmounts` seam (the one place `gopsutil` is imported).

Self-registration mirrors agent-manager's `EnsureProfile`: scenarios
re-register idempotently on boot, so the catalog is reconstructable and
the manager's SQLite store is a cache plus run history, **not** the
single source of truth. Backup artifacts never live under the scenario
source tree — they live in kopia repositories. See
[`INTEGRATIONS.md`](INTEGRATIONS.md) for the resource contracts and
[`DATA.md`](DATA.md) for the two-store split.

## System Boundaries

The scenario owns:

- source code under `api/`, `ui/`, and `cli/`,
- generated-scenario docs under `docs/`,
- scenario lifecycle metadata under `.vrooli/`,
- scenario-specific requirements under `requirements/`,
- scenario proto schemas relocated to
  `packages/proto/schemas/data-backup-manager/`.

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
packages/proto/schemas/data-backup-manager/v1/<domain>/<file>.proto
       │
       ▼
       make generate
       │
       ├──▶ packages/proto/gen/go/data-backup-manager/v1/...              (api, cli)
       ├──▶ packages/proto/gen/go/data-backup-manager/v1/...connect       (Connect-Go)
       ├──▶ packages/proto/gen/typescript/js/data-backup-manager/v1/...   (ui)
       └──▶ packages/proto/gen/python/data_backup_manager/v1/...    (future tools)
```

Use Connect-RPC by default:

- UI to API for proto-typed payloads,
- CLI to API for proto-typed payloads,
- API to API / inter-scenario calls with Vrooli-owned protos.

REST is allowed only for four enumerated reasons, defined as
`RESTReason` constants in `api/internal/module/module.go`:

| Reason | When it applies |
|---|---|
| `RESTReasonMultipartUpload` | Opaque file bytes via `multipart/form-data`. No current backup-manager product endpoint needs this exception. |
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

Note: even for REST exceptions, the **payload shape** should stay
proto-typed wherever possible. Drift between API/UI/CLI is eliminated
as long as the wire payload type is shared.

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
   `packages/proto/schemas/data-backup-manager/v1/<domain>/`.
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

The template ships a mature shape and a reference vertical slice. As of the
API+CLI implementation pass, all six backup-manager domains are **built and
green** (Connect-RPC services, per-domain SQLite schema, wrapped kopia/source
seams). The UI remains a designed-only follow-up. Implementation status lives
in `requirements/` and [`../internal/PROGRESS.md`](../internal/PROGRESS.md).

| Area | Maturity | Evidence | Remaining Drift |
|---|---|---|---|
| API | Built — 5 Connect-RPC services + health rollup | `targets/destinations/plans/runs/restores` Connect services in `api/handlers/<domain>/`, transport-free domain cores in `api/internal/<domain>/`, per-domain schema, `validateTransport` + proto-parity gates green. 25 endpoints in `.vrooli/endpoints.json`. | P1/P2 features (quiesce hooks, GFS retention, restore granularity); owner-filter on `ListTargetStatus` (see PROBLEMS). |
| CLI | Built — command per RPC + self-registration | `cli/domains/<domain>/` wraps each RPC via generated Connect clients; manifest-driven; `RequireProtoServiceCoverage` per domain. | — |
| UI | Designed-only (explicit follow-up plan) | Feature folders + typed client patterns from the template. | Destinations (usage-vs-cap), plans, run history, guided restore/verify to be built per [`UI-ARCHITECTURE.md`](UI-ARCHITECTURE.md). |
| Docs | Filled to the locked design; reconciled to built state | Concept + reference docs; SEAMS registry updated with KopiaEngine/CommandRunner/Capturer; PROBLEMS tracks deferrals. | — |
| Engine integration | Built — wrapped behind the KopiaEngine seam | `api/internal/engine/kopia.go` shells out to the fully-implemented `resource-kopia` CLI; encryption always on; kopia owns repo passphrases via vault. Real-engine paths covered by integration tests gated on `KOPIA_INTEGRATION` / source resources. Snapshot browsing is backed by `resource-kopia snapshot browse --json`. | Source-resource integration coverage remains gated. |

### API surface (built)

| Domain | Service | Methods |
|---|---|---|
| targets | `TargetsService` | RegisterTarget, DeregisterTarget, GetTarget, ListTargets |
| destinations | `DestinationsService` | CreateDestination, GetDestination, ListDestinations, UpdateDestination, DeleteDestination, GetDestinationUsage |
| plans | `PlansService` | CreatePlan, GetPlan, ListPlans, UpdatePlan, DeletePlan |
| runs | `RunsService` | TriggerRun, GetRun, ListRuns, ListTargetStatus, BrowseSnapshot |
| restores | `RestoresService` | RestoreTarget, VerifyTarget, GetRestore, ListRestores |
| health | (REST ops-probe) | GET /health (degrades on overdue/failed backups) |

No REST exceptions beyond the template's `GET /health` ops-probe — every product operation is proto-expressible.

Use `docs/manifest.json` as the documentation contract. The declared
`maturity` values are expected to be maintained by agents and later
grounded by Knowledge Observatory validation.

## Intentional Deviations

Record deviations from the template or from Vrooli scenario standards
when they are deliberate and durable.

| Date | Deviation | Reason | Revisit Trigger |
|---|---|---|---|
| 2026-05-26 | Hard dependency on the `kopia` and `vault` resources (the template default is standalone SQLite). | Wrap-not-use: the engine and secret store are foundational, not optional integrations. | n/a — foundational to the design. |
| 2026-05-26 | Two stores: the manager's SQLite catalog is a cache + run-history anchor, not the single source of truth; backup artifacts live in kopia repositories outside the source tree. | The registration model is reconstructable from scenario re-registration on boot; artifacts must never live under the protected source tree. | Revisit if catalog loss ever needs stronger durability than re-registration provides. |
| 2026-05-26 | Storage cap default is alert + block, never silent eviction. | A backup tool that deletes backups to stay under a cap is unsafe; eviction is only ever explicit retention. | Revisit only if a requirement justifies an opt-in eviction tier (none planned). |
| 2026-05-26 | Supersedes the prior n8n + MinIO source-tree backup design (now in `/tmp`). | That design backed up the repo source tree via an external orchestrator — explicitly the wrong model. | n/a — the old design is not revived. |

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
