# Workspace Sandbox

Fast, storage-efficient, copy-on-write workspaces for running agents and tools against a project folder without risking modification to the canonical repo.

## What It Is

workspace-sandbox creates fast, copy-on-write workspaces over a project directory, captures all changes as diffs, and supports an approval workflow to selectively apply changes back to the canonical repo via a controlled patch process.

**Key Capabilities:**
- Create sandboxes in ~1-2 seconds with overlayfs copy-on-write
- Disk cost proportional only to files actually changed
- Generate unified diffs from sandbox changes
- Review and approve changes at file or hunk level
- Apply approved patches to canonical repo via controlled process

## What It Is NOT

This system is designed for **safety from accidents**, not **security from adversaries**:
- Prevents unintended repo damage
- Reduces concurrency collisions
- Makes agent work reviewable and revertible
- **NOT** a hardened security sandbox - adversarial processes may escape

## Quick Start

```bash
# Start the workspace-sandbox services
cd scenarios/workspace-sandbox
make start

# Or use the CLI directly
vrooli scenario start workspace-sandbox
```

## Usage

### CLI Commands

```bash
# Create a new sandbox (defaults to mounting the full project root configured on the server)
# Reserve one or more paths to (1) avoid overlap with other sandboxes, and (2) set the default approval allowlist.
workspace-sandbox create --reserve /path/to/project/scenarios/my-scenario
workspace-sandbox create --reserve /path/to/project/scenarios/a --reserve /path/to/project/packages/b

# Optional: explicitly set the project root and/or mount scope
workspace-sandbox create --project /path/to/project --reserve /path/to/project/scenarios/my-scenario
workspace-sandbox create --project /path/to/project --scope /path/to/project --reserve /path/to/project/scenarios/my-scenario

# List active sandboxes
workspace-sandbox list

# Inspect a sandbox
workspace-sandbox inspect <sandbox-id>

# Generate diff from sandbox changes
workspace-sandbox diff <sandbox-id>

# Approve changes
workspace-sandbox approve <sandbox-id>
workspace-sandbox approve <sandbox-id> --files file1.go,file2.go

# Reject and cleanup
workspace-sandbox reject <sandbox-id>

# Delete a sandbox
workspace-sandbox delete <sandbox-id>

# Garbage collection
workspace-sandbox gc --older-than 24h
```

### API Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/api/v1/sandboxes` | Create a new sandbox |
| GET | `/api/v1/sandboxes` | List sandboxes |
| GET | `/api/v1/sandboxes/:id` | Get sandbox details |
| GET | `/api/v1/sandboxes/:id/diff` | Get unified diff |
| POST | `/api/v1/sandboxes/:id/approve` | Approve and apply changes |
| POST | `/api/v1/sandboxes/:id/reject` | Reject changes |
| DELETE | `/api/v1/sandboxes/:id` | Delete sandbox |
| POST | `/api/v1/gc` | Run garbage collection |

### UI

The web UI provides a diff viewer for reviewing and approving changes:
- File-level approval toggles
- Hunk-level selection (P1 feature)
- Sandbox metadata display
- One-click approve/reject

Access at `http://localhost:<UI_PORT>` after starting the scenario.

## Architecture

### Linux Driver Stack

```
┌─────────────────────────────────────────────────────────────┐
│                     User Namespace                          │
│  (API runs here with UID 0 mapping for unprivileged mount)  │
│                                                             │
│  ┌─────────────────────────────────────────┐                │
│  │           Sandbox Merged View           │                │
│  │         (read-write mount point)        │                │
│  └─────────────────────────────────────────┘                │
│                      │                                      │
│           ┌──────────┴──────────┐                           │
│           │                     │                           │
│  ┌────────▼────────┐   ┌───────▼────────┐                   │
│  │   Upper Layer   │   │  Lower Layer   │                   │
│  │   (writeable)   │   │  (read-only)   │                   │
│  │ ~/.local/share/ │   │  /project/root │                   │
│  │ workspace-sandbox│  │                │                   │
│  └─────────────────┘   └────────────────┘                   │
│           │                     │                           │
│           └──────────┬──────────┘                           │
│                      │                                      │
│           ┌──────────▼──────────┐                           │
│           │     overlayfs       │                           │
│           │   (kernel 5.11+)    │                           │
│           └─────────────────────┘                           │
└─────────────────────────────────────────────────────────────┘
```

**Key insight:** The overlay mount is only visible inside the user namespace when using the kernel overlayfs + userns driver. When using the FUSE Overlayfs driver, the merged directory is directly accessible on the host. In both cases, approvals follow explicit acceptance rules rather than reserved paths.

## Scope vs Reserved Paths

The sandbox has two distinct path concepts:

