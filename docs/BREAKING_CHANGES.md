# Breaking Changes

## Overview

This document tracks all breaking changes in the Vrooli system, providing migration guides and compatibility notes for each major change.

## Version 2.0.0 (August 2025)

### Resource Directory Restructuring

<!-- EMBED:RESOURCE_RESTRUCTURE:START -->
**Change**: Moved from nested category structure to flat resource organization.

**Before**:
```
scripts/resources/{category}/{resource}/
resources/ollama/
resources/postgres/
```

**After**:
```
resources/{resource}/
resources/ollama/
resources/postgres/
```

**Migration Steps**:
1. Run migration script: `./scripts/migrate-resources.sh`
2. Update any hardcoded paths in custom scripts
3. Update `.vrooli/service.json` resource paths
4. Restart all resources after migration

**Impact**: All scripts referencing old paths will break.
<!-- EMBED:RESOURCE_RESTRUCTURE:END -->

### Bash to Python Orchestration

<!-- EMBED:PYTHON_ORCHESTRATION:START -->
**Change**: Replaced `simple-multi-app-starter.sh` with `app_orchestrator.py`.

**Breaking Changes**:
- Environment variable names changed
- Process management completely different
- Signal handling revised
- Log format changed to structured JSON

**Migration Steps**:
1. Update `service.json` to reference new orchestrator
2. Convert any custom bash hooks to Python
3. Update monitoring to parse new log format
4. Test startup/shutdown sequences thoroughly

**Compatibility**: No backward compatibility - must migrate.
<!-- EMBED:PYTHON_ORCHESTRATION:END -->

### Unified Embedding Service API

<!-- EMBED:EMBEDDING_API:START -->
**Change**: Consolidated duplicate embedding functions into unified service.

**Before**:
```bash
qdrant::embeddings::process_workflows()  # In manage.sh
qdrant::embeddings::process_scenarios()  # In manage.sh
# Each with duplicate logic
```

**After**:
```bash
qdrant::embedding::process_item()  # In embedding-service.sh
# Single unified function for all content types
```

**Migration Steps**:
1. Update any custom extractors to use new API
2. Change function calls in scripts
3. Update test mocks to new signatures

**Compatibility**: Old functions redirect to new API (deprecated).
<!-- EMBED:EMBEDDING_API:END -->

## Version 1.5.0 (July 2025)

### CLI Command Structure

<!-- EMBED:CLI_RESTRUCTURE:START -->
**Change**: Reorganized CLI commands for consistency.

**Before**:
```bash
vrooli develop start
vrooli test run-all
resource-qdrant embeddings init
```

**After**:
```bash
vrooli develop
vrooli test all
vrooli resource qdrant embeddings init
```

**Migration Steps**:
1. Update all scripts using old commands
2. Update CI/CD pipelines
3. Update documentation

**Compatibility**: Old commands show deprecation warning for 3 months.
<!-- EMBED:CLI_RESTRUCTURE:END -->

### Configuration File Format

<!-- EMBED:CONFIG_FORMAT:START -->
**Change**: Moved from JSON to YAML for configuration files.

**Before**: `.vrooli/config.json`
**After**: `.vrooli/config.yaml`

**Migration Steps**:
1. Run converter: `vrooli config migrate`
2. Validate new YAML files
3. Update any config parsers

**Compatibility**: System reads both formats, prefers YAML.
<!-- EMBED:CONFIG_FORMAT:END -->

## Version 1.0.0 (May 2025)

### Initial Stable Release

<!-- EMBED:V1_RELEASE:START -->
**Change**: First stable release with frozen APIs.

**Major Components Stabilized**:
- Resource management API
- Scenario execution engine
- Embedding system interfaces
- CLI command structure

**Breaking from Beta**:
- Removed experimental features
- Standardized error codes
- Formalized event schemas

**Migration from Beta**:
See migration guide: `docs/migration/beta-to-v1.md`
<!-- EMBED:V1_RELEASE:END -->

## Deprecation Notices

### Upcoming Deprecations (Version 2.1.0)

<!-- EMBED:UPCOMING_DEPRECATIONS:START -->
**Deprecated Features**:
1. **Legacy bash extractors** - Use Python extractors instead
2. **Direct Redis pub/sub** - Use event bus abstraction
3. **Manual resource installation** - Use automated installer
4. **Individual test commands** - Use test suites

**Timeline**:
- Version 2.1.0: Deprecation warnings added
- Version 2.2.0: Features hidden but available
- Version 3.0.0: Complete removal

**Migration Preparation**:
- Start using new Python extractors
- Switch to event bus API
- Test automated installation
- Group tests into suites
<!-- EMBED:UPCOMING_DEPRECATIONS:END -->

## API Compatibility Matrix

| Version | Resource API | Scenario API | Embedding API | CLI API |
|---------|-------------|--------------|---------------|---------|
| 2.0.0   | v2          | v2           | v2            | v2      |
| 1.5.0   | v1          | v1           | v2            | v2      |
| 1.0.0   | v1          | v1           | v1            | v1      |
| 0.9.0   | beta        | beta         | beta          | beta    |

