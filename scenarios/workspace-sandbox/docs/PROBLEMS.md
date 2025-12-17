# Problems & Deferred Ideas

This document tracks known issues, open questions, and ideas deferred for future consideration.

## Open Issues

### Linux-Only Implementation
- **Issue**: Primary driver (overlayfs + bwrap) only works on Linux
- **Impact**: macOS and Windows users cannot use the sandbox
- **Mitigation**: Define clear driver interface; document fallback strategy
- **Future**: Consider fuse-based overlay for macOS, VFS filter for Windows

### Root/Privilege Requirements
- **Issue**: Standard overlayfs requires root; fuse-overlayfs is slower
- **Impact**: Unprivileged users face performance penalty
- **Mitigation**: Document both modes; auto-detect and use best available
- **Future**: Investigate user namespace + overlayfs combinations

### Mount Cleanup Reliability
- **Issue**: Crashed processes may leave mounts orphaned
- **Impact**: Disk space leaks; stale mounts blocking new sandboxes
- **Mitigation**: Periodic GC; idempotent cleanup; startup mount scan
- **Future**: systemd unit files for mount tracking; watchdog process

### Build Artifact Growth
- **Issue**: Build caches, node_modules, etc. grow sandbox size rapidly
- **Impact**: Storage efficiency degrades; GC becomes critical
- **Mitigation**: Document footgun; recommend .gitignore-aware exclusions
- **Future**: Smart exclude patterns; size alerts; auto-cleanup triggers

### Background Process Persistence
- **Issue**: Agents may spawn background processes that outlive sandbox stop
- **Impact**: Zombie processes; resource leaks; unexpected behavior
- **Mitigation**: Process group tracking; explicit cleanup on stop
- **Future**: cgroup-based tracking; forceful cleanup option

## Known Limitations

### Not a Security Boundary
- workspace-sandbox prevents accidental changes, not malicious ones
- A determined adversary can escape the sandbox
- Not suitable for running truly untrusted code

### Single-Server Design
- Current design assumes all sandboxes on one server
- Distributed scenarios would need additional coordination
- Metadata in PostgreSQL supports future distributed index

### Git Operations Blocked by Convention
- safe-git wrapper provides guidance, not enforcement
- Agent could bypass by using git binary directly
- Adequate for cooperation, not adversarial use

## Deferred Ideas

### P3+ Features (Future Consideration)

#### Distributed Sandbox Coordination
- Multiple Vrooli servers sharing sandbox state
- Requires distributed locking; adds complexity
- Wait for demand before implementing

#### Template Sandboxes
- Pre-warm sandboxes from common base states
- Faster creation for repeated scenarios
- Storage vs speed tradeoff

#### Snapshot/Restore
- Save sandbox state for later continuation
- Useful for long-running experiments
- Adds storage complexity

#### Real-time Collaboration
- Multiple agents in same sandbox with conflict detection
- Requires operational transforms or similar
- Very complex; defer until clear use case

#### IDE Integration
- VS Code extension for sandbox management
- Direct diff viewer integration
- Nice-to-have after core stabilizes

### Assumptions Requiring Future Validation

1. **Filesystem consistency during diff**: Assumes no writes occur while generating diff.
   If an agent writes to the sandbox while diff is being generated, results may be inconsistent.
   Future: Add optional file locking or copy-then-diff approach.

2. **Mount persistence between calls**: ✓ ADDRESSED - Added `IsMounted()` and `VerifyMountIntegrity()` methods to driver.
   Now operations can verify mount state before proceeding and provide clear errors.

3. **External command availability**: Assumes `diff`, `git`, and `patch` are in PATH.
   On minimal containers these may not be present.
   Future: Check command availability at startup, report in health check.

4. **UpperDir/LowerDir persistence**: ✓ ADDRESSED - Added existence checks in diff generation.
   Clear error messages indicate when directories are missing.

### Idempotency & Concurrency (Recently Addressed)

The following issues were addressed in the 2025-12-16 Idempotency & Temporal Flow session:

1. **Create operation idempotency**: ✓ ADDRESSED - Added idempotency key support.
   Callers can provide an idempotency key; retries with same key return the existing sandbox.

2. **State transition idempotency**: ✓ ADDRESSED - All state transitions are now idempotent.
   Stop/Delete/Approve/Reject return success when already in target state.

3. **Concurrent modification detection**: ✓ ADDRESSED - Added optimistic locking with version field.
   Updates check version; ConcurrentModificationError on conflict.

4. **Scope overlap race condition**: ✓ ADDRESSED - TxRepository uses FOR UPDATE to lock during overlap check.
   Prevents two sandboxes from being created with overlapping scopes simultaneously.

### Technical Debt to Address

1. **Driver abstraction completeness**: ✓ PARTIALLY ADDRESSED - Added IsMounted/VerifyMountIntegrity methods
2. **Error message quality**: ✓ ADDRESSED - All errors now have hints
3. **Test coverage for edge cases**: Empty sandboxes, binary files, symlinks
4. **Performance benchmarks**: Automated tracking of creation latency
5. **Assumption documentation**: ✓ PARTIALLY ADDRESSED - Key assumptions documented in code comments
6. **Property-based testing**: Path normalization should use property-based tests to catch edge cases
7. **Idempotency key cleanup**: Consider adding TTL/expiration for old idempotency keys
8. **Retry metrics**: Track idempotency key hits to monitor replay behavior

### Test Infrastructure Issues (Discovered 2025-12-17)

1. **Binary file bash syntax check**: test-genie unit phase runs `bash -n` on Go binary files (`cli/workspace-sandbox`), causing false failures. Exit code 126 indicates binary cannot be executed as shell script.
   - **Impact**: Unit phase fails despite all Go/Node tests passing
   - **Mitigation**: Test-genie should detect ELF binaries and skip bash syntax checking
   - **Status**: Upstream test-genie issue

2. **BATS test discovery**: test-genie integration phase reports "no .bats suites found" even though BATS files exist in `bas/cases/*/api/*.bats` and run successfully when executed directly.
   - **Impact**: E2E validation not reflected in test coverage
   - **Mitigation**: Check test-genie BATS discovery path configuration
   - **Status**: Configuration/discovery issue to investigate

3. **Requirement validation ref path resolution**: Business phase reports "validation references non-existent file" for valid test references like `api/internal/handlers/handlers_test.go::TestHealthHandler` even though file and test function exist.
   - **Impact**: Completeness score shows 0% even with passing tests
   - **Mitigation**: Verify path resolution base directory in test-genie
   - **Status**: Path resolution issue to investigate

4. **Golangci-lint warnings**: 22 lint issues detected (errcheck, staticcheck, gosimple). Pre-existing issues not blocking functionality but affecting lint phase score.
   - **Impact**: Lint phase reports issues
   - **Mitigation**: Address unchecked errors, deprecated function usage
   - **Status**: Low priority cleanup

## Questions for Product/User Feedback

1. Should sandbox creation require explicit confirmation, or default to auto-create?
2. How long should default TTL be? Hours? Days?
3. Is per-sandbox size limit critical, or total storage limit sufficient?
4. Should rejected sandboxes be auto-deleted or preserved for debugging?
5. What level of git operation blocking is acceptable?

---

*Last updated: 2025-12-17 by Claude Opus 4.5 (Security Fixes + Test Infrastructure Investigation)*
*Next review: After test-genie path resolution issues are resolved*
