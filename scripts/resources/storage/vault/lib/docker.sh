#!/usr/bin/env bash
# Vault Docker Management - Ultra-Simplified
# Uses docker-resource-utils.sh for minimal boilerplate

# Source var.sh to get proper directory variables
_VAULT_DOCKER_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck disable=SC1091
source "${_VAULT_DOCKER_DIR}/../../../../lib/utils/var.sh"

# Source shared libraries
# shellcheck disable=SC1091
source "${var_LIB_SYSTEM_DIR}/trash.sh" 2>/dev/null || true
# shellcheck disable=SC1091
source "${var_LIB_UTILS_DIR}/sudo.sh" 2>/dev/null || true
# shellcheck disable=SC1091
source "${var_SCRIPTS_RESOURCES_LIB_DIR}/docker-resource-utils.sh"
# shellcheck disable=SC1091
source "${var_SCRIPTS_RESOURCES_LIB_DIR}/backup-framework.sh"
# shellcheck disable=SC1091
source "${_VAULT_DOCKER_DIR}/vault-storage-strategies.sh"






# Preflight check for container startup
vault::docker::preflight_check() {
    # Check Docker access using simplified utility
    if ! docker::check_daemon; then
        log::error "Docker daemon not accessible"
        return 1
    fi
    
    # Check port availability
    if ! docker::is_port_available "${VAULT_PORT}"; then
        log::error "Port ${VAULT_PORT} is already in use"
        return 1
    fi
    
    log::info "Preflight check passed"
    return 0
}



# Start Vault container
vault::docker::start_container() {
    if vault::is_running; then
        vault::message "info" "MSG_VAULT_ALREADY_RUNNING"
        return 0
    fi
    
    # Run preflight check
    if ! vault::docker::preflight_check; then
        log::error "Preflight check failed. Cannot start Vault."
        return 1
    fi
    
    # Pull image if needed
    log::info "Pulling Vault image: $VAULT_IMAGE"
    docker::pull_image "$VAULT_IMAGE"
    
    vault::message "info" "MSG_VAULT_START_STARTING"
    
    # Determine storage strategy
    local strategy=$(vault::storage::determine_strategy)
    log::info "Using storage strategy: ${strategy}"
    
    # Prepare storage
    if ! vault::storage::prepare "${strategy}"; then
        log::error "Failed to prepare storage"
        return 1
    fi
    
    # Get volumes based on strategy
    local volumes=$(vault::storage::get_volumes "${strategy}")
    
    # Prepare health check command
    local health_cmd="vault status -tls-skip-verify || exit 1"
    
    # Prepare command arguments based on mode
    local entrypoint_cmd=()
    if [[ "$VAULT_MODE" == "dev" ]]; then
        # Development mode - auto-initialized and unsealed
        entrypoint_cmd=("server" "-dev")
    else
        # Production mode - requires manual initialization
        if [[ "${strategy}" == "inmem" ]]; then
            # Use inline config for in-memory storage via environment variable
            entrypoint_cmd=("server")
        else
            entrypoint_cmd=("server" "-config=/vault/config")
        fi
    fi
    
    # Create network
    docker::create_network "$VAULT_NETWORK_NAME"
    
    # Prepare environment variables
    local env_vars=()
    if [[ "$VAULT_MODE" == "dev" ]]; then
        # Dev mode environment
        env_vars+=(
            "VAULT_ADDR=http://0.0.0.0:${VAULT_PORT}"
            "VAULT_API_ADDR=http://localhost:${VAULT_PORT}"
            "VAULT_DEV_ROOT_TOKEN_ID=${VAULT_DEV_ROOT_TOKEN_ID}"
            "VAULT_DEV_LISTEN_ADDRESS=${VAULT_DEV_LISTEN_ADDRESS}"
        )
    else
        # Production mode environment
        if [[ -f "${VAULT_CONFIG_DIR}/tls/server.crt" ]]; then
            # HTTPS if TLS certificates exist
            env_vars+=(
                "VAULT_ADDR=https://0.0.0.0:${VAULT_PORT}"
                "VAULT_API_ADDR=https://localhost:${VAULT_PORT}"
                "VAULT_SKIP_VERIFY=true"
            )
        else
            # HTTP fallback
            env_vars+=(
                "VAULT_ADDR=http://0.0.0.0:${VAULT_PORT}"
                "VAULT_API_ADDR=http://localhost:${VAULT_PORT}"
            )
        fi
        
        # Add inline config for in-memory storage
        if [[ "${strategy}" == "inmem" ]]; then
            env_vars+=("VAULT_LOCAL_CONFIG={\"storage\": {\"inmem\": {}}, \"listener\": {\"tcp\": {\"address\": \"0.0.0.0:8200\", \"tls_disable\": \"1\"}}, \"ui\": true}")
        fi
    fi
    
    # Prepare Docker options
    local docker_opts=(
        "--cap-add=IPC_LOCK"
        "--restart" "unless-stopped"
        "--health-interval" "10s"
        "--health-timeout" "5s"
        "--health-retries" "3"
        "--health-start-period" "10s"
    )
    
    # Port mapping
    local port_mappings="${VAULT_PORT}:${VAULT_PORT}"
    
    # Create container using advanced function
    if docker_resource::create_service_advanced \
        "$VAULT_CONTAINER_NAME" \
        "$VAULT_IMAGE" \
        "$port_mappings" \
        "$VAULT_NETWORK_NAME" \
        "$volumes" \
        "env_vars" \
        "docker_opts" \
        "$health_cmd" \
        "entrypoint_cmd"; then
        
        if [[ "$VAULT_MODE" == "dev" ]]; then
            vault::message "warn" "MSG_VAULT_DEV_MODE_WARNING"
        fi
    else
        log::error "Failed to create Vault container"
        return 1
    fi
    
    log::info "Storage strategy: ${strategy}"
    
    # Wait for container to be healthy
    if vault::wait_for_health; then
        vault::message "success" "MSG_VAULT_START_SUCCESS"
        
        # Show important information based on strategy
        if [[ "${strategy}" == "inmem" ]]; then
            log::warn "Using in-memory storage - data will be lost on restart!"
        elif [[ "${strategy}" == "volumes" ]]; then
            log::info "Using Docker volumes for persistent storage"
        fi
        
        return 0
    else
        vault::message "error" "MSG_VAULT_START_FAILED"
        vault::docker::show_logs
        return 1
    fi
}

