# Start Here — Ecosystem Manager

This is the orientation document for an agent picking up work on
ecosystem-manager. Unlike a freshly generated scenario, this is an
**existing, mature** internal control plane — there is no generation
protocol to run and no template scaffold to replace. Your job is to
understand the control loop already in place before you change it.

Read this page, then read the documents in "Read These First" in
order, then work.

## Orientation

Ecosystem Manager is Vrooli's internal **control plane for generating
and improving scenarios and resources**. It runs autonomous agent
loops — "auto-steer" — that drive a target (a scenario or resource)
toward a goal, dispatching work through `agent-manager` and measuring
progress against collected metrics until a stop condition is met.

The shape of the system:

- **Tasks** are the unit of work. Each has a `type`
  (`scenario` | `resource`), an `operation` (`generator` | `improver`),
  a target name, and an optional auto-steer profile.
- **The queue** orchestrates task execution: prompt assembly, agent
  dispatch, execution history, and rate-limit backoff.
- **Auto-steer** is the closed-loop controller: it advances a target
  through profile phases, evaluates collected metrics against stop
  conditions, and decides whether to iterate again or stop.
- The **REST/JSON API** (Go) holds all business logic. The **React/Vite
  UI** (kanban board) and the **CLI** are thin clients over it.

Surfaces: API + UI served at the dashboard `http://localhost:30500`;
state in Postgres (`vrooli_ecosystem_manager`) plus filesystem
(`profiles/` for auto-steer profile definitions, `queue/<status>/`
YAML for task state).

## Read These First

Read these in order. The first one is the most important document in
the scenario — do not skip it.

1. [`concepts/CONTROL-MODEL.md`](concepts/CONTROL-MODEL.md) — **read
   this first.** The closed-loop controller model: target, goal,
   metrics, stop conditions, and the iterate/stop decision. Everything
   else in ecosystem-manager is plumbing around this model. The whole
   leverage of this scenario is determined by how good the controller
   is, so internalize this before touching anything.
2. [`concepts/ARCHITECTURE.md`](concepts/ARCHITECTURE.md) — the mental
   model: tasks, queue, auto-steer, and the boundaries between them.
3. [`concepts/DOMAINS.md`](concepts/DOMAINS.md) — the domain map
   (tasks, queue, auto-steer/steering, profiles) and where each lives
   in `api/pkg/`.
4. [`internal/PROBLEMS.md`](internal/PROBLEMS.md) — known issues, sharp
   edges, and open questions. Check this before assuming a behavior is
   a bug you introduced.

## Working Agreements

- **Use the lifecycle, never direct-exec.** Start/stop with
  `vrooli scenario {start,stop,restart,status} ecosystem-manager` or
  the scenario Makefile (`make start|stop|logs|test`). Never run
  `./api/ecosystem-manager-api` directly — it bypasses process naming,
  port allocation, and health checks.
- **Respect the dependencies.** Ecosystem-manager needs Postgres
  (`vrooli-postgres-main`) and the running scenarios `agent-manager`
  (executes tasks) and `scenario-completeness-scoring` (supplies the
  PRD/completeness metric). If those are down, the queue stalls or
  metrics go unavailable — see [`guides/troubleshooting.md`](guides/troubleshooting.md).
- **Change business logic in `api/pkg/`.** The UI and CLI are thin;
  fixing behavior there is almost always wrong. Add the capability to
  the API, then surface it through the CLI/UI.
- **Run the tests** with `vrooli scenario test ecosystem-manager`
  before and after non-trivial changes.
- **Capture reusable work.** Record non-trivial fixes via
  `swarm-manager records create`; file out-of-scope defects via
  `report-bug`.

## Architecture Rules

- **The REST API is the core.** All business logic lives in the Go API
  under `api/pkg/`. The UI (kanban board) and CLI are thin clients.
  *Deviation note:* the project-wide default is proto + Connect-RPC for
  every scenario API. Ecosystem-manager predates that convention and
  intentionally serves **REST/JSON** today. Do not "modernize" it to
  Connect as a side quest; treat the REST surface as the current
  contract and only migrate as a deliberate, planned piece of work.
- **Auto-steer is the heart.** The closed-loop controller is the reason
  this scenario exists and the lever on its value. Improvements to the
  controller (better metrics, smarter stop conditions, better phase
  sequencing) raise leverage on *every* target it touches. Read
  [`concepts/CONTROL-MODEL.md`](concepts/CONTROL-MODEL.md) before
  changing it.
- **Business logic in `api/pkg/`.** Domains (tasks, queue, steer/
  auto-steer, profiles, handlers) live here. Keep transitions and side
  effects (terminate process, force start, wake) surfaced by the
  lifecycle layer and executed by callers — don't bury side effects in
  pure logic.
- **UI and CLI are thin.** They configure tasks and visualize state
  over the HTTP API. Profile *definitions* live on disk (`profiles/`);
  active auto-steer *state* lives in Postgres. Don't duplicate control
  logic in the clients.

## Cross-references

- [`concepts/CONTROL-MODEL.md`](concepts/CONTROL-MODEL.md) — the
  closed-loop controller (read first)
- [`concepts/ARCHITECTURE.md`](concepts/ARCHITECTURE.md) — system mental model
- [`concepts/DOMAINS.md`](concepts/DOMAINS.md) — domain map
- [`QUICKSTART.md`](QUICKSTART.md) — get it running locally
- [`guides/troubleshooting.md`](guides/troubleshooting.md) — common failures
- [`internal/PROBLEMS.md`](internal/PROBLEMS.md) — known issues
- [`operations/RUNBOOK.md`](operations/RUNBOOK.md) — operational procedures
- [`business/MONETIZATION.md`](business/MONETIZATION.md) — role in the portfolio
