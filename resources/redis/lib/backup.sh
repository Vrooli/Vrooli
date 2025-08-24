#!/usr/bin/env bash
# Redis Backup and Restore Functions
# Functions for backing up and restoring Redis data

# Source var.sh to get proper directory variables
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*/../.." && builtin pwd)}"
_REDIS_BACKUP_DIR="$APP_ROOT/resources/redis/lib"
# shellcheck disable=SC1091
source "${APP_ROOT}/lib/utils/var.sh"

# Source shared secrets management library
# shellcheck disable=SC1091
source "${var_LIB_SERVICE_DIR}/secrets.sh"

#######################################
# Create Redis backup
# Arguments:
#   $1 - backup name (optional, defaults to timestamp)
# Returns: 0 on success, 1 on failure
#######################################
redis::backup::create() {
    local backup_name="${1:-redis-backup-$(date +%Y%m%d-%H%M%S)}"
    local project_config_dir
    project_config_dir="$(secrets::get_project_config_dir)"
    local backup_dir="${project_config_dir}/redis/backups"
    local backup_path="${backup_dir}/${backup_name}"
    
    if ! redis::common::is_running; then
        log::error "Redis is not running"
        return 1
    fi
    
    log::info "${MSG_BACKUP_START}"
    
    # Create backup directory
    mkdir -p "$backup_dir"
    
    # Create backup metadata
    cat > "${backup_path}.info" << EOF
{
  "name": "${backup_name}",
  "created": "$(date -Iseconds)",
  "redis_version": "$(redis::common::get_info | grep "redis_version:" | cut -d: -f2 | tr -d '\r' | xargs)",
  "redis_config": {
    "port": ${REDIS_PORT},
    "databases": ${REDIS_DATABASES},
    "max_memory": "${REDIS_MAX_MEMORY}",
    "persistence": "${REDIS_PERSISTENCE}"
  }
}
EOF
    
    # Method 1: Use BGSAVE for RDB backup
    if redis::backup::create_rdb "$backup_path"; then
        log::success "${MSG_BACKUP_SUCCESS}"
        log::info "${MSG_BACKUP_LOCATION}${backup_path}.rdb"
        return 0
    fi
    
    # Method 2: Fallback to data directory copy
    log::warn "RDB backup failed, trying data directory copy..."
    if redis::backup::copy_data_dir "$backup_path"; then
        log::success "${MSG_BACKUP_SUCCESS}"
        log::info "${MSG_BACKUP_LOCATION}${backup_path}"
        return 0
    fi
    
    log::error "${MSG_BACKUP_FAILED}"
    return 1
}

#######################################
# Create RDB backup using BGSAVE
# Arguments:
#   $1 - backup path (without extension)
# Returns: 0 on success, 1 on failure
#######################################
redis::backup::create_rdb() {
    local backup_path="$1"
    
    # Trigger background save
    local cmd=(docker exec "${REDIS_CONTAINER_NAME}" redis-cli)
    if [[ -n "$REDIS_PASSWORD" ]]; then
        cmd+=(-a "$REDIS_PASSWORD")
    fi
    cmd+=(BGSAVE)
    
    if ! "${cmd[@]}" >/dev/null 2>&1; then
        log::debug "BGSAVE command failed"
        return 1
    fi
    
    # Wait for background save to complete
    local max_wait=60
    local elapsed=0
    
    while [ $elapsed -lt $max_wait ]; do
        local cmd_check=(docker exec "${REDIS_CONTAINER_NAME}" redis-cli)
        if [[ -n "$REDIS_PASSWORD" ]]; then
            cmd_check+=(-a "$REDIS_PASSWORD")
        fi
        cmd_check+=(LASTSAVE)
        
        local current_save
        current_save=$("${cmd_check[@]}" 2>/dev/null)
        
        sleep 1
        local new_save
        new_save=$("${cmd_check[@]}" 2>/dev/null)
        
        if [[ "$new_save" != "$current_save" ]]; then
            log::debug "Background save completed after ${elapsed}s"
            break
        fi
        
        elapsed=$((elapsed + 1))
    done
    
    if [ $elapsed -ge $max_wait ]; then
        log::debug "Background save timed out"
        return 1
    fi
    
    # Copy the RDB file
    if docker cp "${REDIS_CONTAINER_NAME}:/data/dump.rdb" "${backup_path}.rdb" 2>/dev/null; then
        log::debug "RDB file copied successfully"
        return 0
    else
        log::debug "Failed to copy RDB file"
        return 1
    fi
}

