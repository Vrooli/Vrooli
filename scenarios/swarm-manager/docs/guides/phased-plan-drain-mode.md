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
- `execute_next` requires a completed prior round and an initiative
  `plan_ref` so the prompt can receive plan-manager phase context.
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
4. Start `prepare_plan`; it creates or updates the canonical plan-manager plan
   and binds its `plan_ref` to the initiative.
5. Start `execute_next` to complete the next contiguous slice of the plan.
6. Start `classify_progress` to decide whether to continue, replan, review, or
   stop on a blocker.
7. Repeat execution/classification until progress is complete.
8. Start `review` and validate the full initiative.
9. Use `complete-items` or `apply-backlog-sync` for audited backlog
   reconciliation. Agents must not edit member item `spec.json` files directly.

## Preview the flow (simulation presets)

The operating-mode detail page's **Flow** tab can walk this graph deterministically
without running agents. Phased-plan-drain ships these presets as mode-owned
example-run data ([CODE: modes/phased-plan-drain/example-runs/]), each walking the
real transition guards:

- **Drains in one slice** (`happy-path`) — classify reports `complete` → `review` →
  `reconcile`.
- **Continue to next slice** (`continue-next-slice`) — the first execute round
  finishes one comprehensively-completable slice and hands off the true `frontier`;
  classify reports `continue`, routing back to `execute_next` to advance the declared
  remainder, which then classifies `complete`. This is the elastic-slice contract in
  action.
- **Progress forces a replan** (`replan-plan`) — classify reports `replan`, routing
  back to `prepare_plan`; the revised plan drains cleanly.
- **Work is blocked** (`blocked`) — classify reports `blocked`, a terminal decision
  with no downstream target; the cycle ends before review for operator intervention.
- **Review requests changes** (`review-changes-requested`) — after the plan drains
  and classify reports `complete`, review returns `verdict=changes_requested`, so the
  guard routes **back to `execute_next`** for one more slice that closes the gaps;
  classify then reports `complete`, review accepts, and reconcile aligns the backlog.

Presets are deterministic previews, not real initiative rounds. In the Flow tab,
the data-source control swaps the same phase viewer between the **Contract**
template, a **Simulation** preset, and a **Live** round; the Instructions tab
renders the literal agent prompt for whichever source is selected.

## Control Boundaries

- Registry and phase graph: [CODE: api/internal/operatingmode/registry.go]
- Sequential handoff validation: [CODE: api/internal/operatingmode/state.go]
- Phase start and AgentManager spawn: [CODE: api/internal/operatingmode/phase_runner.go]
- Plan-manager context injection: [CODE: api/internal/operatingmode/plan_ref.go]
- Round persistence: [CODE: api/internal/operatingmode/rounds.go]
- Backlog reconciliation audit: [CODE: api/internal/operatingmode/backlog_reconciler.go]
- Workspace rendering: [CODE: ui/src/components/initiative/operating-mode/phase-composer.tsx]

## Validation Checklist

- `execute_next` is rejected before `prepare_plan`.
- `review` is rejected until `classify_progress` returns `complete` and
  initiative acceptance criteria exist.
- Classifier decisions change only backend-provided phase action state; the UI
  does not duplicate that logic.
- Completed-item and proposal-applied events carry operating-mode source
  metadata for audit and stats.
