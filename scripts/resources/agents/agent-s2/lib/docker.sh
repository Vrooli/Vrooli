#!/usr/bin/env bash
# Agent S2 Docker Operations - Ultra-Simplified
# Uses docker-resource-utils.sh for minimal boilerplate

# Source var.sh to get proper directory variables
_AGENTS2_DOCKER_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck disable=SC1091
source "${_AGENTS2_DOCKER_DIR}/../../../../lib/utils/var.sh"

# Source shared libraries
# shellcheck disable=SC1091
source "${var_LIB_SERVICE_DIR}/secrets.sh"
# shellcheck disable=SC1091
source "${var_SCRIPTS_RESOURCES_LIB_DIR}/docker-resource-utils.sh"

#######################################
# Build Agent S2 Docker image
# Returns: 0 if successful, 1 if failed
#######################################
agents2::docker_build() {
    log::info "$MSG_BUILDING_IMAGE"
    
    local build_dir="${AGENT_S2_SCRIPT_DIR}"
    local dockerfile="${build_dir}/docker/images/agent-s2/Dockerfile"
    
    # Ensure Dockerfile exists
    if [[ ! -f "$dockerfile" ]]; then
        log::error "Dockerfile not found at: $dockerfile"
        return 1
    fi
    
    # Build with proper arguments
    local build_args=(
        --build-arg "VNC_PASSWORD=$AGENTS2_VNC_PASSWORD"
        --build-arg "USER_ID=$AGENTS2_USER_ID"
        --build-arg "GROUP_ID=$AGENTS2_GROUP_ID"
        -f "$dockerfile" -t "$AGENTS2_IMAGE_NAME" "$build_dir"
    )
    
    if docker build "${build_args[@]}"; then
        log::success "$MSG_BUILD_SUCCESS"
        return 0
    else
        log::error "$MSG_BUILD_FAILED"
        return 1
    fi
}

#######################################
# Create Docker network for Agent S2
# Returns: 0 if successful or exists, 1 if failed
#######################################
agents2::docker_create_network() {
    docker::create_network "$AGENTS2_NETWORK_NAME"
}

#######################################
# Wait for Agent S2 to be ready
# Returns: 0 if ready, 1 if failed
#######################################
agents2::wait_for_ready() {
    local max_wait="${AGENTS2_STARTUP_MAX_WAIT:-120}"
    local wait_interval="${AGENTS2_STARTUP_WAIT_INTERVAL:-5}"
    local elapsed=0
    
    log::info "$MSG_WAITING_READY"
    
    while [[ $elapsed -lt $max_wait ]]; do
        if agents2::is_healthy; then
            return 0
        fi
        sleep "$wait_interval"
        elapsed=$((elapsed + wait_interval))
    done
    
    log::warn "Service did not become ready within ${max_wait} seconds"
    return 1
}

