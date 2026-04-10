# Fix: Initiative Details Progress Bar Showing Incorrect Pending Count

## Required Reading

- `prompt-manager skill read scientific-debugging` — hypothesis-driven root cause analysis
- `prompt-manager skill read swarm-manager-backlog-tools` — backlog file schemas and CLI

## Problem Statement

The initiative details page progress bar reports 2 pending items when all member items are actually completed. The user suspects archived items may be miscounted as pending.

### Root Cause Analysis

The `ComputeRollup` function in `scenarios/swarm-manager/api/internal/initiatives/service.go:246-281` iterates initiative items and classifies them by status. Two code paths produce a `Pending++` increment for items that the user considers "done":

1. **Load failures (lines 262-264):** If `backlogLoader.LoadItem()` returns an error (item deleted from disk, renamed, or moved), the item is counted as pending. Archived-then-deleted items would hit this path.
2. **Unrecognized statuses (line 273-274):** Items with `StatusBacklog` or `StatusReady` fall into `default: rollup.Pending++`. These are legitimately pending, but if an item was archived before transitioning to a terminal status, it would count as pending despite being archived.

**Primary hypothesis:** The initiative references 2 items that either (a) no longer exist on disk, or (b) are archived with a non-terminal status. In both cases `ComputeRollup` counts them as pending.

**The `Archived` counter (lines 276-278) is tracked separately and does NOT affect the progress bar** — the frontend's `rollupTotal()` only sums `completed + inProgress + failed + pending`. So the double-counting concern from initial triage is a red herring for this specific bug.

## Scope

### In Scope
- Fix `ComputeRollup` to correctly handle archived items and load failures
- Update progress bar frontend if needed to reflect archived items
- Add/update unit tests for the fixed behavior

### Out of Scope
- Changing the archive workflow itself
- Modifying the graph projection rollup logic (separate component)
- Redesigning the progress bar UI beyond this fix

### Acceptance Criteria
- Progress bar shows 0 pending when all items are completed (regardless of archive status)
- Archived items are either excluded from progress or counted by their actual status
- Load failures for initiative member items don't silently inflate the pending count
- Existing `TestService_ComputeRollup` updated + new test cases for archived items

## Approach

<!-- TBD — pending decision d1 (how to handle archived items) and d2 (how to handle load failures) -->

## Key Files

| File | Role |
|------|------|
| `scenarios/swarm-manager/api/internal/initiatives/service.go` | `ComputeRollup` function — primary fix target |
| `scenarios/swarm-manager/api/internal/initiatives/model.go` | `RollupStatus` struct definition |
| `scenarios/swarm-manager/api/internal/initiatives/service_test.go` | Unit tests for rollup computation |
| `scenarios/swarm-manager/ui/src/components/ui/rollup-progress-bar.tsx` | Frontend progress bar — may need archived segment |
| `scenarios/swarm-manager/ui/src/components/ui/rollup-progress-bar.test.tsx` | Frontend progress bar tests |

## Phases

### Phase 1: Backend Fix
1. Confirm root cause by checking which initiative items fail to load or have unexpected status
2. Implement chosen fix strategy in `ComputeRollup`
3. Update `RollupStatus` struct if adding/modifying fields
4. Update/add unit tests

### Phase 2: Frontend Update (if needed)
1. If archived items get their own treatment, update `rollupTotal()` and `getSegments()`
2. Update frontend tests

## Test Plan

- **Unit:** Extend `TestService_ComputeRollup` with cases for archived items and load failures
- **Unit:** Ensure `rollupTotal` frontend function handles the updated rollup shape
- **Integration:** Verify the initiative details page displays correct progress for an initiative with archived+completed items

## Risks

| Risk | Mitigation |
|------|------------|
| Changing rollup math could break graph projection rollup | Graph projection uses its own rollup logic in `projection.go`, not `ComputeRollup` |
| Archived items excluded from progress could confuse users about total count | Show archived count separately in labels |
