#!/usr/bin/env bash
# Qdrant Backup Management - Consolidated backup and snapshot functionality
# Combines snapshots.sh and recovery.sh into unified backup system
# Provides both native Qdrant snapshots and framework-integrated backups

set -euo pipefail

# Get directory of this script
QDRANT_BACKUP_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Source required utilities
# shellcheck disable=SC1091
source "${QDRANT_BACKUP_DIR}/../../../../lib/utils/var.sh"
# shellcheck disable=SC1091
source "${var_SCRIPTS_RESOURCES_LIB_DIR}/backup-framework.sh"
# shellcheck disable=SC1091
source "${var_LIB_UTILS_DIR}/log.sh"
# shellcheck disable=SC1091
source "${var_LIB_SYSTEM_DIR}/trash.sh"

# Source configuration and dependencies
# shellcheck disable=SC1091
source "${QDRANT_BACKUP_DIR}/../config/defaults.sh"
# Note: core.sh and collections.sh are sourced by CLI before this file

# Configuration exported by CLI
# qdrant::export_config called by main script

#######################################
# UNIFIED BACKUP INTERFACE
#######################################

#######################################
# Create a backup (unified interface for both native and framework backups)
# Arguments:
#   $1 - label (optional, default: auto-generated)
#   $2 - backup_type (optional: "native"|"framework"|"enhanced", default: "enhanced")
# Returns: 0 on success, 1 on failure
#######################################
qdrant::backup::create() {
    local label="${1:-auto-$(date +%Y%m%d-%H%M%S)}"
    local backup_type="${2:-enhanced}"
    
    log::info "Creating Qdrant backup: $label (type: $backup_type)"
    
    case "$backup_type" in
        "native")
            qdrant::backup::create_native "$label"
            ;;
        "framework") 
            qdrant::backup::create_framework "$label"
            ;;
        "enhanced"|*)
            qdrant::backup::create_enhanced "$label"
            ;;
    esac
}

#######################################
# Restore from a backup (unified interface)
# Arguments:
#   $1 - backup_name (required)
# Returns: 0 on success, 1 on failure
#######################################
qdrant::backup::restore() {
    local backup_name="${1:-}"
    
    if [[ -z "$backup_name" ]]; then
        log::error "Backup name is required"
        return 1
    fi
    
    log::info "Restoring from backup: $backup_name"
    
    # Determine backup type and restore accordingly
    local backup_file
    
    # Try framework backup first
    if backup_file=$(backup::retrieve "qdrant" "$backup_name" 2>/dev/null) && [[ -f "$backup_file" ]]; then
        qdrant::backup::restore_framework "$backup_name"
    # Try native snapshot
    elif [[ -f "${QDRANT_SNAPSHOTS_DIR}/${backup_name}.tar.gz" ]] || [[ -f "${QDRANT_SNAPSHOTS_DIR}/qdrant-snapshot-${backup_name}.tar.gz" ]]; then
        qdrant::backup::restore_native "$backup_name"
    # Try direct file path
    elif [[ -f "$backup_name" ]]; then
        qdrant::backup::restore_from_file "$backup_name"
    else
        log::error "Backup not found: $backup_name"
        return 1
    fi
}

