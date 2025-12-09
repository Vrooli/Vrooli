# SQLite Resource (Go)

A portable, serverless SQLite resource implemented in Go for Vrooli scenarios. This replaces the legacy Bash-based implementation (now in `resources/sqlite-old`) with a cross-platform binary suitable for Electron and other desktop packaging targets.

## Status
- Core Go CLI with manage/status/info and content operations (create, execute, list, get, backup, restore, remove, batch, CSV import/export, encrypt/decrypt).
- Replication, migrations, query helpers, stats implemented in Go. Web UI removed in the Go port; CLI tests now run `go test ./...`.

## Building / Installing
```bash
cd resources/sqlite
# Unix/macOS
./install.sh            # wraps packages/cli-core/install.sh
# Windows (PowerShell)
./install.ps1
```

A compatibility wrapper lives at `resources/sqlite/cli.sh` and will use a local build, an installed binary in `~/.vrooli/bin`, or a PATH binary. There is no `go run` fallback at runtime; build once via the install scripts.

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
resource-sqlite content import_csv <name> <table> <csv_file> [--no-header]
resource-sqlite content export_csv <name> <table> [output_file]
resource-sqlite content encrypt <name> <password>
resource-sqlite content decrypt <name> <password>

# replication
resource-sqlite replicate add --database <name> --target <path> [--interval <seconds>]
resource-sqlite replicate list
resource-sqlite replicate sync --database <name> [--force]
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

# stats
resource-sqlite stats enable <db>
resource-sqlite stats show <db>
resource-sqlite stats analyze <db>
resource-sqlite stats vacuum <db>

# tests
resource-sqlite test smoke|integration|unit   # runs go test ./... inside resources/sqlite (requires Go)
resource-sqlite content remove <name> [--force]
```

## Defaults / environment
Matches the prior resource defaults:
- `VROOLI_DATA` (default: `~/.vrooli/data`)
- `SQLITE_DATABASE_PATH` (`${VROOLI_DATA}/sqlite/databases`)
- `SQLITE_BACKUP_PATH` (`${VROOLI_DATA}/sqlite/backups`)
- `SQLITE_REPLICATION_PATH` (`${VROOLI_DATA}/sqlite/replicas`)
- `SQLITE_JOURNAL_MODE` (`WAL`), `SQLITE_BUSY_TIMEOUT` (10000 ms), `SQLITE_CACHE_SIZE` (2000 pages), `SQLITE_PAGE_SIZE` (4096 bytes), `SQLITE_SYNCHRONOUS` (`NORMAL`), `SQLITE_TEMP_STORE` (`MEMORY`), `SQLITE_MMAP_SIZE` (268435456 bytes), `SQLITE_FILE_PERMISSIONS` (`0600`), `SQLITE_CLI_TIMEOUT` (30s), `SQLITE_BACKUP_RETENTION_DAYS` (7).

## Next steps
- Flesh out replication, migrations, query builder, stats, encryption/decryption, CSV import/export, and batch execution parity.
- Add parity tests mirroring the legacy Bash tests and wire `vrooli resource sqlite test …` to run Go tests (now uses `go test ./...`).
- Integrate build/install hooks into the resource lifecycle so `vrooli resource sqlite manage install` builds the Go binary automatically.
- Confirm Windows/macOS/Linux packaging behavior for Electron scenarios.
