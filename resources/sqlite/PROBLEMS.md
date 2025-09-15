# SQLite Resource - Known Issues and Solutions

## Fixed Issues

### 1. Query Builder Row Count Reporting (FIXED: 2025-09-14)
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

### 2. No Encryption Support
**Description**: SQLite encryption extensions are not supported.
**Workaround**: Use file system encryption or handle encryption at application layer.
**Priority**: P2 - Nice to have

### 3. No Web UI
**Description**: No graphical interface for database exploration.
**Workaround**: Use CLI commands or external SQLite browsers.
**Priority**: P2 - Nice to have

## Best Practices

1. **Always use busy timeout**: The resource sets a 10-second busy timeout for better concurrency
2. **Use WAL mode**: Enabled by default for better concurrent access
3. **Regular backups**: Use `resource-sqlite content backup` before major operations
4. **Monitor performance**: Use `stats analyze` to identify optimization opportunities