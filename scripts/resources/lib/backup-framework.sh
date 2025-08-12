#!/usr/bin/env bash
# Simplified Backup Framework
# Provides easy backup storage, retrieval, and smart iteration for resources

# Source required utilities
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck disable=SC1091
source "${SCRIPT_DIR}/../../lib/utils/var.sh"
# shellcheck disable=SC1091
source "${SCRIPT_DIR}/../../lib/utils/log.sh"

# Global backup storage location
if [[ -z "${BACKUP_ROOT:-}" ]]; then
    readonly BACKUP_ROOT="${HOME}/.vrooli/backups"
fi

#######################################
# Store backup data for a resource
# Args: 
#   $1 - resource name
#   $2 - source path (file or directory)
#   $3 - optional label (default: "backup")
# Returns: backup path on success, empty on failure
#######################################
backup::store() {
    local resource="$1"
    local source_path="$2"
    local label="${3:-backup}"
    
    if [[ -z "$resource" || -z "$source_path" ]]; then
        log::error "backup::store requires resource name and source path"
        return 1
    fi
    
    if [[ ! -e "$source_path" ]]; then
        log::error "Source path does not exist: $source_path"
        return 1
    fi
    
    local backup_dir="${BACKUP_ROOT}/${resource}"
    local timestamp=$(date +%Y%m%d_%H%M%S)
    local backup_path="${backup_dir}/${timestamp}_${label}"
    
    # Create backup directory structure
    mkdir -p "$backup_path" || {
        log::error "Failed to create backup directory: $backup_path"
        return 1
    }
    
    log::info "Creating backup for '$resource' at: $backup_path" >&2
    
    # Copy source to backup location - let cp handle files vs directories
    if [[ -d "$source_path" ]]; then
        # Directory: copy contents
        if ! cp -r "$source_path"/* "$backup_path/" 2>/dev/null; then
            # Handle empty directories
            if [[ $(ls -A "$source_path" 2>/dev/null | wc -l) -eq 0 ]]; then
                log::info "Source directory is empty: $source_path" >&2
            else
                log::error "Failed to copy directory contents from: $source_path" >&2
                rm -rf "$backup_path"
                return 1
            fi
        fi
    else
        # File: copy to backup directory
        if ! cp "$source_path" "$backup_path/"; then
            log::error "Failed to copy file from: $source_path" >&2
            rm -rf "$backup_path"
            return 1
        fi
    fi
    
    # Create simple metadata
    local size
    size=$(du -sb "$backup_path" 2>/dev/null | cut -f1 || echo "0")
    
    cat > "$backup_path/.metadata.json" << EOF
{
    "resource": "$resource",
    "label": "$label",
    "created": "$(date -Iseconds)",
    "source_path": "$source_path",
    "size_bytes": $size
}
EOF
    
    log::success "Backup created: $(basename "$backup_path")" >&2
    
    # Auto-cleanup based on resource policies
    backup::cleanup "$resource"
    
    echo "$backup_path"
}

#######################################
# Get the latest backup for a resource
# Args: $1 - resource name
# Returns: path to latest backup or empty if none
#######################################
backup::get_latest() {
    local resource="$1"
    local backup_dir="${BACKUP_ROOT}/${resource}"
    
    [[ -d "$backup_dir" ]] || return 1
    
    # Return newest backup directory
    ls -dt "$backup_dir"/*/ 2>/dev/null | head -1 | sed 's:/$::'
}

#######################################
# Get backup by ID (timestamp_label format)
# Args: 
#   $1 - resource name
#   $2 - backup ID
# Returns: path to backup or empty if not found
#######################################
backup::get_by_id() {
    local resource="$1"
    local backup_id="$2"
    local backup_path="${BACKUP_ROOT}/${resource}/${backup_id}"
    
    [[ -d "$backup_path" ]] && echo "$backup_path"
}

#######################################
# List all backups for a resource
# Args: $1 - resource name
# Returns: list of backup paths (newest first)
#######################################
backup::list() {
    local resource="$1"
    local backup_dir="${BACKUP_ROOT}/${resource}"
    
    [[ -d "$backup_dir" ]] || return 1
    
    ls -dt "$backup_dir"/*/ 2>/dev/null | sed 's:/$::'
}

#######################################
# Iterate through all backups for a resource (newest first)
# Args: 
#   $1 - resource name
#   $2 - callback function name
# Note: callback receives backup path as argument
#       callback can return 1 to stop iteration early
#######################################
backup::foreach() {
    local resource="$1"
    local callback="$2"
    
    if [[ -z "$callback" ]]; then
        log::error "backup::foreach requires callback function"
        return 1
    fi
    
    local backup_dir="${BACKUP_ROOT}/${resource}"
    [[ -d "$backup_dir" ]] || return 1
    
    # Iterate newest to oldest
    for backup in $(ls -dt "$backup_dir"/*/ 2>/dev/null); do
        # Remove trailing slash
        backup="${backup%/}"
        
        # Call callback function
        if ! "$callback" "$backup"; then
            # Callback returned non-zero, stop iteration
            return 0
        fi
    done
}

