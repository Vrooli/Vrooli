#!/usr/bin/env bash
# PostgreSQL Docker Management - Ultra-Simplified
# Uses docker-resource-utils.sh for minimal boilerplate

# Source var.sh to get proper directory variables
_POSTGRES_DOCKER_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck disable=SC1091
source "${_POSTGRES_DOCKER_DIR}/../../../../lib/utils/var.sh"

# Source shared libraries
# shellcheck disable=SC1091
source "${var_LIB_SERVICE_DIR}/secrets.sh"
# shellcheck disable=SC1091
source "${var_SCRIPTS_RESOURCES_LIB_DIR}/docker-resource-utils.sh"

#######################################
# Create and start PostgreSQL container
# Arguments:
#   $1 - instance name
#   $2 - port number
#   $3 - password
#   $4 - template (optional)
# Returns: 0 on success, 1 on failure
#######################################
postgres::docker::create_container() {
    local instance_name="${1:-main}"
    local port="$2"
    local password="$3"
    local template="${4:-development}"
    local container_name="${POSTGRES_CONTAINER_PREFIX}-${instance_name}"
    local data_dir="${POSTGRES_INSTANCES_DIR}/${instance_name}/data"
    local config_dir="${POSTGRES_INSTANCES_DIR}/${instance_name}/config"
    
    # Pull image if needed
    log::info "${MSG_PULLING_IMAGE}"
    docker::pull_image "$POSTGRES_IMAGE"
    
    # Ensure directories exist
    postgres::common::create_directories "$instance_name"
    
    # Prepare configuration template
    mkdir -p "$config_dir"
    
    # Copy template configuration
    local template_file="${POSTGRES_TEMPLATE_DIR}/${template}.conf"
    if [[ -f "$template_file" ]]; then
        cp "$template_file" "$config_dir/postgresql.conf"
        log::info "Applied '$template' template configuration"
    else
        # Fall back to development template if specified template not found
        local fallback_template="${POSTGRES_TEMPLATE_DIR}/development.conf"
        if [[ -f "$fallback_template" ]]; then
            log::warn "Template '$template' not found, using development template"
            cp "$fallback_template" "$config_dir/postgresql.conf"
        else
            log::error "No PostgreSQL configuration templates available"
            return 1
        fi
    fi
    
    log::info "${MSG_STARTING_CONTAINER}"
    
    # Validate required environment variables
    docker_resource::validate_env_vars "POSTGRES_IMAGE" "POSTGRES_DEFAULT_USER" "POSTGRES_DEFAULT_DB" || return 1
    
    # Prepare environment variables array
    local env_vars=(
        "POSTGRES_USER=${POSTGRES_DEFAULT_USER}"
        "POSTGRES_PASSWORD=${password}"
        "POSTGRES_DB=${POSTGRES_DEFAULT_DB}"
        "POSTGRES_INITDB_ARGS=--encoding=${POSTGRES_DEFAULT_ENCODING} --locale=${POSTGRES_DEFAULT_LOCALE}"
    )
    
    # Port mappings and volumes
    local port_mappings="${port}:5432"
    local volumes="${data_dir}:/var/lib/postgresql/data ${config_dir}/postgresql.conf:/etc/postgresql/postgresql.conf:ro"
    
    # Health check command with configurable params
    local health_cmd="pg_isready -U ${POSTGRES_DEFAULT_USER} -d ${POSTGRES_DEFAULT_DB}"
    
    # Docker options for PostgreSQL-specific health check configuration
    local docker_opts=(
        "--health-interval" "${POSTGRES_HEALTH_CHECK_INTERVAL}s"
        "--health-timeout" "${POSTGRES_HEALTH_CHECK_TIMEOUT}s"
        "--health-retries" "${POSTGRES_HEALTH_CHECK_RETRIES}"
    )
    
    # Command arguments for PostgreSQL
    local entrypoint_cmd=(
        "-c" "config_file=/etc/postgresql/postgresql.conf"
    )
    
    # Use advanced creation function
    if docker_resource::create_service_advanced \
        "${container_name}" \
        "${POSTGRES_IMAGE}" \
        "$port_mappings" \
        "${POSTGRES_NETWORK}" \
        "$volumes" \
        "env_vars" \
        "docker_opts" \
        "$health_cmd" \
        "entrypoint_cmd"; then
        
        sleep "${POSTGRES_INITIALIZATION_WAIT:-3}"
        return 0
    else
        log::error "Failed to create PostgreSQL container"
        return 1
    fi
}

#######################################
# Start PostgreSQL container
# Arguments:
#   $1 - instance name
# Returns: 0 on success, 1 on failure
#######################################
postgres::docker::start() {
    local instance_name="${1:-main}"
    local container_name="${POSTGRES_CONTAINER_PREFIX}-${instance_name}"
    
    if ! postgres::common::container_exists "$instance_name"; then
        log::error "PostgreSQL instance '$instance_name' does not exist. Run create first."
        return 1
    fi
    
    if postgres::common::is_running "$instance_name"; then
        log::info "PostgreSQL instance '$instance_name' is already running"
        return 0
    fi
    
    log::info "Starting PostgreSQL instance '$instance_name'..."
    
    if docker::start_container "${container_name}"; then
        # Wait for container to be ready
        if postgres::common::wait_for_ready "$instance_name"; then
            local port=$(postgres::common::get_instance_config "$instance_name" "port")
            log::success "${MSG_START_SUCCESS}"
            log::info "${MSG_INSTANCE_AVAILABLE}: localhost:${port}"
            return 0
        else
            log::error "${MSG_START_FAILED}"
            return 1
        fi
    else
        log::error "Failed to start PostgreSQL container"
        return 1
    fi
}

