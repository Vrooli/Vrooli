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

- scientific-debugging _(required)_
  - Reason: The defect is a state-machine bug; reproduce before fixing.
  - Instruction: Load this internal skill before implementation.
  ```bash
  prompt-manager skill read scientific-debugging
  ```
- test _(required)_
  - Reason: Golden-file discipline for the renderer work.
  - Instruction: Load this internal skill before implementation.
  ```bash
  prompt-manager skill read test
  ```

### Read Docs

- docs/TESTING.md _(required)_
  - Reason: Server-owned test runs; never poll.
  - Instruction: Read this document before implementation.
  ```bash
  sed -n '1,220p' docs/TESTING.md
  ```

### References

- [CODE: scenarios/plan-manager/api/internal/plans/render.go]

### Execution Feedback

Log typed work products as they happen via `plan-manager log {decision,finding,bug,record,note}-add <plan-or-execution> --phase <n> ...` (full command list: plan-manager CLI reference).

## Phases

### Phase 1 — Renderer

- Status: **todo**
- Intent: Project setup context deterministically.

**Phase Context Setup:**

### Read Docs

- docs/concepts/PLAN-MODEL.md _(required)_
  - Reason: The renderer projects the model this doc defines.
  - Instruction: Read this document before implementation.
  ```bash
  sed -n '1,220p' docs/concepts/PLAN-MODEL.md
  ```

**Ordered Steps:**
1. Render fixtures
2. Compare against goldens

**Phase Validation:**

GOWORK=off go test ./internal/plans

- Acceptance: Goldens match byte-exact.

**References:**
- [CODE: scenarios/plan-manager/api/internal/plans/render.go]

