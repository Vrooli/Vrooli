#!/usr/bin/env bash
# Redis Docker Management
# Functions for managing Redis Docker container

# Source var.sh to get proper directory variables
_REDIS_DOCKER_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck disable=SC1091
source "${_REDIS_DOCKER_DIR}/../../../../lib/utils/var.sh"

# Source shared secrets management library
# shellcheck disable=SC1091
source "${var_LIB_SERVICE_DIR}/secrets.sh"
# shellcheck disable=SC1091
source "${var_LIB_SYSTEM_DIR}/trash.sh"

#######################################
# Create Docker network if it doesn't exist
# Returns: 0 on success, 1 on failure
#######################################
redis::docker::create_network() {
    if ! docker network ls | grep -q "${REDIS_NETWORK_NAME}"; then
        log::debug "Creating Docker network: ${REDIS_NETWORK_NAME}"
        if docker network create "${REDIS_NETWORK_NAME}" >/dev/null 2>&1; then
            log::debug "Docker network created successfully"
            return 0
        else
            log::error "Failed to create Docker network"
            return 1
        fi
    else
        log::debug "Docker network already exists: ${REDIS_NETWORK_NAME}"
        return 0
    fi
}

#######################################
# Pull Redis Docker image
# Returns: 0 on success, 1 on failure
#######################################
redis::docker::pull_image() {
    log::info "${MSG_PULLING_IMAGE}"
    
    if docker pull "${REDIS_IMAGE}" 2>&1 | while read -r line; do
        if [[ "$line" =~ "Pulling from" ]] || [[ "$line" =~ "Status:" ]]; then
            log::debug "$line"
        fi
    done; then
        log::debug "Redis image pulled successfully"
        return 0
    else
        log::error "Failed to pull Redis image"
        return 1
    fi
}

#######################################
# Create and start Redis container
# Returns: 0 on success, 1 on failure
#######################################
redis::docker::create_container() {
    # Ensure network exists
    redis::docker::create_network || return 1
    
    # Ensure volumes exist
    redis::common::create_volumes || return 1
    
    # Generate configuration
    redis::common::generate_config || return 1
    
    log::info "${MSG_STARTING_CONTAINER}"
    
    # Build docker run command
    local docker_cmd=(
        "docker" "run" "-d"
        "--name" "${REDIS_CONTAINER_NAME}"
        "--network" "${REDIS_NETWORK_NAME}"
        "-p" "${REDIS_PORT}:${REDIS_INTERNAL_PORT}"
        "-v" "${REDIS_VOLUME_NAME}:/data"
        "-v" "${REDIS_TEMP_CONFIG_DIR}:/usr/local/etc/redis:ro"
        "-v" "${REDIS_LOG_VOLUME_NAME}:/var/log/redis"
        "--restart" "unless-stopped"
        "--health-cmd" "redis-cli ping || exit 1"
        "--health-interval" "${REDIS_HEALTH_CHECK_INTERVAL}"
        "--health-timeout" "${REDIS_HEALTH_CHECK_TIMEOUT}"
        "--health-retries" "${REDIS_HEALTH_CHECK_RETRIES}"
        "${REDIS_IMAGE}"
        "redis-server" "/usr/local/etc/redis/redis.conf"
    )
    
    # Add password if configured
    if [[ -n "$REDIS_PASSWORD" ]]; then
        docker_cmd+=("--requirepass" "$REDIS_PASSWORD")
    fi
    
    # Run the container
    if "${docker_cmd[@]}" >/dev/null 2>&1; then
        log::debug "Redis container created successfully"
        
        # Wait a moment for container to initialize
        sleep "${REDIS_INITIALIZATION_WAIT}"
        
        return 0
    else
        log::error "Failed to create Redis container"
        return 1
    fi
}

#######################################
# Start Redis container
# Returns: 0 on success, 1 on failure
#######################################
redis::docker::start() {
    if ! redis::common::container_exists; then
        log::error "Redis container does not exist. Run install first."
        return 1
    fi
    
    if redis::common::is_running; then
        log::info "Redis is already running"
        return 0
    fi
    
    log::info "Starting Redis container..."
    
    if docker start "${REDIS_CONTAINER_NAME}" >/dev/null 2>&1; then
        # Wait for container to be ready
        if redis::common::wait_for_ready; then
            log::success "${MSG_START_SUCCESS}"
            redis::docker::show_connection_info
            return 0
        else
            log::error "${MSG_START_FAILED}"
            return 1
        fi
    else
        log::error "Failed to start Redis container"
        return 1
    fi
}

