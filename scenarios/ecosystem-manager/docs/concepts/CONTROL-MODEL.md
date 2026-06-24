# Control Model — Ecosystem Manager

This document is the **canonical mental model** for how Ecosystem Manager
improves a scenario or resource. Read it before
[`ARCHITECTURE.md`](ARCHITECTURE.md) or any auto-steer code.

## The Core Idea

Ecosystem Manager is a **closed-loop controller**, not an open-loop schedule.
It runs the classic *sense → decide → act → measure* loop, driven by the
target's measured state rather than a hand-authored phase order:

```
  ┌───────────────────────────────────────────────────────────┐
  │                                                           │
  ▼                                                           │
┌───────────┐    ┌──────────────┐    ┌───────────┐    ┌────────────┐
│ DIAGNOSE  │───▶│   SELECT     │───▶│  EXECUTE  │───▶│  MEASURE   │
│ open      │    │ skill for    │    │ skill via │    │ re-audit;  │
│ test-genie│    │ the heaviest │    │ agent-mgr │    │ score the  │
│ findings  │    │ open dim     │    │           │    │ findings   │
└───────────┘    └──────────────┘    └───────────┘    └─────┬──────┘
      ▲                                                      │
      └──────────────────────────────────────────────────────┘
                         until objective met or budget spent
```

- **DIAGNOSE** — the controller's state is the set of open `test-genie`
  findings, each carrying a **dimension** (standards, tests, structure,
  security, …) and a **severity**. `pkg/findings` buckets them by dimension.
- **SELECT** — **greedy**: rank dimensions by profile-weighted severity
  (`weight × open-score`), then run the first eligible skill that targets the
  heaviest actionable dimension. `pkg/skillmap` resolves which skill targets
  which dimension; the mapping is declared on the skills themselves. Selection
  is deterministic and fully explainable — see `pkg/autosteer/selector.go`.
- **EXECUTE** — the selected steer skill runs out-of-band via `agent-manager`.
- **MEASURE** — re-audit (targeted to the run's dimensions, or a full preset on
  the profile's cadence), recompute the weighted score, and append a decision-
  trace entry.

The loop repeats until the **objective is met**, the **iteration budget** is
exhausted, or **diminishing returns** are detected.

## Profiles Are Objective Functions

A profile is not a script. It answers *"what does done mean, and what do I care
about most?"* and the controller derives the path:

- **`objective.dimension_weights`** — how much each finding dimension matters.
- **`objective.targets`** — `max_open_severity` (the highest severity tolerated
  at completion) and `operational_targets_pct` (minimum PRD operational-target
  completion, measured by `scenario-completeness-scoring`).
- **`budget`** — `max_iterations` (the bounded backstop), `diminishing_returns_floor`,
  and `reaudit_cadence` (full-audit every N iterations; targeted otherwise).
- **`allowed_skills` / `denied_skills`** — optional masks over the catalog-derived
  eligible set.
- **`baseline_promote`** — optional Baseline Modes engagement (see below).

"Green" means the objective is met: no finding above `max_open_severity` and (if
the scenario declares operational targets) `operational_targets_pct` satisfied.

## Termination

Termination is global, never per-phase. The controller stops when:

- the objective is met (best outcome), **or**
- the iteration budget (`max_iterations`) is reached, **or**
- mean weighted-score improvement over the trailing window falls below
  `diminishing_returns_floor` (grinding without progress).

See `pkg/autosteer/terminator.go`.

## Baseline-Safe Improvement (apply / revert)

An improvement run does not edit the live scenario in place. When a profile sets
`baseline_promote.enabled`, the controller fronts the run with a **baseline
engagement**: it starts a shadow via `git-control-tower baseline start` (which
takes a git-free restore point), routes the agent's edits and the controller's
measurements to the shadow, and at the end either:

- **promotes** the shadow → live when the run ended green, or
- **abandons** it (the shadow is torn down, live is untouched) otherwise — the
  "revert" is a shadow swap, not a git operation.

The shadow / baseline / promote / restore mechanics live **outside** this
scenario (`git-control-tower` + the `vrooli recovery` CLI); Ecosystem Manager
only orchestrates them. See the queue's baseline path and `BaselinePromote` in
`pkg/autosteer/types.go`.

## Anti-Gaming Promote-Safety Gate

A coding agent could make a target *look* green by faking it — weakening
`[REQ:]` tests, deleting PROBLEMS/PROGRESS ledgers, or adding lint/suppression
directives. The **gameguard** classifier (`pkg/autosteer/gameguard`) inspects
each run's code-level diff and stamps a `gaming_cause` on the decision trace when
it sees gaming-shaped work. A run with any `gamed:` iteration is **blocked from
promotion** (`ExecutionOrchestrator.RunGamed`): a faked green is abandoned, not
promoted. This is the only role gameguard plays — it does not alter selection.

## Transparency

Every selection appends a decision-trace entry (durable, survives finalization):
iteration, chosen skill, heaviest dimension, score before/after, realized delta,
`gaming_cause`, and the terminal `halt_reason`. Surfaced via the
`/auto-steer/execution/{taskId}/trace` API, the `steer trace` CLI command, and
the UI decision-trace panel.

## Design History

Earlier iterations layered an effectiveness-weighted contextual bandit, a
development-toolchain-validator (DTV) eligibility gate and trust priors, and a
maturity-ladder rung governor on top of greedy selection. These were removed in
favor of the simple greedy controller above: with `test-genie` as the universal
diagnose engine and steer skills declaring their target dimensions, selection
collapses to "fix the heaviest open dimension," and the extra machinery was
unproven complexity. The system is intentionally simple, maintainable, and
explainable; richer selection can be reintroduced behind evidence if a future
need is demonstrated.

## Cross-References

- [`ARCHITECTURE.md`](ARCHITECTURE.md) — system shape and surfaces
- [`DOMAINS.md`](DOMAINS.md) — auto-steer, queue, steering ownership
- [`FLOWS.md`](FLOWS.md) — the runtime control-loop state machine
- [`GLOSSARY.md`](GLOSSARY.md) — controller vocabulary
- [`../internal/PROBLEMS.md`](../internal/PROBLEMS.md) — known issues
