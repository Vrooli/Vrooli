# Flows — Ecosystem Manager

This document is the canonical workflow and state-transition map for
Ecosystem Manager. The scenario is fundamentally about *ordered,
stateful, long-running loops*, so this document matters more here than in
a CRUD scenario.

The headline flow is the **auto-steer control loop** — the runtime
realization of [`CONTROL-MODEL.md`](CONTROL-MODEL.md). Read the control
model for *why* the loop is shaped this way; this document records *what
states it moves through*.

## Purpose Of This Document

Use this document to answer:

- Which workflows have explicit, ordered states?
- What are the legal and illegal transitions?
- Where does each transition live in code?
- How mature (formally modeled) is each flow?

## Flow Inventory

| Flow | Domain | Trigger | Outcome | Statefulness | Maturity |
|---|---|---|---|---|---|
| Task lifecycle | tasks / queue | Task created or status changed | Task reaches a terminal status | Status = queue directory name; transitions are atomic file moves | Level 1 (inventory) |
| Auto-steer control loop | auto-steer / queue | A steered task is picked up | Profile completes, halts on a quality gate, or hits the iteration cap | Per-task execution state in SQLite; phase/iteration counters | Level 1 (inventory) |
| Skill catalog sync | prompts | Manual or scheduled sync | Local steer-skill cache refreshed | Cache freshness | Level 0 (in-code) |

## Flow Details

### Task lifecycle

- Owner: tasks + queue.
- States (each a `queue/<status>/` directory): `pending`,
  `in-progress`, `completed`, `failed`, `blocked`, plus archival/template
  directories. The directory name is the single source of truth for
  status.
- Transitions: atomic file moves between status directories — handlers do
  not mutate the status field in place. On startup the storage layer
  re-syncs status from directory names and de-duplicates task IDs.
- Illegal transitions: inventing a new status directory; editing the
  status field inside the YAML instead of moving the file.
- Code: `api/pkg/tasks/`, `api/pkg/queue/`.

### Auto-steer control loop

This is the loop the whole scenario exists to run.

- Owner: auto-steer (decision) + queue (execution + requeue).
- Per-iteration sequence (current implementation):
  1. **Select** the current phase's skill set —
     `execution_orchestrator.go::GetCurrentSet` returns
     `profile.Phases[CurrentPhaseIndex].SkillIDs`.
  2. **Execute** — the queue processor assembles a prompt with the
     steering section and runs the agent via agent-manager.
  3. **Measure** — `metrics.go::CollectMetrics` gathers a
     `MetricsSnapshot` (universal + UX/test/refactor/perf/security).
  4. **Evaluate** — `phase_coordinator.go::ShouldAdvancePhase` checks the
     iteration cap and the phase's stop conditions
     (`evaluator.go::Evaluate`, recursive AND/OR).
  5. **Decide requeue vs advance vs stop** —
     `queue/autosteer_integration.go::ShouldContinueTask`: continue in the
     current phase, advance (`AdvancePhase`, after quality-gate checks),
     or finalize.
- Quality gates: a gate with `failure_action: halt` blocks advancement
  (`phase_coordinator.go::ShouldHaltOnQualityGates`).
- Target evolution: the control model replaces step 1's fixed-order
  selection with diagnosis-driven selection, and step 4's per-phase exit
  gates with global gradient termination plus runtime thrashing
  detection. See [`CONTROL-MODEL.md`](CONTROL-MODEL.md).

## State Machines

### Auto-steer execution state

```
                 start
                   │
                   ▼
              ┌─────────┐   pick up task
              │ QUEUED  │ ─────────────────┐
              └─────────┘                  ▼
                                      ┌──────────┐
              ┌───────────────────────│ RUNNING  │
              │  requeue (same phase) └────┬─────┘
              │   no stop condition met    │ run complete
              │                            ▼
              │                     ┌────────────┐
              └─────────────────────│ EVALUATING │
                                    └─────┬──────┘
                         stop condition / cap met
                    ┌────────────────────┼────────────────────┐
                    ▼                     ▼                    ▼
              advance phase         all phases done      quality gate halt
            (reset iteration)             │                    │
                    │                     ▼                    ▼
                    └──────────▶      ┌──────────┐        ┌──────────┐
                       (QUEUED)       │ COMPLETE │        │  HALTED  │
                                      └──────────┘        └──────────┘
```

| State | Meaning | Persisted |
|---|---|---|
| QUEUED | Awaiting a processor slot | queue directory |
| RUNNING | Agent executing this iteration | `profile_execution_state` |
| EVALUATING | Metrics collected; deciding next action | `profile_execution_state` |
| COMPLETE | All phases finished | `profile_executions` |
| HALTED | A quality gate blocked advancement | `profile_executions` |

Illegal transitions: advancing past the final phase without finalizing;
evaluating without a collected `MetricsSnapshot`; referencing a metric in
a stop condition that was not collected (raises `MetricUnavailableError`
in `evaluator.go::GetMetricValue`).

## Maturity Ladder

Ecosystem Manager's flows are **executable but not formally modeled**.
They sit at Level 1 on the Vrooli temporal-flow ladder.

| Level | Name | Status here |
|---|---|---|
| 0 | Unmodeled risk | Skill-catalog sync. |
| 1 | Inventory | Task lifecycle, auto-steer control loop (this document). |
| 2 | Workflow model | Not yet — transition logic lives in handlers/coordinators, not a pure transition function. |
| 3 | Matrix + traces | Auto-steer has strong *unit* coverage of phase/stop logic, but not flow-matrix replay. |
| 4 | Declarative contract | None. |
| 5 | Checked formal model | None. Ecosystem Manager does not use `flow-verifier`. |

Raising the auto-steer loop to Level 2+ (a pure transition function with
`CheckInvariants`, replayed by tests) is a natural companion to the
controller work in [`CONTROL-MODEL.md`](CONTROL-MODEL.md), because the
controller adds new states (diagnosing, learning) and new illegal
transitions (thrashing) worth pinning formally.

## Production Shape

Unlike template scenarios, Ecosystem Manager does **not** use the
`flow-verifier` `flow/` + generated-Quint scaffold. Its flow logic lives
directly in domain code:

- Decision logic: `api/pkg/autosteer/{execution_orchestrator,phase_coordinator,evaluator,iteration_evaluator}.go`.
- Execution + requeue: `api/pkg/queue/autosteer_integration.go`.
- Persisted state: SQLite `profile_execution_state` /
  `profile_executions` (see [`DATA.md`](DATA.md)).

The side-effect boundaries around these flows (agent-manager client,
metrics provider, repositories, clock) are the seams listed in
[`../internal/SEAMS.md`](../internal/SEAMS.md); tests substitute them to
drive transitions deterministically.

## Deferred / Unmodeled Flows

| Flow | Risk | Next Step |
|---|---|---|
| Skill catalog sync | Low — refresh is idempotent | Promote to Level 1 if staleness causes selection errors. |
| Controller learning loop | High once built — new states + thrashing transitions | Model at Level 2+ alongside the controller implementation. |

## Cross-References

- [`CONTROL-MODEL.md`](CONTROL-MODEL.md) — why the loop is shaped this way
- [`DOMAINS.md`](DOMAINS.md) — owning domains
- [`DATA.md`](DATA.md) — persisted execution state and retention
- [`../internal/SEAMS.md`](../internal/SEAMS.md) — side-effect boundaries
- [`../internal/TESTING.md`](../internal/TESTING.md) — control-loop testing