- **Scope (mount scope):** The copy-on-write view that the agent sees. By default this is the full project root, so agents can modify any file in the repo without touching the canonical checkout.
- **Reserved paths (conflict lock only):** One or more prefixes used to prevent overlapping sandboxes. Approval behavior is controlled by acceptance rules, not reserved paths.
- **No-lock runs:** Explicit opt-in mode that disables reserved-path locking. Use only for investigative sandboxes where you plan to discard all changes.

### Process Isolation

Sandboxed processes run under bubblewrap (bwrap) with:
- Canonical repo mounted read-only
- Sandbox merged view as working directory
- Optional network restrictions
- Process group tracking for cleanup

## Requirements

### System Requirements
- **Linux kernel 5.11+** (recommended) - Enables unprivileged overlayfs via user namespaces
- Linux kernel 4.0+ (minimum) - Falls back to copy driver if user namespaces unavailable
- bubblewrap package (`apt install bubblewrap`)
- SQLite for metadata storage

### Driver Selection

workspace-sandbox automatically selects the best available driver:

| Priority | Driver | Requirements | Performance |
|----------|--------|--------------|-------------|
| 1 | `overlayfs-userns` | Linux kernel 5.11+, user namespaces enabled, launcher/safeguard satisfied | Best (~1-2s creation) |
| 2 | `fuse-overlayfs` | Linux with fuse-overlayfs, FUSE, and fusermount | Good, higher memory under load |
| 3 | `overlayfs-root` | Linux with root or CAP_SYS_ADMIN | Fast, privileged |
| 4 | `copy` | Any OS | Slowest, copies files |

**How it works:**

On kernel 5.11+, the lifecycle starts `workspace-sandbox-launcher`,
which reads `driver-preference.json` and places the API inside
`unshare -U -m -r` when the selected/default driver is
`overlayfs-userns`. Inside that namespace, the API appears as UID 0
and can mount overlayfs without actual root privileges.

```
$ vrooli scenario logs workspace-sandbox --step start-api --tail 5
2026/05/08 09:40:57 os=linux driverPreference=default launchMode=apparmor-unshare
2026/05/08 09:40:57 running in user namespace | kernel=6.14.0-33-generic
2026/05/08 09:40:57 driver: selected=overlayfs-userns inUserNamespace=true preferenceUsed=false
```

**Troubleshooting driver selection:**

If `overlayfs-userns` fails to launch, check:
1. Kernel version: `uname -r` (should be 5.11+)
2. User namespaces enabled: `cat /proc/sys/kernel/unprivileged_userns_clone` (should be 1)
3. unshare command available: `which unshare`
4. setup safeguard status: `vrooli setup status --environment development --resources none --scenarios workspace-sandbox`

Driver storage is resolved by the service-owned platform path policy. It is
created and permission-checked before the API serves traffic; there is no
application-level storage override or alternate-path fallback. The copy
driver is the portable execution path on platforms without a supported mount
backend. Protected execution is reported as unavailable when its selected
backend cannot be provided.

### Environment Variables

| Variable | Purpose |
|----------|---------|
| `API_PORT` | Port for the Go API server |
| `UI_PORT` | Port for the Vite dev server |
| `PROJECT_ROOT` | Root path for sandboxable directories |

## Documentation

- **[docs/ARCHITECTURE.md](./docs/ARCHITECTURE.md)** - Start here! Comprehensive guide with diagrams explaining how everything works
- [PRD.md](./PRD.md) - Product requirements and operational targets
- [docs/RESEARCH.md](./docs/RESEARCH.md) - Research notes and external references
- [docs/PROBLEMS.md](./docs/PROBLEMS.md) - Known issues and deferred ideas
- [docs/PROGRESS.md](./docs/PROGRESS.md) - Development progress log
- [requirements/README.md](./requirements/README.md) - Requirements traceability

## Known Footguns

1. **Background processes**: Agents may spawn processes that outlive the sandbox session
2. **Build artifacts**: node_modules, build caches grow sandbox size rapidly
3. **Long-lived sandboxes**: Changes accumulate; prefer ephemeral use
4. **Git operations**: stash/commit/checkout/clean/reset blocked by safe-git wrapper
5. **Namespace isolation**: With kernel overlayfs + userns, overlay mounts are only visible inside the API's user namespace; use the exec API for running commands in sandboxes. With FUSE Overlayfs, the merged dir is host-visible.

## Testing

```bash
# Run test suite
make test

# Or via vrooli
vrooli scenario test workspace-sandbox
```

## Contributing

Future agents should:
1. Update PRD.md and requirements/ when adding features
2. Append entries to docs/PROGRESS.md for significant changes
3. Document new footguns in docs/PROBLEMS.md
4. Stay within `scenarios/workspace-sandbox/` boundaries
