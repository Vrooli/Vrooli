# NSFW Detector - Known Issues

## Current Problems

### npm Vulnerabilities (Low Priority)
**Issue**: The nsfwjs library (v2.4.2) has dependencies with known vulnerabilities:
- jpeg-js: High severity - infinite loop and resource consumption vulnerabilities
- form-data: Critical severity - unsafe random function vulnerability  
- tough-cookie: Moderate severity - prototype pollution vulnerability

**Impact**: Limited - vulnerabilities are in dependencies, not directly exploitable in our use case

**Workaround**: None currently - upgrading would require breaking changes to nsfwjs v4.x

**Resolution**: Wait for nsfwjs to update dependencies or consider alternative libraries

### Model Download Size
**Issue**: Full NSFW.js models are ~5MB each and not included by default

**Impact**: Resource runs in mock mode until models are downloaded

**Workaround**: Mock mode provides consistent test responses

**Resolution**: Implement lazy model downloading on first use

## Resolved Issues

### PID Synchronization (Resolved 2025-09-14)
**Issue**: Service restart tests were failing due to PID file synchronization issues

**Solution**: Replaced `kill -0` with `ps -p` for more reliable process checking

**Files Changed**: 
- lib/core.sh: Updated manage_stop() and manage_start() functions

### Test Failures (Resolved 2025-09-12)
**Issue**: Integration tests failing for restart functionality

**Solution**: Improved PID handling and process verification logic

**Files Changed**:
- lib/core.sh: Enhanced process management functions