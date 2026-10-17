# Architecture — Personal Planner

This document is the scenario's system map. It explains the shape of
Personal Planner — an adaptive personal planning and focus application
with the operator-approved *Observatory* day/night visual direction —
and points to the specialized documents that own product domains,
workflows, data, integrations, deployment, operations, and business
strategy.

Keep this file high-signal. Do not turn it into a warehouse for every
domain, endpoint, workflow, or decision. If a concern has a dedicated
document below, update that document and link it here.

The load-bearing product principles this architecture exists to protect
are **truthful accounting** (capacity, remaining effort, dependencies,
delays, and forecasts never silently lie green) and **calm, beautiful
UX** (the Observatory scene is the wedge that earns the daily open). The
architecture keeps those two promises honest by isolating a pure
scheduling kernel from every side effect and routing every mutation
through a single authoritative command surface.

## Purpose Of This Document

This document owns:

- the scenario's system shape and the role of each surface,
- the boundary between the pure planning kernel and everything with a
  side effect (persistence, auth, HTTP, rendering),
- the command service envelope (actor, workspace, idempotency, revision,
  outbox) that every mutation passes through,
- how contracts and data flow between surfaces,
- the shared-infrastructure boundary,
- the per-domain extension order (proto → API → transport → CLI → UI),
- the distinction between authoritative and derived data,
- architecture maturity and intentional deviations, including the
  Observatory theme's deliberate departure from the default operational
  console kit.

This document does not own:

- product capability inventory: [`DOMAINS.md`](DOMAINS.md),
- temporal and user/system workflows: [`FLOWS.md`](FLOWS.md),
- storage details, revisions, and retention: [`DATA.md`](DATA.md),
- resource and scenario dependencies: [`INTEGRATIONS.md`](INTEGRATIONS.md),
- test seams and fakes: [`../internal/SEAMS.md`](../internal/SEAMS.md),
- test strategy: [`../internal/TESTING.md`](../internal/TESTING.md),
- deployment and operations: [`../operations/DEPLOYMENT.md`](../operations/DEPLOYMENT.md),
- commercial strategy: [`../business/MONETIZATION.md`](../business/MONETIZATION.md).

## Scenario Shape

Personal Planner is one product expressed through three coordinated
surfaces, one canonical contract layer, and — uniquely for this
scenario — a **pure deterministic kernel** that the API surface owns but
that is deliberately fenced off from every side effect.

```
                       ┌─────────────────────────────┐
                       │  Generated proto types      │
                       │  packages/proto/schemas/    │
                       │   personal-planner/v1/...    │
                       └──────────────┬──────────────┘
                                      │ canonical wire shape
              ┌───────────────────────┼───────────────────────┐
              │                       │                       │
              ▼                       ▼                       ▼
        ┌──────────┐    Connect  ┌──────────┐    Connect  ┌──────────┐
        │   ui/    │◀──────────▶ │  api/   │◀──────────▶ │  cli/   │
        │ React    │    -RPC     │   Go     │    -RPC     │   Go     │
        │ + Vite   │             │ HTTP     │             │ cli-core │
        │ (RCL     │             │  edge    │             │          │
        │  shell)  │             └────┬─────┘             └──────────┘
        └──────────┘                  │
                                      ▼
                        ┌──────────────────────────┐
                        │  Command service          │
                        │  actor · workspace ·      │
                        │  idempotency · revision · │
                        │  transaction · outbox     │
                        └───────────┬──────────────┘
                                    │ immutable snapshot in
                                    ▼
                        ┌──────────────────────────┐
                        │  Pure planning kernel     │
                        │  (scheduling · capacity · │
                        │   forecast) — no I/O      │
                        └───────────┬──────────────┘
                                    │ candidate placements + reasons out
                                    ▼
                              ┌───────────┐
                              │  SQLite   │
                              │ per-domain│
                              │  schema   │
                              └───────────┘
```

