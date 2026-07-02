# Empty optional sections

> Status: **draft**

## Purpose

Freeze the renderer's current output for Empty optional sections.

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

### Operator Notes

- NO_SKILL_CONTEXT: fixture plan; skill setup is exercised by dedicated fixtures. _(required)_

### References

- [CODE: scenarios/plan-manager/api/internal/plans/render.go]

### Execution Feedback

Log typed work products as they happen via `plan-manager log {decision,finding,bug,record,note}-add <plan-or-execution> --phase <n> ...` (full command list: plan-manager CLI reference).

On completion, write the learning-loop record — copy, fill the `<...>` placeholders, run:

```bash
swarm-manager records create --kind execute --scenario plan-manager \
  --trigger 'Empty optional sections: <one-line goal>' \
  --approach '<what was built + key decisions>' \
  --evidence '<suites/baselines/live checks that prove it>' \
  --outcome shipped
```

## Phases

### Phase 1 — Only phase

- Status: **todo**
- Intent: Prove empty optional sections render nothing.

- Context: none needed — minimal fixture; no setup needed.

**Ordered Steps:**
1. Render
2. Diff

**Phase Validation:**

go test ./internal/plans

- Acceptance: No optional headings appear.

**References:**
- [CODE: scenarios/plan-manager/api/internal/plans/render.go]

