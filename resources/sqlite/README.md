# SQLite Resource (Go)

A portable, serverless SQLite resource implemented in Go for Vrooli scenarios. This replaces the legacy Bash-based implementation (now in `resources/sqlite-old`) with a cross-platform binary suitable for Electron and other desktop packaging targets.

## Status
- Core Go CLI with manage/status/info and content operations (create, execute, list, get, backup, restore, remove, batch, CSV import/export, encrypt/decrypt).
- Replication, migrations, query helpers, stats implemented in Go. Web UI was intentionally removed from the port (serverless-only contract).
- CLI follows the resource v2.0 shape (manage/content/replicate/migrate/query/stats/test) and uses `packages/cli-core` for fingerprinting and auto-rebuilds when Go is present.

## Building / Installing
```bash
cd resources/sqlite
# Unix/macOS
./install.sh            # wraps packages/cli-core/install.sh
# Windows (PowerShell)
./install.ps1
# From an installed CLI: `resource-sqlite manage install` will attempt to rebuild and install
# the Go binary into ${VROOLI_BIN:-~/.vrooli/bin} when Go is available.
```

A compatibility wrapper lives at `resources/sqlite/cli.sh` and will use a local build, an installed binary in `~/.vrooli/bin`, or a PATH binary. There is no `go run` fallback at runtime; build once via the install scripts.

CLI auto-rebuilds only when a Go toolchain is present and the embedded fingerprint is stale. Otherwise, the bundled binary is used (matching the scenario CLI behavior).

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
Matches the prior resource defaults:
- `VROOLI_DATA` (default: `~/.vrooli/data`)
- `SQLITE_DATABASE_PATH` (`${VROOLI_DATA}/sqlite/databases`)
- `SQLITE_BACKUP_PATH` (`${VROOLI_DATA}/sqlite/backups`)
- `SQLITE_REPLICATION_PATH` (`${VROOLI_DATA}/sqlite/replicas`)
- `SQLITE_JOURNAL_MODE` (`WAL`), `SQLITE_BUSY_TIMEOUT` (10000 ms), `SQLITE_CACHE_SIZE` (2000 pages), `SQLITE_PAGE_SIZE` (4096 bytes), `SQLITE_SYNCHRONOUS` (`NORMAL`), `SQLITE_TEMP_STORE` (`MEMORY`), `SQLITE_MMAP_SIZE` (268435456 bytes), `SQLITE_FILE_PERMISSIONS` (`0600`), `SQLITE_CLI_TIMEOUT` (30s), `SQLITE_BACKUP_RETENTION_DAYS` (7).
- Backups respect `SQLITE_BACKUP_RETENTION_DAYS` by pruning older backups per database name when a new backup is taken.
- Encryption/decryption removes lingering `-wal`/`-shm` files to avoid leaking unencrypted data.

## Next steps
- Flesh out replication, migrations, query builder, stats, encryption/decryption, CSV import/export, and batch execution parity.
- Add parity tests mirroring the legacy Bash tests and wire `vrooli resource sqlite test …` to run Go tests (now uses `go test ./...`).
- Integrate build/install hooks into the resource lifecycle so `vrooli resource sqlite manage install` builds the Go binary automatically.
- Confirm Windows/macOS/Linux packaging behavior for Electron scenarios.