#######################################
# Stop Redis container
# Returns: 0 on success, 1 on failure
#######################################
redis::docker::stop() {
    if ! redis::common::container_exists; then
        log::warn "Redis container does not exist"
        return 0
    fi
    
    if ! redis::common::is_running; then
        log::info "Redis is not running"
        return 0
    fi
    
    log::info "Stopping Redis container..."
    
    if docker stop "${REDIS_CONTAINER_NAME}" >/dev/null 2>&1; then
        log::success "${MSG_STOP_SUCCESS}"
        return 0
    else
        log::error "${MSG_STOP_FAILED}"
        return 1
    fi
}

#######################################
# Restart Redis container
# Returns: 0 on success, 1 on failure
#######################################
redis::docker::restart() {
    log::info "Restarting Redis container..."
    
    redis::docker::stop
    sleep 2
    redis::docker::start
    
    if [[ $? -eq 0 ]]; then
        log::success "${MSG_RESTART_SUCCESS}"
        return 0
    else
        log::error "Failed to restart Redis"
        return 1
    fi
}

#######################################
# Remove Redis container
# Arguments:
#   $1 - remove data (yes/no)
# Returns: 0 on success, 1 on failure
#######################################
redis::docker::remove_container() {
    local remove_data="${1:-no}"
    
    # Stop container if running
    if redis::common::is_running; then
        redis::docker::stop
    fi
    
    # Remove container
    if redis::common::container_exists; then
        log::info "Removing Redis container..."
        if docker rm "${REDIS_CONTAINER_NAME}" >/dev/null 2>&1; then
            log::debug "Redis container removed successfully"
        else
            log::error "Failed to remove Redis container"
            return 1
        fi
    fi
    
    # Remove data if requested
    if [[ "$remove_data" == "yes" ]]; then
        log::warn "${MSG_WARN_DATA_LOSS}"
        if [[ "$YES" != "yes" ]]; then
            echo -n "Are you sure you want to delete all data? (yes/no): "
            read -r confirm
            if [[ "$confirm" != "yes" ]]; then
                log::info "Data removal cancelled"
                return 0
            fi
        fi
        
        log::info "Removing Redis data..."
        trash::safe_remove "${REDIS_DATA_DIR}" --no-confirm 2>/dev/null || true
        trash::safe_remove "${REDIS_CONFIG_DIR}" --no-confirm 2>/dev/null || true
        trash::safe_remove "${REDIS_LOG_DIR}" --no-confirm 2>/dev/null || true
        log::success "Redis data removed"
    fi
    
    return 0
}

#######################################
# Show Redis container logs
# Arguments:
#   $1 - number of lines (optional)
# Returns: 0 on success, 1 on failure
#######################################
redis::docker::show_logs() {
    local lines="${1:-50}"
    
    if ! redis::common::container_exists; then
        log::error "Redis container does not exist"
        return 1
    fi
    
    docker logs --tail "$lines" -f "${REDIS_CONTAINER_NAME}"
}

#######################################
# Execute Redis CLI command
# Arguments:
#   $@ - Redis CLI arguments
# Returns: Redis CLI exit code
#######################################
redis::docker::exec_cli() {
    if ! redis::common::is_running; then
        log::error "Redis is not running"
        return 1
    fi
    
    local cmd=(docker exec -it "${REDIS_CONTAINER_NAME}" redis-cli)
    
    # Add password if configured
    if [[ -n "$REDIS_PASSWORD" ]]; then
        cmd+=(-a "$REDIS_PASSWORD")
    fi
    
    # Add port
    cmd+=(-p "$REDIS_INTERNAL_PORT")
    
    # Add user arguments
    cmd+=("$@")
    
    "${cmd[@]}"
}

#######################################
# Show connection information
#######################################
redis::docker::show_connection_info() {
    log::info "${MSG_CONNECTION_INFO}"
    log::info "${MSG_CONNECTION_HOST}"
    log::info "${MSG_CONNECTION_PORT}"
    log::info "${MSG_CONNECTION_CLI}"
    log::info "${MSG_CONNECTION_URL}"
    
    if [[ -z "$REDIS_PASSWORD" ]]; then
        log::warn "${MSG_WARN_NO_PASSWORD}"
    fi
}

