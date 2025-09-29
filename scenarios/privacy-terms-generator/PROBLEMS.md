# Problems Encountered - Privacy Terms Generator

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