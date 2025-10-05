# Known Problems and Solutions

## UI-API Connection Issues

### Problem: UI Cannot Connect to API
**Symptoms:**
- UI shows "Loading..." indefinitely
- Health status doesn't include API connectivity check
- API requests fail with proxy errors
- UI and API processes are orphaned (not managed by lifecycle system)

**Root Causes:**
1. **Orphaned Processes**: UI or API processes running outside lifecycle management
2. **Missing Health Checks**: UI health endpoint didn't verify API connectivity
3. **Port Conflicts**: Processes holding ports prevent clean restarts
4. **Incomplete Startup**: API binary exists but process not started

**Solution:**
1. **Kill Orphaned Processes:**
   ```bash
   pkill -f "scenario-auditor-api"
   pkill -f "vite.*scenario-auditor"
   ```

2. **Start Through Lifecycle System:**
   ```bash
   vrooli scenario start scenario-auditor
   ```

3. **Verify Connection:**
   ```bash
   # Check health endpoint includes API connectivity
   curl http://localhost:36224/health | jq '.checks.api'

   # Should show:
   # {
   #   "status": "healthy",
   #   "reachable": true,
   #   "url": "http://localhost:18507/api/v1"
   # }
   ```

**Prevention:**
- Always use `vrooli scenario start/stop` instead of manual process management
- Monitor health endpoint to catch connection issues early
- Check `vrooli scenario status scenario-auditor` before debugging

### Enhanced Health Check (Fixed 2025-10-05)

The UI health endpoint now includes API connectivity verification:

**Before:**
```json
{
  "status": "healthy",
  "service": "scenario-auditor-ui",
  "timestamp": "...",
  "uptime": 123
}
```

**After:**
```json
{
  "status": "healthy",
  "service": "scenario-auditor-ui",
  "timestamp": "...",
  "uptime": 123,
  "checks": {
    "api": {
      "status": "healthy",
      "reachable": true,
      "url": "http://localhost:18507/api/v1"
    }
  }
}
```

The status is now "degraded" (503) if API is unreachable, making it easy to detect connection issues.

## Lifecycle Management

### Always Use Lifecycle Commands

**✅ Correct:**
```bash
# Start scenario
vrooli scenario start scenario-auditor

# Check status
vrooli scenario status scenario-auditor

# View logs
vrooli scenario logs scenario-auditor

# Stop scenario
vrooli scenario stop scenario-auditor
```

**❌ Wrong:**
```bash
# Don't manually run processes
cd api && ./scenario-auditor-api &
cd ui && npm run dev &

# Don't use raw make commands for running
make run  # Use only for local development, not production
```

**Why:** The lifecycle system provides:
- Process management and monitoring
- Proper port allocation
- Health check integration
- Clean startup/shutdown
- Centralized logging

### Port Conflicts

If you see port conflicts:
```bash
# Find what's using the port
lsof -i :36224
lsof -i :18507

# Kill specific process
kill <PID>

# Or stop via lifecycle
vrooli scenario stop scenario-auditor

# Clean port locks if needed
rm ~/.vrooli/state/scenarios/.port_*.lock
```

## Testing

### Running Tests
Always ensure scenario is running before integration tests:

```bash
# Start scenario
vrooli scenario start scenario-auditor

# Wait for health
sleep 5

# Run tests
make test

# Or specific test suites
make test-integration
make test-api
```

## API Issues

### Scanner Initialization Failures

If API health shows scanner as "unhealthy":
```json
{
  "scanner": {
    "status": "unhealthy",
    "error": {
      "code": "SCANNER_INITIALIZATION_FAILED",
      "message": "Failed to initialize vulnerability scanner"
    }
  }
}
```

**Current Status:** This is a known limitation. The scanner component is being developed and is not critical for core functionality. The API remains operational with this warning.

**Workaround:** None needed. Core standards enforcement, rule management, and dashboard features work independently of the scanner.

## UI Issues

### Slow Data Loading

The UI may show skeleton loaders while fetching data from the API. This is normal during:
- Initial page load
- After running security scans with large result sets
- When browsing scenarios with many violations

**Expected behavior:** Data should load within 5-10 seconds on first visit, faster on subsequent visits due to caching.

If data never loads:
1. Check browser console for errors
2. Verify API is responding: `curl http://localhost:36224/api/v1/health/summary`
3. Check network tab for failed requests
4. Restart scenario if needed

### Dashboard Shows 0%

If System Status shows "0%" despite having scenarios:
- This means no security scans have been run yet
- Click "Run scan" to perform initial security analysis
- Percentage will update based on standards compliance

## Getting Help

If you encounter issues:

1. **Check status:** `vrooli scenario status scenario-auditor`
2. **View logs:** `vrooli scenario logs scenario-auditor`
3. **Test health:** `curl http://localhost:36224/health | jq .`
4. **Check API:** `curl http://localhost:18507/api/v1/health | jq .`
5. **Restart clean:** `vrooli scenario stop scenario-auditor && vrooli scenario start scenario-auditor`

---

**Last Updated:** 2025-10-05