#######################################
# Copy data directory for backup
# Arguments:
#   $1 - backup path
# Returns: 0 on success, 1 on failure
#######################################
redis::backup::copy_data_dir() {
    local backup_path="$1"
    
    # Create a tar archive of the data directory from the container
    if docker exec "${REDIS_CONTAINER_NAME}" tar -czf - -C /data . > "${backup_path}.tar.gz" 2>/dev/null; then
        log::debug "Data directory archived successfully"
        return 0
    else
        log::debug "Failed to archive data directory"
        return 1
    fi
}

#######################################
# Restore Redis from backup
# Arguments:
#   $1 - backup name or path
# Returns: 0 on success, 1 on failure
#######################################
redis::backup::restore() {
    local backup_identifier="$1"
    
    if [[ -z "$backup_identifier" ]]; then
        log::error "${MSG_ERROR_MISSING_PARAM}backup_identifier"
        return 1
    fi
    
    local backup_path
    local project_config_dir
    project_config_dir="$(secrets::get_project_config_dir)"
    local backup_dir="${project_config_dir}/redis/backups"
    
    # Determine backup path
    if [[ -f "$backup_identifier" ]]; then
        backup_path="$backup_identifier"
    elif [[ -f "${backup_dir}/${backup_identifier}.rdb" ]]; then
        backup_path="${backup_dir}/${backup_identifier}.rdb"
    elif [[ -f "${backup_dir}/${backup_identifier}.tar.gz" ]]; then
        backup_path="${backup_dir}/${backup_identifier}.tar.gz"
    else
        log::error "${MSG_RESTORE_FILE_NOT_FOUND}${backup_identifier}"
        return 1
    fi
    
    log::info "${MSG_RESTORE_START}"
    log::warn "${MSG_WARN_DATA_LOSS}"
    
    # Confirm restoration
    if [[ "$YES" != "yes" ]]; then
        echo -n "Are you sure you want to restore? This will overwrite existing data! (yes/no): "
        read -r confirm
        if [[ "$confirm" != "yes" ]]; then
            log::info "Restore cancelled"
            return 0
        fi
    fi
    
    # Stop Redis temporarily
    local was_running=false
    if redis::common::is_running; then
        was_running=true
        redis::docker::stop
    fi
    
    # Restore based on file type
    if [[ "$backup_path" == *.rdb ]]; then
        redis::backup::restore_rdb "$backup_path"
    elif [[ "$backup_path" == *.tar.gz ]]; then
        redis::backup::restore_tar "$backup_path"
    else
        log::error "Unknown backup format: $backup_path"
        return 1
    fi
    
    local restore_result=$?
    
    # Restart Redis if it was running
    if [[ "$was_running" == true ]]; then
        redis::docker::start
    fi
    
    if [[ $restore_result -eq 0 ]]; then
        log::success "${MSG_RESTORE_SUCCESS}"
        return 0
    else
        log::error "${MSG_RESTORE_FAILED}"
        return 1
    fi
}

#######################################
# Restore from RDB file
# Arguments:
#   $1 - RDB backup path
# Returns: 0 on success, 1 on failure
#######################################
redis::backup::restore_rdb() {
    local backup_path="$1"
    
    # Copy RDB file to container's data directory
    if docker cp "$backup_path" "${REDIS_CONTAINER_NAME}:/data/dump.rdb"; then
        log::debug "RDB file restored successfully"
        return 0
    else
        log::debug "Failed to restore RDB file"
        return 1
    fi
}

#######################################
# Restore from tar archive
# Arguments:
#   $1 - tar backup path
# Returns: 0 on success, 1 on failure
#######################################
redis::backup::restore_tar() {
    local backup_path="$1"
    
    # Clear existing data in container
    # Note: Using rm -rf inside container as trash system not available in Redis container
    docker exec "${REDIS_CONTAINER_NAME}" sh -c 'rm -rf /data/*'
    
    # Extract tar archive to container
    if cat "$backup_path" | docker exec -i "${REDIS_CONTAINER_NAME}" tar -xzf - -C /data; then
        log::debug "Data directory restored successfully"
        return 0
    else
        log::debug "Failed to restore data directory"
        return 1
    fi
}

