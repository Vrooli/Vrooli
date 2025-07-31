#!/usr/bin/env bash
# PostgreSQL Docker Management
# Functions for managing PostgreSQL Docker containers

#######################################
# Create Docker network if it doesn't exist
# Returns: 0 on success, 1 on failure
#######################################
postgres::docker::create_network() {
    if ! docker network ls | grep -q "\s${POSTGRES_NETWORK}\s"; then
        log::debug "Creating Docker network: ${POSTGRES_NETWORK}"
        if docker network create "${POSTGRES_NETWORK}" >/dev/null 2>&1; then
            log::debug "Docker network created successfully"
            return 0
        else
            log::error "Failed to create Docker network"
            return 1
        fi
    else
        log::debug "Docker network already exists: ${POSTGRES_NETWORK}"
        return 0
    fi
}

#######################################
# Pull PostgreSQL Docker image
# Returns: 0 on success, 1 on failure
#######################################
postgres::docker::pull_image() {
    log::info "${MSG_PULLING_IMAGE}"
    
    if docker pull "${POSTGRES_IMAGE}" 2>&1 | while read -r line; do
        if [[ "$line" =~ "Pulling from" ]] || [[ "$line" =~ "Status:" ]]; then
            log::debug "$line"
        fi
    done; then
        log::debug "PostgreSQL image pulled successfully"
        return 0
    else
        log::error "Failed to pull PostgreSQL image"
        return 1
    fi
}

#######################################
# Create basic PostgreSQL configuration
# Arguments:
#   $1 - config file path
#   $2 - template name
# Returns: 0 on success, 1 on failure
#######################################
postgres::docker::create_basic_config() {
    local config_file="$1"
    local template="$2"
    
    cat > "$config_file" << EOF
# Basic PostgreSQL Configuration - Generated for template: $template
# This is a fallback configuration when template file is not found

# Connection settings
max_connections = 100
listen_addresses = '*'
port = 5432

# Memory settings
shared_buffers = 128MB
work_mem = 4MB
maintenance_work_mem = 64MB

# WAL settings
wal_level = replica
fsync = on
synchronous_commit = on

# Logging
log_timezone = 'UTC'
log_statement = 'none'
log_min_duration_statement = -1
log_line_prefix = '%t [%p]: [%l-1] user=%u,db=%d '

# Locale settings
lc_messages = 'en_US.UTF-8'
lc_monetary = 'en_US.UTF-8'
lc_numeric = 'en_US.UTF-8'
lc_time = 'en_US.UTF-8'
default_text_search_config = 'pg_catalog.english'

# Autovacuum
autovacuum = on
EOF
    
    return 0
}

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
    local volume_name="${POSTGRES_VOLUME_PREFIX}-${instance_name}"
    
    # Ensure network exists
    postgres::docker::create_network || return 1
    
    # Ensure directories exist
    postgres::common::create_directories "$instance_name"
    
    # Prepare configuration template
    local config_dir="${POSTGRES_INSTANCES_DIR}/${instance_name}/config"
    mkdir -p "$config_dir"
    
    # Copy template configuration
    local template_file="${POSTGRES_TEMPLATE_DIR}/${template}.conf"
    if [[ -f "$template_file" ]]; then
        cp "$template_file" "$config_dir/postgresql.conf"
        log::info "Applied '$template' template configuration"
    else
        log::warn "Template '$template' not found, using default configuration"
        # Create basic configuration if template doesn't exist
        postgres::docker::create_basic_config "$config_dir/postgresql.conf" "$template"
    fi
    
    log::info "${MSG_STARTING_CONTAINER}"
    
    # Build docker run command
    local docker_cmd=(
        "docker" "run" "-d"
        "--name" "${container_name}"
        "--network" "${POSTGRES_NETWORK}"
        "-p" "${port}:5432"
        "-v" "${data_dir}:/var/lib/postgresql/data"
        "-v" "${config_dir}/postgresql.conf:/etc/postgresql/postgresql.conf"
        "-e" "POSTGRES_USER=${POSTGRES_DEFAULT_USER}"
        "-e" "POSTGRES_PASSWORD=${password}"
        "-e" "POSTGRES_DB=${POSTGRES_DEFAULT_DB}"
        "-e" "POSTGRES_INITDB_ARGS=--encoding=${POSTGRES_DEFAULT_ENCODING} --locale=${POSTGRES_DEFAULT_LOCALE}"
        "--restart" "unless-stopped"
        "--health-cmd" "pg_isready -U ${POSTGRES_DEFAULT_USER} -d ${POSTGRES_DEFAULT_DB}"
        "--health-interval" "${POSTGRES_HEALTH_CHECK_INTERVAL}s"
        "--health-timeout" "${POSTGRES_HEALTH_CHECK_TIMEOUT}s"
        "--health-retries" "${POSTGRES_HEALTH_CHECK_RETRIES}"
        "${POSTGRES_IMAGE}"
        "-c" "config_file=/etc/postgresql/postgresql.conf"
    )
    
    # Template configuration is now handled via mounted config file
    # This provides more comprehensive and maintainable configuration
    log::debug "Using template configuration: $template"
    
    # Run the container
    if "${docker_cmd[@]}" >/dev/null 2>&1; then
        log::debug "PostgreSQL container created successfully"
        
        # Wait a moment for container to initialize
        sleep "${POSTGRES_INITIALIZATION_WAIT}"
        
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
    
    if docker start "${container_name}" >/dev/null 2>&1; then
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
    
    log::info "Stopping PostgreSQL instance '$instance_name'..."
    
    if docker stop "${container_name}" >/dev/null 2>&1; then
        log::success "${MSG_STOP_SUCCESS}"
        return 0
    else
        log::error "Failed to stop PostgreSQL container"
        return 1
    fi
}

