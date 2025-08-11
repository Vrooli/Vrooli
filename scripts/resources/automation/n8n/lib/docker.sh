#!/usr/bin/env bash
# n8n Docker Management Functions
# All Docker-related operations for n8n containers

# Source specialized modules
SCRIPT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
# shellcheck disable=SC1091
source "${SCRIPT_DIR}/../../lib/docker-utils.sh" 2>/dev/null || true
# shellcheck disable=SC1091
source "${SCRIPT_DIR}/../../lib/wait-utils.sh" 2>/dev/null || true
# shellcheck disable=SC1091
source "${SCRIPT_DIR}/health.sh" 2>/dev/null || true
# shellcheck disable=SC1091
source "${SCRIPT_DIR}/recovery.sh" 2>/dev/null || true
# shellcheck disable=SC1091
source "${SCRIPT_DIR}/utils.sh" 2>/dev/null || true
# shellcheck disable=SC1091
source "${SCRIPT_DIR}/constants.sh" 2>/dev/null || true

#######################################
# Build custom n8n Docker image
#######################################
n8n::build_custom_image() {
    n8n::log_with_context "info" "build" "Building custom n8n image with host access..."
    # Ensure files exist
    local docker_dir="$SCRIPT_DIR/docker"
    if [[ ! -f "$docker_dir/Dockerfile" ]] || [[ ! -f "$docker_dir/docker-entrypoint.sh" ]]; then
        n8n::log_with_context "error" "build" "Required files missing in $docker_dir"
        n8n::log_with_context "info" "build" "Please ensure Dockerfile and docker-entrypoint.sh exist"
        return 1
    fi
    # Build image
    if docker build -t "$N8N_CUSTOM_IMAGE" "$docker_dir"; then
        n8n::log_with_context "success" "build" "Custom n8n image built successfully"
        return 0
    else
        n8n::log_with_context "error" "build" "Failed to build custom image"
        return 1
    fi
}

#######################################
# Build base Docker arguments
# Returns: Base docker run command with basic setup
#######################################
n8n::build_base_docker_args() {
    local docker_cmd="docker run -d"
    docker_cmd+=" --name $N8N_CONTAINER_NAME"
    docker_cmd+=" --network $N8N_NETWORK_NAME"
    docker_cmd+=" -p ${N8N_PORT}:5678"
    docker_cmd+=" -v ${N8N_DATA_DIR}:/home/node/.n8n"
    docker_cmd+=" --restart unless-stopped"
    echo "$docker_cmd"
}

#######################################
# Build volume mounts for host access
# Returns: Volume mount arguments
#######################################
n8n::build_volume_mounts() {
    local volume_args=""
    # Add volume mounts for host access
    # Mount only if directories exist on host
    if [[ -d /usr/bin ]]; then
        volume_args+=" -v /usr/bin:/host/usr/bin:ro"
    fi
    if [[ -d /bin ]]; then
        volume_args+=" -v /bin:/host/bin:ro"
    fi
    if [[ -d /usr/local/bin ]]; then
        volume_args+=" -v /usr/local/bin:/host/usr/local/bin:ro"
    fi
    # User home and workspace
    volume_args+=" -v $HOME:/host/home:rw"
    volume_args+=" -v ${var_ROOT_DIR}:/workspace:rw"
    # Docker socket for container control
    if [[ -S /var/run/docker.sock ]]; then
        volume_args+=" -v /var/run/docker.sock:/var/run/docker.sock:rw"
    fi
    echo "$volume_args"
}

