# Heartbeat-list lifecycle-state distinction

## Problem

`prompt-manager team heartbeat-list <team>` human output collapses the lifecycle of a heartbeat into one ambiguous "last: ..." string. Today (`scenarios/prompt-manager/cli/teams/teams.go:1535-1539`):

```
  <agent>: <schedule> [enabled|disabled] - last: never
  <agent>: <schedule> [enabled|disabled] - last: <status> at <ts>
```

This conflates three lifecycle states the operator and prep-agent need to tell apart:

1. **enabled / scheduled-never-fired** — newly enabled, first scheduled fire still in the future. Healthy. Shows as `last: never`, indistinguishable from a broken brand-new config.
2. **enabled / dormant** — has past fires but no recent activity (queue is silent — could be genuinely quiet, or agent stopped surfacing decisions).
3. **enabled / broken** — last fire failed/errored/timed out.
4. **disabled** — turned off entirely.

The JSON response already carries `enabled`, `lastExecution` (with `status` and `startedAt`), `nextExecution`, and `nextExecutions`. The bug is purely in the human formatter dropping `nextExecution` and not synthesizing a lifecycle label.

This work is part of the `prompt-manager-decision-workflow-polish` initiative. Two of its members are already completed (deferral primitive, partial-accept modifications); the remaining `decision-show-options-default-output`, `decision-accept-no-options-ergonomics`, and `decision-accept-initiative-proposal-auto-create` items touch decision UX and do not overlap with heartbeat output. No coordination needed beyond the three-surface parity rule.

## Required Reading

- `prompt-manager skill read scientific-debugging` — hypothesis-driven fix and regression test design
- `prompt-manager skill read cli-steer` — CLI human-output conventions
- `prompt-manager skill read api-steer` — API response surface and three-surface parity
- `prompt-manager skill read test` — test placement and snapshot patterns

## Scope

### In scope
- New human-output formatting in `cmdHeartbeatList` that surfaces an explicit lifecycle label and `next:` timestamp.
- A small pure-function classifier that maps a `HeartbeatConfig` to a lifecycle state, callable from CLI (and reusable by UI).
- Unit tests for the classifier covering all states and edge cases.
- Snapshot/golden tests for the rendered human output.
- Mirror the labelled state in the UI heartbeat list view (three-surface parity rule for the initiative).

### Out of scope
- Changing the JSON response shape (operator confirmed JSON already carries the underlying signals).
- Changing scheduling semantics, dormancy thresholds in the scheduler, or alerting.
- New API fields (lifecycle state stays a derived/presentation concern; consumers can compute or read the label).

### Three-surface parity check
- **API**: no change required — fields already present.
- **CLI**: primary change.
- **UI**: mirror the label in `ui/src/...` heartbeat view (TBD: confirm component path during implementation).

## Lifecycle state model (proposed, pending decisions)

A pure function `classify(c HeartbeatConfig, now time.Time) LifecycleState`:

| State | Condition | Render |
|-------|-----------|--------|
| `disabled` | `!Enabled` | `[disabled]` (omit last/next) |
| `scheduled` | `Enabled && LastExecution == nil && NextExecution != ""` | `[enabled/scheduled] next: <ts>` |
| `broken` | `Enabled && LastExecution != nil && LastExecution.Status in {failed,error,timeout}` | `[enabled/broken] last: <status> at <ts> next: <ts>` |
| `dormant` | `Enabled && LastExecution != nil && now - LastExecution.StartedAt > dormancy_threshold` | `[enabled/dormant] last: <status> at <ts> next: <ts>` |
| `active` | `Enabled && LastExecution != nil && recent success` | `[enabled/active] last: completed at <ts> next: <ts>` |
| `unknown` | Enabled but no schedule / no NextExecution computable | `[enabled/unknown] last: <ts>` (fallback) |

Open questions captured as decisions below.

## Approach (phased)

### Phase 1 — Classifier + unit tests
- Add `LifecycleState` type and `Classify(cfg HeartbeatConfig, now time.Time) LifecycleState` in `cli/teams/` (or a small shared package if UI will import it — see decision d4).
- Table-driven unit tests covering every state and edge cases (nil LastExecution, empty NextExecution, every status string, dormancy boundary).

### Phase 2 — CLI rendering
- Replace lines `cli/teams/teams.go:1535-1539` with a renderer that consumes `Classify` output.
- Add a snapshot/golden test asserting each state's exact rendered line.

### Phase 3 — UI parity
- Locate the UI heartbeat list component (`ui/src/...`) and apply the same labelling using the same classifier semantics. If a Go classifier can't be imported, mirror the rules in TS with shared test cases.

### Phase 4 — Docs
- Update `docs/concepts/HEARTBEATS.md` and `docs/reference/heartbeat-api.md` (if they describe CLI output) with the new labels and the dormancy rule.

## Test plan

- **Classifier unit tests** (Go, `cli/teams/teams_test.go` or new `heartbeat_lifecycle_test.go`):
  - disabled → `disabled`
  - enabled + nil LastExecution + future NextExecution → `scheduled`
  - enabled + LastExecution.Status = "failed" → `broken`
  - enabled + LastExecution.Status = "error"/"timeout" → `broken`
  - enabled + recent successful LastExecution → `active`
  - enabled + LastExecution older than threshold → `dormant`
  - enabled + nil LastExecution + empty NextExecution → `unknown`
- **CLI render snapshot tests**: feed the renderer with one fixture per state, assert exact stdout line.
- **UI parity tests**: same classification matrix in the UI test suite.

## Risks

| Risk | Mitigation |
|------|-----------|
| Dormancy threshold misclassifies legitimately-slow schedules (e.g., weekly) as dormant | Make threshold a function of the schedule cadence (e.g., 2× the next-fire interval), not a fixed wall-clock value. Decision d2. |
| Status string set for `LastExecution.Status` not fully enumerated → "broken" detection is brittle | Enumerate by reading the executor source; default unknown statuses to `active` with `last:` shown, not `broken`. |
| UI surface drifts from CLI labels if implemented separately | Share fixtures/test cases across surfaces; document the canonical state machine in `docs/concepts/HEARTBEATS.md`. |
| Operator's prep-agent parses current human output | Search for parsers (likely none — JSON path is canonical) before changing format. |

## Acceptance

- `acceptance_allow`: `scenarios/prompt-manager/**` (already set; covers CLI, API docs, UI, tests).
- Manual verification: `prompt-manager team heartbeat-list <team>` produces a labelled line for each of: scheduled-never-fired, dormant, broken, active, disabled.

## Decision log

(populated by accepted workshop decisions)
