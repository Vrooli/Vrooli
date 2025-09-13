# Mathlib Resource - Known Issues

## Current Problems

### 1. Port Conflict (Minor)
- **Issue**: Port 11458 may be occupied by parlant service
- **Workaround**: Stop parlant or reassign port in defaults.sh
- **Solution**: Port registry should handle conflicts automatically

### 2. Lean 4 Binary Installation (Major)
- **Issue**: Lean 4 requires internet access to download from GitHub
- **Current State**: Installation code is ready but actual binary download needs network
- **Impact**: Proof execution returns error without Lean binary
- **Solution**: Run `curl -sSfL https://raw.githubusercontent.com/leanprover/elan/master/elan-init.sh | sh` manually

### 3. Mathlib4 Size (Medium)
- **Issue**: Full Mathlib4 requires ~10GB disk space and significant download time
- **Impact**: Initial setup can take 10-30 minutes
- **Solution**: Use cache system and only download required modules

### 4. Memory Requirements (Medium)
- **Issue**: Compiling Mathlib proofs requires 4-8GB RAM
- **Impact**: May fail on resource-constrained systems
- **Solution**: Configure MATHLIB_MAX_MEMORY in defaults.sh

## Testing Issues

### Smoke Test Conflicts
- Tests may fail if service is already running
- Solution: Stop service before running tests or improve test isolation

### Missing Dependencies
- Tests will fail without actual Lean 4 installation
- Solution: Mock proof verification for testing without Lean

## Future Improvements

1. **Offline Installation**: Bundle Lean 4 binaries for offline deployment
2. **Incremental Mathlib**: Only download required Mathlib modules on demand
3. **Container Support**: Provide Docker image with pre-installed Lean/Mathlib
4. **Resource Limits**: Implement proper CPU/memory limits for proof execution
5. **Batch Processing**: Queue management for multiple simultaneous proofs