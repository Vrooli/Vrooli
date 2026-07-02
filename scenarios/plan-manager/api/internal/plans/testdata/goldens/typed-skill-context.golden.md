# Typed skill context

> Status: **draft**

## Purpose

Freeze the renderer's current output for Typed skill context.

## Problem

Renderer behavior is unpinned; changes ship without review.

## Outcome

Every renderer change shows up as a reviewable golden diff.

## Approach & Decisions

Golden-file characterization before any renderer change.

## Boundaries

### Scope

In scope: markdown projection. Out of scope: persistence.

### Work Posture

- Posture: **greenfield**
- Source: default

**This is greenfield work.** Do not include compatibility shims, legacy wrappers, dead code, unused re-exports, `// removed` comments, or renamed `_unused` variables.

### Change Boundary

**Acceptance allow:**
- `scenarios/plan-manager/**`


## Verification

### Regression Anchor

- Strategy: scenario_baseline
- Scenario baseline: `plan-manager` (name `plan-manager-render-golden`)
- Capture status: requested; usable only after `git-control-tower baseline snapshot status --wait --json` reports one or more captured surfaces
**Scenario baseline oracle:**
- `git-control-tower baseline diff --scenario plan-manager --name plan-manager-render-golden --wait`

### Validation Strategy

GOWORK=off go test ./internal/plans compares renders byte-exact.

### Definition of Done

Goldens exist and reproduce current output verbatim.

## Execution Setup

### Load Skills

- scientific-debugging — `prompt-manager skill read scientific-debugging` _(required)_
  - Reason: The defect is a state-machine bug; reproduce before fixing.
- test — `prompt-manager skill read test` _(required)_
  - Reason: Golden-file discipline for the renderer work.

### Read Docs

- docs/TESTING.md — `sed -n '1,220p' docs/TESTING.md` _(required)_
  - Reason: Server-owned test runs; never poll.

### References

- [CODE: scenarios/plan-manager/api/internal/plans/render.go]

### Execution Feedback

Log typed work products as they happen via `plan-manager log {decision,finding,bug,record,note}-add <plan-or-execution> --phase <n> ...` (full command list: plan-manager CLI reference).

On completion, write the learning-loop record — copy, fill the `<...>` placeholders, run:

```bash
swarm-manager records create --kind execute --scenario plan-manager \
  --trigger 'Typed skill context: <one-line goal>' \
  --approach '<what was built + key decisions>' \
  --evidence '<suites/baselines/live checks that prove it>' \
  --outcome shipped
```

## Phases

### Phase 1 — Renderer

- Status: **todo**
- Intent: Project setup context deterministically.

**Phase Context Setup:**

### Read Docs

- docs/concepts/PLAN-MODEL.md — `sed -n '1,220p' docs/concepts/PLAN-MODEL.md` _(required)_
  - Reason: The renderer projects the model this doc defines.

**Ordered Steps:**
1. Render fixtures
2. Compare against goldens

**Phase Validation:**

GOWORK=off go test ./internal/plans

- Acceptance: Goldens match byte-exact.

**References:**
- [CODE: scenarios/plan-manager/api/internal/plans/render.go]

