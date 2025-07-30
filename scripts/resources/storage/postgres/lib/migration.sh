#!/usr/bin/env bash
# PostgreSQL Advanced Migration Support
# Enhanced migration tooling beyond basic migrate function

#######################################
# Initialize migration system for instance
# Arguments:
#   $1 - instance name
#   $2 - database name (optional, default: $POSTGRES_DEFAULT_DB)
# Returns: 0 on success, 1 on failure
#######################################
postgres::migration::init() {
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
    
    log::info "Initializing migration system for instance '$instance_name'..."
    
    # Create enhanced migrations table with more metadata
    local migrations_table_sql="
        CREATE TABLE IF NOT EXISTS schema_migrations (
            version VARCHAR(255) PRIMARY KEY,
            description TEXT,
            applied_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
            applied_by VARCHAR(255) DEFAULT CURRENT_USER,
            execution_time_ms INTEGER,
            checksum VARCHAR(64),
            rollback_sql TEXT
        );
        
        CREATE INDEX IF NOT EXISTS idx_schema_migrations_applied_at 
        ON schema_migrations(applied_at);
        
        CREATE TABLE IF NOT EXISTS migration_history (
            id SERIAL PRIMARY KEY,
            version VARCHAR(255) NOT NULL,
            action VARCHAR(20) NOT NULL, -- 'apply', 'rollback'
            applied_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
            applied_by VARCHAR(255) DEFAULT CURRENT_USER,
            execution_time_ms INTEGER,
            success BOOLEAN DEFAULT TRUE,
            error_message TEXT
        );"
    
    if postgres::database::execute "$instance_name" "$migrations_table_sql" "$database"; then
        log::success "Migration system initialized successfully"
        return 0
    else
        log::error "Failed to initialize migration system"
        return 1
    fi
}

