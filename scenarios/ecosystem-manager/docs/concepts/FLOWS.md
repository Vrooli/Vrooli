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
| Auto-steer control loop | auto-steer / queue | A steered task is picked up | Objective met, diminishing returns detected, or the iteration budget is hit | Per-task execution state in SQLite; iteration counter + decision trace | Level 1 (inventory) |
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
- Per-iteration sequence (greedy controller):
  1. **Diagnose** — re-audit via test-genie; `pkg/findings` buckets the open
     findings by dimension and severity.
  2. **Select** (greedy) — rank dimensions by profile weight × open severity
     and run the first eligible skill targeting the heaviest actionable
     dimension. `execution_orchestrator.go::GetCurrentSet` returns the single
     selected skill; the ranking lives in `selector.go::SelectNextSkill` and
     the skill→dimension map in `pkg/skillmap`.
  3. **Execute** — the queue processor assembles a prompt with the selected
     skill's steering section and runs the agent via agent-manager.
  4. **Measure** — re-audit and read the completeness `Score` from
     scenario-completeness-scoring (`pkg/completeness`); append a decision-
     trace entry (chosen skill, heaviest dimension, score before/after).
  5. **Decide requeue vs stop** — `terminator.go::ShouldStop` halts on
     objective-met / diminishing-returns / budget; otherwise
     `queue/autosteer_integration.go::ShouldContinueTask` requeues the task
     for the next iteration.
- Termination is global, never per-phase: there are no phases, no per-phase
  exit gates, and no quality-gate halts — see
  [`CONTROL-MODEL.md`](CONTROL-MODEL.md).

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
              │  requeue (next iter)  └────┬─────┘
              │   net-improving, budget    │ run complete
              │   remaining                ▼
              │                     ┌────────────┐
              └─────────────────────│ EVALUATING │
                                    └─────┬──────┘
                          terminator: ShouldStop
                    ┌────────────────────┴────────────────────┐
                    ▼                                          ▼
              objective met                       budget hit / diminishing returns
                    │                                          │
                    ▼                                          ▼
              ┌──────────┐                               ┌──────────┐
              │ COMPLETE │                               │  HALTED  │
              └──────────┘                               └──────────┘
```

| State | Meaning | Persisted |
|---|---|---|
| QUEUED | Awaiting a processor slot | queue directory |
| RUNNING | Agent executing this iteration | `profile_execution_state` |
| EVALUATING | Measurement read; terminator deciding next action | `profile_execution_state` |
| COMPLETE | Objective met (no finding above `max_open_severity`, op-targets gated) | `profile_executions` |
| HALTED | Iteration budget exhausted or diminishing returns detected | `profile_executions` |

Illegal transitions: requeuing after the terminator says stop; evaluating
without a completeness `Score` (measurement is load-bearing for termination
and does not fail open — see `pkg/completeness`).

## Maturity Ladder

Ecosystem Manager's flows are **executable but not formally modeled**.
They sit at Level 1 on the Vrooli temporal-flow ladder.

| Level | Name | Status here |
|---|---|---|
| 0 | Unmodeled risk | Skill-catalog sync. |
| 1 | Inventory | Task lifecycle, auto-steer control loop (this document). |
| 2 | Workflow model | Not yet — transition logic lives in handlers/coordinators, not a pure transition function. |
| 3 | Matrix + traces | Auto-steer has strong *unit* coverage of selection/termination logic, but not flow-matrix replay. |
| 4 | Declarative contract | None. |
| 5 | Checked formal model | None. Ecosystem Manager does not use `flow-verifier`. |

Raising the auto-steer loop to Level 2+ (a pure transition function with
`CheckInvariants`, replayed by tests) is a natural companion to the
controller model in [`CONTROL-MODEL.md`](CONTROL-MODEL.md): the
diagnose→select→execute→measure cycle and its termination conditions are
worth pinning formally.

## Production Shape

Unlike template scenarios, Ecosystem Manager does **not** use the
`flow-verifier` `flow/` + generated-Quint scaffold. Its flow logic lives
directly in domain code:

- Decision logic: `api/pkg/autosteer/{execution_orchestrator,selector,terminator,eligibility}.go`.
- Execution + requeue: `api/pkg/queue/autosteer_integration.go`.
- Persisted state: SQLite `profile_execution_state` /
  `profile_executions` (see [`DATA.md`](DATA.md)).

The side-effect boundaries around these flows (agent-manager client,
completeness provider, test-genie audit runner, repositories, clock) are the seams listed in
[`../internal/SEAMS.md`](../internal/SEAMS.md); tests substitute them to
drive transitions deterministically.

## Deferred / Unmodeled Flows

| Flow | Risk | Next Step |
|---|---|---|
| Skill catalog sync | Low — refresh is idempotent | Promote to Level 1 if staleness causes selection errors. |
| Auto-steer control loop | Medium — diagnose→select→execute→measure cycle with global termination | Promote to Level 2+ (pure transition function + replayed invariants) if loop bugs warrant formal modeling. |

## Cross-References

- [`CONTROL-MODEL.md`](CONTROL-MODEL.md) — why the loop is shaped this way
- [`DOMAINS.md`](DOMAINS.md) — owning domains
- [`DATA.md`](DATA.md) — persisted execution state and retention
- [`../internal/SEAMS.md`](../internal/SEAMS.md) — side-effect boundaries
- [`../internal/TESTING.md`](../internal/TESTING.md) — control-loop testing
