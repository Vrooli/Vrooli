# Known Bugs

## CRITICAL: Infinite Loop in Task Loading (2025-10-01)

**Status**: Active
**Severity**: Critical - Causes API crash and fills disk
**Discovered**: 2025-10-01 during autoheal testing

### Symptom
The ecosystem-manager API enters an infinite loop when loading tasks from the queue directories. This causes:
- Log file to grow to 596MB in ~2 minutes
- API to crash (likely OOM or file descriptor exhaustion)
- 1,920,983 "Successfully loaded task" log entries
- System instability

### Evidence
From `/home/matthalloran8/.vrooli/logs/scenarios/ecosystem-manager/vrooli.develop.ecosystem-manager.start-api.log`:
- Started: 14:51:42
- Last log: 14:53:45
- Duration: ~2 minutes
- Log lines: 5,300,949
- Task load calls: 1,920,983

### Root Cause (Hypothesis)
The `storage.go:193-243` code appears to be in an infinite loop reading task files from all queue directories without proper termination condition or rate limiting. Likely causes:
1. Recursive directory traversal without exit condition
2. Event loop or polling mechanism firing too frequently
3. Missing break condition after loading tasks

### Reproduction
```bash
vrooli scenario start ecosystem-manager
# Wait 2-3 minutes
# API will crash with massive log file
```

### Temporary Workaround
1. Truncate the log file before starting: `> ~/.vrooli/logs/scenarios/ecosystem-manager/vrooli.develop.ecosystem-manager.start-api.log`
2. Monitor disk space
3. Restart immediately if log file grows rapidly

### Fix Required
1. Review `api/pkg/tasks/storage.go:193-243` for infinite loop
2. Add rate limiting to task loading
3. Add max iteration check with circuit breaker
4. Review API endpoint that triggers task loading
5. Add log rate limiting to prevent disk filling

### Priority
**CRITICAL** - This blocks autoheal from keeping ecosystem-manager running and will eventually fill the disk.

### Related Files
- `scenarios/ecosystem-manager/api/pkg/tasks/storage.go` (lines 193-243)
- `scenarios/ecosystem-manager/api/main.go` (API endpoint handling)

### Testing Plan
1. Add max iteration counter to task loading loop
2. Add timeout to prevent infinite execution
3. Add log sampling (only log every Nth task load)
4. Test with large number of archived tasks
5. Verify memory usage stays bounded
