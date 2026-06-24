# Domains — Ecosystem Manager

This document is the canonical map of product capabilities, bounded
contexts, and ownership for Ecosystem Manager. Keep it current whenever a
domain is added, renamed, split, merged, or removed.

The center of gravity is **auto-steer** — the improvement control loop.
Read [`CONTROL-MODEL.md`](CONTROL-MODEL.md) for the intent behind it;
this document maps it (and its neighbors) to code.

## Purpose Of This Document

Use this document to answer:

- What capabilities does Ecosystem Manager expose?
- Which domain owns each concept, table, route, UI surface, and CLI command?
- Which concepts are shared, deferred, or deliberately not domains?

System-level architecture belongs in [`ARCHITECTURE.md`](ARCHITECTURE.md).
Workflow details belong in [`FLOWS.md`](FLOWS.md). Storage details belong
in [`DATA.md`](DATA.md).

## Domain Inventory

| Domain | Purpose | Owns Data | Surfaces | Source Paths |
|---|---|---|---|---|
| tasks | The unit of work: type (scenario/resource) × operation (generator/improver), target, priority, status. | queue YAML (run history lives in auto-steer `profile_executions`) | API, CLI, UI | `api/pkg/tasks/`, `cli/` task group, `ui/src/components/kanban/` |
| queue | Schedule, execute, rate-limit, and track task runs via agent-manager. | queue `<status>/` dirs; execution registry | API, CLI, UI | `api/pkg/queue/` |
| auto-steer | The closed-loop **controller**: objective profiles, diagnose (findings)→select→measure→terminate, decision trace, history, analytics. | `profile_execution_state`, `profile_executions`, `decision_trace`; `profiles/*.json` | API, CLI, UI | `api/pkg/autosteer/`, `api/pkg/{findings,skillmap,dimensions}/`, `profiles/`, `ui/src/components/steer/` |
| steering | Selects the steering mode (profile / queue / manual / none) for a task. | `steering_queue_state` | API, UI | `api/pkg/steering/` |
| settings | Processor configuration (concurrency, auto-requeue, runner models). | settings state | API, UI | `api/pkg/settings/` |
| discovery | Catalog of improvable scenarios/resources and their PRD completion. | none (reads repo + `PRD.md`) | API, UI | `api/pkg/discovery/` |
| prompts | Assemble agent prompts; sync the steer-skill catalog. | prompt files; skills cache | API | `api/pkg/prompts/` |
| logs | Date-stamped system audit trail. | `logs/<date>.log` | API, CLI, UI | `api/pkg/systemlog/` |

## Domain Details

### tasks

- Purpose: represent one improvement/generation job. Type × operation ×
  target × priority × status; status is the queue directory name.
- Owns: task records and validation; run history is recorded by
  auto-steer (`profile_executions`).
- Does not own: how a task is executed (queue) or steered (auto-steer).
- Surfaces: `GET/POST /api/tasks`, `PUT /api/tasks/{id}/status`; CLI
  `task` group (`add`/`improve`/`list`/`show`/`status`/`delete`); UI
  kanban cards.
- Tests: storage + handler tests under `api/pkg/tasks/` and handlers.

### queue

- Purpose: the processor that picks runnable tasks, enforces concurrency
  and rate-limit backoff, executes them via agent-manager, records
  history, and decides requeue-vs-stop for steered tasks.
- Owns: processor lifecycle, the execution registry, timeout watchdog.
- Key control-loop seam: `queue/autosteer_integration.go`
  (`ShouldContinueTask`) is where a steered task is re-evaluated by the
  controller (re-audit → select next skill, or finalize) and either
  requeued or stopped.
- Surfaces: `GET /api/queue/status`, `POST /api/queue/{trigger,start,stop,reset-rate-limit}`,
  `POST /api/queue/processes/terminate`, `POST /api/maintenance/state`.

### auto-steer

- Purpose: drive a target toward an objective by diagnosing its open
  findings, selecting the skill that best closes the heaviest dimension,
  re-auditing, and terminating on a global gradient. This is the domain
  the [control model](CONTROL-MODEL.md) describes.
- Owns: objective profiles (`profiles/*.json` + `metadata.json`),
  execution state (`profile_execution_state`), decision trace
  (`decision_trace`), execution history (`profile_executions`), the gap
  metrics collectors, and history/analytics services.
- Internal structure: controller orchestrator (`execution_orchestrator.go`),
  greedy `Selector` (`selector.go`), gradient `Terminator` (`terminator.go`),
  findings ingestion (`pkg/findings`), skill→dimension resolver
  (`pkg/skillmap`), dimension vocabulary (shared `packages/maturity-go/dimensions`),
  decision-trace store (`decision_trace.go`), state manager (SQLite), profile
  repository (filesystem), completeness measurement (`pkg/completeness`, for the
  operational-targets gate), and the anti-gaming promote-safety classifier
  (`pkg/autosteer/gameguard`).
