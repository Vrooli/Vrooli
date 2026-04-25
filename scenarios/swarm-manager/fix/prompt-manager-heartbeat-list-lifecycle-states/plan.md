# Heartbeat-list lifecycle-state distinction

> **Greenfield declaration:** This is greenfield code in a pre-1.0 product. No backwards-compatibility shims, no deprecation windows, no legacy parsers to preserve. The CLI human output format is not a stable contract; the API JSON shape is freely additive.

## Purpose

Operators and the prep-agent need to tell apart three distinct lifecycle states of a heartbeat from a single `prompt-manager team heartbeat-list <team>` command: (1) enabled but scheduled-not-yet-fired (healthy first-day state), (2) enabled but dormant or broken (agent stopped surfacing decisions), and (3) disabled. The current output collapses these into an ambiguous `last: ...` string, forcing operators to dig into JSON or guess. This work eliminates that guesswork by computing an explicit lifecycle label on the server, exposing it across all three surfaces, and rendering it prominently in CLI/UI.

## Problem Statement

`prompt-manager team heartbeat-list <team>` human output today (`scenarios/prompt-manager/cli/teams/teams.go:1535-1539`):

```
  <agent>: <schedule> [enabled|disabled] - last: never
  <agent>: <schedule> [enabled|disabled] - last: <status> at <ts>
```

Failure modes:

1. **enabled / scheduled-never-fired** — newly enabled, first scheduled fire still in the future. Healthy. Renders as `last: never`, indistinguishable from a broken brand-new config.
2. **enabled / dormant** — `NextExecution` is in the past; scheduler should have fired but didn't. Operator sees the same `last: <status> at <old-ts>` and cannot tell whether the queue is genuinely quiet or the agent is broken.
3. **enabled / broken** — last fire failed/errored/timed out. Status string is in the JSON, but the human output buries it.
4. **disabled** — turned off entirely. Already labeled, but the trailing `last:` field is noise in this state.

The JSON response (`HeartbeatConfigResponse` at `scenarios/prompt-manager/api/heartbeat/models.go:9-22`) already carries `Enabled`, `LastExecution.Status/StartedAt`, `NextExecution`, and `NextExecutions`. The fix is (a) compute a derived lifecycle label, (b) expose it on the API, (c) render it across CLI and UI.

## Current Technical Context

- **API model** (`scenarios/prompt-manager/api/heartbeat/models.go:9-22`): `HeartbeatConfigResponse` with `Enabled bool`, `LastExecution *Execution` (carrying `Status string` and `StartedAt time.Time`), `NextExecution string` (RFC3339), `NextExecutions []string`.
- **CLI rendering** (`scenarios/prompt-manager/cli/teams/teams.go:1535-1539`): hard-coded `Sprintf` building the line above; drops `NextExecution` entirely.
- **UI heartbeat list view**: location TBD during Phase 3 — likely under `ui/src/...`. Confirm path during implementation; if no list view exists yet, scope is just rendering the new field where heartbeats are displayed.
- **Initiative neighborhood**: this item is part of `prompt-manager-decision-workflow-polish`. Other open members touch decision UX, not heartbeats, so no scope overlap. The initiative's three-surface parity rule (API + CLI + UI ship together) applies.
- **Status string set** for `LastExecution.Status` must be enumerated by reading the executor source before implementing; "broken" detection depends on knowing the failure-status set exactly.

## Target End State

After this fix:

- `HeartbeatConfigResponse` carries a new field `LifecycleState string` (one of `disabled`, `scheduled`, `active`, `dormant`, `broken`, `unknown`), computed on the API side from existing fields. No new inputs.
- `prompt-manager team heartbeat-list <team>` human output renders one labelled line per heartbeat:
  - `<agent>: <schedule> [disabled]`
  - `<agent>: <schedule> [enabled/scheduled] next: <ts>`
  - `<agent>: <schedule> [enabled/active] last: completed at <ts> next: <ts>`
  - `<agent>: <schedule> [enabled/dormant] last: <status> at <ts> next: <ts>`
  - `<agent>: <schedule> [enabled/broken] last: <status> at <ts> next: <ts>`
  - `<agent>: <schedule> [enabled/unknown] last: <ts>` (fallback)
- The UI heartbeat view displays the same compound label.
- All three surfaces agree on the label for any given config because the API computes it once.
- Snapshot tests lock the CLI line for each state; unit tests cover every classifier branch; UI test asserts the label is rendered from the server field.

## Implementation Strategy

Phased to keep each PR small and reviewable.

