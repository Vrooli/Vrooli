# Workflow Execution: adhoc

**Status**: ✅ Completed  
**Duration**: 3.8s  
**Steps Completed**: 22/22  
**Assertions**: 2/2 passed  
**Execution ID**: `01858ea1-03e4-48db-96c9-044684c31864`

## Execution Timeline

1. ✅ **ACTION_TYPE_NAVIGATE** `743bc0a7-3c92-47cf-9080-332689295ea4` (0.25s)  
2. ✅ **ACTION_TYPE_WAIT** `wait-create` (0.06s)  
3. ✅ **ACTION_TYPE_CLICK** `open-create` (0.10s)  
4. ✅ **ACTION_TYPE_WAIT** `wait-name` (0.09s)  
5. ✅ **ACTION_TYPE_INPUT** `name-project` (0.04s)  
6. ✅ **ACTION_TYPE_CLICK** `create-project` (0.07s)  
7. ✅ **ACTION_TYPE_WAIT** `wait-project` (0.03s)  
8. ✅ **ACTION_TYPE_CLICK** `open-project` (0.08s)  
9. ✅ **ACTION_TYPE_WAIT** `wait-media-panel` (0.02s)  
10. ✅ **ACTION_TYPE_UPLOAD_FILE** `choose-fixture` (0.03s)  
11. ✅ **ACTION_TYPE_CLICK** `submit-import` (0.06s)  
12. ✅ **ACTION_TYPE_WAIT** `wait-first-preview` (0.03s)  
13. ✅ **ACTION_TYPE_CLICK** `back-to-projects` (0.06s)  
14. ✅ **ACTION_TYPE_WAIT** `wait-project-after-back` (0.02s)  
15. ✅ **ACTION_TYPE_CLICK** `reopen-project` (0.07s)  
16. ✅ **ACTION_TYPE_WAIT** `wait-saved-source` (0.03s)  
17. ✅ **ACTION_TYPE_CLICK** `select-saved-source` (0.05s)  
18. ✅ **ACTION_TYPE_WAIT** `wait-reopened-preview` (0.03s)  
19. ✅ **ACTION_TYPE_ASSERT** `assert-original-filename` (0.03s)  
20. ✅ **ACTION_TYPE_ASSERT** `assert-decoded-dimensions` (0.03s)  
21. ✅ **ACTION_TYPE_EVALUATE** `fe6c107d-4a56-491b-b159-40c79e2bc37f` (0.02s)  
22. ✅ **ACTION_TYPE_SCREENSHOT** `57d4879d-b6b7-43e4-9993-0c39cf0673b9` (0.44s)  

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

