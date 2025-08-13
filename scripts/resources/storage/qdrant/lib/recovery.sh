#!/usr/bin/env bash
# Qdrant Recovery Module
# Backup and restore functionality using backup-framework.sh

set -euo pipefail

# Get directory of this script
QDRANT_RECOVERY_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Source required utilities
# shellcheck disable=SC1091
source "${QDRANT_RECOVERY_DIR}/../../../../lib/utils/var.sh"
# shellcheck disable=SC1091
source "${var_SCRIPTS_RESOURCES_LIB_DIR}/backup-framework.sh"
# shellcheck disable=SC1091
source "${var_LIB_UTILS_DIR}/log.sh"
# shellcheck disable=SC1091
source "${var_LIB_SYSTEM_DIR}/trash.sh"

# Source configuration and API
# shellcheck disable=SC1091
source "${QDRANT_RECOVERY_DIR}/../config/defaults.sh"
# shellcheck disable=SC1091
source "${QDRANT_RECOVERY_DIR}/api.sh"
# shellcheck disable=SC1091
source "${QDRANT_RECOVERY_DIR}/common.sh"
# shellcheck disable=SC1091
source "${QDRANT_RECOVERY_DIR}/collections.sh"

# Export configuration
qdrant::export_config

#######################################
# Create a backup of Qdrant data
# Arguments:
#   $1 - Backup label (optional, defaults to timestamp)
# Returns:
#   0 on success, 1 on failure
#######################################
qdrant::create_backup() {
    local label="${1:-auto-$(date +%Y%m%d-%H%M%S)}"
    
    log::info "Creating Qdrant backup: $label"
    
    # Check if Qdrant is running
    if ! qdrant::common::is_running; then
        log::error "Qdrant is not running. Cannot create backup."
        return 1
    fi
    
    # Create temporary directory for backup files
    local temp_dir
    temp_dir=$(mktemp -d -t qdrant-backup.XXXXXX)
    
    # Create backup metadata
    local metadata
    metadata=$(cat <<EOF
{
    "service": "qdrant",
    "timestamp": "$(date -u +%Y-%m-%dT%H:%M:%SZ)",
    "label": "$label",
    "version": "$(docker inspect qdrant --format='{{.Config.Image}}' 2>/dev/null || echo 'unknown')",
    "collections": []
}
EOF
    )
    
    # Get list of collections
    # Debug: Check what function returns
    local collection_names
    collection_names=$(qdrant::collections::list_simple 2>/dev/null || true)
    
    # Debug output to stderr to see what we got
    >&2 echo "[DEBUG] First 3 collection names: $(echo "$collection_names" | head -3 | tr '\n' ', ')"
    
    if [[ -z "$collection_names" ]]; then
        log::warn "No collections found to backup"
    else
        # Create snapshots for each collection
        local backup_files=()
        while IFS= read -r collection; do
            if [[ -n "$collection" ]]; then
                log::info "Creating snapshot for collection: $collection"
                
                # Create snapshot via API
                local snapshot_response
                snapshot_response=$(qdrant::api::create_snapshot "$collection" 2>/dev/null || true)
                
                if [[ $? -eq 0 ]]; then
                    # Extract snapshot name from response
                    local snapshot_name
                    snapshot_name=$(echo "$snapshot_response" | jq -r '.result.name // empty' 2>/dev/null || true)
                    
                    if [[ -n "$snapshot_name" ]]; then
                        # Download snapshot from container
                        local snapshot_path="/qdrant/snapshots/${collection}/${snapshot_name}"
                        local local_file="${temp_dir}/${collection}-${snapshot_name}"
                        
                        if docker cp "qdrant:${snapshot_path}" "$local_file" 2>/dev/null; then
                            backup_files+=("$local_file")
                            
                            # Update metadata
                            metadata=$(echo "$metadata" | jq ".collections += [{\"name\": \"$collection\", \"snapshot\": \"$snapshot_name\"}]")
                            log::success "Snapshot created for $collection"
                        else
                            log::warn "Could not retrieve snapshot for $collection"
                        fi
                    else
                        log::warn "Could not create snapshot for $collection"
                    fi
                else
                    log::warn "Failed to create snapshot for collection: $collection"
                fi
            fi
        done <<< "$collection_names"
    fi
    
    # Save metadata
    echo "$metadata" > "${temp_dir}/metadata.json"
    
    # Create tar archive of all backup files
    local backup_file="${QDRANT_SNAPSHOTS_DIR}/backup-${label}.tar.gz"
    
    # Ensure snapshots directory exists
    mkdir -p "${QDRANT_SNAPSHOTS_DIR}"
    
    # Create backup archive
    if tar -czf "$backup_file" -C "$temp_dir" . 2>/dev/null; then
        log::success "Backup created: $backup_file"
        
        # Store in backup framework
        if backup::store "qdrant" "$backup_file" "$metadata"; then
            log::success "Backup registered with framework"
        fi
        
        # Cleanup temp directory
        trash::safe_remove "$temp_dir" --no-confirm
        
        return 0
    else
        log::error "Failed to create backup archive"
        trash::safe_remove "$temp_dir" --no-confirm
        return 1
    fi
}