#######################################
# Stop PostgreSQL container
# Arguments:
#   $1 - instance name
# Returns: 0 on success, 1 on failure
#######################################
postgres::docker::stop() {
    local instance_name="${1:-main}"
    local container_name="${POSTGRES_CONTAINER_PREFIX}-${instance_name}"
    
    if ! postgres::common::container_exists "$instance_name"; then
        log::warn "PostgreSQL instance '$instance_name' does not exist"
        return 0
    fi
    
    if ! postgres::common::is_running "$instance_name"; then
        log::info "PostgreSQL instance '$instance_name' is not running"
        return 0
    fi
    
    docker::stop_container "${container_name}"
}

#######################################
# Restart PostgreSQL container
# Arguments:
#   $1 - instance name
# Returns: 0 on success, 1 on failure
#######################################
postgres::docker::restart() {
    local instance_name="${1:-main}"
    local container_name="${POSTGRES_CONTAINER_PREFIX}-${instance_name}"
    
    log::info "Restarting PostgreSQL instance '$instance_name'..."
    
    if docker::restart_container "${container_name}"; then
        # Wait for container to be ready
        if postgres::common::wait_for_ready "$instance_name"; then
            log::success "${MSG_RESTART_SUCCESS}"
            return 0
        else
            log::error "PostgreSQL restart failed - container not ready"
            return 1
        fi
    else
        log::error "Failed to restart PostgreSQL"
        return 1
    fi
}

#######################################
# Remove PostgreSQL container
# Arguments:
#   $1 - instance name
#   $2 - Force removal (optional, default: false)
# Returns: 0 on success, 1 on failure
#######################################
postgres::docker::remove() {
    local instance_name="${1:-main}"
    local force=${2:-false}
    local container_name="${POSTGRES_CONTAINER_PREFIX}-${instance_name}"
    
    if ! postgres::common::container_exists "$instance_name"; then
        log::debug "PostgreSQL instance '$instance_name' does not exist"
        return 0
    fi
    
    docker::remove_container "${container_name}" "$force"
}

#######################################
# Execute SQL command in PostgreSQL container
# Arguments:
#   $1 - instance name
#   $2 - SQL command
#   $3 - database name (optional, defaults to POSTGRES_DEFAULT_DB)
# Returns: Command exit code
#######################################
postgres::docker::exec_sql() {
    local instance_name="$1"
    local sql_command="$2"
    local database="${3:-$POSTGRES_DEFAULT_DB}"
    local container_name="${POSTGRES_CONTAINER_PREFIX}-${instance_name}"
    
    if ! postgres::common::is_running "$instance_name"; then
        log::error "PostgreSQL instance '$instance_name' is not running"
        return 1
    fi
    
    # Use shared database command execution
    POSTGRES_USER="${POSTGRES_DEFAULT_USER}" \
    docker_resource::exec_database_command "$container_name" "postgres" "$sql_command"
}

#######################################
# Create client-specific PostgreSQL instance
# Arguments: $1 - client ID
# Returns: 0 on success, 1 on failure
#######################################
postgres::docker::create_client_instance() {
    local client_id="$1"
    
    if [[ -z "$client_id" ]]; then
        log::error "Missing required parameter: client_id"
        return 1
    fi
    
    log::info "Creating PostgreSQL client instance for: ${client_id}"
    
    # Use shared database client creation
    local client_port
    client_port=$(docker_resource::create_database_client_instance \
        "postgres" \
        "$client_id" \
        "$POSTGRES_IMAGE" \
        "5432" \
        "${POSTGRES_CLIENT_PORT_START:-5434}" \
        "${POSTGRES_CLIENT_PORT_END:-5450}" \
        "${POSTGRES_PASSWORD:-postgres}")
    
    if [[ $? -eq 0 && -n "$client_port" ]]; then
        # Save client configuration metadata
        local project_config_dir
        project_config_dir="$(secrets::get_project_config_dir)"
        mkdir -p "${project_config_dir}/clients/${client_id}"
        
        local client_config_file="${project_config_dir}/clients/${client_id}/postgres.json"
        docker_resource::generate_client_metadata \
            "$client_id" \
            "postgres" \
            "postgres-client-${client_id}" \
            "vrooli-${client_id}-network" \
            "$client_port" \
            "postgresql://postgres@localhost:${client_port}/postgres" > "$client_config_file"
        
        log::success "PostgreSQL client instance created successfully"
        log::info "   Client: ${client_id}"
        log::info "   Port: ${client_port}"
        log::info "   URL: postgresql://postgres@localhost:${client_port}/postgres"
        
        return 0
    else
        log::error "Failed to create PostgreSQL client instance"
        return 1
    fi
}

#######################################
# Destroy client-specific PostgreSQL instance
# Arguments: $1 - client ID
# Returns: 0 on success, 1 on failure
#######################################
postgres::docker::destroy_client_instance() {
    local client_id="$1"
    
    if [[ -z "$client_id" ]]; then
        log::error "Missing required parameter: client_id"
        return 1
    fi
    
    log::info "Destroying PostgreSQL client instance: ${client_id}"
    
    # Use simplified client removal
    local project_config_dir
    project_config_dir="$(secrets::get_project_config_dir)"
    
    docker_resource::remove_client_instance \
        "postgres" \
        "$client_id" \
        "$project_config_dir" \
        "true"
    
    # Remove client configuration metadata safely
    trash::safe_remove "${project_config_dir}/clients/${client_id}/postgres.json" --no-confirm 2>/dev/null || true
    
    log::success "PostgreSQL client instance destroyed successfully"
    return 0
}