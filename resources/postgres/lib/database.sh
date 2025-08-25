#!/usr/bin/env bash
# PostgreSQL Database Operations
# Functions for database management, schema operations, and data manipulation

#######################################
# Execute SQL command on PostgreSQL instance
# Arguments:
#   $1 - instance name
#   $2 - SQL command
#   $3 - database name (optional, default: $POSTGRES_DEFAULT_DB)
# Returns: 0 on success, 1 on failure
#######################################
postgres::database::execute() {
    local instance_name="$1"
    local sql_command="$2"
    local database="${3:-$POSTGRES_DEFAULT_DB}"
    
    if [[ -z "$instance_name" || -z "$sql_command" ]]; then
        log::error "Instance name and SQL command are required"
        return 1
    fi
    
    if ! postgres::common::container_exists "$instance_name"; then
        log::error "${MSG_INSTANCE_NOT_FOUND}: $instance_name"
        return 1
    fi
    
    if ! postgres::common::is_running "$instance_name"; then
        log::error "Instance '$instance_name' is not running"
        return 1
    fi
    
    local container_name="${POSTGRES_CONTAINER_PREFIX}-${instance_name}"
    
    # Execute SQL command in container
    local output
    if output=$(docker exec -i "$container_name" psql \
        -U "$POSTGRES_DEFAULT_USER" \
        -d "$database" \
        -c "$sql_command" 2>&1); then
        echo "$output"
        return 0
    else
        local exit_code=$?
        log::error "Failed to execute SQL command on instance '$instance_name'"
        log::error "Exit code: $exit_code"
        log::error "Output: $output"
        return 1
    fi
}

#######################################
# Execute SQL file on PostgreSQL instance
# Arguments:
#   $1 - instance name
#   $2 - SQL file path
#   $3 - database name (optional, default: $POSTGRES_DEFAULT_DB)
# Returns: 0 on success, 1 on failure
#######################################
postgres::database::execute_file() {
    local instance_name="$1"
    local sql_file="$2"
    local database="${3:-$POSTGRES_DEFAULT_DB}"
    
    if [[ -z "$instance_name" || -z "$sql_file" ]]; then
        log::error "Instance name and SQL file are required"
        return 1
    fi
    
    if [[ ! -f "$sql_file" ]]; then
        log::error "SQL file not found: $sql_file"
        return 1
    fi
    
    if ! postgres::common::container_exists "$instance_name"; then
        log::error "${MSG_INSTANCE_NOT_FOUND}: $instance_name"
        return 1
    fi
    
    if ! postgres::common::is_running "$instance_name"; then
        log::error "Instance '$instance_name' is not running"
        return 1
    fi
    
    local container_name="${POSTGRES_CONTAINER_PREFIX}-${instance_name}"
    
    # Execute SQL file in container by piping content
    if cat "$sql_file" | docker exec -i "$container_name" psql \
        -U "$POSTGRES_DEFAULT_USER" \
        -d "$database" \
        -q 2>&1; then
        return 0
    else
        local exit_code=$?
        log::error "Failed to execute SQL file '$sql_file' on instance '$instance_name'"
        log::error "Exit code: $exit_code"
        return 1
    fi
}

#######################################
# Create database in PostgreSQL instance
# Arguments:
#   $1 - instance name
#   $2 - database name
#   $3 - owner (optional, default: $POSTGRES_DEFAULT_USER)
# Returns: 0 on success, 1 on failure
#######################################
postgres::database::create() {
    local instance_name="$1"
    local db_name="$2"
    local owner="${3:-$POSTGRES_DEFAULT_USER}"
    
    if [[ -z "$instance_name" || -z "$db_name" ]]; then
        log::error "Instance name and database name are required"
        return 1
    fi
    
    # Validate database name (basic validation)
    if [[ ! "$db_name" =~ ^[a-zA-Z_][a-zA-Z0-9_]*$ ]]; then
        log::error "Invalid database name: $db_name (only letters, numbers, and underscores allowed)"
        return 1
    fi
    
    log::info "Creating database '$db_name' in instance '$instance_name'..."
    
    # Check if database already exists
    local check_sql="SELECT 1 FROM pg_database WHERE datname = '$db_name';"
    if postgres::database::execute "$instance_name" "$check_sql" "postgres" | grep -q "1"; then
        log::warn "Database '$db_name' already exists in instance '$instance_name'"
        return 0
    fi
    
    # Create database
    local create_sql="CREATE DATABASE \"$db_name\" OWNER \"$owner\";"
    if postgres::database::execute "$instance_name" "$create_sql" "postgres"; then
        log::success "Database '$db_name' created successfully"
        return 0
    else
        log::error "Failed to create database '$db_name'"
        return 1
    fi
}

