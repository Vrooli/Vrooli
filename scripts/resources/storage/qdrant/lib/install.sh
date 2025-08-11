#!/usr/bin/env bash
# Qdrant Installation Management
# Functions for installing, upgrading, and uninstalling Qdrant

# Source required utilities if not already loaded
if ! command -v trash::safe_remove >/dev/null 2>&1; then
    SCRIPT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
    # shellcheck disable=SC1091
    source "${SCRIPT_DIR}/../../../lib/utils/var.sh" 2>/dev/null || true
    # shellcheck disable=SC1091
    source "${var_LIB_SYSTEM_DIR}/trash.sh" 2>/dev/null || true
fi

#######################################
# Install Qdrant with all dependencies
# Returns: 0 on success, 1 on failure
#######################################
qdrant::install() {
    echo "=== Installing Qdrant Vector Database ==="
    echo
    
    # Check prerequisites
    if ! qdrant::install::check_prerequisites; then
        return 1
    fi
    
    # Check if already installed
    if qdrant::common::container_exists; then
        if qdrant::common::is_running; then
            log::info "Qdrant is already installed and running"
            qdrant::docker::show_connection_info
            return 0
        else
            log::info "Qdrant is installed but not running. Starting..."
            if qdrant::docker::start; then
                return 0
            else
                log::error "Failed to start existing Qdrant installation"
                return 1
            fi
        fi
    fi
    
    # Check port availability
    log::info "${MSG_CHECKING_PORTS}"
    if ! qdrant::common::check_ports; then
        log::error "Required ports are not available"
        log::info "${MSG_HELP_PORT_CONFLICT}"
        return 1
    fi
    
    # Check disk space
    if ! qdrant::common::check_disk_space; then
        log::error "${MSG_INSUFFICIENT_DISK}"
        return 1
    fi
    
    # Pull Docker image
    if ! qdrant::docker::pull_image; then
        log::error "Failed to pull Qdrant Docker image"
        return 1
    fi
    
    # Create and start container
    if ! qdrant::docker::create_container; then
        log::error "${MSG_INSTALL_FAILED}"
        return 1
    fi
    
    # Wait for startup
    log::info "${MSG_WAITING_STARTUP}"
    if ! qdrant::common::wait_for_startup; then
        log::error "Qdrant failed to start properly"
        return 1
    fi
    
    # Initialize default collections
    log::info "${MSG_CREATING_COLLECTIONS}"
    if ! qdrant::install::create_default_collections; then
        log::warn "Some default collections could not be created"
    fi
    
    # Update resource configuration
    if ! qdrant::install::update_resource_config; then
        log::warn "Failed to update resource configuration"
    fi
    
    # Success
    log::success "${MSG_INSTALL_SUCCESS}"
    qdrant::docker::show_connection_info
    
    return 0
}

