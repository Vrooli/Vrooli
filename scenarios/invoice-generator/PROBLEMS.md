# Known Issues - Invoice Generator

## Date: 2025-10-03
### Agent: scenario-improver (agi branch)

## Critical Issues (P0)

### 1. Schema Conflict Between helpers.go and schema.sql (ROOT CAUSE)
**Issue**: `api/helpers.go` creates tables with `VARCHAR(36)` for IDs, but `initialization/postgres/schema.sql` expects `UUID` type. Whichever runs first wins, creating type mismatches.

**Current State**:
- `companies` table has `id VARCHAR(36)` (from helpers.go)
- `invoices` table has `id UUID` (from schema.sql)
- `get_next_invoice_number` function expects UUID parameter but gets VARCHAR
- This type mismatch causes the function to fail silently or hang

**Impact**: CRITICAL - Core invoice creation completely broken, API timeouts

**Resolution Implemented**:
- Modified `get_next_invoice_number` function to accept VARCHAR instead of UUID
- Added fallback in API to bypass function and use timestamp-based invoice numbers
- API create endpoint still experiencing issues (likely in transaction or INSERT)

**Proper Fix Required**:
1. Remove ALL schema creation from `api/helpers.go` lines 109-280
2. Ensure `initialization/postgres/schema.sql` is run FIRST and ONLY
3. Remove `initializeDatabase()` call from `api/main.go`
4. Let lifecycle populate script handle schema initialization
5. Rebuild API and test

### 2. API Endpoint Timeouts (RELATED TO #1)
**Issue**: `/api/invoices/create` endpoint times out after 10+ seconds
**Root Cause**: Type mismatches and schema conflicts cause database operations to hang
**Status**: PARTIALLY FIXED - bypassed `get_next_invoice_number` function, but INSERT still hangs
**Next Step**: Need to trace exact SQL causing hang (likely foreign key constraint violation due to type mismatch)

### 3. Missing Test Infrastructure
**Issue**: Scenario has minimal test infrastructure (1/5 components per status output)
**Impact**: Cannot validate fixes or prevent regressions
**Required**:
- Add phased testing in `test/` directory
- Create unit tests for API endpoints
- Add CLI integration tests
- Create UI automation tests

## Working Features

✅ Health endpoint working: `http://localhost:19572/health` (< 10ms)
✅ UI server running and accessible: `http://localhost:35471`
✅ CLI installed and help working
✅ Database connection working
✅ Postgres schema partially loaded (UUID types for some tables)
✅ get_next_invoice_number function fixed to accept VARCHAR

## Failed Attempts This Session

1. ❌ Modifying function signature to accept UUID - table already has VARCHAR
2. ❌ Modifying function to accept VARCHAR - function works but API still hangs
3. ❌ Bypassing function entirely - INSERT still hangs (likely FK constraint)

## Root Cause Analysis

The fundamental problem is **duplicate schema management**:
- `helpers.go` tries to ensure tables exist using `CREATE TABLE IF NOT EXISTS`
- `schema.sql` has the "official" schema with proper types and constraints
- They disagree on data types (VARCHAR vs UUID)
- Race condition: whoever runs first creates incompatible schema
- Functions, foreign keys, and constraints fail due to type mismatches

## Action Plan for Next Agent

1. **FIRST**: Remove duplicate schema creation from `api/helpers.go`
2. **SECOND**: Drop and recreate database tables using ONLY `schema.sql`
3. **THIRD**: Test invoice creation endpoint
4. **FOURTH**: Add comprehensive test suite
5. **FIFTH**: Fix CLI JSON parsing issues
6. **SIXTH**: Test Ollama integration for AI extraction

## Technical Debt

- Duplicate schema management (helpers.go vs schema.sql)
- No automated tests
- No validation of schema consistency
- Mixed VARCHAR/UUID types across related tables
- Missing error handling in API endpoints

## Evidence

```bash
# Companies table (from helpers.go)
\d companies  # Shows id VARCHAR(36)

# Invoices table (from schema.sql)
\d invoices   # Shows id UUID

# Function signature mismatch
SELECT get_next_invoice_number('...'::uuid);  # FAILS with type error
```