#######################################
# Drop database from PostgreSQL instance
# Arguments:
#   $1 - instance name
#   $2 - database name
#   $3 - force flag (optional, "yes" to skip confirmation)
# Returns: 0 on success, 1 on failure
#######################################
postgres::database::drop() {
    local instance_name="$1"
    local db_name="$2"
    local force="${3:-no}"
    
    if [[ -z "$instance_name" || -z "$db_name" ]]; then
        log::error "Instance name and database name are required"
        return 1
    fi
    
    # Safety check - don't drop default database
    if [[ "$db_name" == "$POSTGRES_DEFAULT_DB" ]]; then
        log::error "Cannot drop default database '$POSTGRES_DEFAULT_DB'"
        return 1
    fi
    
    # Confirmation unless forced
    if [[ "$force" != "yes" ]]; then
        log::warn "This will permanently delete database '$db_name' from instance '$instance_name'"
        read -p "Are you sure? (y/N) " -n 1 -r
        echo
        if [[ ! $REPLY =~ ^[Yy]$ ]]; then
            log::info "Database drop cancelled"
            return 0
        fi
    fi
    
    log::info "Dropping database '$db_name' from instance '$instance_name'..."
    
    # Drop database
    local drop_sql="DROP DATABASE IF EXISTS \"$db_name\";"
    if postgres::database::execute "$instance_name" "$drop_sql" "postgres"; then
        log::success "Database '$db_name' dropped successfully"
        return 0
    else
        log::error "Failed to drop database '$db_name'"
        return 1
    fi
}

#######################################
# Create user in PostgreSQL instance
# Arguments:
#   $1 - instance name
#   $2 - username
#   $3 - password
#   $4 - privileges (optional, default: "CREATEDB")
# Returns: 0 on success, 1 on failure
#######################################
postgres::database::create_user() {
    local instance_name="$1"
    local username="$2"
    local password="$3"
    local privileges="${4:-CREATEDB}"
    
    if [[ -z "$instance_name" || -z "$username" || -z "$password" ]]; then
        log::error "Instance name, username, and password are required"
        return 1
    fi
    
    # Validate username (basic validation)
    if [[ ! "$username" =~ ^[a-zA-Z_][a-zA-Z0-9_]*$ ]]; then
        log::error "Invalid username: $username (only letters, numbers, and underscores allowed)"
        return 1
    fi
    
    log::info "Creating user '$username' in instance '$instance_name'..."
    
    # Check if user already exists
    local check_sql="SELECT 1 FROM pg_user WHERE usename = '$username';"
    if postgres::database::execute "$instance_name" "$check_sql" "postgres" | grep -q "1"; then
        log::warn "User '$username' already exists in instance '$instance_name'"
        return 0
    fi
    
    # Create user
    local create_sql="CREATE USER \"$username\" WITH PASSWORD '$password' $privileges;"
    if postgres::database::execute "$instance_name" "$create_sql" "postgres"; then
        log::success "User '$username' created successfully"
        return 0
    else
        log::error "Failed to create user '$username'"
        return 1
    fi
}

