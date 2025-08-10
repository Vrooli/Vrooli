#!/usr/bin/env bash
# Vault Docker Management Functions
# Container lifecycle and Docker-specific operations

# Source required utilities if not already loaded
if ! command -v trash::safe_remove >/dev/null 2>&1; then
    SCRIPT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
    # shellcheck disable=SC1091
    source "${SCRIPT_DIR}/../../../lib/utils/var.sh" 2>/dev/null || true
    # shellcheck disable=SC1091
    source "${var_LIB_SYSTEM_DIR}/trash.sh" 2>/dev/null || true
fi

#######################################
# Determine best storage strategy
# Returns: volumes, bind, or inmem
#######################################
vault::docker::determine_storage_strategy() {
    if [[ -n "${VAULT_STORAGE_STRATEGY:-}" ]] && [[ "${VAULT_STORAGE_STRATEGY}" != "auto" ]]; then
        echo "${VAULT_STORAGE_STRATEGY}"
        return 0
    fi
    
    # Auto-detect best strategy
    if [[ -f /.dockerenv ]]; then
        # Running inside container/CI
        echo "inmem"
    elif command -v docker &>/dev/null && docker volume ls &>/dev/null 2>&1; then
        # Docker available with volume support
        echo "volumes"
    elif [[ -w "${HOME}" ]]; then
        # Can write to home directory
        echo "bind"
    else
        # Restricted environment
        echo "inmem"
    fi
}

#######################################
# Create Docker volumes for Vault
# Returns: 0 on success, 1 on failure
#######################################
vault::docker::create_volumes() {
    local volumes=("${VAULT_VOLUME_DATA}" "${VAULT_VOLUME_CONFIG}" "${VAULT_VOLUME_LOGS}")
    
    for volume in "${volumes[@]}"; do
        if ! docker volume inspect "${volume}" >/dev/null 2>&1; then
            log::info "Creating Docker volume: ${volume}"
            if ! docker volume create "${volume}" >/dev/null 2>&1; then
                log::error "Failed to create volume: ${volume}"
                return 1
            fi
        else
            log::info "Volume already exists: ${volume}"
        fi
    done
    
    return 0
}

#######################################
# Remove Docker volumes for Vault
# Arguments:
#   $1 - force removal (yes/no)
#######################################
vault::docker::remove_volumes() {
    local force="${1:-no}"
    local volumes=("${VAULT_VOLUME_DATA}" "${VAULT_VOLUME_CONFIG}" "${VAULT_VOLUME_LOGS}")
    
    for volume in "${volumes[@]}"; do
        if docker volume inspect "${volume}" >/dev/null 2>&1; then
            if [[ "${force}" == "yes" ]]; then
                log::info "Removing Docker volume: ${volume}"
                docker volume rm "${volume}" >/dev/null 2>&1 || true
            else
                log::warn "Volume exists but not removing (use --remove-data yes): ${volume}"
            fi
        fi
    done
}

#######################################
# Copy configuration to Docker volume
# Returns: 0 on success, 1 on failure
#######################################
vault::docker::copy_config_to_volume() {
    local temp_config="/tmp/vault-config-$$"
    
    # Create temporary config directory
    mkdir -p "${temp_config}"
    
    # Generate configuration
    vault::create_config "${VAULT_MODE}"
    
    # Copy local config to temp
    if [[ -f "${VAULT_CONFIG_DIR}/vault.hcl" ]]; then
        cp "${VAULT_CONFIG_DIR}/vault.hcl" "${temp_config}/vault.hcl"
    fi
    
    # Copy TLS certificates if they exist
    if [[ -d "${VAULT_CONFIG_DIR}/tls" ]]; then
        cp -r "${VAULT_CONFIG_DIR}/tls" "${temp_config}/tls"
    fi
    
    # Use Alpine container to copy files to volume
    log::info "Copying configuration to Docker volume..."
    docker run --rm \
        -v "${VAULT_VOLUME_CONFIG}:/target" \
        -v "${temp_config}:/source:ro" \
        alpine sh -c "cp -r /source/* /target/ && chmod -R 755 /target"
    
    local result=$?
    
    # Clean up temp directory
    if command -v trash::safe_remove >/dev/null 2>&1; then
        trash::safe_remove "${temp_config}" --no-confirm
    else
        rm -rf "${temp_config}"  # fallback if trash system not available
    fi
    
    return ${result}
}