| Surface | Role | Owns | Does Not Own |
|---|---|---|---|
| API (`api/`) | Scenario core | Business rules, the pure kernel, persistence, integrations, command service, transport edge | Browser state, CLI formatting |
| UI (`ui/`) | Browser presentation over the react-component-library shell | Observatory theme, components, i18n, accessibility, appearance model, browser interaction | Business rules, persistence policy, scheduling math |
| CLI (`cli/`) | Operator/agent wrapper | Argument parsing, output formatting, API invocation at parity with the UI | Business rules, duplicated validation, a second command path |
| Contracts (`packages/proto/schemas/personal-planner/`) | Wire shape | Proto messages/services and generated clients | Hand-written route/type mirrors |

The load-bearing principle: the API is the only surface that contains
business logic, and within the API the **pure kernel is the only place
that computes schedules, capacity, and forecasts**. UI and CLI translate
user/operator intent into API calls; they never re-implement scheduling
math. Proto types flow from one source of truth so wire-shape drift
between surfaces is impossible. This is what lets the UI, CLI text, and
API all display *the same values* (INV-16) rather than three independent
approximations.

## System Boundaries

The scenario owns:

- source code under `api/`, `ui/`, and `cli/`,
- the pure kernel under `api/internal/planning/`,
  `api/internal/capacity/`, and `api/internal/forecasts/`,
- the shared command envelope under `api/internal/command/` and the
  temporal helpers under `api/internal/temporal/`,
- generated-scenario docs under `docs/`,
- scenario lifecycle metadata under `.vrooli/`,
- scenario-specific requirements under `requirements/`,
- scenario proto schemas under
  `packages/proto/schemas/personal-planner/`.

The scenario does not own:

- shared package implementation under `packages/`,
- Vrooli resource implementation,
- identity, which belongs to scenario-authenticator (required),
- notification delivery, which belongs to notification-hub (optional),
- provider secrets, which live in the credential-storage owner,
- external calendar events, which the provider owns and this scenario
  imports strictly read-only as revisioned projections,
- source-app content and authoritative domain status, which each source
  app owns and this scenario references as projections,
- generated proto outputs under `packages/proto/gen/`.

The single most load-bearing ownership rule is **one authoritative owner
per mutable fact** (INV-02): the planner owns native planning records,
accepted allocations, availability, private activity, forecast
snapshots, proposals, and sharing projections; imported and referenced
objects retain their origin identity and revision. Document dependency
and resource decisions in [`INTEGRATIONS.md`](INTEGRATIONS.md), not here.

## Contracts And Data Flow

Wire shapes do not live in TypeScript interfaces, Go structs, or
hand-written JSON schemas. They live in `.proto` files. Connect-RPC is
the default transport for this scenario. For a proto-typed API call the
`.proto` file also declares the service block that generates Connect
handlers and clients.

```
packages/proto/schemas/personal-planner/v1/<domain>/<file>.proto
       │
       ▼
       make generate
       │
       ├──▶ packages/proto/gen/go/personal-planner/v1/...              (api, cli)
       ├──▶ packages/proto/gen/go/personal-planner/v1/...connect       (Connect-Go)
       ├──▶ packages/proto/gen/typescript/personal-planner/v1/...   (ui)
       └──▶ packages/proto/gen/python/personal_planner/v1/...    (future tools)
```

Use Connect-RPC by default:

- UI to API for proto-typed payloads,
- CLI to API for proto-typed payloads,
- API to API / inter-scenario calls with Vrooli-owned protos, including
  the shared scheduling-intent and schedule-query contracts that
  consumer scenarios (Daily, Cadence, agent work) call.

REST is allowed only for four enumerated reasons, defined as
`RESTReason` constants in `api/internal/module/module.go`:

| Reason | When it applies |
|---|---|
| `RESTReasonMultipartUpload` | Opaque file bytes via `multipart/form-data` (an ICS import upload is the canonical case here). |
| `RESTReasonWebhookReceiver` | Endpoint shape is dictated by a third-party system (e.g. a provider push channel) we do not own. |
| `RESTReasonThirdPartyShape` | Request or response is an externally-defined contract (an OAuth provider-connection callback). |
| `RESTReasonOpsProbe` | Lifecycle systems, load balancers, and `curl` must reach the endpoint without a generated client (plain `GET /health`). |

