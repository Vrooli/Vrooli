# Problems Encountered - Privacy Terms Generator

## Date: 2025-10-20

### Security & Standards Improvements
**Changes**: Fixed all critical and high severity security/standards violations identified by scenario-auditor.

**Security Fixes**:
1. **CORS Wildcard (HIGH)**: Changed from wildcard `*` to configurable origin via `CORS_ALLOWED_ORIGIN` environment variable with safe default to `http://localhost:35000`
2. **Lifecycle Protection (CRITICAL)**: Added mandatory lifecycle checks to both API (main.go) and UI (server.js) to prevent direct execution
3. **Environment Variable Defaults (HIGH)**: Removed dangerous defaults for `API_PORT` and `UI_PORT` - now fail fast when missing

**Standards Fixes**:
1. **Makefile Structure**: Added missing `start` target and updated `.PHONY` declarations
2. **Makefile Help Text**: Updated to properly document `make start` as the required entry point
3. **Test Infrastructure**: Created comprehensive `test/run-tests.sh` script for phased test execution
4. **Service Configuration**: Fixed `lifecycle.setup.condition` to properly reference `api/privacy-terms-generator-api` binary

**Results**:
- Security vulnerabilities: 1 HIGH → 0 (100% reduction)
- Standards violations (critical/high): 9 → 0 (100% reduction)
- All core functionality preserved and verified through testing

## Date: 2025-10-03

### 1. Database Integration Silent Failures
**Problem**: The `db_query` function in `generator.sh` has issues with output parsing, causing database operations to fail silently.

**Impact**: Version history tracking and document storage aren't being recorded in the database despite the schema being in place.

**Workaround**: The scenario falls back to `simple-generator.sh` which generates documents without database persistence. This maintains functionality but loses version history benefits.

**Solution Needed**:
- Debug the `db_query` function's output parsing logic
- Add explicit error handling and logging for database operations
- Test database operations independently before integrating with generation flow

### 2. Qdrant Integration Not Fully Tested
**Problem**: Semantic search implementation created but not verified against running Qdrant instance.

**Status**: Code implements proper fallback to PostgreSQL full-text search when Qdrant is unavailable.

**Testing Needed**:
- Enable Qdrant resource in `.vrooli/service.json`
- Run `privacy-terms-generator search "data collection" --limit 5`
- Verify semantic search returns relevant clauses

## Date: 2025-09-28

### 1. Ollama Command Interface Issue
**Problem**: The `resource-ollama content prompt` command doesn't exist. The correct interface uses `content add`.

**Solution**: Updated all Ollama calls to use:
```bash
echo "${prompt}" | resource-ollama content add --model "${model}" --query "Generate"
```

### 2. Database JSON Syntax Errors
**Problem**: PostgreSQL throwing "invalid input syntax for type json" errors during template updates.

**Issue**: The metadata column update was trying to set invalid JSON values.

**Impact**: Template fetch operations fail but don't prevent functionality due to fallback mechanisms.

### 3. PDF Generation Timeout
**Problem**: PDF generation via Browserless times out when called from CLI.

**Possible Causes**:
- Browserless resource may need different invocation pattern
- File URL handling in headless browser context
- Missing HTML-to-PDF conversion parameters

**Workaround**: PDF generation code is in place but may need adjustment for the specific Browserless resource implementation.

### 4. PostgreSQL Direct Access
**Problem**: Direct `psql` command not available, must use `resource-postgres` CLI.

**Solution**: All database operations use `resource-postgres content execute` command with proper schema prefixing.

## Recommendations for Future Improvements

1. **Ollama Integration**: Consider creating a wrapper function that abstracts the Ollama CLI interface changes
2. **Database Migrations**: Implement proper migration system for schema changes
3. **PDF Generation**: Test with alternative PDF generation methods or investigate Browserless resource documentation
4. **Error Handling**: Add more robust error handling for resource command failures
5. **Testing**: Add integration tests that verify resource command outputs