#######################################
# Drop user from PostgreSQL instance
# Arguments:
#   $1 - instance name
#   $2 - username
#   $3 - force flag (optional, "yes" to skip confirmation)
# Returns: 0 on success, 1 on failure
#######################################
postgres::database::drop_user() {
    local instance_name="$1"
    local username="$2"
    local force="${3:-no}"
    
    if [[ -z "$instance_name" || -z "$username" ]]; then
        log::error "Instance name and username are required"
        return 1
    fi
    
    # Safety check - don't drop default user
    if [[ "$username" == "$POSTGRES_DEFAULT_USER" ]]; then
        log::error "Cannot drop default user '$POSTGRES_DEFAULT_USER'"
        return 1
    fi
    
    # Confirmation unless forced
    if [[ "$force" != "yes" ]]; then
        log::warn "This will permanently delete user '$username' from instance '$instance_name'"
        read -p "Are you sure? (y/N) " -n 1 -r
        echo
        if [[ ! $REPLY =~ ^[Yy]$ ]]; then
            log::info "User drop cancelled"
            return 0
        fi
    fi
    
    log::info "Dropping user '$username' from instance '$instance_name'..."
    
    # Drop user
    local drop_sql="DROP USER IF EXISTS \"$username\";"
    if postgres::database::execute "$instance_name" "$drop_sql" "postgres"; then
        log::success "User '$username' dropped successfully"
        return 0
    else
        log::error "Failed to drop user '$username'"
        return 1
    fi
}

#######################################
# Run schema migrations on instance
# Arguments:
#   $1 - instance name
#   $2 - migrations directory path
#   $3 - database name (optional, default: $POSTGRES_DEFAULT_DB)
# Returns: 0 on success, 1 on failure
#######################################
postgres::database::migrate() {
    local instance_name="$1"
    local migrations_dir="$2"
    local database="${3:-$POSTGRES_DEFAULT_DB}"
    
    if [[ -z "$instance_name" || -z "$migrations_dir" ]]; then
        log::error "Instance name and migrations directory are required"
        return 1
    fi
    
    if [[ ! -d "$migrations_dir" ]]; then
        log::error "Migrations directory not found: $migrations_dir"
        return 1
    fi
    
    if ! postgres::common::container_exists "$instance_name"; then
        log::error "${MSG_INSTANCE_NOT_FOUND}: $instance_name"
        return 1
    fi
    
    if ! postgres::common::is_running "$instance_name"; then
        log::error "Instance '$instance_name' is not running"
        return 1
    fi
    
    log::info "Running migrations on instance '$instance_name' database '$database'..."
    
    # Create migrations table if it doesn't exist
    local migrations_table_sql="
        CREATE TABLE IF NOT EXISTS schema_migrations (
            version VARCHAR(255) PRIMARY KEY,
            applied_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
        );"
    
    if ! postgres::database::execute "$instance_name" "$migrations_table_sql" "$database"; then
        log::error "Failed to create migrations table"
        return 1
    fi
    
    # Find and run migration files
    local migration_files=($(find "$migrations_dir" -name "*.sql" | sort))
    local applied_count=0
    local skipped_count=0
    
    for migration_file in "${migration_files[@]}"; do
        local migration_name=$(basename "$migration_file" .sql)
        
        # Check if migration already applied
        local check_sql="SELECT 1 FROM schema_migrations WHERE version = '$migration_name';"
        if postgres::database::execute "$instance_name" "$check_sql" "$database" | grep -q "1"; then
            log::info "  Skipping already applied migration: $migration_name"
            ((skipped_count++))
            continue
        fi
        
        # Apply migration
        log::info "  Applying migration: $migration_name"
        if postgres::database::execute_file "$instance_name" "$migration_file" "$database"; then
            # Record migration as applied
            local record_sql="INSERT INTO schema_migrations (version) VALUES ('$migration_name');"
            postgres::database::execute "$instance_name" "$record_sql" "$database"
            ((applied_count++))
        else
            log::error "Failed to apply migration: $migration_name"
            return 1
        fi
    done
    
    log::success "Migrations completed: $applied_count applied, $skipped_count skipped"
    return 0
}