#######################################
# Prepare bind mount directories with correct permissions
# Returns: 0 on success, 1 on failure
#######################################
vault::docker::prepare_bind_directories() {
    local dirs=("${VAULT_DATA_DIR}" "${VAULT_CONFIG_DIR}" "${VAULT_LOGS_DIR}")
    
    for dir in "${dirs[@]}"; do
        if [[ ! -d "${dir}" ]]; then
            if ! mkdir -p "${dir}" 2>/dev/null; then
                log::error "Cannot create directory: ${dir}"
                log::info "Try: sudo mkdir -p ${dir} && sudo chown \$USER:\$USER ${dir}"
                return 1
            fi
        fi
        
        # Check if directory is writable
        if [[ ! -w "${dir}" ]]; then
            log::error "Directory not writable: ${dir}"
            log::info "Try: sudo chown \$USER:\$USER ${dir}"
            return 1
        fi
        
        # Set permissions
        chmod 755 "${dir}" 2>/dev/null || true
    done
    
    # Special handling for TLS files
    if [[ -d "${VAULT_CONFIG_DIR}/tls" ]]; then
        chmod 755 "${VAULT_CONFIG_DIR}/tls" 2>/dev/null || true
        chmod 644 "${VAULT_CONFIG_DIR}/tls"/*.crt 2>/dev/null || true
        chmod 640 "${VAULT_CONFIG_DIR}/tls"/*.key 2>/dev/null || true
    fi
    
    return 0
}

#######################################
# Preflight check for container startup
# Returns: 0 if ready, 1 if issues found
#######################################
vault::docker::preflight_check() {
    local issues=()
    
    # Check Docker access
    if ! docker ps >/dev/null 2>&1; then
        issues+=("Docker daemon not accessible")
    fi
    
    # Check port availability
    if lsof -i ":${VAULT_PORT}" >/dev/null 2>&1; then
        issues+=("Port ${VAULT_PORT} is already in use")
    fi
    
    # Check storage strategy requirements
    local strategy=$(vault::docker::determine_storage_strategy)
    
    case "${strategy}" in
        volumes)
            # Check if we can create volumes
            if ! docker volume ls >/dev/null 2>&1; then
                issues+=("Cannot access Docker volumes")
            fi
            ;;
        bind)
            # Check directory permissions
            if ! vault::docker::prepare_bind_directories; then
                issues+=("Cannot prepare bind mount directories")
            fi
            ;;
    esac
    
    if [[ ${#issues[@]} -gt 0 ]]; then
        log::error "Preflight check failed:"
        for issue in "${issues[@]}"; do
            log::error "  - ${issue}"
        done
        return 1
    fi
    
    log::info "Preflight check passed"
    return 0
}

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
    
    # Run preflight check
    if ! vault::docker::preflight_check; then
        log::error "Preflight check failed. Cannot start Vault."
        return 1
    fi
    
    vault::message "info" "MSG_VAULT_START_STARTING"
    
    # Determine storage strategy
    local strategy=$(vault::docker::determine_storage_strategy)
    log::info "Using storage strategy: ${strategy}"
    
    # Ensure network exists
    vault::docker::create_network
    
    # Prepare storage based on strategy
    case "${strategy}" in
        volumes)
            # Create Docker volumes
            if ! vault::docker::create_volumes; then
                log::error "Failed to create Docker volumes"
                return 1
            fi
            # Copy configuration to volume
            if ! vault::docker::copy_config_to_volume; then
                log::error "Failed to copy configuration to volume"
                return 1
            fi
            ;;
        bind)
            # Prepare bind mount directories
            if ! vault::docker::prepare_bind_directories; then
                log::error "Failed to prepare bind mount directories"
                return 1
            fi
            # Ensure directories and config exist
            vault::ensure_directories
            vault::create_config "$VAULT_MODE"
            ;;
        inmem)
            # In-memory storage, minimal setup needed
            vault::ensure_directories
            vault::create_config "$VAULT_MODE"
            ;;
    esac
    
    # Build Docker command
    local docker_args=(
        --detach
        --name "$VAULT_CONTAINER_NAME"
        --network "$VAULT_NETWORK_NAME"
        --publish "${VAULT_PORT}:${VAULT_PORT}"
        --cap-add=IPC_LOCK
        --restart unless-stopped
        --health-cmd "vault status -tls-skip-verify || exit 1"
        --health-interval 10s
        --health-timeout 5s
        --health-retries 3
        --health-start-period 10s
    )
    
    # Add storage volumes based on strategy
    case "${strategy}" in
        volumes)
            docker_args+=(
                --volume "${VAULT_VOLUME_DATA}:/vault/data"
                --volume "${VAULT_VOLUME_CONFIG}:/vault/config"
                --volume "${VAULT_VOLUME_LOGS}:/vault/logs"
            )
            ;;
        bind)
            docker_args+=(
                --volume "${VAULT_DATA_DIR}:/vault/data"
                --volume "${VAULT_CONFIG_DIR}:/vault/config"
                --volume "${VAULT_LOGS_DIR}:/vault/logs"
            )
            ;;
        inmem)
            # No volumes for in-memory storage
            # Only mount config if in production mode
            if [[ "$VAULT_MODE" == "prod" ]]; then
                docker_args+=(--volume "${VAULT_CONFIG_DIR}:/vault/config:ro")
            fi
            ;;
    esac
    
    # Add environment variables
    local env_vars=()
    
    if [[ "$VAULT_MODE" == "dev" ]]; then
        env_vars+=(
            "VAULT_ADDR=http://0.0.0.0:${VAULT_PORT}"
            "VAULT_API_ADDR=http://localhost:${VAULT_PORT}"
            "VAULT_DEV_ROOT_TOKEN_ID=${VAULT_DEV_ROOT_TOKEN_ID}"
            "VAULT_DEV_LISTEN_ADDRESS=${VAULT_DEV_LISTEN_ADDRESS}"
        )
        vault::message "warn" "MSG_VAULT_DEV_MODE_WARNING"
    else
        # Production mode
        if [[ -f "${VAULT_CONFIG_DIR}/tls/server.crt" ]]; then
            # HTTPS if TLS certificates exist
            env_vars+=(
                "VAULT_ADDR=https://0.0.0.0:${VAULT_PORT}"
                "VAULT_API_ADDR=https://localhost:${VAULT_PORT}"
                "VAULT_SKIP_VERIFY=true"  # For self-signed certificates
            )
        else
            # HTTP fallback
            env_vars+=(
                "VAULT_ADDR=http://0.0.0.0:${VAULT_PORT}"
                "VAULT_API_ADDR=http://localhost:${VAULT_PORT}"
            )
        fi
    fi
    
    # Add environment variables to Docker args
    for env_var in "${env_vars[@]}"; do
        docker_args+=(--env "$env_var")
    done
    
    # Start container
    local container_id
    if [[ "$VAULT_MODE" == "dev" ]]; then
        # Development mode - auto-initialized and unsealed
        container_id=$(docker run "${docker_args[@]}" "$VAULT_IMAGE" server -dev)
    else
        # Production mode - requires manual initialization
        if [[ "${strategy}" == "inmem" ]]; then
            # Use inline config for in-memory storage
            container_id=$(docker run "${docker_args[@]}" \
                -e 'VAULT_LOCAL_CONFIG={"storage": {"inmem": {}}, "listener": {"tcp": {"address": "0.0.0.0:8200", "tls_disable": "1"}}, "ui": true}' \
                "$VAULT_IMAGE" server)
        else
            container_id=$(docker run "${docker_args[@]}" "$VAULT_IMAGE" server -config=/vault/config)
        fi
    fi
    
    if [[ -n "$container_id" ]]; then
        log::info "Started Vault container: $container_id"
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
    
    # Check storage strategy for volume cleanup
    local strategy=$(vault::docker::determine_storage_strategy)
    
    if [[ "${strategy}" == "volumes" ]]; then
        # Optionally remove volumes based on VAULT_REMOVE_DATA
        if [[ "${VAULT_REMOVE_DATA:-no}" == "yes" ]]; then
            vault::docker::remove_volumes "yes"
        else
            log::info "Docker volumes preserved (use --remove-data yes to remove)"
        fi
    fi
    
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
# Repair permissions for Vault directories
# Returns: 0 on success, 1 on failure
#######################################
vault::docker::repair_permissions() {
    log::info "Attempting to repair Vault directory permissions..."
    
    local dirs=("${VAULT_DATA_DIR}" "${VAULT_CONFIG_DIR}" "${VAULT_LOGS_DIR}")
    local repaired=0
    local failed=0
    
    for dir in "${dirs[@]}"; do
        if [[ -d "${dir}" ]]; then
            # Try to fix ownership without sudo
            if chown -R "$(id -u):$(id -g)" "${dir}" 2>/dev/null; then
                log::info "Fixed ownership for: ${dir}"
                repaired=$((repaired + 1))
            else
                log::warn "Cannot fix ownership for ${dir} without elevated permissions"
                failed=$((failed + 1))
            fi
            
            # Try to fix permissions
            if chmod -R 755 "${dir}" 2>/dev/null; then
                log::info "Fixed permissions for: ${dir}"
            else
                log::warn "Cannot fix permissions for ${dir}"
            fi
        fi
    done
    
    if [[ ${failed} -gt 0 ]]; then
        log::error "Some directories need elevated permissions to repair"
        log::info "Run the following command with sudo:"
        log::info "  sudo chown -R \$USER:\$USER ${VAULT_DATA_DIR%/*}"
        log::info "  sudo chmod -R 755 ${VAULT_DATA_DIR%/*}"
        return 1
    fi
    
    if [[ ${repaired} -gt 0 ]]; then
        log::info "Successfully repaired ${repaired} directories"
    fi
    
    return 0
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