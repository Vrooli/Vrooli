#!/usr/bin/env bash
################################################################################
# AutoGPT Core Library - v2.0 Contract Compliant
# Core functionality for AutoGPT resource management
################################################################################

# Ensure common.sh functions are available
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${SCRIPT_DIR}/common.sh"

################################################################################
# Lifecycle Management Functions
################################################################################

# Install AutoGPT and dependencies
autogpt::core::install() {
    log::info "Installing AutoGPT..."
    
    # Create required directories
    local dirs=(
        "${AUTOGPT_DATA_DIR}"
        "${AUTOGPT_AGENTS_DIR}"
        "${AUTOGPT_TOOLS_DIR}"
        "${AUTOGPT_WORKSPACE_DIR}"
        "${AUTOGPT_LOGS_DIR}"
    )
    
    for dir in "${dirs[@]}"; do
        if ! mkdir -p "${dir}"; then
            log::error "Failed to create directory: ${dir}"
            return 1
        fi
    done
    
    # Pull Docker image
    log::info "Pulling AutoGPT Docker image..."
    if ! docker pull "${AUTOGPT_IMAGE}"; then
        log::error "Failed to pull AutoGPT image"
        return 1
    fi
    
    # Create default configuration
    autogpt::core::create_default_config
    
    log::success "AutoGPT installed successfully"
    return 0
}

# Start AutoGPT service
autogpt::core::start() {
    local wait_flag="${1:-}"
    
    if autogpt_container_running; then
        log::warning "AutoGPT is already running"
        return 0
    fi
    
    log::info "Starting AutoGPT..."
    
    # Prepare environment variables
    local env_vars=(
        "-e AUTOGPT_API_PORT=${AUTOGPT_PORT_API}"
        "-e AUTOGPT_AI_PROVIDER=${AUTOGPT_AI_PROVIDER}"
        "-e AUTOGPT_MODEL=${AUTOGPT_MODEL}"
        "-e AUTOGPT_MAX_ITERATIONS=${AUTOGPT_MAX_ITERATIONS}"
        "-e AUTOGPT_MEMORY_BACKEND=${AUTOGPT_MEMORY_BACKEND}"
        "-e AUTOGPT_LOG_LEVEL=${AUTOGPT_LOG_LEVEL}"
    )
    
    # Add Redis configuration if using Redis backend
    if [[ "${AUTOGPT_MEMORY_BACKEND}" == "redis" ]]; then
        env_vars+=(
            "-e REDIS_HOST=${AUTOGPT_REDIS_HOST}"
            "-e REDIS_PORT=${AUTOGPT_REDIS_PORT}"
        )
    fi
    
    # Add PostgreSQL configuration if using PostgreSQL backend
    if [[ "${AUTOGPT_MEMORY_BACKEND}" == "postgres" ]]; then
        env_vars+=(
            "-e POSTGRES_HOST=${AUTOGPT_POSTGRES_HOST}"
            "-e POSTGRES_PORT=${AUTOGPT_POSTGRES_PORT}"
            "-e POSTGRES_DB=${AUTOGPT_POSTGRES_DB}"
        )
    fi
    
    # Remove existing container if it exists
    if autogpt_container_exists; then
        docker rm -f "${AUTOGPT_CONTAINER_NAME}" > /dev/null 2>&1
    fi
    
    # Start container
    # shellcheck disable=SC2068
    if ! docker run -d \
        --name "${AUTOGPT_CONTAINER_NAME}" \
        --restart unless-stopped \
        -p "${AUTOGPT_PORT_API}:${AUTOGPT_PORT_API}" \
        -v "${AUTOGPT_DATA_DIR}:/app/data" \
        -v "${AUTOGPT_AGENTS_DIR}:/app/agents" \
        -v "${AUTOGPT_TOOLS_DIR}:/app/tools" \
        -v "${AUTOGPT_WORKSPACE_DIR}:/app/workspace" \
        -v "${AUTOGPT_LOGS_DIR}:/app/logs" \
        ${env_vars[@]} \
        "${AUTOGPT_IMAGE}"; then
        log::error "Failed to start AutoGPT container"
        return 1
    fi
    
    # Wait for service to be ready if requested
    if [[ "${wait_flag}" == "--wait" ]]; then
        autogpt::core::wait_for_ready
        return $?
    fi
    
    log::success "AutoGPT started"
    return 0
}

