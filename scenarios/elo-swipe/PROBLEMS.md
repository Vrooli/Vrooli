# Problems & Known Issues

## Current Issues

### 1. UI Import Path Issue (FIXED)
**Status:** ✅ Resolved  
**Date Fixed:** 2025-10-03  
**Problem:** UI failed to build due to incorrect iframe-bridge import path  
**Root Cause:** Import used `/child` subpath that doesn't exist in package exports  
**Solution:** Changed from `@vrooli/iframe-bridge/child` to `@vrooli/iframe-bridge`  
**Impact:** UI now builds and runs without errors

### 2. Test Infrastructure Missing (FIXED)
**Status:** ✅ Resolved  
**Date Fixed:** 2025-10-03  
**Problem:** No phased testing architecture, only legacy scenario-test.yaml  
**Solution Implemented:**
- Created `test/phases/` directory with smoke, unit, and integration tests
- Created `test/run-tests.sh` main test runner
- Updated Makefile to use new phased testing
- Added port auto-detection in test scripts
**Coverage Now:**
- Smoke tests: API health, CLI status, database connectivity
- Unit tests: Smart pairing logic, AI response parsing
- Integration tests: Full API workflows, CSV/JSON export

### 3. Go Unit Test Coverage Missing (FIXED)
**Status:** ✅ Resolved  
**Date Fixed:** 2025-10-03  
**Problem:** No unit tests for smart pairing logic  
**Solution:** Created `api/smart_pairing_test.go` with comprehensive tests for:
- Fallback pair generation
- AI response parsing  
- Edge cases (empty lists, single items, duplicate pairs)
**Test Results:** All tests passing (8 test cases)

## Design Limitations

### 1. Port Configuration Complexity
**Status:** ⚠️ Known Issue  
**Impact:** Low  
**Description:** API port assignment happens dynamically at runtime, making it harder to predict the exact port for testing  
**Workaround:** Test scripts now auto-detect the running port using `lsof`  
**Future Enhancement:** Consider using fixed port ranges or environment variable override

### 2. AI-Powered Features Require Ollama
**Status:** ⚠️ Limitation  
**Impact:** Medium  
**Description:** Smart pairing features depend on Ollama resource being available  
**Fallback:** System automatically falls back to algorithmic pairing if Ollama unavailable  
**Future Enhancement:** Make AI suggestions optional/configurable

### 3. Multi-User Collaborative Ranking Not Implemented
**Status:** 📋 P2 Feature  
**Impact:** Low (planned for v2.0)  
**Description:** Current system supports single-user ranking only  
**Workaround:** Users can export/import rankings to share results  
**Future Enhancement:** Real-time collaborative ranking sessions

## Historical Issues (Resolved)

### CLI Port Configuration Issue (Sept 2024)
**Fixed:** 2025-09-24  
**Problem:** CLI hardcoded to port 19294  
**Solution:** Updated CLI to read from API_PORT environment variable

### Database Connection Reliability (Sept 2024)  
**Fixed:** 2025-09-24  
**Problem:** Database connection failures on startup  
**Solution:** Implemented exponential backoff retry logic

### Health Check Endpoint Missing (Sept 2024)
**Fixed:** 2025-09-24  
**Problem:** No health check endpoint for monitoring  
**Solution:** Added `/api/v1/health` endpoint with database connectivity check

## Testing Gaps (Addressed)

### Previous Gaps:
- ❌ No smoke tests → ✅ Now have comprehensive smoke tests
- ❌ No unit tests → ✅ Now have Go unit tests for core logic
- ❌ No phased testing → ✅ Now have smoke/unit/integration phases

### Remaining Enhancements:
- UI automation tests using browser-automation-studio (P2)
- Load testing for performance validation (P2)
- Security penetration testing (P1 for production)

## Recommendations

### Immediate Actions
1. ✅ Migrate to phased testing architecture (DONE)
2. ✅ Add unit tests for critical logic (DONE)
3. ✅ Fix UI build issues (DONE)

### Short-term (Next Sprint)
1. Add P1 features: Undo/skip functionality during swiping
2. Implement ranking visualization (graph/chart view)
3. Add import lists from external sources

### Long-term (v2.0)
1. Implement collaborative ranking
2. Add historical preference analytics
3. Create preference prediction ML model
4. Optimize for mobile swiping experience

## Problem Resolution Process

When encountering new issues:
1. Document in this file with date and description
2. Assess severity (Critical/High/Medium/Low)
3. Implement fix or workaround
4. Update status and add to resolved section
5. Add regression test to prevent recurrence

---

**Last Updated:** 2025-10-03  
**Maintained By:** AI Agent  
**Review Frequency:** After each major change
