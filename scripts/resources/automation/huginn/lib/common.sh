#!/usr/bin/env bash
# Huginn Common Utility Functions
# Shared utilities used across all modules

#######################################
# Check if Docker is available and running
# Returns: 0 if available, 1 otherwise
#######################################
huginn::check_docker() {
    if ! system::is_command "docker"; then
        huginn::show_docker_error
        return 1
    fi
    
    # Check if Docker daemon is running
    if ! docker info >/dev/null 2>&1; then
        huginn::show_docker_error
        return 1
    fi
    
    # Check if user has permissions
    if ! docker ps >/dev/null 2>&1; then
        huginn::show_docker_error
        return 1
    fi
    
    return 0
}

#######################################
# Check if Huginn container exists
# Returns: 0 if exists, 1 otherwise
#######################################
huginn::container_exists() {
    docker ps -a --format '{{.Names}}' | grep -q "^${CONTAINER_NAME}$"
}

#######################################
# Check if Huginn database container exists
# Returns: 0 if exists, 1 otherwise
#######################################
huginn::db_container_exists() {
    docker ps -a --format '{{.Names}}' | grep -q "^${DB_CONTAINER_NAME}$"
}

#######################################
# Check if Huginn is installed
# Returns: 0 if installed, 1 otherwise
#######################################
huginn::is_installed() {
    huginn::container_exists && huginn::db_container_exists
}

#######################################
# Check if Huginn containers are running
# Returns: 0 if running, 1 otherwise
#######################################
huginn::is_running() {
    local huginn_state=$(docker container inspect -f '{{.State.Running}}' "$CONTAINER_NAME" 2>/dev/null || echo "false")
    local db_state=$(docker container inspect -f '{{.State.Running}}' "$DB_CONTAINER_NAME" 2>/dev/null || echo "false")
    [[ "$huginn_state" == "true" && "$db_state" == "true" ]]
}

#######################################
# Check if Huginn is healthy and responding
# Returns: 0 if healthy, 1 otherwise
#######################################
huginn::is_healthy() {
    if ! huginn::is_running; then
        return 1
    fi
    
    if system::is_command "curl"; then
        # Try health check endpoint first
        if curl -f -s --max-time "$HUGINN_HEALTH_CHECK_TIMEOUT" "$HUGINN_BASE_URL" >/dev/null 2>&1; then
            return 0
        fi
    fi
    return 1
}

#######################################
# Wait for Huginn to be ready
# Returns: 0 if ready, 1 if timeout
#######################################
huginn::wait_for_ready() {
    local max_attempts=${HUGINN_HEALTH_CHECK_MAX_ATTEMPTS:-30}
    local attempt=0
    
    huginn::show_waiting_message
    
    while [[ $attempt -lt $max_attempts ]]; do
        if huginn::is_healthy; then
            log::success "🤖 Huginn is ready!"
            return 0
        fi
        
        attempt=$((attempt + 1))
        sleep "$HUGINN_HEALTH_CHECK_INTERVAL"
        
        # Show progress
        if [[ $((attempt % 5)) -eq 0 ]]; then
            log::info "   Still waiting... ($attempt/$max_attempts)"
        fi
    done
    
    huginn::show_health_check_failed
    return 1
}

#######################################
# Get container logs
# Arguments:
#   $1 - container name (optional, defaults to main container)
#   $2 - number of lines (optional, defaults to 50)
#######################################
huginn::get_logs() {
    local container="${1:-$CONTAINER_NAME}"
    local lines="${2:-50}"
    
    if docker ps -a --format '{{.Names}}' | grep -q "^${container}$"; then
        docker logs --tail "$lines" "$container" 2>&1
    else
        log::error "Container '$container' not found"
        return 1
    fi
}

