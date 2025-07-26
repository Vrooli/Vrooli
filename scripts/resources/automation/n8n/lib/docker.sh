#!/usr/bin/env bash
# n8n Docker Management Functions
# All Docker-related operations for n8n containers

#######################################
# Build custom n8n Docker image
#######################################
n8n::build_custom_image() {
    log::info "Building custom n8n image with host access..."
    
    # Ensure files exist
    local docker_dir="$SCRIPT_DIR/docker"
    if [[ ! -f "$docker_dir/Dockerfile" ]] || [[ ! -f "$docker_dir/docker-entrypoint.sh" ]]; then
        log::error "Required files missing in $docker_dir"
        log::info "Please ensure Dockerfile and docker-entrypoint.sh exist"
        return 1
    fi
    
    # Build image
    if docker build -t "$N8N_CUSTOM_IMAGE" "$docker_dir"; then
        log::success "Custom n8n image built successfully"
        return 0
    else
        log::error "Failed to build custom image"
        return 1
    fi
}

#######################################
# Build n8n Docker command
#######################################
n8n::build_docker_command() {
    local webhook_url="$1"
    local auth_password="$2"
    
    local docker_cmd="docker run -d"
    docker_cmd+=" --name $N8N_CONTAINER_NAME"
    docker_cmd+=" --network $N8N_NETWORK_NAME"
    docker_cmd+=" -p ${N8N_PORT}:5678"
    docker_cmd+=" -v ${N8N_DATA_DIR}:/home/node/.n8n"
    docker_cmd+=" --restart unless-stopped"
    
    # Add volume mounts for host access
    # Mount only if directories exist on host
    if [[ -d /usr/bin ]]; then
        docker_cmd+=" -v /usr/bin:/host/usr/bin:ro"
    fi
    if [[ -d /bin ]]; then
        docker_cmd+=" -v /bin:/host/bin:ro"
    fi
    if [[ -d /usr/local/bin ]]; then
        docker_cmd+=" -v /usr/local/bin:/host/usr/local/bin:ro"
    fi
    
    # User home and workspace
    docker_cmd+=" -v $HOME:/host/home:rw"
    docker_cmd+=" -v $HOME/Vrooli:/workspace:rw"
    
    # Docker socket for container control
    if [[ -S /var/run/docker.sock ]]; then
        docker_cmd+=" -v /var/run/docker.sock:/var/run/docker.sock:rw"
    fi
    
    # Environment variables
    docker_cmd+=" -e GENERIC_TIMEZONE=$(timedatectl show -p Timezone --value 2>/dev/null || echo 'UTC')"
    docker_cmd+=" -e TZ=$(timedatectl show -p Timezone --value 2>/dev/null || echo 'UTC')"
    
    # Webhook configuration
    if [[ -n "$webhook_url" ]]; then
        docker_cmd+=" -e WEBHOOK_URL=$webhook_url"
        docker_cmd+=" -e N8N_PROTOCOL=https"
        docker_cmd+=" -e N8N_HOST=$(echo "$webhook_url" | sed 's|https://||' | sed 's|/.*||')"
    fi
    
    # Basic authentication
    if [[ "$BASIC_AUTH" == "yes" ]]; then
        docker_cmd+=" -e N8N_BASIC_AUTH_ACTIVE=true"
        docker_cmd+=" -e N8N_BASIC_AUTH_USER=$AUTH_USERNAME"
        docker_cmd+=" -e N8N_BASIC_AUTH_PASSWORD=$auth_password"
    fi
    
    # Database configuration
    if [[ "$DATABASE_TYPE" == "postgres" ]]; then
        docker_cmd+=" -e DB_TYPE=postgresdb"
        docker_cmd+=" -e DB_POSTGRESDB_HOST=$N8N_DB_CONTAINER_NAME"
        docker_cmd+=" -e DB_POSTGRESDB_PORT=5432"
        docker_cmd+=" -e DB_POSTGRESDB_DATABASE=n8n"
        docker_cmd+=" -e DB_POSTGRESDB_USER=n8n"
        docker_cmd+=" -e DB_POSTGRESDB_PASSWORD=$N8N_DB_PASSWORD"
    else
        docker_cmd+=" -e DB_TYPE=sqlite"
        docker_cmd+=" -e DB_SQLITE_VACUUM_ON_STARTUP=true"
    fi
    
    # Additional settings
    docker_cmd+=" -e N8N_DIAGNOSTICS_ENABLED=false"
    docker_cmd+=" -e N8N_TEMPLATES_ENABLED=true"
    docker_cmd+=" -e N8N_PERSONALIZATION_ENABLED=false"
    docker_cmd+=" -e N8N_RUNNERS_ENABLED=true"
    docker_cmd+=" -e N8N_ENFORCE_SETTINGS_FILE_PERMISSIONS=true"
    docker_cmd+=" -e EXECUTIONS_DATA_SAVE_ON_ERROR=all"
    docker_cmd+=" -e EXECUTIONS_DATA_SAVE_ON_SUCCESS=all"
    docker_cmd+=" -e EXECUTIONS_DATA_SAVE_ON_PROGRESS=true"
    docker_cmd+=" -e EXECUTIONS_DATA_SAVE_MANUAL_EXECUTIONS=true"
    
    # Use custom image if available
    local image_to_use="$N8N_IMAGE"
    if docker images --format "{{.Repository}}:{{.Tag}}" | grep -q "^${N8N_CUSTOM_IMAGE}$"; then
        image_to_use="$N8N_CUSTOM_IMAGE"
        log::info "Using custom n8n image: $N8N_CUSTOM_IMAGE"
    fi
    
    # Image and command
    docker_cmd+=" $image_to_use"
    
    # Add tunnel flag if enabled (development only)
    if [[ "$TUNNEL_ENABLED" == "yes" ]]; then
        docker_cmd+=" n8n start --tunnel"
        log::warn "⚠️  Tunnel mode enabled - for development only!"
    fi
    
    echo "$docker_cmd"
}