Mechanical enforcement: `cmd/gen-endpoints` rejects any
`EndpointDescriptor.Path` that is not a generated Connect procedure
constant unless the descriptor carries a `RESTException` with one of the
four reasons. A REST endpoint without that tag fails `make endpoints`,
which fails `make test`, which fails CI. Even for REST exceptions the
**payload shape** stays proto-typed wherever possible — an ICS import
handler accepts multipart bytes but returns a proto-typed validation
report.

### The command envelope

Every mutation — from HTTP, CLI, internal tools, or a source adapter —
flows through **one application-command layer** (`api/internal/command/`,
plan §20.1, OT-P0-007). That layer, not the transport, enforces the
consistent behavior the product depends on:

- **Actor and workspace** come from authenticated context, never from a
  client-supplied tenant identifier (INV-01).
- **Idempotency keys** are accepted where effects could be retried,
  scoped by caller, command type, and workspace, so a timed-out
  proposal-apply or focus-start can be safely re-sent.
- **Expected-revision checks** guard overwrites; opaque revision tokens
  are compared for equality only, never ordered numerically.
- **Transactions** are drawn around commands, not renders: an allocation
  split writes its allocation changes, its effort-conservation check, its
  revision, its history, and its **outbox record** together, or none of
  them.
- The **outbox** captures domain events (`AllocationMoved`,
  `CommitmentRevisionAccepted`, `SessionFinished`, …) in the same local
  transaction as the mutation, and delivers at-least-once to consumers
  and to notification-hub afterward, so a restart between commit and
  delivery replays safely rather than losing or duplicating an effect.

Commands return the canonical stored record and resulting revision, not
merely `success: true`, so a caller can reconcile its own state. The
error model is a fixed, safe-by-construction vocabulary
(`VALIDATION_FAILED`, `REVISION_CONFLICT`, `PROPOSAL_STALE`,
`SESSION_ALREADY_ACTIVE`, `SOURCE_UNAVAILABLE`, `PROVIDER_REAUTH_REQUIRED`,
…) that never leaks raw provider errors, SQL, private payloads, or hidden
record IDs.

### The pure kernel

Scheduling, capacity, and forecasting are computed by a **pure,
side-effect-free Go kernel** (plan §22.1). The kernel accepts an
immutable normalized snapshot (availability, busy facts, work demand,
constraints, an explicit clock and timezone service, and a seeded
deterministic tie-break) and returns candidate placements with reason
codes and unresolved demand. It **never** reads or writes the database,
calls auth, touches HTTP, renders anything, or counts a hypothetical
forecast placement as a reservation (INV-15). Generation never mutates
accepted records; apply is a *separate* command that reloads the stored
proposal, revalidates every input against current revisions, and commits
atomically (INV-11). This isolation is what makes the planner
deterministic, unit-testable against fixtures (F01–F16), and honest:
the same snapshot always yields the same explainable result.

### Authoritative versus derived data

The system draws a hard line between facts it authors and facts it
computes:

| Kind | Examples | Rule |
|---|---|---|
| Authoritative (owned) | Native work, effort revisions, accepted allocations, events/routines, commitments and their revision history, focus sessions and actuals, availability, share grants | Stored as current state plus revisioned history; edited only through revision-checked commands. |
| Derived (computed) | Capacity intervals/budgets, proposals, forecast snapshots, review read models, viewer projections, search indexes | Reproducible from authoritative inputs plus an input fingerprint; a read model is a disposable projection, never a second authority. |
| Projected (external) | Source-object projections, imported provider events | Read-only copies carrying origin identity and source revision; only the verified adapter updates source-owned fields (INV-02, INV-10). |

