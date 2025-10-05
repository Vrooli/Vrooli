# Known Issues & Resolutions

## Port Conflict with app-issue-tracker - Round 2 (RESOLVED - 2025-10-04)

### Issue
During ecosystem improvements, accessibility-compliance-hub was discovered running on port 36221, which is the **fixed required port** for app-issue-tracker (needed for Cloudflare secure tunnel access via app-issue-tracker.itsagitime.com).

### Root Cause
1. **Stale process**: An old python http.server process from accessibility-compliance-hub was still running on port 36221
2. **Legacy resources format**: The service.json still used the old v1.0 resources format (arrays) instead of v2.0 format (objects with enabled/required properties), causing jq parsing errors

### Resolution Steps
1. Killed stale python processes using port 36221
2. Updated resources section in `.vrooli/service.json` to v2.0 format with proper object structure
3. Restarted both scenarios:
   - **app-issue-tracker**: Confirmed on port 36221 (fixed, required for Cloudflare)
   - **accessibility-compliance-hub**: Auto-assigned port 40912 (within 40000-40999 range)

### Files Modified
- `.vrooli/service.json` - Migrated resources from v1.0 array format to v2.0 object format

### Verification
```bash
# Verify app-issue-tracker on required port
lsof -i :36221 -sTCP:LISTEN  # Should show app-issue-tracker

# Verify accessibility-compliance-hub in correct range
lsof -i -sTCP:LISTEN | grep python3 | grep "40[0-9][0-9][0-9]"  # Should show port 40000-40999

# Test UI access
curl -sf http://localhost:36221  # app-issue-tracker ✅
```

### Prevention
- Service.json v2.0 resources format prevents jq errors during port allocation
- Fixed ports (like 36221) are reserved and protected by the lifecycle system
- Port range allocation ensures scenarios stay within their designated ranges
- Regular cleanup of stale processes prevents port conflicts

---

## Port Conflict with app-issue-tracker - Round 1 (RESOLVED - 2025-10-03)

### Issue
The accessibility-compliance-hub scenario was using ports 3400 (API) and 3401 (UI), which fell within the reserved `vrooli_core` range (3000-4100). This caused a conflict with the app-issue-tracker scenario.

### Root Cause
- Original service.json used hardcoded ports in the reserved range
- Did not follow the port allocation pattern used by other scenarios
- Port ranges weren't properly defined in the v2.0 service.json format

### Resolution
Updated port configuration to use proper ranges:
- **API Port**: Changed from 3400 to range 20000-20999 (auto-assigned)
- **UI Port**: Changed from 3401 to range 40000-40999 (auto-assigned)

### Files Modified
1. `.vrooli/service.json` - Added proper `ports` section with ranges
2. `ui/app.js` - Updated hardcoded API URL to use environment variable
3. `README.md` - Updated documentation to reflect dynamic port assignment
4. `scenario-test.yaml` - Updated test URLs to use environment variables
5. `PRD.md` - Updated API endpoint documentation

### Verification
```bash
# Verify no hardcoded ports remain
grep -r "3400\|3401" scenarios/accessibility-compliance-hub

# Check JSON validity
jq empty scenarios/accessibility-compliance-hub/.vrooli/service.json

# Verify port ranges
jq '.ports' scenarios/accessibility-compliance-hub/.vrooli/service.json
```

### Prevention
- Always use port ranges in v2.0 service.json format
- Avoid hardcoding ports in application code
- Use environment variables (${API_PORT}, ${UI_PORT})
- Check port registry before assigning new ports
- Follow the port allocation patterns from other scenarios

### Related
- app-issue-tracker maintains its original port assignments (15000-19999 API, 36221 UI)
- No conflicts remain between these scenarios
