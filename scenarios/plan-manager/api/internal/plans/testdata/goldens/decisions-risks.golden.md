# Decisions and risks

> Status: **draft**

## Purpose

Freeze the renderer's current output for Decisions and risks.

## Problem

Renderer behavior is unpinned; changes ship without review.

## Outcome

Every renderer change shows up as a reviewable golden diff.

## Approach & Decisions

Golden-file characterization before any renderer change.

### Decisions

_Pinned at plan time; do not relitigate during execution._

- **D1 — Cluster names and order:** Nine clusters, wizard asks in render order.
- **D2 — Field identity preserved:** Regrouping is catalog-order + render-grouping only.
- **D3 — Dependency posture:** prompt-manager and search-hub become required:false dependencies.

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


## Assumptions & Risks

| Assumption | If wrong → mitigation |
|---|---|
| prompt-manager --json output shape is stable | pin parsing behind the probe seam with contract fixtures |
| search-hub may be systemically degraded | per-probe timeout and independent degradation |

### Assumptions

The regression baseline is captured before any code change.

### Risks / Hazards

Render regrouping could break a hidden markdown consumer.

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
  --trigger 'Decisions and risks: <one-line goal>' \
  --approach '<what was built + key decisions>' \
  --evidence '<suites/baselines/live checks that prove it>' \
  --outcome shipped
```

## Phases

### Phase 1 — Only phase

- Status: **todo**
- Intent: Carry the decision fixtures.

- Context: none needed — fixture phase.

**Ordered Steps:**
1. Render
2. Diff

**Phase Validation:**

go test ./internal/plans

- Acceptance: D-list and table render.

**References:**
- [CODE: scenarios/plan-manager/api/internal/plans/render.go]

