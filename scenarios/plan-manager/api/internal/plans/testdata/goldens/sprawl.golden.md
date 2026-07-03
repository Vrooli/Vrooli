# Sprawl measurement

> Status: **draft**

## Purpose

Freeze the renderer's current output for Sprawl measurement.

## Problem

Renderer behavior is unpinned; changes ship without review.

## Outcome

Every renderer change shows up as a reviewable golden diff.

## Approach & Decisions

Golden-file characterization before any renderer change.

## Boundaries

### Scope

In scope: markdown projection. Out of scope: persistence.

### Non-Goals

No redesign of the execution domain.

### Constraints

Keep the wizard usable by small local models.

### Prohibited Approaches

Do not make markdown the source of truth.

### Work Posture

- Posture: **greenfield**
- Source: default

**This is greenfield work.** Do not include compatibility shims, legacy wrappers, dead code, unused re-exports, `// removed` comments, or renamed `_unused` variables.

### Change Boundary

**Acceptance allow:**
- `scenarios/plan-manager/**`


## Assumptions & Risks

### Assumptions

The baseline is captured before changes.

### Risks / Hazards

Section regroup could break a hidden markdown consumer.

## Verification

### Regression Anchor

- Strategy: scenario_baseline
- Scenario baseline: `plan-manager` (name `plan-manager-render-golden`)
- Capture status: requested; usable only after `git-control-tower baseline snapshot status --wait --json` reports one or more captured surfaces
**Scenario baseline oracle:**
- `git-control-tower baseline diff --scenario plan-manager --name plan-manager-render-golden --wait`

### Validation Strategy

GOWORK=off go test ./internal/plans compares renders byte-exact.

**Final validation commands:**
- `vrooli scenario test plan-manager`

### Definition of Done

Goldens exist and reproduce current output verbatim.

## Execution Setup

### Load Skills

- scientific-debugging — `prompt-manager skill read scientific-debugging` _(required)_
  - Reason: State-machine bug; reproduce before fixing.

### References

- [CODE: scenarios/plan-manager/api/internal/plans/render.go]

### Execution Feedback

Log typed work products as they happen. Example:

```bash
plan-manager log decision-add <execution-id> --phase <phase-id> --title "..." --detail "..."
```

Other variants: `finding-add`, `bug-add`, `record-add`, `note-add`. When the handle is an execution id, omitting `--phase` uses that execution's current phase; `--phase` also accepts a phase id or 1-based ordinal. If the computed scope is wrong, run `plan-manager log reassign <entry-id> --phase <phase-id-or-ordinal>`.

On completion, write the learning-loop record — copy, fill the `<...>` placeholders, run:

```bash
swarm-manager records create --kind execute --scenario plan-manager \
  --trigger 'Sprawl measurement: <one-line goal>' \
  --approach '<what was built + key decisions>' \
  --evidence '<suites/baselines/live checks that prove it>' \
  --outcome shipped
```

## Phases

### Phase 1 — State model

- Status: **todo**
- Intent: Deliver the state model slice end to end.

**Affected Areas:**
- scenarios/web-console/ui/src/hooks/voice/

- Context: none needed — covered by the global skill and doc setup.

**Ordered Steps:**
1. Implement the slice
2. Add focused tests
3. Run the suite

**Expected Outputs:**
- Slice implemented with tests green

**Phase Validation:**

Run the focused UI vitest suite for the touched hooks; then the full UI suite.

- Acceptance: The state model slice is verifiably in place.

**References:**
- [CODE: scenarios/web-console/ui/src/hooks/voice/useVoiceCapture.ts]

### Phase 2 — Ownership registry

- Status: **todo**
- Intent: Deliver the ownership registry slice end to end.

**Affected Areas:**
- scenarios/web-console/ui/src/hooks/voice/

- Context: none needed — covered by the global skill and doc setup.

**Ordered Steps:**
1. Implement the slice
2. Add focused tests
3. Run the suite

**Expected Outputs:**
- Slice implemented with tests green

**Phase Validation:**

Run the focused UI vitest suite for the touched hooks; then the full UI suite.

- Acceptance: The ownership registry slice is verifiably in place.

**References:**
- [CODE: scenarios/web-console/ui/src/hooks/voice/useVoiceCapture.ts]

### Phase 3 — Wedge detection

- Status: **todo**
- Intent: Deliver the wedge detection slice end to end.

**Affected Areas:**
- scenarios/web-console/ui/src/hooks/voice/

- Context: none needed — covered by the global skill and doc setup.

**Ordered Steps:**
1. Implement the slice
2. Add focused tests
3. Run the suite

**Expected Outputs:**
- Slice implemented with tests green

**Phase Validation:**

Run the focused UI vitest suite for the touched hooks; then the full UI suite.

- Acceptance: The wedge detection slice is verifiably in place.

**References:**
- [CODE: scenarios/web-console/ui/src/hooks/voice/useVoiceCapture.ts]

### Phase 4 — Recovery affordance

- Status: **todo**
- Intent: Deliver the recovery affordance slice end to end.

**Affected Areas:**
- scenarios/web-console/ui/src/hooks/voice/

- Context: none needed — covered by the global skill and doc setup.

**Ordered Steps:**
1. Implement the slice
2. Add focused tests
3. Run the suite

**Expected Outputs:**
- Slice implemented with tests green

**Phase Validation:**

Run the focused UI vitest suite for the touched hooks; then the full UI suite.

- Acceptance: The recovery affordance slice is verifiably in place.

**References:**
- [CODE: scenarios/web-console/ui/src/hooks/voice/useVoiceCapture.ts]

### Phase 5 — Clock unification

- Status: **todo**
- Intent: Deliver the clock unification slice end to end.

**Affected Areas:**
- scenarios/web-console/ui/src/hooks/voice/

- Context: none needed — covered by the global skill and doc setup.

**Ordered Steps:**
1. Implement the slice
2. Add focused tests
3. Run the suite

**Expected Outputs:**
- Slice implemented with tests green

**Phase Validation:**

Run the focused UI vitest suite for the touched hooks; then the full UI suite.

- Acceptance: The clock unification slice is verifiably in place.

**References:**
- [CODE: scenarios/web-console/ui/src/hooks/voice/useVoiceCapture.ts]

### Phase 6 — Snappy start

- Status: **todo**
- Intent: Deliver the snappy start slice end to end.

**Affected Areas:**
- scenarios/web-console/ui/src/hooks/voice/

- Context: none needed — covered by the global skill and doc setup.

**Ordered Steps:**
1. Implement the slice
2. Add focused tests
3. Run the suite

**Expected Outputs:**
- Slice implemented with tests green

**Phase Validation:**

Run the focused UI vitest suite for the touched hooks; then the full UI suite.

- Acceptance: The snappy start slice is verifiably in place.

**References:**
- [CODE: scenarios/web-console/ui/src/hooks/voice/useVoiceCapture.ts]

