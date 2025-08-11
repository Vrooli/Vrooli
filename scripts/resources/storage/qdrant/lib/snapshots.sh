#!/usr/bin/env bash
# Qdrant Snapshot Management
# Functions for creating and restoring snapshots/backups

# Source required utilities
SCRIPT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
# shellcheck disable=SC1091
source "${SCRIPT_DIR}/../../../lib/utils/var.sh" 2>/dev/null || true
# shellcheck disable=SC1091
source "${var_LIB_SYSTEM_DIR}/trash.sh" 2>/dev/null || true

#######################################
# Create a snapshot of collections
# Arguments:
#   $1 - Collections list (comma-separated or "all")
#   $2 - Snapshot name (optional, auto-generated if not provided)
# Returns: 0 on success, 1 on failure
#######################################
qdrant::snapshots::create() {
    local collections_input="${1:-all}"
    local snapshot_name="${2:-}"
    
    # Generate snapshot name if not provided
    if [[ -z "$snapshot_name" ]]; then
        snapshot_name="qdrant-snapshot-$(date +%Y%m%d-%H%M%S)"
    fi
    
    log::info "${MSG_CREATING_SNAPSHOT}: $snapshot_name"
    
    # Create snapshots directory
    mkdir -p "${QDRANT_SNAPSHOTS_DIR}"
    local snapshot_dir="${QDRANT_SNAPSHOTS_DIR}/${snapshot_name}"
    mkdir -p "$snapshot_dir"
    
    # Determine which collections to backup
    local collections_to_backup=()
    
    if [[ "$collections_input" == "all" ]]; then
        # Get all collections
        local all_collections
        all_collections=$(qdrant::collections::list_simple 2>/dev/null)
        
        if [[ $? -ne 0 || -z "$all_collections" ]]; then
            log::warn "No collections found to backup"
            return 0
        fi
        
        while IFS= read -r collection; do
            if [[ -n "$collection" ]]; then
                collections_to_backup+=("$collection")
            fi
        done <<< "$all_collections"
    else
        # Parse comma-separated list
        IFS=',' read -ra collections_array <<< "$collections_input"
        for collection in "${collections_array[@]}"; do
            # Trim whitespace
            collection=$(echo "$collection" | xargs)
            if [[ -n "$collection" ]]; then
                collections_to_backup+=("$collection")
            fi
        done
    fi
    
    if [[ ${#collections_to_backup[@]} -eq 0 ]]; then
        log::warn "No collections to backup"
        return 0
    fi
    
    log::info "Backing up ${#collections_to_backup[@]} collections..."
    
    # Create metadata file
    local metadata_file="${snapshot_dir}/metadata.json"
    local timestamp
    timestamp=$(date -Iseconds)
    
    cat > "$metadata_file" << EOF
{
  "snapshot_name": "$snapshot_name",
  "created_at": "$timestamp",
  "qdrant_version": "$(qdrant::api::get_version 2>/dev/null || echo 'unknown')",
  "total_collections": ${#collections_to_backup[@]},
  "collections": []
}
EOF
    
    # Backup each collection
    local success_count=0
    local total_count=${#collections_to_backup[@]}
    
    for collection in "${collections_to_backup[@]}"; do
        log::info "  Creating snapshot for collection: $collection"
        
        if qdrant::snapshots::create_collection_snapshot "$collection" "$snapshot_dir"; then
            success_count=$((success_count + 1))
            
            # Update metadata
            local collection_info
            collection_info=$(qdrant::api::request "GET" "/collections/${collection}" 2>/dev/null)
            if [[ $? -eq 0 ]]; then
                local vector_count
                vector_count=$(echo "$collection_info" | jq -r '.result.vectors_count // 0' 2>/dev/null || echo "0")
                
                # Add collection to metadata
                local temp_metadata
                temp_metadata=$(jq --arg name "$collection" --arg count "$vector_count" \
                    '.collections += [{"name": $name, "vector_count": ($count | tonumber), "status": "success"}]' \
                    "$metadata_file" 2>/dev/null)
                
                if [[ -n "$temp_metadata" ]]; then
                    echo "$temp_metadata" > "$metadata_file"
                fi
            fi
        else
            log::warn "  Failed to create snapshot for collection: $collection"
            
            # Add failed collection to metadata
            local temp_metadata
            temp_metadata=$(jq --arg name "$collection" \
                '.collections += [{"name": $name, "vector_count": 0, "status": "failed"}]' \
                "$metadata_file" 2>/dev/null)
            
            if [[ -n "$temp_metadata" ]]; then
                echo "$temp_metadata" > "$metadata_file"
            fi
        fi
    done
    
    # Create archive
    local archive_path="${QDRANT_SNAPSHOTS_DIR}/${snapshot_name}.tar.gz"
    
    if tar -czf "$archive_path" -C "${QDRANT_SNAPSHOTS_DIR}" "$snapshot_name" 2>/dev/null; then
        # Remove temporary directory
        trash::safe_remove "$snapshot_dir" --no-confirm
        
        local archive_size
        archive_size=$(du -h "$archive_path" | cut -f1)
        
        if [[ $success_count -eq $total_count ]]; then
            log::success "${MSG_SNAPSHOT_CREATED}: $snapshot_name.tar.gz ($archive_size)"
            log::info "Snapshot location: $archive_path"
            return 0
        else
            log::warn "Snapshot created with some failures: $success_count/$total_count collections"
            log::info "Snapshot location: $archive_path"
            return 1
        fi
    else
        log::error "${MSG_SNAPSHOT_FAILED}: Failed to create archive"
        trash::safe_remove "$snapshot_dir" --no-confirm
        return 1
    fi
}

#######################################
# Create snapshot for a single collection
# Arguments:
#   $1 - Collection name
#   $2 - Snapshot directory
# Returns: 0 on success, 1 on failure
#######################################
qdrant::snapshots::create_collection_snapshot() {
    local collection_name="$1"
    local snapshot_dir="$2"
    
    # Create collection snapshot via API
    local response
    response=$(qdrant::api::request "POST" "/collections/${collection_name}/snapshots" "{}" 2>/dev/null)
    
    if [[ $? -ne 0 ]]; then
        return 1
    fi
    
    local status
    status=$(echo "$response" | jq -r '.status // "unknown"' 2>/dev/null)
    
    if [[ "$status" != "ok" ]]; then
        return 1
    fi
    
    # Get snapshot name from response
    local snapshot_file
    snapshot_file=$(echo "$response" | jq -r '.result.name // ""' 2>/dev/null)
    
    if [[ -z "$snapshot_file" ]]; then
        return 1
    fi
    
    # Wait a moment for snapshot to be ready
    sleep 2
    
    # Download the snapshot
    local download_url="${QDRANT_BASE_URL}/collections/${collection_name}/snapshots/${snapshot_file}"
    local local_file="${snapshot_dir}/${collection_name}.snapshot"
    
    local curl_cmd=(
        "curl" "-s" "-o" "$local_file"
        "--max-time" "300"  # 5 minutes timeout for large snapshots
    )
    
    # Add authentication if required
    if [[ -n "${QDRANT_API_KEY:-}" ]]; then
        curl_cmd+=("-H" "api-key: ${QDRANT_API_KEY}")
    fi
    
    curl_cmd+=("$download_url")
    
    if "${curl_cmd[@]}"; then
        # Verify the file was downloaded
        if [[ -f "$local_file" && -s "$local_file" ]]; then
            # Clean up server-side snapshot
            qdrant::api::request "DELETE" "/collections/${collection_name}/snapshots/${snapshot_file}" >/dev/null 2>&1
            return 0
        else
            trash::safe_remove "$local_file" --temp
            return 1 
        fi
    else
        return 1
    fi
}

#######################################
# List available snapshots
# Outputs: List of available snapshots
# Returns: 0 on success, 1 on failure
#######################################
qdrant::snapshots::list() {
    echo "=== Available Qdrant Snapshots ==="
    echo
    
    if [[ ! -d "${QDRANT_SNAPSHOTS_DIR}" ]]; then
        echo "No snapshots directory found"
        return 0
    fi
    
    local snapshots
    snapshots=$(find "${QDRANT_SNAPSHOTS_DIR}" -name "*.tar.gz" -type f 2>/dev/null | sort -r)
    
    if [[ -z "$snapshots" ]]; then
        echo "No snapshots found"
        return 0
    fi
    
    while IFS= read -r snapshot_path; do
        local snapshot_file
        snapshot_file=$(basename "$snapshot_path")
        local snapshot_name="${snapshot_file%.tar.gz}"
        local file_size
        file_size=$(du -h "$snapshot_path" | cut -f1)
        local file_date
        file_date=$(date -r "$snapshot_path" '+%Y-%m-%d %H:%M:%S' 2>/dev/null || echo "unknown")
        
        echo "Name: $snapshot_name"
        echo "  Size: $file_size"
        echo "  Created: $file_date"
        echo "  Path: $snapshot_path"
        
        # Try to extract metadata if available
        local temp_dir=$(mktemp -d)
        if tar -xzf "$snapshot_path" -C "$temp_dir" "${snapshot_name}/metadata.json" 2>/dev/null; then
            local metadata_file="${temp_dir}/${snapshot_name}/metadata.json"
            if [[ -f "$metadata_file" ]]; then
                local collections_count
                collections_count=$(jq -r '.total_collections // "unknown"' "$metadata_file" 2>/dev/null)
                echo "  Collections: $collections_count"
            fi
        fi
        trash::safe_remove "$temp_dir" --no-confirm
        
        echo
    done <<< "$snapshots"
}

#######################################
# Restore from a snapshot
# Arguments:
#   $1 - Snapshot name (without .tar.gz extension)
# Returns: 0 on success, 1 on failure
#######################################
qdrant::snapshots::restore() {
    local snapshot_name="$1"
    
    if [[ -z "$snapshot_name" ]]; then
        log::error "Snapshot name is required"
        return 1
    fi
    
    local snapshot_path="${QDRANT_SNAPSHOTS_DIR}/${snapshot_name}.tar.gz"
    
    if [[ ! -f "$snapshot_path" ]]; then
        log::error "Snapshot not found: $snapshot_path"
        return 1
    fi
    
    log::info "Restoring from snapshot: $snapshot_name"
    
    # Extract snapshot to temporary directory
    local temp_dir=$(mktemp -d)
    
    if ! tar -xzf "$snapshot_path" -C "$temp_dir" 2>/dev/null; then
        log::error "Failed to extract snapshot"
        trash::safe_remove "$temp_dir" --no-confirm
        return 1
    fi
    
    local snapshot_content_dir="${temp_dir}/${snapshot_name}"
    
    # Read metadata
    local metadata_file="${snapshot_content_dir}/metadata.json"
    local collections_to_restore=()
    
    if [[ -f "$metadata_file" ]]; then
        log::info "Reading snapshot metadata..."
        
        local created_at
        created_at=$(jq -r '.created_at // "unknown"' "$metadata_file" 2>/dev/null)
        local total_collections
        total_collections=$(jq -r '.total_collections // 0' "$metadata_file" 2>/dev/null)
        
        log::info "  Created: $created_at"
        log::info "  Collections: $total_collections"
        
        # Get list of collections from metadata
        while IFS= read -r collection_name; do
            if [[ -n "$collection_name" ]]; then
                collections_to_restore+=("$collection_name")
            fi
        done < <(jq -r '.collections[]?.name // empty' "$metadata_file" 2>/dev/null)
    else
        # Fallback: find .snapshot files
        while IFS= read -r snapshot_file; do
            if [[ -n "$snapshot_file" ]]; then
                local collection_name
                collection_name=$(basename "$snapshot_file" .snapshot)
                collections_to_restore+=("$collection_name")
            fi
        done < <(find "$snapshot_content_dir" -name "*.snapshot" -type f 2>/dev/null)
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
        
        if qdrant::snapshots::restore_collection "$collection" "$snapshot_content_dir"; then
            success_count=$((success_count + 1))
        else
            log::warn "  Failed to restore collection: $collection"
        fi
    done
    
    # Cleanup
    trash::safe_remove "$temp_dir" --no-confirm
    
    if [[ $success_count -eq $total_count ]]; then
        log::success "${MSG_BACKUP_RESTORED}: $snapshot_name"
        log::info "Restored $success_count collections successfully"
        return 0
    elif [[ $success_count -gt 0 ]]; then
        log::warn "Partial restore completed: $success_count/$total_count collections"
        return 1
    else
        log::error "${MSG_RESTORE_FAILED}: No collections were restored"
        return 1
    fi
}

#######################################
# Restore a single collection from snapshot
# Arguments:
#   $1 - Collection name
#   $2 - Snapshot content directory
# Returns: 0 on success, 1 on failure
#######################################
qdrant::snapshots::restore_collection() {
    local collection_name="$1"
    local snapshot_dir="$2"
    local snapshot_file="${snapshot_dir}/${collection_name}.snapshot"
    
    if [[ ! -f "$snapshot_file" ]]; then
        return 1
    fi
    
    # Upload snapshot file
    local upload_response
    upload_response=$(curl -s -X POST \
        "${QDRANT_BASE_URL}/collections/${collection_name}/snapshots/upload" \
        $(if [[ -n "${QDRANT_API_KEY:-}" ]]; then echo "-H \"api-key: ${QDRANT_API_KEY}\""; fi) \
        -F "snapshot=@${snapshot_file}" \
        --max-time 600 2>/dev/null)
    
    if [[ $? -eq 0 ]]; then
        local status
        status=$(echo "$upload_response" | jq -r '.status // "unknown"' 2>/dev/null)
        
        if [[ "$status" == "ok" ]]; then
            # Wait for restore to complete
            sleep 3
            
            # Verify collection was restored
            if qdrant::collections::exists "$collection_name"; then
                return 0
            fi
        fi
    fi
    
    return 1
}

#######################################
# Delete a snapshot
# Arguments:
#   $1 - Snapshot name (without .tar.gz extension)
# Returns: 0 on success, 1 on failure
#######################################
qdrant::snapshots::delete() {
    local snapshot_name="$1"
    
    if [[ -z "$snapshot_name" ]]; then
        log::error "Snapshot name is required"
        return 1
    fi
    
    local snapshot_path="${QDRANT_SNAPSHOTS_DIR}/${snapshot_name}.tar.gz"
    
    if [[ ! -f "$snapshot_path" ]]; then
        log::warn "Snapshot not found: $snapshot_name"
        return 0
    fi
    
    if rm "$snapshot_path" 2>/dev/null; then
        log::success "Snapshot deleted: $snapshot_name"
        return 0
    else
        log::error "Failed to delete snapshot: $snapshot_name"
        return 1
    fi
}

#######################################
# Clean up old snapshots
# Arguments:
#   $1 - Number of snapshots to keep (default: 5)
# Returns: 0 on success, 1 on failure
#######################################
qdrant::snapshots::cleanup() {
    local keep_count="${1:-5}"
    
    if [[ ! -d "${QDRANT_SNAPSHOTS_DIR}" ]]; then
        return 0
    fi
    
    local snapshots
    snapshots=$(find "${QDRANT_SNAPSHOTS_DIR}" -name "*.tar.gz" -type f -printf '%T@ %p\n' 2>/dev/null | \
                sort -rn | \
                tail -n +$((keep_count + 1)) | \
                cut -d' ' -f2-)
    
    if [[ -z "$snapshots" ]]; then
        log::info "No old snapshots to clean up"
        return 0
    fi
    
    local deleted_count=0
    
    while IFS= read -r snapshot_path; do
        if [[ -n "$snapshot_path" ]]; then
            local snapshot_name
            snapshot_name=$(basename "$snapshot_path" .tar.gz)
            
            if rm "$snapshot_path" 2>/dev/null; then
                log::info "Deleted old snapshot: $snapshot_name"
                deleted_count=$((deleted_count + 1))
            fi
        fi
    done <<< "$snapshots"
    
    if [[ $deleted_count -gt 0 ]]; then
        log::success "Cleaned up $deleted_count old snapshots"
    fi
    
    return 0
}