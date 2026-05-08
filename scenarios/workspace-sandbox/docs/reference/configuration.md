# Configuration

This document captures deployment-shape requirements that aren't
expressed in environment variables — things the binary alone can't fix,
that are wired into `.vrooli/service.json` instead.

## Home overlay base directory (REQUIRED, validated at startup)

The per-sandbox host-`$HOME` overlay's upper/work/merged dirs live
**outside `$HOME`**. Putting them inside `$HOME` (the lower layer)
creates a self-referential overlayfs mount whose behavior is undefined
per kernel docs.

`config.ResolveHomeOverlayBaseDir` resolves the base directory at
startup; the resolved path is stored on `cfg.Driver.HomeOverlayBaseDir`
and threaded through every driver `Mount`/`Unmount`/`CleanupOrphan`
call.

Resolution order (first match wins):

1. **`WORKSPACE_SANDBOX_HOME_OVERLAY_BASE`** — explicit operator override.
   Most common in tests and in containerized deployments where
   `XDG_RUNTIME_DIR` isn't reliable.
2. **`$XDG_RUNTIME_DIR/workspace-sandbox`** — the systemd-blessed
   per-user runtime dir, e.g. `/run/user/1000/workspace-sandbox`. The
   default on logind-managed Linux desktops.
3. **Per-user workspace-sandbox directory under the system temporary
   area** — the cron/SSH/non-logind fallback. Created with mode `0700`
   if missing.

The resolver validates that the chosen path is **NOT** a subpath of
`$HOME` and fails fatally otherwise. There is no silent fallback. If
you need to override the location for a one-off run:

```bash
WORKSPACE_SANDBOX_HOME_OVERLAY_BASE=/var/cache/wsbox-home make start
```

[CODE: `api/internal/config/config.go::ResolveHomeOverlayBaseDir`] •
[CODE: `api/internal/driver/helpers.go::mountHomeOverlay`]

## Portable launcher (REQUIRED)

The default driver is **kernel overlayfs in an unprivileged user
namespace** (`overlayfs-userns`). For the kernel to allow unprivileged
overlayfs mounts (5.11+), the API process must already be inside a user
namespace before it tries to mount.

The lifecycle starts the portable launcher, which reads the saved driver
preference and chooses the correct process launch shape for the host:

```jsonc
// .vrooli/service.json — develop.start-api
"run": "cd api && exec ./workspace-sandbox-launcher ./workspace-sandbox-api"
```

The boot self-check in `main.go::NewServer` reads `/proc/self/uid_map`
via `driver.InUserNamespace`. If the selected driver is
`overlayfs-userns` and we're not inside a user namespace, the API
exits fatally with a message pointing at the launcher/safeguard
contract.

**Do not** invoke `./workspace-sandbox-api` directly outside the
lifecycle. If you need to run the binary by hand for debugging, invoke
the launcher:

