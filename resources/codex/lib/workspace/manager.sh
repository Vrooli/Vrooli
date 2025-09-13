#!/usr/bin/env bash
################################################################################
# Workspace Manager - Sandboxing and Security for Code Execution
# 
# Manages isolated workspaces for safe code execution with security controls
# Provides workspace creation, cleanup, monitoring, and access control
################################################################################

# Setup paths
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../../.." && builtin pwd)}"
source "${APP_ROOT}/scripts/lib/utils/log.sh"

# Workspace configuration
WORKSPACE_BASE_DIR="${CODEX_WORKSPACE_BASE:-/tmp/codex-workspaces}"
WORKSPACE_MAX_SIZE="${CODEX_WORKSPACE_MAX_SIZE:-100M}"
WORKSPACE_MAX_FILES="${CODEX_WORKSPACE_MAX_FILES:-1000}"
WORKSPACE_TIMEOUT="${CODEX_WORKSPACE_TIMEOUT:-3600}"  # 1 hour default

################################################################################
# Workspace Management Interface
################################################################################

#######################################
# Create a new workspace
# Arguments:
#   $1 - Workspace ID (optional, auto-generated if not provided)
#   $2 - Security level (strict, moderate, relaxed)
#   $3 - Options (JSON, optional)
# Returns:
#   Workspace info JSON
#######################################
workspace_manager::create() {
    local workspace_id="${1:-$(workspace_manager::generate_id)}"
    local security_level="${2:-moderate}"
    local options="${3:-{}}"
    
    log::info "Creating workspace: $workspace_id with security level: $security_level"
    
    # Validate workspace ID
    if ! workspace_manager::validate_id "$workspace_id"; then
        echo '{"success": false, "error": "Invalid workspace ID"}'
        return 1
    fi
    
    # Create workspace directory structure
    local workspace_dir="$WORKSPACE_BASE_DIR/$workspace_id"
    
    if [[ -d "$workspace_dir" ]]; then
        echo '{"success": false, "error": "Workspace already exists"}'
        return 1
    fi
    
    # Create directory structure
    mkdir -p "$workspace_dir"/{data,tmp,logs,config} || {
        echo '{"success": false, "error": "Failed to create workspace directories"}'
        return 1
    }
    
    # Set up workspace configuration
    local config_file="$workspace_dir/config/workspace.json"
    workspace_manager::create_config "$workspace_id" "$security_level" "$options" > "$config_file"
    
    # Set up security permissions
    workspace_manager::setup_security "$workspace_dir" "$security_level"
    
    # Create workspace metadata
    local metadata_file="$workspace_dir/config/metadata.json"
    workspace_manager::create_metadata "$workspace_id" > "$metadata_file"
    
    # Register workspace
    workspace_manager::register_workspace "$workspace_id" "$workspace_dir"
    
    # Set up monitoring if enabled
    if [[ $(echo "$options" | jq -r '.monitoring // false') == "true" ]]; then
        workspace_manager::setup_monitoring "$workspace_id"
    fi
    
    log::success "Workspace created: $workspace_id at $workspace_dir"
    
    # Return workspace info
    cat << EOF
{
  "success": true,
  "workspace_id": "$workspace_id",
  "workspace_dir": "$workspace_dir",
  "security_level": "$security_level",
  "created_at": "$(date -Iseconds)",
  "status": "active",
  "data_dir": "$workspace_dir/data",
  "tmp_dir": "$workspace_dir/tmp",
  "logs_dir": "$workspace_dir/logs"
}
EOF
}

