# Architecture — Ecosystem Manager

This document is the scenario's system map. It explains the shape of
Ecosystem Manager and points to the specialized documents that own
domains, the control model, workflows, data, integrations, operations,
and business strategy.

Read [`CONTROL-MODEL.md`](CONTROL-MODEL.md) **first**. This document
describes the *surfaces*; the control model describes the *intent* those
surfaces serve. Architecture without the control model reads like a CRUD
app; it is not one.

Keep this file high-signal. If a concern has a dedicated document below,
update that document and link it here.

## Purpose Of This Document

This document owns:

- the scenario's system shape and the role of each surface,
- how contracts and data flow between surfaces,
- the shared infrastructure boundary,
- extension rules for future code,
- architecture maturity and intentional deviations.

This document does not own:

- the improvement-loop mental model: [`CONTROL-MODEL.md`](CONTROL-MODEL.md),
- product capability inventory: [`DOMAINS.md`](DOMAINS.md),
- runtime workflows and state transitions: [`FLOWS.md`](FLOWS.md),
- storage details and retention: [`DATA.md`](DATA.md),
- resource and scenario dependencies: [`INTEGRATIONS.md`](INTEGRATIONS.md),
- test seams and fakes: [`../internal/SEAMS.md`](../internal/SEAMS.md),
- test strategy: [`../internal/TESTING.md`](../internal/TESTING.md).

## Scenario Shape

Ecosystem Manager is one product — an autonomous generation-and-improvement
**control plane** — expressed through three coordinated surfaces over a
Go core, plus two external execution dependencies.

```
        ┌──────────┐      REST/JSON      ┌──────────────┐    REST/JSON   ┌──────────┐
        │   ui/    │ ◀─────────────────▶ │    api/      │ ◀────────────▶ │   cli/   │
        │ React +  │      + WS /ws       │   Go core    │                │   Go     │
        │ Vite     │                     │ (gorilla/mux)│                └──────────┘
        └──────────┘                     └──────┬───────┘
                                                │
                  ┌─────────────────────────────┼─────────────────────────────┐
                  ▼                             ▼                              ▼
         ┌─────────────────┐        ┌──────────────────────┐        ┌──────────────────┐
         │ PostgreSQL      │        │ Filesystem stores    │        │ Scenario deps    │
         │ vrooli_         │        │ profiles/*.json      │        │ agent-manager    │
         │ ecosystem_      │        │ queue/<status>/*.yaml│        │ (executes every  │
         │ manager         │        │ logs/<date>.log      │        │ agent run)       │
         └─────────────────┘        └──────────────────────┘        └──────────────────┘
```

| Surface | Role | Owns | Does Not Own |
|---|---|---|---|
| API (`api/`) | Scenario core | Task lifecycle, the auto-steer control loop, persistence, agent-manager orchestration | Browser state, CLI formatting |
| UI (`ui/`) | Browser presentation | Kanban board, steering config panels, execution views, live updates over `/ws` | Business rules, persistence policy |
| CLI (`cli/`) | Operator/agent wrapper | Argument parsing, output formatting, API invocation | Business rules, control-loop logic |

The load-bearing principle is unchanged from the Vrooli standard: **the
API is the only surface with business logic.** UI and CLI translate
intent into API calls.

## System Boundaries

The scenario owns:

- source under `api/`, `ui/`, and `cli/`,
- generated-scenario docs under `docs/`,
- lifecycle metadata under `.vrooli/`,
- the auto-steer profile registry under `profiles/`,
- the task queue under `queue/<status>/`,
- the Postgres schema under `initialization/postgres/schema.sql`,
- proto schemas under `packages/proto/schemas/ecosystem-manager/`.

The scenario does not own:

- the agents it spawns (owned by `agent-manager`),
- the steer skills it applies (owned by `prompt-manager`),
- shared package implementations under `packages/`.

PRD completion percentage — a key control-loop metric — is computed
today by parsing each target's `PRD.md` locally
(`api/pkg/discovery/scenarios.go`), not by a wired
`scenario-completeness-scoring` call. See
[`INTEGRATIONS.md`](INTEGRATIONS.md) for the full, honest dependency
inventory.

Dependency decisions are documented in
[`INTEGRATIONS.md`](INTEGRATIONS.md), not here.

## Contracts And Data Flow

Ecosystem Manager predates the Vrooli proto/Connect-RPC transport
standard, and its current shape reflects that history:

- The HTTP API is **REST/JSON over `gorilla/mux`** (`api/pkg/server/server.go`).
  Handlers largely exchange JSON objects, not generated Connect types.
- Proto schemas **do** exist under
  `packages/proto/schemas/ecosystem-manager/` (domain + API messages),
  but today they back **agent-manager client serialization and UI-side
  validation**, not Connect-RPC *serving*. The HTTP edge is still REST.

