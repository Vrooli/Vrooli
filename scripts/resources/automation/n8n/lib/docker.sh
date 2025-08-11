#!/usr/bin/env bash
# n8n Docker Management Functions
# Simplified Docker operations using unified container lifecycle management

# Source required modules
N8N_LIB_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
# shellcheck disable=SC1091
source "${N8N_LIB_DIR}/../../lib/docker-utils.sh" 2>/dev/null || true
# shellcheck disable=SC1091
source "${N8N_LIB_DIR}/../../lib/wait-utils.sh" 2>/dev/null || true
# shellcheck disable=SC1091
source "${N8N_LIB_DIR}/health.sh" 2>/dev/null || true
# shellcheck disable=SC1091
source "${N8N_LIB_DIR}/recovery.sh" 2>/dev/null || true
# shellcheck disable=SC1091
source "${N8N_LIB_DIR}/utils.sh" 2>/dev/null || true
# shellcheck disable=SC1091
source "${N8N_LIB_DIR}/constants.sh" 2>/dev/null || true

#######################################
# Build custom n8n Docker image
#######################################
n8n::build_custom_image() {
    n8n::log_with_context "info" "build" "Building custom n8n image with host access..."
    local docker_dir="$(dirname "$N8N_LIB_DIR")/docker"
    if [[ ! -f "$docker_dir/Dockerfile" ]] || [[ ! -f "$docker_dir/docker-entrypoint.sh" ]]; then
        n8n::log_with_context "error" "build" "Required files missing in $docker_dir"
        return 1
    fi
    if docker build -t "$N8N_CUSTOM_IMAGE" "$docker_dir"; then
        n8n::log_with_context "success" "build" "Custom n8n image built successfully"
        return 0
    else
        n8n::log_with_context "error" "build" "Failed to build custom image"
        return 1
    fi
}

#######################################
# Get n8n container configuration for unified lifecycle management
# Args: $1 - webhook_url (optional), $2 - auth_password (optional)
# Returns: JSON container configuration
#######################################
n8n::get_container_config() {
    local webhook_url="${1:-}"
    local auth_password="${2:-}"
    
    # Select image (custom if available, otherwise standard)
    local image_to_use="$N8N_IMAGE"
    if docker images --format "{{.Repository}}:{{.Tag}}" | grep -q "^${N8N_CUSTOM_IMAGE}$"; then
        image_to_use="$N8N_CUSTOM_IMAGE"
    fi
    
    # Build base volumes
    local volumes='{
        "'${N8N_DATA_DIR}'": "/home/node/.n8n"
    }'
    
    # Add host access volumes if using custom image
    if [[ "$BUILD_IMAGE" == "yes" ]]; then
        volumes=$(echo "$volumes" | jq '. + {
            "/var/run/docker.sock": "/var/run/docker.sock:rw",
            "'${PWD}'": "/workspace:rw",
            "/usr/bin": "/host/usr/bin:ro",
            "/bin": "/host/bin:ro",
            "/usr/local/bin": "/host/usr/local/bin:ro",
            "'$HOME'": "/host/home:rw"
        }')
    fi
    
    # Build environment variables
    local timezone
    timezone=$(timedatectl show -p Timezone --value 2>/dev/null || echo 'UTC')
    
    local env='{
        "GENERIC_TIMEZONE": "'$timezone'",
        "TZ": "'$timezone'",
        "N8N_DIAGNOSTICS_ENABLED": "false",
        "N8N_TEMPLATES_ENABLED": "true", 
        "N8N_PERSONALIZATION_ENABLED": "false",
        "N8N_RUNNERS_ENABLED": "true",
        "N8N_ENFORCE_SETTINGS_FILE_PERMISSIONS": "true",
        "EXECUTIONS_DATA_SAVE_ON_ERROR": "all",
        "EXECUTIONS_DATA_SAVE_ON_SUCCESS": "all",
        "EXECUTIONS_DATA_SAVE_ON_PROGRESS": "true",
        "EXECUTIONS_DATA_SAVE_MANUAL_EXECUTIONS": "true",
        "N8N_PUBLIC_API_DISABLED": "false"
    }'
    
    # Add webhook configuration if provided
    if [[ -n "$webhook_url" ]]; then
        local protocol host
        protocol=$(echo "$webhook_url" | sed 's|://.*||')
        host=$(echo "$webhook_url" | sed 's|.*://||' | sed 's|/.*||')
        env=$(echo "$env" | jq '. + {
            "WEBHOOK_URL": "'$webhook_url'",
            "N8N_PROTOCOL": "'$protocol'", 
            "N8N_HOST": "'$host'"
        }')
    fi
    
    # Add basic authentication if enabled
    if [[ "$BASIC_AUTH" == "yes" ]]; then
        env=$(echo "$env" | jq '. + {
            "N8N_BASIC_AUTH_ACTIVE": "true",
            "N8N_BASIC_AUTH_USER": "'$AUTH_USERNAME'",
            "N8N_BASIC_AUTH_PASSWORD": "'$auth_password'"
        }')
    fi
    
    # Add database configuration
    if [[ "$DATABASE_TYPE" == "postgres" ]]; then
        env=$(echo "$env" | jq '. + {
            "DB_TYPE": "postgresdb",
            "DB_POSTGRESDB_HOST": "'$N8N_DB_CONTAINER_NAME'",
            "DB_POSTGRESDB_PORT": "5432",
            "DB_POSTGRESDB_DATABASE": "n8n", 
            "DB_POSTGRESDB_USER": "n8n",
            "DB_POSTGRESDB_PASSWORD": "'$N8N_DB_PASSWORD'"
        }')
    else
        env=$(echo "$env" | jq '. + {
            "DB_TYPE": "sqlite",
            "DB_SQLITE_VACUUM_ON_STARTUP": "true"
        }')
    fi
    
    # Build extra args for tunnel mode
    local extra_args='[]'
    if [[ "$TUNNEL_ENABLED" == "yes" ]]; then
        extra_args='["--tunnel"]'
    fi
    
    # Return complete configuration
    local config='{
        "name": "'$N8N_CONTAINER_NAME'",
        "image": "'$image_to_use'",
        "ports": {"'$N8N_PORT'": "5678"},
        "volumes": '$volumes',
        "environment": '$env',
        "network": "'$N8N_NETWORK_NAME'",
        "restart_policy": "unless-stopped",
        "health": {
            "endpoint": "/healthz",
            "timeout": 60,
            "interval": 5
        },
        "extra_args": '$extra_args'
    }'
    
    echo "$config"
}