# Stop Vault container
vault::docker::stop_container() {
    if ! vault::is_running; then
        vault::message "info" "MSG_VAULT_NOT_RUNNING"
        return 0
    fi
    
    vault::message "info" "MSG_VAULT_STOP_STOPPING"
    
    if docker::stop_container "$VAULT_CONTAINER_NAME"; then
        vault::message "success" "MSG_VAULT_STOP_SUCCESS"
        return 0
    else
        vault::message "error" "MSG_VAULT_STOP_FAILED"
        return 1
    fi
}

# Restart Vault container
vault::docker::restart_container() {
    vault::message "info" "MSG_VAULT_RESTART_RESTARTING"
    
    if docker::restart_container "$VAULT_CONTAINER_NAME"; then
        vault::message "success" "MSG_VAULT_RESTART_SUCCESS"
        return 0
    else
        vault::message "error" "MSG_VAULT_RESTART_FAILED"
        return 1
    fi
}

# Remove Vault container
vault::docker::remove_container() {
    if vault::is_running; then
        vault::docker::stop_container
    fi
    
    if vault::is_installed; then
        log::info "Removing Vault container: $VAULT_CONTAINER_NAME"
        docker::remove_container "$VAULT_CONTAINER_NAME" "true"
    fi
}



# Get Vault container resource usage
vault::docker::get_resource_usage() {
    log::info "Vault container resource usage:"
    docker_resource::get_stats "$VAULT_CONTAINER_NAME"
}

# Clean up all Vault Docker resources
vault::docker::cleanup() {
    log::info "Cleaning up Vault Docker resources..."
    
    # Remove container
    vault::docker::remove_container
    
    # Clean up storage resources
    local strategy=$(vault::storage::determine_strategy)
    vault::storage::cleanup "${strategy}" "${VAULT_REMOVE_DATA:-no}"
    
    # Remove network only if empty
    docker::cleanup_network_if_empty "$VAULT_NETWORK_NAME"
}

#######################################
# Repair permissions for Vault directories
# Delegates to storage strategies module
# Returns: 0 on success, 1 on failure
#######################################
vault::docker::repair_permissions() {
    vault::storage::repair_permissions
}

# Check Docker prerequisites
vault::docker::check_prerequisites() {
    # Use simplified Docker check
    docker::check_daemon
}