# Stop AutoGPT service
autogpt::core::stop() {
    if ! autogpt_container_running; then
        log::warning "AutoGPT is not running"
        return 0
    fi
    
    log::info "Stopping AutoGPT..."
    
    if docker stop "${AUTOGPT_CONTAINER_NAME}" > /dev/null 2>&1; then
        log::success "AutoGPT stopped"
        return 0
    else
        log::error "Failed to stop AutoGPT"
        return 1
    fi
}

# Restart AutoGPT service
autogpt::core::restart() {
    log::info "Restarting AutoGPT..."
    
    if autogpt::core::stop; then
        sleep 2
        if autogpt::core::start "--wait"; then
            log::success "AutoGPT restarted"
            return 0
        fi
    fi
    
    log::error "Failed to restart AutoGPT"
    return 1
}

# Uninstall AutoGPT
autogpt::core::uninstall() {
    log::info "Uninstalling AutoGPT..."
    
    # Stop and remove container
    if autogpt_container_exists; then
        docker rm -f "${AUTOGPT_CONTAINER_NAME}" > /dev/null 2>&1
    fi
    
    # Optionally remove data (with confirmation)
    if [[ "${1:-}" == "--remove-data" ]]; then
        log::warning "Removing AutoGPT data directory: ${AUTOGPT_DATA_DIR}"
        rm -rf "${AUTOGPT_DATA_DIR}"
    else
        log::info "Data preserved at: ${AUTOGPT_DATA_DIR}"
    fi
    
    log::success "AutoGPT uninstalled"
    return 0
}

################################################################################
# Health and Status Functions
################################################################################

# Wait for AutoGPT to be ready
autogpt::core::wait_for_ready() {
    local max_attempts=30
    local attempt=0
    
    log::info "Waiting for AutoGPT to be ready..."
    
    while [[ ${attempt} -lt ${max_attempts} ]]; do
        if timeout 5 curl -sf "http://localhost:${AUTOGPT_PORT_API}/health" > /dev/null 2>&1; then
            log::success "AutoGPT is ready"
            return 0
        fi
        
        ((attempt++))
        log::debug "Waiting for AutoGPT... (${attempt}/${max_attempts})"
        sleep 2
    done
    
    log::error "AutoGPT failed to become ready"
    return 1
}

# Get detailed status
autogpt::core::status() {
    local json_format="${1:-}"
    
    if [[ "${json_format}" == "--json" ]]; then
        autogpt::core::status_json
    else
        autogpt::core::status_text
    fi
}

# Get status in JSON format
autogpt::core::status_json() {
    local status="stopped"
    local health="unknown"
    local agent_count=0
    local version="unknown"
    
    if autogpt_container_running; then
        status="running"
        
        # Check health
        if timeout 5 curl -sf "http://localhost:${AUTOGPT_PORT_API}/health" > /dev/null 2>&1; then
            health="healthy"
        else
            health="unhealthy"
        fi
        
        # Get version from container
        version=$(docker inspect "${AUTOGPT_CONTAINER_NAME}" --format='{{.Config.Image}}' 2>/dev/null | cut -d: -f2)
    fi
    
    # Count agents
    if [[ -d "${AUTOGPT_AGENTS_DIR}" ]]; then
        agent_count=$(find "${AUTOGPT_AGENTS_DIR}" -name "*.yaml" 2>/dev/null | wc -l)
    fi
    
    cat <<EOF
{
  "status": "${status}",
  "health": "${health}",
  "container": {
    "name": "${AUTOGPT_CONTAINER_NAME}",
    "image": "${AUTOGPT_IMAGE}",
    "version": "${version}"
  },
  "configuration": {
    "api_port": ${AUTOGPT_PORT_API},
    "llm_provider": "${AUTOGPT_AI_PROVIDER}",
    "memory_backend": "${AUTOGPT_MEMORY_BACKEND}"
  },
  "metrics": {
    "agents": ${agent_count}
  }
}
EOF
}