#######################################
# Create client-specific Redis instance
# Arguments:
#   $1 - client ID
# Returns: 0 on success, 1 on failure
#######################################
redis::docker::create_client_instance() {
    local client_id="$1"
    
    if [[ -z "$client_id" ]]; then
        log::error "${MSG_ERROR_MISSING_PARAM}client_id"
        return 1
    fi
    
    local client_container="${REDIS_CLIENT_PREFIX}-${client_id}"
    local client_network="vrooli-${client_id}-network"
    
    # Check if client instance already exists
    if docker ps -a --format "{{.Names}}" | grep -q "^${client_container}$"; then
        log::error "${MSG_ERROR_CLIENT_EXISTS}${client_id}"
        return 1
    fi
    
    # Find available port
    local client_port
    client_port=$(redis::common::find_available_port "$REDIS_CLIENT_PORT_START" "$REDIS_CLIENT_PORT_END")
    
    if [[ -z "$client_port" ]]; then
        log::error "No available ports in range ${REDIS_CLIENT_PORT_START}-${REDIS_CLIENT_PORT_END}"
        return 1
    fi
    
    log::info "${MSG_CLIENT_CREATE_START}${client_id}"
    log::info "${MSG_CLIENT_PORT_ALLOCATED}${client_port}"
    
    # Create client network
    if ! docker network ls | grep -q "^${client_network}"; then
        docker network create "${client_network}" >/dev/null 2>&1
    fi
    
    # Create client directories in project root
    local project_config_dir
    project_config_dir="$(secrets::get_project_config_dir)"
    local client_data_dir="${project_config_dir}/clients/${client_id}/redis/data"
    local client_config_dir="${project_config_dir}/clients/${client_id}/redis/config"
    local client_log_dir="${project_config_dir}/clients/${client_id}/redis/logs"
    
    mkdir -p "${client_data_dir}" "${client_config_dir}" "${client_log_dir}"
    
    # Generate client-specific config
    local client_config="${client_config_dir}/redis.conf"
    cp "${REDIS_CONFIG_FILE}" "${client_config}"
    
    # Create client container
    local docker_cmd=(
        "docker" "run" "-d"
        "--name" "${client_container}"
        "--network" "${client_network}"
        "-p" "${client_port}:${REDIS_INTERNAL_PORT}"
        "-v" "${client_data_dir}:/data"
        "-v" "${client_config_dir}:/usr/local/etc/redis"
        "-v" "${client_log_dir}:/var/log/redis"
        "--restart" "unless-stopped"
        "--health-cmd" "redis-cli ping || exit 1"
        "--health-interval" "${REDIS_HEALTH_CHECK_INTERVAL}"
        "--health-timeout" "${REDIS_HEALTH_CHECK_TIMEOUT}"
        "--health-retries" "${REDIS_HEALTH_CHECK_RETRIES}"
        "--label" "vrooli.client=${client_id}"
        "--label" "vrooli.resource=redis"
        "${REDIS_IMAGE}"
        "redis-server" "/usr/local/etc/redis/redis.conf"
    )
    
    if "${docker_cmd[@]}" >/dev/null 2>&1; then
        # Save client configuration
        local project_config_dir
        project_config_dir="$(secrets::get_project_config_dir)"
        local client_config_file="${project_config_dir}/clients/${client_id}/redis.json"
        cat > "${client_config_file}" << EOF
{
  "clientId": "${client_id}",
  "resource": "redis",
  "container": "${client_container}",
  "network": "${client_network}",
  "port": ${client_port},
  "baseUrl": "redis://localhost:${client_port}",
  "dataDir": "${client_data_dir}",
  "configDir": "${client_config_dir}",
  "created": "$(date -Iseconds)"
}
EOF
        
        log::success "${MSG_CLIENT_CREATE_SUCCESS}"
        log::info "   Client: ${client_id}"
        log::info "   Port: ${client_port}"
        log::info "   URL: redis://localhost:${client_port}"
        log::info "   Container: ${client_container}"
        
        return 0
    else
        log::error "${MSG_CLIENT_CREATE_FAILED}"
        return 1
    fi
}

#######################################
# Destroy client-specific Redis instance
# Arguments:
#   $1 - client ID
# Returns: 0 on success, 1 on failure
#######################################
redis::docker::destroy_client_instance() {
    local client_id="$1"
    
    if [[ -z "$client_id" ]]; then
        log::error "${MSG_ERROR_MISSING_PARAM}client_id"
        return 1
    fi
    
    local client_container="${REDIS_CLIENT_PREFIX}-${client_id}"
    local client_network="vrooli-${client_id}-network"
    
    # Check if client instance exists
    if ! docker ps -a --format "{{.Names}}" | grep -q "^${client_container}$"; then
        log::error "${MSG_ERROR_CLIENT_NOT_FOUND}${client_id}"
        return 1
    fi
    
    log::info "${MSG_CLIENT_DESTROY_START}${client_id}"
    
    # Stop and remove container
    docker stop "${client_container}" >/dev/null 2>&1
    docker rm "${client_container}" >/dev/null 2>&1
    
    # Remove network if no other containers are using it
    local network_containers
    network_containers=$(docker network inspect "${client_network}" --format '{{len .Containers}}' 2>/dev/null || echo "0")
    if [[ "$network_containers" == "0" ]]; then
        docker network rm "${client_network}" >/dev/null 2>&1
    fi
    
    # Remove client directories and config
    local project_config_dir
    project_config_dir="$(secrets::get_project_config_dir)"
    trash::safe_remove "${project_config_dir}/clients/${client_id}/redis" --no-confirm 2>/dev/null || true
    rm -f "${project_config_dir}/clients/${client_id}/redis.json"
    
    log::success "${MSG_CLIENT_DESTROY_SUCCESS}"
    return 0
}