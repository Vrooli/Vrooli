# Workflow Execution: adhoc

**Status**: ✅ Completed  
**Duration**: 1.6s  
**Steps Completed**: 12/12  
**Assertions**: 2/2 passed  
**Execution ID**: `7ea261cf-3c06-4149-8c5d-00a9461388ec`

## Execution Timeline

1. ✅ **ACTION_TYPE_NAVIGATE** `3cffdb5f-438a-4a43-8852-5e53aceefa9b` (0.12s)  
2. ✅ **ACTION_TYPE_WAIT** `wait-featured` (0.04s)  
3. ✅ **ACTION_TYPE_ASSERT** `assert-same-project` (0.03s)  
4. ✅ **ACTION_TYPE_CLICK** `open-same-project` (0.09s)  
5. ✅ **ACTION_TYPE_WAIT** `wait-project-media` (0.03s)  
6. ✅ **ACTION_TYPE_WAIT** `wait-saved-image` (0.03s)  
7. ✅ **ACTION_TYPE_CLICK** `select-saved-image` (0.07s)  
8. ✅ **ACTION_TYPE_WAIT** `wait-image-preview` (0.02s)  
9. ✅ **ACTION_TYPE_ASSERT** `assert-original-image` (0.02s)  
10. ✅ **ACTION_TYPE_EVALUATE** `record-proof-identifiers` (0.01s)  
11. ✅ **ACTION_TYPE_EVALUATE** `881027ad-297b-4a62-855f-c7903c46eaae` (0.02s)  
12. ✅ **ACTION_TYPE_SCREENSHOT** `44c2cba4-0331-42f0-a354-bdfeeace31cc` (0.17s)  

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

