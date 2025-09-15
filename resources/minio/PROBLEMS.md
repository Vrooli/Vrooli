# MinIO Resource - Known Issues and Solutions

## Issues Fixed (2025-09-13)

### 1. Hardcoded Port Fallback
**Problem**: The config/defaults.sh had a fallback to hardcoded port 9000 when port registry was unavailable.
**Solution**: Removed hardcoded fallback. Now fails explicitly if port registry is unavailable, following the port-allocation principles.
**Files Changed**: config/defaults.sh

### 2. Missing schema.json
**Problem**: v2.0 contract requires config/schema.json but it was missing.
**Solution**: Created comprehensive schema.json with all configuration options properly documented.
**Files Changed**: config/schema.json (created)

### 3. Obsolete Test Files
**Problem**: Old .bats test files were present alongside the new v2.0 test structure.
**Solution**: Removed all .bats files as testing is now handled by test/ directory.
**Files Removed**: *.bats files in config/ and lib/ directories

### 4. Backup Files
**Problem**: Backup files (cli.backup.sh, manage.backup.sh) were cluttering the codebase.
**Solution**: Removed backup files.
**Files Removed**: cli.backup.sh, manage.backup.sh

## Issues Fixed (2025-09-14)

### 5. AWS CLI Dependency
**Problem**: Tests and examples relied on AWS CLI which might not be available on all systems.
**Solution**: Replaced all AWS CLI operations with mc (MinIO Client) which is included in the MinIO container.
**Files Changed**: lib/test.sh, test/phases/test-integration.sh
**Benefit**: Improved portability - no external dependencies required for full functionality.

## Current Limitations

### 1. Default Credentials
**Impact**: Default credentials "minioadmin/minioadmin" are used if not overridden.
**Mitigation**: The system generates secure credentials on first install and saves them with 600 permissions.
**Recommendation**: Always use generated or custom credentials in production.

### 2. Console Port Not in Registry
**Impact**: Console port 9001 is not in the central port registry.
**Reason**: Secondary ports are typically not registered, only primary service ports.
**Status**: Working as designed.

## Performance Notes

- Health check response time: ~6-10ms (well under 1s requirement)
- Memory usage: ~262MB idle (well under 2GB limit)
- All smoke, integration, and unit tests passing
- v2.0 contract fully compliant