#######################################
# Create and start Agent S2 container
# Returns: 0 on success, 1 on failure
#######################################
agents2::docker_start() {
    if agents2::is_running; then
        if [[ "$FORCE" != "yes" ]]; then
            log::info "$MSG_CONTAINER_RUNNING"
            return 0
        else
            log::info "Force specified, stopping existing container..."
            agents2::docker_stop
        fi
    fi
    
    # Remove existing stopped container
    if agents2::container_exists && ! agents2::is_running; then
        log::info "Removing existing stopped container..."
        docker::remove_container "$AGENTS2_CONTAINER_NAME" "true"
    fi
    
    # Load environment and save configuration
    agents2::load_environment_from_config
    agents2::save_env_config || return 1
    
    log::info "$MSG_STARTING_CONTAINER"
    
    # Skip pull for locally-built images - agent-s2 is always built locally
    if ! docker image inspect "$AGENTS2_IMAGE_NAME" >/dev/null 2>&1; then
        log::error "Docker image $AGENTS2_IMAGE_NAME not found. Please run install first to build the image."
        return 1
    fi
    
    # Prepare volumes
    local volumes="${AGENTS2_DATA_DIR}/logs:/var/log/supervisor:rw ${AGENTS2_DATA_DIR}/cache:/home/agents2/.cache:rw ${AGENTS2_DATA_DIR}/models:/home/agents2/.agent-s2/models:rw ${AGENTS2_DATA_DIR}/sessions:/home/agents2/.agent-s2/sessions:rw"
    
    if [[ "$AGENTS2_ENABLE_HOST_DISPLAY" == "yes" ]]; then
        log::warn "⚠️  Host display access enabled - this is a security risk!"
        volumes="$volumes /tmp/.X11-unix:/tmp/.X11-unix:rw"
    fi
    
    # Environment variables
    local env_vars=(
        "OPENAI_API_KEY=$AGENTS2_OPENAI_API_KEY"
        "ANTHROPIC_API_KEY=$AGENTS2_ANTHROPIC_API_KEY"
        "AGENTS2_LLM_PROVIDER=$AGENTS2_LLM_PROVIDER"
        "AGENTS2_LLM_MODEL=$AGENTS2_LLM_MODEL"
        "AGENTS2_OLLAMA_BASE_URL=$AGENTS2_OLLAMA_BASE_URL"
        "AGENTS2_ENABLE_AI=$AGENTS2_ENABLE_AI"
        "AGENTS2_ENABLE_SEARCH=$AGENTS2_ENABLE_SEARCH"
        "DISPLAY=$AGENTS2_DISPLAY"
        "AGENT_S2_VIRUSTOTAL_API_KEY=${AGENTS2_VIRUSTOTAL_API_KEY:-}"
        "AGENT_S2_ENABLE_PROXY=$AGENTS2_ENABLE_PROXY"
    )
    
    # Add display environment if enabled
    [[ "$AGENTS2_ENABLE_HOST_DISPLAY" == "yes" ]] && env_vars+=("DISPLAY=${DISPLAY:-:0}")
    
    # Docker options for security and resources
    local docker_opts=(
        "--add-host=host.docker.internal:host-gateway"
        "--cap-add" "NET_ADMIN"
        "--cap-add" "NET_RAW"
        "--security-opt" "$AGENTS2_SECURITY_OPT"
        "--shm-size" "$AGENTS2_SHM_SIZE"
        "--memory" "$AGENTS2_MEMORY_LIMIT"
        "--cpus" "$AGENTS2_CPU_LIMIT"
    )
    
    # Port mappings for main app and VNC
    local port_mappings="${AGENTS2_PORT}:4113 ${AGENTS2_VNC_PORT}:5900"
    
    # Use advanced creation
    docker_resource::create_service_advanced \
        "$AGENTS2_CONTAINER_NAME" \
        "$AGENTS2_IMAGE_NAME" \
        "$port_mappings" \
        "$AGENTS2_NETWORK_NAME" \
        "$volumes" \
        "env_vars" \
        "docker_opts" \
        "" \
        ""
    
    if [[ $? -eq 0 ]]; then
        log::success "$MSG_CONTAINER_STARTED"
        sleep "$AGENTS2_INITIALIZATION_WAIT"
        
        if agents2::wait_for_ready; then
            log::success "$MSG_SERVICE_HEALTHY"
            return 0
        else
            log::warn "$MSG_HEALTH_CHECK_FAILED"
            log::info "The service may still be initializing. Check logs with: $0 --action logs"
            return 0
        fi
    else
        log::error "Failed to start container"
        return 1
    fi
}

#######################################
# Stop Agent S2 container
# Returns: 0 if successful, 1 if failed
#######################################
agents2::docker_stop() {
    if ! agents2::is_running; then
        log::info "$MSG_CONTAINER_NOT_RUNNING"
        agents2::cleanup_nat_rules
        return 0
    fi
    
    log::info "$MSG_STOPPING_CONTAINER"
    if docker::stop_container "$AGENTS2_CONTAINER_NAME"; then
        log::success "$MSG_CONTAINER_STOPPED"
        agents2::cleanup_nat_rules
        return 0
    else
        log::error "Failed to stop container"
        return 1
    fi
}

#######################################
# Restart Agent S2 container
# Returns: 0 if successful, 1 if failed
#######################################
agents2::docker_restart() {
    log::info "Restarting Agent S2..."
    if docker::restart_container "$AGENTS2_CONTAINER_NAME"; then
        sleep "$AGENTS2_INITIALIZATION_WAIT"
        
        if agents2::wait_for_ready; then
            log::success "$MSG_SERVICE_HEALTHY"
            return 0
        else
            log::warn "$MSG_HEALTH_CHECK_FAILED"
            return 0
        fi
    else
        log::error "Failed to restart Agent S2"
        return 1
    fi
}

#######################################
# Show container logs
# Arguments: $1 - number of lines to show (optional, default: follow)
#######################################
agents2::docker_logs() {
    local lines="${1:-50}"
    local follow="${2:-true}"
    docker_resource::show_logs_with_follow "$AGENTS2_CONTAINER_NAME" "$lines" "$follow"
}

#######################################
# Execute command in container
# Arguments: $@ - command to execute
# Returns: exit code from command
#######################################
agents2::docker_exec() {
    docker_resource::exec_interactive "$AGENTS2_CONTAINER_NAME" "$@"
}