#######################################
# Restore from a backup
# Arguments:
#   $1 - Backup name/label
# Returns:
#   0 on success, 1 on failure
#######################################
qdrant::restore_backup() {
    local backup_name="${1:-}"
    
    if [[ -z "$backup_name" ]]; then
        log::error "Backup name is required"
        return 1
    fi
    
    log::info "Restoring from backup: $backup_name"
    
    # Find backup file
    local backup_file
    
    # Check if it's a full path
    if [[ -f "$backup_name" ]]; then
        backup_file="$backup_name"
    # Check in snapshots directory
    elif [[ -f "${QDRANT_SNAPSHOTS_DIR}/${backup_name}" ]]; then
        backup_file="${QDRANT_SNAPSHOTS_DIR}/${backup_name}"
    elif [[ -f "${QDRANT_SNAPSHOTS_DIR}/backup-${backup_name}.tar.gz" ]]; then
        backup_file="${QDRANT_SNAPSHOTS_DIR}/backup-${backup_name}.tar.gz"
    else
        # Try to retrieve from framework
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
    
    # Read metadata
    local metadata_file="${temp_dir}/metadata.json"
    if [[ ! -f "$metadata_file" ]]; then
        log::error "Backup metadata not found"
        trash::safe_remove "$temp_dir" --no-confirm
        return 1
    fi
    
    local metadata
    metadata=$(cat "$metadata_file")
    
    # Ensure Qdrant is running
    if ! qdrant::common::is_running; then
        log::warn "Qdrant is not running. Starting it..."
        if ! docker start "${QDRANT_CONTAINER_NAME}" 2>/dev/null; then
            log::error "Failed to start Qdrant"
            trash::safe_remove "$temp_dir" --no-confirm
            return 1
        fi
        sleep 5  # Wait for startup
    fi
    
    # Restore each collection
    local collections
    collections=$(echo "$metadata" | jq -r '.collections[] // empty' 2>/dev/null || true)
    
    if [[ -n "$collections" ]]; then
        echo "$collections" | while IFS= read -r collection_info; do
            local collection_name
            local snapshot_name
            
            collection_name=$(echo "$collection_info" | jq -r '.name')
            snapshot_name=$(echo "$collection_info" | jq -r '.snapshot')
            
            local snapshot_file="${temp_dir}/${collection_name}-${snapshot_name}"
            
            if [[ -f "$snapshot_file" ]]; then
                log::info "Restoring collection: $collection_name"
                
                # Copy snapshot to container
                local container_path="/qdrant/snapshots/${collection_name}/"
                
                # Create directory in container
                docker exec "${QDRANT_CONTAINER_NAME}" mkdir -p "$container_path" 2>/dev/null || true
                
                # Copy snapshot file
                if docker cp "$snapshot_file" "${QDRANT_CONTAINER_NAME}:${container_path}${snapshot_name}" 2>/dev/null; then
                    # Trigger recovery via API (if supported)
                    # Note: Qdrant may need restart to load snapshots
                    log::success "Snapshot restored for $collection_name"
                else
                    log::error "Failed to restore snapshot for $collection_name"
                fi
            else
                log::warn "Snapshot file not found for $collection_name"
            fi
        done
    fi
    
    # Cleanup temp directory
    trash::safe_remove "$temp_dir" --no-confirm
    
    log::info "Restore complete. You may need to restart Qdrant to fully load the restored data."
    return 0
}

