#!/usr/bin/env bash
# PostgreSQL Backup and Restore Operations
# Functions for backing up and restoring PostgreSQL instance data

# Source required utilities
POSTGRES_LIB_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
# shellcheck disable=SC1091
source "${POSTGRES_LIB_DIR}/../../../lib/utils/var.sh" 2>/dev/null || true
# shellcheck disable=SC1091
source "${var_LIB_SYSTEM_DIR}/trash.sh" 2>/dev/null || true

#######################################
# Create backup of PostgreSQL instance
# Arguments:
#   $1 - instance name
#   $2 - backup name (optional, defaults to timestamp)
#   $3 - backup type (optional: "full", "schema", "data", default: "full")
# Returns: 0 on success, 1 on failure
#######################################
postgres::backup::create() {
    local instance_name="$1"
    local backup_name="${2:-$(date +%Y%m%d_%H%M%S)}"
    local backup_type="${3:-full}"
    
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
    
    # Validate backup type
    case "$backup_type" in
        "full"|"schema"|"data") ;;
        *)
            log::error "Invalid backup type: $backup_type (must be 'full', 'schema', or 'data')"
            return 1
            ;;
    esac
    
    # Create backup directory structure
    local backup_dir="${POSTGRES_BACKUP_DIR}/${instance_name}"
    local backup_path="${backup_dir}/${backup_name}"
    mkdir -p "$backup_path"
    
    log::info "Creating $backup_type backup of instance '$instance_name'..."
    log::info "Backup path: $backup_path"
    
    local container_name="${POSTGRES_CONTAINER_PREFIX}-${instance_name}"
    local database="${POSTGRES_DEFAULT_DB}"
    local success=true
    
    # Create backup metadata
    postgres::backup::create_metadata "$instance_name" "$backup_name" "$backup_type" "$backup_path"
    
    case "$backup_type" in
        "full")
            # Full backup (schema + data)
            if ! docker exec "$container_name" pg_dump \
                -U "$POSTGRES_DEFAULT_USER" \
                -d "$database" \
                --verbose \
                --no-owner \
                --no-privileges \
                --file="/tmp/backup.sql" 2>/dev/null; then
                success=false
            else
                docker cp "$container_name:/tmp/backup.sql" "$backup_path/full.sql"
                docker exec "$container_name" rm -f "/tmp/backup.sql"
            fi
            ;;
        "schema")
            # Schema-only backup
            if ! docker exec "$container_name" pg_dump \
                -U "$POSTGRES_DEFAULT_USER" \
                -d "$database" \
                --schema-only \
                --verbose \
                --no-owner \
                --no-privileges \
                --file="/tmp/schema.sql" 2>/dev/null; then
                success=false
            else
                docker cp "$container_name:/tmp/schema.sql" "$backup_path/schema.sql"
                docker exec "$container_name" rm -f "/tmp/schema.sql"
            fi
            ;;
        "data")
            # Data-only backup
            if ! docker exec "$container_name" pg_dump \
                -U "$POSTGRES_DEFAULT_USER" \
                -d "$database" \
                --data-only \
                --verbose \
                --no-owner \
                --no-privileges \
                --file="/tmp/data.sql" 2>/dev/null; then
                success=false
            else
                docker cp "$container_name:/tmp/data.sql" "$backup_path/data.sql"
                docker exec "$container_name" rm -f "/tmp/data.sql"
            fi
            ;;
    esac
    
    # Copy instance configuration
    local instance_config="${POSTGRES_INSTANCES_DIR}/${instance_name}/config.json"
    if [[ -f "$instance_config" ]]; then
        cp "$instance_config" "$backup_path/config.json"
    fi
    
    # Create backup summary
    if [[ "$success" == "true" ]]; then
        postgres::backup::finalize_metadata "$backup_path" "success"
        log::success "Backup created successfully: $backup_name"
        log::info "Backup location: $backup_path"
        
        # Show backup size
        local backup_size=$(du -sh "$backup_path" 2>/dev/null | cut -f1 || echo "N/A")
        log::info "Backup size: $backup_size"
        
        return 0
    else
        postgres::backup::finalize_metadata "$backup_path" "failed"
        log::error "Backup creation failed"
        return 1
    fi
}

