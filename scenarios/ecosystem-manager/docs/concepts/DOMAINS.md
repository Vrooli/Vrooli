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
| tasks | The unit of work: type (scenario/resource) × operation (generator/improver), target, priority, status. | `task_executions`; queue YAML | API, CLI, UI | `api/pkg/tasks/`, `cli/` task group, `ui/src/components/kanban/` |
| queue | Schedule, execute, rate-limit, and track task runs via agent-manager. | queue `<status>/` dirs; execution registry | API, CLI, UI | `api/pkg/queue/` |
| auto-steer | The improvement **control loop**: profiles, phase execution, stop conditions, metrics, history, analytics. | `profile_execution_state`, `profile_executions`; `profiles/*.json` | API, CLI, UI | `api/pkg/autosteer/`, `profiles/`, `ui/src/components/steer/` |
| steering | Selects the steering mode (profile / queue / manual / none) for a task. | `steering_queue_state` | API, UI | `api/pkg/steering/` |
| settings | Processor configuration (concurrency, auto-requeue, runner models). | settings state | API, UI | `api/pkg/settings/` |
| discovery | Catalog of improvable scenarios/resources and their PRD completion. | none (reads repo + `PRD.md`) | API, UI | `api/pkg/discovery/` |
| insights | Post-run analysis: structured feedback and applicable suggestions. | `execution_feedback_entries` | API, UI | `api/pkg/insights/` |
| prompts | Assemble agent prompts; sync the steer-skill catalog. | prompt files; skills cache | API | `api/pkg/prompts/` |
| logs | Date-stamped system audit trail. | `logs/<date>.log` | API, CLI, UI | `api/pkg/systemlog/` |

## Domain Details

### tasks

- Purpose: represent one improvement/generation job. Type × operation ×
  target × priority × status; status is the queue directory name.
- Owns: task records and run history (`task_executions`), task validation.
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
  (`ShouldContinueTask`) is where a steered task is re-evaluated and
  either requeued in the current phase, advanced, or stopped.
- Surfaces: `GET /api/queue/status`, `POST /api/queue/{trigger,start,stop,reset-rate-limit}`,
  `POST /api/queue/processes/terminate`, `POST /api/maintenance/state`.

### auto-steer

- Purpose: drive a target toward an objective by applying steer skills
  across iterations and deciding when to advance/stop. This is the
  domain the [control model](CONTROL-MODEL.md) describes.
- Owns: profiles (`profiles/*.json` + `metadata.json`), execution state
  (`profile_execution_state`), execution history (`profile_executions`),
  the metrics collectors, condition evaluation, phase coordination.
- Internal structure: orchestrator (`execution_orchestrator.go`),
  iteration evaluator, phase coordinator (`phase_coordinator.go`),
  condition evaluator (`evaluator.go`), metrics collectors (`metrics*.go`),
  state manager (Postgres), profile repository (filesystem),
  history/analytics services.
- Today vs target: today a profile is a fixed ordered phase list with
  metric exit gates (an open-loop schedule); the target is a closed-loop
  controller with diagnosis-driven selection and a learned effectiveness
  table — see [`CONTROL-MODEL.md`](CONTROL-MODEL.md).
- Surfaces: the `/api/auto-steer/*` route group; CLI `steer` group; UI
  steering panels.
- Tests: extensive unit tests (orchestrator, phase coordinator,
  evaluator, metrics) — see [`../internal/TESTING.md`](../internal/TESTING.md).

### steering

- Purpose: choose how a task is steered — a saved auto-steer profile, an
  ad-hoc mode queue, a manual single mode, or none.
- Owns: `steering_queue_state` (selected provider + queue order).
- Relationship: the profile provider delegates to auto-steer; the others
  are lighter-weight steering strategies.

### settings

- Purpose: processor tuning — concurrency, auto-requeue, runner/model
  selection propagated to agent-manager profiles.
- Surfaces: `GET/PUT /api/settings`, `POST /api/settings/reset`,
  `GET /api/settings/recycler/models`.

### discovery

- Purpose: enumerate improvable scenarios/resources and report each
  target's PRD completion (parsed from its `PRD.md`).
- Note: PRD completion is computed locally here, not via a wired
  `scenario-completeness-scoring` call — see
  [`INTEGRATIONS.md`](INTEGRATIONS.md).
- Surfaces: `GET /api/{resources,scenarios}`, `GET /api/operations`,
  `GET /api/categories`.

### insights

- Purpose: turn execution history into structured, severity-tagged
  suggestions an operator (or future controller) can apply.
- Owns: `execution_feedback_entries`.
- Surfaces: `GET/POST /api/tasks/{id}/insights*`.

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
| Profile | An auto-steer objective + phase configuration. Target model: an objective function. | auto-steer; [`CONTROL-MODEL.md`](CONTROL-MODEL.md) |
| Phase | One steering step within a profile (a skill set + stop conditions). | auto-steer |
| Metric / Finding | A measured gap (today: scalar metric; target: a `test-genie` finding). | auto-steer; [`CONTROL-MODEL.md`](CONTROL-MODEL.md) |
| Seam | Test-substitutable boundary wired once in production. | [`../internal/SEAMS.md`](../internal/SEAMS.md) |

## Deferred Domains

| Candidate Domain | Why Deferred | Revisit Trigger |
|---|---|---|
| Effectiveness store | Needs the controller's learning loop (v1). | When the closed-loop controller lands; see [`CONTROL-MODEL.md`](CONTROL-MODEL.md). |
| DTV trust gate | Needs development-toolchain-validator wiring (v2). | When skill eligibility/priors are consumed. |

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
