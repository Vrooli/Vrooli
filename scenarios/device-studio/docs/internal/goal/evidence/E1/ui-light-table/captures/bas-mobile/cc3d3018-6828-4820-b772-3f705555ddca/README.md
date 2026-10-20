# Workflow Execution: adhoc

**Status**: ✅ Completed  
**Duration**: 1.1s  
**Steps Completed**: 3/3  
**Execution ID**: `cc3d3018-6828-4820-b772-3f705555ddca`

## Execution Timeline

1. ✅ **ACTION_TYPE_NAVIGATE** `3688dc25-f6d7-4cb8-b5e8-2e48feec351d` (0.61s)  
2. ✅ **ACTION_TYPE_EVALUATE** `39ec76b0-b340-4c4a-ae97-fc11c0f19ba2` (0.04s)  
3. ✅ **ACTION_TYPE_SCREENSHOT** `cd7926b3-06f1-448d-a522-9cfc7410fd59` (0.17s)  

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

