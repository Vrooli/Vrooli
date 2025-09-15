#!/usr/bin/env bash
################################################################################
# SQLite Resource - Core Library
#
# Core functionality for managing SQLite databases
################################################################################

set -euo pipefail

# Validate database name to prevent path traversal
sqlite::validate_name() {
    local name="${1:-}"
    local type="${2:-database}"
    
    # Check for empty name
    if [[ -z "$name" ]]; then
        log::error "Empty $type name provided"
        return 1
    fi
    
    # Check for path traversal attempts
    if [[ "$name" =~ \.\. ]] || [[ "$name" =~ / ]]; then
        log::error "Invalid $type name: contains path characters"
        return 1
    fi
    
    # Check for special characters that could cause issues
    if [[ ! "$name" =~ ^[a-zA-Z0-9_.-]+$ ]]; then
        log::error "Invalid $type name: contains special characters"
        return 1
    fi
    
    # Check reasonable length
    if [[ ${#name} -gt 255 ]]; then
        log::error "Invalid $type name: too long (max 255 characters)"
        return 1
    fi
    
    return 0
}

# Ensure required directories exist
sqlite::ensure_directories() {
    local dirs=(
        "${SQLITE_DATABASE_PATH}"
        "${SQLITE_BACKUP_PATH}"
    )
    
    for dir in "${dirs[@]}"; do
        if [[ ! -d "$dir" ]]; then
            mkdir -p "$dir"
            chmod 755 "$dir"
            log::info "Created directory: $dir"
        fi
    done
}

# Install SQLite if not present
sqlite::install() {
    log::info "Installing SQLite resource..."
    
    # Check if sqlite3 is installed
    if ! command -v sqlite3 &> /dev/null; then
        log::info "SQLite3 not found, installing..."
        
        if [[ "$OSTYPE" == "darwin"* ]]; then
            # macOS
            if command -v brew &> /dev/null; then
                brew install sqlite3
            else
                log::error "Homebrew not found. Please install sqlite3 manually."
                return 1
            fi
        elif [[ -f /etc/debian_version ]]; then
            # Debian/Ubuntu
            sudo apt-get update
            sudo apt-get install -y sqlite3
        elif [[ -f /etc/redhat-release ]]; then
            # RHEL/CentOS/Fedora
            sudo yum install -y sqlite
        else
            log::error "Unsupported operating system. Please install sqlite3 manually."
            return 1
        fi
    fi
    
    # Create required directories
    sqlite::ensure_directories
    
    # Verify installation
    local version
    version=$(sqlite3 --version | awk '{print $1}')
    log::info "SQLite installed successfully: version $version"
    
    return 0
}

# Uninstall (cleanup data only, don't remove sqlite3)
sqlite::uninstall() {
    log::info "Cleaning up SQLite resource data..."
    
    # Confirm before removing data
    if [[ -d "${SQLITE_DATABASE_PATH}" ]]; then
        log::warning "This will remove all SQLite databases and backups"
        read -p "Are you sure? (y/N): " -n 1 -r
        echo
        if [[ $REPLY =~ ^[Yy]$ ]]; then
            rm -rf "${SQLITE_DATABASE_PATH}"
            rm -rf "${SQLITE_BACKUP_PATH}"
            log::info "SQLite data removed"
        else
            log::info "Uninstall cancelled"
            return 1
        fi
    fi
    
    return 0
}

# Start (no-op for serverless, but required by contract)
sqlite::start() {
    log::info "SQLite is serverless - no process to start"
    sqlite::ensure_directories
    return 0
}

# Stop (no-op for serverless, but required by contract)
sqlite::stop() {
    log::info "SQLite is serverless - no process to stop"
    return 0
}

# Restart (no-op for serverless, but required by contract)
sqlite::restart() {
    log::info "SQLite is serverless - no process to restart"
    sqlite::ensure_directories
    return 0
}

# Get status
sqlite::status() {
    echo "SQLite Status"
    echo "============="
    
    # Check if sqlite3 is available
    if command -v sqlite3 &> /dev/null; then
        local version
        version=$(sqlite3 --version | awk '{print $1}')
        echo "Status: Available"
        echo "Version: $version"
    else
        echo "Status: Not installed"
        return 1
    fi
    
    # Show database statistics
    if [[ -d "${SQLITE_DATABASE_PATH}" ]]; then
        local db_count
        db_count=$(find "${SQLITE_DATABASE_PATH}" -name "*.db" -o -name "*.sqlite" -o -name "*.sqlite3" | wc -l)
        echo "Database Path: ${SQLITE_DATABASE_PATH}"
        echo "Number of Databases: $db_count"
        
        if [[ $db_count -gt 0 ]]; then
            echo ""
            echo "Databases:"
            find "${SQLITE_DATABASE_PATH}" -name "*.db" -o -name "*.sqlite" -o -name "*.sqlite3" | while read -r db; do
                local size
                size=$(du -h "$db" | cut -f1)
                echo "  - $(basename "$db") ($size)"
            done
        fi
    fi
    
    return 0
}

# Show logs (no server logs, but we can show recent operations)
sqlite::logs() {
    log::info "SQLite is serverless - no server logs available"
    echo "For query logs, enable logging in your application"
    return 0
}

# Show info from runtime.json
sqlite::info() {
    local runtime_file="${SQLITE_CLI_DIR}/config/runtime.json"
    if [[ -f "$runtime_file" ]]; then
        cat "$runtime_file"
    else
        log::error "Runtime configuration not found: $runtime_file"
        return 1
    fi
}

# Create a new database
sqlite::content::create() {
    local db_name="${1:-}"
    
    if [[ -z "$db_name" ]]; then
        log::error "Database name required"
        echo "Usage: resource-sqlite content create <database_name>"
        return 1
    fi
    
    # Validate database name for security
    if ! sqlite::validate_name "$db_name" "database"; then
        return 1
    fi
    
    # Add extension if not present
    if [[ ! "$db_name" =~ \.(db|sqlite|sqlite3)$ ]]; then
        db_name="${db_name}.db"
    fi
    
    local db_path="${SQLITE_DATABASE_PATH}/${db_name}"
    
    if [[ -f "$db_path" ]]; then
        log::warning "Database already exists: $db_name"
        return 1
    fi
    
    # Create database with initial settings
    sqlite3 "$db_path" <<EOF
PRAGMA journal_mode = ${SQLITE_JOURNAL_MODE};
PRAGMA busy_timeout = ${SQLITE_BUSY_TIMEOUT};
PRAGMA cache_size = ${SQLITE_CACHE_SIZE};
PRAGMA page_size = ${SQLITE_PAGE_SIZE};
PRAGMA synchronous = ${SQLITE_SYNCHRONOUS};
PRAGMA temp_store = ${SQLITE_TEMP_STORE};
PRAGMA mmap_size = ${SQLITE_MMAP_SIZE};
.quit
EOF
    
    # Set proper permissions
    chmod "${SQLITE_FILE_PERMISSIONS}" "$db_path"
    
    log::info "Database created: $db_name"
    echo "Database path: $db_path"
    return 0
}

# Execute SQL query
sqlite::content::execute() {
    local db_name="${1:-}"
    local query="${2:-}"
    
    if [[ -z "$db_name" ]] || [[ -z "$query" ]]; then
        log::error "Database name and query required"
        echo "Usage: resource-sqlite content execute <database_name> <query>"
        return 1
    fi
    
    # Validate database name for security
    if ! sqlite::validate_name "$db_name" "database"; then
        return 1
    fi
    
    # Add extension if not present
    if [[ ! "$db_name" =~ \.(db|sqlite|sqlite3)$ ]]; then
        db_name="${db_name}.db"
    fi
    
    local db_path="${SQLITE_DATABASE_PATH}/${db_name}"
    
    if [[ ! -f "$db_path" ]]; then
        log::error "Database not found: $db_name"
        return 1
    fi
    
    # Execute query with timeout
    timeout "${SQLITE_CLI_TIMEOUT}" sqlite3 "$db_path" "$query"
    return $?
}

# List databases
sqlite::content::list() {
    echo "SQLite Databases"
    echo "==============="
    
    if [[ ! -d "${SQLITE_DATABASE_PATH}" ]]; then
        echo "No databases found"
        return 0
    fi
    
    local databases
    databases=$(find "${SQLITE_DATABASE_PATH}" \( -name "*.db" -o -name "*.sqlite" -o -name "*.sqlite3" \) -type f 2>/dev/null)
    
    if [[ -z "$databases" ]]; then
        echo "No databases found"
    else
        echo "$databases" | while read -r db; do
            local name size modified
            name=$(basename "$db")
            size=$(du -h "$db" | cut -f1)
            modified=$(stat -c %y "$db" 2>/dev/null || stat -f %Sm "$db" 2>/dev/null)
            echo "$name ($size) - Modified: $modified"
        done
    fi
    
    return 0
}

# Remove a database
sqlite::content::remove() {
    local db_name="${1:-}"
    
    if [[ -z "$db_name" ]]; then
        log::error "Database name required"
        echo "Usage: resource-sqlite content remove <database_name>"
        return 1
    fi
    
    # Add extension if not present
    if [[ ! "$db_name" =~ \.(db|sqlite|sqlite3)$ ]]; then
        db_name="${db_name}.db"
    fi
    
    local db_path="${SQLITE_DATABASE_PATH}/${db_name}"
    
    if [[ ! -f "$db_path" ]]; then
        log::error "Database not found: $db_name"
        return 1
    fi
    
    # Confirm deletion
    read -p "Remove database '$db_name'? This cannot be undone. (y/N): " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        rm -f "$db_path"
        rm -f "${db_path}-wal"  # Remove WAL file if exists
        rm -f "${db_path}-shm"  # Remove shared memory file if exists
        log::info "Database removed: $db_name"
    else
        log::info "Removal cancelled"
        return 1
    fi
    
    return 0
}

# Backup a database
sqlite::content::backup() {
    local db_name="${1:-}"
    
    if [[ -z "$db_name" ]]; then
        log::error "Database name required"
        echo "Usage: resource-sqlite content backup <database_name>"
        return 1
    fi
    
    # Validate database name for security
    if ! sqlite::validate_name "$db_name" "database"; then
        return 1
    fi
    
    # Add extension if not present
    if [[ ! "$db_name" =~ \.(db|sqlite|sqlite3)$ ]]; then
        db_name="${db_name}.db"
    fi
    
    local db_path="${SQLITE_DATABASE_PATH}/${db_name}"
    
    if [[ ! -f "$db_path" ]]; then
        log::error "Database not found: $db_name"
        return 1
    fi
    
    # Create backup with timestamp
    local timestamp
    timestamp=$(date +%Y%m%d_%H%M%S)
    local backup_name="${db_name%.db}_${timestamp}.db"
    local backup_path="${SQLITE_BACKUP_PATH}/${backup_name}"
    
    # Use SQLite backup command for consistency
    sqlite3 "$db_path" ".backup '$backup_path'"
    
    # Set proper permissions
    chmod "${SQLITE_FILE_PERMISSIONS}" "$backup_path"
    
    log::info "Database backed up: $backup_name"
    echo "Backup path: $backup_path"
    
    # Clean old backups
    find "${SQLITE_BACKUP_PATH}" -name "*.db" -mtime +${SQLITE_BACKUP_RETENTION_DAYS} -delete
    
    return 0
}

# Restore a database from backup
sqlite::content::restore() {
    local db_name="${1:-}"
    local backup_file="${2:-}"
    
    if [[ -z "$db_name" ]] || [[ -z "$backup_file" ]]; then
        log::error "Database name and backup file required"
        echo "Usage: resource-sqlite content restore <database_name> <backup_file>"
        return 1
    fi
    
    # Validate database name for security
    if ! sqlite::validate_name "$db_name" "database"; then
        return 1
    fi
    
    # Validate backup file name
    if ! sqlite::validate_name "$backup_file" "backup file"; then
        return 1
    fi
    
    # Add extension if not present
    if [[ ! "$db_name" =~ \.(db|sqlite|sqlite3)$ ]]; then
        db_name="${db_name}.db"
    fi
    
    local db_path="${SQLITE_DATABASE_PATH}/${db_name}"
    local backup_path
    
    # Check if backup_file is absolute or relative path
    if [[ "$backup_file" == /* ]]; then
        backup_path="$backup_file"
    else
        # Look in backup directory
        backup_path="${SQLITE_BACKUP_PATH}/${backup_file}"
    fi
    
    if [[ ! -f "$backup_path" ]]; then
        log::error "Backup file not found: $backup_file"
        return 1
    fi
    
    # If database exists, confirm overwrite
    if [[ -f "$db_path" ]]; then
        read -p "Database '$db_name' exists. Overwrite? (y/N): " -n 1 -r
        echo
        if [[ ! $REPLY =~ ^[Yy]$ ]]; then
            log::info "Restore cancelled"
            return 1
        fi
        
        # Create backup of current database before overwriting
        local timestamp
        timestamp=$(date +%Y%m%d_%H%M%S)
        local safety_backup="${db_path%.db}_before_restore_${timestamp}.db"
        sqlite3 "$db_path" ".backup '$safety_backup'"
        log::info "Current database backed up to: $(basename "$safety_backup")"
    fi
    
    # Restore database using SQLite restore command
    sqlite3 "$db_path" ".restore '$backup_path'"
    
    # Set proper permissions
    chmod "${SQLITE_FILE_PERMISSIONS}" "$db_path"
    
    log::info "Database restored: $db_name from $backup_file"
    echo "Database path: $db_path"
    
    return 0
}

# Get database info or query result
sqlite::content::get() {
    local db_name="${1:-}"
    local query="${2:-}"
    
    if [[ -z "$db_name" ]]; then
        log::error "Database name required"
        echo "Usage: resource-sqlite content get <database_name> [query]"
        echo "  Without query: shows database schema and statistics"
        echo "  With query: executes SELECT query and returns results"
        return 1
    fi
    
    # Add extension if not present
    if [[ ! "$db_name" =~ \.(db|sqlite|sqlite3)$ ]]; then
        db_name="${db_name}.db"
    fi
    
    local db_path="${SQLITE_DATABASE_PATH}/${db_name}"
    
    if [[ ! -f "$db_path" ]]; then
        log::error "Database not found: $db_name"
        return 1
    fi
    
    if [[ -z "$query" ]]; then
        # Show database info
        echo "Database: $db_name"
        echo "================"
        
        # Get file size
        local size
        size=$(du -h "$db_path" | cut -f1)
        echo "Size: $size"
        
        # Get page count and size
        local page_count page_size
        page_count=$(sqlite3 "$db_path" "PRAGMA page_count;")
        page_size=$(sqlite3 "$db_path" "PRAGMA page_size;")
        echo "Pages: $page_count (${page_size} bytes each)"
        
        # Get journal mode
        local journal_mode
        journal_mode=$(sqlite3 "$db_path" "PRAGMA journal_mode;")
        echo "Journal Mode: $journal_mode"
        
        # Get schema
        echo ""
        echo "Schema:"
        echo "-------"
        sqlite3 "$db_path" ".schema"
        
        # Get table statistics
        echo ""
        echo "Tables:"
        echo "-------"
        sqlite3 "$db_path" "SELECT name, sql FROM sqlite_master WHERE type='table' ORDER BY name;" | while IFS='|' read -r name sql; do
            local count
            count=$(sqlite3 "$db_path" "SELECT COUNT(*) FROM $name;")
            echo "  $name: $count rows"
        done
    else
        # Execute SELECT query
        if [[ ! "$query" =~ ^[[:space:]]*SELECT ]]; then
            log::error "Only SELECT queries are allowed with 'get' command"
            echo "Use 'execute' command for other SQL operations"
            return 1
        fi
        
        # Execute query with column headers
        sqlite3 -header -column "$db_path" "$query"
    fi
    
    return 0
}

################################################################################
# Migration Support Functions
################################################################################

# Initialize migration tracking for a database
sqlite::migrate::init() {
    local db_name="${1:-}"
    
    if [[ -z "$db_name" ]]; then
        log::error "Database name required"
        echo "Usage: resource-sqlite migrate init <database_name>"
        return 1
    fi
    
    # Add extension if not present
    if [[ ! "$db_name" =~ \.(db|sqlite|sqlite3)$ ]]; then
        db_name="${db_name}.db"
    fi
    
    local db_path="${SQLITE_DATABASE_PATH}/${db_name}"
    
    if [[ ! -f "$db_path" ]]; then
        log::error "Database not found: $db_name"
        return 1
    fi
    
    # Create migration tracking table
    sqlite3 "$db_path" <<EOF
CREATE TABLE IF NOT EXISTS schema_migrations (
    version TEXT PRIMARY KEY,
    applied_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    checksum TEXT,
    description TEXT
);

CREATE INDEX IF NOT EXISTS idx_migrations_applied_at 
ON schema_migrations(applied_at);
EOF
    
    log::info "Migration tracking initialized for: $db_name"
    return 0
}

# Create a new migration file
sqlite::migrate::create() {
    local name="${1:-}"
    
    if [[ -z "$name" ]]; then
        log::error "Migration name required"
        echo "Usage: resource-sqlite migrate create <name>"
        return 1
    fi
    
    # Create migrations directory if it doesn't exist
    local migrations_dir="${SQLITE_DATABASE_PATH}/migrations"
    mkdir -p "$migrations_dir"
    
    # Generate timestamp-based version
    local version
    version=$(date +%Y%m%d%H%M%S)
    
    # Clean migration name (replace spaces with underscores)
    name=$(echo "$name" | tr ' ' '_' | tr '[:upper:]' '[:lower:]')
    
    # Create migration file
    local migration_file="${migrations_dir}/${version}_${name}.sql"
    
    cat > "$migration_file" <<EOF
-- Migration: ${version}_${name}
-- Created: $(date)
-- Description: ${name}

-- Up Migration
BEGIN TRANSACTION;

-- Add your schema changes here
-- Example:
-- CREATE TABLE example (
--     id INTEGER PRIMARY KEY AUTOINCREMENT,
--     name TEXT NOT NULL,
--     created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
-- );

COMMIT;

-- Down Migration (for rollback)
-- BEGIN TRANSACTION;
-- DROP TABLE IF EXISTS example;
-- COMMIT;
EOF
    
    chmod 644 "$migration_file"
    
    log::info "Migration created: ${version}_${name}.sql"
    echo "Edit migration file: $migration_file"
    return 0
}

# Apply pending migrations
sqlite::migrate::up() {
    local db_name="${1:-}"
    local target_version="${2:-}"
    
    if [[ -z "$db_name" ]]; then
        log::error "Database name required"
        echo "Usage: resource-sqlite migrate up <database_name> [target_version]"
        return 1
    fi
    
    # Add extension if not present
    if [[ ! "$db_name" =~ \.(db|sqlite|sqlite3)$ ]]; then
        db_name="${db_name}.db"
    fi
    
    local db_path="${SQLITE_DATABASE_PATH}/${db_name}"
    
    if [[ ! -f "$db_path" ]]; then
        log::error "Database not found: $db_name"
        return 1
    fi
    
    # Check if migration table exists
    local has_migrations_table
    has_migrations_table=$(sqlite3 "$db_path" "SELECT name FROM sqlite_master WHERE type='table' AND name='schema_migrations';")
    
    if [[ -z "$has_migrations_table" ]]; then
        log::info "Initializing migration tracking..."
        sqlite::migrate::init "$db_name" || return 1
    fi
    
    # Get list of migration files
    local migrations_dir="${SQLITE_DATABASE_PATH}/migrations"
    
    if [[ ! -d "$migrations_dir" ]]; then
        log::info "No migrations directory found"
        return 0
    fi
    
    # Get applied migrations
    local applied_versions
    applied_versions=$(sqlite3 "$db_path" "SELECT version FROM schema_migrations ORDER BY version;")
    
    # Find and apply pending migrations
    local applied_count=0
    for migration_file in "$migrations_dir"/*.sql; do
        [[ ! -f "$migration_file" ]] && continue
        
        local filename
        filename=$(basename "$migration_file")
        local version="${filename%%_*}"
        
        # Skip if already applied
        if echo "$applied_versions" | grep -q "^${version}$"; then
            continue
        fi
        
        # Skip if past target version
        if [[ -n "$target_version" ]] && [[ "$version" > "$target_version" ]]; then
            continue
        fi
        
        log::info "Applying migration: $filename"
        
        # Calculate checksum
        local checksum
        checksum=$(sha256sum "$migration_file" | cut -d' ' -f1)
        
        # Apply migration
        if sqlite3 "$db_path" < "$migration_file"; then
            # Record migration
            sqlite3 "$db_path" "INSERT INTO schema_migrations (version, checksum, description) VALUES ('$version', '$checksum', '${filename#*_}');"
            log::info "✓ Applied: $filename"
            ((applied_count++))
        else
            log::error "Failed to apply migration: $filename"
            return 1
        fi
    done
    
    if [[ $applied_count -eq 0 ]]; then
        log::info "No pending migrations"
    else
        log::info "Applied $applied_count migration(s)"
    fi
    
    return 0
}

# Show migration status
sqlite::migrate::status() {
    local db_name="${1:-}"
    
    if [[ -z "$db_name" ]]; then
        log::error "Database name required"
        echo "Usage: resource-sqlite migrate status <database_name>"
        return 1
    fi
    
    # Add extension if not present
    if [[ ! "$db_name" =~ \.(db|sqlite|sqlite3)$ ]]; then
        db_name="${db_name}.db"
    fi
    
    local db_path="${SQLITE_DATABASE_PATH}/${db_name}"
    
    if [[ ! -f "$db_path" ]]; then
        log::error "Database not found: $db_name"
        return 1
    fi
    
    echo "Migration Status for: $db_name"
    echo "================================"
    
    # Check if migration table exists
    local has_migrations_table
    has_migrations_table=$(sqlite3 "$db_path" "SELECT name FROM sqlite_master WHERE type='table' AND name='schema_migrations';")
    
    if [[ -z "$has_migrations_table" ]]; then
        echo "Migration tracking not initialized"
        echo "Run: resource-sqlite migrate init $db_name"
        return 0
    fi
    
    # Get applied migrations
    echo ""
    echo "Applied Migrations:"
    sqlite3 -header -column "$db_path" "SELECT version, applied_at, description FROM schema_migrations ORDER BY version DESC LIMIT 10;"
    
    # Check for pending migrations
    local migrations_dir="${SQLITE_DATABASE_PATH}/migrations"
    
    if [[ -d "$migrations_dir" ]]; then
        echo ""
        echo "Pending Migrations:"
        local pending_count=0
        
        for migration_file in "$migrations_dir"/*.sql; do
            [[ ! -f "$migration_file" ]] && continue
            
            local filename
            filename=$(basename "$migration_file")
            local version="${filename%%_*}"
            
            # Check if already applied
            local is_applied
            is_applied=$(sqlite3 "$db_path" "SELECT version FROM schema_migrations WHERE version='$version';")
            
            if [[ -z "$is_applied" ]]; then
                echo "  - $filename"
                ((pending_count++))
            fi
        done
        
        if [[ $pending_count -eq 0 ]]; then
            echo "  None"
        fi
    fi
    
    return 0
}

################################################################################
# Query Builder Functions
################################################################################

# Build and execute SELECT query with conditions
sqlite::query::select() {
    local db_name="${1:-}"
    local table="${2:-}"
    shift 2
    
    if [[ -z "$db_name" ]] || [[ -z "$table" ]]; then
        log::error "Database name and table required"
        echo "Usage: resource-sqlite query select <database> <table> [options]"
        echo "Options:"
        echo "  --columns <cols>    Columns to select (default: *)"
        echo "  --where <condition> WHERE clause"
        echo "  --order <column>    ORDER BY clause"
        echo "  --limit <n>         LIMIT clause"
        echo "  --join <table> <on> JOIN clause"
        return 1
    fi
    
    # Validate database and table names
    if ! sqlite::validate_name "$db_name" "database"; then
        return 1
    fi
    if ! sqlite::validate_name "$table" "table"; then
        return 1
    fi
    
    # Add extension if not present
    if [[ ! "$db_name" =~ \.(db|sqlite|sqlite3)$ ]]; then
        db_name="${db_name}.db"
    fi
    
    local db_path="${SQLITE_DATABASE_PATH}/${db_name}"
    
    if [[ ! -f "$db_path" ]]; then
        log::error "Database not found: $db_name"
        return 1
    fi
    
    # Parse options
    local columns="*"
    local where_clause=""
    local order_clause=""
    local limit_clause=""
    local join_clause=""
    
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --columns)
                columns="$2"
                shift 2
                ;;
            --where)
                where_clause="WHERE $2"
                shift 2
                ;;
            --order)
                order_clause="ORDER BY $2"
                shift 2
                ;;
            --limit)
                limit_clause="LIMIT $2"
                shift 2
                ;;
            --join)
                join_clause="JOIN $2 ON $3"
                shift 3
                ;;
            *)
                shift
                ;;
        esac
    done
    
    # Build query
    local query="SELECT ${columns} FROM ${table}"
    [[ -n "$join_clause" ]] && query="${query} ${join_clause}"
    [[ -n "$where_clause" ]] && query="${query} ${where_clause}"
    [[ -n "$order_clause" ]] && query="${query} ${order_clause}"
    [[ -n "$limit_clause" ]] && query="${query} ${limit_clause}"
    
    # Execute query
    sqlite3 -header -column "$db_path" "$query"
    return $?
}

# Insert data with automatic value escaping
sqlite::query::insert() {
    local db_name="${1:-}"
    local table="${2:-}"
    shift 2
    
    if [[ -z "$db_name" ]] || [[ -z "$table" ]]; then
        log::error "Database name and table required"
        echo "Usage: resource-sqlite query insert <database> <table> <column=value> ..."
        return 1
    fi
    
    # Validate database and table names
    if ! sqlite::validate_name "$db_name" "database"; then
        return 1
    fi
    if ! sqlite::validate_name "$table" "table"; then
        return 1
    fi
    
    # Add extension if not present
    if [[ ! "$db_name" =~ \.(db|sqlite|sqlite3)$ ]]; then
        db_name="${db_name}.db"
    fi
    
    local db_path="${SQLITE_DATABASE_PATH}/${db_name}"
    
    if [[ ! -f "$db_path" ]]; then
        log::error "Database not found: $db_name"
        return 1
    fi
    
    # Parse column=value pairs
    local columns=""
    local values=""
    
    while [[ $# -gt 0 ]]; do
        if [[ "$1" =~ ^([^=]+)=(.*)$ ]]; then
            local col="${BASH_REMATCH[1]}"
            local val="${BASH_REMATCH[2]}"
            
            if [[ -n "$columns" ]]; then
                columns="${columns}, ${col}"
                values="${values}, '${val//\'/\'\'}'"
            else
                columns="${col}"
                values="'${val//\'/\'\'}'"
            fi
        fi
        shift
    done
    
    if [[ -z "$columns" ]]; then
        log::error "No column=value pairs provided"
        return 1
    fi
    
    # Build and execute INSERT query with last_insert_rowid() in the same session
    local query="INSERT INTO ${table} (${columns}) VALUES (${values}); SELECT last_insert_rowid();"
    local last_id
    last_id=$(sqlite3 "$db_path" "$query")
    
    if [[ $? -eq 0 ]]; then
        log::info "Row inserted into ${table}"
        echo "Last inserted ID: $last_id"
    fi
    
    return $?
}

# Update data with conditions
sqlite::query::update() {
    local db_name="${1:-}"
    local table="${2:-}"
    local where="${3:-}"
    shift 3
    
    if [[ -z "$db_name" ]] || [[ -z "$table" ]] || [[ -z "$where" ]]; then
        log::error "Database name, table, and WHERE clause required"
        echo "Usage: resource-sqlite query update <database> <table> <where_clause> <column=value> ..."
        return 1
    fi
    
    # Validate database and table names
    if ! sqlite::validate_name "$db_name" "database"; then
        return 1
    fi
    if ! sqlite::validate_name "$table" "table"; then
        return 1
    fi
    
    # Add extension if not present
    if [[ ! "$db_name" =~ \.(db|sqlite|sqlite3)$ ]]; then
        db_name="${db_name}.db"
    fi
    
    local db_path="${SQLITE_DATABASE_PATH}/${db_name}"
    
    if [[ ! -f "$db_path" ]]; then
        log::error "Database not found: $db_name"
        return 1
    fi
    
    # Parse column=value pairs
    local set_clause=""
    
    while [[ $# -gt 0 ]]; do
        if [[ "$1" =~ ^([^=]+)=(.*)$ ]]; then
            local col="${BASH_REMATCH[1]}"
            local val="${BASH_REMATCH[2]}"
            
            if [[ -n "$set_clause" ]]; then
                set_clause="${set_clause}, ${col} = '${val//\'/\'\'}'"
            else
                set_clause="${col} = '${val//\'/\'\'}'"
            fi
        fi
        shift
    done
    
    if [[ -z "$set_clause" ]]; then
        log::error "No column=value pairs provided"
        return 1
    fi
    
    # Build and execute UPDATE query with changes() in the same session
    local query="UPDATE ${table} SET ${set_clause} WHERE ${where}; SELECT changes();"
    local affected
    affected=$(sqlite3 "$db_path" "$query")
    
    if [[ $? -eq 0 ]]; then
        log::info "Updated ${affected} row(s) in ${table}"
    fi
    
    return $?
}

################################################################################
# Performance Monitoring Functions
################################################################################

# Enable query statistics
sqlite::stats::enable() {
    local db_name="${1:-}"
    
    if [[ -z "$db_name" ]]; then
        log::error "Database name required"
        echo "Usage: resource-sqlite stats enable <database_name>"
        return 1
    fi
    
    # Add extension if not present
    if [[ ! "$db_name" =~ \.(db|sqlite|sqlite3)$ ]]; then
        db_name="${db_name}.db"
    fi
    
    local db_path="${SQLITE_DATABASE_PATH}/${db_name}"
    
    if [[ ! -f "$db_path" ]]; then
        log::error "Database not found: $db_name"
        return 1
    fi
    
    # Create statistics table
    sqlite3 "$db_path" <<EOF
CREATE TABLE IF NOT EXISTS query_stats (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    query_hash TEXT,
    query_text TEXT,
    execution_time_ms REAL,
    rows_examined INTEGER,
    rows_returned INTEGER,
    timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_stats_timestamp 
ON query_stats(timestamp);

CREATE INDEX IF NOT EXISTS idx_stats_query_hash 
ON query_stats(query_hash);
EOF
    
    log::info "Query statistics enabled for: $db_name"
    return 0
}

# Show query statistics
sqlite::stats::show() {
    local db_name="${1:-}"
    local limit="${2:-10}"
    
    if [[ -z "$db_name" ]]; then
        log::error "Database name required"
        echo "Usage: resource-sqlite stats show <database_name> [limit]"
        return 1
    fi
    
    # Add extension if not present
    if [[ ! "$db_name" =~ \.(db|sqlite|sqlite3)$ ]]; then
        db_name="${db_name}.db"
    fi
    
    local db_path="${SQLITE_DATABASE_PATH}/${db_name}"
    
    if [[ ! -f "$db_path" ]]; then
        log::error "Database not found: $db_name"
        return 1
    fi
    
    echo "Query Statistics for: $db_name"
    echo "================================"
    
    # Check if stats table exists
    local has_stats_table
    has_stats_table=$(sqlite3 "$db_path" "SELECT name FROM sqlite_master WHERE type='table' AND name='query_stats';")
    
    if [[ -z "$has_stats_table" ]]; then
        echo "Statistics not enabled"
        echo "Run: resource-sqlite stats enable $db_name"
        return 0
    fi
    
    # Show slow queries
    echo ""
    echo "Slowest Queries (last ${limit}):"
    sqlite3 -header -column "$db_path" "
        SELECT 
            substr(query_text, 1, 60) as query,
            printf('%.2f', execution_time_ms) as time_ms,
            rows_examined,
            rows_returned
        FROM query_stats 
        ORDER BY execution_time_ms DESC 
        LIMIT ${limit};
    "
    
    # Show query frequency
    echo ""
    echo "Most Frequent Queries:"
    sqlite3 -header -column "$db_path" "
        SELECT 
            substr(query_text, 1, 60) as query,
            COUNT(*) as count,
            printf('%.2f', AVG(execution_time_ms)) as avg_time_ms
        FROM query_stats 
        GROUP BY query_hash 
        ORDER BY count DESC 
        LIMIT ${limit};
    "
    
    # Show general statistics
    echo ""
    echo "General Statistics:"
    sqlite3 -column "$db_path" "
        SELECT 
            'Total Queries: ' || COUNT(*) as metric
        FROM query_stats
        UNION ALL
        SELECT 
            'Avg Execution Time: ' || printf('%.2f', AVG(execution_time_ms)) || ' ms'
        FROM query_stats
        UNION ALL
        SELECT 
            'Total Time: ' || printf('%.2f', SUM(execution_time_ms)/1000) || ' seconds'
        FROM query_stats;
    "
    
    return 0
}

# Analyze database for optimization opportunities
sqlite::stats::analyze() {
    local db_name="${1:-}"
    
    if [[ -z "$db_name" ]]; then
        log::error "Database name required"
        echo "Usage: resource-sqlite stats analyze <database_name>"
        return 1
    fi
    
    # Add extension if not present
    if [[ ! "$db_name" =~ \.(db|sqlite|sqlite3)$ ]]; then
        db_name="${db_name}.db"
    fi
    
    local db_path="${SQLITE_DATABASE_PATH}/${db_name}"
    
    if [[ ! -f "$db_path" ]]; then
        log::error "Database not found: $db_name"
        return 1
    fi
    
    echo "Database Analysis for: $db_name"
    echo "================================"
    
    # Run ANALYZE to update internal statistics
    sqlite3 "$db_path" "ANALYZE;"
    
    # Check for missing indexes
    echo ""
    echo "Tables without indexes:"
    sqlite3 -column "$db_path" "
        SELECT m.name as table_name
        FROM sqlite_master m
        WHERE m.type = 'table'
        AND m.name NOT LIKE 'sqlite_%'
        AND NOT EXISTS (
            SELECT 1 FROM sqlite_master i
            WHERE i.type = 'index'
            AND i.tbl_name = m.name
        );
    "
    
    # Check table sizes
    echo ""
    echo "Table sizes:"
    for table in $(sqlite3 "$db_path" "SELECT name FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%';"); do
        local count
        count=$(sqlite3 "$db_path" "SELECT COUNT(*) FROM $table;")
        echo "  $table: $count rows"
    done
    
    # Check database integrity
    echo ""
    echo "Integrity check:"
    local integrity
    integrity=$(sqlite3 "$db_path" "PRAGMA integrity_check;")
    if [[ "$integrity" == "ok" ]]; then
        echo "  ✓ Database integrity verified"
    else
        echo "  ⚠ Issues found: $integrity"
    fi
    
    # Check for auto-vacuum
    echo ""
    echo "Database settings:"
    local auto_vacuum
    auto_vacuum=$(sqlite3 "$db_path" "PRAGMA auto_vacuum;")
    case "$auto_vacuum" in
        0) echo "  Auto-vacuum: NONE (manual VACUUM required)" ;;
        1) echo "  Auto-vacuum: FULL (automatic)" ;;
        2) echo "  Auto-vacuum: INCREMENTAL" ;;
    esac
    
    # Check page utilization
    local page_count page_size freelist_count
    page_count=$(sqlite3 "$db_path" "PRAGMA page_count;")
    page_size=$(sqlite3 "$db_path" "PRAGMA page_size;")
    freelist_count=$(sqlite3 "$db_path" "PRAGMA freelist_count;")
    
    echo "  Page count: $page_count"
    echo "  Page size: $page_size bytes"
    echo "  Free pages: $freelist_count"
    
    if [[ $freelist_count -gt $((page_count / 10)) ]]; then
        echo "  ⚠ High fragmentation detected. Consider running VACUUM."
    fi
    
    return 0
}