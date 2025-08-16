#!/usr/bin/env bash
# Generic Docker Utility Functions
# Provides reusable Docker operations for all resource managers

# Source guard to prevent multiple sourcing
[[ -n "${_DOCKER_UTILS_SOURCED:-}" ]] && return 0
export _DOCKER_UTILS_SOURCED=1

# Source required utilities
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck disable=SC1091
source "${SCRIPT_DIR}/../../lib/utils/var.sh" 2>/dev/null || true
# shellcheck disable=SC1091
source "${SCRIPT_DIR}/../../lib/utils/log.sh" 2>/dev/null || true
# shellcheck disable=SC1091
source "${var_LIB_SYSTEM_DIR}/system_commands.sh"

#######################################
# Check if Docker is installed and accessible
# Returns: 0 if Docker is ready, 1 otherwise
#######################################
docker::check_daemon() {
    if ! system::is_command "docker"; then
        log::error "Docker is not installed"
        log::info "Please install Docker first: https://docs.docker.com/get-docker/"
        return 1
    fi
    
    # Check if Docker daemon is running
    if ! docker info >/dev/null 2>&1; then
        log::error "Docker daemon is not running"
        log::info "Start Docker with: sudo systemctl start docker"
        return 1
    fi
    
    # Check if user has permissions
    if ! docker ps >/dev/null 2>&1; then
        log::error "Current user doesn't have Docker permissions"
        log::info "Add user to docker group: sudo usermod -aG docker $USER"
        log::info "Then log out and back in for changes to take effect"
        return 1
    fi
    
    return 0
}

#######################################
# Check if a container exists
# Args: $1 - container_name
# Returns: 0 if exists, 1 otherwise
#######################################
docker::container_exists() {
    local container_name="$1"
    docker ps -a --format '{{.Names}}' | grep -q "^${container_name}$"
}

#######################################
# Check if a container is running
# Args: $1 - container_name
# Returns: 0 if running, 1 otherwise
#######################################
docker::is_running() {
    local container_name="$1"
    docker ps --format '{{.Names}}' | grep -q "^${container_name}$"
}

#######################################
# Simplified running check (for backward compatibility)
# Args: $1 - container_name
# Returns: 0 if running, 1 otherwise
#######################################
docker::container_running() {
    docker::is_running "$1"
}

#######################################
# Check if port is available
# Args: $1 - port number
# Returns: 0 if available, 1 if in use
#######################################
docker::is_port_available() {
    local port="$1"
    
    if system::is_command "lsof"; then
        ! lsof -i :"$port" >/dev/null 2>&1
    elif system::is_command "netstat"; then
        ! netstat -tln | grep -q ":$port "
    elif system::is_command "ss"; then
        ! ss -tln | grep -q ":$port "
    else
        # If we can't check, assume it's available
        return 0
    fi
}

#######################################
# Create Docker network if it doesn't exist
# Args: $1 - network_name
# Returns: 0 on success, 1 on failure
#######################################
docker::create_network() {
    local network_name="$1"
    
    if ! docker network ls | grep -q "$network_name"; then
        log::info "Creating Docker network: $network_name"
        
        if docker network create "$network_name" >/dev/null 2>&1; then
            log::success "Docker network created: $network_name"
            return 0
        else
            log::warn "Failed to create Docker network (may already exist)"
            return 1
        fi
    fi
    return 0
}

#######################################
# Remove Docker network
# Args: $1 - network_name
# Returns: 0 on success (even if network didn't exist)
#######################################
docker::remove_network() {
    local network_name="$1"
    
    if docker network ls | grep -q "$network_name"; then
        log::info "Removing Docker network: $network_name"
        if docker network rm "$network_name" >/dev/null 2>&1; then
            log::success "Network removed: $network_name"
            return 0
        else
            log::warn "Failed to remove network (may have attached containers): $network_name"
            return 1
        fi
    else
        log::debug "Network $network_name does not exist, nothing to remove"
        return 0
    fi
}

