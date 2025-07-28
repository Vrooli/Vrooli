#!/usr/bin/env bash
# Huginn Docker Management Functions
# Container lifecycle and Docker operations

#######################################
# Start Huginn containers
# Returns: 0 if successful, 1 otherwise
#######################################
huginn::start() {
    if ! huginn::check_docker; then
        return 1
    fi
    
    if huginn::is_running; then
        log::info "✅ Huginn is already running"
        return 0
    fi
    
    huginn::show_starting
    
    # Ensure network exists
    if ! huginn::ensure_network; then
        return 1
    fi
    
    # Start database container first
    if ! huginn::start_database; then
        return 1
    fi
    
    # Wait for database to be ready
    if ! huginn::wait_for_database; then
        return 1
    fi
    
    # Start Huginn container
    if ! huginn::start_huginn_container; then
        return 1
    fi
    
    # Wait for Huginn to be ready
    if ! huginn::wait_for_ready; then
        return 1
    fi
    
    log::success "✅ Huginn started successfully"
    return 0
}

#######################################
# Stop Huginn containers
# Returns: 0 if successful, 1 otherwise
#######################################
huginn::stop() {
    if ! huginn::is_running; then
        log::info "✅ Huginn is already stopped"
        return 0
    fi
    
    huginn::show_stopping
    
    # Stop Huginn container first
    if huginn::container_exists; then
        log::info "Stopping Huginn container..."
        docker stop "$CONTAINER_NAME" >/dev/null 2>&1 || true
    fi
    
    # Stop database container
    if huginn::db_container_exists; then
        log::info "Stopping database container..."
        docker stop "$DB_CONTAINER_NAME" >/dev/null 2>&1 || true
    fi
    
    log::success "✅ Huginn stopped successfully"
    return 0
}

#######################################
# Restart Huginn containers
# Returns: 0 if successful, 1 otherwise
#######################################
huginn::restart() {
    huginn::show_restarting
    
    if ! huginn::stop; then
        return 1
    fi
    
    sleep 2
    
    if ! huginn::start; then
        return 1
    fi
    
    log::success "✅ Huginn restarted successfully"
    return 0
}

#######################################
# Start PostgreSQL database container
# Returns: 0 if successful, 1 otherwise
#######################################
huginn::start_database() {
    if docker ps --format '{{.Names}}' | grep -q "^${DB_CONTAINER_NAME}$"; then
        log::info "Database container already running"
        return 0
    fi
    
    log::info "Starting PostgreSQL database..."
    
    # Remove existing stopped container
    if huginn::db_container_exists; then
        docker rm "$DB_CONTAINER_NAME" >/dev/null 2>&1 || true
    fi
    
    # Create database volume if it doesn't exist
    if ! docker volume ls --format '{{.Name}}' | grep -q "^${DB_VOLUME_NAME}$"; then
        docker volume create "$DB_VOLUME_NAME" >/dev/null 2>&1
    fi
    
    # Start database container
    docker run -d \
        --name "$DB_CONTAINER_NAME" \
        --network "$NETWORK_NAME" \
        --volume "$DB_VOLUME_NAME:/var/lib/postgresql/data" \
        --env "POSTGRES_DB=huginn" \
        --env "POSTGRES_USER=huginn" \
        --env "POSTGRES_PASSWORD=$DEFAULT_DB_PASSWORD" \
        --env "POSTGRES_INITDB_ARGS=--encoding=UTF-8" \
        --health-cmd="pg_isready -U huginn -d huginn" \
        --health-interval="$DOCKER_HEALTH_INTERVAL" \
        --health-timeout="$DOCKER_HEALTH_TIMEOUT" \
        --health-retries="$DOCKER_HEALTH_RETRIES" \
        --restart=unless-stopped \
        "$POSTGRES_IMAGE" >/dev/null 2>&1
    
    if [[ $? -ne 0 ]]; then
        huginn::show_database_error
        return 1
    fi
    
    return 0
}

#######################################
# Start Huginn application container
# Returns: 0 if successful, 1 otherwise
#######################################
huginn::start_huginn_container() {
    if docker ps --format '{{.Names}}' | grep -q "^${CONTAINER_NAME}$"; then
        log::info "Huginn container already running"
        return 0
    fi
    
    log::info "Starting Huginn application..."
    
    # Remove existing stopped container
    if huginn::container_exists; then
        docker rm "$CONTAINER_NAME" >/dev/null 2>&1 || true
    fi
    
    # Ensure data directories exist
    if ! huginn::ensure_data_directories; then
        return 1
    fi
    
    # Create Huginn volume if it doesn't exist  
    if ! docker volume ls --format '{{.Name}}' | grep -q "^${VOLUME_NAME}$"; then
        docker volume create "$VOLUME_NAME" >/dev/null 2>&1
    fi
    
    # Start Huginn container
    docker run -d \
        --name "$CONTAINER_NAME" \
        --network "$NETWORK_NAME" \
        --publish "${HUGINN_PORT}:3000" \
        --volume "$VOLUME_NAME:/var/lib/huginn" \
        --volume "$HUGINN_UPLOADS_DIR:/app/public/uploads" \
        --env "DATABASE_URL=postgres://huginn:${DEFAULT_DB_PASSWORD}@${DB_CONTAINER_NAME}:5432/huginn" \
        --env "HUGINN_DATABASE_NAME=huginn" \
        --env "HUGINN_DATABASE_USERNAME=huginn" \
        --env "HUGINN_DATABASE_PASSWORD=$DEFAULT_DB_PASSWORD" \
        --env "HUGINN_DATABASE_HOST=$DB_CONTAINER_NAME" \
        --env "HUGINN_DATABASE_PORT=5432" \
        --env "PORT=3000" \
        --env "RAILS_ENV=production" \
        --env "SEED_USERNAME=$DEFAULT_ADMIN_USERNAME" \
        --env "SEED_EMAIL=$DEFAULT_ADMIN_EMAIL" \
        --env "SEED_PASSWORD=$DEFAULT_ADMIN_PASSWORD" \
        --env "APP_SECRET_TOKEN=$(openssl rand -hex 64)" \
        --env "INVITATION_CODE=" \
        --env "REQUIRE_CONFIRMED_EMAIL=false" \
        --health-cmd="curl -f http://localhost:3000/ || exit 1" \
        --health-interval="$DOCKER_HEALTH_INTERVAL" \
        --health-timeout="$DOCKER_HEALTH_TIMEOUT" \
        --health-retries="$DOCKER_HEALTH_RETRIES" \
        --restart=unless-stopped \
        "$HUGINN_IMAGE" >/dev/null 2>&1
    
    if [[ $? -ne 0 ]]; then
        huginn::show_rails_error
        return 1
    fi
    
    return 0
}

