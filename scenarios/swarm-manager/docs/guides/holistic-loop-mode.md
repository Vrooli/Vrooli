# Holistic Loop Mode

Holistic-loop mode runs an initiative as one coupled unit of work. Use it when
member items touch the same substrate, item boundaries are likely to shift, or
the only meaningful validation question is whether the system works as a whole.

## Phase Order

The backend computes startable phases from [CODE: api/internal/operatingmode/state.go].
The UI and CLI render that state; they do not decide sequencing locally.

```text
investigate -> plan -> execute -> review
      ^             |
      |             v
      +------ replan_needed=true
```

Rules:

- `investigate` is the first phase.
- `plan` can start after completed `investigate`.
- `execute` can start after completed `plan`.
- `execute` with `replan_needed=false` enables `review`.
- `execute` with `replan_needed=true` enables `investigate`.
- `review` requires initiative acceptance criteria.
- Failed or canceled rounds do not advance the phase graph.
- Any reserved or agent-running round blocks all new phase starts.

## Operator Workflow

1. Create or select the initiative. New initiatives start in `item-level`.
2. Add initiative acceptance criteria before the review phase is needed.
3. Switch mode through `mode-switch`; do not edit `mode` through generic
   initiative update.
4. Start `investigate` and review the generated findings artifact.
5. Start `plan` and review `modes/holistic-loop/initiative-plan.md`.
6. Start `execute`. If the result requests replanning, loop back through
   investigation. If not, proceed to review.
7. Start `review` and validate the whole initiative against acceptance criteria.
8. Use `complete-items` or `apply-backlog-sync` for any backlog status or scope
   reconciliation. Agents must not edit member item `spec.json` files directly.

## Control Boundaries

- Registry and phase graph: [CODE: api/internal/operatingmode/registry.go]
- Phase start and AgentManager spawn: [CODE: api/internal/operatingmode/phase_runner.go]
- Prompt fail-closed behavior: [CODE: api/internal/operatingmode/prompt.go]
- Round refresh and terminal result parsing: [CODE: api/internal/operatingmode/round_refresher.go]
- Artifact application: [CODE: api/internal/operatingmode/artifact_applier.go]
- Backlog reconciliation audit: [CODE: api/internal/operatingmode/backlog_reconciler.go]
- Workspace hook: [CODE: ui/src/components/initiative/operating-mode/use-operating-mode-workspace.ts]

## Validation Checklist

- The workspace shows only backend-startable phase actions as enabled.
- Starting `review` before acceptance criteria returns a backend validation
  error.
- Prompt catalog failure fails the reserved round and does not spawn
  AgentManager.
- Completed-item events include source metadata for initiative, mode, phase,
  round, run ID, requested-by, and affected item refs.
