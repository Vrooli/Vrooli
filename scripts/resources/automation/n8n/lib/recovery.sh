#!/usr/bin/env bash
# n8n Recovery and Backup Functions
# Automatic recovery, backup management, and data restoration

# Source required utilities
N8N_LIB_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
# shellcheck disable=SC1091
source "${N8N_LIB_DIR}/../../../lib/utils/var.sh" 2>/dev/null || true
# shellcheck disable=SC1091
source "${var_LIB_SYSTEM_DIR}/trash.sh" 2>/dev/null || true
# shellcheck disable=SC1091
source "${var_LIB_UTILS_DIR}/sudo.sh" 2>/dev/null || true

#######################################
# Find best available backup
# Returns: path to best backup or empty string
#######################################
n8n::find_best_backup() {
    local backup_pattern="${HOME}/n8n-backup-*/database.sqlite"
    local best_backup=""
    local best_size=0
    # Find largest, most recent backup
    for backup in $backup_pattern; do
        if [[ -f "$backup" ]]; then
            local size
            size=$(stat -c '%s' "$backup" 2>/dev/null || echo "0")
            if [[ "$size" -gt "$best_size" ]]; then
                best_backup="$backup"
                best_size="$size"
            fi
        fi
    done
    echo "$best_backup"
}

#######################################
# Create backup of current n8n data
# Args: $1 - backup directory (optional, defaults to timestamped dir)
# Returns: 0 if successful, 1 if failed
#######################################
n8n::create_backup() {
    local backup_dir="${1:-${HOME}/n8n-backup-$(date +%Y%m%d-%H%M%S)}"
    if [[ ! -d "$N8N_DATA_DIR" ]]; then
        log::warn "No n8n data directory to backup: $N8N_DATA_DIR"
        return 1
    fi
    log::info "Creating backup: $backup_dir"
    # Create backup directory
    if ! mkdir -p "$backup_dir"; then
        log::error "Failed to create backup directory: $backup_dir"
        return 1
    fi
    # Copy data directory
    if cp -r "$N8N_DATA_DIR"/* "$backup_dir/" 2>/dev/null; then
        log::success "Backup created successfully: $backup_dir"
        echo "$backup_dir"
        return 0
    else
        log::error "Failed to create backup"
        trash::safe_remove "$backup_dir" --no-confirm 2>/dev/null || true
        return 1
    fi
}

#######################################
# Automatically recover from filesystem corruption
# Returns: 0 if recovered, 1 if recovery failed
#######################################
n8n::auto_recover() {
    log::warn "Attempting automatic recovery..."
    # Stop container if running
    if n8n::container_running; then
        log::info "Stopping corrupted n8n container..."
        docker stop "$N8N_CONTAINER_NAME" >/dev/null 2>&1 || true
        docker rm "$N8N_CONTAINER_NAME" >/dev/null 2>&1 || true
    fi
    # Recreate data directory
    log::info "Recreating data directory..."
    trash::safe_remove "$N8N_DATA_DIR" --no-confirm 2>/dev/null || true
    if ! n8n::create_directories; then
        log::error "Failed to recreate data directory"
        return 1
    fi
    # Restore from backup if available
    local best_backup
    best_backup=$(n8n::find_best_backup)
    if [[ -n "$best_backup" ]]; then
        log::info "Restoring database from backup: $best_backup"
        cp "$best_backup" "$N8N_DATA_DIR/database.sqlite" || {
            log::error "Failed to restore database from backup"
            return 1
        }
        # Fix permissions
        if command -v sudo::restore_owner &>/dev/null; then
            sudo::restore_owner "$N8N_DATA_DIR/database.sqlite"
        else
            chown "$(whoami):$(whoami)" "$N8N_DATA_DIR/database.sqlite" 2>/dev/null || true
        fi
        chmod 644 "$N8N_DATA_DIR/database.sqlite" || true
        log::success "Database restored from backup"
    else
        log::warn "No backup found - starting with fresh database"
    fi
    return 0
}

#######################################
# Restore from specific backup
# Args: $1 - backup_file or backup_directory
# Returns: 0 if successful, 1 if failed
#######################################
n8n::restore_from_backup() {
    local backup_source="$1"
    if [[ -z "$backup_source" ]]; then
        log::error "Backup source is required"
        return 1
    fi
    if [[ ! -e "$backup_source" ]]; then
        log::error "Backup source does not exist: $backup_source"
        return 1
    fi
    # Stop n8n if running
    if n8n::container_running; then
        log::info "Stopping n8n for restore..."
        docker stop "$N8N_CONTAINER_NAME" >/dev/null 2>&1 || true
    fi
    # Backup current data before restore
    log::info "Backing up current data before restore..."
    local emergency_backup
    emergency_backup=$(n8n::create_backup "${HOME}/n8n-emergency-backup-$(date +%Y%m%d-%H%M%S)")
    # Restore from backup
    if [[ -f "$backup_source" ]]; then
        # Single file restore (database.sqlite)
        log::info "Restoring database from file: $backup_source"
        cp "$backup_source" "$N8N_DATA_DIR/database.sqlite" || {
            log::error "Failed to restore database file"
            return 1
        }
    elif [[ -d "$backup_source" ]]; then
        # Directory restore (full data directory)
        log::info "Restoring data directory from: $backup_source"
        # Clear existing data
        trash::safe_remove "$N8N_DATA_DIR" --no-confirm 2>/dev/null || true
        # Restore from backup directory
        cp -r "$backup_source" "$N8N_DATA_DIR" || {
            log::error "Failed to restore data directory"
            return 1
        }
    fi
    # Fix permissions
    if command -v sudo::restore_owner &>/dev/null; then
        sudo::restore_owner "$N8N_DATA_DIR"
    else
        local current_user
        current_user=$(whoami)
        chown -R "$current_user:$current_user" "$N8N_DATA_DIR" 2>/dev/null || true
    fi
    chmod -R 755 "$N8N_DATA_DIR" || true
    chmod 644 "$N8N_DATA_DIR"/database.sqlite 2>/dev/null || true
    log::success "Restore completed from: $backup_source"
    log::info "Emergency backup saved to: $emergency_backup"
    return 0
}

#######################################
# Clean up old backups
# Args: $1 - max_age_days (optional, default 30)
# Returns: number of backups removed
#######################################
n8n::cleanup_old_backups() {
    local max_age_days="${1:-30}"
    local removed_count=0
    log::info "Cleaning up backups older than $max_age_days days..."
    # Find and remove old backup directories
    local backup_pattern="${HOME}/n8n-backup-*"
    for backup_dir in $backup_pattern; do
        if [[ -d "$backup_dir" ]]; then
            # Check if directory is older than max_age_days
            local age_days
            age_days=$(( ($(date +%s) - $(stat -c %Y "$backup_dir")) / 86400 ))
            if [[ "$age_days" -gt "$max_age_days" ]]; then
                log::info "Removing old backup: $backup_dir (age: ${age_days} days)"
                trash::safe_remove "$backup_dir" --no-confirm 2>/dev/null || true
                removed_count=$((removed_count + 1))
            fi
        fi
    done
    if [[ "$removed_count" -gt 0 ]]; then
        log::success "Removed $removed_count old backup(s)"
    else
        log::info "No old backups found to remove"
    fi
    return "$removed_count"
}

#######################################
# List available backups
# Returns: list of backup directories with timestamps and sizes
#######################################
n8n::list_backups() {
    log::info "Available n8n backups:"
    local backup_pattern="${HOME}/n8n-backup-*"
    local backup_count=0
    for backup_dir in $backup_pattern; do
        if [[ -d "$backup_dir" ]]; then
            backup_count=$((backup_count + 1))
            # Get backup info
            local size
            size=$(du -sh "$backup_dir" 2>/dev/null | cut -f1 || echo "unknown")
            local created
            created=$(stat -c '%Y' "$backup_dir" 2>/dev/null | xargs -I {} date -d '@{}' '+%Y-%m-%d %H:%M:%S' || echo "unknown")
            # Check if database exists
            local db_status="❌ No database"
            if [[ -f "$backup_dir/database.sqlite" ]]; then
                local db_size
                db_size=$(stat -c '%s' "$backup_dir/database.sqlite" 2>/dev/null || echo "0")
                if [[ "$db_size" -gt 0 ]]; then
                    db_status="✅ Database ($db_size bytes)"
                else
                    db_status="⚠️  Empty database"
                fi
            fi
            echo "  $backup_dir"
            echo "    Created: $created"
            echo "    Size: $size"
            echo "    Status: $db_status"
            echo
        fi
    done
    if [[ "$backup_count" -eq 0 ]]; then
        log::warn "No backups found in ${HOME}/n8n-backup-*"
    else
        log::info "Total backups found: $backup_count"
    fi
}
