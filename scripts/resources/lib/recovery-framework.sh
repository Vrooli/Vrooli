#!/usr/bin/env bash
# Generic Recovery Framework
# Provides automated recovery operations for all resources

# Source required utilities
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck disable=SC1091
source "${SCRIPT_DIR}/../../lib/utils/var.sh"
# shellcheck disable=SC1091
source "${SCRIPT_DIR}/../../lib/utils/log.sh"
# shellcheck disable=SC1091
source "${SCRIPT_DIR}/docker-utils.sh"
# shellcheck disable=SC1091
source "${SCRIPT_DIR}/health-framework.sh"

#######################################
# Automatically recover from corruption using backups
# Args: $1 - recovery_config (JSON configuration)
# Returns: 0 on success, 1 on failure
#
# Recovery Config Schema:
# {
#   "container_name": "container-name",
#   "data_dir": "/path/to/data",
#   "backup_dir": "/path/to/backups",
#   "backup_pattern": "*.db",
#   "stop_func": "function_to_stop_service",
#   "start_func": "function_to_start_service",
#   "owner": "1000:1000"
# }
#######################################
recovery::auto_recover() {
    local recovery_config="$1"
    
    local container_name
    local data_dir
    local backup_dir
    local backup_pattern
    local stop_func
    local start_func
    local owner
    
    container_name=$(echo "$recovery_config" | jq -r '.container_name // empty')
    data_dir=$(echo "$recovery_config" | jq -r '.data_dir // empty')
    backup_dir=$(echo "$recovery_config" | jq -r '.backup_dir // empty')
    backup_pattern=$(echo "$recovery_config" | jq -r '.backup_pattern // "*"')
    stop_func=$(echo "$recovery_config" | jq -r '.stop_func // empty')
    start_func=$(echo "$recovery_config" | jq -r '.start_func // empty')
    owner=$(echo "$recovery_config" | jq -r '.owner // "1000:1000"')
    
    log::info "Starting automatic recovery for $container_name..."
    
    # Stop the service if running
    if [[ -n "$stop_func" ]] && command -v "$stop_func" &>/dev/null; then
        log::info "Stopping service..."
        "$stop_func"
    elif [[ -n "$container_name" ]] && docker::is_running "$container_name"; then
        log::info "Stopping container..."
        docker stop "$container_name" >/dev/null 2>&1
    fi
    
    # Find best backup
    local backup_file
    backup_file=$(recovery::find_best_backup "$backup_dir" "$backup_pattern")
    
    if [[ -z "$backup_file" ]]; then
        log::error "No backup found for recovery"
        return 1
    fi
    
    log::info "Found backup: $backup_file"
    
    # Recreate data directory
    if ! recovery::recreate_data_dir "$data_dir" "$owner"; then
        log::error "Failed to recreate data directory"
        return 1
    fi
    
    # Restore from backup
    if ! recovery::restore_backup "$backup_file" "$data_dir"; then
        log::error "Failed to restore backup"
        return 1
    fi
    
    # Fix permissions
    recovery::fix_permissions "$data_dir" "$owner"
    
    # Start the service
    if [[ -n "$start_func" ]] && command -v "$start_func" &>/dev/null; then
        log::info "Starting service..."
        "$start_func"
    elif [[ -n "$container_name" ]]; then
        log::info "Starting container..."
        docker start "$container_name" >/dev/null 2>&1
    fi
    
    log::success "Recovery completed successfully"
    return 0
}

#######################################
# Find the best backup file based on size and recency
# Args: $1 - backup_directory
#       $2 - file_pattern (default: "*")
# Returns: Path to best backup or empty string
#######################################
recovery::find_best_backup() {
    local backup_dir="$1"
    local pattern="${2:-*}"
    
    if [[ ! -d "$backup_dir" ]]; then
        return
    fi
    
    # Find backups sorted by size (prefer larger) and modification time (prefer newer)
    find "$backup_dir" -name "$pattern" -type f -size +0 2>/dev/null | \
        xargs -r ls -lt 2>/dev/null | \
        head -1 | \
        awk '{print $NF}'
}

