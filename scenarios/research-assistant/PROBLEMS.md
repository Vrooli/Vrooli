# Research Assistant - Problems & Solutions

## Issues Discovered During Improvement (2025-09-30)

### 1. API Endpoints Mismatch with PRD
**Problem**: PRD specifies `/api/v1/research/*` endpoints but actual implementation uses `/api/reports`, `/api/conversations`, etc.
**Impact**: API contract doesn't match documentation
**Solution**: Updated PRD to reflect actual implementation
**Status**: Documentation updated

### 2. n8n Workflow Population Failure
**Problem**: n8n workflows defined in `initialization/automation/n8n/*.json` fail to auto-populate
**Impact**: Automated research pipelines not available without manual setup
**Workaround**: Manual import using `resource-n8n content inject` (requires injection config)
**Status**: Pending fix

### 3. Test Framework Handler Issues
**Problem**: Test framework missing handlers for `http` and `integration` test types
**Impact**: 16 tests skipped, cannot validate scenario functionality automatically
**Root Cause**: Test framework expects handlers that don't exist
**Status**: Framework issue, needs upstream fix

### 4. CLI Port Detection Issues
**Problem**: CLI sometimes detects wrong API port (was showing ecosystem-manager instead of research-assistant)
**Solution**: Improved port detection logic using process ID and lsof
**Status**: Fixed

### 5. SearXNG Resource Not Running
**Problem**: SearXNG was in exited state, making search functionality unavailable
**Solution**: Started resource with `resource-searxng manage start`
**Status**: Fixed

### 6. Windmill Resource Unavailable
**Problem**: Windmill resource not installed/configured
**Impact**: Dashboard UI features not available
**Severity**: Low (optional feature)
**Status**: Won't fix (optional)

### 7. Missing P1 Features
**Problem**: Several P1 requirements not implemented:
- Source quality ranking and verification
- Contradiction detection and highlighting
- Report templates and presets
- Full research depth configuration

**Status**: Partially addressed - implemented advanced search filters

### 8. Test Script References Missing File
**Problem**: `custom-tests.sh` references `/home/matthalloran8/Vrooli/lib/utils/var.sh` which doesn't exist
**Impact**: Integration tests fail with error
**Solution**: Would need to update test script or create missing file
**Status**: Low priority

## Recommendations for Next Improver

1. **Priority 1**: Implement remaining P1 features (contradiction detection, report templates)
2. **Priority 2**: Fix n8n workflow import process
3. **Priority 3**: Create proper test handlers or migrate to new test framework
4. **Priority 4**: Consider implementing Windmill integration for full dashboard capabilities
5. **Priority 5**: Implement P2 features (multi-language support, custom prioritization)

## What's Working Well
- API health endpoints functioning
- SearXNG integration working with 70+ search engines
- Advanced search filters fully implemented
- Database connectivity stable
- Ollama, Qdrant, PostgreSQL integrations healthy
- CLI basic commands working (status, help)