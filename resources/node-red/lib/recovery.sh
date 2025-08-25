#!/usr/bin/env bash
# Node-RED Backup and Recovery Functions
# Provides comprehensive backup/recovery using backup framework

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
NODE_RED_RECOVERY_LIB_DIR="${APP_ROOT}/resources/node-red/lib"

# shellcheck disable=SC1091
source "${NODE_RED_RECOVERY_LIB_DIR}/../../../lib/utils/var.sh"
# shellcheck disable=SC1091
source "${var_SCRIPTS_RESOURCES_LIB_DIR}/backup-framework.sh"

#######################################
# Create comprehensive Node-RED backup
# Arguments:
#   $1 - backup label (optional, default: "manual")
# Returns: 0 on success, 1 on failure
#######################################
node_red::create_backup() {
    local label="${1:-manual}"
    local temp_dir
    temp_dir=$(mktemp -d)
    
    log::info "Creating Node-RED backup with label: $label"
    
    # Create backup structure
    mkdir -p "$temp_dir"/{data,config,flows}
    
    # Backup data directory if it exists
    if [[ -d "$NODE_RED_DATA_DIR" ]]; then
        log::info "Backing up data directory..."
        cp -r "$NODE_RED_DATA_DIR"/* "$temp_dir/data/" 2>/dev/null || true
    fi
    
    # Backup configuration files
    log::info "Backing up configuration files..."
    local config_files=(
        "${NODE_RED_SCRIPT_DIR}/config/defaults.sh"
        "${NODE_RED_SCRIPT_DIR}/config/messages.sh"
        "${NODE_RED_SCRIPT_DIR}/config/settings.js"
    )
    
    for config_file in "${config_files[@]}"; do
        if [[ -f "$config_file" ]]; then
            cp "$config_file" "$temp_dir/config/"
        fi
    done
    
    # Backup flows file
    node_red::backup_flows_file "$temp_dir/flows"
    
    # Create metadata file
    local metadata_file="$temp_dir/metadata.json"
    cat > "$metadata_file" << EOF
{
    "service": "node-red",
    "backup_type": "full",
    "label": "$label",
    "timestamp": "$(date -Iseconds)",
    "version": "$(node_red::get_version)",
    "container_name": "$NODE_RED_CONTAINER_NAME",
    "port": $NODE_RED_PORT,
    "data_dir": "$NODE_RED_DATA_DIR",
    "authentication_enabled": $([[ "$BASIC_AUTH" == "yes" ]] && echo "true" || echo "false"),
    "created_by": "$(whoami)",
    "hostname": "$(hostname)"
}
EOF
    
    # Store using backup framework
    local backup_path
    if backup_path=$(backup::store "node-red" "$temp_dir" "$label"); then
        log::success "Node-RED backup created: $(basename "$backup_path")"
        rm -rf "$temp_dir"
        echo "$backup_path"
        return 0
    else
        log::error "Failed to create Node-RED backup"
        rm -rf "$temp_dir"
        return 1
    fi
}

#######################################
# List available Node-RED backups
# Returns: 0 on success, 1 on failure
#######################################
node_red::list_backups() {
    backup::list "node-red"
}

#######################################
# Restore Node-RED from backup
# Arguments:
#   $1 - backup path or name
# Returns: 0 on success, 1 on failure
#######################################
node_red::restore_backup() {
    local backup_identifier="${1:-}"
    
    if [[ -z "$backup_identifier" ]]; then
        log::error "Backup identifier required"
        log::info "Usage: node_red::restore_backup <backup_path_or_name>"
        return 1
    fi
    
    log::info "Restoring Node-RED from backup: $backup_identifier"
    
    # Stop Node-RED if running
    if docker::container_running "$NODE_RED_CONTAINER_NAME"; then
        log::info "Stopping Node-RED for restore..."
        node_red::stop
    fi
    
    # Extract backup
    local temp_dir
    if ! temp_dir=$(backup::extract "node-red" "$backup_identifier"); then
        log::error "Failed to extract backup"
        return 1
    fi
    
    # Validate backup structure
    if ! node_red::validate_backup_structure "$temp_dir"; then
        log::error "Invalid backup structure"
        rm -rf "$temp_dir"
        return 1
    fi
    
    # Create backup of current data before restore
    if [[ -d "$NODE_RED_DATA_DIR" ]]; then
        local current_backup
        current_backup=$(node_red::create_backup "pre-restore-$(date +%Y%m%d-%H%M%S)")
        log::info "Current data backed up to: $(basename "$current_backup")"
    fi
    
    # Restore data directory
    if [[ -d "$temp_dir/data" ]]; then
        log::info "Restoring data directory..."
        mkdir -p "$NODE_RED_DATA_DIR"
        rm -rf "${NODE_RED_DATA_DIR:?}"/*
        cp -r "$temp_dir/data"/* "$NODE_RED_DATA_DIR/" 2>/dev/null || true
        
        # Set correct permissions
        chown -R 1000:1000 "$NODE_RED_DATA_DIR" 2>/dev/null || true
    fi
    
    # Restore configuration files
    if [[ -d "$temp_dir/config" ]]; then
        log::info "Restoring configuration files..."
        for config_file in "$temp_dir/config"/*; do
            if [[ -f "$config_file" ]]; then
                local filename
                filename=$(basename "$config_file")
                cp "$config_file" "${NODE_RED_SCRIPT_DIR}/config/$filename"
            fi
        done
    fi
    
    # Clean up
    rm -rf "$temp_dir"
    
    log::success "Node-RED restore completed successfully"
    log::info "You may now start Node-RED: ./manage.sh --action start"
    
    return 0
}

#######################################
# Validate backup structure
# Arguments:
#   $1 - backup directory path
# Returns: 0 if valid, 1 if invalid
#######################################
node_red::validate_backup_structure() {
    local backup_dir="${1:-}"
    
    if [[ ! -d "$backup_dir" ]]; then
        log::error "Backup directory does not exist: $backup_dir"
        return 1
    fi
    
    # Check for metadata file
    if [[ ! -f "$backup_dir/metadata.json" ]]; then
        log::error "Backup metadata file missing"
        return 1
    fi
    
    # Validate metadata
    if ! jq -e '.service == "node-red"' "$backup_dir/metadata.json" >/dev/null 2>&1; then
        log::error "Invalid backup metadata (not a node-red backup)"
        return 1
    fi
    
    # Check for expected directories
    local expected_dirs=("data" "config")
    for dir in "${expected_dirs[@]}"; do
        if [[ ! -d "$backup_dir/$dir" ]]; then
            log::warn "Missing backup directory: $dir"
        fi
    done
    
    return 0
}

#######################################
# Get backup information
# Arguments:
#   $1 - backup path or name
# Returns: JSON backup information
#######################################
node_red::get_backup_info() {
    local backup_identifier="${1:-}"
    
    if [[ -z "$backup_identifier" ]]; then
        echo '{"error": "backup identifier required"}'
        return 1
    fi
    
    local temp_dir
    if temp_dir=$(backup::extract "node-red" "$backup_identifier"); then
        if [[ -f "$temp_dir/metadata.json" ]]; then
            cat "$temp_dir/metadata.json"
        else
            echo '{"error": "metadata not found"}'
        fi
        rm -rf "$temp_dir"
    else
        echo '{"error": "failed to extract backup"}'
        return 1
    fi
}

#######################################
# Delete a Node-RED backup
# Arguments:
#   $1 - backup path or name
# Returns: 0 on success, 1 on failure
#######################################
node_red::delete_backup() {
    local backup_identifier="${1:-}"
    
    if [[ -z "$backup_identifier" ]]; then
        log::error "Backup identifier required"
        return 1
    fi
    
    backup::delete "node-red" "$backup_identifier"
}

# Internal backup helpers for flow files
node_red::backup_flows_file() {
    local dest_dir="${1:-}"
    
    if [[ ! -f "$NODE_RED_DATA_DIR/$NODE_RED_FLOWS_FILE" ]]; then
        return 0  # No flows to backup, not an error
    fi
    
    if [[ -n "$dest_dir" ]]; then
        cp "$NODE_RED_DATA_DIR/$NODE_RED_FLOWS_FILE" "$dest_dir/"
    fi
}

node_red::restore_flows_file() {
    local src_file="${1:-}"
    
    if [[ -f "$src_file" ]] && jq empty "$src_file" 2>/dev/null; then
        mkdir -p "$NODE_RED_DATA_DIR"
        cp "$src_file" "$NODE_RED_DATA_DIR/$NODE_RED_FLOWS_FILE"
        chown 1000:1000 "$NODE_RED_DATA_DIR/$NODE_RED_FLOWS_FILE" 2>/dev/null || true
    fi
}

#######################################
# Get Node-RED version
# Returns: version string
#######################################
node_red::get_version() {
    if docker::container_running "$NODE_RED_CONTAINER_NAME"; then
        docker exec "$NODE_RED_CONTAINER_NAME" node -e "console.log(require('/usr/src/node-red/package.json').version)" 2>/dev/null || echo "unknown"
    else
        echo "unknown"
    fi
}

#######################################
# Automated backup creation (cron-friendly)
# Creates daily backups and cleans up old ones
#######################################
node_red::automated_backup() {
    local retention_days="${1:-7}"
    local label="auto-$(date +%Y%m%d)"
    
    log::info "Creating automated Node-RED backup (retention: $retention_days days)"
    
    # Create backup
    if node_red::create_backup "$label"; then
        log::info "Automated backup created successfully"
        
        # Clean up old backups
        backup::cleanup "node-red" "$retention_days" "auto-*"
        
        return 0
    else
        log::error "Automated backup failed"
        return 1
    fi
}