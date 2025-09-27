# SQLite Resource - Known Issues and Solutions

## Fixed Issues

### 1. Input Validation Security (FIXED: 2025-09-15)
**Problem**: Database and table names were not validated, potentially allowing path traversal attacks.
- No validation on database names could allow `../../../etc/passwd` as a database name
- Special characters in names could cause command injection

**Root Cause**: Missing input validation in all database operation functions.

**Solution**: Added `sqlite::validate_name()` function that checks for:
- Empty names
- Path traversal attempts (`..` or `/`)
- Special characters (only alphanumeric, underscore, dot, and hyphen allowed)
- Maximum length (255 characters)

**Test Commands**:
```bash
# Test path traversal protection
vrooli resource sqlite content create "../../../etc/passwd"
# Should show: "Invalid database name: contains path characters"

# Test special character protection  
vrooli resource sqlite content create "test;rm -rf /"
# Should show: "Invalid database name: contains path characters"

# Test valid name
vrooli resource sqlite content create "test_valid_db"
# Should work normally
```

### 2. Query Builder Row Count Reporting (FIXED: 2025-09-14)
**Problem**: UPDATE and INSERT query builders were reporting incorrect affected rows and last insert IDs.
- UPDATE always showed 0 rows affected even when rows were updated
- INSERT always showed last ID as 0

**Root Cause**: The `changes()` and `last_insert_rowid()` functions were being called in separate SQLite sessions, which don't share state.

**Solution**: Combined the SQL commands into single session executions:
- UPDATE: `UPDATE ... ; SELECT changes();`
- INSERT: `INSERT ... ; SELECT last_insert_rowid();`

**Test Commands**:
```bash
# Test UPDATE row count
vrooli resource sqlite query update test_db test_table "id=1" "name='Updated'"
# Should show: "Updated 1 row(s) in test_table"

# Test INSERT last ID
vrooli resource sqlite query insert test_db test_table "name='New'"
# Should show: "Last inserted ID: [actual_id]"
```

## Current Limitations

### 1. Migration Rollback Not Implemented
**Description**: While migration tracking and application work, rollback functionality is not yet implemented.
**Workaround**: Manual database restoration from backups if rollback is needed.
**Priority**: P2 - Nice to have

### 2. Encryption Support (FIXED: 2025-09-15)
**Description**: SQLite encryption extensions were not supported.
**Solution**: Implemented database encryption using OpenSSL with AES-256 encryption.
**Usage**: Use `vrooli resource sqlite content encrypt <db_name>` and `decrypt <db_name>` commands.
**Status**: Fully implemented and tested

### 3. Web UI (FIXED: 2025-09-26)
**Description**: No graphical interface for database exploration.
**Solution**: Implemented comprehensive web UI with batch operations, CSV import/export, and statistics visualization.
**Usage**: Use `vrooli resource sqlite webui start` to launch the interface on port 8297.
**Status**: Fully implemented and tested

## Best Practices

1. **Always use busy timeout**: The resource sets a 10-second busy timeout for better concurrency
2. **Use WAL mode**: Enabled by default for better concurrent access
3. **Regular backups**: Use `resource-sqlite content backup` before major operations
4. **Monitor performance**: Use `stats analyze` to identify optimization opportunities