Derived results carry the fingerprint of the inputs they were computed
from, so the UI can show freshness and detect staleness rather than
presenting an out-of-date forecast as current. Storage details, revision
semantics, and retention live in [`DATA.md`](DATA.md); the temporal and
system flows that move data between these kinds live in
[`FLOWS.md`](FLOWS.md).

## Shared Infrastructure

Shared infrastructure is allowed only when the code is
business-vocabulary-free and used by unrelated domains or surfaces.

| Package/Folder | Purpose | Why Not Domain-Owned | Consumers |
|---|---|---|---|
| `api/internal/server/` | Compose modules and middleware into one HTTP server. | Server lifecycle is not a product capability. | API entrypoint and handler modules. |
| `api/internal/module/` | Shared module and endpoint descriptor types. | Domain modules return this common shape. | Handler packages, server, endpoint codegen. |
| `api/internal/modules/` | Thin registry for schemas and endpoints. | Boot/codegen need central lists; logic stays domain-owned. | `main.go`, `gen-endpoints`. |
| `api/internal/command/` | The command envelope (actor/workspace/idempotency/revision/transaction/outbox). | Cross-cutting; every domain mutation shares it. | All domain services. |
| `api/internal/temporal/` | Timezone, interval, and recurrence/instance-identity helpers. | A library many domains call; owns no state. | calendar, capacity, planning, forecasts. |
| `api/internal/database/` | System schema, per-domain schema registration seam, and DB reachability. | Cross-cutting DB infrastructure, not one domain's data. | API boot, health, domain repositories. |
| `api/internal/clock/` | Deterministic time seam injected into date-sensitive logic and the kernel. | Time is cross-cutting and test-substitutable. | Middleware, repositories, kernel. |
| `api/internal/testutil/` | Cross-domain test harnesses and fakes. | Used by unrelated domains; domain fakes stay domain-local. | API tests. |
| `ui/src/layout/` | react-component-library shell configuration (shell constants, nav data, brand mark). | Shell is configured, not redrawn; not a product capability. | UI entrypoint, features. |
| `ui/src/components/` | Shared presentation primitives and the illustrative reusable domain components. | Used by unrelated UI features and both the full planner and compact source-app widgets. | UI features. |
| `ui/src/test-utils/` | Cross-feature render helpers, a11y helpers, and model tests. | Used by unrelated UI features. | UI tests. |

The pure kernel (`planning`/`capacity`/`forecasts`) is **not** generic
infrastructure — it is product logic and lives in its owning domains.
If shared infrastructure starts using product vocabulary, move that piece
back into the owning domain or split a new domain first.

## Extension Rules

Add product behavior by adding or updating the owning domain, not by
growing generic buckets. For a normal proto-backed domain, extend in this
fixed order so contracts lead and every surface stays in sync:

1. **Proto** — add messages and service methods under
   `packages/proto/schemas/personal-planner/v1/<domain>/` and run
   `make generate`.
2. **API domain** — add domain code under `api/internal/<domain>/`,
   including a repository behind an interface and a domain-local
   `schema.sql`; route every mutation through `api/internal/command/`.
3. **Transport** — add handler code under `api/handlers/<domain>/`
   translating Connect requests into command/query calls.
4. **CLI** — add commands under `cli/domains/<domain>/` at parity with
   the UI, emitting both machine-readable JSON and concise human output.
5. **UI** — add API wrappers under `ui/src/api/<domain>.ts` and feature
   code under `ui/src/features/<domain>/`, composing the shared domain
   components rather than a per-screen task card.
6. **Register and document** — register schemas/endpoints in
   `api/internal/modules/registry.go`, mount the module in
   `api/main.go`, update selectors/strings/tests, and update the docs
   contract in `docs/manifest.json`.

Scheduling, capacity, or forecast changes additionally extend the pure
kernel first (as a snapshot-in/result-out change with fixtures) and only
then wire the command and transport around it. For detailed product
ownership update [`DOMAINS.md`](DOMAINS.md); for persistence and
retention update [`DATA.md`](DATA.md); for temporal behavior update
[`FLOWS.md`](FLOWS.md).

