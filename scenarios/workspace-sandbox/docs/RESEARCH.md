# Research Notes

## Uniqueness Check

**Search Date:** 2025-12-17

### Within Vrooli Repository
- `rg -l 'workspace-sandbox'` in scenarios/ - **No matches found**
- `rg -l 'sandbox|overlay|copy-on-write|bwrap|bubblewrap'` - Found references but no overlapping capability

### Related Scenarios

| Scenario | Relationship | Notes |
|----------|--------------|-------|
| `test-genie` | Partial overlap | Has containment package with Docker/bubblewrap providers for **process isolation**, but not file-system overlay sandboxing with diff/approval workflow |
| `agent-inbox` | Potential consumer | Could use workspace-sandbox for safe agent task execution |
| `ecosystem-manager` | Potential consumer | Could orchestrate sandboxed scenario runs |

### Key Differentiation from test-genie

The `test-genie` containment system (see `scenarios/test-genie/api/internal/containment/provider.go`) focuses on:
- **Process-level isolation** via Docker or bubblewrap
- **Security boundaries** for running untrusted code
- **Resource limits** (memory, CPU, network)

`workspace-sandbox` differs fundamentally:
- **File-system-level overlay** using overlayfs for copy-on-write
- **Path-scoped workspaces** with mutual exclusion semantics
- **Diff/patch generation** and approval workflow
- **Storage efficiency** focus (disk cost proportional to changes only)
- **Review UI** for hunk-level change approval
- **Safety from accidents**, not security from adversaries

## External References

### Linux Kernel Documentation
- [overlayfs Documentation](https://www.kernel.org/doc/html/latest/filesystems/overlayfs.html)
- Key concepts: lowerdir (read-only base), upperdir (writable layer), workdir (internal), merged (union view)

### bubblewrap (bwrap)
- [bubblewrap GitHub](https://github.com/containers/bubblewrap)
- Unprivileged sandboxing tool using Linux namespaces
- Used by Flatpak and other containerization tools
- Can mount filesystems with specific read/write permissions

### Git Diff/Patch Format
- [Unified Diff Format](https://www.gnu.org/software/diffutils/manual/html_node/Unified-Format.html)
- [Git-Format-Patch Documentation](https://git-scm.com/docs/git-format-patch)
- Important for diff generation and hunk-level approval parsing

### Prior Art

| Project | Description | Relevance |
|---------|-------------|-----------|
| `git worktree` | Multiple working directories from same repo | Alternative approach; full copy, not copy-on-write |
| Docker overlay2 | Container storage driver | Uses overlayfs; inspiration for layer management |
| LXC/LXD | Linux containers | Full isolation; heavier than needed |
| nsjail | Process isolation | Similar to bwrap; Google's approach |
| firejail | Security sandbox | More features than needed; overkill for safety focus |

## Technical Considerations

### overlayfs Requirements
- Linux kernel 3.18+ (4.0+ recommended for features)
- `CONFIG_OVERLAY_FS` kernel option enabled
- For unprivileged users: fuse-overlayfs or user namespaces with CAP_SYS_ADMIN

### bubblewrap Requirements
- Package: `bubblewrap` (Debian/Ubuntu), `bubblewrap` (Fedora), `bwrap` (Arch)
- No root required (uses user namespaces)
- Can bind-mount directories with specific permissions

### Disk Layout Proposal
```
/var/lib/vrooli/workspace-sandbox/
  ├── sandboxes/
  │   └── {uuid}/
  │       ├── meta.json           # sandbox metadata
  │       ├── upper/              # writable layer (changes)
  │       ├── work/               # overlayfs workdir
  │       └── merged/             # mount point
  └── config.json                 # global configuration
```

### Path Normalization
- Use `realpath()` to resolve symlinks
- Canonicalize path separators
- Ensure paths stay within project root
- Handle edge cases: `.`, `..`, trailing slashes

### Mutual Exclusion Algorithm
```
isAncestor(path, potentialAncestor):
  return path.startsWith(potentialAncestor + "/") || path == potentialAncestor

canCreateSandbox(newPath, activeSandboxes):
  for sandbox in activeSandboxes:
    if isAncestor(newPath, sandbox.scopePath):
      return false  # new path is under existing sandbox
    if isAncestor(sandbox.scopePath, newPath):
      return false  # existing sandbox is under new path
  return true
```

## Open Questions for Implementation

1. **fuse-overlayfs vs privileged overlayfs**: Should we support both? fuse-overlayfs is slower but doesn't need root.

2. **Database vs filesystem for metadata**: PostgreSQL gives us querying/transactions but adds dependency. SQLite would be simpler for single-server use.

3. **Process tracking granularity**: Track all child processes via cgroups, or just direct children via process groups?

4. **Diff format**: Standard unified diff, or Git-style with binary detection and rename tracking?

5. **Conflict resolution**: Re-run sandbox, regenerate diff on latest, or manual merge?

## Resources in Vrooli Ecosystem

Potentially useful resources for workspace-sandbox:
- `postgres` - Metadata storage
- `redis` - Optional caching for sandbox lookup
- `browserless` - UI testing for diff viewer

## Conclusion

workspace-sandbox fills a unique gap in the Vrooli ecosystem: providing fast, storage-efficient, reviewable workspace isolation for agent and tool execution. While test-genie provides process-level containment for security, workspace-sandbox focuses on file-system-level isolation for safety and reviewability. The overlayfs + bwrap combination provides the optimal balance of speed, storage efficiency, and isolation for the stated goals.
