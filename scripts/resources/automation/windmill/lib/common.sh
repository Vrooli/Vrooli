#!/usr/bin/env bash
# Windmill Common Functions
# Utility functions specific to Windmill management

#######################################
# Check if Docker and Docker Compose are available
# Returns: 0 if available, 1 otherwise
#######################################
windmill::check_docker_requirements() {
    if ! system::is_command "docker"; then
        windmill::show_docker_error
        return 1
    fi
    
    if ! docker info >/dev/null 2>&1; then
        windmill::show_permission_error
        return 1
    fi
    
    if ! system::is_command "docker" || ! docker compose version >/dev/null 2>&1; then
        log::error "Docker Compose is required but not available"
        log::info "Please install Docker Compose v2: https://docs.docker.com/compose/install/"
        return 1
    fi
    
    return 0
}

#######################################
# Check system requirements for Windmill
# Returns: 0 if requirements met, 1 otherwise
#######################################
windmill::check_system_requirements() {
    local errors=()
    
    # Check available memory
    local memory_gb
    if memory_gb=$(system::get_memory_gb 2>/dev/null); then
        if [[ $memory_gb -lt $WINDMILL_MIN_MEMORY_GB ]]; then
            errors+=("Insufficient memory: ${memory_gb}GB available, ${WINDMILL_MIN_MEMORY_GB}GB required")
        fi
    else
        log::warn "Could not determine available memory"
    fi
    
    # Check available disk space
    local disk_gb
    if disk_gb=$(df "$HOME" --output=avail --block-size=1G | tail -n1 | tr -d ' ' 2>/dev/null); then
        if [[ $disk_gb -lt $WINDMILL_MIN_DISK_GB ]]; then
            errors+=("Insufficient disk space: ${disk_gb}GB available, ${WINDMILL_MIN_DISK_GB}GB required")
        fi
    else
        log::warn "Could not determine available disk space"
    fi
    
    # Check CPU cores
    local cpu_cores
    if cpu_cores=$(nproc 2>/dev/null); then
        if [[ $cpu_cores -lt $WINDMILL_MIN_CPU_CORES ]]; then
            errors+=("Insufficient CPU cores: ${cpu_cores} available, ${WINDMILL_MIN_CPU_CORES} recommended")
        fi
    fi
    
    if [[ ${#errors[@]} -gt 0 ]]; then
        log::error "System requirements not met:"
        for error in "${errors[@]}"; do
            log::error "  • $error"
        done
        
        if [[ "$FORCE" != "yes" ]]; then
            log::info "Use --force yes to proceed anyway (not recommended)"
            return 1
        else
            log::warn "Proceeding with insufficient resources (--force specified)"
        fi
    fi
    
    return 0
}

#######################################
# Generate environment file from template and configuration
# Arguments:
#   $1 - Path to environment file (optional, defaults to $WINDMILL_ENV_FILE)
#######################################
windmill::generate_env_file() {
    local env_file="${1:-$WINDMILL_ENV_FILE}"
    local template_file="${SCRIPT_DIR}/docker/.env.template"
    
    if [[ ! -f "$template_file" ]]; then
        log::error "Environment template not found: $template_file"
        return 1
    fi
    
    log::info "Generating environment file: $env_file"
    
    # Copy template and substitute variables
    cp "$template_file" "$env_file"
    
    # Update values with current configuration
    sed -i "s|^WINDMILL_SERVER_PORT=.*|WINDMILL_SERVER_PORT=$WINDMILL_SERVER_PORT|" "$env_file"
    sed -i "s|^WINDMILL_PROJECT_NAME=.*|WINDMILL_PROJECT_NAME=$WINDMILL_PROJECT_NAME|" "$env_file"
    sed -i "s|^WINDMILL_WORKER_REPLICAS=.*|WINDMILL_WORKER_REPLICAS=$WORKER_COUNT|" "$env_file"
    sed -i "s|^WINDMILL_WORKER_MEMORY_LIMIT=.*|WINDMILL_WORKER_MEMORY_LIMIT=$WORKER_MEMORY_LIMIT|" "$env_file"
    sed -i "s|^WINDMILL_SUPERADMIN_EMAIL=.*|WINDMILL_SUPERADMIN_EMAIL=$SUPERADMIN_EMAIL|" "$env_file"
    sed -i "s|^WINDMILL_SUPERADMIN_PASSWORD=.*|WINDMILL_SUPERADMIN_PASSWORD=$SUPERADMIN_PASSWORD|" "$env_file"
    sed -i "s|^WINDMILL_JWT_SECRET=.*|WINDMILL_JWT_SECRET=$WINDMILL_JWT_SECRET|" "$env_file"
    sed -i "s|^WINDMILL_LOG_LEVEL=.*|WINDMILL_LOG_LEVEL=$WINDMILL_LOG_LEVEL|" "$env_file"
    
    # Configure database
    if [[ "$EXTERNAL_DB" == "yes" && -n "$DB_URL" ]]; then
        sed -i "s|^WINDMILL_DB_EXTERNAL=.*|WINDMILL_DB_EXTERNAL=yes|" "$env_file"
        sed -i "s|^# WINDMILL_DATABASE_URL=.*|WINDMILL_DATABASE_URL=$DB_URL|" "$env_file"
    else
        sed -i "s|^WINDMILL_DB_PASSWORD=.*|WINDMILL_DB_PASSWORD=$WINDMILL_DB_PASSWORD|" "$env_file"
    fi
    
    # Configure profiles based on enabled features
    local profiles="internal-db,workers"
    
    if [[ "$DISABLE_NATIVE_WORKER" != "true" ]]; then
        profiles+=",native-worker"
    fi
    
    if [[ "$DISABLE_LSP" != "true" ]]; then
        profiles+=",lsp"
    fi
    
    if [[ "$ENABLE_MULTIPLAYER" == "true" ]]; then
        profiles+=",multiplayer"
    fi
    
    sed -i "s|^COMPOSE_PROFILES=.*|COMPOSE_PROFILES=$profiles|" "$env_file"
    
    # Set proper file permissions
    chmod 600 "$env_file"
    
    log::success "Environment file generated successfully"
    return 0
}

#######################################
# Validate port availability
# Arguments:
#   $1 - Port number to check
# Returns: 0 if available, 1 if in use
#######################################
windmill::validate_port() {
    local port="$1"
    
    if ! resources::validate_port "windmill" "$port" "$FORCE"; then
        windmill::show_port_error "$port"
        return 1
    fi
    
    return 0
}

#######################################
# Get Docker Compose command with proper context
# Arguments:
#   $@ - Docker Compose subcommand and arguments
# Outputs: Complete docker compose command
#######################################
windmill::compose_cmd() {
    local compose_args=()
    
    # Add project name
    compose_args+=("--project-name" "$WINDMILL_PROJECT_NAME")
    
    # Add compose file
    if [[ -f "$WINDMILL_COMPOSE_FILE" ]]; then
        compose_args+=("--file" "$WINDMILL_COMPOSE_FILE")
    else
        log::error "Docker Compose file not found: $WINDMILL_COMPOSE_FILE"
        return 1
    fi
    
    # Add environment file if it exists
    if [[ -f "$WINDMILL_ENV_FILE" ]]; then
        compose_args+=("--env-file" "$WINDMILL_ENV_FILE")
    fi
    
    # Execute docker compose with arguments
    docker compose "${compose_args[@]}" "$@"
}

#######################################
# Check if Windmill is installed (compose file and env exist)
# Returns: 0 if installed, 1 otherwise
#######################################
windmill::is_installed() {
    [[ -f "$WINDMILL_COMPOSE_FILE" && -f "$WINDMILL_ENV_FILE" ]]
}

#######################################
# Check if Windmill services are running
# Returns: 0 if running, 1 otherwise
#######################################
windmill::is_running() {
    if ! windmill::is_installed; then
        return 1
    fi
    
    # Check if server container is running
    windmill::compose_cmd ps --services --filter "status=running" | grep -q "windmill-server"
}

#######################################
# Check if Windmill API is healthy
# Returns: 0 if healthy, 1 otherwise
#######################################
windmill::is_healthy() {
    if ! windmill::is_running; then
        return 1
    fi
    
    # Check API health endpoint
    resources::check_http_health "$WINDMILL_BASE_URL" "/api/version"
}

#######################################
# Wait for Windmill to become healthy
# Arguments:
#   $1 - Timeout in seconds (optional, default: 120)
# Returns: 0 if healthy, 1 if timeout
#######################################
windmill::wait_for_health() {
    local timeout="${1:-$WINDMILL_STARTUP_TIMEOUT}"
    local interval=5
    local attempts=$((timeout / interval))
    
    log::info "Waiting for Windmill to become healthy (timeout: ${timeout}s)..."
    
    for ((i=1; i<=attempts; i++)); do
        if windmill::is_healthy; then
            log::success "Windmill is healthy and ready!"
            return 0
        fi
        
        if [[ $i -lt $attempts ]]; then
            log::info "Health check $i/$attempts failed, retrying in ${interval}s..."
            sleep $interval
        fi
    done
    
    log::error "Windmill failed to become healthy within ${timeout}s"
    return 1
}

#######################################
# Get the status of Windmill services
# Outputs: Service status information
#######################################
windmill::get_service_status() {
    if ! windmill::is_installed; then
        echo "not_installed"
        return 1
    fi
    
    local running_services
    running_services=$(windmill::compose_cmd ps --services --filter "status=running" 2>/dev/null | wc -l)
    
    if [[ $running_services -eq 0 ]]; then
        echo "stopped"
    elif windmill::is_healthy; then
        echo "healthy"
    else
        echo "unhealthy"
    fi
}

#######################################
# Create necessary directories
#######################################
windmill::create_directories() {
    local dirs=(
        "$WINDMILL_DATA_DIR"
        "$WINDMILL_BACKUP_DIR"
        "$(dirname "$WINDMILL_ENV_FILE")"
    )
    
    for dir in "${dirs[@]}"; do
        if [[ ! -d "$dir" ]]; then
            log::info "Creating directory: $dir"
            mkdir -p "$dir" || {
                log::error "Failed to create directory: $dir"
                return 1
            }
        fi
    done
    
    return 0
}

#######################################
# Update Windmill configuration in Vrooli resources
#######################################
windmill::update_vrooli_config() {
    local additional_config='{"api":{"version":"v1","workspacesEndpoint":"/api/w/list","scriptsEndpoint":"/api/w/{workspace}/scripts","jobsEndpoint":"/api/w/{workspace}/jobs","authEndpoint":"/api/auth"},"features":{"codeFirst":true,"multiLanguage":true,"workflows":true,"scheduling":true,"webhooks":true}}'
    
    resources::update_config "automation" "windmill" "$WINDMILL_BASE_URL" "$additional_config"
}

#######################################
# Remove Windmill configuration from Vrooli resources
#######################################
windmill::remove_vrooli_config() {
    resources::remove_config "automation" "windmill"
}