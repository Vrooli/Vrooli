# Phased Plan Drain Mode

Phased-plan-drain mode runs an initiative through a stable sequential plan. Use
it when the work has an ordered handoff shape: one agent can complete the next
safe slice, persist context, and let a later round continue without losing the
thread.

## Phase Order

The backend computes startable phases from [CODE: api/internal/operatingmode/state.go].
The classifier decision after `classify_progress` determines the next phase.

```text
prepare_plan -> execute_next -> classify_progress
                      ^              |
                      |              +-- continue -> execute_next
                      |              +-- replan   -> prepare_plan
                      |              +-- complete -> review
                      +--------------+-- blocked  -> no next phase
```

Rules:

- `prepare_plan` is the first phase.
- `execute_next` requires a completed prior round with durable handoff,
  artifact, or progress context.
- `classify_progress` starts after completed `execute_next`.
- `classify_progress` with `continue` enables `execute_next`.
- `classify_progress` with `replan` enables `prepare_plan`.
- `classify_progress` with `complete` enables `review`.
- `classify_progress` with `blocked` enables no phase until the blockage is
  resolved by operator action or a new valid round state.
- `review` requires initiative acceptance criteria.
- Failed or canceled rounds do not advance the phase graph.
- Any reserved or agent-running round blocks all new phase starts.

## Operator Workflow

1. Create or select the initiative. New initiatives start in `item-level`.
2. Add initiative acceptance criteria before final review.
3. Switch mode through `mode-switch`; do not edit `mode` through generic
   initiative update.
4. Start `prepare_plan` and review `modes/phased-plan-drain/phased-plan.md`.
5. Start `execute_next` to complete the next contiguous slice of the plan.
6. Start `classify_progress` to decide whether to continue, replan, review, or
   stop on a blocker.
7. Repeat execution/classification until progress is complete.
8. Start `review` and validate the full initiative.
9. Use `complete-items` or `apply-backlog-sync` for audited backlog
   reconciliation. Agents must not edit member item `spec.json` files directly.

## Control Boundaries

- Registry and phase graph: [CODE: api/internal/operatingmode/registry.go]
- Sequential handoff validation: [CODE: api/internal/operatingmode/state.go]
- Phase start and AgentManager spawn: [CODE: api/internal/operatingmode/phase_runner.go]
- Progress parsing: [CODE: api/internal/operatingmode/progress.go]
- Round persistence: [CODE: api/internal/operatingmode/rounds.go]
- Backlog reconciliation audit: [CODE: api/internal/operatingmode/backlog_reconciler.go]
- Workspace rendering: [CODE: ui/src/components/initiative/operating-mode/phase-controls.tsx]

## Validation Checklist

- `execute_next` is rejected before `prepare_plan`.
- `review` is rejected until `classify_progress` returns `complete` and
  initiative acceptance criteria exist.
- Classifier decisions change only backend-provided phase action state; the UI
  does not duplicate that logic.
- Completed-item and proposal-applied events carry operating-mode source
  metadata for audit and stats.