#######################################
# Backup Vault container data
# Arguments:
#   $1 - backup label (optional)
#######################################
vault::docker::backup() {
    local label="${1:-manual}"
    
    if ! vault::is_installed; then
        log::error "Vault container not found"
        return 1
    fi
    
    vault::message "info" "MSG_VAULT_BACKUP_START"
    
    # Use backup framework to store backup
    local backup_path
    if backup_path=$(backup::store "vault" "$VAULT_DATA_DIR" "$label"); then
        vault::message "success" "MSG_VAULT_BACKUP_SUCCESS"
        log::info "Backup saved to: $backup_path"
        return 0
    else
        vault::message "error" "MSG_VAULT_BACKUP_FAILED"
        return 1
    fi
}

#######################################
# Restore Vault container data
# Arguments:
#   $1 - backup identifier (ID or "latest")
#######################################
vault::docker::restore() {
    local backup_id="${1:-latest}"
    
    vault::message "info" "MSG_VAULT_RESTORE_START"
    
    # Stop Vault if running
    if vault::is_running; then
        vault::docker::stop_container
    fi
    
    # Get backup path
    local backup_path
    if [[ "$backup_id" == "latest" ]]; then
        backup_path=$(backup::get_latest "vault")
    else
        backup_path=$(backup::get_by_id "vault" "$backup_id")
    fi
    
    if [[ -z "$backup_path" ]]; then
        log::error "Backup not found: $backup_id"
        return 1
    fi
    
    # Extract backup to temp location
    local temp_dir
    if temp_dir=$(backup::extract "vault" "$backup_path"); then
        # Clear existing data and restore from backup
        rm -rf "${VAULT_DATA_DIR:?}"/*
        cp -r "$temp_dir"/* "$VAULT_DATA_DIR/"
        rm -rf "$temp_dir"
        
        vault::message "success" "MSG_VAULT_RESTORE_SUCCESS"
        log::info "Restored from: $backup_path"
        return 0
    else
        vault::message "error" "MSG_VAULT_RESTORE_FAILED"
        return 1
    fi
}

#######################################
# Create client-specific Vault instance
# Arguments: $1 - client ID
# Returns: 0 on success, 1 on failure
#######################################
vault::docker::create_client_instance() {
    local client_id="$1"
    
    if [[ -z "$client_id" ]]; then
        log::error "Missing client_id parameter"
        return 1
    fi
    
    log::info "Creating Vault client instance for: ${client_id}"
    
    # Use simplified client creation from docker-resource-utils
    local client_port project_config_dir
    project_config_dir="$(secrets::get_project_config_dir)"
    
    # Generate client-specific configuration
    vault::create_config "prod"  # Always use prod mode for clients
    
    client_port=$(docker_resource::create_client_instance \
        "vault" \
        "$client_id" \
        "$VAULT_IMAGE" \
        "8200" \
        "8300" \
        "8399" \
        "$project_config_dir" \
        "${VAULT_CONFIG_DIR}/vault.hcl")
    
    if [[ $? -eq 0 && -n "$client_port" ]]; then
        # Generate metadata using helper
        local client_container="vault-client-${client_id}"
        local client_network="vrooli-${client_id}-network"
        local metadata
        metadata=$(docker_resource::generate_client_metadata \
            "${client_id}" \
            "vault" \
            "${client_container}" \
            "${client_network}" \
            "${client_port}" \
            "http://localhost:${client_port}")
        
        # Save metadata
        local client_config_file="${project_config_dir}/clients/${client_id}/vault.json"
        echo "$metadata" > "$client_config_file"
        
        log::success "Vault client instance created successfully"
        log::info "   Client: ${client_id}"
        log::info "   Port: ${client_port}"
        log::info "   URL: http://localhost:${client_port}"
        
        return 0
    else
        log::error "Failed to create Vault client instance"
        return 1
    fi
}

#######################################
# Destroy client-specific Vault instance
# Arguments: $1 - client ID
# Returns: 0 on success, 1 on failure
#######################################
vault::docker::destroy_client_instance() {
    local client_id="$1"
    
    if [[ -z "$client_id" ]]; then
        log::error "Missing client_id parameter"
        return 1
    fi
    
    log::info "Destroying Vault client instance: ${client_id}"
    
    # Use simplified client removal
    local project_config_dir
    project_config_dir="$(secrets::get_project_config_dir)"
    
    docker_resource::remove_client_instance \
        "vault" \
        "$client_id" \
        "$project_config_dir" \
        "true"
    
    # Remove client configuration metadata safely
    trash::safe_remove "${project_config_dir}/clients/${client_id}/vault.json" --no-confirm 2>/dev/null || true
    
    log::success "Vault client instance destroyed successfully"
    return 0
}