# SQLite Resource

A portable, serverless SQLite resource implemented fully in Go for Vrooli scenarios. This replaces the legacy Bash-based implementation with a cross-platform binary suitable for Electron and other desktop packaging targets.

## Status
- The resource is Go-native end to end. Resource-specific behavior no longer depends on Bash libraries.
- The single binary entrypoint lives in `cli/`, with all custom Go logic organized under `cli/internal/...`.
- SQLite-specific operations for content, replication, migrations, query helpers, and stats are implemented in Go.

## Architecture

`sqlite` now follows the updated `native-cli` resource philosophy. It is a repo-owned Go resource binary with a real operator command surface, not a wrapper over a third-party executable.

- `resource.json` is the declarative authority for install metadata, binary metadata, environment exports, portability, and CLI freshness.
- `cli/` is the single binary entrypoint and keeps bootstrap logic minimal.
- `cli/internal/app` owns SQLite-specific command registration and startup wiring.
- `cli/internal/env` owns canonical storage and environment configuration.
- `cli/internal/discovery` owns manifest and source-root discovery.
- `cli/internal/install` owns rebuild/install and Go test execution helpers.
- `cli/internal/version` owns manifest loading and runtime metadata access.
- `cli/internal/sqlite` owns SQLite-specific operations.

This is intentionally not scenario-style `cli/domains/...`. The architecture center is the resource itself and its repo-owned CLI, not a thin wrapper over an external tool.

## Go SQLite Driver Selection for Scenarios

When building Go scenarios that need SQLite, choosing the right driver is critical for portability:

### Recommended: `modernc.org/sqlite` (Pure Go)

**Use this driver for:**
- Desktop apps (Electron, Tauri) built with `scenario-to-desktop`
- Cross-compilation targets (e.g., building Linux binaries on macOS)
- Static binaries (`CGO_ENABLED=0`)
- Any scenario that may be packaged for distribution

```go
import (
    "database/sql"
    _ "modernc.org/sqlite" // Registers as "sqlite"
)

db, err := sql.Open("sqlite", "path/to/database.db")
```

**Key characteristics:**
- Pure Go implementation—no C compiler or CGO required
- Works with `CGO_ENABLED=0` static builds
- Cross-compiles without toolchain complexity
- Slightly larger binary size (~5MB overhead) but fully portable
- Driver name: `"sqlite"` (not `"sqlite3"`)

### Alternative: `github.com/mattn/go-sqlite3` (CGO)

**Use only when:**
- Running on servers with CGO available
- Maximum SQLite performance is critical
- No cross-compilation or static builds needed

```go
import (
    "database/sql"
    _ "github.com/mattn/go-sqlite3" // Registers as "sqlite3"
)

db, err := sql.Open("sqlite3", "path/to/database.db")
```

**Limitations:**
- **Fails with `CGO_ENABLED=0`**: Returns stub error at runtime
- Requires C compiler on build machine
- Cross-compilation requires target-specific C toolchains
- **Not compatible with `scenario-to-desktop`** or other packaging pipelines

### Migration Guide

If migrating from `go-sqlite3` to `modernc.org/sqlite`:

1. Update `go.mod`:
   ```diff
   - github.com/mattn/go-sqlite3 v1.14.22
   + modernc.org/sqlite v1.34.5
   ```

2. Update import:
   ```diff
   - _ "github.com/mattn/go-sqlite3"
   + _ "modernc.org/sqlite"
   ```

3. Update driver name in `sql.Open()`:
   ```diff
   - db, err := sql.Open("sqlite3", dsn)
   + db, err := sql.Open("sqlite", dsn)
   ```

4. Run `go mod tidy` and rebuild

## Building / Installing
```bash
cd resources/sqlite/cli
# Unix/macOS
./install.sh
# Windows (PowerShell)
./install.ps1
# From an installed CLI: `resource-sqlite manage install` will attempt to rebuild and install
# the Go binary into ${VROOLI_BIN:-~/.vrooli/bin} when Go is available.
```

`manage install` creates the canonical data directories and then attempts a best-effort binary rebuild through `packages/cli-core`.