- Status: **greedy controller** — diagnose open `test-genie` findings → select
  the skill targeting the heaviest profile-weighted dimension → execute via
  agent-manager → re-audit → terminate on objective-met / budget / diminishing
  returns. Baseline-safe via the optional `baseline_promote` engagement; a gamed
  run is blocked from promotion. See [`CONTROL-MODEL.md`](CONTROL-MODEL.md).
- Surfaces: the `/api/auto-steer/*` route group (incl.
  `GET /execution/{taskId}/trace`); CLI `steer` group (incl. `trace`); UI
  steering panels + decision-trace panel.
- Tests: unit tests for selector, terminator, controller loop, findings,
  skillmap, dimensions, profile validation/repository, gaming promote-gate, and
  handlers — see [`../internal/TESTING.md`](../internal/TESTING.md).

### steering

- Purpose: choose how a task is steered — a saved auto-steer profile, an
  ad-hoc mode queue, a manual single mode, or none.
- Owns: `steering_queue_state` (selected provider + queue order).
- Relationship: the profile provider delegates to auto-steer; the others
  are lighter-weight steering strategies.

### settings

- Purpose: processor tuning — concurrency, auto-requeue, runner/model
  selection propagated to agent-manager profiles, and the default-off
  importance-aware queue ordering flag.
- Surfaces: `GET/PUT /api/settings`, `POST /api/settings/reset`,
  `GET /api/settings/recycler/models`.

### discovery

- Purpose: enumerate improvable scenarios/resources and report each
  target's PRD completion (parsed from its `PRD.md`).
- Note: PRD completion is computed locally here, not via a wired
  `scenario-completeness-scoring` call; SCS is the operator-facing cached
  status reader over shared maturity/freshness packages — see
  [`INTEGRATIONS.md`](INTEGRATIONS.md).
- Surfaces: `GET /api/{resources,scenarios}`, `GET /api/operations`,
  `GET /api/categories`.

### prompts

- Purpose: assemble the agent prompt for a run (including the steering
  section) and synchronize the steer-skill catalog from prompt-manager.
- Surfaces: `GET /api/tasks/{id}/prompt[/assembled]`, `GET/POST /api/prompts*`,
  `GET /api/skills`, `POST /api/skills/sync`.

### logs

- Purpose: durable, human-readable audit trail of task/profile/error events.
- Surfaces: `GET /api/logs`; CLI `logs`.

## Shared Concepts

| Concept | Meaning | Owner |
|---|---|---|
| Task | The unit of generation/improvement work. | tasks domain |
| Profile | An auto-steer objective function: dimension weights, target thresholds, allowed-skill set, budget. | auto-steer; [`CONTROL-MODEL.md`](CONTROL-MODEL.md) |
| Dimension | A canonical improvement axis both findings and skills map to. | shared `packages/maturity-go/dimensions` |
| Finding | An open `test-genie` problem resolved to a dimension + severity; the controller's primary state. | auto-steer; `pkg/findings` |
| Gap metric | A scalar measurement (e.g. operational-targets %) used for the objective's operational target. | auto-steer |
| Seam | Test-substitutable boundary wired once in production. | [`../internal/SEAMS.md`](../internal/SEAMS.md) |

## Deferred Domains

| Candidate Domain | Why Deferred | Revisit Trigger |
|---|---|---|
| _None outstanding._ | — | — |

The auto-steer controller is intentionally greedy: it has no learned
effectiveness ledger, no development-toolchain-validator eligibility gate, and no
maturity-ladder rung governor. Those layers were removed in favor of a simple,
explainable "fix the heaviest open dimension" selection — see the *Design
History* section of [`CONTROL-MODEL.md`](CONTROL-MODEL.md).

## Non-Domains

These are infrastructure, not product domains:

- `api/pkg/server/` — HTTP composition and routing.
- `api/pkg/agentmanager/` — outbound agent-manager client.
- `api/pkg/websocket/` — live UI push.
- `api/pkg/systemlog/` is borderline; it is treated as the `logs` domain
  because it has a dedicated surface and operator value.
- `api/pkg/internal/*` — generic utilities.

If one of these starts using product vocabulary, split the product piece
into an owning domain.

## Cross-References

- [`CONTROL-MODEL.md`](CONTROL-MODEL.md) — the improvement-loop intent
- [`ARCHITECTURE.md`](ARCHITECTURE.md) — system shape and extension rules
- [`FLOWS.md`](FLOWS.md) — workflows and state transitions
- [`DATA.md`](DATA.md) — data ownership and storage
- [`../internal/SEAMS.md`](../internal/SEAMS.md) — boundary registry
- [`../internal/TESTING.md`](../internal/TESTING.md) — test strategy