# Get status in text format
autogpt::core::status_text() {
    # Implementation provided by status.sh
    if [[ -f "${SCRIPT_DIR}/status.sh" ]]; then
        source "${SCRIPT_DIR}/status.sh"
        autogpt_status
    else
        log::error "Status implementation not found"
        return 1
    fi
}

################################################################################
# Content Management Functions
################################################################################

# Add agent configuration
autogpt::core::content_add() {
    local config_file="${1}"
    
    if [[ ! -f "${config_file}" ]]; then
        log::error "Configuration file not found: ${config_file}"
        return 1
    fi
    
    # Copy to agents directory
    local agent_name
    agent_name=$(basename "${config_file}" .yaml)
    cp "${config_file}" "${AUTOGPT_AGENTS_DIR}/${agent_name}.yaml"
    
    log::success "Agent configuration added: ${agent_name}"
    return 0
}

# List agents
autogpt::core::content_list() {
    if [[ ! -d "${AUTOGPT_AGENTS_DIR}" ]]; then
        log::warning "No agents directory found"
        return 0
    fi
    
    local agents
    agents=$(find "${AUTOGPT_AGENTS_DIR}" -name "*.yaml" -exec basename {} .yaml \; 2>/dev/null | sort)
    
    if [[ -z "${agents}" ]]; then
        log::info "No agents found"
    else
        echo "Available agents:"
        echo "${agents}" | sed 's/^/  - /'
    fi
    
    return 0
}

# Get agent configuration
autogpt::core::content_get() {
    local agent_name="${1}"
    local agent_file="${AUTOGPT_AGENTS_DIR}/${agent_name}.yaml"
    
    if [[ ! -f "${agent_file}" ]]; then
        log::error "Agent not found: ${agent_name}"
        return 1
    fi
    
    cat "${agent_file}"
    return 0
}

# Remove agent
autogpt::core::content_remove() {
    local agent_name="${1}"
    local agent_file="${AUTOGPT_AGENTS_DIR}/${agent_name}.yaml"
    
    if [[ ! -f "${agent_file}" ]]; then
        log::error "Agent not found: ${agent_name}"
        return 1
    fi
    
    rm -f "${agent_file}"
    log::success "Agent removed: ${agent_name}"
    return 0
}

# Execute agent
autogpt::core::content_execute() {
    local agent_name="${1}"
    
    if ! autogpt_container_running; then
        log::error "AutoGPT is not running"
        return 1
    fi
    
    # Execute agent via API
    local response
    response=$(curl -sf -X POST \
        "http://localhost:${AUTOGPT_PORT_API}/agents/${agent_name}/run" \
        -H "Content-Type: application/json" \
        2>/dev/null)
    
    if [[ $? -eq 0 ]]; then
        echo "${response}"
        return 0
    else
        log::error "Failed to execute agent: ${agent_name}"
        return 1
    fi
}

################################################################################
# Helper Functions
################################################################################

# Create default configuration
autogpt::core::create_default_config() {
    # Create sample agent configuration
    cat > "${AUTOGPT_AGENTS_DIR}/example-agent.yaml" <<EOF
name: example-agent
description: Example AutoGPT agent for testing
goal: Demonstrate AutoGPT capabilities
model: ${AUTOGPT_MODEL}
max_iterations: ${AUTOGPT_MAX_ITERATIONS}
memory_backend: ${AUTOGPT_MEMORY_BACKEND}
tools:
  - web_search
  - file_operations
  - terminal_commands
EOF
    
    log::info "Created example agent configuration"
}

# Get logs
autogpt::core::logs() {
    local options="${1:---tail=100}"
    
    if ! autogpt_container_exists; then
        log::error "AutoGPT container does not exist"
        return 1
    fi
    
    docker logs "${options}" "${AUTOGPT_CONTAINER_NAME}"
}

# Export functions for use by CLI
export -f autogpt::core::install
export -f autogpt::core::start
export -f autogpt::core::stop
export -f autogpt::core::restart
export -f autogpt::core::uninstall
export -f autogpt::core::status
export -f autogpt::core::wait_for_ready
export -f autogpt::core::content_add
export -f autogpt::core::content_list
export -f autogpt::core::content_get
export -f autogpt::core::content_remove
export -f autogpt::core::content_execute
export -f autogpt::core::logs