#######################################
# Start n8n container
#######################################
n8n::start_container() {
    local webhook_url="$1"
    local auth_password="$2"
    
    log::info "Starting n8n container..."
    
    # Build and execute Docker command
    local docker_cmd
    docker_cmd=$(n8n::build_docker_command "$webhook_url" "$auth_password")
    
    if eval "$docker_cmd" >/dev/null 2>&1; then
        log::success "n8n container started"
        
        # Add rollback action
        resources::add_rollback_action \
            "Stop and remove n8n container" \
            "docker stop $N8N_CONTAINER_NAME 2>/dev/null; docker rm $N8N_CONTAINER_NAME 2>/dev/null || true" \
            25
        
        return 0
    else
        log::error "Failed to start n8n container"
        return 1
    fi
}

#######################################
# Stop n8n
#######################################
n8n::stop() {
    if ! n8n::is_running; then
        log::info "n8n is not running"
        return 0
    fi
    
    log::info "Stopping n8n..."
    
    # Stop n8n container
    if docker stop "$N8N_CONTAINER_NAME" >/dev/null 2>&1; then
        log::success "n8n stopped"
    else
        log::error "Failed to stop n8n"
        return 1
    fi
    
    # Stop PostgreSQL if used
    if [[ "$DATABASE_TYPE" == "postgres" ]] && docker ps --format '{{.Names}}' | grep -q "^${N8N_DB_CONTAINER_NAME}$"; then
        log::info "Stopping PostgreSQL..."
        docker stop "$N8N_DB_CONTAINER_NAME" >/dev/null 2>&1
    fi
}

#######################################
# Start n8n
#######################################
n8n::start() {
    if n8n::is_running && [[ "$FORCE" != "yes" ]]; then
        log::info "n8n is already running on port $N8N_PORT"
        return 0
    fi
    
    log::info "Starting n8n..."
    
    # Check if container exists
    if ! n8n::container_exists; then
        log::error "n8n container does not exist. Run install first."
        return 1
    fi
    
    # Start PostgreSQL first if needed
    if docker ps -a --format '{{.Names}}' | grep -q "^${N8N_DB_CONTAINER_NAME}$"; then
        if ! docker ps --format '{{.Names}}' | grep -q "^${N8N_DB_CONTAINER_NAME}$"; then
            log::info "Starting PostgreSQL..."
            docker start "$N8N_DB_CONTAINER_NAME" >/dev/null 2>&1
            sleep 3
        fi
    fi
    
    # Start n8n container
    if docker start "$N8N_CONTAINER_NAME" >/dev/null 2>&1; then
        log::success "n8n started"
        
        # Wait for service to be ready
        if resources::wait_for_service "n8n" "$N8N_PORT" 30; then
            log::success "✅ n8n is running on port $N8N_PORT"
            log::info "Access n8n at: $N8N_BASE_URL"
        else
            log::warn "n8n started but may not be fully ready yet"
        fi
    else
        log::error "Failed to start n8n"
        return 1
    fi
}

#######################################
# Restart n8n
#######################################
n8n::restart() {
    log::info "Restarting n8n..."
    n8n::stop
    sleep 2
    n8n::start
}

#######################################
# Show n8n logs
#######################################
n8n::logs() {
    if ! n8n::container_exists; then
        log::error "n8n container does not exist"
        return 1
    fi
    
    log::info "Showing n8n logs (Ctrl+C to exit)..."
    docker logs -f "$N8N_CONTAINER_NAME"
}

#######################################
# Remove n8n container
#######################################
n8n::remove_container() {
    if n8n::container_exists; then
        log::info "Removing n8n container..."
        docker rm -f "$N8N_CONTAINER_NAME" >/dev/null 2>&1 || true
    fi
}

#######################################
# Remove PostgreSQL container
#######################################
n8n::remove_postgres_container() {
    if docker ps -a --format '{{.Names}}' | grep -q "^${N8N_DB_CONTAINER_NAME}$"; then
        log::info "Removing PostgreSQL container..."
        docker rm -f "$N8N_DB_CONTAINER_NAME" >/dev/null 2>&1 || true
    fi
}

#######################################
# Remove Docker network
#######################################
n8n::remove_network() {
    if docker network ls | grep -q "$N8N_NETWORK_NAME"; then
        log::info "Removing Docker network..."
        docker network rm "$N8N_NETWORK_NAME" >/dev/null 2>&1 || true
    fi
}

#######################################
# Pull n8n image
#######################################
n8n::pull_image() {
    log::info "Pulling n8n Docker image..."
    
    if docker pull "$N8N_IMAGE"; then
        log::success "n8n image pulled successfully"
        return 0
    else
        log::error "Failed to pull n8n image"
        return 1
    fi
}