#######################################
# Remove Docker network only if empty (no containers attached)
# Args: $1 - network_name
# Returns: 0 on success or if network doesn't exist, 1 if network has containers
#######################################
docker::cleanup_network_if_empty() {
    local network_name="$1"
    
    # Check if network exists
    if ! docker network inspect "$network_name" >/dev/null 2>&1; then
        return 0  # Network doesn't exist, nothing to do
    fi
    
    # Check if network has any containers attached
    local network_containers
    network_containers=$(docker network inspect "$network_name" --format '{{len .Containers}}' 2>/dev/null || echo "0")
    
    if [[ "$network_containers" -eq 0 ]]; then
        log::info "Removing empty network: $network_name"
        if docker network rm "$network_name" >/dev/null 2>&1; then
            log::success "Empty network removed: $network_name"
            return 0
        else
            log::warn "Failed to remove empty network: $network_name"
            return 1
        fi
    else
        log::debug "Network $network_name has $network_containers container(s) attached, keeping it"
        return 1
    fi
}

#######################################
# Get container environment variable safely
# Args: $1 - container_name, $2 - variable_name
# Returns: Variable value via stdout, empty if not found
#######################################
docker::extract_env() {
    local container_name="$1"
    local var_name="$2"
    
    if docker::is_running "$container_name"; then
        docker exec "$container_name" env 2>/dev/null | grep "^${var_name}=" | cut -d'=' -f2- || echo ""
    else
        echo ""
    fi
}

#######################################
# Stop a Docker container gracefully
# Args: $1 - container_name, $2 - timeout (optional, default 10)
# Returns: 0 on success, 1 on failure
#######################################
docker::stop_container() {
    local container_name="$1"
    local timeout="${2:-10}"
    
    if ! docker::is_running "$container_name"; then
        log::debug "Container $container_name is not running, nothing to stop"
        return 0  # Already stopped
    fi
    
    log::info "Stopping container: $container_name"
    if docker stop -t "$timeout" "$container_name" >/dev/null 2>&1; then
        log::success "Container stopped successfully: $container_name"
        return 0
    else
        log::error "Failed to stop container: $container_name"
        return 1
    fi
}

#######################################
# Remove a Docker container
# Args: $1 - container_name, $2 - force (optional, "true" to force)
# Returns: 0 on success, 1 on failure
#######################################
docker::remove_container() {
    local container_name="$1"
    local force="${2:-false}"
    
    if ! docker::container_exists "$container_name"; then
        log::debug "Container $container_name does not exist, nothing to remove"
        return 0  # Already removed
    fi
    
    log::info "Removing container: $container_name"
    
    # Stop first if running and force is true
    if [[ "$force" == "true" ]] && docker::is_running "$container_name"; then
        log::debug "Stopping running container before removal: $container_name"
        docker::stop_container "$container_name" 10 || true
    fi
    
    # Build remove command properly
    local cmd=("docker" "rm")
    if [[ "$force" == "true" ]]; then
        cmd+=("-f")
    fi
    cmd+=("$container_name")
    
    if "${cmd[@]}" >/dev/null 2>&1; then
        log::success "Container removed successfully: $container_name"
        return 0
    else
        log::error "Failed to remove container: $container_name"
        return 1
    fi
}

#######################################
# Pull Docker image with retry
# Args: $1 - image_name, $2 - max_attempts (optional, default 3)
# Returns: 0 on success, 1 on failure
#######################################
docker::pull_image() {
    local image_name="$1"
    local max_attempts="${2:-3}"
    local attempt=1
    
    log::info "Pulling Docker image: $image_name"
    
    while [ $attempt -le $max_attempts ]; do
        if docker pull "$image_name" 2>&1; then
            log::success "Image pulled successfully: $image_name"
            return 0
        fi
        
        if [ $attempt -lt $max_attempts ]; then
            log::warn "Pull attempt $attempt failed, retrying..."
            sleep 5
        fi
        attempt=$((attempt + 1))
    done
    
    log::error "Failed to pull image after $max_attempts attempts: $image_name"
    return 1
}