#######################################
# Seed database with initial data
# Arguments:
#   $1 - instance name
#   $2 - seed file or directory path
#   $3 - database name (optional, default: $POSTGRES_DEFAULT_DB)
# Returns: 0 on success, 1 on failure
#######################################
postgres::database::seed() {
    local instance_name="$1"
    local seed_path="$2"
    local database="${3:-$POSTGRES_DEFAULT_DB}"
    
    if [[ -z "$instance_name" || -z "$seed_path" ]]; then
        log::error "Instance name and seed path are required"
        return 1
    fi
    
    if [[ ! -e "$seed_path" ]]; then
        log::error "Seed path not found: $seed_path"
        return 1
    fi
    
    if ! postgres::common::container_exists "$instance_name"; then
        log::error "${MSG_INSTANCE_NOT_FOUND}: $instance_name"
        return 1
    fi
    
    if ! postgres::common::is_running "$instance_name"; then
        log::error "Instance '$instance_name' is not running"
        return 1
    fi
    
    log::info "Seeding database '$database' in instance '$instance_name'..."
    
    if [[ -f "$seed_path" ]]; then
        # Single seed file
        if postgres::database::execute_file "$instance_name" "$seed_path" "$database"; then
            log::success "Seed file applied successfully: $(basename "$seed_path")"
        else
            log::error "Failed to apply seed file: $(basename "$seed_path")"
            return 1
        fi
    elif [[ -d "$seed_path" ]]; then
        # Directory of seed files
        local seed_files=($(find "$seed_path" -name "*.sql" | sort))
        local applied_count=0
        
        for seed_file in "${seed_files[@]}"; do
            local seed_name=$(basename "$seed_file")
            log::info "  Applying seed: $seed_name"
            
            if postgres::database::execute_file "$instance_name" "$seed_file" "$database"; then
                ((applied_count++))
            else
                log::error "Failed to apply seed: $seed_name"
                return 1
            fi
        done
        
        log::success "All seeds applied successfully: $applied_count file(s)"
    else
        log::error "Invalid seed path: $seed_path (must be file or directory)"
        return 1
    fi
    
    return 0
}

#######################################
# Get database statistics for instance
# Arguments:
#   $1 - instance name
#   $2 - database name (optional, default: $POSTGRES_DEFAULT_DB)
# Returns: 0 on success, 1 on failure
#######################################
postgres::database::stats() {
    local instance_name="$1"
    local database="${2:-$POSTGRES_DEFAULT_DB}"
    
    if [[ -z "$instance_name" ]]; then
        log::error "Instance name is required"
        return 1
    fi
    
    if ! postgres::common::container_exists "$instance_name"; then
        log::error "${MSG_INSTANCE_NOT_FOUND}: $instance_name"
        return 1
    fi
    
    if ! postgres::common::is_running "$instance_name"; then
        log::error "Instance '$instance_name' is not running"
        return 1
    fi
    
    log::info "Database Statistics for '$database' in instance '$instance_name':"
    log::info "============================================================"
    
    # Get database size
    local size_sql="SELECT pg_size_pretty(pg_database_size('$database'));"
    local db_size=$(postgres::database::execute "$instance_name" "$size_sql" "$database" 2>/dev/null | tail -n +3 | head -n 1 | tr -d ' ' || echo "N/A")
    log::info "Database Size: $db_size"
    
    # Get table count
    local table_sql="SELECT count(*) FROM information_schema.tables WHERE table_schema = 'public';"
    local table_count=$(postgres::database::execute "$instance_name" "$table_sql" "$database" 2>/dev/null | tail -n +3 | head -n 1 | tr -d ' ' || echo "N/A")
    log::info "Tables: $table_count"
    
    # Get connection count
    local conn_sql="SELECT count(*) FROM pg_stat_activity WHERE datname = '$database';"
    local conn_count=$(postgres::database::execute "$instance_name" "$conn_sql" "$database" 2>/dev/null | tail -n +3 | head -n 1 | tr -d ' ' || echo "N/A")
    log::info "Active Connections: $conn_count"
    
    # Get version
    local version_sql="SELECT version();"
    local pg_version=$(postgres::database::execute "$instance_name" "$version_sql" "$database" 2>/dev/null | tail -n +3 | head -n 1 | cut -d',' -f1 || echo "N/A")
    log::info "PostgreSQL Version: $pg_version"
    
    return 0
}