#######################################
# Check installation prerequisites
# Returns: 0 if all prerequisites met, 1 otherwise
#######################################
qdrant::install::check_prerequisites() {
    log::info "Checking prerequisites..."
    
    # Check Docker
    if ! qdrant::docker::check_docker; then
        return 1
    fi
    
    # Check required commands
    local required_commands=("curl" "jq")
    local missing_commands=()
    
    for cmd in "${required_commands[@]}"; do
        if ! command -v "$cmd" >/dev/null 2>&1; then
            missing_commands+=("$cmd")
        fi
    done
    
    if [[ ${#missing_commands[@]} -gt 0 ]]; then
        log::error "Missing required commands: ${missing_commands[*]}"
        log::info "Please install the missing commands and try again"
        return 1
    fi
    
    log::debug "All prerequisites met"
    return 0
}

#######################################
# Create default collections for Vrooli
# Returns: 0 on success, 1 if any failures
#######################################
qdrant::install::create_default_collections() {
    local success_count=0
    local total_count=${#QDRANT_COLLECTION_CONFIGS[@]}
    
    for config in "${QDRANT_COLLECTION_CONFIGS[@]}"; do
        # Parse config: name:vector_size:distance_metric
        local name
        local vector_size
        local distance_metric
        
        IFS=':' read -r name vector_size distance_metric <<< "$config"
        
        log::info "Creating collection: $name"
        
        if qdrant::collections::create "$name" "$vector_size" "$distance_metric" >/dev/null 2>&1; then
            log::debug "Collection '$name' created successfully"
            success_count=$((success_count + 1))
        else
            log::warn "Failed to create collection '$name'"
        fi
    done
    
    log::info "Created $success_count of $total_count default collections"
    
    if [[ $success_count -eq $total_count ]]; then
        log::success "${MSG_COLLECTIONS_INITIALIZED}"
        return 0
    else
        return 1
    fi
}

#######################################
# Update Vrooli resource configuration
# Returns: 0 on success, 1 on failure
#######################################
qdrant::install::update_resource_config() {
    local config_file="${VROOLI_RESOURCES_CONFIG}"
    
    # Ensure config directory exists
    mkdir -p "$(dirname "$config_file")"
    
    # Create basic config structure if it doesn't exist
    if [[ ! -f "$config_file" ]]; then
        echo '{"services": {}}' > "$config_file"
    fi
    
    # Prepare Qdrant configuration
    local qdrant_config
    read -r -d '' qdrant_config << EOF || true
{
  "enabled": true,
  "baseUrl": "${QDRANT_BASE_URL}",
  "grpcUrl": "${QDRANT_GRPC_URL}",
  "healthCheck": {
    "endpoint": "/",
    "intervalMs": 60000,
    "timeoutMs": 5000
  },
  "collections": {
    "agent_memory": {
      "vector_size": 1536,
      "distance": "Cosine"
    },
    "code_embeddings": {
      "vector_size": 768,
      "distance": "Dot"
    },
    "document_chunks": {
      "vector_size": 1536,
      "distance": "Cosine"
    },
    "conversation_history": {
      "vector_size": 1536,
      "distance": "Cosine"
    }
  },
  "performance": {
    "optimized_segment_size": ${QDRANT_STORAGE_OPTIMIZED_SEGMENT_SIZE},
    "memmap_threshold": ${QDRANT_STORAGE_MEMMAP_THRESHOLD},
    "indexing_threshold": ${QDRANT_STORAGE_INDEXING_THRESHOLD}
  }
}
EOF
    
    # Add API key if configured
    if [[ -n "${QDRANT_API_KEY:-}" ]]; then
        qdrant_config=$(echo "$qdrant_config" | jq --arg key "$QDRANT_API_KEY" '. + {"apiKey": $key}')
    fi
    
    # Update the configuration file
    if jq --argjson qdrant_config "$qdrant_config" \
       '.services.storage.qdrant = $qdrant_config' \
       "$config_file" > "${config_file}.tmp" 2>/dev/null; then
        mv "${config_file}.tmp" "$config_file"
        log::debug "Resource configuration updated"
        return 0
    else
        trash::safe_remove "${config_file}.tmp" --temp
        log::warn "Failed to update resource configuration"
        return 1
    fi
}

#######################################
# Uninstall Qdrant
# Arguments:
#   $1 - Whether to preserve data (true/false)
# Returns: 0 on success, 1 on failure
#######################################
qdrant::uninstall() {
    local preserve_data="${1:-true}"
    
    echo "=== Uninstalling Qdrant ==="
    echo
    
    if [[ "$preserve_data" == "false" ]]; then
        log::warn "This will permanently delete all Qdrant data!"
    fi
    
    # Remove container and optionally data
    if qdrant::docker::remove_container "$preserve_data"; then
        log::success "${MSG_UNINSTALL_SUCCESS}"
        
        if [[ "$preserve_data" == "true" ]]; then
            log::info "Data has been preserved in: ${QDRANT_DATA_DIR}"
            log::info "Run with --remove-data yes to also remove data"
        fi
        
        # Remove from resource configuration
        qdrant::install::remove_from_resource_config
        
        return 0
    else
        log::error "Failed to uninstall Qdrant"
        return 1
    fi
}

#######################################
# Remove Qdrant from resource configuration
# Returns: 0 on success, 1 on failure
#######################################
qdrant::install::remove_from_resource_config() {
    local config_file="${VROOLI_RESOURCES_CONFIG}"
    
    if [[ ! -f "$config_file" ]]; then
        return 0
    fi
    
    if jq 'del(.services.storage.qdrant)' "$config_file" > "${config_file}.tmp" 2>/dev/null; then
        mv "${config_file}.tmp" "$config_file"
        log::debug "Removed from resource configuration"
        return 0
    else
        trash::safe_remove "${config_file}.tmp" --temp
        log::warn "Failed to remove from resource configuration"
        return 1
    fi
}

#######################################
# Upgrade Qdrant to the latest version
# Returns: 0 on success, 1 on failure
#######################################
qdrant::install::upgrade() {
    echo "=== Upgrading Qdrant ==="
    echo
    
    if ! qdrant::common::container_exists; then
        log::error "Qdrant is not installed. Use 'install' action first."
        return 1
    fi
    
    # Check current version
    local current_version
    if qdrant::common::is_running; then
        current_version=$(qdrant::api::get_version)
        log::info "Current version: $current_version"
    else
        log::info "Qdrant is not running, unable to check current version"
    fi
    
    # Create backup before upgrade
    log::info "Creating backup before upgrade..."
    if qdrant::common::is_running; then
        local backup_name="pre-upgrade-$(date +%Y%m%d-%H%M%S)"
        if ! qdrant::snapshots::create "all" "$backup_name" >/dev/null 2>&1; then
            log::warn "Failed to create backup - continuing with upgrade"
        else
            log::info "Backup created: $backup_name"
        fi
    fi
    
    # Stop current container
    log::info "Stopping current Qdrant container..."
    if ! qdrant::docker::stop; then
        log::error "Failed to stop Qdrant container"
        return 1
    fi
    
    # Remove old container (preserve data)
    log::info "Removing old container..."
    if ! qdrant::docker::remove_container true; then
        log::error "Failed to remove old container"
        return 1
    fi
    
    # Pull latest image
    log::info "Pulling latest Qdrant image..."
    if ! qdrant::docker::pull_image; then
        log::error "Failed to pull latest image"
        return 1
    fi
    
    # Create new container with existing data
    log::info "Creating new container with existing data..."
    if ! qdrant::docker::create_container; then
        log::error "Failed to create new container"
        return 1
    fi
    
    # Wait for startup
    if ! qdrant::common::wait_for_startup; then
        log::error "Upgraded Qdrant failed to start"
        return 1
    fi
    
    # Check new version
    local new_version
    new_version=$(qdrant::api::get_version)
    log::success "Upgrade completed successfully"
    log::info "New version: $new_version"
    
    # Show connection info
    qdrant::docker::show_connection_info
    
    return 0
}

#######################################
# Reset Qdrant configuration to defaults
# Returns: 0 on success, 1 on failure
#######################################
qdrant::install::reset_configuration() {
    log::info "Resetting Qdrant configuration to defaults..."
    
    if qdrant::common::is_running; then
        log::info "Stopping Qdrant to reset configuration..."
        qdrant::docker::stop || return 1
    fi
    
    # Remove configuration files but preserve data
    if command -v trash::safe_remove >/dev/null 2>&1; then
        trash::safe_remove "${QDRANT_CONFIG_DIR}" --no-confirm 2>/dev/null || true
    else
        trash::safe_remove "${QDRANT_CONFIG_DIR}" --no-confirm 2>/dev/null || true
    fi
    
    # Recreate directories
    qdrant::docker::create_directories || return 1
    
    # Start with default configuration
    if qdrant::docker::start; then
        log::success "Configuration reset completed"
        return 0
    else
        log::error "Failed to start with reset configuration"
        return 1
    fi
}