#######################################
# List all available backups (both native and framework)
# Returns: List of backups with details
#######################################
qdrant::backup::list() {
    log::info "Available Qdrant backups:"
    echo
    
    # List native snapshots
    if [[ -d "${QDRANT_SNAPSHOTS_DIR}" ]]; then
        local snapshot_files
        snapshot_files=$(find "${QDRANT_SNAPSHOTS_DIR}" -name "*.tar.gz" 2>/dev/null | sort -r)
        
        if [[ -n "$snapshot_files" ]]; then
            echo "Native snapshots:"
            while IFS= read -r snapshot_file; do
                [[ -z "$snapshot_file" ]] && continue
                local basename
                basename=$(basename "$snapshot_file")
                local size
                size=$(du -h "$snapshot_file" | cut -f1)
                local modified
                modified=$(stat -c "%y" "$snapshot_file" 2>/dev/null | cut -d' ' -f1,2 | cut -d'.' -f1)
                
                printf "  %-30s %8s  %s\\n" "$basename" "$size" "$modified"
                
                # Show metadata if available
                if tar -tzf "$snapshot_file" metadata.json >/dev/null 2>&1; then
                    local metadata
                    metadata=$(tar -xzOf "$snapshot_file" metadata.json 2>/dev/null || true)
                    if [[ -n "$metadata" ]]; then
                        local collections_count
                        collections_count=$(echo "$metadata" | jq -r '.total_collections // "unknown"' 2>/dev/null)
                        printf "    └── Collections: %s\\n" "$collections_count"
                    fi
                fi
            done <<< "$snapshot_files"
        else
            echo "  No native snapshots found"
        fi
        echo
    fi
    
    # List framework backups
    local backup_dir="${BACKUP_ROOT:-$HOME/.vrooli/backups}/qdrant"
    
    if [[ -d "$backup_dir" ]]; then
        echo "Framework-managed backups:"
        # List directories safely, filtering out malformed names
        find "$backup_dir" -maxdepth 1 -type d -name "20*" 2>/dev/null | while IFS= read -r backup_path; do
            [[ "$backup_path" == "$backup_dir" ]] && continue
            
            # Get just the directory name
            local backup_name
            backup_name=$(basename "$backup_path")
            
            # Skip if name looks malformed (contains special chars, newlines, or is too long)
            if [[ ${#backup_name} -gt 50 ]] || [[ "$backup_name" =~ [{}\"\'\ ] ]] || [[ "$backup_name" =~ $'\n' ]]; then
                log::debug "Skipping malformed backup directory: $backup_name"
                continue
            fi
            
            # Additional validation - ensure it matches expected timestamp format
            if [[ ! "$backup_name" =~ ^[0-9]{8}_[0-9]{6}_ ]]; then
                log::debug "Skipping directory with invalid timestamp format: $backup_name"
                continue
            fi
            
            # Format the output nicely with metadata if available
            local formatted_entry
            if [[ -f "$backup_path/backup_info.json" ]]; then
                local timestamp
                local label
                timestamp=$(jq -r '.created_at // ""' "$backup_path/backup_info.json" 2>/dev/null)
                label=$(jq -r '.label // ""' "$backup_path/backup_info.json" 2>/dev/null)
                if [[ -n "$timestamp" && -n "$label" ]]; then
                    # Convert ISO timestamp to readable format
                    local readable_time
                    readable_time=$(date -d "$timestamp" "+%Y-%m-%d %H:%M:%S" 2>/dev/null || echo "$timestamp")
                    formatted_entry="  $backup_name  ($label, created: $readable_time)"
                else
                    formatted_entry="  $backup_name"
                fi
            else
                # Extract timestamp from directory name for display
                local date_part
                local time_part
                date_part=$(echo "$backup_name" | cut -d'_' -f1)
                time_part=$(echo "$backup_name" | cut -d'_' -f2)
                local readable_time
                readable_time=$(date -d "${date_part:0:4}-${date_part:4:2}-${date_part:6:2} ${time_part:0:2}:${time_part:2:2}:${time_part:4:2}" "+%Y-%m-%d %H:%M:%S" 2>/dev/null || echo "unknown")
                formatted_entry="  $backup_name  (created: $readable_time)"
            fi
            
            echo "$formatted_entry"
        done | sort -r
    else
        echo "  No framework backups found"
    fi
}

#######################################
# Get detailed backup information
# Arguments:
#   $1 - backup_name (required)
# Returns: Detailed backup information
#######################################
qdrant::backup::info() {
    local backup_name="${1:-}"
    
    if [[ -z "$backup_name" ]]; then
        log::error "Backup name is required"
        return 1
    fi
    
    echo "Backup Information: $backup_name"
    
    # Find backup file
    local backup_file
    
    if [[ -f "${QDRANT_SNAPSHOTS_DIR}/${backup_name}.tar.gz" ]]; then
        backup_file="${QDRANT_SNAPSHOTS_DIR}/${backup_name}.tar.gz"
    elif [[ -f "${QDRANT_SNAPSHOTS_DIR}/qdrant-snapshot-${backup_name}.tar.gz" ]]; then
        backup_file="${QDRANT_SNAPSHOTS_DIR}/qdrant-snapshot-${backup_name}.tar.gz"
    elif [[ -f "${QDRANT_SNAPSHOTS_DIR}/${backup_name}" ]]; then
        backup_file="${QDRANT_SNAPSHOTS_DIR}/${backup_name}"
    elif [[ -f "$backup_name" ]]; then
        backup_file="$backup_name"
    else
        log::error "Backup not found: $backup_name"
        return 1
    fi
    
    echo "  File: $backup_file"
    echo "  Size: $(du -h "$backup_file" | cut -f1)"
    echo "  Modified: $(stat -c "%y" "$backup_file" 2>/dev/null | cut -d'.' -f1)"
    
    # Show detailed metadata if available
    echo "  Contents:"
    if tar -tzf "$backup_file" metadata.json >/dev/null 2>&1; then
        echo "    Metadata:"
        local metadata
        metadata=$(tar -xzOf "$backup_file" metadata.json 2>/dev/null || true)
        if [[ -n "$metadata" ]]; then
            echo "$metadata" | jq . 2>/dev/null || echo "    Invalid metadata format"
        fi
    fi
    
    echo "  Archive contents (first 20 files):"
    tar -tzf "$backup_file" 2>/dev/null | head -20 | sed 's/^/    /'
}

#######################################
# Delete a backup
# Arguments:
#   $1 - backup_name (required)
# Returns: 0 on success, 1 on failure
#######################################
qdrant::backup::delete() {
    local backup_name="${1:-}"
    
    if [[ -z "$backup_name" ]]; then
        log::error "Backup name is required"
        return 1
    fi
    
    # Try to find and delete the backup
    local backup_file=""
    
    if [[ -f "${QDRANT_SNAPSHOTS_DIR}/${backup_name}.tar.gz" ]]; then
        backup_file="${QDRANT_SNAPSHOTS_DIR}/${backup_name}.tar.gz"
    elif [[ -f "${QDRANT_SNAPSHOTS_DIR}/qdrant-snapshot-${backup_name}.tar.gz" ]]; then
        backup_file="${QDRANT_SNAPSHOTS_DIR}/qdrant-snapshot-${backup_name}.tar.gz"
    fi
    
    if [[ -n "$backup_file" ]] && [[ -f "$backup_file" ]]; then
        if trash::safe_remove "$backup_file" --no-confirm; then
            log::success "Backup deleted: $backup_name"
            return 0
        else
            log::error "Failed to delete backup: $backup_name"
            return 1
        fi
    else
        log::warn "Backup not found: $backup_name"
        return 1
    fi
}

#######################################
# Cleanup old backups
# Arguments:
#   $1 - days (optional, default: 30)
# Returns: 0 on success
#######################################
qdrant::backup::cleanup() {
    local days="${1:-30}"
    
    log::info "Cleaning up Qdrant backups older than $days days..."
    
    # Cleanup framework backups
    backup::cleanup "qdrant" "$days"
    
    # Cleanup native snapshots
    if [[ -d "${QDRANT_SNAPSHOTS_DIR}" ]]; then
        local old_snapshots
        old_snapshots=$(find "${QDRANT_SNAPSHOTS_DIR}" -name "*.tar.gz" -mtime +"$days" 2>/dev/null || true)
        
        if [[ -z "$old_snapshots" ]]; then
            log::info "No old native snapshots to clean up"
        else
            local deleted_count=0
            while IFS= read -r snapshot_file; do
                [[ -z "$snapshot_file" ]] && continue
                local snapshot_name
                snapshot_name=$(basename "$snapshot_file")
                if trash::safe_remove "$snapshot_file" --no-confirm; then
                    log::info "Deleted old snapshot: $snapshot_name"
                    ((deleted_count++))
                fi
            done <<< "$old_snapshots"
            
            if [[ $deleted_count -gt 0 ]]; then
                log::success "Cleaned up $deleted_count old native snapshots"
            fi
        fi
    fi
}

#######################################
# NATIVE SNAPSHOT FUNCTIONS
#######################################

#######################################
# Create native Qdrant snapshot
# Arguments:
#   $1 - snapshot_name
# Returns: 0 on success, 1 on failure
#######################################
qdrant::backup::create_native() {
    local snapshot_name="${1:-qdrant-snapshot-$(date +%Y%m%d-%H%M%S)}"
    
    log::info "Creating native snapshot: $snapshot_name"
    
    # Create snapshots directory
    mkdir -p "${QDRANT_SNAPSHOTS_DIR}"
    local snapshot_dir="${QDRANT_SNAPSHOTS_DIR}/${snapshot_name}"
    mkdir -p "$snapshot_dir"
    
    # Get all collections
    local collections_to_backup=()
    local all_collections
    all_collections=$(qdrant::collections::list_simple 2>/dev/null)
    
    if [[ $? -ne 0 || -z "$all_collections" ]]; then
        log::warn "No collections found to backup"
        trash::safe_remove "$snapshot_dir" --no-confirm
        return 0
    fi
    
    while IFS= read -r collection; do
        if [[ -n "$collection" ]]; then
            collections_to_backup+=("$collection")
        fi
    done <<< "$all_collections"
    
    if [[ ${#collections_to_backup[@]} -eq 0 ]]; then
        log::warn "No collections to backup"
        trash::safe_remove "$snapshot_dir" --no-confirm
        return 0
    fi
    
    log::info "Backing up ${#collections_to_backup[@]} collections..."
    
    # Create metadata
    cat > "$snapshot_dir/metadata.json" << EOF
{
  "backup_type": "native_snapshot",
  "created_at": "$(date -Iseconds)",
  "qdrant_version": "$(qdrant::version 2>/dev/null || echo 'unknown')",
  "total_collections": ${#collections_to_backup[@]},
  "collections": $(printf '%s\n' "${collections_to_backup[@]}" | jq -R . | jq -s .)
}
EOF
    
    # Create snapshots for each collection
    local success_count=0
    local total_count=${#collections_to_backup[@]}
    
    for collection in "${collections_to_backup[@]}"; do
        log::info "  Creating snapshot for collection: $collection"
        
        if qdrant::backup::create_collection_snapshot "$collection" "$snapshot_dir"; then
            ((success_count++))
        else
            log::warn "  Failed to create snapshot for collection: $collection"
        fi
    done
    
    # Create archive
    local archive_path="${QDRANT_SNAPSHOTS_DIR}/${snapshot_name}.tar.gz"
    
    if (cd "${QDRANT_SNAPSHOTS_DIR}" && tar -czf "${snapshot_name}.tar.gz" "$snapshot_name"); then
        trash::safe_remove "$snapshot_dir" --no-confirm
        
        local archive_size
        archive_size=$(du -h "$archive_path" | cut -f1)
        
        if [[ $success_count -eq $total_count ]]; then
            log::success "Native snapshot created: ${snapshot_name}.tar.gz ($archive_size)"
            log::info "Snapshot location: $archive_path"
            return 0
        else
            log::warn "Snapshot created with some failures: $success_count/$total_count collections"
            log::info "Snapshot location: $archive_path"
            return 0
        fi
    else
        log::error "Failed to create snapshot archive"
        trash::safe_remove "$snapshot_dir" --no-confirm
        return 1
    fi
}

#######################################
# Restore from native snapshot
# Arguments:
#   $1 - snapshot_name
# Returns: 0 on success, 1 on failure
#######################################
qdrant::backup::restore_native() {
    local snapshot_name="${1:-}"
    
    if [[ -z "$snapshot_name" ]]; then
        log::error "Snapshot name is required"
        return 1
    fi
    
    # Find snapshot file
    local snapshot_path=""
    if [[ -f "${QDRANT_SNAPSHOTS_DIR}/${snapshot_name}.tar.gz" ]]; then
        snapshot_path="${QDRANT_SNAPSHOTS_DIR}/${snapshot_name}.tar.gz"
    elif [[ -f "${QDRANT_SNAPSHOTS_DIR}/qdrant-snapshot-${snapshot_name}.tar.gz" ]]; then
        snapshot_path="${QDRANT_SNAPSHOTS_DIR}/qdrant-snapshot-${snapshot_name}.tar.gz"
    else
        log::error "Snapshot not found: $snapshot_name"
        return 1
    fi
    
    log::info "Restoring from native snapshot: $snapshot_name"
    
    # Extract snapshot
    local temp_dir
    temp_dir=$(mktemp -d -t qdrant-restore.XXXXXX)
    
    if ! tar -xzf "$snapshot_path" -C "$temp_dir" 2>/dev/null; then
        log::error "Failed to extract snapshot"
        trash::safe_remove "$temp_dir" --no-confirm
        return 1
    fi
    
    # Find the extracted snapshot directory
    local snapshot_content_dir
    snapshot_content_dir=$(find "$temp_dir" -maxdepth 1 -type d -name "qdrant-snapshot-*" | head -1)
    
    if [[ -z "$snapshot_content_dir" ]]; then
        snapshot_content_dir="$temp_dir/$snapshot_name"
    fi
    
    if [[ -f "$snapshot_content_dir/metadata.json" ]]; then
        log::info "Reading snapshot metadata..."
        local metadata
        metadata=$(cat "$snapshot_content_dir/metadata.json")
        local created_at
        created_at=$(echo "$metadata" | jq -r '.created_at // "unknown"' 2>/dev/null)
        local total_collections
        total_collections=$(echo "$metadata" | jq -r '.total_collections // "unknown"' 2>/dev/null)
        
        log::info "  Created: $created_at"
        log::info "  Collections: $total_collections"
    fi
    
    # Get list of collections to restore
    local collections_to_restore=()
    
    if [[ -f "$snapshot_content_dir/metadata.json" ]]; then
        local collections_json
        collections_json=$(jq -r '.collections[]' "$snapshot_content_dir/metadata.json" 2>/dev/null || true)
        
        if [[ -n "$collections_json" ]]; then
            while IFS= read -r collection_name; do
                [[ -n "$collection_name" ]] && collections_to_restore+=("$collection_name")
            done <<< "$collections_json"
        fi
    fi
    
    if [[ ${#collections_to_restore[@]} -eq 0 ]]; then
        log::error "No collections found in snapshot"
        trash::safe_remove "$temp_dir" --no-confirm
        return 1
    fi
    
    log::info "Restoring ${#collections_to_restore[@]} collections..."
    
    # Restore each collection
    local success_count=0
    local total_count=${#collections_to_restore[@]}
    
    for collection in "${collections_to_restore[@]}"; do
        log::info "  Restoring collection: $collection"
        
        if qdrant::backup::restore_collection_snapshot "$collection" "$snapshot_content_dir"; then
            ((success_count++))
        else
            log::warn "  Failed to restore collection: $collection"
        fi
    done
    
    trash::safe_remove "$temp_dir" --no-confirm
    
    if [[ $success_count -eq $total_count ]]; then
        log::success "Native snapshot restored: $snapshot_name"
        log::info "Restored $success_count collections successfully"
        return 0
    elif [[ $success_count -gt 0 ]]; then
        log::warn "Partial restore completed: $success_count/$total_count collections"
        return 0
    else
        log::error "Failed to restore any collections"
        return 1
    fi
}

#######################################
# FRAMEWORK BACKUP FUNCTIONS  
#######################################

#######################################
# Create framework-integrated backup
# Arguments:
#   $1 - label
# Returns: 0 on success, 1 on failure
#######################################
qdrant::backup::create_framework() {
    local label="${1:-auto-$(date +%Y%m%d-%H%M%S)}"
    
    log::info "Creating framework backup: $label"
    
    if ! qdrant::health::is_healthy; then
        log::error "Qdrant is not running. Cannot create backup."
        return 1
    fi
    
    # Create temporary directory for backup files
    local temp_dir
    temp_dir=$(mktemp -d -t qdrant-backup.XXXXXX)
    
    # Create backup metadata
    local metadata
    metadata=$(cat << EOF
{
  "service": "qdrant",
  "backup_type": "framework",
  "label": "$label",
  "created_at": "$(date -Iseconds)",
  "qdrant_version": "$(qdrant::version 2>/dev/null || echo 'unknown')"
}
EOF
)
    
    # Get collections and create snapshots
    local collections
    collections=$(qdrant::collections::list_simple 2>/dev/null || true)
    
    if [[ -z "$collections" ]]; then
        log::warn "No collections found to backup"
        trash::safe_remove "$temp_dir" --no-confirm
        return 0
    else
        local backup_files=()
        
        while IFS= read -r collection; do
            [[ -z "$collection" ]] && continue
            
            log::info "Creating snapshot for collection: $collection"
            
            # Create snapshot via API
            local snapshot_response
            if snapshot_response=$(qdrant::api::create_snapshot "$collection" 2>/dev/null || true); then
                local snapshot_name
                snapshot_name=$(echo "$snapshot_response" | jq -r '.result.name // empty' 2>/dev/null)
                
                if [[ -n "$snapshot_name" ]]; then
                    # Download snapshot file
                    local local_file="$temp_dir/${collection}_${snapshot_name}"
                    local download_url="${QDRANT_BASE_URL}/collections/${collection}/snapshots/${snapshot_name}"
                    
                    if curl -s "${QDRANT_CURL_OPTS[@]}" "$download_url" -o "$local_file" && [[ -s "$local_file" ]]; then
                        log::success "Snapshot created for $collection"
                        backup_files+=("$local_file")
                    else
                        log::warn "Could not retrieve snapshot for $collection"
                    fi
                else
                    log::warn "Could not create snapshot for $collection"
                fi
            else
                log::warn "Failed to create snapshot for collection: $collection"
            fi
        done <<< "$collections"
        
        if [[ ${#backup_files[@]} -eq 0 ]]; then
            log::warn "No backup files created"
            trash::safe_remove "$temp_dir" --no-confirm
            return 0
        fi
    fi
    
    # Create backup archive
    local backup_file="${QDRANT_SNAPSHOTS_DIR}/backup-${label}.tar.gz"
    mkdir -p "${QDRANT_SNAPSHOTS_DIR}"
    
    # Create backup archive
    if tar -czf "$backup_file" -C "$temp_dir" . 2>/dev/null; then
        log::success "Backup created: $backup_file"
        
        # Store in backup framework
        if backup::store "qdrant" "$backup_file" "$metadata"; then
            log::success "Backup registered with framework"
        else
            log::warn "Backup created but not registered with framework"
        fi
        
        trash::safe_remove "$temp_dir" --no-confirm
        return 0
    else
        log::error "Failed to create backup archive"
        trash::safe_remove "$temp_dir" --no-confirm
        return 1
    fi
}

#######################################
# Restore framework backup
# Arguments:
#   $1 - backup_name
# Returns: 0 on success, 1 on failure
#######################################
qdrant::backup::restore_framework() {
    local backup_name="${1:-}"
    
    if [[ -z "$backup_name" ]]; then
        log::error "Backup name is required"
        return 1
    fi
    
    log::info "Restoring from framework backup: $backup_name"
    
    # Find backup file
    local backup_file
    
    if [[ -f "$backup_name" ]]; then
        backup_file="$backup_name"
    elif [[ -f "${QDRANT_SNAPSHOTS_DIR}/${backup_name}" ]]; then
        backup_file="${QDRANT_SNAPSHOTS_DIR}/${backup_name}"
    elif [[ -f "${QDRANT_SNAPSHOTS_DIR}/backup-${backup_name}.tar.gz" ]]; then
        backup_file="${QDRANT_SNAPSHOTS_DIR}/backup-${backup_name}.tar.gz"
    else
        # Try framework retrieval
        backup_file=$(backup::retrieve "qdrant" "$backup_name" 2>/dev/null || echo "")
        
        if [[ -z "$backup_file" ]] || [[ ! -f "$backup_file" ]]; then
            log::error "Backup not found: $backup_name"
            return 1
        fi
    fi
    
    # Extract backup to temp directory
    local temp_dir
    temp_dir=$(mktemp -d -t qdrant-restore.XXXXXX)
    
    if ! tar -xzf "$backup_file" -C "$temp_dir" 2>/dev/null; then
        log::error "Failed to extract backup"
        trash::safe_remove "$temp_dir" --no-confirm
        return 1
    fi
    
    # Check metadata
    if [[ ! -f "$temp_dir/metadata.json" ]]; then
        log::error "Backup metadata not found"
        trash::safe_remove "$temp_dir" --no-confirm
        return 1
    fi
    
    # Ensure Qdrant is running
    if ! qdrant::health::is_healthy; then
        log::warn "Qdrant is not running. Starting it..."
        if ! qdrant::start; then
            log::error "Failed to start Qdrant"
            trash::safe_remove "$temp_dir" --no-confirm
            return 1
        fi
        
        if ! qdrant::wait_for_ready 30; then
            log::error "Qdrant failed to start properly"
            trash::safe_remove "$temp_dir" --no-confirm
            return 1
        fi
    fi
    
    # Restore collections from snapshot files
    local snapshot_files
    snapshot_files=$(find "$temp_dir" -name "*.snapshot" -o -name "*_*" 2>/dev/null | sort)
    
    if [[ -n "$snapshot_files" ]]; then
        while IFS= read -r snapshot_file; do
            [[ -z "$snapshot_file" ]] && continue
            
            local filename
            filename=$(basename "$snapshot_file")
            local collection_name
            collection_name=$(echo "$filename" | cut -d'_' -f1)
            
            if [[ -n "$collection_name" ]]; then
                log::info "Restoring collection: $collection_name"
                
                if qdrant::backup::upload_and_restore_snapshot "$collection_name" "$snapshot_file"; then
                    log::success "Snapshot restored for $collection_name"
                else
                    log::error "Failed to restore snapshot for $collection_name"
                fi
            else
                log::warn "Snapshot file not found for $collection_name"
            fi
        done <<< "$snapshot_files"
    fi
    
    trash::safe_remove "$temp_dir" --no-confirm
    
    log::info "Restore complete. You may need to restart Qdrant to fully load the restored data."
    return 0
}

#######################################
# Enhanced backup (framework + metadata)
# Arguments:
#   $1 - label
# Returns: 0 on success, 1 on failure
#######################################
qdrant::backup::create_enhanced() {
    local label="${1:-auto-$(date +%Y%m%d-%H%M%S)}"
    
    if ! qdrant::health::is_healthy; then
        log::error "Qdrant must be running to create backup"
        return 1
    fi
    
    log::info "Creating enhanced Qdrant backup..."
    
    # Create temporary directory structure
    local temp_dir
    temp_dir=$(mktemp -d -t qdrant-enhanced-backup.XXXXXX)
    mkdir -p "$temp_dir/snapshots"
    mkdir -p "$temp_dir/metadata"
    
    # Backup collection data
    if ! qdrant::backup::backup_collections_data "$temp_dir/snapshots"; then
        log::error "Failed to backup collection data"
        trash::safe_remove "$temp_dir" --no-confirm
        return 1
    fi
    
    # Copy configuration files if they exist
    if [[ -d "${QDRANT_CONFIG_DIR}" ]]; then
        cp -r "${QDRANT_CONFIG_DIR}" "$temp_dir/config" 2>/dev/null || true
    fi
    
    # Create backup metadata
    qdrant::backup::create_enhanced_metadata "$temp_dir/metadata"
    
    # Use backup framework to store
    local backup_path
    if backup_path=$(backup::store "qdrant" "$temp_dir" "$label"); then
        log::success "Enhanced backup created: $(basename "$backup_path")"
        trash::safe_remove "$temp_dir" --no-confirm
        return 0
    else
        log::error "Failed to store backup"
        trash::safe_remove "$temp_dir" --no-confirm
        return 1
    fi
}

#######################################
# HELPER FUNCTIONS
#######################################

#######################################
# Create collection snapshot
# Arguments:
#   $1 - collection_name
#   $2 - output_directory
# Returns: 0 on success, 1 on failure
#######################################
qdrant::backup::create_collection_snapshot() {
    local collection_name="$1"
    local output_dir="$2"
    
    # Create snapshot via API
    local snapshot_response
    snapshot_response=$(qdrant::api::create_snapshot "$collection_name" 2>/dev/null || echo "")
    
    if [[ -z "$snapshot_response" ]]; then
        return 1
    fi
    
    local snapshot_name
    snapshot_name=$(echo "$snapshot_response" | jq -r '.result.name // empty' 2>/dev/null)
    
    if [[ -z "$snapshot_name" ]]; then
        return 1
    fi
    
    # Download snapshot
    local download_url="${QDRANT_BASE_URL}/collections/${collection_name}/snapshots/${snapshot_name}"
    local local_file="$output_dir/${collection_name}_${snapshot_name}"
    
    if curl -s "${QDRANT_CURL_OPTS[@]}" "$download_url" -o "$local_file" && [[ -s "$local_file" ]]; then
        # Cleanup remote snapshot
        qdrant::api::request "DELETE" "/collections/${collection_name}/snapshots/${snapshot_name}" >/dev/null 2>&1 || true
        return 0
    else
        # Cleanup on failure
        trash::safe_remove "$local_file" --temp 2>/dev/null || true
        qdrant::api::request "DELETE" "/collections/${collection_name}/snapshots/${snapshot_name}" >/dev/null 2>&1 || true
        return 1
    fi
}

#######################################
# Restore collection from snapshot
# Arguments:
#   $1 - collection_name
#   $2 - snapshot_content_dir
# Returns: 0 on success, 1 on failure
#######################################
qdrant::backup::restore_collection_snapshot() {
    local collection_name="$1"
    local snapshot_content_dir="$2"
    
    # Find snapshot file for this collection
    local snapshot_file
    snapshot_file=$(find "$snapshot_content_dir" -name "${collection_name}_*" -o -name "*${collection_name}*" 2>/dev/null | head -1)
    
    if [[ -z "$snapshot_file" ]] || [[ ! -f "$snapshot_file" ]]; then
        return 1
    fi
    
    # Upload and restore the snapshot
    return qdrant::backup::upload_and_restore_snapshot "$collection_name" "$snapshot_file"
}

#######################################
# Upload and restore snapshot
# Arguments:
#   $1 - collection_name
#   $2 - snapshot_file
# Returns: 0 on success, 1 on failure
#######################################
qdrant::backup::upload_and_restore_snapshot() {
    local collection_name="$1"
    local snapshot_file="$2"
    
    # Upload snapshot via API
    local upload_response
    upload_response=$(curl -s "${QDRANT_CURL_OPTS[@]}" \
        -X POST \
        -F "snapshot=@$snapshot_file" \
        "${QDRANT_BASE_URL}/collections/${collection_name}/snapshots/upload" 2>/dev/null || echo "")
    
    if [[ -z "$upload_response" ]]; then
        return 1
    fi
    
    # Check if upload was successful
    local status
    status=$(echo "$upload_response" | jq -r '.status // "error"' 2>/dev/null)
    
    if [[ "$status" == "ok" ]]; then
        # Wait for restore to complete
        sleep 2
        
        # Verify collection was restored
        if qdrant::collections::exists "$collection_name"; then
            return 0
        fi
    fi
    
    return 1
}

#######################################
# Backup collections data helper
# Arguments:
#   $1 - output_directory
# Returns: 0 on success, 1 on failure
#######################################
qdrant::backup::backup_collections_data() {
    local output_dir="$1"
    
    # Get list of collections
    local collections
    collections=$(qdrant::collections::list_simple 2>/dev/null || true)
    
    if [[ -z "$collections" ]]; then
        log::warn "Unable to retrieve collections list"
        return 1
    fi
    
    # Create snapshots for each collection
    while IFS= read -r collection; do
        [[ -z "$collection" ]] && continue
        
        log::info "Creating snapshot for collection: $collection"
        
        if qdrant::backup::create_collection_snapshot "$collection" "$output_dir"; then
            log::debug "Snapshot created for $collection"
        else
            log::warn "Failed to create snapshot for $collection"
        fi
    done <<< "$collections"
    
    return 0
}

#######################################
# Create enhanced backup metadata
# Arguments:
#   $1 - metadata_directory
# Returns: 0 on success
#######################################
qdrant::backup::create_enhanced_metadata() {
    local metadata_dir="$1"
    
    # Create backup info file
    cat > "$metadata_dir/backup_info.json" << EOF
{
  "service": "qdrant",
  "backup_type": "enhanced",
  "created_at": "$(date -Iseconds)",
  "qdrant_version": "$(qdrant::version 2>/dev/null || echo 'unknown')",
  "backup_type": "full",
  "cluster_info": $(qdrant::api::request "GET" "/cluster" 2>/dev/null || echo '{}'),
  "collections": $(qdrant::collections::list_simple | jq -R . | jq -s . 2>/dev/null || echo '[]')
}
EOF
    
    # Save system information
    cat > "$metadata_dir/system_info.json" << EOF
{
  "hostname": "$(hostname)",
  "os": "$(uname -s)",
  "arch": "$(uname -m)",
  "docker_version": "$(docker --version 2>/dev/null || echo 'unknown')",
  "backup_framework_version": "1.0"
}
EOF
    
    # Export environment variables
    env | grep '^QDRANT_' > "$metadata_dir/qdrant_env.txt" 2>/dev/null || true
    
    return 0
}

#######################################
# Restore from file helper
# Arguments:
#   $1 - backup_file_path
# Returns: 0 on success, 1 on failure
#######################################
qdrant::backup::restore_from_file() {
    local backup_file="$1"
    
    # Determine backup type by inspecting the archive
    if tar -tzf "$backup_file" metadata.json >/dev/null 2>&1; then
        # Has metadata, likely native or framework backup
        local temp_dir
        temp_dir=$(mktemp -d -t qdrant-restore-inspect.XXXXXX)
        
        if tar -xzf "$backup_file" -C "$temp_dir" metadata.json 2>/dev/null; then
            local backup_type
            backup_type=$(jq -r '.backup_type // "unknown"' "$temp_dir/metadata.json" 2>/dev/null)
            
            trash::safe_remove "$temp_dir" --no-confirm
            
            case "$backup_type" in
                "native_snapshot")
                    qdrant::backup::restore_native "$(basename "$backup_file" .tar.gz)"
                    ;;
                "framework"|"enhanced")
                    qdrant::backup::restore_framework "$backup_file"
                    ;;
                *)
                    # Default to framework restore
                    qdrant::backup::restore_framework "$backup_file"
                    ;;
            esac
        else
            trash::safe_remove "$temp_dir" --no-confirm
            log::error "Could not inspect backup file"
            return 1
        fi
    else
        # No metadata, assume it's a simple archive
        qdrant::backup::restore_framework "$backup_file"
    fi
}

#######################################
# EXPORT FUNCTIONS
#######################################

# Export all backup functions
export -f qdrant::backup::create
export -f qdrant::backup::restore
export -f qdrant::backup::list
export -f qdrant::backup::info
export -f qdrant::backup::delete
export -f qdrant::backup::cleanup
export -f qdrant::backup::create_native
export -f qdrant::backup::restore_native
export -f qdrant::backup::create_framework
export -f qdrant::backup::restore_framework
export -f qdrant::backup::create_enhanced
export -f qdrant::backup::create_collection_snapshot
export -f qdrant::backup::restore_collection_snapshot
export -f qdrant::backup::upload_and_restore_snapshot
export -f qdrant::backup::backup_collections_data
export -f qdrant::backup::create_enhanced_metadata
export -f qdrant::backup::restore_from_file

# Backward compatibility aliases - Fixed export syntax
qdrant::create_backup() { qdrant::backup::create "$@"; }
qdrant::restore_backup() { qdrant::backup::restore "$@"; }
qdrant::list_backups() { qdrant::backup::list "$@"; }
qdrant::backup_info() { qdrant::backup::info "$@"; }
qdrant::cleanup_backups() { qdrant::backup::cleanup "$@"; }
qdrant::create_backup_enhanced() { qdrant::backup::create_enhanced "$@"; }

export -f qdrant::create_backup
export -f qdrant::restore_backup
export -f qdrant::list_backups
export -f qdrant::backup_info
export -f qdrant::cleanup_backups
export -f qdrant::create_backup_enhanced

# Legacy snapshot functions for backward compatibility - Fixed export syntax
qdrant::snapshots::create() { qdrant::backup::create_native "$@"; }
qdrant::snapshots::restore() { qdrant::backup::restore_native "$@"; }
qdrant::snapshots::delete() { qdrant::backup::delete "$@"; }
qdrant::snapshots::cleanup() { qdrant::backup::cleanup "$@"; }

export -f qdrant::snapshots::create
export -f qdrant::snapshots::restore
export -f qdrant::snapshots::delete
export -f qdrant::snapshots::cleanup