#######################################
# Find first backup that passes test function
# Args: 
#   $1 - resource name
#   $2 - test function name
# Returns: path to first matching backup or empty if none
# Note: test function receives backup path, returns 0 if match
#######################################
backup::find_first() {
    local resource="$1"
    local test_func="$2"
    
    if [[ -z "$test_func" ]]; then
        log::error "backup::find_first requires test function"
        return 1
    fi
    
    local found_backup=""
    
    # Use helper function to capture result
    _backup_find_helper() {
        local backup_path="$1"
        if "$test_func" "$backup_path"; then
            found_backup="$backup_path"
            return 1  # Stop iteration
        fi
        return 0  # Continue iteration
    }
    
    backup::foreach "$resource" "_backup_find_helper"
    
    [[ -n "$found_backup" ]] && echo "$found_backup"
}

#######################################
# Clean up old backups based on resource policies
# Args: $1 - resource name
# Returns: 0 always (cleanup is best-effort)
#######################################
backup::cleanup() {
    local resource="$1"
    local backup_dir="${BACKUP_ROOT}/${resource}"
    
    [[ -d "$backup_dir" ]] || return 0
    
    # Get resource-specific policies (uppercase resource name)
    local resource_upper="${resource^^}"
    local max_count_var="${resource_upper}_BACKUP_MAX_COUNT"
    local max_size_var="${resource_upper}_BACKUP_MAX_SIZE_MB"
    local min_age_var="${resource_upper}_BACKUP_MIN_AGE_DAYS"
    
    # Default policies
    local max_count="${!max_count_var:-20}"
    local max_size_mb="${!max_size_var:-1000}"
    local min_age_days="${!min_age_var:-0}"
    
    log::debug "Cleanup policies for $resource: count=$max_count, size=${max_size_mb}MB, age=${min_age_days}d"
    
    # Remove excess backups beyond count limit (but respect min age)
    local backups=($(ls -dt "$backup_dir"/*/ 2>/dev/null))
    local backup_count=${#backups[@]}
    
    if [[ $backup_count -gt $max_count ]]; then
        log::info "Removing old backups (keeping $max_count most recent)"
        
        # Remove oldest backups beyond limit, but check age
        for ((i=$max_count; i<$backup_count; i++)); do
            local backup_path="${backups[$i]%/}"  # Remove trailing slash
            
            # Check minimum age (skip if too young)
            if [[ $min_age_days -gt 0 ]]; then
                local backup_age
                backup_age=$(find "$backup_path" -maxdepth 0 -type d -mtime +$min_age_days 2>/dev/null)
                if [[ -z "$backup_age" ]]; then
                    log::debug "Skipping backup (too young): $(basename "$backup_path")"
                    continue
                fi
            fi
            
            log::info "Removing old backup: $(basename "$backup_path")"
            rm -rf "$backup_path"
        done
    fi
    
    # Remove backups if total size exceeds limit
    if [[ $max_size_mb -gt 0 ]]; then
        local current_size
        current_size=$(du -sm "$backup_dir" 2>/dev/null | cut -f1 || echo "0")
        
        if [[ $current_size -gt $max_size_mb ]]; then
            log::info "Backup size ${current_size}MB exceeds limit ${max_size_mb}MB, removing oldest backups"
            
            # Remove oldest backups until under size limit
            local remaining_backups=($(ls -dt "$backup_dir"/*/ 2>/dev/null))
            for backup in "${remaining_backups[@]}"; do
                local backup_path="${backup%/}"
                current_size=$(du -sm "$backup_dir" 2>/dev/null | cut -f1 || echo "0")
                
                if [[ $current_size -le $max_size_mb ]]; then
                    break
                fi
                
                # Check minimum age
                if [[ $min_age_days -gt 0 ]]; then
                    local backup_age
                    backup_age=$(find "$backup_path" -maxdepth 0 -type d -mtime +$min_age_days 2>/dev/null)
                    if [[ -z "$backup_age" ]]; then
                        continue  # Skip young backups
                    fi
                fi
                
                log::info "Removing backup for size limit: $(basename "$backup_path")"
                rm -rf "$backup_path"
            done
        fi
    fi
    
    return 0
}

