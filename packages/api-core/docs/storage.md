# Storage Module (`api-core/storage`)

## Purpose

`api-core/storage` provides a single, testable seam for runtime storage path policy and safe file operations.

It avoids leaking OS-specific path decisions into scenario code and keeps mutable data out of source trees.

## Architecture Boundaries

1. `Resolver`
- Responsibility: resolve absolute storage directories by class and scenario.
- Non-responsibility: IO migration, retention, backup policies.

2. Filesystem helpers
- Responsibility: create storage directories and perform atomic file writes.
- Non-responsibility: domain serialization and schema evolution.

3. Error model
- Responsibility: classify caller-facing storage failures.

## Storage Classes

1. `config`
- Durable configuration (small files, user/admin controlled).

2. `data`
- Primary mutable application data.

3. `cache`
- Rebuildable artifacts (safe to evict).

4. `logs`
- Operational logs and diagnostics.

5. `state`
- Runtime state/checkpoints/lockfiles.

## Profiles

1. `auto`
- Default; resolves user-scoped paths under the **operator runtime home** (`~/.vrooli`).

2. `desktop`
- Explicit desktop behavior (currently same policy as `auto`).

3. `mobile`
- Reserved profile for future mobile adapters (currently same policy as `auto`).

4. `vps`
- Server-style roots (`/etc`, `/var/lib`, `/var/cache`, `/var/log`, `/var/lib/vrooli-state`).

### User-profile default = the operator runtime home (`~/.vrooli`)

For the user-scoped profiles (`auto`/`desktop`/`mobile`), the five class roots resolve
under the operator runtime home via the `repo-contract-go` `runtime_home` authority
(`RuntimeHomeEntryPath`), **not** XDG/`os.UserConfigDir`/macOS `Library`/Windows `AppData`:

| Class | Default root |
|---|---|
| `config` | `~/.vrooli/config` |
| `data` | `~/.vrooli/data` |
| `cache` | `~/.vrooli/cache` |
| `logs` | `~/.vrooli/logs` |
| `state` | `~/.vrooli/state` |

Each is then class-scoped to `<root>/<app>/<scenario>` (e.g. `~/.vrooli/data/vrooli/<scenario>`).

This default is **OS-agnostic** — the runtime home is operator-home-shaped, so there is no
per-OS branching. `~/.vrooli` is the single machine-readable storage authority
(`.vrooli/repo-contract.json` `runtime_home`); see `docs/repo-contract.md` §"Operator Runtime Home".

**No fallback.** If the runtime-home contract cannot be loaded, resolution returns an error
rather than silently producing an XDG path. The sanctioned escape for a genuinely standalone
consumer outside a Vrooli checkout is the env/per-call overrides below — never an XDG fallback.

`home` is supplied via the `UserHomeDir` seam (defaults to `os.UserHomeDir`); composition roots
that may run under sudo should inject a sudo-aware resolver so data lands under the invoking
user's home, not `/root`.

## Overrides

### Global Root Override
- `VROOLI_STORAGE_ROOT`
- Maps all classes under `<root>/{config,data,cache,logs,state}`.

### Class-Specific Overrides
- `VROOLI_CONFIG_ROOT`
- `VROOLI_DATA_ROOT`
- `VROOLI_CACHE_ROOT`
- `VROOLI_LOGS_ROOT`
- `VROOLI_STATE_ROOT`

### Per-call Override
- `Options.RootOverride` (must be absolute)

## Safety Guarantees

1. Scenario ID validation
- Rejects empty/invalid IDs.
- Restricts to alnum, `-`, `_`, `.`.

2. Path traversal protection
- `Resolver.Path(...)` rejects absolute rel paths.
- Rejects `..` escapes from class root.

3. Atomic writes
- `WriteFileAtomic` writes to temp file in target dir, fsyncs, chmods, then renames.

## Testing Seams

`ResolverConfig` supports deterministic seams for:
- env reads (`EnvGet`)
- the operator home from which `~/.vrooli` is derived (`UserHomeDir`)

This allows policy testing without mutating process-global env. The `RuntimeOS`,
`UserConfigDir`, and `UserCacheDir` fields are retained accepted seams for API
compatibility but no longer influence the user-profile default (which is the
OS-agnostic runtime home). Tests that need a hermetic root should set
`VROOLI_STORAGE_ROOT` / per-class `VROOLI_*_ROOT` / `Options.RootOverride`, or pin
`HOME` to a temp dir.

## Usage Pattern

1. Build resolver once at startup.
2. Resolve/ensure class directories for active scenario.
3. Use `Resolver.Path` + `WriteFileAtomic` for persisted files.

## Future Extensions

1. Mobile-native adapters for sandboxed path providers.
2. Optional advisory locks in `state` class.
3. Retention/rotation helpers for logs/cache (separate package to preserve boundary clarity).