#######################################
# Get container logs
# Args: $1 - container_name, $2 - lines (optional, default 50)
# Returns: Logs via stdout
#######################################
docker::get_logs() {
    local container_name="$1"
    local lines="${2:-50}"
    
    if docker::container_exists "$container_name"; then
        docker logs --tail "$lines" "$container_name" 2>&1
    fi
}

#######################################
# Check if image exists locally
# Args: $1 - image_name
# Returns: 0 if exists, 1 otherwise
#######################################
docker::image_exists() {
    local image_name="$1"
    docker images --format "{{.Repository}}:{{.Tag}}" | grep -q "^${image_name}$"
}

#######################################
# Start existing container
# Args: $1 - container_name
# Returns: 0 on success, 1 on failure
#######################################
docker::start_container() {
    local container_name="$1"
    
    if docker::is_running "$container_name"; then
        return 0  # Already running
    fi
    
    if ! docker::container_exists "$container_name"; then
        log::error "Container does not exist: $container_name"
        return 1
    fi
    
    log::info "Starting container: $container_name"
    if docker start "$container_name" >/dev/null 2>&1; then
        log::success "Container started: $container_name"
        return 0
    else
        log::error "Failed to start container: $container_name"
        return 1
    fi
}

#######################################
# Restart container
# Args: $1 - container_name, $2 - timeout (optional)
# Returns: 0 on success, 1 on failure
#######################################
docker::restart_container() {
    local container_name="$1"
    local timeout="${2:-10}"
    
    log::info "Restarting container: $container_name"
    
    if docker::is_running "$container_name"; then
        docker::stop_container "$container_name" "$timeout" || true
        sleep 2
    fi
    
    docker::start_container "$container_name"
}

#######################################
# Execute command in container
# Args: $1 - container_name, $2 - command, $3+ - args
# Returns: Command exit code, output via stdout
#######################################
docker::exec() {
    local container_name="$1"
    shift
    
    if ! docker::is_running "$container_name"; then
        log::error "Container is not running: $container_name"
        return 1
    fi
    
    docker exec "$container_name" "$@"
}

#######################################
# Check container health status
# Args: $1 - container_name
# Returns: 0 if healthy, 1 if unhealthy/starting, 2 if no health check
#######################################
docker::check_health() {
    local container_name="$1"
    
    if ! docker::is_running "$container_name"; then
        return 1
    fi
    
    local health_status
    health_status=$(docker inspect --format='{{.State.Health.Status}}' "$container_name" 2>/dev/null || echo "none")
    
    case "$health_status" in
        healthy)
            return 0
            ;;
        unhealthy|starting)
            return 1
            ;;
        *)
            return 2  # No health check defined
            ;;
    esac
}

#######################################
# Wait for container to be healthy
# Args: $1 - container_name, $2 - max_wait (seconds, default 60)
# Returns: 0 if healthy, 1 if timeout
#######################################
docker::wait_for_healthy() {
    local container_name="$1"
    local max_wait="${2:-60}"
    local elapsed=0
    
    log::info "Waiting for container to be healthy: $container_name"
    
    while [ $elapsed -lt $max_wait ]; do
        if docker::check_health "$container_name"; then
            log::success "Container is healthy: $container_name"
            return 0
        fi
        
        sleep 2
        elapsed=$((elapsed + 2))
        echo -n "."
    done
    
    echo
    log::error "Container failed to become healthy after ${max_wait}s: $container_name"
    return 1
}

#######################################
# Get container stats
# Args: $1 - container_name
# Returns: Stats via stdout
#######################################
docker::get_stats() {
    local container_name="$1"
    
    if docker::is_running "$container_name"; then
        docker stats "$container_name" --no-stream --format "table {{.Container}}\t{{.CPUPerc}}\t{{.MemUsage}}" 2>/dev/null
    fi
}

#######################################
# Check if container port is bound
# Args: $1 - container_name, $2 - port
# Returns: 0 if bound, 1 otherwise
#######################################
docker::is_port_bound() {
    local container_name="$1"
    local port="$2"
    
    if docker::is_running "$container_name"; then
        docker port "$container_name" "$port" 2>/dev/null | grep -q "0.0.0.0"
    else
        return 1
    fi
}

