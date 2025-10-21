# Problems Found During Development

## Security Audit Findings
**Date**: 2025-10-20
**Status**: Documented - Acceptable for local development tool

### CORS Wildcard Configuration
The scenario-auditor flagged 3 HIGH severity findings for CORS wildcard configuration in server files.

**Analysis**: The CORS configuration implements origin reflection:
- When an Origin header is present, it's reflected back with `Vary: Origin`
- Wildcard (`*`) is only used when no Origin header is present (typically non-browser requests)
- This is acceptable for a local development tool that runs on localhost

**Decision**: No changes needed. The current implementation balances security with local development usability. For production deployments, users should configure specific allowed origins via environment variables.

**Reference**: ui/server.js:42-49, ui/server-express.js:40-43, ui/server-vite.js:40-43

## Database Schema Mismatch
**Date**: 2025-09-28
**Issue**: The PostgreSQL database schema doesn't match the expected structure defined in `initialization/storage/postgres/schema.sql`

### Expected columns (from schema.sql):
- campaigns table: id, name, description, color, icon, parent_id, sort_order, is_favorite, prompt_count, last_used, created_at, updated_at

### Actual database state:
- Missing columns: icon, parent_id
- This prevents campaigns API from working properly

### Root cause:
- Database initialization during resource population may have failed
- The schema might have been partially applied from an older version

### Workaround implemented:
- Modified API code to handle missing columns gracefully
- Set default values for missing fields (icon = "folder")
- Removed icon column from SELECT queries

### Permanent fix needed:
1. Ensure postgres resource is running before scenario starts
2. Run proper database migration to add missing columns:
```sql
ALTER TABLE campaigns ADD COLUMN IF NOT EXISTS icon VARCHAR(50) DEFAULT 'folder';
ALTER TABLE campaigns ADD COLUMN IF NOT EXISTS parent_id UUID REFERENCES campaigns(id) ON DELETE CASCADE;
```
3. Consider adding a migration system to handle schema updates

## Export/Import Implementation Status
**Date**: 2025-09-28
**Status**: Partially Complete

### Completed:
- Added ExportData struct with proper JSON structure
- Implemented exportData function with filtering options
- Implemented importData function with transaction support
- Added ID mapping to maintain relationships during import

### Issues found:
- Database schema prevents testing due to missing columns
- Cannot fully validate without working database

### Testing needed:
- Export of campaigns, prompts, tags
- Import with ID remapping
- Filter by campaign_id
- Include/exclude archived prompts

## Version History Implementation
**Date**: 2025-09-28
**Status**: Not Started

The prompt_versions table exists in schema but functions are still stubbed:
- getPromptVersions 
- revertPromptVersion

This feature requires:
1. Tracking prompt changes
2. Storing versions with change summaries
3. Ability to revert to previous versions