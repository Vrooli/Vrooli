#!/usr/bin/env bash
# Redis Docker Management - Ultra-Simplified
# Uses docker-resource-utils.sh for minimal boilerplate

# Source var.sh to get proper directory variables
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*/../../.." && builtin pwd)}"
_REDIS_DOCKER_DIR="${APP_ROOT}/resources/redis/lib"
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/var.sh"

# Source shared libraries
# shellcheck disable=SC1091
source "${var_LIB_SERVICE_DIR}/secrets.sh"
# shellcheck disable=SC1091
source "${var_SCRIPTS_RESOURCES_LIB_DIR}/docker-resource-utils.sh"

# Create and start Redis container
redis::docker::create_container() {
    # Ensure configuration is generated
    redis::common::generate_config || return 1
    
    # Ensure volumes exist
    redis::common::create_volumes || return 1
    
    # Pull image if needed
    log::info "${MSG_PULLING_IMAGE}"
    docker::pull_image "$REDIS_IMAGE"
    
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
        
        # Wait a moment for container to initialize
        sleep "${REDIS_INITIALIZATION_WAIT:-2}"
        return 0
    else
        log::error "Failed to create Redis container"
        return 1
    fi
}

# Start Redis container
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

# Stop Redis container
redis::docker::stop() {
    if ! docker::container_exists "$REDIS_CONTAINER_NAME"; then
        log::warn "Redis container does not exist"
        return 0
    fi
    
    docker::stop_container "$REDIS_CONTAINER_NAME"
}

# Restart Redis container
redis::docker::restart() {
    log::info "Restarting Redis container..."
    
    if docker::restart_container "$REDIS_CONTAINER_NAME"; then
        log::success "${MSG_RESTART_SUCCESS}"
        redis::docker::show_connection_info
    fi
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
    
    # For interactive mode with TTY
    if [[ -t 0 ]]; then
        docker_resource::exec_interactive "$REDIS_CONTAINER_NAME" \
            redis-cli ${REDIS_PASSWORD:+-a "$REDIS_PASSWORD"} -p "$REDIS_INTERNAL_PORT" "$@"
    else
        # For single commands, use database command execution
        docker_resource::exec_database_command "$REDIS_CONTAINER_NAME" "redis" "$*"
    fi
}

# Show connection information
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
    
    # Use shared database client creation
    local client_port
    client_port=$(docker_resource::create_database_client_instance \
        "redis" \
        "$client_id" \
        "$REDIS_IMAGE" \
        "$REDIS_INTERNAL_PORT" \
        "$REDIS_CLIENT_PORT_START" \
        "$REDIS_CLIENT_PORT_END" \
        "${REDIS_PASSWORD:-}")
    
    if [[ $? -eq 0 && -n "$client_port" ]]; then
        # Save client configuration metadata
        local project_config_dir
        project_config_dir="$(secrets::get_project_config_dir)"
        mkdir -p "${project_config_dir}/clients/${client_id}"
        
        local client_config_file="${project_config_dir}/clients/${client_id}/redis.json"
        docker_resource::generate_client_metadata \
            "$client_id" \
            "redis" \
            "redis-client-${client_id}" \
            "vrooli-${client_id}-network" \
            "$client_port" \
            "redis://localhost:${client_port}" > "$client_config_file"
        
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