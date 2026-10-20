# Workflow Execution: adhoc

**Status**: ✅ Completed  
**Duration**: 1.9s  
**Steps Completed**: 3/3  
**Execution ID**: `924da4b3-73ba-40a4-8310-5004a673a271`

## Execution Timeline

1. ✅ **ACTION_TYPE_NAVIGATE** `253ac926-1c55-46b8-90db-2c6d311437f3` (0.88s)  
2. ✅ **ACTION_TYPE_EVALUATE** `697060f4-e6ce-4d39-966b-f0e7a49988cd` (0.02s)  
3. ✅ **ACTION_TYPE_SCREENSHOT** `e9f555de-1f88-43cc-b766-81f8e0f418a7` (0.30s)  

## Quick Start

For programmatic access:

```bash
# Check pass/fail
cat result.json | jq '.success'

# Get error message (if failed)
cat result.json | jq '.error'

# View execution summary
cat result.json | jq '{status, steps_completed: .steps_completed, steps_failed: .steps_failed, duration_ms}'
```

## Artifacts

This execution generated the following artifacts:

- **result.json** - Quick pass/fail summary for programmatic access
- **timeline.json** - Full machine-readable execution data
- **execution-summary.md** - Detailed step-by-step breakdown
- **console-logs.md** - Console output organized by step
- **network-activity.md** - Network requests organized by step
- **assertions.md** - Detailed assertion results
- **screenshots/** - 1 screenshots captured during execution