#######################################
# Get workspace information
# Arguments:
#   $1 - Workspace ID
# Returns:
#   Workspace info JSON
#######################################
workspace_manager::get_info() {
    local workspace_id="$1"
    
    if [[ -z "$workspace_id" ]]; then
        echo '{"success": false, "error": "Workspace ID required"}'
        return 1
    fi
    
    local workspace_dir="$WORKSPACE_BASE_DIR/$workspace_id"
    
    if [[ ! -d "$workspace_dir" ]]; then
        echo '{"success": false, "error": "Workspace not found"}'
        return 1
    fi
    
    # Load workspace configuration and metadata
    local config_file="$workspace_dir/config/workspace.json"
    local metadata_file="$workspace_dir/config/metadata.json"
    
    local config="{}"
    local metadata="{}"
    
    if [[ -f "$config_file" ]]; then
        config=$(cat "$config_file" 2>/dev/null || echo '{}')
    fi
    
    if [[ -f "$metadata_file" ]]; then
        metadata=$(cat "$metadata_file" 2>/dev/null || echo '{}')
    fi
    
    # Get workspace statistics
    local stats
    stats=$(workspace_manager::get_statistics "$workspace_id")
    
    # Combine information
    jq -n \
        --argjson config "$config" \
        --argjson metadata "$metadata" \
        --argjson stats "$stats" \
        --arg workspace_dir "$workspace_dir" \
        '{
            success: true,
            workspace_id: $config.workspace_id,
            workspace_dir: $workspace_dir,
            security_level: $config.security_level,
            created_at: $metadata.created_at,
            last_accessed: $metadata.last_accessed,
            status: $metadata.status,
            statistics: $stats,
            configuration: $config
        }'
}

