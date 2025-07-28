#!/usr/bin/env bash
# Vault Docker Management Functions
# Container lifecycle and Docker-specific operations

#######################################
# Create Docker network for Vault
#######################################
vault::docker::create_network() {
    if ! docker network inspect "$VAULT_NETWORK_NAME" >/dev/null 2>&1; then
        log::info "Creating Vault network: $VAULT_NETWORK_NAME"
        docker network create "$VAULT_NETWORK_NAME" >/dev/null 2>&1
    fi
}

#######################################
# Remove Docker network for Vault
#######################################
vault::docker::remove_network() {
    if docker network inspect "$VAULT_NETWORK_NAME" >/dev/null 2>&1; then
        log::info "Removing Vault network: $VAULT_NETWORK_NAME"
        docker network rm "$VAULT_NETWORK_NAME" >/dev/null 2>&1 || true
    fi
}

#######################################
# Pull Vault Docker image
#######################################
vault::docker::pull_image() {
    log::info "Pulling Vault image: $VAULT_IMAGE"
    docker pull "$VAULT_IMAGE"
}

#######################################
# Start Vault container
#######################################
vault::docker::start_container() {
    if vault::is_running; then
        vault::message "info" "MSG_VAULT_ALREADY_RUNNING"
        return 0
    fi
    
    vault::message "info" "MSG_VAULT_START_STARTING"
    
    # Ensure network and directories exist
    vault::docker::create_network
    vault::ensure_directories
    vault::create_config "$VAULT_MODE"
    
    # Build Docker command based on mode
    local docker_args=(
        --detach
        --name "$VAULT_CONTAINER_NAME"
        --network "$VAULT_NETWORK_NAME"
        --publish "${VAULT_PORT}:${VAULT_PORT}"
        --volume "${VAULT_DATA_DIR}:/vault/data:Z"
        --volume "${VAULT_CONFIG_DIR}:/vault/config:Z"
        --volume "${VAULT_LOGS_DIR}:/vault/logs:Z"
        --cap-add=IPC_LOCK
        --restart unless-stopped
    )
    
    # Add environment variables
    local env_vars=(
        "VAULT_ADDR=http://0.0.0.0:${VAULT_PORT}"
        "VAULT_API_ADDR=${VAULT_BASE_URL}"
    )
    
    if [[ "$VAULT_MODE" == "dev" ]]; then
        env_vars+=(
            "VAULT_DEV_ROOT_TOKEN_ID=${VAULT_DEV_ROOT_TOKEN_ID}"
            "VAULT_DEV_LISTEN_ADDRESS=${VAULT_DEV_LISTEN_ADDRESS}"
        )
        vault::message "warn" "MSG_VAULT_DEV_MODE_WARNING"
    fi
    
    # Add environment variables to Docker args
    for env_var in "${env_vars[@]}"; do
        docker_args+=(--env "$env_var")
    done
    
    # Start container
    local container_id
    if [[ "$VAULT_MODE" == "dev" ]]; then
        # Development mode - auto-initialized and unsealed
        container_id=$(docker run "${docker_args[@]}" "$VAULT_IMAGE" server -dev -config=/vault/config)
    else
        # Production mode - requires manual initialization
        container_id=$(docker run "${docker_args[@]}" "$VAULT_IMAGE" server -config=/vault/config)
    fi
    
    if [[ -n "$container_id" ]]; then
        log::info "Started Vault container: $container_id"
        
        # Wait for container to be healthy
        if vault::wait_for_health; then
            vault::message "success" "MSG_VAULT_START_SUCCESS"
            return 0
        else
            vault::message "error" "MSG_VAULT_START_FAILED"
            vault::docker::show_logs
            return 1
        fi
    else
        vault::message "error" "MSG_VAULT_START_FAILED"
        return 1
    fi
}

#######################################
# Stop Vault container
#######################################
vault::docker::stop_container() {
    if ! vault::is_running; then
        vault::message "info" "MSG_VAULT_NOT_RUNNING"
        return 0
    fi
    
    vault::message "info" "MSG_VAULT_STOP_STOPPING"
    
    if docker stop "$VAULT_CONTAINER_NAME" >/dev/null 2>&1; then
        vault::message "success" "MSG_VAULT_STOP_SUCCESS"
        return 0
    else
        vault::message "error" "MSG_VAULT_STOP_FAILED"
        return 1
    fi
}

#######################################
# Restart Vault container
#######################################
vault::docker::restart_container() {
    vault::message "info" "MSG_VAULT_RESTART_RESTARTING"
    
    vault::docker::stop_container
    sleep 2
    vault::docker::start_container
    
    if [[ $? -eq 0 ]]; then
        vault::message "success" "MSG_VAULT_RESTART_SUCCESS"
        return 0
    else
        vault::message "error" "MSG_VAULT_RESTART_FAILED"
        return 1
    fi
}