#######################################
# List all databases in instance
# Arguments:
#   $1 - instance name
# Returns: 0 on success, 1 on failure
#######################################
postgres::database::list() {
    local instance_name="$1"
    
    if [[ -z "$instance_name" ]]; then
        log::error "Instance name is required"
        return 1
    fi
    
    if ! postgres::common::container_exists "$instance_name"; then
        log::error "${MSG_INSTANCE_NOT_FOUND}: $instance_name"
        return 1
    fi
    
    if ! postgres::common::is_running "$instance_name"; then
        log::error "Instance '$instance_name' is not running"
        return 1
    fi
    
    log::info "Databases in instance '$instance_name':"
    log::info "======================================"
    
    # List databases with owner and size
    local list_sql="
        SELECT 
            d.datname as \"Database\",
            pg_catalog.pg_get_userbyid(d.datdba) as \"Owner\",
            pg_size_pretty(pg_database_size(d.datname)) as \"Size\"
        FROM pg_catalog.pg_database d
        WHERE d.datistemplate = false
        ORDER BY d.datname;"
    
    if postgres::database::execute "$instance_name" "$list_sql" "postgres"; then
        return 0
    else
        log::error "Failed to list databases"
        return 1
    fi
}

#######################################
# Dump database schema to file
# Arguments:
#   $1 - instance name
#   $2 - output file path
#   $3 - database name (optional, default: $POSTGRES_DEFAULT_DB)
# Returns: 0 on success, 1 on failure
#######################################
postgres::database::dump_schema() {
    local instance_name="$1"
    local output_file="$2"
    local database="${3:-$POSTGRES_DEFAULT_DB}"
    
    if [[ -z "$instance_name" || -z "$output_file" ]]; then
        log::error "Instance name and output file are required"
        return 1
    fi
    
    if ! postgres::common::container_exists "$instance_name"; then
        log::error "${MSG_INSTANCE_NOT_FOUND}: $instance_name"
        return 1
    fi
    
    if ! postgres::common::is_running "$instance_name"; then
        log::error "Instance '$instance_name' is not running"
        return 1
    fi
    
    log::info "Dumping schema from database '$database' in instance '$instance_name'..."
    
    local container_name="${POSTGRES_CONTAINER_PREFIX}-${instance_name}"
    
    # Create output directory if it doesn't exist
    mkdir -p "${output_file%/*"
    
    # Dump schema only (no data)
    if docker exec "$container_name" pg_dump \
        -U "$POSTGRES_DEFAULT_USER" \
        -d "$database" \
        --schema-only \
        --no-owner \
        --no-privileges > "$output_file" 2>/dev/null; then
        log::success "Schema dumped to: $output_file"
        return 0
    else
        log::error "Failed to dump schema"
        return 1
    fi
}

#######################################
# Dump database data to file
# Arguments:
#   $1 - instance name
#   $2 - output file path
#   $3 - database name (optional, default: $POSTGRES_DEFAULT_DB)
#   $4 - tables (optional, space-separated list)
# Returns: 0 on success, 1 on failure
#######################################
postgres::database::dump_data() {
    local instance_name="$1"
    local output_file="$2"
    local database="${3:-$POSTGRES_DEFAULT_DB}"
    local tables="${4:-}"
    
    if [[ -z "$instance_name" || -z "$output_file" ]]; then
        log::error "Instance name and output file are required"
        return 1
    fi
    
    if ! postgres::common::container_exists "$instance_name"; then
        log::error "${MSG_INSTANCE_NOT_FOUND}: $instance_name"
        return 1
    fi
    
    if ! postgres::common::is_running "$instance_name"; then
        log::error "Instance '$instance_name' is not running"
        return 1
    fi
    
    log::info "Dumping data from database '$database' in instance '$instance_name'..."
    
    local container_name="${POSTGRES_CONTAINER_PREFIX}-${instance_name}"
    
    # Create output directory if it doesn't exist
    mkdir -p "${output_file%/*"
    
    # Build pg_dump command
    local dump_cmd="pg_dump -U $POSTGRES_DEFAULT_USER -d $database --data-only --no-owner --no-privileges"
    
    # Add table filtering if specified
    if [[ -n "$tables" ]]; then
        for table in $tables; do
            dump_cmd="$dump_cmd --table=$table"
        done
    fi
    
    # Dump data
    if docker exec "$container_name" $dump_cmd > "$output_file" 2>/dev/null; then
        log::success "Data dumped to: $output_file"
        return 0
    else
        log::error "Failed to dump data"
        return 1
    fi
}