#######################################
# Stop n8n containers
#######################################
n8n::stop() {
    local config=$(n8n::get_container_config)
    docker::stop_with_config "$config" "false"
    
    # Stop PostgreSQL if used
    if [[ "$DATABASE_TYPE" == "postgres" ]] && n8n::postgres_running; then
        n8n::log_with_context "info" "docker" "Stopping PostgreSQL..."
        docker::stop_container "$N8N_DB_CONTAINER_NAME"
    fi
}

#######################################
# Handle existing running instance
# Returns: 0 if should continue, 1 if should stop
#######################################
n8n::handle_existing_instance() {
    if n8n::container_running && [[ "$FORCE" != "yes" ]]; then
        n8n::log_with_context "info" "docker" "$N8N_ERR_ALREADY_RUNNING on port $N8N_PORT"
        if n8n::comprehensive_health_check; then
            n8n::log_with_context "success" "docker" "Running instance is healthy"
            return 1  # Stop - already running and healthy
        else
            n8n::log_with_context "warn" "docker" "Running instance has health issues, restarting..."
            n8n::stop
        fi
    fi
    return 0  # Continue with startup
}

#######################################
# Run pre-flight health checks and recovery
# Returns: 0 if successful, 1 if failed
#######################################
n8n::run_preflight_checks() {
    log::info "Running pre-flight health checks..."
    # Check for filesystem corruption and recover
    if ! n8n::detect_filesystem_corruption; then
        log::warn "Filesystem corruption detected during startup"
        if ! n8n::auto_recover; then
            log::error "Failed to recover from filesystem corruption"
            return 1
        fi
    fi
    # Ensure directories are properly created
    if ! n8n::create_directories; then
        log::error "Failed to create required directories"
        return 1
    fi
    # Check if container exists
    if ! n8n::container_exists_any; then
        log::error "$N8N_ERR_CONTAINER_NOT_EXISTS. $N8N_FIX_RUN_INSTALL"
        return 1
    fi
    return 0
}

#######################################
# Start PostgreSQL database if needed
#######################################
n8n::start_database_if_needed() {
    if n8n::postgres_exists && ! n8n::postgres_running; then
        log::info "Starting PostgreSQL..."
        docker::start_container "$N8N_DB_CONTAINER_NAME"
        sleep 3
    fi
}


#######################################
# Start n8n container using unified lifecycle management
# Args: $1 - webhook_url (optional), $2 - auth_password (optional)
# Returns: 0 if successful, 1 if failed
#######################################
n8n::start_container() {
    local webhook_url="${1:-}"
    local auth_password="${2:-}"
    
    # Get container configuration
    local config
    config=$(n8n::get_container_config "$webhook_url" "$auth_password")
    
    # Use unified container lifecycle management
    if docker::start_with_config "$config"; then
        log::success "✅ n8n is healthy and ready on port $N8N_PORT"
        log::info "Access n8n at: $N8N_BASE_URL"
        return 0
    else
        n8n::log_with_context "error" "docker" "Failed to start n8n container"
        return 1
    fi
}

#######################################
# Start n8n with all dependencies
#######################################
n8n::start() {
    # Handle existing instance
    if ! n8n::handle_existing_instance; then
        return 0  # Already running and healthy
    fi
    
    log::info "Starting n8n..."
    
    # Run pre-flight checks
    if ! n8n::run_preflight_checks; then
        return 1
    fi
    
    # Start database if needed
    n8n::start_database_if_needed
    
    # Start n8n container
    n8n::start_container "$WEBHOOK_URL" "$AUTH_PASSWORD"
}

#######################################
# Restart n8n
#######################################
n8n::restart() {
    log::info "Restarting n8n..."
    n8n::stop
    sleep 2
    AUTO_RECOVER="yes" n8n::start
}

#######################################
# Show n8n logs
#######################################
n8n::logs() {
    if ! n8n::container_exists_any; then
        log::error "$N8N_ERR_CONTAINER_NOT_EXISTS"
        return 1
    fi
    log::info "Showing n8n logs (last ${LINES:-50} lines)..."
    log::info "Use 'docker logs -f $N8N_CONTAINER_NAME' to follow logs in real-time"
    echo
    docker logs --tail "${LINES:-50}" "$N8N_CONTAINER_NAME"
}

#######################################
# Remove n8n container (simplified wrapper)
#######################################
n8n::remove_container() {
    docker::remove_container "$N8N_CONTAINER_NAME" "true"
}

#######################################
# Remove PostgreSQL container (simplified wrapper)
#######################################
n8n::remove_postgres_container() {
    docker::remove_container "$N8N_DB_CONTAINER_NAME" "true"
}

#######################################
# Remove Docker network (simplified wrapper)
#######################################
n8n::remove_network() {
    docker::remove_network "$N8N_NETWORK_NAME"
}

#######################################
# Pull n8n image (simplified wrapper)
#######################################
n8n::pull_image() {
    docker::pull_image "$N8N_IMAGE"
}