#######################################
# Restore PostgreSQL instance from backup
# Arguments:
#   $1 - instance name
#   $2 - backup name
#   $3 - restore type (optional: "full", "schema", "data", default: "full")
#   $4 - force flag (optional, "yes" to skip confirmation)
# Returns: 0 on success, 1 on failure
#######################################
postgres::backup::restore() {
    local instance_name="$1"
    local backup_name="$2"
    local restore_type="${3:-full}"
    local force="${4:-no}"
    
    if [[ -z "$instance_name" || -z "$backup_name" ]]; then
        log::error "Instance name and backup name are required"
        return 1
    fi
    
    local backup_path="${POSTGRES_BACKUP_DIR}/${instance_name}/${backup_name}"
    
    if [[ ! -d "$backup_path" ]]; then
        log::error "Backup not found: $backup_path"
        return 1
    fi
    
    # Validate restore type
    case "$restore_type" in
        "full"|"schema"|"data") ;;
        *)
            log::error "Invalid restore type: $restore_type (must be 'full', 'schema', or 'data')"
            return 1
            ;;
    esac
    
    # Check backup metadata
    local metadata_file="$backup_path/metadata.json"
    if [[ -f "$metadata_file" ]]; then
        local backup_status=$(grep '"status"' "$metadata_file" | cut -d'"' -f4)
        if [[ "$backup_status" != "success" ]]; then
            log::warn "Backup status is '$backup_status', restore may fail"
        fi
    fi
    
    # Confirmation unless forced
    if [[ "$force" != "yes" ]]; then
        log::warn "This will restore instance '$instance_name' from backup '$backup_name'"
        log::warn "All current data in the instance will be replaced!"
        read -p "Are you sure? (y/N) " -n 1 -r
        echo
        if [[ ! $REPLY =~ ^[Yy]$ ]]; then
            log::info "Restore cancelled"
            return 0
        fi
    fi
    
    # Check if instance exists and is running
    local instance_running=false
    if postgres::common::container_exists "$instance_name"; then
        if postgres::common::is_running "$instance_name"; then
            instance_running=true
            log::info "Stopping instance '$instance_name' for restore..."
            postgres::instance::stop "$instance_name"
        fi
    else
        log::error "Instance '$instance_name' does not exist"
        return 1
    fi
    
    log::info "Restoring instance '$instance_name' from backup '$backup_name'..."
    
    # Start instance for restore
    if ! postgres::instance::start "$instance_name"; then
        log::error "Failed to start instance for restore"
        return 1
    fi
    
    # Wait for instance to be ready
    sleep "$POSTGRES_INITIALIZATION_WAIT"
    
    local container_name="${POSTGRES_CONTAINER_PREFIX}-${instance_name}"
    local database="${POSTGRES_DEFAULT_DB}"
    local success=true
    
    case "$restore_type" in
        "full")
            local backup_file="$backup_path/full.sql"
            if [[ ! -f "$backup_file" ]]; then
                log::error "Full backup file not found: $backup_file"
                success=false
            else
                # Drop and recreate database for clean restore
                postgres::database::execute "$instance_name" "DROP DATABASE IF EXISTS \"$database\";" "postgres"
                postgres::database::execute "$instance_name" "CREATE DATABASE \"$database\" OWNER \"$POSTGRES_DEFAULT_USER\";" "postgres"
                
                # Restore full backup
                docker cp "$backup_file" "$container_name:/tmp/restore.sql"
                if ! docker exec "$container_name" psql \
                    -U "$POSTGRES_DEFAULT_USER" \
                    -d "$database" \
                    -f "/tmp/restore.sql" 2>/dev/null; then
                    success=false
                fi
                docker exec "$container_name" rm -f "/tmp/restore.sql"
            fi
            ;;
        "schema")
            local backup_file="$backup_path/schema.sql"
            if [[ ! -f "$backup_file" ]]; then
                log::error "Schema backup file not found: $backup_file"
                success=false
            else
                # Restore schema backup
                docker cp "$backup_file" "$container_name:/tmp/restore_schema.sql"
                if ! docker exec "$container_name" psql \
                    -U "$POSTGRES_DEFAULT_USER" \
                    -d "$database" \
                    -f "/tmp/restore_schema.sql" 2>/dev/null; then
                    success=false
                fi
                docker exec "$container_name" rm -f "/tmp/restore_schema.sql"
            fi
            ;;
        "data")
            local backup_file="$backup_path/data.sql"
            if [[ ! -f "$backup_file" ]]; then
                log::error "Data backup file not found: $backup_file"
                success=false
            else
                # Restore data backup
                docker cp "$backup_file" "$container_name:/tmp/restore_data.sql"
                if ! docker exec "$container_name" psql \
                    -U "$POSTGRES_DEFAULT_USER" \
                    -d "$database" \
                    -f "/tmp/restore_data.sql" 2>/dev/null; then
                    success=false
                fi
                docker exec "$container_name" rm -f "/tmp/restore_data.sql"
            fi
            ;;
    esac
    
    # Restore instance configuration if available
    local config_backup="$backup_path/config.json"
    if [[ -f "$config_backup" ]]; then
        local instance_config="${POSTGRES_INSTANCES_DIR}/${instance_name}/config.json"
        cp "$config_backup" "$instance_config"
    fi
    
    if [[ "$success" == "true" ]]; then
        log::success "Restore completed successfully"
        
        # Restart instance if it was running before
        if [[ "$instance_running" == "true" ]]; then
            postgres::instance::restart "$instance_name"
        fi
        
        return 0
    else
        log::error "Restore failed"
        return 1
    fi
}