### Phase 1 — API: classifier + `LifecycleState` field

- Add a pure function `classifyLifecycle(c HeartbeatConfig, now time.Time) string` in the heartbeat API package (e.g., `scenarios/prompt-manager/api/heartbeat/lifecycle.go`).
- Rules (from decisions d1–d3):
  - `!Enabled` → `disabled`
  - `Enabled && LastExecution == nil && NextExecution != ""` → `scheduled`
  - `Enabled && LastExecution != nil && Status ∈ failureSet` → `broken` (failure-status dominates regardless of age)
  - `Enabled && LastExecution != nil && NextExecution != "" && now > NextExecution` → `dormant` (scheduler should have fired; deterministic schedule-aware rule, no tunable threshold)
  - `Enabled && LastExecution != nil && Status ∉ failureSet && now ≤ NextExecution` → `active`
  - else → `unknown`
- `failureSet`: enumerate by reading the executor; default unknown statuses to NOT match `broken` (fail safe toward `active`).
- Populate `LifecycleState` on `HeartbeatConfigResponse` at the response-construction site.
- Table-driven unit tests covering every state, every status string in `failureSet`, dormancy boundary (now == NextExecution and now == NextExecution + 1ns), nil LastExecution, empty NextExecution.

### Phase 2 — CLI rendering

- Replace `cli/teams/teams.go:1535-1539` with a renderer that switches on `cfg.LifecycleState` and emits the compound label + appropriate timestamps per the table in Target End State.
- Snapshot/golden test (one fixture per state) asserting exact stdout.
- Confirm no callers parse current human output (search for parsers; JSON path is canonical).

### Phase 3 — UI parity

- Locate the UI heartbeat view; render the `lifecycleState` field as the same compound label using the same vocabulary. Pure presentation — no client-side classifier.
- UI test verifies the label is rendered for each state value.

### Phase 4 — Docs

- Update `docs/concepts/HEARTBEATS.md` with the lifecycle state machine (the six states and their transition rules).
- Update `docs/reference/heartbeat-api.md` to document the new `lifecycleState` response field and its possible values.

## Contract Decisions

These are the locked-in decisions from workshop round 001:

| ID | Topic | Decision |
|----|-------|----------|
| d1 | Label vocabulary | **Compound** `[enabled/scheduled]`, `[enabled/active]`, `[enabled/dormant]`, `[enabled/broken]`, `[enabled/unknown]`, `[disabled]`. Reading any line communicates both axes (toggle + lifecycle) without inference. |
| d2 | Dormancy rule | **`now > NextExecution`** — schedule-aware by construction, uses an already-computed field, no tunable threshold, naturally correct for weekly/monthly cadences. |
| d3 | Broken vs dormant precedence | **Failure status dominates.** Any `LastExecution.Status ∈ failureSet` → `broken` regardless of age. A `broken` heartbeat stays broken until a successful execution replaces the last one. |
| d4 | Classifier placement | **API-side, exposed as `lifecycleState` on `HeartbeatConfigResponse`.** Single source of truth; CLI and UI render a server-supplied label. Eliminates cross-surface drift entirely. Accepts the API contract addition as worth the cost. |
| d5 | UI scope | **Mirror in the UI as part of this fix.** Honors the initiative's three-surface parity rule; ships a coherent operator experience in one cut. |

Implications of d4: `HeartbeatConfigResponse` gains one additive field (`lifecycleState string`). This is a pure addition in a pre-1.0 contract — no migration concerns.

## Testing Plan

- **API classifier unit tests** (Go, alongside `lifecycle.go`):
  - disabled (`Enabled=false`, all permutations of other fields ignored) → `disabled`
  - enabled + nil LastExecution + future NextExecution → `scheduled`
  - enabled + nil LastExecution + empty NextExecution → `unknown`
  - enabled + LastExecution.Status = each entry of `failureSet` → `broken` (parameterised)
  - enabled + LastExecution.Status = unknown string → `active` (fail-safe)
  - enabled + recent successful LastExecution + future NextExecution → `active`
  - enabled + LastExecution + NextExecution in past → `dormant`
  - enabled + LastExecution + empty NextExecution → `unknown`
  - boundary: `now == NextExecution` → `active` (use `>`, not `>=`)
- **API response integration test**: build a `HeartbeatConfigResponse` end-to-end and assert `LifecycleState` matches the classifier for representative configs.
- **CLI snapshot tests** (`scenarios/prompt-manager/cli/teams/teams_test.go` or sibling): one fixture per `lifecycleState` value, assert exact stdout line.
- **UI test**: render the heartbeat view with each `lifecycleState` value, assert the corresponding compound label appears.
- **Manual verification**: against a running `vrooli scenario` instance, exercise each state by toggling enabled, waiting past NextExecution, forcing a failed execution.

