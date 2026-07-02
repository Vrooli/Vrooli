# Legacy required reading

> Status: **draft**

> Plan quality: **needs_review**
> - warning `legacy_import_requires_review` at `plan.import`: plan was imported from legacy markdown and should be validated/repaired before execution

## Purpose

Freeze the renderer's current output for Legacy required reading.

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
  --trigger 'Legacy required reading: <one-line goal>' \
  --approach '<what was built + key decisions>' \
  --evidence '<suites/baselines/live checks that prove it>' \
  --outcome shipped
```

## Phases

### Phase 1 — Wedge fix

- Status: **todo**
- Intent: Fix the stuck-recording wedge.

**Phase Context Setup:**

### Load Skills

- scientific-debugging _(required, migrated)_
  - Instruction: Load this internal skill before implementation.
  ```bash
  prompt-manager skill read scientific-debugging
  ```

### Read Docs

- docs/TESTING.md _(required, migrated)_
  - Instruction: Read this document before implementation.
  ```bash
  sed -n '1,120p' docs/TESTING.md
  ```

### Run Discovery Searches

- search-hub query 'microphone ownership' --type record _(required, migrated)_
  - Instruction: Run this discovery search before implementation.
  ```bash
  search-hub query 'microphone ownership' --type record
  ```

**Ordered Steps:**
1. Reproduce the wedge
2. Fix ownership

**Phase Validation:**

UI vitest suite for the voice hooks

- Acceptance: Wedge no longer reproducible.

**References:**
- [CODE: scenarios/web-console/ui/src/hooks/voice/useVoiceCapture.ts]

## Import Provenance

- Source: `docs/plans/legacy-required-reading.md`
- Original format: legacy_markdown
- Imported at: `2026-07-01T00:00:00Z`

