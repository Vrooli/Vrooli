# Phased Plan Drain Mode (generic plan drain)

Phased-plan-drain is the **generic, plan-first execution loop**: point it at a
plan-manager plan and drain it slice-by-slice. It is not an initiative mode —
its declared target is `plan-manager-plan` (see the target concept in
[EXECUTION-MODES](../concepts/EXECUTION-MODES.md)). No initiative, member
items, or backlog ceremony are involved; work that warrants initiative
tracking layers it on by composing this mode (`executed_by` — holistic-loop
does exactly this for its execute phase, a shipped example of
the v2 rebuild).

## Shape

One phase, one loop. Mode data: [CODE: modes/phased-plan-drain/mode.json].

```text
execute --(progress=continue)--> execute
        --(progress=complete)--> stop  (plan fully drained)
        --(progress=blocked) --> stop  (operator attention)
```

- **`execute`** — a fresh agent reads the target plan (`PLAN_ID` +
  `PLAN_CONTEXT_JSON`), executes the next drainable slice (the earliest
  contiguous unit it can complete comprehensively — the elastic-slice
  contract), and emits a `handoff` (summary, blockers, next_step,
  changed_files, tests, **frontier**).
- **Classification-on-transition** — there is no classifier phase. The single
  classified edge derives `progress` ∈ {continue, complete, blocked} from the
  handoff: emitted directly → short-circuit; inline on the handoff → L1
  deterministic extraction; otherwise the schema-steered L2 classifier;
  underivable → honest abstain and the round parks in `needs_attention`.
- **Stops are guarded** — the mode declares no terminal phase; `complete` and
  `blocked` are guarded stops (a matched route with no target phase).

## Reads

Composed as generic-base ∪ plan-manager-plan adapter — no initiative
variables exist for this target:

- Base: `ROUND_NUMBER`, `OPERATOR_NOTE`, `PRIOR_ROUNDS_JSON` (the accumulated
  handoffs — continuity between fresh agents), `ELASTIC_SLICE_SNIPPET`.
- Plan adapter: `PLAN_ID`, `PLAN_CONTEXT_JSON` (plan-manager execution
  context resolved via Resume).

Prompt skill: `swarm-manager-phased-plan-execute-next` (prompt-manager).

## Preview the flow (simulation presets)

Example-runs under [CODE: modes/phased-plan-drain/example-runs/] walk the real
guards at load time (guard-replay) and back the UI Flow tab's simulation
presets:

- **`happy-path`** — two slices: the first handoff classifies `continue` and
  loops execute from the declared frontier; the second classifies `complete`
  and the drain stops. The elastic-slice contract in action.
- **`complete-first-slice`** — the whole plan drains in one round.
- **`blocked`** — the first slice hits a real blocker; the handoff records it
  and the loop stops for operator attention instead of skipping ahead.

```bash
swarm-manager operating-mode get --mode phased-plan-drain --phase execute --show-prompt
swarm-manager operating-mode get --mode phased-plan-drain --phase execute --show-prompt --preset blocked
```

## Choosing it

Use the drain when a stable plan already exists and the work is "execute this
plan". If the plan is exploratory or the unit of validation is the system as
a whole, use `holistic-loop`; if items are independent and parallelism wins,
use `item-level`. Initiatives cannot switch to this mode — it targets a plan,
not an initiative — and initiative-keyed phase surfaces reject it with a
typed error.

## Control Boundaries

- Mode data (SSOT): [CODE: modes/phased-plan-drain/mode.json]
- Loader/validation and guard expansion: [CODE: api/internal/operatingmode/loader.go]
- Classification-on-transition: [CODE: api/internal/operatingmode/transition_classification.go]
- Target adapters and ownership keys: [CODE: api/internal/operatingmode/target.go]
- Sequential handoff validation: [CODE: api/internal/operatingmode/state.go]
- Plan-manager context injection: [CODE: api/internal/operatingmode/plan_ref.go]
- Round persistence: [CODE: api/internal/operatingmode/rounds.go]

## Starting it on a bare plan

The plan-first entry point is `OperatingModeService.StartTargetPhase`, exposed
as a CLI command:

```bash
swarm-manager operating-mode start --mode phased-plan-drain --target <plan-id|slug> [--note "..."]
```

The round spawns with the plan's reads (`PLAN_ID`, `PLAN_CONTEXT_JSON`),
stores under `<dataRoot>/mode-targets/plan-manager-plan/<execution-id>/`, and
holds the plan ownership lock (`plan--<execution-id>`). Follow up with the
ordinary round actions, addressing the round by its resolved scope id plus an
explicit mode:

```bash
swarm-manager initiatives mode-refresh --name <execution-id> --mode phased-plan-drain --round 1
swarm-manager initiatives mode-cancel  --name <execution-id> --mode phased-plan-drain --round 1
```