#######################################
# List available backups for instance
# Arguments:
#   $1 - instance name (optional, lists all if not specified)
# Returns: 0 on success, 1 on failure
#######################################
postgres::backup::list() {
    local instance_name="${1:-}"
    
    if [[ -n "$instance_name" ]]; then
        # List backups for specific instance
        local backup_dir="${POSTGRES_BACKUP_DIR}/${instance_name}"
        
        if [[ ! -d "$backup_dir" ]]; then
            log::warn "No backups found for instance '$instance_name'"
            return 0
        fi
        
        log::info "Backups for instance '$instance_name':"
        log::info "===================================="
        
        local backups=($(find "$backup_dir" -maxdepth 1 -type d ! -path "$backup_dir" | sort -r))
        
        if [[ ${#backups[@]} -eq 0 ]]; then
            log::info "No backups found"
            return 0
        fi
        
        printf "%-20s %-10s %-8s %-12s %s\\n" "Backup Name" "Type" "Status" "Size" "Created"
        printf "%-20s %-10s %-8s %-12s %s\\n" "------------" "----" "------" "----" "-------"
        
        for backup_path in "${backups[@]}"; do
            postgres::backup::show_backup_info "$backup_path"
        done
    else
        # List all backups
        if [[ ! -d "$POSTGRES_BACKUP_DIR" ]]; then
            log::warn "No backup directory found"
            return 0
        fi
        
        log::info "All PostgreSQL Instance Backups:"
        log::info "================================"
        
        local instances=($(find "$POSTGRES_BACKUP_DIR" -maxdepth 1 -type d -name "*" | grep -v "^${POSTGRES_BACKUP_DIR}$" | sort))
        
        if [[ ${#instances[@]} -eq 0 ]]; then
            log::info "No backups found"
            return 0
        fi
        
        for instance_dir in "${instances[@]}"; do
            local instance_name=$(basename "$instance_dir")
            log::info ""
            log::info "Instance: $instance_name"
            log::info "$(printf '%*s' ${#instance_name} | tr ' ' '-')--------"
            
            local backups=($(find "$instance_dir" -maxdepth 1 -type d ! -path "$instance_dir" | sort -r))
            
            if [[ ${#backups[@]} -eq 0 ]]; then
                log::info "  No backups found"
            else
                printf "  %-18s %-10s %-8s %-12s %s\\n" "Backup Name" "Type" "Status" "Size" "Created"
                printf "  %-18s %-10s %-8s %-12s %s\\n" "----------" "----" "------" "----" "-------"
                
                for backup_path in "${backups[@]}"; do
                    printf "  "
                    postgres::backup::show_backup_info "$backup_path"
                done
            fi
        done
    fi
    
    return 0
}

#######################################
# Show backup information
# Arguments:
#   $1 - backup path
#######################################
postgres::backup::show_backup_info() {
    local backup_path="$1"
    local backup_name=$(basename "$backup_path")
    
    # Default values
    local backup_type="unknown"
    local backup_status="unknown"
    local backup_size="N/A"
    local created_date="N/A"
    
    # Read metadata if available
    local metadata_file="$backup_path/metadata.json"
    if [[ -f "$metadata_file" ]]; then
        backup_type=$(grep '"type"' "$metadata_file" | cut -d'"' -f4 2>/dev/null || echo "unknown")
        backup_status=$(grep '"status"' "$metadata_file" | cut -d'"' -f4 2>/dev/null || echo "unknown")
        created_date=$(grep '"created"' "$metadata_file" | cut -d'"' -f4 2>/dev/null || echo "N/A")
    fi
    
    # Get backup size
    if [[ -d "$backup_path" ]]; then
        backup_size=$(du -sh "$backup_path" 2>/dev/null | cut -f1 || echo "N/A")
    fi
    
    # Format created date for display
    if [[ "$created_date" != "N/A" ]]; then
        created_date=$(date -d "$created_date" "+%Y-%m-%d %H:%M" 2>/dev/null || echo "$created_date")
    fi
    
    printf "%-20s %-10s %-8s %-12s %s\\n" "$backup_name" "$backup_type" "$backup_status" "$backup_size" "$created_date"
}

#######################################
# Delete backup
# Arguments:
#   $1 - instance name
#   $2 - backup name
#   $3 - force flag (optional, "yes" to skip confirmation)
# Returns: 0 on success, 1 on failure
#######################################
postgres::backup::delete() {
    local instance_name="$1"
    local backup_name="$2"
    local force="${3:-no}"
    
    if [[ -z "$instance_name" || -z "$backup_name" ]]; then
        log::error "Instance name and backup name are required"
        return 1
    fi
    
    local backup_path="${POSTGRES_BACKUP_DIR}/${instance_name}/${backup_name}"
    
    if [[ ! -d "$backup_path" ]]; then
        log::error "Backup not found: $backup_path"
        return 1
    fi
    
    # Confirmation unless forced
    if [[ "$force" != "yes" ]]; then
        log::warn "This will permanently delete backup '$backup_name' for instance '$instance_name'"
        read -p "Are you sure? (y/N) " -n 1 -r
        echo
        if [[ ! $REPLY =~ ^[Yy]$ ]]; then
            log::info "Backup deletion cancelled"
            return 0
        fi
    fi
    
    log::info "Deleting backup '$backup_name'..."
    
    if trash::safe_remove "$backup_path" --production; then
        log::success "Backup deleted successfully"
        return 0
    else
        log::error "Failed to delete backup"
        return 1
    fi
}

#######################################
# Clean up old backups based on retention policy
# Arguments:
#   $1 - instance name (optional, cleans all if not specified)
#   $2 - retention days (optional, uses default if not specified)
# Returns: 0 on success, 1 on failure
#######################################
postgres::backup::cleanup() {
    local instance_name="${1:-}"
    local retention_days="${2:-$POSTGRES_BACKUP_RETENTION_DAYS}"
    
    log::info "Cleaning up backups older than $retention_days days..."
    
    local deleted_count=0
    local base_dir="$POSTGRES_BACKUP_DIR"
    
    if [[ -n "$instance_name" ]]; then
        base_dir="$POSTGRES_BACKUP_DIR/$instance_name"
        if [[ ! -d "$base_dir" ]]; then
            log::info "No backups found for instance '$instance_name'"
            return 0
        fi
    fi
    
    if [[ ! -d "$base_dir" ]]; then
        log::info "No backup directory found"
        return 0
    fi
    
    # Find and delete old backups
    while IFS= read -r -d '' backup_path; do
        local backup_name=$(basename "$backup_path")
        
        # Skip if not a backup directory (should contain backup files or metadata)
        if [[ ! -f "$backup_path/metadata.json" && ! -f "$backup_path/full.sql" && ! -f "$backup_path/schema.sql" && ! -f "$backup_path/data.sql" ]]; then
            continue
        fi
        
        # Check if backup is older than retention period
        if [[ $(find "$backup_path" -maxdepth 0 -mtime +$retention_days) ]]; then
            log::info "Removing old backup: $backup_name"
            if trash::safe_remove "$backup_path" --production; then
                ((deleted_count++))
            else
                log::warn "Failed to remove backup: $backup_name"
            fi
        fi
    done < <(find "$base_dir" -maxdepth 2 -type d -print0)
    
    if [[ $deleted_count -eq 0 ]]; then
        log::info "No old backups found to clean up"
    else
        log::success "Cleaned up $deleted_count old backup(s)"
    fi
    
    return 0
}

#######################################
# Verify backup integrity
# Arguments:
#   $1 - instance name
#   $2 - backup name
# Returns: 0 if valid, 1 if invalid
#######################################
postgres::backup::verify() {
    local instance_name="$1"
    local backup_name="$2"
    
    if [[ -z "$instance_name" || -z "$backup_name" ]]; then
        log::error "Instance name and backup name are required"
        return 1
    fi
    
    local backup_path="${POSTGRES_BACKUP_DIR}/${instance_name}/${backup_name}"
    
    if [[ ! -d "$backup_path" ]]; then
        log::error "Backup not found: $backup_path"
        return 1
    fi
    
    log::info "Verifying backup '$backup_name' for instance '$instance_name'..."
    
    local issues=0
    
    # Check metadata file
    local metadata_file="$backup_path/metadata.json"
    if [[ ! -f "$metadata_file" ]]; then
        log::error "Missing metadata file: $metadata_file"
        ((issues++))
    else
        # Verify metadata structure
        if ! grep -q '"instance"' "$metadata_file" || \
           ! grep -q '"backup_name"' "$metadata_file" || \
           ! grep -q '"type"' "$metadata_file"; then
            log::error "Invalid metadata format"
            ((issues++))
        fi
    fi
    
    # Check for backup files based on type
    if [[ -f "$metadata_file" ]]; then
        local backup_type=$(grep '"type"' "$metadata_file" | cut -d'"' -f4 2>/dev/null)
        
        case "$backup_type" in
            "full")
                if [[ ! -f "$backup_path/full.sql" ]]; then
                    log::error "Missing full backup file: full.sql"
                    ((issues++))
                fi
                ;;
            "schema")
                if [[ ! -f "$backup_path/schema.sql" ]]; then
                    log::error "Missing schema backup file: schema.sql"
                    ((issues++))
                fi
                ;;
            "data")
                if [[ ! -f "$backup_path/data.sql" ]]; then
                    log::error "Missing data backup file: data.sql"
                    ((issues++))
                fi
                ;;
        esac
    fi
    
    # Check file sizes (basic sanity check)
    for sql_file in "$backup_path"/*.sql; do
        if [[ -f "$sql_file" ]]; then
            local file_size=$(stat -f%z "$sql_file" 2>/dev/null || stat -c%s "$sql_file" 2>/dev/null || echo 0)
            if [[ $file_size -eq 0 ]]; then
                log::error "Empty backup file: $(basename "$sql_file")"
                ((issues++))
            fi
        fi
    done
    
    # Summary
    if [[ $issues -eq 0 ]]; then
        log::success "Backup verification passed"
        return 0
    else
        log::error "Backup verification failed: $issues issue(s) found"
        return 1
    fi
}

#######################################
# Create backup metadata file
# Arguments:
#   $1 - instance name
#   $2 - backup name
#   $3 - backup type
#   $4 - backup path
#######################################
postgres::backup::create_metadata() {
    local instance_name="$1"
    local backup_name="$2"
    local backup_type="$3"
    local backup_path="$4"
    
    local metadata_file="$backup_path/metadata.json"
    
    # Get instance configuration
    local port=$(postgres::common::get_instance_config "$instance_name" "port")
    local template=$(postgres::common::get_instance_config "$instance_name" "template")
    
    cat > "$metadata_file" << EOF
{
    "instance": "$instance_name",
    "backup_name": "$backup_name",
    "type": "$backup_type",
    "status": "in_progress",
    "created": "$(date -Iseconds)",
    "postgres_version": "$POSTGRES_IMAGE",
    "instance_config": {
        "port": "$port",
        "template": "$template"
    }
}
EOF
}

#######################################
# Finalize backup metadata
# Arguments:
#   $1 - backup path
#   $2 - status ("success" or "failed")
#######################################
postgres::backup::finalize_metadata() {
    local backup_path="$1"
    local status="$2"
    
    local metadata_file="$backup_path/metadata.json"
    
    if [[ -f "$metadata_file" ]]; then
        # Update status and add completion time
        local temp_file=$(mktemp)
        
        # Replace the in_progress status and add completion time before the closing brace
        sed "s/\"status\": \"in_progress\"/\"status\": \"$status\",\\n    \"completed\": \"$(date -Iseconds)\"/" "$metadata_file" > "$temp_file"
        
        mv "$temp_file" "$metadata_file"
    fi
}