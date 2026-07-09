# Holistic Loop Mode

Holistic-loop mode runs an initiative as one coupled unit of work. Use it when
member items touch the same substrate, item boundaries are likely to shift, or
the only meaningful validation question is whether the system works as a whole.

## Phase Order

The backend computes startable phases from [CODE: api/internal/operatingmode/state.go].
The UI and CLI render that state; they do not decide sequencing locally.

```text
investigate -> plan -> execute (executed_by: phased-plan-drain) -> review -> reconcile
      ^                 |  ^_|                  |
      |                 |  progress=continue    v (verdict=changes_requested -> execute)
      +--- progress=blocked
```

Rules:

- `investigate` is the first phase.
- `plan` can start after completed `investigate`. It authors the canonical
  plan-manager plan and binds it to the initiative (`plan_ref`, required
  output) — that bound plan is what the delegated drain executes.
- `execute` can start after completed `plan`. It is **delegated** to the
  generic `phased-plan-drain` (`executed_by`): each round is one drain slice,
  the classified edge derives `progress` from the handoff, and
  `progress=continue` keeps `execute` startable again (the inline drain loop).
- `execute` with `progress=complete` enables `review`.
- `execute` with `progress=blocked` enables `investigate` (the composed
  replan loop).
- `review` requires initiative acceptance criteria; `changes_requested`
  routes back to `execute` (a fresh drain entry).
- Failed or canceled rounds do not advance the phase graph.
- Any reserved or agent-running round blocks all new phase starts.

## Operator Workflow

1. Create or select the initiative. New initiatives start in `item-level`.
2. Add initiative acceptance criteria before the review phase is needed.
3. Switch mode through `mode-switch`; do not edit `mode` through generic
   initiative update.
4. Start `investigate` and review the generated findings artifact.
5. Start `plan` and review the canonical plan-manager plan it binds
   (`plan_ref` on the initiative).
6. Start `execute` repeatedly to drain the plan (each round is one slice of
   the delegated phased-plan-drain). A `blocked` drain routes back through
   investigation; `complete` proceeds to review.
7. Start `review` and validate the whole initiative against acceptance criteria.
8. Use `complete-items` or `apply-backlog-sync` for any backlog status or scope
   reconciliation. Agents must not edit member item `spec.json` files directly.

## Preview the flow (simulation presets)

The operating-mode detail page's **Flow** tab can walk this graph deterministically
without running agents. Holistic-loop ships these presets as mode-owned example-run
data ([CODE: modes/holistic-loop/example-runs/]):

- **Clean composed pass** (`happy-path`) — `investigate → plan → execute →
  execute → review → reconcile`: the delegated drain continues once, completes,
  review accepts.
- **Blocked drain triggers replan** (`drain-blocked-replan`) — the drain hands
  off `progress=blocked`, so the parent guard routes back to `investigate`; the
  revised plan drains cleanly on the second pass.
- **Review requests changes** (`review-changes-requested`) — review returns
  `verdict=changes_requested`, so the guard routes **back to `execute`** to close the
  specific gaps; a second execute pass finishes and review then accepts before
  reconcile. This is the review reloop.
- **Review rejects, records the gap** (`review-not-accepted`) — review returns a
  plain non-accepting verdict that is *not* `changes_requested`, so it does not
  reloop; it advances to reconcile, which proposes follow-up items to track the gap.

Presets are deterministic previews, not real initiative rounds. In the Flow tab,
the data-source control swaps the same phase viewer between the **Contract**
template, a **Simulation** preset, and a **Live** round; the Instructions tab
renders the literal agent prompt for whichever source is selected.

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
