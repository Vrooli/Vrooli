#!/usr/bin/env bash
# QuestDB Docker Management Functions

QUESTDB_LIB_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)

#######################################
# Check if QuestDB container is running
# Returns:
#   0 if running, 1 otherwise
#######################################
questdb::docker::is_running() {
    docker ps --format '{{.Names}}' | grep -q "^${QUESTDB_CONTAINER_NAME}$"
}

#######################################
# Check if QuestDB container exists
# Returns:
#   0 if exists, 1 otherwise
#######################################
questdb::docker::container_exists() {
    docker ps -a --format '{{.Names}}' | grep -q "^${QUESTDB_CONTAINER_NAME}$"
}

#######################################
# Create Docker network if not exists
# Returns:
#   0 on success, 1 on failure
#######################################
questdb::docker::create_network() {
    if ! docker network ls --format '{{.Name}}' | grep -q "^${QUESTDB_NETWORK_NAME}$"; then
        echo_info "${QUESTDB_INSTALL_MESSAGES["creating_network"]}"
        docker network create "${QUESTDB_NETWORK_NAME}" || return 1
    fi
    return 0
}

#######################################
# Start QuestDB container
# Returns:
#   0 on success, 1 on failure
#######################################
questdb::docker::start() {
    echo_info "${QUESTDB_STATUS_MESSAGES["starting"]}"
    
    # Check if already running
    if questdb::docker::is_running; then
        echo_info "QuestDB is already running"
        return 0
    fi
    
    # Create network if needed
    questdb::docker::create_network || return 1
    
    # Start container
    if questdb::docker::container_exists; then
        docker start "${QUESTDB_CONTAINER_NAME}" || return 1
    else
        # Run new container
        cd "${QUESTDB_LIB_DIR}/../docker" || return 1
        docker-compose up -d || return 1
        cd - > /dev/null || return 1
    fi
    
    # Wait for container to be healthy
    questdb::docker::wait_healthy || return 1
    
    echo_success "${QUESTDB_STATUS_MESSAGES["ready"]}"
    return 0
}

#######################################
# Stop QuestDB container
# Returns:
#   0 on success, 1 on failure
#######################################
questdb::docker::stop() {
    echo_info "${QUESTDB_STATUS_MESSAGES["stopping"]}"
    
    if ! questdb::docker::is_running; then
        echo_info "QuestDB is not running"
        return 0
    fi
    
    docker stop "${QUESTDB_CONTAINER_NAME}" || return 1
    echo_success "QuestDB stopped"
    return 0
}

#######################################
# Restart QuestDB container
# Returns:
#   0 on success, 1 on failure
#######################################
questdb::docker::restart() {
    questdb::docker::stop || return 1
    sleep 2
    questdb::docker::start || return 1
}

#######################################
# Wait for QuestDB to be healthy
# Returns:
#   0 if healthy, 1 on timeout
#######################################
questdb::docker::wait_healthy() {
    echo_info "${QUESTDB_STATUS_MESSAGES["waiting"]}"
    
    local max_wait="${QUESTDB_STARTUP_MAX_WAIT}"
    local interval="${QUESTDB_STARTUP_WAIT_INTERVAL}"
    local elapsed=0
    
    while (( elapsed < max_wait )); do
        if questdb::docker::health_check; then
            return 0
        fi
        
        sleep "${interval}"
        ((elapsed += interval))
        echo -n "."
    done
    
    echo ""
    echo_error "QuestDB failed to become healthy after ${max_wait} seconds"
    return 1
}

#######################################
# Check QuestDB health
# Returns:
#   0 if healthy, 1 otherwise
#######################################
questdb::docker::health_check() {
    if ! questdb::docker::is_running; then
        return 1
    fi
    
    # Check HTTP endpoint
    local response
    response=$(curl -s -o /dev/null -w "%{http_code}" "${QUESTDB_BASE_URL}/status" 2>/dev/null || echo "000")
    
    [[ "$response" == "200" ]]
}

#######################################
# View QuestDB logs
# Arguments:
#   $1 - Follow logs (true/false)
# Returns:
#   0 on success, 1 on failure
#######################################
questdb::docker::logs() {
    local follow="${1:-false}"
    
    if ! questdb::docker::container_exists; then
        echo_error "QuestDB container does not exist"
        return 1
    fi
    
    if [[ "$follow" == "true" ]]; then
        docker logs -f "${QUESTDB_CONTAINER_NAME}"
    else
        docker logs --tail 100 "${QUESTDB_CONTAINER_NAME}"
    fi
}

#######################################
# Get container statistics
# Returns:
#   JSON with container stats
#######################################
questdb::docker::stats() {
    if ! questdb::docker::is_running; then
        echo "{}"
        return 1
    fi
    
    docker stats "${QUESTDB_CONTAINER_NAME}" --no-stream --format "json" | jq -r '.'
}

#######################################
# Execute command in container
# Arguments:
#   $@ - Command to execute
# Returns:
#   Command exit code
#######################################
questdb::docker::exec() {
    if ! questdb::docker::is_running; then
        echo_error "QuestDB is not running"
        return 1
    fi
    
    docker exec "${QUESTDB_CONTAINER_NAME}" "$@"
}

#######################################
# Export Docker functions
#######################################
export -f questdb::docker::is_running
export -f questdb::docker::container_exists
export -f questdb::docker::create_network
export -f questdb::docker::start
export -f questdb::docker::stop
export -f questdb::docker::restart
export -f questdb::docker::wait_healthy
export -f questdb::docker::health_check
export -f questdb::docker::logs
export -f questdb::docker::stats
export -f questdb::docker::exec