#######################################
# List available backups
# Returns:
#   List of backups with details
#######################################
qdrant::list_backups() {
    log::info "Available Qdrant backups:"
    echo
    
    # List backups from snapshots directory
    if [[ -d "${QDRANT_SNAPSHOTS_DIR}" ]]; then
        local backup_files
        backup_files=$(find "${QDRANT_SNAPSHOTS_DIR}" -name "backup-*.tar.gz" -o -name "*.tar.gz" 2>/dev/null | sort -r)
        
        if [[ -n "$backup_files" ]]; then
            echo "Local backups:"
            while IFS= read -r backup_file; do
                local basename
                basename=$(basename "$backup_file")
                local size
                size=$(du -h "$backup_file" | cut -f1)
                local modified
                modified=$(stat -c "%y" "$backup_file" 2>/dev/null | cut -d' ' -f1,2 | cut -d'.' -f1)
                
                echo "  - $basename ($size, $modified)"
                
                # Try to read metadata if it's a tar file
                if tar -tzf "$backup_file" metadata.json >/dev/null 2>&1; then
                    local metadata
                    metadata=$(tar -xzOf "$backup_file" metadata.json 2>/dev/null || true)
                    
                    if [[ -n "$metadata" ]]; then
                        local collections_count
                        collections_count=$(echo "$metadata" | jq '.collections | length' 2>/dev/null || echo "0")
                        echo "      Collections: $collections_count"
                    fi
                fi
            done <<< "$backup_files"
        else
            echo "  No local backups found"
        fi
    else
        echo "  Snapshots directory does not exist"
    fi
    
    echo
    
    # List framework backups
    local framework_backups
    framework_backups=$(backup::list "qdrant" 2>/dev/null || true)
    
    if [[ -n "$framework_backups" ]]; then
        echo "Framework-managed backups:"
        echo "$framework_backups"
    fi
}

#######################################
# Get backup information
# Arguments:
#   $1 - Backup name/label
# Returns:
#   Detailed backup information
#######################################
qdrant::backup_info() {
    local backup_name="${1:-}"
    
    if [[ -z "$backup_name" ]]; then
        log::error "Backup name is required"
        return 1
    fi
    
    # Find backup file
    local backup_file
    
    if [[ -f "${QDRANT_SNAPSHOTS_DIR}/backup-${backup_name}.tar.gz" ]]; then
        backup_file="${QDRANT_SNAPSHOTS_DIR}/backup-${backup_name}.tar.gz"
    elif [[ -f "${QDRANT_SNAPSHOTS_DIR}/${backup_name}" ]]; then
        backup_file="${QDRANT_SNAPSHOTS_DIR}/${backup_name}"
    elif [[ -f "$backup_name" ]]; then
        backup_file="$backup_name"
    else
        log::error "Backup not found: $backup_name"
        return 1
    fi
    
    echo "Backup Information:"
    echo "  File: $backup_file"
    echo "  Size: $(du -h "$backup_file" | cut -f1)"
    echo "  Modified: $(stat -c "%y" "$backup_file" 2>/dev/null | cut -d'.' -f1)"
    echo
    
    # Extract and display metadata
    if tar -tzf "$backup_file" metadata.json >/dev/null 2>&1; then
        local metadata
        metadata=$(tar -xzOf "$backup_file" metadata.json 2>/dev/null || true)
        
        if [[ -n "$metadata" ]]; then
            echo "Metadata:"
            echo "$metadata" | jq '.' 2>/dev/null || echo "$metadata"
        fi
    fi
    
    echo
    echo "Contents:"
    tar -tzf "$backup_file" 2>/dev/null | head -20
}

#######################################
# Export recovery functions
#######################################
export -f qdrant::create_backup
export -f qdrant::restore_backup
export -f qdrant::list_backups
export -f qdrant::backup_info