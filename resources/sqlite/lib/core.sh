#!/usr/bin/env bash
################################################################################
# SQLite Resource - Core Library
#
# Core functionality for managing SQLite databases
################################################################################

set -euo pipefail

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