This is a deliberate, recorded deviation — see
[Intentional Deviations](#intentional-deviations),
[`../internal/DECISIONS.md`](../internal/DECISIONS.md), and the migration
note in [`../internal/PROBLEMS.md`](../internal/PROBLEMS.md). New scenarios
should use proto + Connect from the start; Ecosystem Manager is on the
migration backlog, not the reference path.

The endpoint surface is enumerated in
[`../reference/api-endpoints.md`](../reference/api-endpoints.md). Live
updates flow UI-ward over the `/ws` WebSocket.

## Shared Infrastructure

Shared infrastructure is allowed only when the code is
business-vocabulary-free and used by unrelated domains.

| Package | Purpose | Consumers |
|---|---|---|
| `api/pkg/server/` | Compose handlers and middleware into one HTTP server; route registration. | API entrypoint, all handler groups. |
| `api/pkg/systemlog/` | Date-stamped structured audit log. | All domains; served via `GET /api/logs`. |
| `api/pkg/agentmanager/` | Outbound agent-manager client (run start/stop/stream). | Queue processor, auto-steer execution. |
| `api/pkg/websocket/` | Live UI push. | Queue/task state changes. |
| `api/pkg/internal/*` (paths, slices, timeutil) | Generic utilities. | Cross-domain. |

If a shared package starts using product vocabulary (tasks, profiles,
phases), move that piece into the owning domain.

## Extension Rules

Add product behavior by adding or updating the owning domain — see
[`DOMAINS.md`](DOMAINS.md) — not by growing generic buckets.

For a new HTTP capability today:

1. Add domain logic under `api/pkg/<domain>/`.
2. Add an HTTP handler under `api/pkg/handlers/` and register the route in
   `api/pkg/server/server.go`.
3. Add a UI API wrapper and feature under `ui/src/`.
4. Add a CLI command under `cli/` if operators/agents need it.
5. Update [`../reference/api-endpoints.md`](../reference/api-endpoints.md),
   [`../reference/cli-commands.md`](../reference/cli-commands.md), and the
   docs contract in `docs/manifest.json`.

For changes to the improvement loop itself (selection, state, metrics,
termination, thrashing), the design authority is
[`CONTROL-MODEL.md`](CONTROL-MODEL.md) — update it first, then the
auto-steer code under `api/pkg/autosteer/`.

## Architecture Maturity

| Area | Maturity | Evidence | Remaining Drift |
|---|---|---|---|
| API core | Production | Unified task/queue/auto-steer service; Postgres + filesystem stores; broad handler + unit test coverage. | REST/JSON transport instead of Connect-RPC. |
| Control loop | Transitional | Auto-steer runs metric-gated multi-phase profiles live. | Open-loop schedule, not yet the closed-loop controller in [`CONTROL-MODEL.md`](CONTROL-MODEL.md). |
| UI | Production | Kanban board, steering panels, execution views, live `/ws`. | Has not adopted the slot/adoption-resolver UI manifest (see [`UI-ARCHITECTURE.md`](UI-ARCHITECTURE.md)). |
| CLI | Production | Task/steer/queue/logs command groups over the API. | Thin by design. |
| Docs | Contract-ready (2026-05-30) | Manifest v2, this overhaul. | Maturity values to be grounded by Knowledge Observatory. |

## Intentional Deviations

| Date | Deviation | Reason | Revisit Trigger |
|---|---|---|---|
| pre-2026 | REST/JSON HTTP transport instead of proto/Connect-RPC | Scenario predates the Connect-RPC standard | Transport migration (backlog); see [`../internal/PROBLEMS.md`](../internal/PROBLEMS.md) |
| pre-2026 | Auto-steer profiles stored on the filesystem, not in Postgres | Profiles are human-authored, version-controlled config | If profiles become user-generated at runtime |
| pre-2026 | UI has not adopted the slot/adoption-resolver manifest | Scenario predates the UI manifest system | If the UI is regenerated from the template |
| 2026-05-30 | Docs describe the closed-loop controller ahead of implementation | Pin the model and vocabulary before code hardens | Implementation tracked in [`../internal/PROBLEMS.md`](../internal/PROBLEMS.md) |

## Documentation Architecture

| Concern | Canonical Document |
|---|---|
| Improvement-loop mental model | `docs/concepts/CONTROL-MODEL.md` |
| System map and extension rules | `docs/concepts/ARCHITECTURE.md` |
| Product capabilities and bounded contexts | `docs/concepts/DOMAINS.md` |
| Vocabulary | `docs/concepts/GLOSSARY.md` |
| Runtime workflows and state transitions | `docs/concepts/FLOWS.md` |
| Data ownership, retention, migrations | `docs/concepts/DATA.md` |
| Resources, scenarios, external services | `docs/concepts/INTEGRATIONS.md` |
| Deployment, runbook, observability | `docs/operations/*` |
| Seams, testing, errors, decisions, problems, progress | `docs/internal/*` |
| Monetization and go-to-market | `docs/business/*` |

Every durable document is registered in `docs/manifest.json`.

## Cross-References

- [`CONTROL-MODEL.md`](CONTROL-MODEL.md) — the improvement-loop intent (read first)
- [`DOMAINS.md`](DOMAINS.md) — bounded contexts and ownership
- [`FLOWS.md`](FLOWS.md) — control-loop state machine
- [`DATA.md`](DATA.md) — data ownership and storage
- [`INTEGRATIONS.md`](INTEGRATIONS.md) — dependency contracts
- [`../internal/SEAMS.md`](../internal/SEAMS.md) — seam registry
- [`../internal/TESTING.md`](../internal/TESTING.md) — test patterns
- [`../operations/DEPLOYMENT.md`](../operations/DEPLOYMENT.md) — deployment readiness
- [`../business/MONETIZATION.md`](../business/MONETIZATION.md) — commercial story
