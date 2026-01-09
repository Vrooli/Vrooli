# Picker Wheel - Known Problems & Solutions

## Problems Encountered

### 1. Standards Compliance
**Initial Issue**: Scenario auditor detected 361 standards violations, mostly formatting issues.
**Solution**: Applied Go formatting with `gofmt`, reduced violations significantly.
**Status**: ✅ Resolved

**Update 2025-10-26**: Further improvements made:
- Added Content-Type headers to all JSON API endpoints (health, wheels, history)
- Fixed HTTP status code for database errors (now returns 500 instead of 200)
- Enhanced Makefile usage documentation with complete command reference
- **Result**: 59 violations (down from 60), 0 security issues
- **Remaining**: 6 high-severity (Makefile format expectations), 53 medium-severity (logging, env validation, hardcoded values)

**Update 2025-10-27**: Health endpoint schema compliance fixed:
- Added `timestamp` and `readiness` fields to API health response
- Updated HealthResponse struct and healthHandler implementation
- Updated unit tests to validate new fields
- Added comprehensive test coverage suite (coverage_test.go)
- **Result**: 20 violations (down from 59 - 66% improvement), 0 security issues
- API health endpoint now fully compliant with schema requirements
- **Remaining**: Medium-severity violations for env validation and logging

**Update 2025-10-28**: Code quality improvements:
- Removed hardcoded port fallbacks from test-business.sh (improved test reliability)
- Removed duplicate Content-Type header in getWheelsHandler (cleaner code)
- **Result**: 19 violations (down from 20), 0 security issues
- **Note**: Remaining violations are mostly auditor false positives (SVG namespace constant, env vars that ARE validated, shell script variables)
- All critical env vars (API_PORT, UI_PORT, database config) properly validated
- Test coverage stable at 39.3% (documented technical debt requiring PostgreSQL for improvement)

### 2. Custom Wheel UI Implementation
**Issue**: Initial assessment indicated custom wheel UI wasn't implemented.
**Solution**: Verified that UI actually IS implemented - all elements and JavaScript functions exist and work.
**Status**: ✅ Resolved (was false alarm)

### 3. Missing Phased Test Structure
**Issue**: Scenario lacked modern phased testing structure.
**Solution**: Created comprehensive test phases:
- test-structure.sh: Validates file structure and configuration
- test-unit.sh: Tests Go API and CLI
- test-integration.sh: Tests API endpoints and UI
- test-performance.sh: Validates response times
- run-tests.sh: Orchestrates all phases
**Status**: ✅ Resolved

### 4. PostgreSQL Not Running
**Issue**: PostgreSQL resource not installed/running, using in-memory fallback.
**Solution**: API gracefully handles missing database with in-memory storage.
**Status**: ⚠️ Works but should be addressed for production

### 5. N8n Workflows Removed
**Issue**: Prior shared n8n workflow files have been removed.
**Solution**: Integrate automation directly in the service or provide scenario-specific workflows if needed.
**Status**: ⚠️ Action needed for automation coverage

## Recommendations

### High Priority
1. **Address Makefile Format**: Investigate exact format expectations for auditor compliance
2. **Structured Logging**: Replace log.Printf with structured logger to reduce logging violations
3. **Environment Validation**: Add validation for N8N_PORT and other env vars in UI server

### Medium Priority
4. **Database Persistence**: Start PostgreSQL resource for proper data persistence
5. **Workflow Automation**: Implement automation without shared n8n workflows or provide scenario-specific imports
6. **Performance Monitoring**: Add metrics collection for spin response times

### Low Priority
7. **Custom Themes**: Implement theme switching in UI
8. **Weight Testing**: Add tests for weighted probability selection

## Testing Commands

```bash
# Run all tests
cd scenarios/picker-wheel && make test

# Run specific phase
./test/phases/test-integration.sh

# Check API health
curl http://localhost:19899/health

# Test spin
curl -X POST http://localhost:19899/api/spin \
  -H "Content-Type: application/json" \
  -d '{"wheel_id": "yes-or-no"}'

# Create custom wheel
curl -X POST http://localhost:19899/api/wheels \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Test Wheel",
    "options": [
      {"label": "A", "color": "#FF0000", "weight": 1},
      {"label": "B", "color": "#00FF00", "weight": 1}
    ]
  }'
```