## Rollout/Validation Checklist

- [ ] `swarm-manager skill read scientific-debugging`, `cli-steer`, `api-steer`, `test` skills consulted before coding.
- [ ] `failureSet` enumerated against the executor source and listed in a code comment alongside the constant.
- [ ] All API classifier unit tests pass (`go test ./scenarios/prompt-manager/api/heartbeat/...`).
- [ ] `lifecycleState` field appears in the JSON response for `prompt-manager team heartbeat-list <team>` (verify with `--output json`).
- [ ] CLI snapshot tests pass for all six states.
- [ ] `prompt-manager team heartbeat-list <team>` human output verified manually for at least: `disabled`, `scheduled` (newly enabled team), and `active` (team with successful recent fire).
- [ ] UI heartbeat view renders the compound label for every state value.
- [ ] `docs/concepts/HEARTBEATS.md` and `docs/reference/heartbeat-api.md` updated.
- [ ] **Final cleanup:** `vrooli scenario restart prompt-manager` to ensure no stale binaries/state, then re-run heartbeat-list once more to confirm clean output.

## Non-goals / Prohibited Patterns

- **Do not** change scheduling semantics, dormancy thresholds in the scheduler itself, or alerting behavior. Lifecycle is a derived presentation concern.
- **Do not** add tunable wall-clock dormancy thresholds; the rule is `now > NextExecution`, period.
- **Do not** re-implement the classifier client-side in the UI. UI consumes `lifecycleState` from the server.
- **Do not** change the existing JSON field names or types; `lifecycleState` is purely additive.
- **Do not** preserve the legacy `last: never` / `last: <status> at <ts>` line shape for "compatibility" — this is greenfield, replace cleanly.
- **Do not** add backwards-compat shims, feature flags, or dual-rendering paths.

## Definition of Done

- `HeartbeatConfigResponse.LifecycleState` is populated server-side via `classifyLifecycle` for every heartbeat in every API response that returns a config.
- Six lifecycle states (`disabled`, `scheduled`, `active`, `dormant`, `broken`, `unknown`) each have a unit test and a CLI snapshot test, all passing.
- `prompt-manager team heartbeat-list <team>` human output renders the compound label and `next:` timestamp per the Target End State table for every state.
- UI heartbeat view shows the same compound label, sourced from the server field.
- `docs/concepts/HEARTBEATS.md` documents the state machine; `docs/reference/heartbeat-api.md` documents the new field.
- `vrooli scenario restart prompt-manager` run as the final step; subsequent heartbeat-list invocation produces the new output cleanly.
- No new tunable thresholds, feature flags, or compat shims introduced.

## Required Reading

- `prompt-manager skill read scientific-debugging` — hypothesis-driven fix and regression test design
- `prompt-manager skill read cli-steer` — CLI human-output conventions
- `prompt-manager skill read api-steer` — API response surface and three-surface parity
- `prompt-manager skill read test` — test placement and snapshot patterns

## Risks + Mitigations

| Risk | Mitigation |
|------|-----------|
| `failureSet` is incomplete → some real failures classify as `active` | Enumerate by reading the executor source; add a code comment listing every status string considered; failing to match is the safer default than over-broadcasting `broken`. |
| Boundary condition at `now == NextExecution` flickers between `active` and `dormant` | Use strict `>` (not `>=`); add explicit boundary tests. |
| API field added but UI ships without consuming it → drift | Phase 3 is part of the same item; DoD requires UI parity before close. |
| Operator's prep-agent parses current human output | Search for parsers before changing format; JSON path is canonical so likelihood is low. |
| `unknown` state becomes a dumping ground that hides real bugs | Treat any `unknown` count > 0 in production as a follow-up signal; document in `HEARTBEATS.md` that `unknown` indicates a config we haven't reasoned about. |

## Decision log

- **d1** (locked, round 001): Compound label vocabulary `[enabled/<lifecycle>]` and `[disabled]`.
- **d2** (locked, round 001): Dormancy rule = `now > NextExecution`.
- **d3** (locked, round 001): Failure status dominates regardless of age.
- **d4** (locked, round 001): Classifier lives on the API; expose `lifecycleState` field on `HeartbeatConfigResponse`.
- **d5** (locked, round 001): UI parity ships within this same fix.