The `cli/` entrypoint is intentionally small. The richer SQLite command surface is wired from `cli/internal/app` and the other `cli/internal/...` packages.

## CLI usage (current)
```bash
# manage
resource-sqlite manage install
resource-sqlite manage start|stop|restart

# info
resource-sqlite status
resource-sqlite info

# content
resource-sqlite content create <name>
resource-sqlite content execute <name> "<sql>"
resource-sqlite content list
resource-sqlite content get <name> [select query]
resource-sqlite content backup <name>
resource-sqlite content restore <name> <backup_file> [--force]
resource-sqlite content remove <name> --force
resource-sqlite content batch <name> [sql_file|-]
resource-sqlite content import_csv <name> <table> <csv_file> [--no-header] [--columns col1,col2]
resource-sqlite content export_csv <name> <table> [output_file]
resource-sqlite content encrypt <name> <password>
resource-sqlite content decrypt <name> <password>

# replication
resource-sqlite replicate add --database <name> --target <path> [--interval <seconds>]
resource-sqlite replicate list
resource-sqlite replicate sync --database <name> [--force]
resource-sqlite replicate sync --all [--force]   # syncs due replicas based on stored intervals
resource-sqlite replicate verify --database <name>
resource-sqlite replicate toggle --database <name> --target <path> [--enable|--disable]
resource-sqlite replicate remove --database <name> --target <path>

# migrations
resource-sqlite migrate init <name>
resource-sqlite migrate create "description"
resource-sqlite migrate up <name> [target_version]
resource-sqlite migrate status <name>

# query helpers
resource-sqlite query select <db> <table> [--where expr] [--order expr] [--limit n]
resource-sqlite query insert <db> <table> col=val [col=val...]
resource-sqlite query update <db> <table> col=val [col=val...] --where expr

# stats (dbstat must be available in the SQLite build; otherwise stats show will report unavailability)
resource-sqlite stats enable <db>
resource-sqlite stats show <db>
resource-sqlite stats analyze <db>
resource-sqlite stats vacuum <db>

# tests
resource-sqlite test smoke|integration|unit|all   # runs go test ./... inside resources/sqlite (requires Go toolchain)
resource-sqlite content remove <name> [--force]
```

## Defaults / environment
Canonical resource storage:
- `RESOURCE_DATA_DIR` (default: `~/.local/share/vrooli/resources/sqlite`)
- `RESOURCE_STATE_DIR` (default: `~/.local/state/vrooli/resources/sqlite`)
- `SQLITE_DATABASE_PATH` (`${RESOURCE_DATA_DIR}/databases`)
- `SQLITE_BACKUP_PATH` (`${RESOURCE_DATA_DIR}/backups`)
- `SQLITE_REPLICATION_PATH` (`${RESOURCE_DATA_DIR}/replicas`)
- `SQLITE_MIGRATION_PATH` (`${RESOURCE_DATA_DIR}/migrations`)
- `SQLITE_REPLICATION_STATE_PATH` (`${RESOURCE_STATE_DIR}/replication`)
- `SQLITE_JOURNAL_MODE` (`WAL`), `SQLITE_BUSY_TIMEOUT` (10000 ms), `SQLITE_CACHE_SIZE` (2000 pages), `SQLITE_PAGE_SIZE` (4096 bytes), `SQLITE_SYNCHRONOUS` (`NORMAL`), `SQLITE_TEMP_STORE` (`MEMORY`), `SQLITE_MMAP_SIZE` (268435456 bytes), `SQLITE_FILE_PERMISSIONS` (`0600`), `SQLITE_CLI_TIMEOUT` (30s), `SQLITE_BACKUP_RETENTION_DAYS` (7).
- Backups respect `SQLITE_BACKUP_RETENTION_DAYS` by pruning older backups per database name when a new backup is taken.
- Encryption/decryption removes lingering `-wal`/`-shm` files to avoid leaking unencrypted data.

## Operations

See [docs/OPERATIONS.md](/home/matthalloran8/Vrooli/resources/sqlite/docs/OPERATIONS.md) for the architecture boundary and operator guidance.