## Architecture Maturity

**This scenario is at documentation-first initialization.** The
contracts, the thirteen product domains, the derived/authoritative data
model, and the module boundaries are *modeled and documented*; **there is
no product code yet**. Nothing in this scenario is "implemented" — the
tables below describe the target shape to build against and the template
scaffolding that ships with a generated scenario, not shipped product
behavior.

| Area | Maturity | Evidence | Remaining Work |
|---|---|---|---|
| Contracts / domains | Designed | 13 product domains + `health` infra domain modeled in `DOMAINS.md`; API surface and command envelope modeled in this doc and the plan (§20, §22). | Author proto services per domain; no `.proto`, Go, or TS product code exists yet. |
| Pure kernel | Designed | Snapshot-in/result-out contract and invariants (INV-06, INV-11, INV-15) specified; fixtures F01–F16 enumerated. | Implement the deterministic solver and its fixtures. |
| API | Template scaffold + design | Generated `react-vite` template ships module registry, per-domain schema seam, and a fenced `notes` example; command envelope is designed, not built. | Replace the fenced `notes` example with the real domains; build the command service. |
| UI | Template scaffold + design | Observatory theme and the navigated-console shell configuration are designed in `DESIGN.md`/`EXPERIENCE.md`; template shell exists. | Configure `AppShell`, build the five destinations and domain components. |
| CLI | Template scaffold + design | Logical verbs (`schedule.query`, `proposal.apply`, `focus.start`, …) enumerated in the plan (§21). | Build domain command groups at UI parity. |
| Docs | In progress | This doc, `DOMAINS.md`, `DATA.md`, `FLOWS.md`, `INTEGRATIONS.md`, `DESIGN.md`, and `EXPERIENCE.md` authored; registered in `docs/manifest.json`. | Fill remaining stubs (SEAMS, TESTING, operations, business) as they become real. |

The removable `notes` example domain is a copyable worked vertical slice,
never product scope; `template-manager detemplate personal-planner`
removes every fenced example once the first real domain is green. Use
`docs/manifest.json` as the documentation contract; declared `maturity`
values are maintained by agents and later grounded by Knowledge
Observatory validation.

## Intentional Deviations

Record deviations from the template or from Vrooli scenario standards
when they are deliberate and durable.

| Date | Deviation | Reason | Revisit Trigger |
|---|---|---|---|
| 2026-09-18 | **Observatory theme instead of the default operational-console kit.** The UI intentionally departs from the console's "avoid atmospheric backgrounds / compact-operational-only" defaults, adding a coordinated day/night decorative landscape band, editorial serif headings, and comfortable/editorial density. | Operator-approved decision D08. Beauty is a stated product requirement: the day/night scene is the wedge that earns the daily open, and repeated daily use is what makes the honest planning loop valuable (see [`../../DESIGN.md`](../../DESIGN.md), [`EXPERIENCE.md`](EXPERIENCE.md)). It is built on the Vrooli token plumbing and shell — it does not fork the design system. | If the navigated-console archetype cannot carry the decorative full-bleed scene band, record a scoped `shell-ejection` in `../reference/component-library-gaps.md` naming the exact `ui/src/` files. |
| 2026-09-18 | **Pure kernel fenced from all side effects**, held to a snapshot-in/result-out contract rather than a service that reads the DB directly. | Determinism, explainability, and honest arithmetic (INV-11, INV-15); the kernel must be unit-testable against fixtures with no persistence, auth, HTTP, or rendering in scope. | Revisit only if a measured performance need forces streaming inputs; even then the purity boundary is preserved. |
| 2026-09-18 | **Connect-RPC scheduling contracts, not a shared database, as the ecosystem integration API.** Consumer scenarios (Daily, Cadence, agent work) submit normalized scheduling intent and read accepted schedules through proto contracts. | Personal Planner is the single scheduling authority (INV-02); a shared database would create duplicate write authorities. Where a consumer is absent, ship the contract plus a fixture adapter and keep the live-integration gate explicitly open. | When a real consumer integrates and the contract earns a versioned revision. |

