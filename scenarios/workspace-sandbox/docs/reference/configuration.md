# Configuration

This document captures deployment-shape requirements that aren't
expressed in environment variables — things the binary alone can't fix,
that are wired into `.vrooli/service.json` instead.

## User-namespace wrapper (REQUIRED)

The default driver is **kernel overlayfs in an unprivileged user
namespace** (`overlayfs-userns`). For the kernel to allow unprivileged
overlayfs mounts (5.11+), the API process must already be inside a user
namespace before it tries to mount.

The lifecycle wrapper does this:

```jsonc
// .vrooli/service.json — develop.start-api
"run": "cd api && exec unshare -U -m -r ./workspace-sandbox-api"
```

The boot self-check in `main.go::NewServer` reads `/proc/self/uid_map`
via `driver.InUserNamespace`. If the selected driver is
`overlayfs-userns` and we're not inside a user namespace, the API
exits fatally with a message pointing at this file.

**Do not** invoke `./workspace-sandbox-api` directly outside the
lifecycle. If you need to run the binary by hand for debugging, wrap it
yourself:

```bash
unshare -U -m -r ./workspace-sandbox-api
```

### Why a fatal self-check, not a silent fallback?

Pre-Phase 5 the code tried to re-exec itself via `unshare` from inside
`main`. That worked but produced two confusing failure modes when the
deployment shape drifted:

- Some callers got a sandbox that mounted via fuse-overlayfs daemon-per-mount,
  burning ~5 GB of RSS at 100 sandboxes (the original memory-pressure
  incident).
- Others silently fell back to the copy driver, doubling disk usage and
  dropping process isolation, with no clear log signal that anything was
  off.

Failing fatally at boot makes the contract explicit: either the wrapper
runs the binary inside `unshare -U -m -r`, or you switch to the
`fuse-overlayfs` or `copy` drivers via `/api/v1/driver/select` (which
saves the preference for next boot).

## Switching drivers

Operators can change the active driver at runtime:

```bash
curl -X POST http://localhost:$API_PORT/api/v1/driver/select \
  -H 'Content-Type: application/json' \
  -d '{"driverId": "fuse-overlayfs"}'
```

`POST /api/v1/driver/select` invokes `driver.SwitchDriver`, which
validates `IsAvailable`, swaps the atomic pointer, and persists the
preference under `~/.local/share/workspace-sandbox/driver-preference.json`.
The change applies to new operations immediately; in-flight ops finish
with the prior driver.

## Database column

The active driver per sandbox is persisted in `sandboxes.driver_id`
(`TEXT NOT NULL DEFAULT 'overlayfs-userns'`). Older databases that
predate this column rename land with a `driver` column containing the
legacy `overlayfs` value; `main.go::migrateDriverColumn` runs at
startup and idempotently renames the column and backfills the value to
`overlayfs-userns`. Greenfield: there is no rollback path; the column
name and value space are the only truth.

## Isolation profiles

Process isolation is declared **only** by the active `IsolationProfile`
loaded from `FileProfileStore`. There is no preset fallback: requesting
an unknown profile ID returns HTTP 400 with `IsolationProfileNotFoundError`.
The two builtin profiles (`full`, `vrooli-aware`) are guaranteed by
`config.DefaultProfiles()`.

Available driver IDs:

| ID                  | Description                                                    |
|---------------------|----------------------------------------------------------------|
| `overlayfs-userns`  | Kernel overlayfs in a user namespace (default; flat memory).   |
| `fuse-overlayfs`    | Userspace daemon-per-mount fallback. Higher memory under load. |
| `overlayfs-root`    | Kernel overlayfs with CAP_SYS_ADMIN. Rarely correct.           |
| `copy`              | Cross-platform fallback (file copies). Slowest.                |

Use `GET /api/v1/driver/options` to list which drivers are available on
the current host along with their unmet requirements (kernel version,
fuse-overlayfs binary, etc.).

## Diagnostic access to merged directories

Because the API runs inside its own user/mount namespace, sandbox merged
directories are visible **only inside the API's namespace**. To inspect
one from a host shell:

```bash
sudo nsenter -t $(pidof workspace-sandbox-api) -U -m
cd ~/.local/share/workspace-sandbox/<sandbox-id>/merged
```

Or — preferred for scripting — use the file CRUD endpoints which already
run in-namespace:

```bash
curl http://localhost:$API_PORT/api/v1/sandboxes/$ID/files
curl "http://localhost:$API_PORT/api/v1/sandboxes/$ID/files/content?path=README.md"
```

See `docs/SEAMS.md` for the full driver-layer contract.