#######################################
# List all workspaces
# Arguments:
#   $1 - Filter (active, expired, all) - default: active
# Returns:
#   JSON array of workspaces
#######################################
workspace_manager::list() {
    local filter="${1:-active}"
    
    log::debug "Listing workspaces with filter: $filter"
    
    local workspaces="[]"
    
    if [[ ! -d "$WORKSPACE_BASE_DIR" ]]; then
        echo "$workspaces"
        return 0
    fi
    
    for workspace_dir in "$WORKSPACE_BASE_DIR"/*; do
        if [[ ! -d "$workspace_dir" ]]; then
            continue
        fi
        
        local workspace_id
        workspace_id=$(basename "$workspace_dir")
        
        # Check if workspace is valid
        if [[ ! -f "$workspace_dir/config/workspace.json" ]]; then
            continue
        fi
        
        # Get workspace status
        local workspace_info
        workspace_info=$(workspace_manager::get_info "$workspace_id")
        
        if [[ $(echo "$workspace_info" | jq -r '.success') == "true" ]]; then
            local status
            status=$(echo "$workspace_info" | jq -r '.status // "unknown"')
            
            # Apply filter
            case "$filter" in
                active)
                    if [[ "$status" == "active" ]]; then
                        workspaces=$(echo "$workspaces" | jq ". + [$workspace_info]")
                    fi
                    ;;
                expired)
                    if [[ "$status" == "expired" ]]; then
                        workspaces=$(echo "$workspaces" | jq ". + [$workspace_info]")
                    fi
                    ;;
                all)
                    workspaces=$(echo "$workspaces" | jq ". + [$workspace_info]")
                    ;;
            esac
        fi
    done
    
    echo "$workspaces"
}

#######################################
# Delete a workspace
# Arguments:
#   $1 - Workspace ID
#   $2 - Force deletion (true/false) - default: false
# Returns:
#   Operation result JSON
#######################################
workspace_manager::delete() {
    local workspace_id="$1"
    local force="${2:-false}"
    
    log::info "Deleting workspace: $workspace_id (force: $force)"
    
    if [[ -z "$workspace_id" ]]; then
        echo '{"success": false, "error": "Workspace ID required"}'
        return 1
    fi
    
    local workspace_dir="$WORKSPACE_BASE_DIR/$workspace_id"
    
    if [[ ! -d "$workspace_dir" ]]; then
        echo '{"success": false, "error": "Workspace not found"}'
        return 1
    fi
    
    # Check if workspace is in use (unless forced)
    if [[ "$force" != "true" ]]; then
        if workspace_manager::is_in_use "$workspace_id"; then
            echo '{"success": false, "error": "Workspace is currently in use"}'
            return 1
        fi
    fi
    
    # Create backup if requested
    local backup_created=false
    if [[ $(workspace_manager::get_config "$workspace_id" | jq -r '.backup_on_delete // false') == "true" ]]; then
        if workspace_manager::create_backup "$workspace_id"; then
            backup_created=true
        fi
    fi
    
    # Clean up monitoring
    workspace_manager::cleanup_monitoring "$workspace_id"
    
    # Unregister workspace
    workspace_manager::unregister_workspace "$workspace_id"
    
    # Remove workspace directory
    if rm -rf "$workspace_dir" 2>/dev/null; then
        log::success "Workspace deleted: $workspace_id"
        
        cat << EOF
{
  "success": true,
  "workspace_id": "$workspace_id",
  "deleted_at": "$(date -Iseconds)",
  "backup_created": $backup_created,
  "message": "Workspace successfully deleted"
}
EOF
    else
        log::error "Failed to delete workspace directory: $workspace_dir"
        echo '{"success": false, "error": "Failed to delete workspace directory"}'
        return 1
    fi
}

################################################################################
# Workspace Security Setup
################################################################################

#######################################
# Setup security for workspace
# Arguments:
#   $1 - Workspace directory
#   $2 - Security level
#######################################
workspace_manager::setup_security() {
    local workspace_dir="$1"
    local security_level="$2"
    
    log::debug "Setting up security for $workspace_dir with level: $security_level"
    
    case "$security_level" in
        strict)
            # Strict permissions - owner only
            chmod 700 "$workspace_dir"
            chmod -R 700 "$workspace_dir"/*
            
            # Create .security file with strict rules
            cat > "$workspace_dir/config/.security" << 'EOF'
SECURITY_LEVEL=strict
NETWORK_ACCESS=false
SYSTEM_CALLS=restricted
FILE_SYSTEM_ACCESS=workspace_only
PROCESS_LIMITS=strict
RESOURCE_LIMITS=strict
EOF
            ;;
        moderate)
            # Moderate permissions - owner read/write, group read
            chmod 750 "$workspace_dir"
            chmod -R 640 "$workspace_dir"/*
            chmod u+x "$workspace_dir"/{data,tmp,logs,config}
            
            cat > "$workspace_dir/config/.security" << 'EOF'
SECURITY_LEVEL=moderate
NETWORK_ACCESS=limited
SYSTEM_CALLS=filtered
FILE_SYSTEM_ACCESS=workspace_and_tmp
PROCESS_LIMITS=moderate
RESOURCE_LIMITS=moderate
EOF
            ;;
        relaxed)
            # Relaxed permissions - more permissive but still contained
            chmod 755 "$workspace_dir"
            chmod -R 644 "$workspace_dir"/*
            chmod u+x "$workspace_dir"/{data,tmp,logs,config}
            
            cat > "$workspace_dir/config/.security" << 'EOF'
SECURITY_LEVEL=relaxed
NETWORK_ACCESS=true
SYSTEM_CALLS=monitored
FILE_SYSTEM_ACCESS=extended
PROCESS_LIMITS=relaxed
RESOURCE_LIMITS=relaxed
EOF
            ;;
    esac
    
    # Set up resource limits
    workspace_manager::setup_resource_limits "$workspace_dir" "$security_level"
}

#######################################
# Setup resource limits for workspace
# Arguments:
#   $1 - Workspace directory
#   $2 - Security level
#######################################
workspace_manager::setup_resource_limits() {
    local workspace_dir="$1"
    local security_level="$2"
    
    # Create resource limits configuration
    local limits_file="$workspace_dir/config/resource_limits.json"
    
    case "$security_level" in
        strict)
            cat > "$limits_file" << 'EOF'
{
  "disk_space": "50M",
  "max_files": 500,
  "max_processes": 5,
  "cpu_time": 300,
  "memory": "128M",
  "network_connections": 0,
  "execution_timeout": 60
}
EOF
            ;;
        moderate)
            cat > "$limits_file" << 'EOF'
{
  "disk_space": "100M",
  "max_files": 1000,
  "max_processes": 10,
  "cpu_time": 600,
  "memory": "256M",
  "network_connections": 5,
  "execution_timeout": 300
}
EOF
            ;;
        relaxed)
            cat > "$limits_file" << 'EOF'
{
  "disk_space": "500M",
  "max_files": 5000,
  "max_processes": 25,
  "cpu_time": 1800,
  "memory": "512M",
  "network_connections": 20,
  "execution_timeout": 900
}
EOF
            ;;
    esac
    
    # Set up disk quota if available
    if command -v quota >/dev/null 2>&1; then
        workspace_manager::setup_disk_quota "$workspace_dir" "$security_level"
    fi
}

################################################################################
# Workspace Utilities
################################################################################

#######################################
# Generate unique workspace ID
# Returns:
#   Workspace ID string
#######################################
workspace_manager::generate_id() {
    local timestamp=$(date +%s)
    local random=$(tr -dc 'a-z0-9' < /dev/urandom | head -c 8)
    echo "ws-${timestamp}-${random}"
}

#######################################
# Validate workspace ID format
# Arguments:
#   $1 - Workspace ID
# Returns:
#   0 if valid, 1 if invalid
#######################################
workspace_manager::validate_id() {
    local workspace_id="$1"
    
    # Check format: alphanumeric, hyphens, underscores only
    if [[ ! "$workspace_id" =~ ^[a-zA-Z0-9_-]+$ ]]; then
        return 1
    fi
    
    # Check length
    if [[ ${#workspace_id} -lt 3 || ${#workspace_id} -gt 64 ]]; then
        return 1
    fi
    
    # Check for reserved names
    local reserved_names=("system" "admin" "root" "tmp" "config" "logs")
    for reserved in "${reserved_names[@]}"; do
        if [[ "$workspace_id" == "$reserved" ]]; then
            return 1
        fi
    done
    
    return 0
}

#######################################
# Create workspace configuration
# Arguments:
#   $1 - Workspace ID
#   $2 - Security level
#   $3 - Options JSON
# Returns:
#   Configuration JSON
#######################################
workspace_manager::create_config() {
    local workspace_id="$1"
    local security_level="$2"
    local options="$3"
    
    # Extract options
    local description auto_cleanup backup_on_delete monitoring
    description=$(echo "$options" | jq -r '.description // ""')
    auto_cleanup=$(echo "$options" | jq -r '.auto_cleanup // true')
    backup_on_delete=$(echo "$options" | jq -r '.backup_on_delete // false')
    monitoring=$(echo "$options" | jq -r '.monitoring // false')
    
    jq -n \
        --arg workspace_id "$workspace_id" \
        --arg security_level "$security_level" \
        --arg description "$description" \
        --argjson auto_cleanup "$auto_cleanup" \
        --argjson backup_on_delete "$backup_on_delete" \
        --argjson monitoring "$monitoring" \
        --arg created_at "$(date -Iseconds)" \
        '{
            workspace_id: $workspace_id,
            security_level: $security_level,
            description: $description,
            auto_cleanup: $auto_cleanup,
            backup_on_delete: $backup_on_delete,
            monitoring: $monitoring,
            created_at: $created_at,
            version: "1.0"
        }'
}

#######################################
# Create workspace metadata
# Arguments:
#   $1 - Workspace ID
# Returns:
#   Metadata JSON
#######################################
workspace_manager::create_metadata() {
    local workspace_id="$1"
    
    jq -n \
        --arg workspace_id "$workspace_id" \
        --arg created_at "$(date -Iseconds)" \
        --arg last_accessed "$(date -Iseconds)" \
        --arg status "active" \
        --arg owner "$(whoami)" \
        --arg hostname "$(hostname)" \
        '{
            workspace_id: $workspace_id,
            created_at: $created_at,
            last_accessed: $last_accessed,
            status: $status,
            owner: $owner,
            hostname: $hostname,
            access_count: 0,
            last_operation: "created"
        }'
}

#######################################
# Get workspace statistics
# Arguments:
#   $1 - Workspace ID
# Returns:
#   Statistics JSON
#######################################
workspace_manager::get_statistics() {
    local workspace_id="$1"
    local workspace_dir="$WORKSPACE_BASE_DIR/$workspace_id"
    
    if [[ ! -d "$workspace_dir" ]]; then
        echo '{"error": "Workspace not found"}'
        return 1
    fi
    
    # Calculate disk usage
    local disk_usage
    disk_usage=$(du -sb "$workspace_dir" 2>/dev/null | cut -f1 || echo "0")
    
    # Count files
    local file_count
    file_count=$(find "$workspace_dir" -type f 2>/dev/null | wc -l || echo "0")
    
    # Count directories
    local dir_count
    dir_count=$(find "$workspace_dir" -type d 2>/dev/null | wc -l || echo "0")
    
    # Get last modified time
    local last_modified
    last_modified=$(find "$workspace_dir" -type f -exec stat -f%m {} \; 2>/dev/null | sort -n | tail -1 || echo "0")
    if [[ "$last_modified" != "0" ]]; then
        last_modified=$(date -r "$last_modified" -Iseconds 2>/dev/null || echo "unknown")
    else
        last_modified="unknown"
    fi
    
    jq -n \
        --arg disk_usage "$disk_usage" \
        --arg file_count "$file_count" \
        --arg dir_count "$dir_count" \
        --arg last_modified "$last_modified" \
        '{
            disk_usage_bytes: ($disk_usage | tonumber),
            disk_usage_human: ($disk_usage | tonumber | if . > 1073741824 then "\(. / 1073741824 | floor)GB" elif . > 1048576 then "\(. / 1048576 | floor)MB" elif . > 1024 then "\(. / 1024 | floor)KB" else "\(.)B" end),
            file_count: ($file_count | tonumber),
            directory_count: ($dir_count | tonumber),
            last_modified: $last_modified
        }'
}

################################################################################
# Workspace Registry
################################################################################

#######################################
# Register workspace in global registry
# Arguments:
#   $1 - Workspace ID
#   $2 - Workspace directory
#######################################
workspace_manager::register_workspace() {
    local workspace_id="$1"
    local workspace_dir="$2"
    
    local registry_file="$WORKSPACE_BASE_DIR/.registry"
    mkdir -p "$WORKSPACE_BASE_DIR"
    
    # Create registry if it doesn't exist
    if [[ ! -f "$registry_file" ]]; then
        echo '{}' > "$registry_file"
    fi
    
    # Add workspace to registry
    local registry
    registry=$(cat "$registry_file" 2>/dev/null || echo '{}')
    
    registry=$(echo "$registry" | jq \
        --arg id "$workspace_id" \
        --arg dir "$workspace_dir" \
        --arg created "$(date -Iseconds)" \
        '. + {($id): {directory: $dir, created_at: $created, status: "active"}}')
    
    echo "$registry" > "$registry_file"
}

#######################################
# Unregister workspace from global registry
# Arguments:
#   $1 - Workspace ID
#######################################
workspace_manager::unregister_workspace() {
    local workspace_id="$1"
    local registry_file="$WORKSPACE_BASE_DIR/.registry"
    
    if [[ -f "$registry_file" ]]; then
        local registry
        registry=$(cat "$registry_file" 2>/dev/null || echo '{}')
        
        registry=$(echo "$registry" | jq --arg id "$workspace_id" 'del(.[$id])')
        
        echo "$registry" > "$registry_file"
    fi
}

#######################################
# Check if workspace is currently in use
# Arguments:
#   $1 - Workspace ID
# Returns:
#   0 if in use, 1 if not in use
#######################################
workspace_manager::is_in_use() {
    local workspace_id="$1"
    local workspace_dir="$WORKSPACE_BASE_DIR/$workspace_id"
    
    # Check for lock file
    if [[ -f "$workspace_dir/.lock" ]]; then
        local lock_pid
        lock_pid=$(cat "$workspace_dir/.lock" 2>/dev/null)
        
        # Check if process is still running
        if [[ -n "$lock_pid" ]] && kill -0 "$lock_pid" 2>/dev/null; then
            return 0
        else
            # Remove stale lock
            rm -f "$workspace_dir/.lock"
        fi
    fi
    
    # Check for active processes in workspace
    if pgrep -f "$workspace_dir" >/dev/null 2>&1; then
        return 0
    fi
    
    return 1
}

#######################################
# Get workspace configuration
# Arguments:
#   $1 - Workspace ID
# Returns:
#   Configuration JSON
#######################################
workspace_manager::get_config() {
    local workspace_id="$1"
    local workspace_dir="$WORKSPACE_BASE_DIR/$workspace_id"
    local config_file="$workspace_dir/config/workspace.json"
    
    if [[ -f "$config_file" ]]; then
        cat "$config_file"
    else
        echo '{}'
    fi
}

################################################################################
# Cleanup and Maintenance
################################################################################

#######################################
# Clean up expired workspaces
# Arguments:
#   $1 - Dry run (true/false) - default: false
# Returns:
#   Cleanup report JSON
#######################################
workspace_manager::cleanup_expired() {
    local dry_run="${1:-false}"
    
    log::info "Starting workspace cleanup (dry_run: $dry_run)"
    
    local cleaned_count=0
    local error_count=0
    local cleaned_workspaces=()
    local errors=()
    
    # Get all workspaces
    local workspaces
    workspaces=$(workspace_manager::list "all")
    
    # Process each workspace
    while IFS= read -r workspace; do
        local workspace_id created_at status
        workspace_id=$(echo "$workspace" | jq -r '.workspace_id')
        created_at=$(echo "$workspace" | jq -r '.created_at')
        status=$(echo "$workspace" | jq -r '.status')
        
        # Check if workspace has expired
        if workspace_manager::is_expired "$workspace_id" "$created_at"; then
            if [[ "$dry_run" == "true" ]]; then
                cleaned_workspaces+=("$workspace_id")
                ((cleaned_count++))
            else
                # Actually delete the workspace
                local delete_result
                delete_result=$(workspace_manager::delete "$workspace_id" "true")
                
                if [[ $(echo "$delete_result" | jq -r '.success') == "true" ]]; then
                    cleaned_workspaces+=("$workspace_id")
                    ((cleaned_count++))
                else
                    local error_msg
                    error_msg=$(echo "$delete_result" | jq -r '.error')
                    errors+=("$workspace_id: $error_msg")
                    ((error_count++))
                fi
            fi
        fi
    done < <(echo "$workspaces" | jq -c '.[]')
    
    # Build report
    local cleaned_json errors_json
    cleaned_json=$(printf '%s\n' "${cleaned_workspaces[@]}" | jq -R . | jq -s .)
    errors_json=$(printf '%s\n' "${errors[@]}" | jq -R . | jq -s .)
    
    cat << EOF
{
  "success": true,
  "dry_run": $dry_run,
  "cleaned_count": $cleaned_count,
  "error_count": $error_count,
  "cleaned_workspaces": $cleaned_json,
  "errors": $errors_json,
  "cleanup_time": "$(date -Iseconds)"
}
EOF
}

#######################################
# Check if workspace has expired
# Arguments:
#   $1 - Workspace ID
#   $2 - Created timestamp
# Returns:
#   0 if expired, 1 if not expired
#######################################
workspace_manager::is_expired() {
    local workspace_id="$1"
    local created_at="$2"
    
    # Get workspace timeout setting
    local config
    config=$(workspace_manager::get_config "$workspace_id")
    local timeout
    timeout=$(echo "$config" | jq -r '.timeout // null')
    
    # Use default timeout if not specified
    if [[ "$timeout" == "null" ]]; then
        timeout="$WORKSPACE_TIMEOUT"
    fi
    
    # Convert created_at to epoch
    local created_epoch
    created_epoch=$(date -d "$created_at" +%s 2>/dev/null || echo "0")
    
    local current_epoch
    current_epoch=$(date +%s)
    
    local age=$((current_epoch - created_epoch))
    
    if [[ $age -gt $timeout ]]; then
        return 0  # Expired
    else
        return 1  # Not expired
    fi
}

#######################################
# Setup workspace monitoring
# Arguments:
#   $1 - Workspace ID
#######################################
workspace_manager::setup_monitoring() {
    local workspace_id="$1"
    local workspace_dir="$WORKSPACE_BASE_DIR/$workspace_id"
    
    # Create monitoring script
    cat > "$workspace_dir/config/monitor.sh" << 'EOF'
#!/bin/bash
# Workspace monitoring script
WORKSPACE_ID="$1"
WORKSPACE_DIR="$2"

while true; do
    # Log workspace statistics
    echo "$(date -Iseconds) STATS $(du -sb "$WORKSPACE_DIR" | cut -f1) $(find "$WORKSPACE_DIR" -type f | wc -l)" >> "$WORKSPACE_DIR/logs/monitor.log"
    
    sleep 300  # 5 minute intervals
done
EOF
    
    chmod +x "$workspace_dir/config/monitor.sh"
    
    # Start monitoring in background (basic implementation)
    # Note: In production, this should use a proper process manager
    if command -v nohup >/dev/null 2>&1; then
        nohup "$workspace_dir/config/monitor.sh" "$workspace_id" "$workspace_dir" &
        echo $! > "$workspace_dir/.monitor_pid"
    fi
}

#######################################
# Cleanup workspace monitoring
# Arguments:
#   $1 - Workspace ID
#######################################
workspace_manager::cleanup_monitoring() {
    local workspace_id="$1"
    local workspace_dir="$WORKSPACE_BASE_DIR/$workspace_id"
    
    # Stop monitoring process
    if [[ -f "$workspace_dir/.monitor_pid" ]]; then
        local monitor_pid
        monitor_pid=$(cat "$workspace_dir/.monitor_pid")
        
        if [[ -n "$monitor_pid" ]] && kill -0 "$monitor_pid" 2>/dev/null; then
            kill "$monitor_pid" 2>/dev/null
        fi
        
        rm -f "$workspace_dir/.monitor_pid"
    fi
}

#######################################
# Create workspace backup
# Arguments:
#   $1 - Workspace ID
# Returns:
#   0 if successful, 1 if failed
#######################################
workspace_manager::create_backup() {
    local workspace_id="$1"
    local workspace_dir="$WORKSPACE_BASE_DIR/$workspace_id"
    
    local backup_dir="${WORKSPACE_BASE_DIR}/.backups"
    mkdir -p "$backup_dir"
    
    local backup_file="$backup_dir/${workspace_id}-$(date +%Y%m%d-%H%M%S).tar.gz"
    
    if tar -czf "$backup_file" -C "$WORKSPACE_BASE_DIR" "$workspace_id" 2>/dev/null; then
        log::info "Workspace backup created: $backup_file"
        return 0
    else
        log::error "Failed to create backup for workspace: $workspace_id"
        return 1
    fi
}

# Export functions
export -f workspace_manager::create
export -f workspace_manager::get_info
export -f workspace_manager::list
export -f workspace_manager::delete
export -f workspace_manager::cleanup_expired