## Migration Tools

### Available Migration Scripts

<!-- EMBED:MIGRATION_TOOLS:START -->
```bash
# Resource migration
./scripts/migration/migrate-resources.sh

# Configuration migration
./scripts/migration/migrate-config.sh

# Embedding index rebuild
./scripts/migration/rebuild-embeddings.sh

# Full system migration
./scripts/migration/migrate-all.sh --from 1.5.0 --to 2.0.0
```
<!-- EMBED:MIGRATION_TOOLS:END -->

### Rollback Procedures

<!-- EMBED:ROLLBACK:START -->
**Before Any Major Upgrade**:
1. Backup current state: `vrooli backup create --full`
2. Test upgrade in staging environment
3. Document custom modifications
4. Plan maintenance window

**Rollback Steps**:
1. Stop all services: `vrooli stop --all`
2. Restore backup: `vrooli backup restore --latest`
3. Downgrade version: `git checkout <previous-version>`
4. Restart services: `vrooli start`
5. Verify functionality

**Rollback Window**: Keep backups for 30 days after upgrade.
<!-- EMBED:ROLLBACK:END -->

## Compatibility Guidelines

### Semantic Versioning

<!-- EMBED:SEMVER:START -->
We follow semantic versioning (MAJOR.MINOR.PATCH):

- **MAJOR**: Breaking changes requiring migration
- **MINOR**: New features, backward compatible
- **PATCH**: Bug fixes, backward compatible

**Examples**:
- 2.0.0 -> 2.0.1: Safe, automatic update
- 2.0.0 -> 2.1.0: Safe, new features available
- 2.0.0 -> 3.0.0: Breaking, requires migration
<!-- EMBED:SEMVER:END -->

### Extension Points

<!-- EMBED:EXTENSION_POINTS:START -->
**Stable Extension Points** (won't break):
- Resource CLI interface (`cli.sh` contract)
- Event bus message schemas
- Scenario PRD format
- Workflow JSON structure

**Unstable Areas** (may change):
- Internal function signatures
- Private APIs (prefixed with `_`)
- Experimental features (marked EXPERIMENTAL)
- Implementation details
<!-- EMBED:EXTENSION_POINTS:END -->

## Testing for Compatibility

### Compatibility Test Suite

<!-- EMBED:COMPAT_TESTING:START -->
```bash
# Test backward compatibility
vrooli test compatibility --version 1.5.0

# Test migration scripts
vrooli test migration --from 1.5.0 --to 2.0.0

# Test API compatibility
vrooli test api --compatibility-mode

# Full compatibility check
vrooli test compatibility --full
```
<!-- EMBED:COMPAT_TESTING:END -->

### Breaking Change Detection

<!-- EMBED:BREAKING_DETECTION:START -->
**Automated Detection**:
- CI/CD runs compatibility tests on every PR
- API changes tracked with OpenAPI diff
- Database schema changes validated
- Configuration changes detected

**Manual Review Required For**:
- Semantic changes (same API, different behavior)
- Performance regressions
- Resource requirement changes
- Security model changes
<!-- EMBED:BREAKING_DETECTION:END -->

## Communication Policy

### Breaking Change Announcements

<!-- EMBED:ANNOUNCEMENT_POLICY:START -->
**Timeline**:
- **90 days before**: Initial announcement
- **60 days before**: Detailed migration guide
- **30 days before**: Final reminder
- **Release day**: Full documentation
- **30 days after**: Migration support period

**Channels**:
- GitHub releases page
- Project documentation
- CLI warnings (for deprecated features)
- Email to registered users (major changes)
<!-- EMBED:ANNOUNCEMENT_POLICY:END -->

## Historical Breaking Changes

### Pre-1.0 Breaking Changes

<!-- EMBED:PRE_V1:START -->
During beta (versions 0.x), breaking changes were frequent:

- **0.9.0**: Complete resource system rewrite
- **0.8.0**: Scenario format standardization  
- **0.7.0**: Event bus implementation
- **0.6.0**: CLI command restructure
- **0.5.0**: Database schema overhaul

**Note**: Beta versions had no compatibility guarantees.
<!-- EMBED:PRE_V1:END -->

## Support Policy

### Version Support Lifecycle

<!-- EMBED:SUPPORT_LIFECYCLE:START -->
- **Current Major**: Full support
- **Previous Major**: Security updates for 12 months
- **Older Versions**: Community support only

**Current Support Status**:
- Version 2.0.x: Full support ✅
- Version 1.5.x: Security updates until July 2026 ⚠️
- Version 1.0.x: End of life ❌
<!-- EMBED:SUPPORT_LIFECYCLE:END -->

## Conclusion

Breaking changes are sometimes necessary for system evolution, but we strive to minimize their impact through:

1. Clear communication and timeline
2. Comprehensive migration tools
3. Backward compatibility where possible
4. Thorough testing procedures
5. Adequate support periods

Always consult this document before upgrading and test in a non-production environment first.