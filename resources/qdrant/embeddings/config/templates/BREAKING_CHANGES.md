# Breaking Changes

This document tracks all breaking changes, migration guides, and version history for the project.

## Version History

<!-- EMBED:VERSION:START -->
### Version X.Y.Z - YYYY-MM-DD
**Type:** [Major|Minor|Patch]
**Breaking:** [Yes|No]
**Migration Required:** [Yes|No]
**Summary:** Brief description of the release

#### Breaking Changes
- List all breaking changes
- API endpoints removed or changed
- Configuration format changes
- Database schema changes
- Dependency updates requiring code changes

#### Migration Guide
1. Step-by-step migration instructions
2. Scripts or commands to run
3. Configuration changes needed
4. Code changes required
5. Testing recommendations

#### Rollback Procedure
- How to rollback if issues arise
- Data backup recommendations
- Compatibility notes
<!-- EMBED:VERSION:END -->

## API Changes

<!-- EMBED:API:START -->
### [YYYY-MM-DD] API Endpoint Change
**Endpoint:** `/old/path` → `/new/path`
**Method:** GET/POST/PUT/DELETE
**Version:** Introduced in vX.Y.Z
**Deprecation Date:** When old endpoint was deprecated
**Removal Date:** When old endpoint will be/was removed
**Reason:** Why was this change necessary?

#### Request Changes
```json
// Old format
{
  "oldField": "value"
}

// New format
{
  "newField": "value",
  "additionalField": "required"
}
```

#### Response Changes
```json
// Old format
{
  "data": []
}

// New format
{
  "results": [],
  "metadata": {}
}
```

#### Migration Example
```javascript
// Old code
const response = await api.get('/old/path')
const items = response.data

// New code
const response = await api.get('/new/path')
const items = response.results
```
<!-- EMBED:API:END -->

## Configuration Changes

<!-- EMBED:CONFIG:START -->
### [YYYY-MM-DD] Configuration Format Change
**File:** config.yaml / .env / settings.json
**Version:** Changed in vX.Y.Z
**Required:** [Yes|No]
**Default Behavior:** What happens if not updated?

#### Old Format
```yaml
old_setting: value
deprecated_option: true
```

#### New Format
```yaml
new_setting: value
enhanced_option:
  enabled: true
  additional_config: value
```

#### Migration Script
```bash
#!/bin/bash
# Script to migrate configuration
./scripts/migrate-config.sh --from v1 --to v2
```
<!-- EMBED:CONFIG:END -->

## Database Schema Changes

<!-- EMBED:SCHEMA:START -->
### [YYYY-MM-DD] Schema Migration
**Version:** vX.Y.Z
**Migration File:** migrations/YYYYMMDD_description.sql
**Reversible:** [Yes|No]
**Downtime Required:** [Yes|No]
**Data Loss Risk:** [None|Low|High]

#### Changes
- Tables added: `new_table`
- Tables removed: `old_table`
- Columns added: `table.new_column`
- Columns removed: `table.old_column`
- Indexes changed: Description of index changes

#### Migration Commands
```bash
# Backup before migration
pg_dump database > backup_$(date +%Y%m%d).sql

# Run migration
npm run migrate:up

# Rollback if needed
npm run migrate:down
```

#### Data Migration
```sql
-- Example data migration
UPDATE users SET new_field = old_field WHERE old_field IS NOT NULL;
ALTER TABLE users DROP COLUMN old_field;
```
<!-- EMBED:SCHEMA:END -->

## Dependency Updates

<!-- EMBED:DEPENDENCY:START -->
### [YYYY-MM-DD] Major Dependency Update
**Package:** package-name
**Old Version:** 1.x.x
**New Version:** 2.x.x
**Breaking Changes:** List of breaking changes
**Required Code Changes:** What needs to be updated

#### Code Changes Required
```javascript
// Old usage
import OldAPI from 'package-name'
const instance = new OldAPI()

// New usage
import { NewAPI } from 'package-name'
const instance = NewAPI.create()
```

#### Common Issues
- Issue: Error message or symptom
  - Solution: How to fix it
<!-- EMBED:DEPENDENCY:END -->

## Feature Deprecations

<!-- EMBED:DEPRECATION:START -->
### [YYYY-MM-DD] Feature Deprecation
**Feature:** Name of deprecated feature
**Deprecated In:** vX.Y.Z
**Removed In:** vX.Y.Z (planned)
**Replacement:** What to use instead
**Reason:** Why is this being deprecated?

#### Migration Path
1. Identify usage of deprecated feature
2. Update to use replacement
3. Test thoroughly
4. Remove deprecated code

#### Detection Script
```bash
# Find deprecated usage in codebase
grep -r "deprecatedFunction" --include="*.js" --include="*.ts"
```
<!-- EMBED:DEPRECATION:END -->

## Behavioral Changes

<!-- EMBED:BEHAVIOR:START -->
### [YYYY-MM-DD] Behavioral Change
**Component:** What component changed behavior
**Old Behavior:** How it used to work
**New Behavior:** How it works now
**Reason:** Why was this changed?
**Impact:** Who/what is affected?

#### Examples
```javascript
// Old behavior
// Function returned null on error
const result = await process()
if (result === null) { /* handle error */ }

// New behavior
// Function throws exception on error
try {
  const result = await process()
} catch (error) { /* handle error */ }
```
<!-- EMBED:BEHAVIOR:END -->

## Environment Changes

<!-- EMBED:ENVIRONMENT:START -->
### [YYYY-MM-DD] Environment Requirement Change
**Requirement:** Node.js / Python / Database version
**Old Minimum:** version X
**New Minimum:** version Y
**Features Required:** Why the new version is needed
**Compatibility Notes:** What breaks with old versions

#### Upgrade Instructions
```bash
# Example upgrade commands
nvm install 18
nvm use 18
npm rebuild
```
<!-- EMBED:ENVIRONMENT:END -->

---

## Migration Checklist Template

Use this checklist when planning migrations:

### Pre-Migration
- [ ] Review all breaking changes
- [ ] Backup data and configurations
- [ ] Test migration in staging environment
- [ ] Prepare rollback plan
- [ ] Notify stakeholders
- [ ] Schedule maintenance window

### During Migration
- [ ] Run migration scripts
- [ ] Verify data integrity
- [ ] Update configurations
- [ ] Deploy new code
- [ ] Run smoke tests

### Post-Migration
- [ ] Verify all services are running
- [ ] Check error logs
- [ ] Monitor performance metrics
- [ ] Confirm with stakeholders
- [ ] Document any issues encountered

## Version Support Policy

| Version | Status | Support End | Notes |
|---------|--------|-------------|-------|
| 3.x.x | Current | - | Active development |
| 2.x.x | LTS | YYYY-MM-DD | Security fixes only |
| 1.x.x | EOL | YYYY-MM-DD | No longer supported |

## Communication Channels

- **Announcements:** Where breaking changes are announced
- **Discussion:** Where to discuss migration issues
- **Support:** How to get help with migrations

## Useful Links

- [Semantic Versioning](https://semver.org/)
- [Migration Scripts](./scripts/migrations/)
- [Compatibility Matrix](./docs/compatibility.md)
- [Release Notes](./CHANGELOG.md)