#######################################
# Wait for database to be ready
# Returns: 0 if ready, 1 if timeout
#######################################
huginn::wait_for_database() {
    local max_attempts=30
    local attempt=0
    
    log::info "⏳ Waiting for database to be ready..."
    
    while [[ $attempt -lt $max_attempts ]]; do
        if docker exec "$DB_CONTAINER_NAME" pg_isready -U huginn -d huginn >/dev/null 2>&1; then
            log::success "✅ Database is ready!"
            return 0
        fi
        
        attempt=$((attempt + 1))
        sleep 2
        
        if [[ $((attempt % 5)) -eq 0 ]]; then
            log::info "   Still waiting for database... ($attempt/$max_attempts)"
        fi
    done
    
    huginn::show_database_error
    return 1
}

#######################################
# View container logs
# Arguments:
#   $1 - container type: "app" or "db" (optional, defaults to "app")
#   $2 - number of lines (optional, defaults to 50)
#   $3 - follow flag: "true" or "false" (optional, defaults to "false")
#######################################
huginn::view_logs() {
    local container_type="${1:-app}"
    local lines="${2:-50}"
    local follow="${3:-false}"
    
    local container_name
    case "$container_type" in
        "app"|"huginn")
            container_name="$CONTAINER_NAME"
            ;;
        "db"|"database"|"postgres")
            container_name="$DB_CONTAINER_NAME"
            ;;
        *)
            log::error "Invalid container type: $container_type"
            log::info "Valid options: app, db"
            return 1
            ;;
    esac
    
    if ! docker ps -a --format '{{.Names}}' | grep -q "^${container_name}$"; then
        log::error "Container '$container_name' not found"
        return 1
    fi
    
    log::info "📋 Showing logs for $container_name (last $lines lines):"
    echo "----------------------------------------"
    
    if [[ "$follow" == "true" ]]; then
        docker logs --tail "$lines" --follow "$container_name"
    else
        docker logs --tail "$lines" "$container_name"
    fi
}

#######################################
# Get container resource usage statistics
# Returns: formatted resource usage info
#######################################
huginn::get_resource_usage() {
    if ! huginn::is_running; then
        log::warn "Huginn is not running"
        return 1
    fi
    
    log::info "📊 Container Resource Usage:"
    echo
    
    # Get stats for both containers
    local containers=("$CONTAINER_NAME" "$DB_CONTAINER_NAME")
    
    for container in "${containers[@]}"; do
        if docker ps --format '{{.Names}}' | grep -q "^${container}$"; then
            echo "🐳 $container:"
            docker stats --no-stream --format "   CPU: {{.CPUPerc}}\tMemory: {{.MemUsage}}\tNetwork: {{.NetIO}}\tDisk: {{.BlockIO}}" "$container"
            echo
        fi
    done
}

#######################################
# Clean up all Huginn Docker resources
# Returns: 0 if successful, 1 otherwise
#######################################
huginn::cleanup_docker_resources() {
    log::info "🧹 Cleaning up Docker resources..."
    
    # Stop and remove containers
    for container in "$CONTAINER_NAME" "$DB_CONTAINER_NAME"; do
        if docker ps -a --format '{{.Names}}' | grep -q "^${container}$"; then
            log::info "Removing container: $container"
            docker stop "$container" >/dev/null 2>&1 || true
            docker rm "$container" >/dev/null 2>&1 || true
        fi
    done
    
    # Remove volumes if requested
    if [[ "${REMOVE_VOLUMES:-no}" == "yes" ]]; then
        for volume in "$VOLUME_NAME" "$DB_VOLUME_NAME"; do
            if docker volume ls --format '{{.Name}}' | grep -q "^${volume}$"; then
                log::info "Removing volume: $volume"
                docker volume rm "$volume" >/dev/null 2>&1 || true
            fi
        done
    fi
    
    # Remove network if no other containers are using it
    if docker network ls --format '{{.Name}}' | grep -q "^${NETWORK_NAME}$"; then
        local network_containers
        network_containers=$(docker network inspect "$NETWORK_NAME" --format '{{len .Containers}}' 2>/dev/null || echo "0")
        if [[ "$network_containers" == "0" ]]; then
            log::info "Removing network: $NETWORK_NAME"
            docker network rm "$NETWORK_NAME" >/dev/null 2>&1 || true
        fi
    fi
    
    return 0
}