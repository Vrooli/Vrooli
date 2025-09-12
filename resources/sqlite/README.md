# SQLite Resource

Lightweight, serverless SQL database engine for local data persistence in Vrooli scenarios.

## Overview

SQLite is a self-contained, serverless, zero-configuration SQL database engine that provides ACID transactions and SQL92 compliance. Unlike traditional database servers, SQLite reads and writes directly to ordinary disk files, making it perfect for scenarios that need simple, reliable data storage without the overhead of a database server.

## Features

- **Zero Configuration**: No setup or administration required
- **Serverless**: No separate server process to manage
- **ACID Compliant**: Full transactional support with rollback
- **SQL92 Support**: Standard SQL syntax and features
- **Lightweight**: ~50MB memory footprint
- **Fast**: Optimized for local file I/O
- **Cross-Platform**: Works on Linux, macOS, and Windows
- **WAL Mode**: Write-Ahead Logging for better concurrency

## Quick Start

```bash
# Install SQLite resource
vrooli resource sqlite manage install

# Create a new database
vrooli resource sqlite content create myapp

# Execute SQL queries
vrooli resource sqlite content execute myapp "CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT)"
vrooli resource sqlite content execute myapp "INSERT INTO users (name) VALUES ('Alice')"
vrooli resource sqlite content execute myapp "SELECT * FROM users"

# List all databases
vrooli resource sqlite content list

# Backup a database
vrooli resource sqlite content backup myapp
```

## CLI Commands

### Management Commands

```bash
# Install SQLite
vrooli resource sqlite manage install

# Start (no-op for serverless)
vrooli resource sqlite manage start

# Stop (no-op for serverless)
vrooli resource sqlite manage stop

# Restart (no-op for serverless)
vrooli resource sqlite manage restart

# Uninstall and clean up data
vrooli resource sqlite manage uninstall
```

### Content Commands

```bash
# Create a new database
vrooli resource sqlite content create <db_name>

# Execute SQL query
vrooli resource sqlite content execute <db_name> "<sql_query>"

# List all databases
vrooli resource sqlite content list

# Remove a database
vrooli resource sqlite content remove <db_name>

# Backup a database
vrooli resource sqlite content backup <db_name>

# Restore a database from backup
vrooli resource sqlite content restore <db_name> <backup_file>

# Get database info or run SELECT query
vrooli resource sqlite content get <db_name> [query]
```

### Testing Commands

```bash
# Run smoke test (quick validation)
vrooli resource sqlite test smoke

# Run integration tests
vrooli resource sqlite test integration

# Run unit tests
vrooli resource sqlite test unit

# Run all tests
vrooli resource sqlite test all
```

### Information Commands

```bash
# Show help
vrooli resource sqlite help

# Show status
vrooli resource sqlite status

# Show runtime info
vrooli resource sqlite info

# Show logs (no-op for serverless)
vrooli resource sqlite logs
```

## Configuration

Configuration is managed through environment variables. Default values are defined in `config/defaults.sh`.

### Key Settings

| Variable | Default | Description |
|----------|---------|-------------|
| `SQLITE_DATABASE_PATH` | `${VROOLI_DATA}/sqlite/databases` | Where databases are stored |
| `SQLITE_BACKUP_PATH` | `${VROOLI_DATA}/sqlite/backups` | Where backups are stored |
| `SQLITE_JOURNAL_MODE` | `WAL` | Journal mode (WAL recommended) |
| `SQLITE_BUSY_TIMEOUT` | `10000` | Busy timeout in milliseconds |
| `SQLITE_CACHE_SIZE` | `2000` | Cache size in pages |
| `SQLITE_FILE_PERMISSIONS` | `600` | Unix permissions for database files |

## Usage Examples

### Basic CRUD Operations

```bash
# Create database and table
vrooli resource sqlite content create todo
vrooli resource sqlite content execute todo "CREATE TABLE tasks (id INTEGER PRIMARY KEY, title TEXT, done BOOLEAN DEFAULT 0)"

# Insert data
vrooli resource sqlite content execute todo "INSERT INTO tasks (title) VALUES ('Write documentation')"
vrooli resource sqlite content execute todo "INSERT INTO tasks (title) VALUES ('Test application')"

# Query data
vrooli resource sqlite content execute todo "SELECT * FROM tasks WHERE done = 0"

# Update data
vrooli resource sqlite content execute todo "UPDATE tasks SET done = 1 WHERE id = 1"

# Delete data
vrooli resource sqlite content execute todo "DELETE FROM tasks WHERE done = 1"
```

### Working with JSON

SQLite has built-in JSON support:

```bash
# Create table with JSON column
vrooli resource sqlite content execute myapp "CREATE TABLE config (key TEXT PRIMARY KEY, data JSON)"

# Insert JSON data
vrooli resource sqlite content execute myapp "INSERT INTO config VALUES ('app_settings', '{\"theme\":\"dark\",\"lang\":\"en\"}')"

# Query JSON data
vrooli resource sqlite content execute myapp "SELECT json_extract(data, '$.theme') FROM config WHERE key = 'app_settings'"
```

### Full-Text Search

```bash
# Create FTS5 virtual table
vrooli resource sqlite content execute docs "CREATE VIRTUAL TABLE documents USING fts5(title, content)"

# Insert documents
vrooli resource sqlite content execute docs "INSERT INTO documents VALUES ('SQLite Guide', 'SQLite is a lightweight database engine')"

# Search documents
vrooli resource sqlite content execute docs "SELECT * FROM documents WHERE documents MATCH 'database'"
```

## Integration with Scenarios

SQLite is perfect for scenarios that need:
- Local data persistence without server overhead
- Quick prototyping with SQL support
- Configuration storage
- Session management
- Caching layers
- Audit logs
- Development and testing

Example scenario integration:

```bash
# In your scenario's API code
DB_PATH="${VROOLI_DATA}/sqlite/databases/my_scenario.db"
sqlite3 "$DB_PATH" "SELECT * FROM users WHERE active = 1"
```

## Performance Tips

1. **Use WAL Mode**: Already enabled by default for better concurrency
2. **Batch Operations**: Group multiple INSERT/UPDATE statements in transactions
3. **Create Indexes**: Add indexes for frequently queried columns
4. **VACUUM Periodically**: Reclaim space and optimize database file
5. **Monitor Size**: SQLite works best with databases under 1GB

## Troubleshooting

### Database Locked Error
- Ensure WAL mode is enabled (default)
- Check for long-running transactions
- Increase busy timeout if needed

### Permission Denied
- Check file permissions (should be 600)
- Ensure database directory is writable

### Performance Issues
- Create indexes for slow queries
- Run VACUUM to optimize database
- Consider splitting large databases

## Migration to Other Databases

When your scenario outgrows SQLite, migrate to PostgreSQL:

```bash
# Export SQLite to SQL
sqlite3 myapp.db .dump > myapp.sql

# Import to PostgreSQL (after some syntax adjustments)
psql -d myapp -f myapp.sql
```

## Security Considerations

- Databases are stored with 600 permissions (owner read/write only)
- No network exposure (local filesystem only)
- Use parameterized queries to prevent SQL injection
- Consider encryption for sensitive data

## Support

For issues or questions:
- Check the [PRD.md](PRD.md) for requirements and specifications
- Run tests with `vrooli resource sqlite test all`
- Review logs and error messages for debugging