#!/usr/bin/env bash
# Windmill Database Management Functions
# Functions for managing PostgreSQL database operations

#######################################
# Check if database is healthy
# Returns: 0 if healthy, 1 otherwise
#######################################
windmill::check_database_health() {
    if [[ "$EXTERNAL_DB" == "yes" ]]; then
        windmill::check_external_database_health
    else
        windmill::check_internal_database_health
    fi
}

#######################################
# Check internal PostgreSQL database health
# Returns: 0 if healthy, 1 otherwise
#######################################
windmill::check_internal_database_health() {
    if ! windmill::compose_cmd ps --services | grep -q "windmill-db"; then
        log::error "Internal database service not found"
        return 1
    fi
    
    # Check if container is running
    if ! docker ps --format "{{.Names}}" | grep -q "${WINDMILL_PROJECT_NAME}-db"; then
        log::error "Database container is not running"
        return 1
    fi
    
    # Check PostgreSQL readiness
    if windmill::compose_cmd exec -T windmill-db pg_isready -U postgres >/dev/null 2>&1; then
        return 0
    else
        log::error "PostgreSQL is not ready"
        return 1
    fi
}

#######################################
# Check external database health
# Returns: 0 if healthy, 1 otherwise
#######################################
windmill::check_external_database_health() {
    if [[ -z "$DB_URL" ]]; then
        log::error "External database URL not configured"
        return 1
    fi
    
    # Extract connection details from URL
    local db_host db_port db_user db_name
    if [[ "$DB_URL" =~ postgresql://([^:]+):([^@]+)@([^:]+):([0-9]+)/(.+) ]]; then
        db_user="${BASH_REMATCH[1]}"
        db_host="${BASH_REMATCH[3]}"
        db_port="${BASH_REMATCH[4]}"
        db_name="${BASH_REMATCH[5]}"
    else
        log::error "Invalid database URL format"
        return 1
    fi
    
    # Test connection using pg_isready
    if docker run --rm postgres:16 pg_isready -h "$db_host" -p "$db_port" -U "$db_user" >/dev/null 2>&1; then
        return 0
    else
        log::error "Cannot connect to external database: $db_host:$db_port"
        return 1
    fi
}

#######################################
# Initialize database schema (for fresh installations)
# Returns: 0 if successful, 1 otherwise
#######################################
windmill::init_database() {
    log::info "Initializing Windmill database..."
    
    if ! windmill::check_database_health; then
        log::error "Database is not healthy - cannot initialize"
        return 1
    fi
    
    # Windmill handles schema initialization automatically
    # Just verify the connection works
    if [[ "$EXTERNAL_DB" == "yes" ]]; then
        log::info "Using external database - schema will be initialized by Windmill server"
    else
        log::info "Using internal database - schema will be initialized by Windmill server"
    fi
    
    return 0
}

#######################################
# Backup internal database
# Arguments:
#   $1 - Backup file path
# Returns: 0 if successful, 1 otherwise
#######################################
windmill::backup_database() {
    local backup_file="$1"
    
    if [[ "$EXTERNAL_DB" == "yes" ]]; then
        log::warn "External database backup should be handled by your database administrator"
        return 1
    fi
    
    if ! windmill::check_internal_database_health; then
        log::error "Database is not healthy - cannot backup"
        return 1
    fi
    
    log::info "Creating database backup: $backup_file"
    
    if windmill::compose_cmd exec -T windmill-db pg_dump -U postgres -d windmill > "$backup_file"; then
        log::success "Database backup completed: $backup_file"
        
        # Show backup info
        local backup_size
        backup_size=$(du -h "$backup_file" | cut -f1)
        log::info "Backup size: $backup_size"
        
        return 0
    else
        log::error "Database backup failed"
        return 1
    fi
}

#######################################
# Restore database from backup
# Arguments:
#   $1 - Backup file path
# Returns: 0 if successful, 1 otherwise
#######################################
windmill::restore_database() {
    local backup_file="$1"
    
    if [[ ! -f "$backup_file" ]]; then
        log::error "Backup file not found: $backup_file"
        return 1
    fi
    
    if [[ "$EXTERNAL_DB" == "yes" ]]; then
        log::warn "External database restore should be handled by your database administrator"
        log::info "Backup file: $backup_file"
        return 1
    fi
    
    if ! windmill::check_internal_database_health; then
        log::error "Database is not healthy - cannot restore"
        return 1
    fi
    
    log::info "Restoring database from: $backup_file"
    log::warn "This will overwrite all existing data!"
    
    # Drop and recreate database
    log::info "Recreating database..."
    if ! windmill::compose_cmd exec -T windmill-db psql -U postgres -c "DROP DATABASE IF EXISTS windmill;" >/dev/null 2>&1; then
        log::error "Failed to drop existing database"
        return 1
    fi
    
    if ! windmill::compose_cmd exec -T windmill-db psql -U postgres -c "CREATE DATABASE windmill;" >/dev/null 2>&1; then
        log::error "Failed to create new database"
        return 1
    fi
    
    # Restore from backup
    log::info "Restoring data..."
    if windmill::compose_cmd exec -T windmill-db psql -U postgres -d windmill < "$backup_file"; then
        log::success "Database restored successfully"
        return 0
    else
        log::error "Database restore failed"
        return 1
    fi
}

#######################################
# Show database status and information
#######################################
windmill::database_status() {
    log::info "📊 Database Status:"
    
    if [[ "$EXTERNAL_DB" == "yes" ]]; then
        echo "  Type: External PostgreSQL"
        echo "  URL: $DB_URL"
        
        if windmill::check_external_database_health; then
            echo "  Status: ✅ Connected"
        else
            echo "  Status: ❌ Not accessible"
        fi
    else
        echo "  Type: Internal PostgreSQL (Docker)"
        echo "  Container: ${WINDMILL_PROJECT_NAME}-db"
        
        if windmill::check_internal_database_health; then
            echo "  Status: ✅ Running and ready"
            
            # Get additional info for internal database
            local db_version connections db_size
            
            # Database version
            db_version=$(windmill::compose_cmd exec -T windmill-db psql -U postgres -t -c "SELECT version();" 2>/dev/null | head -n1 | xargs || echo "unknown")
            echo "  Version: $db_version"
            
            # Active connections
            connections=$(windmill::compose_cmd exec -T windmill-db psql -U postgres -t -c "SELECT count(*) FROM pg_stat_activity WHERE datname='windmill';" 2>/dev/null | xargs || echo "unknown")
            echo "  Active Connections: $connections"
            
            # Database size
            db_size=$(windmill::compose_cmd exec -T windmill-db psql -U postgres -t -c "SELECT pg_size_pretty(pg_database_size('windmill'));" 2>/dev/null | xargs || echo "unknown")
            echo "  Database Size: $db_size"
            
            # Show tables
            local table_count
            table_count=$(windmill::compose_cmd exec -T windmill-db psql -U postgres -d windmill -t -c "SELECT count(*) FROM information_schema.tables WHERE table_schema = 'public';" 2>/dev/null | xargs || echo "unknown")
            echo "  Tables: $table_count"
            
        else
            echo "  Status: ❌ Not running or not ready"
        fi
    fi
}

#######################################
# Connect to database (interactive psql session)
# Returns: 0 if successful, 1 otherwise
#######################################
windmill::connect_database() {
    if [[ "$EXTERNAL_DB" == "yes" ]]; then
        log::error "Interactive connection to external database not supported"
        log::info "Use your preferred PostgreSQL client with: $DB_URL"
        return 1
    fi
    
    if ! windmill::check_internal_database_health; then
        log::error "Database is not running or not healthy"
        return 1
    fi
    
    log::info "Connecting to Windmill database..."
    log::info "Type \\q to exit psql session"
    echo
    
    windmill::compose_cmd exec windmill-db psql -U postgres -d windmill
}

#######################################
# Execute SQL query on database
# Arguments:
#   $1 - SQL query
# Returns: 0 if successful, 1 otherwise
#######################################
windmill::execute_sql() {
    local query="$1"
    
    if [[ -z "$query" ]]; then
        log::error "SQL query is required"
        return 1
    fi
    
    if [[ "$EXTERNAL_DB" == "yes" ]]; then
        log::error "SQL execution on external database not supported via this tool"
        log::info "Use your preferred PostgreSQL client with: $DB_URL"
        return 1
    fi
    
    if ! windmill::check_internal_database_health; then
        log::error "Database is not running or not healthy"
        return 1
    fi
    
    log::info "Executing SQL query..."
    
    windmill::compose_cmd exec -T windmill-db psql -U postgres -d windmill -c "$query"
}

#######################################
# Show database logs
# Arguments:
#   $1 - Number of lines (optional, default: 100)
#   $2 - Follow logs flag (optional, default: false)
#######################################
windmill::database_logs() {
    local lines="${1:-100}"
    local follow="${2:-false}"
    
    if [[ "$EXTERNAL_DB" == "yes" ]]; then
        log::error "Cannot show logs for external database"
        return 1
    fi
    
    local compose_args=("logs")
    
    if [[ "$follow" == "true" ]]; then
        compose_args+=("--follow")
    fi
    
    compose_args+=("--timestamps" "--tail=$lines" "windmill-db")
    
    log::info "Database logs (last $lines lines):"
    echo
    
    windmill::compose_cmd "${compose_args[@]}"
}

#######################################
# Vacuum and analyze database (maintenance)
# Returns: 0 if successful, 1 otherwise
#######################################
windmill::vacuum_database() {
    if [[ "$EXTERNAL_DB" == "yes" ]]; then
        log::error "Database maintenance on external database should be handled by your DBA"
        return 1
    fi
    
    if ! windmill::check_internal_database_health; then
        log::error "Database is not running or not healthy"
        return 1
    fi
    
    log::info "Performing database maintenance (VACUUM ANALYZE)..."
    
    if windmill::compose_cmd exec -T windmill-db psql -U postgres -d windmill -c "VACUUM ANALYZE;" >/dev/null 2>&1; then
        log::success "Database maintenance completed"
        return 0
    else
        log::error "Database maintenance failed"
        return 1
    fi
}

#######################################
# Show database performance statistics
#######################################
windmill::database_stats() {
    if [[ "$EXTERNAL_DB" == "yes" ]]; then
        log::error "Database statistics not available for external database"
        return 1
    fi
    
    if ! windmill::check_internal_database_health; then
        log::error "Database is not running or not healthy"
        return 1
    fi
    
    log::info "📈 Database Performance Statistics:"
    echo
    
    # Connection stats
    echo "Connection Statistics:"
    windmill::compose_cmd exec -T windmill-db psql -U postgres -d windmill -c "
        SELECT 
            state,
            count(*) as connections
        FROM pg_stat_activity 
        WHERE datname = 'windmill'
        GROUP BY state;
    " 2>/dev/null || echo "  Unable to retrieve connection stats"
    
    echo
    
    # Table sizes
    echo "Top Tables by Size:"
    windmill::compose_cmd exec -T windmill-db psql -U postgres -d windmill -c "
        SELECT 
            schemaname,
            tablename,
            pg_size_pretty(pg_total_relation_size(schemaname||'.'||tablename)) as size
        FROM pg_tables 
        WHERE schemaname = 'public'
        ORDER BY pg_total_relation_size(schemaname||'.'||tablename) DESC 
        LIMIT 10;
    " 2>/dev/null || echo "  Unable to retrieve table stats"
    
    echo
    
    # Database activity
    echo "Database Activity:"
    windmill::compose_cmd exec -T windmill-db psql -U postgres -d windmill -c "
        SELECT 
            tup_returned + tup_fetched as reads,
            tup_inserted + tup_updated + tup_deleted as writes
        FROM pg_stat_database 
        WHERE datname = 'windmill';
    " 2>/dev/null || echo "  Unable to retrieve activity stats"
}

#######################################
# Reset database (dangerous - removes all data)
# Returns: 0 if successful, 1 otherwise
#######################################
windmill::reset_database() {
    if [[ "$EXTERNAL_DB" == "yes" ]]; then
        log::error "Cannot reset external database"
        return 1
    fi
    
    log::warn "⚠️  WARNING: This will permanently delete ALL Windmill data!"
    log::warn "This includes:"
    log::warn "  • All workspaces and scripts"
    log::warn "  • All job history and executions"
    log::warn "  • All user accounts and settings"
    log::warn "  • All resources and variables"
    echo
    
    if ! flow::is_yes "$YES"; then
        read -p "Type 'DELETE ALL DATA' to confirm: " -r
        if [[ "$REPLY" != "DELETE ALL DATA" ]]; then
            log::info "Database reset cancelled"
            return 0
        fi
    fi
    
    if ! windmill::check_internal_database_health; then
        log::error "Database is not running - cannot reset"
        return 1
    fi
    
    log::info "Resetting database..."
    
    # Drop all tables
    if windmill::compose_cmd exec -T windmill-db psql -U postgres -d windmill -c "
        DROP SCHEMA public CASCADE;
        CREATE SCHEMA public;
        GRANT ALL ON SCHEMA public TO postgres;
        GRANT ALL ON SCHEMA public TO public;
    " >/dev/null 2>&1; then
        log::success "Database reset completed"
        log::info "Restart Windmill services to reinitialize the schema"
        return 0
    else
        log::error "Database reset failed"
        return 1
    fi
}