#######################################
# Remove Vault container
#######################################
vault::docker::remove_container() {
    if vault::is_running; then
        vault::docker::stop_container
    fi
    
    if vault::is_installed; then
        log::info "Removing Vault container: $VAULT_CONTAINER_NAME"
        docker rm "$VAULT_CONTAINER_NAME" >/dev/null 2>&1 || true
    fi
}

#######################################
# Show Vault container logs
# Arguments:
#   $1 - number of lines (optional, default 50)
#   $2 - follow flag (optional, 'follow' to tail)
#######################################
vault::docker::show_logs() {
    local lines="${1:-50}"
    local follow_flag="${2:-}"
    
    if ! vault::is_installed; then
        log::error "Vault container not found"
        return 1
    fi
    
    local docker_logs_args=(
        --timestamps
        --tail "$lines"
    )
    
    if [[ "$follow_flag" == "follow" ]]; then
        docker_logs_args+=(--follow)
    fi
    
    log::info "Showing Vault logs (last $lines lines):"
    docker logs "${docker_logs_args[@]}" "$VAULT_CONTAINER_NAME"
}

#######################################
# Execute command inside Vault container
# Arguments:
#   $@ - command to execute
#######################################
vault::docker::exec() {
    if ! vault::is_running; then
        log::error "Vault container is not running"
        return 1
    fi
    
    docker exec -it "$VAULT_CONTAINER_NAME" "$@"
}

#######################################
# Get Vault container resource usage
#######################################
vault::docker::get_resource_usage() {
    if ! vault::is_running; then
        log::error "Vault container is not running"
        return 1
    fi
    
    log::info "Vault container resource usage:"
    docker stats --no-stream --format "table {{.Container}}\t{{.CPUPerc}}\t{{.MemUsage}}\t{{.NetIO}}\t{{.BlockIO}}" "$VAULT_CONTAINER_NAME"
}

#######################################
# Clean up all Vault Docker resources
#######################################
vault::docker::cleanup() {
    log::info "Cleaning up Vault Docker resources..."
    
    # Remove container
    vault::docker::remove_container
    
    # Remove network (only if no other containers are using it)
    if docker network inspect "$VAULT_NETWORK_NAME" >/dev/null 2>&1; then
        local network_containers
        network_containers=$(docker network inspect "$VAULT_NETWORK_NAME" --format '{{len .Containers}}' 2>/dev/null || echo "0")
        
        if [[ "$network_containers" -eq 0 ]]; then
            vault::docker::remove_network
        else
            log::info "Network $VAULT_NETWORK_NAME has other containers, keeping it"
        fi
    fi
}

#######################################
# Check Docker prerequisites
#######################################
vault::docker::check_prerequisites() {
    # Check if Docker is installed
    if ! command -v docker >/dev/null 2>&1; then
        log::error "Docker is not installed or not in PATH"
        return 1
    fi
    
    # Check if Docker daemon is running
    if ! docker info >/dev/null 2>&1; then
        log::error "Docker daemon is not running or not accessible"
        log::info "Try: sudo systemctl start docker"
        return 1
    fi
    
    # Check if user can run Docker commands
    if ! docker ps >/dev/null 2>&1; then
        log::error "Cannot execute Docker commands - check permissions"
        log::info "Try: sudo usermod -aG docker \$USER && newgrp docker"
        return 1
    fi
    
    return 0
}

#######################################
# Backup Vault container data
# Arguments:
#   $1 - backup file path (optional)
#######################################
vault::docker::backup() {
    local backup_file="${1:-${HOME}/vault-backup-$(date +%Y%m%d-%H%M%S).tar.gz}"
    
    if ! vault::is_installed; then
        log::error "Vault container not found"
        return 1
    fi
    
    vault::message "info" "MSG_VAULT_BACKUP_START"
    
    # Create backup of data directory
    if tar -czf "$backup_file" -C "$(dirname "$VAULT_DATA_DIR")" "$(basename "$VAULT_DATA_DIR")" 2>/dev/null; then
        vault::message "success" "MSG_VAULT_BACKUP_SUCCESS"
        log::info "Backup saved to: $backup_file"
        return 0
    else
        vault::message "error" "MSG_VAULT_BACKUP_FAILED"
        return 1
    fi
}

#######################################
# Restore Vault container data
# Arguments:
#   $1 - backup file path
#######################################
vault::docker::restore() {
    local backup_file="$1"
    
    if [[ -z "$backup_file" ]]; then
        log::error "Backup file path required"
        return 1
    fi
    
    if [[ ! -f "$backup_file" ]]; then
        log::error "Backup file not found: $backup_file"
        return 1
    fi
    
    vault::message "info" "MSG_VAULT_RESTORE_START"
    
    # Stop Vault if running
    if vault::is_running; then
        vault::docker::stop_container
    fi
    
    # Restore data directory
    if tar -xzf "$backup_file" -C "$(dirname "$VAULT_DATA_DIR")" 2>/dev/null; then
        vault::message "success" "MSG_VAULT_RESTORE_SUCCESS"
        log::info "Restored from: $backup_file"
        return 0
    else
        vault::message "error" "MSG_VAULT_RESTORE_FAILED"
        return 1
    fi
}