```bash
./workspace-sandbox-launcher ./workspace-sandbox-api
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

Failing fatally at boot makes the contract explicit: either the launcher
can run the binary inside `unshare -U -m -r`, or you switch to the
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
The change applies to new operations immediately when the requested
driver is available in the current process; in-flight ops finish with
the prior driver. If the selected driver requires a different outer
launch shape, such as `overlayfs-userns` from a host-namespace API
process, the endpoint saves the preference and returns
`requiresRestart: true` so the launcher can activate it on next boot.

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

# Environment Variables

This section is the canonical reference for every workspace-sandbox
environment variable. The order tracks how `config.LoadFromEnv` reads
them; each entry lists the type, default, valid range, audience, and
related variables. A meta-test
(`internal/config/config_test.go::TestExposedKnobs_DocumentationParity`)
asserts that every env var referenced in `internal/config/config.go`
appears here, and vice versa, so this document cannot drift silently.

> Convention: durations are Go `time.ParseDuration` strings (`30s`,
> `15m`, `2h`, `24h`). Booleans accept `true`/`1`/`yes`/`on` (and
> case-insensitive variants); anything else parses as the documented
> default.

## Server

### `API_PORT` (string, REQUIRED)
HTTP listen port. Set by the lifecycle wrapper from the scenario port
allocator; no built-in default. Must parse as a number in `1..65535`.

- Audience: lifecycle / operators
- Validated: yes — `Validate()` rejects empty values and out-of-range numbers

### `WORKSPACE_SANDBOX_READ_TIMEOUT` (Duration, default `30s`)
Maximum duration `http.Server.ReadTimeout` will wait for a request
body. Higher values tolerate slow uploads but allow bad clients to
hold connections longer.

- Range: `1s` – any
- Audience: operators
- Related: `WORKSPACE_SANDBOX_WRITE_TIMEOUT`, `WORKSPACE_SANDBOX_IDLE_TIMEOUT`

### `WORKSPACE_SANDBOX_WRITE_TIMEOUT` (Duration, default `24h`)
Per-response write deadline. The default of `24h` effectively disables
the deadline so SSE log streams (`/processes/{pid}/logs/stream`) can
stay open for the lifetime of long-running agent runs. `0` is
explicitly allowed (and treated as "disabled" by the api-core stack);
any non-zero value must clear `1s`.

- Range: `0` (disabled) or `>= 1s`
- Audience: operators
- Related: `WORKSPACE_SANDBOX_READ_TIMEOUT`

### `WORKSPACE_SANDBOX_IDLE_TIMEOUT` (Duration, default `120s`)
Keep-alive idle timeout for the HTTP server. Distinct from
`WORKSPACE_SANDBOX_IDLE_TTL` (which governs sandbox-level GC eligibility).

- Range: `>= 1s`
- Audience: operators

### `WORKSPACE_SANDBOX_SHUTDOWN_TIMEOUT` (Duration, default `10s`)
Graceful-shutdown deadline. After this elapses, in-flight requests are
forcibly cancelled.

- Range: `>= 1s`
- Audience: operators

### `WORKSPACE_SANDBOX_CORS_ORIGINS` (comma-separated list, default empty = allow all)
Strict CORS allowlist for the response middleware. Empty disables the
allowlist (allow `*`). Otherwise responses include
`Access-Control-Allow-Origin` only for matching origins.

- Audience: operators / multi-tenant deployments
- Related: `internal/server/middleware.go::corsMiddleware`

## Capacity Limits

### `WORKSPACE_SANDBOX_MAX_SANDBOXES` (int, default `1000`)
Hard cap on active sandboxes. Operators raise/lower this to match the
host's filesystem and memory budget.

- Range: `1..100000`
- Audience: operators
- Related: `WORKSPACE_SANDBOX_MAX_TOTAL_SIZE_MB`

### `WORKSPACE_SANDBOX_MAX_SIZE_MB` (int, default `10240` — 10 GB)
Per-sandbox upper bound. Larger sandboxes are refused at create time.

- Range: `>= 1`
- Audience: operators

### `WORKSPACE_SANDBOX_MAX_TOTAL_SIZE_MB` (int, default `102400` — 100 GB)
Aggregate cap across all sandboxes. When exceeded, GC is more
aggressive.

- Range: `>= WORKSPACE_SANDBOX_MAX_SIZE_MB`
- Audience: operators

### `WORKSPACE_SANDBOX_DEFAULT_LIST_LIMIT` (int, default `100`)
Default page size for list endpoints (`/api/v1/sandboxes`,
`/api/v1/audit`, etc.).

- Range: `1..MaxListLimit`
- Audience: operators

### `WORKSPACE_SANDBOX_MAX_LIST_LIMIT` (int, default `1000`)
Maximum page size accepted by list endpoints. Requests above this
return 400.

- Range: `>= 1`
- Audience: operators
- Related: `WORKSPACE_SANDBOX_DEFAULT_LIST_LIMIT`

## Lifecycle & GC

### `WORKSPACE_SANDBOX_DEFAULT_TTL` (Duration, default `24h`)
Maximum sandbox lifetime. Sandboxes older than this are GC eligible
even if active.

- Range: `>= 1m`
- Audience: operators
- Related: `WORKSPACE_SANDBOX_IDLE_TTL`, `WORKSPACE_SANDBOX_GC_INTERVAL`

### `WORKSPACE_SANDBOX_IDLE_TTL` (Duration, default `4h`)
How long a sandbox can be unused (`LastUsedAt < now - IdleTTL`) before
becoming GC eligible. Must be `<= DefaultTTL` — the runtime cannot
reclaim "old idle" sandboxes after the absolute TTL has passed.

- Range: `> 0` and `<= DefaultTTL`
- Audience: operators
- Related: `WORKSPACE_SANDBOX_DEFAULT_TTL`

### `WORKSPACE_SANDBOX_GC_INTERVAL` (Duration, default `15m`)
How often the lifecycle reconciler runs.

- Range: `>= 1m`
- Audience: operators

### `WORKSPACE_SANDBOX_AUTO_CLEANUP_TERMINAL` (bool, default `true`)
When `true`, approved/rejected sandboxes are cleaned up after
`TerminalCleanupDelay`. When `false`, terminal sandboxes persist
until explicit operator action — and `TerminalCleanupDelay` MUST be
`0` (mutually exclusive: a non-zero delay with auto-cleanup off is
operator-confusing and rejected by `Validate()`).

- Range: bool
- Audience: operators
- Related: `WORKSPACE_SANDBOX_TERMINAL_CLEANUP_DELAY`

### `WORKSPACE_SANDBOX_TERMINAL_CLEANUP_DELAY` (Duration, default `1h`)
Delay before approved/rejected sandboxes are deleted. Honored only
when `AutoCleanupTerminal` is `true`.

- Range: `>= 0`
- Audience: operators
- Related: `WORKSPACE_SANDBOX_AUTO_CLEANUP_TERMINAL`

### `WORKSPACE_SANDBOX_PROCESS_GRACE_PERIOD` (Duration, default `100ms`)
Time between SIGTERM and SIGKILL when killing a process. Higher =
more graceful shutdown but slower cleanup.

- Range: `> 0`
- Audience: operators

### `WORKSPACE_SANDBOX_PROCESS_KILL_WAIT` (Duration, default `50ms`)
Time after SIGKILL we wait for the process to disappear from
`/proc`.

- Range: `> 0`
- Audience: operators

### `WORKSPACE_SANDBOX_AUTOHEAL_IDLE_GRACE` (Duration, default `30s`)
Minimum idle time before an unhealthy mount is auto-remounted.
Prevents the heal loop from interrupting an active run.

- Range: `>= 0` (0 disables the safety guard)
- Audience: operators
- Related: `WORKSPACE_SANDBOX_AUTOHEAL_MAX_RETRIES`,
  `WORKSPACE_SANDBOX_AUTOHEAL_BASE_BACKOFF`

### `WORKSPACE_SANDBOX_AUTOHEAL_MAX_RETRIES` (int, default `5`)
Consecutive remount failures before a sandbox is marked `Error` and
no longer auto-healed. The heal_state table persists this counter
across restarts (see `internal/repository/heal_state.go`).

- Range: `>= 1`
- Audience: operators

### `WORKSPACE_SANDBOX_AUTOHEAL_BASE_BACKOFF` (Duration, default `30s`)
Initial backoff after a failed remount. Doubled on each retry,
capped at 1h.

- Range: `> 0`
- Audience: operators

### `WORKSPACE_SANDBOX_MANUAL_REVIEW_TTL` (Duration, default `168h` — 7 days)
Maximum time a `manualReview=true` sandbox can sit idle past
`LastUsedAt` before the GC reconciler auto-denies all pending
changes and tears it down. `0` disables expiry.

- Range: `>= 0`
- Audience: operators / agent-manager integrations

## Driver

### `SANDBOX_BASE_DIR` (path, default `~/.local/share/workspace-sandbox`)
Root directory for sandbox upper/work/merged dirs. Must be writable
by the API user.

- Audience: operators

### `WORKSPACE_SANDBOX_HOME_OVERLAY_BASE` (path, default `$XDG_RUNTIME_DIR/workspace-sandbox` or per-user system temporary directory)
Where per-sandbox host-`$HOME` overlay dirs live. **Must be outside
`$HOME`** — see "Home overlay base directory" above. Failures here
are fatal at boot.

- Audience: operators / containerized deployments
- Related: `XDG_RUNTIME_DIR`, `HOME`

### `PROJECT_ROOT` (path, default discovered)
Default project root for sandboxes that don't pass one explicitly.
Discovered from `VROOLI_SOURCE_ROOT` / `VROOLI_ROOT` / `CWD` if
unset.

- Audience: operators / wrappers

### `SQLITE_PATH` (path, default derived via api-core/storage)
Explicit SQLite database file location. When unset, the API resolves
the path under the cross-platform data directory.

- Audience: operators / tests

## Policy

### `WORKSPACE_SANDBOX_BINARY_THRESHOLD` (int, default `8000`)
Number of leading bytes scanned for null bytes when classifying
files as binary. Higher = more accurate but slower for large files.

- Range: `>= 1`
- Audience: rare; tune only when binary detection misclassifies

### `WORKSPACE_SANDBOX_DEFAULT_NO_LOCK` (bool, default `true`)
When `true`, new sandboxes default to `noLock=true` (no scope
exclusion). Operators with strict mutual-exclusion needs flip this
to `false`.

- Audience: operators / security teams

### `WORKSPACE_SANDBOX_COMMIT_TEMPLATE` (string, default `"Apply sandbox changes ({{.FileCount}} files)"`)
Go-template for auto-generated commit messages. Placeholders:
`{{.SandboxID}}`, `{{.FileCount}}`, `{{.Actor}}`.

- Audience: operators

### `WORKSPACE_SANDBOX_COMMIT_AUTHOR_MODE` (enum, default `agent`)
Who gets attributed as the commit author. Valid: `agent`, `reviewer`,
`coauthored`.

- Audience: operators

### `WORKSPACE_SANDBOX_TEARDOWN_HOOK_CMD` (string, default auto-detected)
Explicit pre-teardown hook command. When unset, the API auto-enables
`vrooli scenario heal-from-sandbox` if `vrooli` is on `$PATH`.

- Audience: integrators

### `WORKSPACE_SANDBOX_TEARDOWN_TIMEOUT` (Duration, default `90s`)
Combined budget for all pre-teardown hooks. After this elapses, hook
processes get SIGKILL.

- Range: `> 0`
- Audience: operators

## Execution Defaults & Maxes

### `WORKSPACE_SANDBOX_DEFAULT_MEMORY_MB` (int, default `0` — unlimited)
Default memory cap (RSS, in MB) when a request omits the limit.

- Range: `0..MaxMemoryLimitMB`
- Audience: operators

### `WORKSPACE_SANDBOX_DEFAULT_CPU_SEC` (int, default `0` — unlimited)
Default CPU-time cap (seconds).

- Range: `0..MaxCPUTimeSec`
- Audience: operators

### `WORKSPACE_SANDBOX_DEFAULT_MAX_PROCS` (int, default `0` — unlimited)
Default process cap.

- Range: `0..MaxProcesses`
- Audience: operators

### `WORKSPACE_SANDBOX_DEFAULT_MAX_FILES` (int, default `0` — unlimited)
Default open-file-descriptor cap.

- Range: `0..MaxOpenFiles`
- Audience: operators

### `WORKSPACE_SANDBOX_DEFAULT_TIMEOUT_SEC` (int, default `0` — unlimited)
Default wall-clock timeout (seconds).

- Range: `0..MaxTimeoutSec`
- Audience: operators

### `WORKSPACE_SANDBOX_MAX_MEMORY_MB` (int, default `16384` — 16 GB)
Hard ceiling for memory requests. Requests above this are clamped.

- Range: `>= Default if Default > 0`
- Audience: operators

### `WORKSPACE_SANDBOX_MAX_CPU_SEC` (int, default `3600` — 1 hour)
Ceiling for CPU-time requests.

- Range: `>= Default if Default > 0`
- Audience: operators

### `WORKSPACE_SANDBOX_MAX_PROCS` (int, default `1000`)
Ceiling for process count.

- Range: `>= Default if Default > 0`
- Audience: operators

### `WORKSPACE_SANDBOX_MAX_FILES` (int, default `65536`)
Ceiling for open-file-descriptor count.

- Range: `>= Default if Default > 0`
- Audience: operators

### `WORKSPACE_SANDBOX_MAX_TIMEOUT_SEC` (int, default `7200` — 2 hours)
Ceiling for wall-clock timeout.

- Range: `>= Default if Default > 0`
- Audience: operators

### `WORKSPACE_SANDBOX_DEFAULT_PROFILE` (string, default `full`)
Default isolation profile when a request omits `isolationLevel`.
Builtin profiles are `full` and `vrooli-aware`; custom profiles are
loaded from the profile registry at startup.

- Audience: operators
- Related: `/api/v1/config/profiles`, "Isolation profiles" section above

## Integration

### `WORKSPACE_SANDBOX_AGENT_MANAGER_URL` (URL, default empty — discovery fallback)
Base URL for agent-manager callbacks. When empty, the Service
resolves the URL via `discovery.ResolveScenarioURLDefault` at request
time.

- Audience: operators
- Related: `WORKSPACE_SANDBOX_AGENT_MANAGER_SYNC_ENABLED`

### `WORKSPACE_SANDBOX_AGENT_MANAGER_SYNC_ENABLED` (bool, default `true`)
Whether the Service sends sync POSTs to agent-manager when sandbox
state changes (approved, rejected, etc.). Setting `false` makes the
Service silently skip the callback.

- Audience: operators

### `WORKSPACE_SANDBOX_AGENT_MANAGER_SYNC_TIMEOUT` (Duration, default `5s`)
Per-request deadline for outbound agent-manager sync.

- Range: `>= 0`
- Audience: operators

## Diff-archive retention

Diff archives (`sandbox_diff_archives` rows + their on-disk content
blobs) outlive the sandboxes they were captured from. The retention
reconciler enforces three independent levers; any archive matching ANY
active lever is evicted oldest-first. The reconciler runs on the same
cadence as the lifecycle reconciler (`WORKSPACE_SANDBOX_GC_INTERVAL`)
and can also be triggered on demand via
`POST /api/v1/admin/reconcilers/archive-retention`.

Env vars seed the initial values on first boot. Subsequent runtime
updates via `PUT /api/v1/config/retention` persist to a JSON file under
`ClassConfig` (`retention.json`) and take precedence on the next boot.

### `WORKSPACE_SANDBOX_RETENTION_MAX_AGE_DAYS` (int, default `90`)
Archives older than this many days are evicted unconditionally. Set
to `0` to disable age-based eviction.

- Range: `>= 0`
- Audience: operators
- Related: `WORKSPACE_SANDBOX_RETENTION_MAX_SIZE_BYTES`,
  `WORKSPACE_SANDBOX_RETENTION_MAX_PER_PROJECT`,
  `/api/v1/config/retention`,
  `/api/v1/admin/reconcilers/archive-retention`

### `WORKSPACE_SANDBOX_RETENTION_MAX_SIZE_BYTES` (int64, default `10737418240` — 10 GiB)
Total disk budget for archive blobs across all archives. When the sum
of `total_blob_bytes` exceeds this, the oldest archives are evicted
oldest-first until the running total falls under budget. Set to `0`
to disable size-based eviction.

- Range: `>= 0`
- Audience: operators

### `WORKSPACE_SANDBOX_RETENTION_MAX_PER_PROJECT` (int, default `0` = unlimited)
Caps how many archives may exist per `project_root`. Excess archives
within a project are evicted oldest-first to bring the count to the
cap. Set to `0` to disable the per-project cap.

- Range: `>= 0`
- Audience: operators
