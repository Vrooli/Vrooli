# Windmill Community Edition Fixes - August 2025

## Overview
This document describes critical fixes applied to make Windmill Community Edition (CE) fully functional in the Vrooli environment.

## Issues Fixed

### 1. LSP Service Restart Loop ✅
**Problem**: LSP container continuously restarting with exit code 101
**Root Cause**: Community Edition lacks tantivy indexer support required for LSP
**Error**: `Cannot start the indexer because tantivy is not included in this binary/image`
**Solution**: Disabled LSP service in configuration with clear documentation

### 2. Native Worker Startup Failure ✅  
**Problem**: Native worker container failing to start
**Root Cause**: Volume mount of `/usr/local/bin` from host overriding container's windmill binary
**Error**: `exec: "windmill": executable file not found in $PATH`
**Solution**: Removed problematic volume mounts that were hiding the windmill executable

### 3. Image Version Stability ✅
**Problem**: Using dynamic `main` tag causing potential inconsistencies
**Solution**: Pinned to specific SHA256 digest for reproducible deployments

## Configuration Changes Applied

### `/docker/.env` Changes
```bash
# LSP disabled for CE compatibility
WINDMILL_ENABLE_LSP=no

# Native worker re-enabled after fix
WINDMILL_NATIVE_WORKER=yes

# Profiles updated
COMPOSE_PROFILES=internal-db,workers,native-worker

# Image pinned to specific digest
WINDMILL_IMAGE=ghcr.io/windmill-labs/windmill@sha256:b8b578c89adb3f0049459644bdad32437fd876c4a6d215396904378ddafec16f
```

### `/docker/docker-compose.yml` Changes
```yaml
# Removed volume mounts that caused issues:
# - /usr/local/bin:/usr/local/bin:ro  # REMOVED - hides windmill binary
# - /usr/bin:/usr/bin:ro               # REMOVED - causes GLIBC mismatch
# - /bin:/bin:ro                       # REMOVED - causes GLIBC mismatch
```

## Validation Results

### ✅ All Services Running
- **Main App**: Healthy at http://localhost:5681
- **Database**: PostgreSQL healthy
- **Workers**: 3 default workers + 1 native worker operational
- **API**: Responding with version CE v1.518.0

### ✅ Native Worker Functional
```bash
# Test command successful
docker exec windmill-vrooli-worker-native bash -c "echo 'Test'"
# Output: Test

# Windmill binary accessible
docker exec windmill-vrooli-worker-native windmill --version
# Output: Windmill v1.518.0
```

## Key Learnings

### Volume Mount Conflicts
**Lesson**: Never mount host directories that contain critical container binaries
- Mounting `/usr/local/bin` hides container's windmill executable
- Mounting `/bin` or `/usr/bin` causes GLIBC version mismatches

### Community vs Enterprise Features
**Lesson**: CE lacks certain features that must be disabled:
- LSP/Indexer (requires tantivy, EE only)
- Multiplayer support (EE only)
- Some advanced worker features

### Image Pinning Best Practice
**Lesson**: Always pin to specific digests for production stability
- Avoids unexpected updates
- Ensures reproducible deployments
- Simplifies debugging

## Remaining Limitations

### Native Worker Scope
The native worker now runs but with limited host access:
- Docker socket access ✅ (can manage containers)
- Container's own bash/tools ✅ (no GLIBC issues)
- Host filesystem access ❌ (no volume mounts)
- Host binaries ❌ (would cause library conflicts)

For full host system access, consider:
1. Running worker in privileged mode
2. Using host network namespace
3. Creating specific bind mounts as needed

## Maintenance Notes

### Future Updates
When updating Windmill:
1. Test in development first
2. Check CE vs EE feature compatibility
3. Pin to specific digest after validation
4. Update this documentation

### Monitoring
Watch for:
- Container restart loops
- Worker registration issues
- API connectivity problems
- Memory/CPU usage spikes

## Version Information
- **Windmill Version**: v1.518.0 CE
- **Image Digest**: sha256:b8b578c89adb3f0049459644bdad32437fd876c4a6d215396904378ddafec16f
- **Date Fixed**: August 7, 2025
- **Fixed By**: Claude (AI Assistant)