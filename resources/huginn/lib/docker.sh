#!/usr/bin/env bash
# Huginn Docker Management - Simplified with docker-resource-utils.sh
# Container lifecycle and Docker operations

# Source var.sh to get proper directory variables
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*/../../.." && builtin pwd)}"
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/var.sh"

# Source shared libraries
# shellcheck disable=SC1091
source "${var_LIB_SERVICE_DIR}/secrets.sh"
# shellcheck disable=SC1091
source "${var_SCRIPTS_RESOURCES_LIB_DIR}/docker-resource-utils.sh"

#######################################
# Start Huginn containers
# Returns: 0 if successful, 1 otherwise
#######################################
huginn::start() {
    if ! docker::check_daemon; then
        return 1
    fi
    
    if huginn::is_running; then
        log::info "✅ Huginn is already running"
        return 0
    fi
    
    log::info "Starting Huginn..."
    
    # Ensure network exists
    docker::create_network "$NETWORK_NAME"
    
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
    
    log::info "Stopping Huginn..."
    
    # Stop Huginn container first
    if huginn::container_exists; then
        log::info "Stopping Huginn container..."
        docker::stop_container "$CONTAINER_NAME"
    fi
    
    # Stop database container
    if huginn::db_container_exists; then
        log::info "Stopping database container..."
        docker::stop_container "$DB_CONTAINER_NAME"
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
    
    sleep ${HUGINN_RESTART_WAIT:-2}
    
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
    if docker::is_running "$DB_CONTAINER_NAME"; then
        log::info "Database container already running"
        return 0
    fi
    
    log::info "Starting PostgreSQL database..."
    
    # Remove existing stopped container
    if docker::container_exists "$DB_CONTAINER_NAME"; then
        docker::remove_container "$DB_CONTAINER_NAME" "true"
    fi
    
    # Pull image
    docker::pull_image "$POSTGRES_IMAGE"
    
    # Environment variables
    local env_vars=(
        "POSTGRES_DB=huginn"
        "POSTGRES_USER=huginn"
        "POSTGRES_PASSWORD=$DEFAULT_DB_PASSWORD"
        "POSTGRES_INITDB_ARGS=--encoding=UTF-8"
    )
    
    # Health check
    local health_cmd="pg_isready"
    
    # Use advanced creation
    if docker_resource::create_service_advanced \
        "$DB_CONTAINER_NAME" \
        "$POSTGRES_IMAGE" \
        "" \
        "$NETWORK_NAME" \
        "$DB_VOLUME_NAME:/var/lib/postgresql/data" \
        "env_vars" \
        "" \
        "$health_cmd" \
        ""; then
        return 0
    else
        log::error "Failed to start Huginn database"
        return 1
    fi
}

#######################################
# Start Huginn application container
# Returns: 0 if successful, 1 otherwise
#######################################
huginn::start_huginn_container() {
    if docker::is_running "$CONTAINER_NAME"; then
        log::info "Huginn container already running"
        return 0
    fi
    
    log::info "Starting Huginn application..."
    
    # Remove existing stopped container
    if docker::container_exists "$CONTAINER_NAME"; then
        docker::remove_container "$CONTAINER_NAME" "true"
    fi
    
    # Ensure data directories exist
    if ! huginn::ensure_data_directories; then
        return 1
    fi
    
    # Pull image
    docker::pull_image "$HUGINN_IMAGE"
    
    # Environment variables
    local env_vars=(
        "DATABASE_URL=postgres://huginn:${DEFAULT_DB_PASSWORD}@${DB_CONTAINER_NAME}:5432/huginn"
        "HUGINN_DATABASE_NAME=huginn"
        "HUGINN_DATABASE_USERNAME=huginn"
        "HUGINN_DATABASE_PASSWORD=$DEFAULT_DB_PASSWORD"
        "HUGINN_DATABASE_HOST=$DB_CONTAINER_NAME"
        "HUGINN_DATABASE_PORT=5432"
        "PORT=3000"
        "RAILS_ENV=production"
        "SEED_USERNAME=$DEFAULT_ADMIN_USERNAME"
        "SEED_EMAIL=$DEFAULT_ADMIN_EMAIL"
        "SEED_PASSWORD=$DEFAULT_ADMIN_PASSWORD"
        "APP_SECRET_TOKEN=$(openssl rand -hex 64)"
        "INVITATION_CODE="
        "REQUIRE_CONFIRMED_EMAIL=false"
    )
    
    # Volumes
    local volumes="$VOLUME_NAME:/var/lib/huginn $HUGINN_UPLOADS_DIR:/app/public/uploads"
    
    # Health check
    local health_cmd="curl -f http://localhost:3000/ || exit 1"
    
    # Use advanced creation
    if docker_resource::create_service_advanced \
        "$CONTAINER_NAME" \
        "$HUGINN_IMAGE" \
        "${HUGINN_PORT}:3000" \
        "$NETWORK_NAME" \
        "$volumes" \
        "env_vars" \
        "" \
        "$health_cmd" \
        ""; then
        return 0
    else
        huginn::show_rails_error
        return 1
    fi
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
        sleep ${HUGINN_DB_WAIT_INTERVAL:-2}
        
        if [[ $((attempt % 5)) -eq 0 ]]; then
            log::info "   Still waiting for database... ($attempt/$max_attempts)"
        fi
    done
    
    log::error "Failed to start Huginn database"
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
    
    if ! docker::container_exists "$container_name"; then
        log::error "Container '$container_name' not found"
        return 1
    fi
    
    log::info "📋 Showing logs for $container_name (last $lines lines):"
    echo "----------------------------------------"
    
    if [[ "$follow" == "true" ]]; then
        docker logs --tail "$lines" --follow "$container_name"
    else
        docker::get_logs "$container_name" "$lines"
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
        if docker::is_running "$container"; then
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
        if docker::container_exists "$container"; then
            log::info "Removing container: $container"
            docker::remove_container "$container" "true"
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
    
    # Remove network only if empty
    docker::cleanup_network_if_empty "$NETWORK_NAME"
    
    return 0
}

#######################################
# Start Huginn using docker-compose
# Returns: 0 if successful, 1 otherwise
#######################################
huginn::compose_up() {
    local compose_file="${_HUGINN_DOCKER_DIR}/../docker/docker-compose.yml"
    docker_resource::compose_up "$compose_file"
}

#######################################
# Stop Huginn using docker-compose
# Arguments: $1 - remove volumes (true/false)
# Returns: 0 if successful, 1 otherwise
#######################################
huginn::compose_down() {
    local remove_volumes="${1:-false}"
    local compose_file="${_HUGINN_DOCKER_DIR}/../docker/docker-compose.yml"
    docker_resource::compose_down "$compose_file" "$remove_volumes"
}

#######################################
# Restart Huginn using docker-compose
# Returns: 0 if successful, 1 otherwise
#######################################
huginn::compose_restart() {
    local compose_file="${_HUGINN_DOCKER_DIR}/../docker/docker-compose.yml"
    docker_resource::compose_restart "$compose_file"
}