#######################################
# Get container statistics
# Returns: container stats as string
#######################################
agents2::docker_stats() {
    docker_resource::get_stats "$AGENTS2_CONTAINER_NAME"
}

#######################################
# Remove Agent S2 container and network
# Returns: 0 if successful, 1 if failed
#######################################
agents2::docker_cleanup() {
    local success=true
    
    # Stop and remove container
    agents2::is_running && { agents2::docker_stop || success=false; }
    
    if agents2::container_exists; then
        log::info "Removing container..."
        docker::remove_container "$AGENTS2_CONTAINER_NAME" "true" || success=false
    fi
    
    # Remove network if empty
    if docker network ls --format "{{.Name}}" | grep -q "^${AGENTS2_NETWORK_NAME}$"; then
        if docker network inspect "$AGENTS2_NETWORK_NAME" --format '{{len .Containers}}' | grep -q '^0$'; then
            log::info "Removing network..."
            docker network rm "$AGENTS2_NETWORK_NAME" >/dev/null 2>&1 || success=false
        else
            log::info "Network has other containers, skipping removal"
        fi
    fi
    
    # Remove image if requested
    if [[ "${REMOVE_IMAGE:-no}" == "yes" ]] && agents2::image_exists; then
        log::info "Removing Docker image..."
        docker rmi "$AGENTS2_IMAGE_NAME" >/dev/null 2>&1 || success=false
    fi
    
    $success && return 0 || return 1
}

#######################################
# Clean up NAT redirection rules created by Agent-S2
# This prevents traffic hijacking when the proxy service is stopped
# Returns: 0 if successful, 1 if failed
#######################################
agents2::cleanup_nat_rules() {
    log::info "Cleaning up Agent-S2 NAT redirection rules..."
    
    # Use the clean-iptables.sh script if available
    local cleanup_script="${AGENT_S2_SCRIPT_DIR}/docker/scripts/clean-iptables.sh"
    
    if [[ -f "$cleanup_script" ]]; then
        log::debug "Using cleanup script: $cleanup_script"
        if "$cleanup_script" 2>/dev/null; then
            log::success "NAT redirection rules cleaned successfully"
            return 0
        else
            log::warn "Cleanup script failed, attempting manual cleanup..."
        fi
    else
        log::debug "Cleanup script not found, attempting manual cleanup..."
    fi
    
    # Manual cleanup as fallback
    local removed=0
    local success=true
    
    # Use sudo if available and needed, otherwise skip (non-critical for some environments)
    local SUDO_CMD=""
    if [[ $EUID -ne 0 ]] && command -v sudo >/dev/null 2>&1 && sudo -n true 2>/dev/null; then
        SUDO_CMD="sudo"
    fi
    
    if [[ -n "$SUDO_CMD" ]] || [[ $EUID -eq 0 ]]; then
        # Remove all OUTPUT rules in nat table that redirect to Agent-S2 ports
        while true; do
            # Find first rule that matches our patterns
            local rule_num
            rule_num=$($SUDO_CMD iptables -t nat -L OUTPUT --line-numbers -n 2>/dev/null | \
                grep -E "REDIRECT.*tcp.*dpt:(80|443|8080|8443).*redir ports (8080|8085)" | \
                head -1 | awk '{print $1}')
            
            if [[ -n "$rule_num" ]]; then
                if $SUDO_CMD iptables -t nat -D OUTPUT "$rule_num" 2>/dev/null; then
                    ((removed++))
                else
                    success=false
                    break
                fi
            else
                break
            fi
        done
        
        # Also remove loopback exclusion rules
        while $SUDO_CMD iptables -t nat -D OUTPUT -o lo -j RETURN 2>/dev/null; do
            ((removed++))
        done
        
        if [[ $removed -gt 0 ]]; then
            log::success "Removed $removed NAT redirection rules"
        else
            log::debug "No Agent-S2 NAT rules found to remove"
        fi
    else
        log::warn "Cannot clean NAT rules - requires root privileges or sudo access"
        log::info "If you experience connectivity issues, manually run: sudo ${cleanup_script}"
        return 1
    fi
    
    $success && return 0 || return 1
}

# Export functions for subshell availability
export -f agents2::docker_build
export -f agents2::docker_create_network
export -f agents2::wait_for_ready
export -f agents2::docker_start
export -f agents2::docker_stop
export -f agents2::docker_restart
export -f agents2::docker_logs
export -f agents2::docker_exec
export -f agents2::docker_stats
export -f agents2::docker_cleanup
export -f agents2::cleanup_nat_rules