#######################################
# Docker Inspect Utility Functions
# Standardized container information extraction
#######################################

#######################################
# Get container creation time
# Args: $1 - container_name
# Returns: Creation timestamp via stdout
#######################################
docker::get_created() {
    local container_name="$1"
    docker inspect "$container_name" --format='{{.Created}}' 2>/dev/null || echo ""
}

#######################################
# Get container image
# Args: $1 - container_name
# Returns: Image name via stdout
#######################################
docker::get_image() {
    local container_name="$1"
    docker inspect "$container_name" --format='{{.Config.Image}}' 2>/dev/null || echo ""
}

#######################################
# Get all container environment variables
# Args: $1 - container_name
# Returns: Environment variables (one per line) via stdout
#######################################
docker::get_all_env() {
    local container_name="$1"
    docker inspect "$container_name" --format='{{range .Config.Env}}{{println .}}{{end}}' 2>/dev/null
}

#######################################
# Get container environment variable using inspect
# Args: $1 - container_name, $2 - variable_name
# Returns: Variable value via stdout, empty if not found
#######################################
docker::inspect_env() {
    local container_name="$1"
    local var_name="$2"
    
    # Get all env vars and filter for the one we want - simpler and more maintainable
    docker inspect "$container_name" --format='{{range .Config.Env}}{{println .}}{{end}}' 2>/dev/null | \
        grep "^${var_name}=" | \
        cut -d'=' -f2- || echo ""
}

#######################################
# Get container network information
# Args: $1 - container_name
# Returns: Network details (formatted) via stdout
#######################################
docker::get_networks() {
    local container_name="$1"
    docker inspect "$container_name" --format='{{range $net, $conf := .NetworkSettings.Networks}}  {{$net}}: {{$conf.IPAddress}}{{end}}' 2>/dev/null
}

#######################################
# Get container volume mounts
# Args: $1 - container_name
# Returns: Volume mount details via stdout
#######################################
docker::get_mounts() {
    local container_name="$1"
    docker inspect "$container_name" --format='{{range .Mounts}}  {{.Source}} -> {{.Destination}}{{println}}{{end}}' 2>/dev/null
}

#######################################
# Get container port mappings for recreation
# Args: $1 - container_name
# Returns: Port mapping arguments for docker run
#######################################
docker::get_port_args() {
    local container_name="$1"
    docker inspect "$container_name" --format='{{range $p, $conf := .NetworkSettings.Ports}}{{if $conf}}-p {{(index $conf 0).HostPort}}:{{$p}} {{end}}{{end}}' 2>/dev/null | sed 's|/tcp||g'
}

#######################################
# Get container volume mount arguments for recreation  
# Args: $1 - container_name
# Returns: Volume mount arguments for docker run
#######################################
docker::get_volume_args() {
    local container_name="$1"
    docker inspect "$container_name" --format='{{range .Mounts}}{{if eq .Type "bind"}}-v {{.Source}}:{{.Destination}} {{end}}{{end}}' 2>/dev/null
}

#######################################
# Get container restart policy
# Args: $1 - container_name
# Returns: Restart policy via stdout
#######################################
docker::get_restart_policy() {
    local container_name="$1"
    docker inspect "$container_name" --format='{{.HostConfig.RestartPolicy.Name}}' 2>/dev/null || echo ""
}

#######################################
# Get container state status
# Args: $1 - container_name
# Returns: Container state (running, exited, etc.) via stdout
#######################################
docker::get_state() {
    local container_name="$1"
    docker inspect "$container_name" --format='{{.State.Status}}' 2>/dev/null || echo ""
}

################################################################################
# MIGRATION NOTICE
################################################################################

# NOTE: Complex container lifecycle patterns have been moved to:
# docker-resource-utils.sh - Use this for resource container creation patterns
#
# Available functions:
# - docker_resource::create_service()
# - docker_resource::create_service_with_command()  
# - docker_resource::create_client_instance()
# - docker_resource::remove_client_instance()
# - docker_resource::remove_data()
# - docker_resource::find_available_port()
#
# These functions provide simple, intuitive APIs for common resource patterns
# instead of complex JSON configuration building.



