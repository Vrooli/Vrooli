# Known Issues - Invoice Generator

## Date: 2025-09-27
### Agent: scenario-improver

## Issues Discovered

### 1. Database Schema Mismatch (FIXED)
**Issue**: The API was creating its own schema in `helpers.go` that didn't match the official schema in `initialization/postgres/schema.sql`
**Impact**: Runtime errors for missing columns `last_reminder_sent` and `recurring_invoice_id`
**Resolution**: Added missing columns to official schema:
- Added `last_reminder_sent DATE` to invoices table
- Added `recurring_invoice_id UUID` to invoices table

### 2. API Endpoint Timeouts
**Issue**: Some API endpoints are timing out when called:
- `/api/invoices/create` - times out after 10+ seconds
- `/api/invoices/extract` - times out (likely trying to call Ollama for AI extraction)
**Impact**: Core functionality not working properly
**Status**: UNRESOLVED - needs investigation into API handler implementations

### 3. CLI JSON Parsing Errors
**Issue**: CLI commands returning "jq: parse error" when creating invoices
**Impact**: CLI not properly formatting responses
**Status**: UNRESOLVED - CLI script needs debugging

### 4. Duplicate Schema Management
**Issue**: Schema is being created in two places:
- Official: `initialization/postgres/schema.sql`
- API: `api/helpers.go` has CREATE TABLE statements
**Impact**: Confusion and potential schema drift
**Recommendation**: Remove schema creation from API code, rely solely on initialization schema

## Working Features

✅ Health endpoint working: `http://localhost:PORT/health`
✅ UI server running and accessible
✅ CLI installed and help working
✅ Client creation via CLI working
✅ Database connection working
✅ Schema fixes applied successfully

## Recommendations for Next Agent

1. **Debug API Handlers**: Investigate why create and extract endpoints are timing out
2. **Fix CLI Response Parsing**: Update CLI script to properly handle API responses
3. **Remove Duplicate Schema**: Clean up `helpers.go` to remove CREATE TABLE statements
4. **Test Ollama Integration**: Verify AI extraction is properly configured with Ollama
5. **Add Integration Tests**: Create proper test suite to validate all P0 requirements