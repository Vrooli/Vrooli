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
- Default; resolves user-scoped paths from OS defaults.

2. `desktop`
- Explicit desktop behavior (currently same policy as `auto`).

3. `mobile`
- Reserved profile for future mobile adapters (currently same policy as `auto`).

4. `vps`
- Server-style roots (`/etc`, `/var/lib`, `/var/cache`, `/var/log`, `/var/lib/vrooli-state`).

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
- OS identity (`RuntimeOS`)
- user directories (`UserHomeDir`, `UserConfigDir`, `UserCacheDir`)

This allows cross-platform policy testing without mutating process-global env.

## Usage Pattern

1. Build resolver once at startup.
2. Resolve/ensure class directories for active scenario.
3. Use `Resolver.Path` + `WriteFileAtomic` for persisted files.

## Future Extensions

1. Mobile-native adapters for sandboxed path providers.
2. Optional advisory locks in `state` class.
3. Retention/rotation helpers for logs/cache (separate package to preserve boundary clarity).