## Documentation Architecture

Scenario docs follow the same ownership rule as code: one durable
question, one canonical home. The `docs/` tree is organized into six
areas, each registered in `docs/manifest.json` (the documentation
contract that carries maturity, stage, and validation hints for each
document):

| Area | Purpose | Representative Documents |
|---|---|---|
| `docs/concepts/` | The durable mental model of the product and system. | `ARCHITECTURE.md` (this file), `DOMAINS.md`, `DATA.md`, `FLOWS.md`, `INTEGRATIONS.md`, `EXPERIENCE.md`, `UI-ARCHITECTURE.md` |
| `docs/guides/` | Task-oriented how-tos for building within the scenario. | `choosing-ui.md`, onboarding and contribution guides |
| `docs/reference/` | Lookup material and enumerations. | `api-endpoints.md`, `component-library-gaps.md` |
| `docs/internal/` | Engineering-facing contracts. | `SEAMS.md`, `TESTING.md`, `SECURITY.md`, `PROBLEMS.md`, `PROGRESS.md` |
| `docs/operations/` | Running the deployed scenario. | `DEPLOYMENT.md`, `RUNBOOK.md`, `OBSERVABILITY.md` |
| `docs/business/` | Commercial framing. | `MONETIZATION.md`, `GO-TO-MARKET.md` |

Alongside `docs/`, the binding UI design contract lives at the scenario
root in [`../../DESIGN.md`](../../DESIGN.md), and machine-readable UX
specs live under `experience/`. Put deep domain-specific documentation
under `docs/domains/<domain>/` when `DOMAINS.md` would become noisy.
Every durable scenario document should be registered in
`docs/manifest.json`.

| Concern | Canonical Document |
|---|---|
| System map and extension rules | `docs/concepts/ARCHITECTURE.md` |
| Product capabilities and bounded contexts | `docs/concepts/DOMAINS.md` |
| Workflows and state transitions | `docs/concepts/FLOWS.md` |
| Data ownership, revisions, retention | `docs/concepts/DATA.md` |
| Resources, scenarios, external services | `docs/concepts/INTEGRATIONS.md` |
| UI decision and information architecture | `docs/concepts/EXPERIENCE.md` |
| Shell/slot taxonomy and components | `docs/concepts/UI-ARCHITECTURE.md` |
| Binding UI design language | `DESIGN.md` |
| Monetization and packaging | `docs/business/MONETIZATION.md` |
| Deployment tiers and readiness | `docs/operations/DEPLOYMENT.md` |
| Seams and test doubles | `docs/internal/SEAMS.md` |
| Testing strategy | `docs/internal/TESTING.md` |

## Cross-References

- [`DOMAINS.md`](DOMAINS.md) — bounded contexts and ownership
- [`FLOWS.md`](FLOWS.md) — workflow and state-transition map
- [`DATA.md`](DATA.md) — data ownership, revisions, and storage
- [`INTEGRATIONS.md`](INTEGRATIONS.md) — dependency contracts
- [`EXPERIENCE.md`](EXPERIENCE.md) — UI decision and information architecture
- [`UI-ARCHITECTURE.md`](UI-ARCHITECTURE.md) — shell configuration and components
- [`../../DESIGN.md`](../../DESIGN.md) — the binding Observatory design contract
- [`../business/MONETIZATION.md`](../business/MONETIZATION.md) — why beauty is the wedge
- [`../operations/DEPLOYMENT.md`](../operations/DEPLOYMENT.md) — deployment readiness
- [`../internal/SEAMS.md`](../internal/SEAMS.md) — seam registry
- [`../internal/TESTING.md`](../internal/TESTING.md) — test patterns
- [`../internal/PROBLEMS.md`](../internal/PROBLEMS.md) — known issues / tech debt
- [`../internal/PROGRESS.md`](../internal/PROGRESS.md) — lifecycle log