#######################################
# Get backup information and statistics
# Args: $1 - resource name (optional, shows all if empty)
#######################################
backup::info() {
    local resource="$1"
    
    if [[ -n "$resource" ]]; then
        # Show info for specific resource
        local backup_dir="${BACKUP_ROOT}/${resource}"
        if [[ ! -d "$backup_dir" ]]; then
            echo "No backups found for resource: $resource"
            return 0
        fi
        
        echo "Backups for resource '$resource':"
        echo "================================"
        
        local backups=($(ls -dt "$backup_dir"/*/ 2>/dev/null))
        local total_size
        total_size=$(du -sh "$backup_dir" 2>/dev/null | cut -f1 || echo "0")
        
        echo "Total backups: ${#backups[@]}"
        echo "Total size: $total_size"
        echo
        
        if [[ ${#backups[@]} -gt 0 ]]; then
            printf "%-25s %-15s %-10s %s\n" "Backup ID" "Label" "Size" "Created"
            printf "%-25s %-15s %-10s %s\n" "---------" "-----" "----" "-------"
            
            for backup in "${backups[@]}"; do
                local backup_path="${backup%/}"
                local backup_id=$(basename "$backup_path")
                
                # Extract info from metadata if available
                local metadata_file="$backup_path/.metadata.json"
                if [[ -f "$metadata_file" ]]; then
                    local label=$(jq -r '.label // "unknown"' "$metadata_file" 2>/dev/null)
                    local created=$(jq -r '.created // "unknown"' "$metadata_file" 2>/dev/null)
                    local size=$(du -sh "$backup_path" 2>/dev/null | cut -f1 || echo "N/A")
                    
                    # Format created date
                    if [[ "$created" != "unknown" ]]; then
                        created=$(date -d "$created" "+%Y-%m-%d %H:%M" 2>/dev/null || echo "$created")
                    fi
                    
                    printf "%-25s %-15s %-10s %s\n" "$backup_id" "$label" "$size" "$created"
                else
                    printf "%-25s %-15s %-10s %s\n" "$backup_id" "unknown" "$(du -sh "$backup_path" | cut -f1)" "$(date -r "$backup_path")"
                fi
            done
        fi
    else
        # Show info for all resources
        echo "All Resource Backups:"
        echo "===================="
        
        if [[ ! -d "$BACKUP_ROOT" ]]; then
            echo "No backup directory found: $BACKUP_ROOT"
            return 0
        fi
        
        local resources=($(find "$BACKUP_ROOT" -maxdepth 1 -type d -name "*" | grep -v "^${BACKUP_ROOT}$" | sort))
        
        if [[ ${#resources[@]} -eq 0 ]]; then
            echo "No resource backups found"
            return 0
        fi
        
        for resource_dir in "${resources[@]}"; do
            local resource_name=$(basename "$resource_dir")
            local backup_count=$(ls -1 "$resource_dir" 2>/dev/null | wc -l)
            local total_size=$(du -sh "$resource_dir" 2>/dev/null | cut -f1 || echo "0")
            
            printf "%-20s %3d backups %10s\n" "$resource_name" "$backup_count" "$total_size"
        done
    fi
}

#######################################
# Delete specific backup
# Args: 
#   $1 - resource name
#   $2 - backup ID
# Returns: 0 on success, 1 on failure
#######################################
backup::delete() {
    local resource="$1"
    local backup_id="$2"
    
    if [[ -z "$resource" || -z "$backup_id" ]]; then
        log::error "backup::delete requires resource name and backup ID"
        return 1
    fi
    
    local backup_path="${BACKUP_ROOT}/${resource}/${backup_id}"
    
    if [[ ! -d "$backup_path" ]]; then
        log::error "Backup not found: $backup_id"
        return 1
    fi
    
    log::info "Deleting backup: $backup_id"
    rm -rf "$backup_path"
    
    log::success "Backup deleted: $backup_id"
    return 0
}

#######################################
# Initialize backup framework
# Creates necessary directories and sets up environment
#######################################
backup::init() {
    mkdir -p "$BACKUP_ROOT" || {
        log::error "Failed to create backup root directory: $BACKUP_ROOT"
        return 1
    }
    
    log::info "Backup framework initialized at: $BACKUP_ROOT"
    return 0
}

# Initialize on source
backup::init >/dev/null 2>&1

# Export functions for use by resources
export -f backup::store
export -f backup::get_latest  
export -f backup::get_by_id
export -f backup::list
export -f backup::foreach
export -f backup::find_first
export -f backup::cleanup
export -f backup::info
export -f backup::delete
export -f backup::init