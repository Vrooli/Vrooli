# Branch Index Selection Bug Fix Summary

## Overview
Fixed the critical bug where all parallel branches were executing the same path (index 0) instead of their intended different paths due to a hardcoded branch index.

## The Problem
When the navigator's `getParallelBranches()` returns multiple paths like:
```
[
  [path1-step1, path1-step2],  // Branch 0 should execute this
  [path2-step1, path2-step2],  // Branch 1 should execute this
  [path3-step1, path3-step2],  // Branch 2 should execute this
]
```

All branches were executing path 0 because of the hardcoded `branchIndex = 0` on line 649.

## Solution Implemented

### 1. Added `branchIndex` to BranchExecution Interface
```typescript
interface BranchExecution {
    id: string;
    parentStepId: string;
    steps?: StepStatus[];
    state: "pending" | "running" | "completed" | "failed";
    parallel: boolean;
    branchIndex?: number; // NEW: Index for selecting which parallel path to execute
}
```

### 2. Updated createBranches to Assign Indices
```typescript
for (let i = 0; i < locations.length; i++) {
    const branch: BranchExecution = {
        // ... other properties
        branchIndex: parallel ? i : undefined, // Assign index for parallel branches
    };
}
```

### 3. Fixed determineBranchPath to Use Branch Index
```typescript
// Use the branch's index to select the correct parallel path
const branchIndex = branch.branchIndex ?? 0;

// Validate index is within bounds
if (branchIndex < parallelBranches.length && parallelBranches[branchIndex]) {
    path.push(...parallelBranches[branchIndex]);
    // Added debug logging
} else {
    // Log warning and fallback to first path
    this.logger.warn("[BranchCoordinator] Branch index out of bounds", {...});
}
```

## Benefits

1. **Correct Parallel Execution**: Each branch now executes its intended path
2. **Graceful Fallback**: Out-of-bounds indices fall back to the first path with a warning
3. **Better Debugging**: Added debug logging for path selection
4. **Backward Compatible**: Sequential branches (non-parallel) remain unaffected

## Test Coverage

Created comprehensive tests in `branchParallelPaths.test.ts`:
- ✅ Parallel branches execute different paths
- ✅ Out-of-bounds branch indices handled gracefully
- ✅ Sequential branches work without indices
- ✅ Debug logging verified

## Example Usage

When creating 3 parallel branches:
- Branch 0 executes parallelBranches[0]
- Branch 1 executes parallelBranches[1]  
- Branch 2 executes parallelBranches[2]

If only 2 paths exist but 3 branches are created:
- Branch 2 logs a warning and falls back to parallelBranches[0]

## Impact
- **No Breaking Changes**: This is a bug fix that makes the system work as intended
- **Performance**: Minimal impact, just an index lookup
- **All Tests Pass**: 17 tests across 3 test files