#######################################
# Recreate data directory safely
# Args: $1 - data_directory
#       $2 - owner (user:group)
# Returns: 0 on success, 1 on failure
#######################################
recovery::recreate_data_dir() {
    local data_dir="$1"
    local owner="${2:-1000:1000}"
    
    log::info "Recreating data directory: $data_dir"
    
    # Backup existing directory if it exists
    if [[ -d "$data_dir" ]]; then
        local backup_name="${data_dir}.corrupted.$(date +%Y%m%d_%H%M%S)"
        log::warn "Moving corrupted directory to: $backup_name"
        mv "$data_dir" "$backup_name" || return 1
    fi
    
    # Create fresh directory
    mkdir -p "$data_dir" || return 1
    
    # Set ownership
    if [[ -n "$owner" ]] && [[ "$owner" != ":" ]]; then
        chown -R "$owner" "$data_dir" 2>/dev/null || true
    fi
    
    return 0
}

#######################################
# Restore backup to data directory
# Args: $1 - backup_file
#       $2 - data_directory
# Returns: 0 on success, 1 on failure
#######################################
recovery::restore_backup() {
    local backup_file="$1"
    local data_dir="$2"
    
    if [[ ! -f "$backup_file" ]]; then
        log::error "Backup file not found: $backup_file"
        return 1
    fi
    
    log::info "Restoring from backup: $backup_file"
    
    # Determine backup type and restore accordingly
    case "$backup_file" in
        *.tar.gz|*.tgz)
            tar -xzf "$backup_file" -C "$data_dir" 2>/dev/null || return 1
            ;;
        *.tar)
            tar -xf "$backup_file" -C "$data_dir" 2>/dev/null || return 1
            ;;
        *.zip)
            unzip -q "$backup_file" -d "$data_dir" 2>/dev/null || return 1
            ;;
        *.db|*.sqlite|*.sqlite3)
            cp "$backup_file" "$data_dir/" 2>/dev/null || return 1
            ;;
        *)
            # Generic copy for other file types
            cp -r "$backup_file" "$data_dir/" 2>/dev/null || return 1
            ;;
    esac
    
    return 0
}

#######################################
# Fix ownership and permissions
# Args: $1 - directory
#       $2 - owner (user:group)
# Returns: 0 always
#######################################
recovery::fix_permissions() {
    local dir="$1"
    local owner="${2:-1000:1000}"
    
    if [[ -d "$dir" ]] && [[ -n "$owner" ]] && [[ "$owner" != ":" ]]; then
        log::info "Fixing permissions for: $dir"
        chown -R "$owner" "$dir" 2>/dev/null || true
        chmod -R u+rw "$dir" 2>/dev/null || true
    fi
    
    return 0
}

#######################################
# Create backup of current data
# Args: $1 - data_directory
#       $2 - backup_directory
#       $3 - backup_prefix (optional)
# Returns: 0 on success, 1 on failure
#######################################
recovery::create_backup() {
    local data_dir="$1"
    local backup_dir="$2"
    local prefix="${3:-backup}"
    
    if [[ ! -d "$data_dir" ]]; then
        log::error "Data directory not found: $data_dir"
        return 1
    fi
    
    mkdir -p "$backup_dir" || return 1
    
    local backup_file="${backup_dir}/${prefix}_$(date +%Y%m%d_%H%M%S).tar.gz"
    
    log::info "Creating backup: $backup_file"
    tar -czf "$backup_file" -C "$data_dir" . 2>/dev/null || return 1
    
    log::success "Backup created: $backup_file"
    return 0
}

#######################################
# Clean old backups keeping only recent ones
# Args: $1 - backup_directory
#       $2 - keep_count (default: 5)
#       $3 - file_pattern (default: "*.tar.gz")
# Returns: 0 always
#######################################
recovery::cleanup_old_backups() {
    local backup_dir="$1"
    local keep_count="${2:-5}"
    local pattern="${3:-*.tar.gz}"
    
    if [[ ! -d "$backup_dir" ]]; then
        return 0
    fi
    
    log::info "Cleaning old backups (keeping $keep_count most recent)"
    
    # Find and delete old backups
    find "$backup_dir" -name "$pattern" -type f | \
        xargs -r ls -t | \
        tail -n +$((keep_count + 1)) | \
        xargs -r rm -f
    
    return 0
}