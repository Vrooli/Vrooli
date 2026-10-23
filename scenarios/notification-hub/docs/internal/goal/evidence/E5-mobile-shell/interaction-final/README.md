# Workflow Execution: adhoc

**Status**: ✅ Completed  
**Duration**: 6.0s  
**Steps Completed**: 19/19  
**Execution ID**: `cac057e6-af32-4faf-a495-d50d3f2e5f3b`

## Execution Timeline

1. ✅ **ACTION_TYPE_NAVIGATE** `6166e83a-8aa0-43ee-97f5-11bc1dc96bda` (0.66s)  
2. ✅ **ACTION_TYPE_KEYBOARD** `tab-skip` (0.02s)  
3. ✅ **ACTION_TYPE_WAIT** `wait-skip-transition` (0.26s)  
4. ✅ **ACTION_TYPE_EVALUATE** `observe-skip` (0.01s)  
5. ✅ **ACTION_TYPE_KEYBOARD** `enter-skip` (0.02s)  
6. ✅ **ACTION_TYPE_WAIT** `wait-main-focus` (0.16s)  
7. ✅ **ACTION_TYPE_EVALUATE** `observe-main` (0.01s)  
8. ✅ **ACTION_TYPE_CLICK** `click-dashboard` (0.07s)  
9. ✅ **ACTION_TYPE_EVALUATE** `observe-dashboard` (0.01s)  
10. ✅ **ACTION_TYPE_CLICK** `click-asks` (0.06s)  
11. ✅ **ACTION_TYPE_EVALUATE** `observe-asks` (0.01s)  
12. ✅ **ACTION_TYPE_CLICK** `click-history` (0.06s)  
13. ✅ **ACTION_TYPE_EVALUATE** `observe-history` (0.02s)  
14. ✅ **ACTION_TYPE_CLICK** `click-delivery` (0.06s)  
15. ✅ **ACTION_TYPE_EVALUATE** `observe-delivery` (0.01s)  
16. ✅ **ACTION_TYPE_CLICK** `click-settings` (0.09s)  
17. ✅ **ACTION_TYPE_EVALUATE** `observe-settings` (0.02s)  
18. ✅ **ACTION_TYPE_EVALUATE** `be9e531f-d977-40de-ad25-9400f1727e3a` (0.17s)  
19. ✅ **ACTION_TYPE_SCREENSHOT** `9b593757-6e6d-4956-90fb-c4f78f619dc1` (0.16s)  

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

