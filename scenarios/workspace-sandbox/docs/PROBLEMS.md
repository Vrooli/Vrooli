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

### Technical Debt to Address

1. **Driver abstraction completeness**: Ensure interface covers all operations
2. **Error message quality**: User-facing errors should guide resolution
3. **Test coverage for edge cases**: Empty sandboxes, binary files, symlinks
4. **Performance benchmarks**: Automated tracking of creation latency

## Questions for Product/User Feedback

1. Should sandbox creation require explicit confirmation, or default to auto-create?
2. How long should default TTL be? Hours? Days?
3. Is per-sandbox size limit critical, or total storage limit sufficient?
4. Should rejected sandboxes be auto-deleted or preserved for debugging?
5. What level of git operation blocking is acceptable?

---

*Last updated: 2025-12-17 by Generator Agent*
*Next review: After P0 implementation*