#######################################
# Restart PostgreSQL container
# Arguments:
#   $1 - instance name
# Returns: 0 on success, 1 on failure
#######################################
postgres::docker::restart() {
    local instance_name="${1:-main}"
    
    log::info "Restarting PostgreSQL instance '$instance_name'..."
    
    # Stop if running
    if postgres::common::is_running "$instance_name"; then
        postgres::docker::stop "$instance_name" || return 1
        sleep 2
    fi
    
    # Start again
    if postgres::docker::start "$instance_name"; then
        log::success "${MSG_RESTART_SUCCESS}"
        return 0
    else
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
    
    # Stop container if running
    if postgres::common::is_running "$instance_name"; then
        # Try graceful stop first, with timeout
        if ! timeout 30 docker stop "${container_name}" >/dev/null 2>&1; then
            if [[ "$force" == "true" ]]; then
                log::warn "Graceful stop timed out, force stopping container"
                docker kill "${container_name}" >/dev/null 2>&1 || true
            else
                log::error "Failed to stop container within 30 seconds. Use --force yes to force removal."
                return 1
            fi
        fi
    fi
    
    log::info "Removing PostgreSQL instance '$instance_name'..."
    
    if docker rm "${container_name}" >/dev/null 2>&1; then
        log::debug "PostgreSQL container removed successfully"
        return 0
    else
        log::error "Failed to remove PostgreSQL container"
        return 1
    fi
}

#######################################
# Execute command in PostgreSQL container
# Arguments:
#   $1 - instance name
#   $@ - Command to execute
# Returns: Command exit code
#######################################
postgres::docker::exec() {
    local instance_name="$1"
    shift
    local container_name="${POSTGRES_CONTAINER_PREFIX}-${instance_name}"
    
    if ! postgres::common::is_running "$instance_name"; then
        log::error "PostgreSQL instance '$instance_name' is not running"
        return 1
    fi
    
    docker exec "${container_name}" "$@"
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
    
    docker exec "${container_name}" psql -U "${POSTGRES_DEFAULT_USER}" -d "$database" -c "$sql_command"
}

#######################################
# Get container statistics
# Arguments:
#   $1 - instance name
# Output: JSON with container stats
#######################################
postgres::docker::stats() {
    local instance_name="${1:-main}"
    local container_name="${POSTGRES_CONTAINER_PREFIX}-${instance_name}"
    
    if ! postgres::common::is_running "$instance_name"; then
        echo "{\"error\": \"Container not running\"}"
        return 1
    fi
    
    docker stats "${container_name}" --no-stream --format "json" 2>/dev/null || echo "{}"
}

#######################################
# Check Docker availability
# Returns: 0 if available, 1 if not
#######################################
postgres::docker::check_docker() {
    if ! command -v docker >/dev/null 2>&1; then
        log::error "Docker is not installed"
        log::info "Please install Docker: https://docs.docker.com/get-docker/"
        return 1
    fi
    
    if ! docker info >/dev/null 2>&1; then
        log::error "Docker daemon is not running"
        log::info "Please start Docker and try again"
        return 1
    fi
    
    log::debug "Docker is available and running"
    return 0
}