#######################################
# List available backups
#######################################
redis::backup::list() {
    local project_config_dir
    project_config_dir="$(secrets::get_project_config_dir)"
    local backup_dir="${project_config_dir}/redis/backups"
    
    if [[ ! -d "$backup_dir" ]]; then
        log::info "No backups found (backup directory doesn't exist)"
        return 0
    fi
    
    log::info "📦 Available Redis Backups:"
    
    local found_backups=false
    
    # List RDB backups
    for backup in "$backup_dir"/*.rdb; do
        if [[ -f "$backup" ]]; then
            found_backups=true
            local backup_name
            backup_name=$(basename "$backup" .rdb)
            local backup_size
            backup_size=$(du -h "$backup" | cut -f1)
            local backup_date
            backup_date=$(stat -c %y "$backup" | cut -d. -f1)
            
            echo "   📄 ${backup_name} (RDB)"
            echo "      Size: ${backup_size}"
            echo "      Date: ${backup_date}"
            
            # Show metadata if available
            local info_file="${backup%%.rdb}.info"
            if [[ -f "$info_file" ]]; then
                local redis_version
                redis_version=$(jq -r '.redis_version // "unknown"' "$info_file" 2>/dev/null)
                echo "      Redis Version: ${redis_version}"
            fi
            echo
        fi
    done
    
    # List tar backups
    for backup in "$backup_dir"/*.tar.gz; do
        if [[ -f "$backup" ]]; then
            found_backups=true
            local backup_name
            backup_name=$(basename "$backup" .tar.gz)
            local backup_size
            backup_size=$(du -h "$backup" | cut -f1)
            local backup_date
            backup_date=$(stat -c %y "$backup" | cut -d. -f1)
            
            echo "   📦 ${backup_name} (Archive)"
            echo "      Size: ${backup_size}"
            echo "      Date: ${backup_date}"
            echo
        fi
    done
    
    if [[ "$found_backups" == false ]]; then
        log::info "   No backups found"
    fi
}

#######################################
# Delete backup
# Arguments:
#   $1 - backup name
# Returns: 0 on success, 1 on failure
#######################################
redis::backup::delete() {
    local backup_name="$1"
    
    if [[ -z "$backup_name" ]]; then
        log::error "${MSG_ERROR_MISSING_PARAM}backup_name"
        return 1
    fi
    
    local project_config_dir
    project_config_dir="$(secrets::get_project_config_dir)"
    local backup_dir="${project_config_dir}/redis/backups"
    local deleted_any=false
    
    # Delete RDB backup
    if [[ -f "${backup_dir}/${backup_name}.rdb" ]]; then
        rm -f "${backup_dir}/${backup_name}.rdb"
        rm -f "${backup_dir}/${backup_name}.info"
        deleted_any=true
        log::debug "Deleted RDB backup: ${backup_name}"
    fi
    
    # Delete tar backup
    if [[ -f "${backup_dir}/${backup_name}.tar.gz" ]]; then
        rm -f "${backup_dir}/${backup_name}.tar.gz"
        rm -f "${backup_dir}/${backup_name}.info"
        deleted_any=true
        log::debug "Deleted archive backup: ${backup_name}"
    fi
    
    if [[ "$deleted_any" == true ]]; then
        log::success "Backup deleted: ${backup_name}"
        return 0
    else
        log::error "Backup not found: ${backup_name}"
        return 1
    fi
}

#######################################
# Clean old backups
# Arguments:
#   $1 - days to keep (default: 30)
# Returns: 0 on success
#######################################
redis::backup::cleanup() {
    local days_to_keep="${1:-30}"
    local project_config_dir
    project_config_dir="$(secrets::get_project_config_dir)"
    local backup_dir="${project_config_dir}/redis/backups"
    
    if [[ ! -d "$backup_dir" ]]; then
        log::info "No backup directory to clean"
        return 0
    fi
    
    log::info "🧹 Cleaning backups older than ${days_to_keep} days..."
    
    local deleted_count=0
    
    # Find and delete old backups
    while IFS= read -r -d '' file; do
        rm -f "$file"
        # Also remove associated .info file
        rm -f "${file%.*}.info"
        deleted_count=$((deleted_count + 1))
        log::debug "Deleted old backup: $(basename "$file")"
    done < <(find "$backup_dir" -name "*.rdb" -o -name "*.tar.gz" -mtime +$days_to_keep -print0 2>/dev/null)
    
    if [[ $deleted_count -gt 0 ]]; then
        log::success "Deleted ${deleted_count} old backups"
    else
        log::info "No old backups to delete"
    fi
    
    return 0
}