#######################################
# Build environment variables
# Args: $1 - webhook_url, $2 - auth_password
# Returns: Environment variable arguments
#######################################
n8n::build_environment_vars() {
    local webhook_url="$1"
    local auth_password="$2"
    local env_args=""
    # Timezone configuration
    env_args+=" -e GENERIC_TIMEZONE=$(timedatectl show -p Timezone --value 2>/dev/null || echo 'UTC')"
    env_args+=" -e TZ=$(timedatectl show -p Timezone --value 2>/dev/null || echo 'UTC')"
    # Webhook configuration
    if [[ -n "$webhook_url" ]]; then
        env_args+=" -e WEBHOOK_URL=$webhook_url"
        # Extract protocol from webhook URL
        local protocol=$(echo "$webhook_url" | sed 's|://.*||')
        env_args+=" -e N8N_PROTOCOL=$protocol"
        # Extract host by removing protocol and path
        env_args+=" -e N8N_HOST=$(echo "$webhook_url" | sed 's|.*://||' | sed 's|/.*||')"
    fi
    # Basic authentication
    if [[ "$BASIC_AUTH" == "yes" ]]; then
        env_args+=" -e N8N_BASIC_AUTH_ACTIVE=true"
        env_args+=" -e N8N_BASIC_AUTH_USER=$AUTH_USERNAME"
        env_args+=" -e N8N_BASIC_AUTH_PASSWORD=$auth_password"
    fi
    # Database configuration
    if [[ "$DATABASE_TYPE" == "postgres" ]]; then
        env_args+=" -e DB_TYPE=postgresdb"
        env_args+=" -e DB_POSTGRESDB_HOST=$N8N_DB_CONTAINER_NAME"
        env_args+=" -e DB_POSTGRESDB_PORT=5432"
        env_args+=" -e DB_POSTGRESDB_DATABASE=n8n"
        env_args+=" -e DB_POSTGRESDB_USER=n8n"
        env_args+=" -e DB_POSTGRESDB_PASSWORD=$N8N_DB_PASSWORD"
    else
        env_args+=" -e DB_TYPE=sqlite"
        env_args+=" -e DB_SQLITE_VACUUM_ON_STARTUP=true"
    fi
    # Additional n8n settings
    env_args+=" -e N8N_DIAGNOSTICS_ENABLED=false"
    env_args+=" -e N8N_TEMPLATES_ENABLED=true"
    env_args+=" -e N8N_PERSONALIZATION_ENABLED=false"
    env_args+=" -e N8N_RUNNERS_ENABLED=true"
    env_args+=" -e N8N_ENFORCE_SETTINGS_FILE_PERMISSIONS=true"
    env_args+=" -e EXECUTIONS_DATA_SAVE_ON_ERROR=all"
    env_args+=" -e EXECUTIONS_DATA_SAVE_ON_SUCCESS=all"
    env_args+=" -e EXECUTIONS_DATA_SAVE_ON_PROGRESS=true"
    env_args+=" -e EXECUTIONS_DATA_SAVE_MANUAL_EXECUTIONS=true"
    env_args+=" -e N8N_PUBLIC_API_DISABLED=false"
    echo "$env_args"
}

#######################################
# Select appropriate Docker image
# Returns: Image name to use
#######################################
n8n::select_docker_image() {
    local image_to_use="$N8N_IMAGE"
    # Use custom image if available
    if docker images --format "{{.Repository}}:{{.Tag}}" | grep -q "^${N8N_CUSTOM_IMAGE}$"; then
        image_to_use="$N8N_CUSTOM_IMAGE"
        n8n::log_with_context "info" "docker" "Using custom n8n image: $N8N_CUSTOM_IMAGE"
    fi
    echo "$image_to_use"
}

#######################################
# Add tunnel configuration if enabled
# Returns: Tunnel arguments or empty string
#######################################
n8n::add_tunnel_config() {
    local tunnel_args=""
    # Add tunnel flag if enabled (development only)
    if [[ "$TUNNEL_ENABLED" == "yes" ]]; then
        tunnel_args+=" --tunnel"
        n8n::log_with_context "warn" "docker" "Tunnel mode enabled - for development only!"
    fi
    echo "$tunnel_args"
}

#######################################
# Build n8n Docker command
# REFACTORED: Main orchestrator function (was 130 lines, now 20 lines)
# Args: $1 - webhook_url, $2 - auth_password
# Returns: Complete docker run command
#######################################
n8n::build_docker_command() {
    local webhook_url="$1"
    local auth_password="$2"
    # Build command components
    local docker_cmd
    docker_cmd="$(n8n::build_base_docker_args)"
    docker_cmd+=" $(n8n::build_volume_mounts)"
    docker_cmd+=" $(n8n::build_environment_vars "$webhook_url" "$auth_password")"
    docker_cmd+=" $(n8n::select_docker_image)"
    docker_cmd+=" $(n8n::add_tunnel_config)"
    echo "$docker_cmd"
}

#######################################
# Start n8n container
#######################################
n8n::start_container() {
    local webhook_url="$1"
    local auth_password="$2"
    n8n::log_with_context "info" "docker" "Starting n8n container..."
    # Build and execute Docker command
    local docker_cmd
    docker_cmd=$(n8n::build_docker_command "$webhook_url" "$auth_password")
    if eval "$docker_cmd" >/dev/null 2>&1; then
        n8n::log_with_context "success" "docker" "n8n container started"
        # Add rollback action
        resources::add_rollback_action \
            "Stop and remove n8n container" \
            "docker stop $N8N_CONTAINER_NAME 2>/dev/null; docker rm $N8N_CONTAINER_NAME 2>/dev/null || true" \
            25
        return 0
    else
        n8n::log_with_context "error" "docker" "Failed to start n8n container"
        return 1
    fi
}