#######################################
# Run migrations with enhanced features
# Arguments:
#   $1 - instance name
#   $2 - migrations directory path
#   $3 - database name (optional, default: $POSTGRES_DEFAULT_DB)
#   $4 - dry run flag (optional, "yes" to simulate)
# Returns: 0 on success, 1 on failure
#######################################
postgres::migration::run() {
    local instance_name="$1"
    local migrations_dir="$2"
    local database="${3:-$POSTGRES_DEFAULT_DB}"
    local dry_run="${4:-no}"
    
    if [[ -z "$instance_name" || -z "$migrations_dir" ]]; then
        log::error "Instance name and migrations directory are required"
        return 1
    fi
    
    if [[ ! -d "$migrations_dir" ]]; then
        log::error "Migrations directory not found: $migrations_dir"
        return 1
    fi
    
    # Initialize migration system if needed
    postgres::migration::init "$instance_name" "$database"
    
    log::info "Running migrations on instance '$instance_name' database '$database'..."
    if [[ "$dry_run" == "yes" ]]; then
        log::info "[DRY RUN MODE] - No changes will be applied"
    fi
    
    # Find and validate migration files
    local migration_files=($(find "$migrations_dir" -name "*.sql" | sort))
    local applied_count=0
    local skipped_count=0
    local failed_count=0
    
    if [[ ${#migration_files[@]} -eq 0 ]]; then
        log::warn "No migration files found in $migrations_dir"
        return 0
    fi
    
    log::info "Found ${#migration_files[@]} migration file(s)"
    
    for migration_file in "${migration_files[@]}"; do
        local migration_name=$(basename "$migration_file" .sql)
        local start_time=$(date +%s%3N)
        
        # Check if migration already applied
        local check_sql="SELECT version, checksum FROM schema_migrations WHERE version = '$migration_name';"
        local existing_migration=$(postgres::database::execute "$instance_name" "$check_sql" "$database" 2>/dev/null | tail -n +3 | head -n 1)
        
        if [[ -n "$existing_migration" ]]; then
            # Verify checksum if migration exists
            local current_checksum=$(postgres::migration::calculate_checksum "$migration_file")
            local stored_checksum=$(echo "$existing_migration" | awk '{print $2}')
            
            if [[ "$current_checksum" == "$stored_checksum" ]]; then
                log::info "  ✓ Skipping already applied migration: $migration_name"
                ((skipped_count++))
                continue
            else
                log::warn "  ⚠ Migration checksum mismatch for: $migration_name"
                log::warn "    Stored: $stored_checksum, Current: $current_checksum"
                if [[ "$dry_run" != "yes" ]]; then
                    read -p "    Continue anyway? (y/N) " -n 1 -r
                    echo
                    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
                        log::info "    Skipping migration due to checksum mismatch"
                        ((skipped_count++))
                        continue
                    fi
                fi
            fi
        fi
        
        # Parse migration metadata
        local description=$(postgres::migration::extract_description "$migration_file")
        local rollback_sql=$(postgres::migration::extract_rollback "$migration_file")
        
        log::info "  → Applying migration: $migration_name"
        if [[ -n "$description" ]]; then
            log::info "    Description: $description"
        fi
        
        if [[ "$dry_run" == "yes" ]]; then
            log::info "    [DRY RUN] Would apply migration: $migration_name"
            ((applied_count++))
            continue
        fi
        
        # Validate migration before applying
        if ! postgres::migration::validate_migration "$migration_file"; then
            log::error "  ✗ Migration validation failed: $migration_name"
            ((failed_count++))
            continue
        fi
        
        # Apply migration in transaction
        local migration_success=true
        local error_message=""
        
        # Execute migration file directly
        if postgres::database::execute_file "$instance_name" "$migration_file" "$database"; then
            local end_time=$(date +%s%3N)
            local execution_time=$((end_time - start_time))
            local checksum=$(postgres::migration::calculate_checksum "$migration_file")
            
            # Record successful migration
            local record_sql="
                INSERT INTO schema_migrations (version, description, execution_time_ms, checksum, rollback_sql) 
                VALUES ('$migration_name', '$description', $execution_time, '$checksum', '$rollback_sql');
                
                INSERT INTO migration_history (version, action, execution_time_ms, success) 
                VALUES ('$migration_name', 'apply', $execution_time, TRUE);"
            
            postgres::database::execute "$instance_name" "$record_sql" "$database"
            
            log::success "  ✓ Applied migration: $migration_name (${execution_time}ms)"
            ((applied_count++))
        else
            migration_success=false
            error_message="Failed to execute migration SQL"
            
            # Record failed migration
            local record_error_sql="
                INSERT INTO migration_history (version, action, success, error_message) 
                VALUES ('$migration_name', 'apply', FALSE, '$error_message');"
            
            postgres::database::execute "$instance_name" "$record_error_sql" "$database" 2>/dev/null
            
            log::error "  ✗ Failed to apply migration: $migration_name"
            log::error "    Error: $error_message"
            ((failed_count++))
            
            # Stop on first failure unless in continue mode
            break
        fi
    done
    
    # Summary
    log::info ""
    log::info "Migration Summary:"
    log::info "  Applied: $applied_count"
    log::info "  Skipped: $skipped_count"
    log::info "  Failed:  $failed_count"
    
    if [[ $failed_count -eq 0 ]]; then
        if [[ "$dry_run" == "yes" ]]; then
            log::success "[DRY RUN] All migrations would be applied successfully"
        else
            log::success "All migrations applied successfully"
        fi
        return 0
    else
        log::error "Migration failed with $failed_count error(s)"
        return 1
    fi
}

#######################################
# Rollback migration
# Arguments:
#   $1 - instance name
#   $2 - migration version to rollback
#   $3 - database name (optional, default: $POSTGRES_DEFAULT_DB)
# Returns: 0 on success, 1 on failure
#######################################
postgres::migration::rollback() {
    local instance_name="$1"
    local migration_version="$2"
    local database="${3:-$POSTGRES_DEFAULT_DB}"
    
    if [[ -z "$instance_name" || -z "$migration_version" ]]; then
        log::error "Instance name and migration version are required"
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
    
    log::info "Rolling back migration '$migration_version' on instance '$instance_name'..."
    
    # Get rollback SQL
    local get_rollback_sql="SELECT rollback_sql FROM schema_migrations WHERE version = '$migration_version';"
    local rollback_sql=$(postgres::database::execute "$instance_name" "$get_rollback_sql" "$database" 2>/dev/null | tail -n +3 | head -n 1)
    
    if [[ -z "$rollback_sql" || "$rollback_sql" == "NULL" ]]; then
        log::error "No rollback SQL found for migration '$migration_version'"
        log::info "This migration may not support rollback"
        return 1
    fi
    
    # Confirm rollback
    log::warn "This will rollback migration '$migration_version'"
    read -p "Are you sure? (y/N) " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        log::info "Rollback cancelled"
        return 0
    fi
    
    local start_time=$(date +%s%3N)
    
    # Execute rollback in transaction
    local transaction_sql="
        BEGIN;
        $rollback_sql
        DELETE FROM schema_migrations WHERE version = '$migration_version';
        COMMIT;"
    
    if postgres::database::execute "$instance_name" "$transaction_sql" "$database"; then
        local end_time=$(date +%s%3N)
        local execution_time=$((end_time - start_time))
        
        # Record rollback in history
        local record_sql="
            INSERT INTO migration_history (version, action, execution_time_ms, success) 
            VALUES ('$migration_version', 'rollback', $execution_time, TRUE);"
        
        postgres::database::execute "$instance_name" "$record_sql" "$database"
        
        log::success "Migration '$migration_version' rolled back successfully (${execution_time}ms)"
        return 0
    else
        # Record failed rollback
        local record_error_sql="
            INSERT INTO migration_history (version, action, success, error_message) 
            VALUES ('$migration_version', 'rollback', FALSE, 'Rollback SQL execution failed');"
        
        postgres::database::execute "$instance_name" "$record_error_sql" "$database" 2>/dev/null
        
        log::error "Failed to rollback migration '$migration_version'"
        return 1
    fi
}

#######################################
# Show migration status
# Arguments:
#   $1 - instance name
#   $2 - database name (optional, default: $POSTGRES_DEFAULT_DB)
# Returns: 0 on success, 1 on failure
#######################################
postgres::migration::status() {
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
    
    # Check if migration system is initialized
    local check_table_sql="SELECT to_regclass('schema_migrations');"
    local table_exists=$(postgres::database::execute "$instance_name" "$check_table_sql" "$database" 2>/dev/null | tail -n +3 | head -n 1)
    
    if [[ -z "$table_exists" || "$table_exists" == "NULL" ]]; then
        log::warn "Migration system not initialized for instance '$instance_name'"
        log::info "Run: manage.sh --action migrate-init --instance $instance_name"
        return 1
    fi
    
    log::info "Migration Status for instance '$instance_name' database '$database':"
    log::info "================================================================"
    
    # Show applied migrations
    local status_sql="
        SELECT 
            version,
            description,
            applied_at,
            execution_time_ms
        FROM schema_migrations 
        ORDER BY applied_at DESC
        LIMIT 20;"
    
    log::info "Recent Applied Migrations:"
    printf "%-30s %-50s %-20s %s\\n" "Version" "Description" "Applied At" "Time (ms)"
    printf "%-30s %-50s %-20s %s\\n" "$(printf '%*s' 30 | tr ' ' '-')" "$(printf '%*s' 50 | tr ' ' '-')" "$(printf '%*s' 20 | tr ' ' '-')" "$(printf '%*s' 10 | tr ' ' '-')"
    
    postgres::database::execute "$instance_name" "$status_sql" "$database" | tail -n +3 | while IFS='|' read -r version description applied_at execution_time; do
        # Clean up the fields
        version=$(echo "$version" | tr -d ' ')
        description=$(echo "$description" | tr -d ' ' | cut -c1-48)
        applied_at=$(echo "$applied_at" | tr -d ' ')
        execution_time=$(echo "$execution_time" | tr -d ' ')
        
        printf "%-30s %-50s %-20s %s\\n" "$version" "$description" "$applied_at" "$execution_time"
    done
    
    # Show migration statistics
    local stats_sql="
        SELECT 
            COUNT(*) as total_migrations,
            COUNT(CASE WHEN applied_at > NOW() - INTERVAL '24 hours' THEN 1 END) as last_24h,
            AVG(execution_time_ms) as avg_time_ms
        FROM schema_migrations;"
    
    local stats=$(postgres::database::execute "$instance_name" "$stats_sql" "$database" 2>/dev/null | tail -n +3 | head -n 1)
    if [[ -n "$stats" ]]; then
        IFS='|' read -r total_migrations last_24h avg_time <<< "$stats"
        total_migrations=$(echo "$total_migrations" | tr -d ' ')
        last_24h=$(echo "$last_24h" | tr -d ' ')
        avg_time=$(echo "$avg_time" | tr -d ' ' | cut -d'.' -f1)
        
        log::info ""
        log::info "Migration Statistics:"
        log::info "  Total Migrations Applied: $total_migrations"
        log::info "  Applied in Last 24h: $last_24h"
        log::info "  Average Execution Time: ${avg_time}ms"
    fi
    
    return 0
}

#######################################
# Calculate checksum for migration file
# Arguments:
#   $1 - migration file path
# Returns: checksum string
#######################################
postgres::migration::calculate_checksum() {
    local migration_file="$1"
    
    if [[ -f "$migration_file" ]]; then
        # Use sha256sum if available, fallback to md5
        if command -v sha256sum >/dev/null 2>&1; then
            sha256sum "$migration_file" | cut -d' ' -f1
        elif command -v md5sum >/dev/null 2>&1; then
            md5sum "$migration_file" | cut -d' ' -f1
        else
            # Fallback to simple hash
            echo "$(stat -c%Y "$migration_file")_$(wc -c < "$migration_file")"
        fi
    else
        echo "file_not_found"
    fi
}

#######################################
# Extract description from migration file
# Arguments:
#   $1 - migration file path
# Returns: description string
#######################################
postgres::migration::extract_description() {
    local migration_file="$1"
    
    if [[ -f "$migration_file" ]]; then
        # Look for description in comments at the top of the file
        grep -E '^-- Description:' "$migration_file" | head -n 1 | sed 's/^-- Description: //'
    fi
}

#######################################
# Extract rollback SQL from migration file
# Arguments:
#   $1 - migration file path
# Returns: rollback SQL string
#######################################
postgres::migration::extract_rollback() {
    local migration_file="$1"
    
    if [[ -f "$migration_file" ]]; then
        # Look for rollback section in the file
        # Format: -- ROLLBACK: DROP TABLE example;
        grep -E '^-- ROLLBACK:' "$migration_file" | head -n 1 | sed 's/^-- ROLLBACK: //'
    fi
}

#######################################
# Validate migration file
# Arguments:
#   $1 - migration file path
# Returns: 0 if valid, 1 if invalid
#######################################
postgres::migration::validate_migration() {
    local migration_file="$1"
    
    if [[ ! -f "$migration_file" ]]; then
        log::error "Migration file not found: $migration_file"
        return 1
    fi
    
    # Check file is not empty
    if [[ ! -s "$migration_file" ]]; then
        log::error "Migration file is empty: $migration_file"
        return 1
    fi
    
    # Check for dangerous operations (basic validation)
    if grep -i "DROP DATABASE" "$migration_file" >/dev/null 2>&1; then
        log::error "Migration contains dangerous DROP DATABASE operation"
        return 1
    fi
    
    # Check for valid SQL syntax (basic check)
    if ! grep -E "^[[:space:]]*(CREATE|ALTER|INSERT|UPDATE|DELETE|SELECT)" "$migration_file" >/dev/null 2>&1; then
        log::warn "Migration file may not contain valid SQL statements"
    fi
    
    return 0
}

#######################################
# List available migrations in directory
# Arguments:
#   $1 - migrations directory path
# Returns: 0 on success, 1 on failure
#######################################
postgres::migration::list_available() {
    local migrations_dir="$1"
    
    if [[ -z "$migrations_dir" ]]; then
        log::error "Migrations directory is required"
        return 1
    fi
    
    if [[ ! -d "$migrations_dir" ]]; then
        log::error "Migrations directory not found: $migrations_dir"
        return 1
    fi
    
    log::info "Available Migrations in $migrations_dir:"
    log::info "=========================================="
    
    local migration_files=($(find "$migrations_dir" -name "*.sql" | sort))
    
    if [[ ${#migration_files[@]} -eq 0 ]]; then
        log::info "No migration files found"
        return 0
    fi
    
    printf "%-30s %-60s %-10s\\n" "Version" "Description" "Size"
    printf "%-30s %-60s %-10s\\n" "$(printf '%*s' 30 | tr ' ' '-')" "$(printf '%*s' 60 | tr ' ' '-')" "$(printf '%*s' 10 | tr ' ' '-')"
    
    for migration_file in "${migration_files[@]}"; do
        local migration_name=$(basename "$migration_file" .sql)
        local description=$(postgres::migration::extract_description "$migration_file")
        local file_size=$(ls -lh "$migration_file" | awk '{print $5}')
        
        if [[ -z "$description" ]]; then
            description="No description"
        fi
        
        printf "%-30s %-60s %-10s\\n" "$migration_name" "$description" "$file_size"
    done
    
    return 0
}