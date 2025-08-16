#!/usr/bin/env bash
# Redis Docker Management - Ultra-Simplified
# Uses docker-resource-utils.sh for minimal boilerplate

# Source var.sh to get proper directory variables
_REDIS_DOCKER_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck disable=SC1091
source "${_REDIS_DOCKER_DIR}/../../../../lib/utils/var.sh"

# Source shared libraries
# shellcheck disable=SC1091
source "${var_LIB_SERVICE_DIR}/secrets.sh"
# shellcheck disable=SC1091
source "${var_SCRIPTS_RESOURCES_LIB_DIR}/docker-resource-utils.sh"

#######################################
# Create and start Redis container
# Returns: 0 on success, 1 on failure
#######################################
redis::docker::create_container() {
    # Ensure configuration is generated
    redis::common::generate_config || return 1
    
    # Ensure volumes exist
    redis::common::create_volumes || return 1
    
    log::info "${MSG_STARTING_CONTAINER}"
    
    # Create Redis service with ultra-simple API (data + config volumes, logs to stdout)
    if docker_resource::create_service_with_command \
        "$REDIS_CONTAINER_NAME" \
        "$REDIS_IMAGE" \
        "$REDIS_PORT" \
        "$REDIS_INTERNAL_PORT" \
        "$REDIS_NETWORK_NAME" \
        "$REDIS_VOLUME_NAME:/data $REDIS_TEMP_CONFIG_DIR:/usr/local/etc/redis:ro" \
        "redis-cli ping || exit 1" \
        "redis-server" "/usr/local/etc/redis/redis.conf" ${REDIS_PASSWORD:+--requirepass "$REDIS_PASSWORD"}; then
        
        log::debug "Redis container created successfully"
        
        # Wait a moment for container to initialize
        sleep "${REDIS_INITIALIZATION_WAIT:-2}"
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
    if ! docker::container_exists "$REDIS_CONTAINER_NAME"; then
        log::error "Redis container does not exist. Run install first."
        return 1
    fi
    
    log::info "Starting Redis container..."
    
    if docker::start_container "$REDIS_CONTAINER_NAME"; then
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
    if ! docker::container_exists "$REDIS_CONTAINER_NAME"; then
        log::warn "Redis container does not exist"
        return 0
    fi
    
    log::info "Stopping Redis container..."
    
    if docker::stop_container "$REDIS_CONTAINER_NAME"; then
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
    
    if docker::restart_container "$REDIS_CONTAINER_NAME"; then
        log::success "${MSG_RESTART_SUCCESS}"
        redis::docker::show_connection_info
        return 0
    else
        log::error "Failed to restart Redis"
        return 1
    fi
}

#######################################
# Remove Redis container
# Arguments: $1 - remove data (yes/no)
# Returns: 0 on success, 1 on failure
#######################################
redis::docker::remove_container() {
    local remove_data="${1:-no}"
    
    # Stop and remove container
    docker::remove_container "$REDIS_CONTAINER_NAME" "true"
    
    # Note: Data removal is handled by Docker volume management
    # Docker volumes will be removed automatically when containers are removed
    
    return 0
}

#######################################
# Pull Redis Docker image
# Returns: 0 on success, 1 on failure
#######################################
redis::docker::pull_image() {
    log::info "${MSG_PULLING_IMAGE}"
    docker::pull_image "$REDIS_IMAGE"
}

#######################################
# Show Redis container logs
# Arguments: $1 - number of lines (optional)
# Returns: 0 on success, 1 on failure
#######################################
redis::docker::show_logs() {
    local lines="${1:-50}"
    docker::get_logs "$REDIS_CONTAINER_NAME" "$lines"
}

#######################################
# Execute Redis CLI command
# Arguments: $@ - Redis CLI arguments
# Returns: Redis CLI exit code
#######################################
redis::docker::exec_cli() {
    if ! docker::is_running "$REDIS_CONTAINER_NAME"; then
        log::error "Redis is not running"
        return 1
    fi
    
    # Use -it only if we have a TTY (interactive terminal)
    local docker_flags=""
    if [[ -t 0 ]]; then
        docker_flags="-it"
    fi
    local cmd=(docker exec $docker_flags "$REDIS_CONTAINER_NAME" redis-cli)
    
    # Add password if configured
    if [[ -n "$REDIS_PASSWORD" ]]; then
        cmd+=(-a "$REDIS_PASSWORD")
    fi
    
    # Add port and user arguments
    cmd+=(-p "$REDIS_INTERNAL_PORT" "$@")
    
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
# Arguments: $1 - client ID
# Returns: 0 on success, 1 on failure
#######################################
redis::docker::create_client_instance() {
    local client_id="$1"
    
    if [[ -z "$client_id" ]]; then
        log::error "${MSG_ERROR_MISSING_PARAM}client_id"
        return 1
    fi
    
    log::info "${MSG_CLIENT_CREATE_START}${client_id}"
    
    # Use simplified client creation
    local client_port project_config_dir
    project_config_dir="$(secrets::get_project_config_dir)"
    
    client_port=$(docker_resource::create_client_instance \
        "redis" \
        "$client_id" \
        "$REDIS_IMAGE" \
        "$REDIS_INTERNAL_PORT" \
        "$REDIS_CLIENT_PORT_START" \
        "$REDIS_CLIENT_PORT_END" \
        "$project_config_dir" \
        "$REDIS_CONFIG_FILE")
    
    if [[ $? -eq 0 && -n "$client_port" ]]; then
        # Save client configuration metadata
        local client_config_file="${project_config_dir}/clients/${client_id}/redis.json"
        cat > "$client_config_file" << EOF
{
  "clientId": "${client_id}",
  "resource": "redis",
  "container": "redis-client-${client_id}",
  "network": "vrooli-${client_id}-network",
  "port": ${client_port},
  "baseUrl": "redis://localhost:${client_port}",
  "created": "$(date -Iseconds)"
}
EOF
        
        log::success "${MSG_CLIENT_CREATE_SUCCESS}"
        log::info "   Client: ${client_id}"
        log::info "   Port: ${client_port}"
        log::info "   URL: redis://localhost:${client_port}"
        
        return 0
    else
        log::error "${MSG_CLIENT_CREATE_FAILED}"
        return 1
    fi
}

#######################################
# Destroy client-specific Redis instance
# Arguments: $1 - client ID
# Returns: 0 on success, 1 on failure
#######################################
redis::docker::destroy_client_instance() {
    local client_id="$1"
    
    if [[ -z "$client_id" ]]; then
        log::error "${MSG_ERROR_MISSING_PARAM}client_id"
        return 1
    fi
    
    log::info "${MSG_CLIENT_DESTROY_START}${client_id}"
    
    # Use simplified client removal
    local project_config_dir
    project_config_dir="$(secrets::get_project_config_dir)"
    
    docker_resource::remove_client_instance \
        "redis" \
        "$client_id" \
        "$project_config_dir" \
        "true"
    
    # Remove client configuration metadata safely
    trash::safe_remove "${project_config_dir}/clients/${client_id}/redis.json" --no-confirm 2>/dev/null || true
    
    log::success "${MSG_CLIENT_DESTROY_SUCCESS}"
    return 0
}