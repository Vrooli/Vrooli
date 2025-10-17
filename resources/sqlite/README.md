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
- **Database Encryption**: AES-256 encryption using OpenSSL
- **Batch Operations**: High-performance batch SQL execution
- **CSV Import/Export**: Easy data migration and integration
- **Migration Support**: Schema versioning and migration tracking
- **Query Builder**: Safe query construction with automatic escaping
- **Performance Monitoring**: Query statistics and optimization analysis
- **Database Replication**: Basic replication to other SQLite instances with automatic sync

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

# Encrypt a database with a password
vrooli resource sqlite content encrypt <db_name> [password]

# Decrypt an encrypted database
vrooli resource sqlite content decrypt <db_name> [password]

# Execute batch SQL operations from file or stdin
vrooli resource sqlite content batch <db_name> [sql_file]

# Import CSV data into a table
vrooli resource sqlite content import_csv <db_name> <table_name> <csv_file> [has_header]

# Export table data to CSV
vrooli resource sqlite content export_csv <db_name> <table_name> [output_file]
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
| `SQLITE_REPLICATION_PATH` | `${VROOLI_DATA}/sqlite/replicas` | Where replicas are stored |
| `SQLITE_JOURNAL_MODE` | `WAL` | Journal mode (WAL recommended) |
| `SQLITE_BUSY_TIMEOUT` | `10000` | Busy timeout in milliseconds |
| `SQLITE_CACHE_SIZE` | `2000` | Cache size in pages |
| `SQLITE_FILE_PERMISSIONS` | `600` | Unix permissions for database files |
| `SQLITE_REPLICATION_ENABLED` | `true` | Enable replication features |
| `SQLITE_REPLICATION_SYNC_INTERVAL` | `300` | Default sync interval (seconds) |
| `SQLITE_REPLICATION_MONITOR_ENABLED` | `false` | Auto-monitoring off by default |

## Advanced Features

### Database Migrations

```bash
# Initialize migration tracking
vrooli resource sqlite migrate init myapp

# Create a new migration
vrooli resource sqlite migrate create "add users table"

# Apply pending migrations
vrooli resource sqlite migrate up myapp

# Check migration status
vrooli resource sqlite migrate status myapp
```

### Query Builder

```bash
# SELECT with conditions
vrooli resource sqlite query select myapp users --where "active=1" --order "created_at DESC" --limit 10

# INSERT with automatic escaping
vrooli resource sqlite query insert myapp users "name=John Doe" "email=john@example.com"

# UPDATE with conditions
vrooli resource sqlite query update myapp users "id=1" "name=Jane Doe" "updated_at=CURRENT_TIMESTAMP"
```

### Performance Monitoring

```bash
# Enable query statistics
vrooli resource sqlite stats enable myapp

# Show performance statistics
vrooli resource sqlite stats show myapp

# Analyze database for optimization
vrooli resource sqlite stats analyze myapp
```

### Database Encryption

```bash
# Encrypt an existing database
vrooli resource sqlite content encrypt myapp
# Enter password when prompted (or provide as argument)

# Decrypt a database for normal use
vrooli resource sqlite content decrypt myapp
# Enter password when prompted
```

### Batch Operations

```bash
# Execute batch SQL from file
vrooli resource sqlite content batch myapp /path/to/script.sql

# Execute batch SQL from stdin
cat script.sql | vrooli resource sqlite content batch myapp

# Batch operations are wrapped in transactions for better performance
```

### CSV Import/Export

```bash
# Export table to CSV
vrooli resource sqlite content export_csv myapp users /tmp/users.csv

# Import CSV into table
vrooli resource sqlite content import_csv myapp users /tmp/data.csv true
# Last parameter indicates if CSV has header row
```

### Database Replication

```bash
# Add a replica for backup/high availability
vrooli resource sqlite replicate add --database myapp.db --target /backup/myapp_replica.db --interval 300

# List configured replicas
vrooli resource sqlite replicate list

# Manually sync database to replicas
vrooli resource sqlite replicate sync --database myapp.db

# Verify replica consistency
vrooli resource sqlite replicate verify --database myapp.db

# Enable/disable a replica
vrooli resource sqlite replicate toggle --database myapp.db --target /backup/myapp_replica.db --disable

# Remove a replica
vrooli resource sqlite replicate remove --database myapp.db --target /backup/myapp_replica.db

# Start automatic replication monitor
vrooli resource sqlite replicate monitor --interval 60
# Monitor will sync all configured replicas every 60 seconds
```

### Web UI for Batch Operations

The SQLite resource includes a web interface for batch operations, CSV import/export, and database statistics:

```bash
# Start the web UI server
vrooli resource sqlite webui start
# Access at: http://127.0.0.1:8297/

# Check web UI status
vrooli resource sqlite webui status

# Stop the web UI
vrooli resource sqlite webui stop

# Restart the web UI
vrooli resource sqlite webui restart
```

The web UI provides:
- **Database Explorer**: Browse and select databases
- **Batch SQL Executor**: Run multiple SQL commands with transaction support
- **CSV Import/Export**: Upload CSV files or export tables to CSV
- **Statistics Viewer**: Analyze database performance and optimization recommendations
- **Table Browser**: View table schemas and row counts

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

- **Input Validation**: All database and table names are validated to prevent path traversal attacks
- **File Permissions**: Databases are stored with 600 permissions (owner read/write only)
- **No Network Exposure**: Local filesystem only, no remote access
- **SQL Injection Protection**: Query builders automatically escape values
- **Name Restrictions**: Only alphanumeric characters, underscores, dots, and hyphens allowed in names
- **Path Traversal Prevention**: Names containing `..` or `/` are rejected
- **Encryption Available**: Use AES-256 encryption for sensitive databases

## Support

For issues or questions:
- Check the [PRD.md](PRD.md) for requirements and specifications
- Run tests with `vrooli resource sqlite test all`
- Review logs and error messages for debugging