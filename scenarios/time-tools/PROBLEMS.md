# Time Tools - Known Issues & Solutions

## Current Issues

### 1. CLI Port Discovery (Medium Severity)
**Problem**: CLI uses `vrooli scenario port` command which doesn't work with current lifecycle
**Impact**: CLI commands fail to connect to API
**Workaround**: Hardcode port 18765 or use environment variable
**Solution**: Update CLI to use environment variable or port registry

### 2. Business Hours Calculation (Low Severity)
**Problem**: Business hours calculation doesn't fully exclude weekends/holidays
**Impact**: Duration calculations may include non-working time
**Workaround**: Use standard duration calculation
**Solution**: Implement complete business calendar logic with holiday support

### 3. Test Suite Integration (Low Severity)
**Problem**: Test scripts expect different port discovery mechanism
**Impact**: `make test` fails on API connection tests
**Workaround**: Run API tests directly with hardcoded port
**Solution**: Update test scripts to match lifecycle port allocation

## Resolved Issues

### 1. CORS Security Vulnerability (HIGH) - RESOLVED 2025-09-28
**Problem**: CORS configured with wildcard origin (*)
**Solution**: Implemented explicit allowed origins list
**Verification**: Security audit now shows 0 high vulnerabilities

### 2. API Startup Failure - RESOLVED 2025-09-28
**Problem**: PORT environment variable not set by lifecycle
**Solution**: Added VROOLI_LIFECYCLE_MANAGED and default port to service.json
**Verification**: API starts successfully via `make run`

### 3. Missing P0 Requirements - RESOLVED 2025-09-28
**Problem**: Date arithmetic and parsing endpoints were missing
**Solution**: Implemented `/time/add`, `/time/subtract`, and `/time/parse` endpoints
**Verification**: All endpoints respond correctly to test requests

## Performance Notes

- API response times: <50ms for most operations
- Startup time: ~2 seconds including database connection
- Memory usage: ~50MB steady state
- Database queries: Optimized with proper indexes

## Next Improvements

1. Complete business hours calculation with holiday calendar
2. Add timezone abbreviation support (PST, EST, etc.)
3. Implement recurring event generation
4. Add batch operations for bulk time conversions
5. Integrate with external calendar systems