#######################################
# Stop n8n
#######################################
n8n::stop() {
    if ! n8n::container_running; then
        n8n::log_with_context "info" "docker" "$N8N_ERR_CONTAINER_NOT_RUNNING"
        return 0
    fi
    n8n::log_with_context "info" "docker" "Stopping n8n..."
    # Stop n8n container
    if docker stop "$N8N_CONTAINER_NAME" >/dev/null 2>&1; then
        n8n::log_with_context "success" "docker" "n8n stopped"
    else
        n8n::log_with_context "error" "docker" "Failed to stop n8n"
        return 1
    fi
    # Stop PostgreSQL if used
    if [[ "$DATABASE_TYPE" == "postgres" ]] && n8n::postgres_running; then
        n8n::log_with_context "info" "docker" "Stopping PostgreSQL..."
        docker stop "$N8N_DB_CONTAINER_NAME" >/dev/null 2>&1
    fi
}

#######################################
# Handle existing running instance
# Returns: 0 if should continue, 1 if should stop
#######################################
n8n::handle_existing_instance() {
    if n8n::container_running && [[ "$FORCE" != "yes" ]]; then
        n8n::log_with_context "info" "docker" "$N8N_ERR_ALREADY_RUNNING on port $N8N_PORT"
        # Still run health check to ensure it's working properly
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
    # Start PostgreSQL first if needed
    if n8n::postgres_exists; then
        if ! n8n::postgres_running; then
            log::info "Starting PostgreSQL..."
            docker start "$N8N_DB_CONTAINER_NAME" >/dev/null 2>&1
            sleep 3
        fi
    fi
}

#######################################
# Health check with corruption detection and recovery
# Returns: 0 if healthy, 1 if unhealthy, 2 if recovery attempted
#######################################
n8n::health_check_with_recovery() {
    if n8n::is_healthy; then
        return 0
    fi
    
    # Check for corruption during startup using shared utility
    local recent_logs
    recent_logs=$(docker logs "$N8N_CONTAINER_NAME" --tail 10 2>&1 || echo "")
    if n8n::detect_and_suggest_fix "$recent_logs" >/dev/null; then
        if echo "$recent_logs" | grep -qi "SQLITE_READONLY\|database.*locked"; then
            log::error "Database corruption detected during startup"
            log::info "Stopping container and attempting recovery..."
            docker stop "$N8N_CONTAINER_NAME" >/dev/null 2>&1 || true
            if n8n::auto_recover; then
                log::info "Recovery completed, restarting..."
                docker start "$N8N_CONTAINER_NAME" >/dev/null 2>&1
                return 2  # Recovery attempted
            else
                log::error "Recovery failed"
                return 1
            fi
        fi
    fi
    return 1
}

#######################################
# Start container and wait for readiness with recovery
# Returns: 0 if successful, 1 if failed
#######################################
n8n::start_container_and_wait() {
    # Start n8n container
    if ! docker start "$N8N_CONTAINER_NAME" >/dev/null 2>&1; then
        n8n::log_with_context "error" "docker" "Failed to start n8n container"
        return 1
    fi
    log::success "n8n container started"
    
    # Use standardized wait utility with enhanced health checks
    if ! wait::for_condition \
        "n8n::health_check_with_recovery" \
        120 \
        "n8n to be ready with auto-recovery"; then
        log::error "n8n failed to become ready within timeout"
        log::info "Check logs with: $0 --action logs"
        return 1
    fi
    
    log::success "✅ n8n is healthy and ready on port $N8N_PORT"
    log::info "Access n8n at: $N8N_BASE_URL"
    
    # Final comprehensive health check
    if n8n::comprehensive_health_check; then
        log::success "✅ All systems operational"
    else
        log::warn "⚠️  Minor health issues detected but service is running"
    fi
    return 0
}

#######################################
# Start n8n with enhanced health checks and recovery
# REFACTORED: Main orchestrator function (was 100 lines, now 25 lines)
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
    # Start container and wait for readiness
    n8n::start_container_and_wait
}

#######################################
# Restart n8n with automatic recovery
#######################################
n8n::restart() {
    log::info "Restarting n8n with health checks..."
    # Run pre-restart diagnostics
    if ! n8n::detect_filesystem_corruption; then
        log::warn "Filesystem corruption detected before restart"
    fi
    n8n::stop
    sleep 2
    # Enable automatic recovery during restart
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
# Remove n8n container
#######################################
n8n::remove_container() {
    if n8n::container_exists_any; then
        log::info "Removing n8n container..."
        docker rm -f "$N8N_CONTAINER_NAME" >/dev/null 2>&1 || true
    fi
}

#######################################
# Remove PostgreSQL container
#######################################
n8n::remove_postgres_container() {
    if n8n::postgres_exists; then
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