#######################################
# Execute Rails runner command in Huginn container
# Arguments:
#   $1 - Ruby code to execute
# Returns: 0 if successful, 1 otherwise
#######################################
huginn::rails_runner() {
    local ruby_code="$1"
    
    if ! huginn::is_running; then
        huginn::show_not_running
        return 1
    fi
    
    # Execute Ruby code via Rails runner with timeout
    # Use printf to avoid shell interpretation issues
    printf '%s\n' "$ruby_code" | timeout "$RAILS_RUNNER_TIMEOUT" docker exec -i "$CONTAINER_NAME" bundle exec rails runner - 2>/dev/null
}

#######################################
# Create Docker network if it doesn't exist
# Returns: 0 if successful, 1 otherwise
#######################################
huginn::ensure_network() {
    if ! docker network ls --format '{{.Name}}' | grep -q "^${NETWORK_NAME}$"; then
        log::info "Creating Docker network: $NETWORK_NAME"
        if ! docker network create "$NETWORK_NAME" >/dev/null 2>&1; then
            huginn::show_network_error
            return 1
        fi
    fi
    return 0
}

#######################################
# Create data directories
# Returns: 0 if successful, 1 otherwise
#######################################
huginn::ensure_data_directories() {
    local directories=(
        "$HUGINN_DATA_DIR"
        "$HUGINN_DB_DIR"
        "$HUGINN_UPLOADS_DIR"
    )
    
    for dir in "${directories[@]}"; do
        if [[ ! -d "$dir" ]]; then
            log::info "Creating data directory: $dir"
            mkdir -p "$dir" || {
                log::error "Failed to create directory: $dir"
                return 1
            }
        fi
    done
    
    return 0
}

#######################################
# Get system statistics
# Returns: JSON object with system stats
#######################################
huginn::get_system_stats() {
    huginn::rails_runner '
    stats = {
        users: User.count,
        agents: Agent.count,
        scenarios: Scenario.count,
        events: Event.count,
        links: Link.count,
        active_agents: Agent.where("last_check_at > ?", 1.day.ago).count,
        recent_events: Event.where("created_at > ?", 1.hour.ago).count
    }
    puts stats.to_json
    '
}

#######################################
# Check database connectivity
# Returns: 0 if connected, 1 otherwise
#######################################
huginn::check_database() {
    local result
    result=$(huginn::rails_runner 'puts "DB_OK" if ActiveRecord::Base.connection.active?' 2>/dev/null)
    [[ "$result" == "DB_OK" ]]
}

#######################################
# Get Huginn version information
# Returns: version string
#######################################
huginn::get_version() {
    huginn::rails_runner 'puts "Huginn #{Rails.application.class.module_parent_name} on Rails #{Rails.version}"' 2>/dev/null || echo "Unknown"
}

#######################################
# Validate agent ID format
# Arguments:
#   $1 - agent ID to validate
# Returns: 0 if valid, 1 otherwise
#######################################
huginn::validate_agent_id() {
    local agent_id="$1"
    [[ "$agent_id" =~ ^[0-9]+$ ]]
}

#######################################
# Validate scenario ID format  
# Arguments:
#   $1 - scenario ID to validate
# Returns: 0 if valid, 1 otherwise
#######################################
huginn::validate_scenario_id() {
    local scenario_id="$1"
    [[ "$scenario_id" =~ ^[0-9]+$ ]]
}

#######################################
# Clean up old containers and volumes
# Returns: 0 if successful, 1 otherwise
#######################################
huginn::cleanup_old_resources() {
    # Remove stopped containers
    if huginn::container_exists && ! huginn::is_running; then
        log::info "Removing stopped Huginn container..."
        docker rm "$CONTAINER_NAME" >/dev/null 2>&1 || true
    fi
    
    if huginn::db_container_exists && ! docker ps --format '{{.Names}}' | grep -q "^${DB_CONTAINER_NAME}$"; then
        log::info "Removing stopped database container..."
        docker rm "$DB_CONTAINER_NAME" >/dev/null 2>&1 || true
    fi
    
    return 0
}