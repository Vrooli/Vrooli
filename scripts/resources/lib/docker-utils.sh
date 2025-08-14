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
source "${SCRIPT_DIR}/../../lib/system/system_commands.sh" 2>/dev/null || true

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
        docker network rm "$network_name" >/dev/null 2>&1 || true
    fi
    return 0
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
        return 0  # Already stopped
    fi
    
    log::info "Stopping container: $container_name"
    if docker stop -t "$timeout" "$container_name" >/dev/null 2>&1; then
        log::success "Container stopped: $container_name"
        return 0
    else
        log::error "Failed to stop container: $container_name"
        return 1
    fi
}

#######################################
# Remove a Docker container
# Args: $1 - container_name, $2 - force (optional, "true" to force)
# Returns: 0 on success
#######################################
docker::remove_container() {
    local container_name="$1"
    local force="${2:-false}"
    
    if ! docker::container_exists "$container_name"; then
        return 0  # Already removed
    fi
    
    log::info "Removing container: $container_name"
    
    local remove_args=""
    if [[ "$force" == "true" ]]; then
        remove_args="-f"
    fi
    
    docker rm $remove_args "$container_name" >/dev/null 2>&1 || true
    return 0
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
    docker inspect "$container_name" --format="{{range .Config.Env}}{{if eq (index (split . \"=\") 0) \"$var_name\"}}{{index (split . \"=\") 1}}{{end}}{{end}}" 2>/dev/null
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
# CONTAINER LIFECYCLE MANAGEMENT
# High-level container orchestration functions built on top of basic utilities
################################################################################

#######################################
# Start container with configuration-driven approach
# Args: $1 - container_config (JSON configuration)
# Returns: 0 on success, 1 on failure
#
# Configuration Schema:
# {
#   "name": "container-name",
#   "image": "image:tag",
#   "ports": {"host_port": "container_port"},
#   "volumes": {
#     "volume_name": "container_path",
#     "host_path": "container_path:mode"
#   },
#   "environment": {"ENV_VAR": "value"},
#   "network": "network-name" (optional),
#   "restart_policy": "unless-stopped|always|on-failure" (optional),
#   "health": {
#     "endpoint": "/health" (optional),
#     "timeout": 30 (optional),
#     "interval": 5 (optional)
#   },
#   "extra_args": ["--extra", "args"] (optional)
# }
#######################################
docker::start_with_config() {
    local container_config="$1"
    
    # Validate configuration
    if ! docker::_validate_container_config "$container_config"; then
        log::error "Invalid container configuration"
        return 1
    fi
    
    local container_name
    local image
    
    container_name=$(echo "$container_config" | jq -r '.name')
    image=$(echo "$container_config" | jq -r '.image')
    
    # Check if already running
    if docker::is_running "$container_name"; then
        log::info "Container already running: $container_name"
        return 0
    fi
    
    # Remove existing container if it exists but is stopped
    if docker::container_exists "$container_name"; then
        log::info "Removing existing stopped container: $container_name"
        docker::remove_container "$container_name" "true"
    fi
    
    # Ensure image is available
    if ! docker::image_exists "$image"; then
        log::info "Image not found locally, pulling: $image"
        if ! docker::pull_image "$image"; then
            log::error "Failed to pull image: $image"
            return 1
        fi
    fi
    
    # Build docker run command from configuration
    local docker_cmd
    if ! docker_cmd=$(docker::_build_run_command "$container_config"); then
        log::error "Failed to build docker run command"
        return 1
    fi
    
    # Create network if specified
    local network
    network=$(echo "$container_config" | jq -r '.network // empty')
    if [[ -n "$network" ]]; then
        docker::create_network "$network" || true
    fi
    
    # Start container
    log::info "Starting container: $container_name"
    if eval "$docker_cmd"; then
        log::success "Container started: $container_name"
        
        # Wait for health if health check is configured
        if echo "$container_config" | jq -e '.health' >/dev/null 2>&1; then
            docker::_wait_for_container_health "$container_config"
        fi
        
        return 0
    else
        log::error "Failed to start container: $container_name"
        return 1
    fi
}

#######################################
# Stop container with configuration-based cleanup
# Args: $1 - container_config (JSON) or container_name (string)
#       $2 - cleanup_volumes (optional, "true" to remove volumes)
# Returns: 0 on success
#######################################
docker::stop_with_config() {
    local config_or_name="$1"
    local cleanup_volumes="${2:-false}"
    
    local container_name
    local network
    
    # Handle both config JSON and simple container name
    if echo "$config_or_name" | jq empty 2>/dev/null; then
        # It's JSON config
        container_name=$(echo "$config_or_name" | jq -r '.name')
        network=$(echo "$config_or_name" | jq -r '.network // empty')
    else
        # It's a simple container name
        container_name="$config_or_name"
        network=""
    fi
    
    # Stop and remove container
    if docker::is_running "$container_name"; then
        docker::stop_container "$container_name"
    fi
    
    if docker::container_exists "$container_name"; then
        docker::remove_container "$container_name" "true"
    fi
    
    # Clean up volumes if requested
    if [[ "$cleanup_volumes" == "true" ]]; then
        docker::_cleanup_container_volumes "$container_name"
    fi
    
    # Remove network if it was created and no other containers use it
    if [[ -n "$network" ]] && docker network ls | grep -q "$network"; then
        local network_usage
        network_usage=$(docker network inspect "$network" --format='{{len .Containers}}' 2>/dev/null || echo "0")
        if [[ "$network_usage" == "0" ]]; then
            log::info "Removing unused network: $network"
            docker::remove_network "$network" || true
        fi
    fi
    
    return 0
}

#######################################
# Restart container with configuration
# Args: $1 - container_config (JSON configuration)
# Returns: 0 on success, 1 on failure
#######################################
docker::restart_with_config() {
    local container_config="$1"
    
    local container_name
    container_name=$(echo "$container_config" | jq -r '.name')
    
    log::info "Restarting container: $container_name"
    
    # Stop with cleanup but preserve volumes
    docker::stop_with_config "$container_config" "false"
    
    # Start again
    docker::start_with_config "$container_config"
}

#######################################
# Wait for container to be healthy using multiple methods
# Args: $1 - container_config (JSON configuration)
# Returns: 0 if healthy, 1 if timeout
#######################################
docker::wait_for_container_health() {
    local container_config="$1"
    
    docker::_wait_for_container_health "$container_config"
}

#######################################
# Build standardized volume configuration
# Args: $1 - volume_type (data|logs|workspace|docker_sock|custom)
#       $2 - container_path
#       $3 - host_path_or_name (optional for named volumes)
#       $4 - mode (optional: ro|rw, default rw)
# Returns: Volume mount argument via stdout
#######################################
docker::build_volume_mount() {
    local volume_type="$1"
    local container_path="$2"
    local host_path_or_name="${3:-}"
    local mode="${4:-rw}"
    
    case "$volume_type" in
        "data")
            # Named volume for persistent data
            local volume_name="${host_path_or_name}"
            echo "-v ${volume_name}:${container_path}"
            ;;
        "logs")
            # Host directory for logs
            local log_dir="${host_path_or_name:-${HOME}/logs}"
            mkdir -p "$log_dir"
            echo "-v ${log_dir}:${container_path}:${mode}"
            ;;
        "workspace")
            # Mount workspace directory
            local workspace_dir="${host_path_or_name:-${PWD}}"
            echo "-v ${workspace_dir}:${container_path}:${mode}"
            ;;
        "docker_sock")
            # Docker socket for container control
            if [[ -S /var/run/docker.sock ]]; then
                echo "-v /var/run/docker.sock:/var/run/docker.sock:${mode}"
            fi
            ;;
        "host_binaries")
            # Mount host binaries for extended capabilities
            local mounts=""
            for bin_dir in /usr/bin /bin /usr/local/bin; do
                if [[ -d "$bin_dir" ]]; then
                    mounts="${mounts} -v ${bin_dir}:/host${bin_dir}:ro"
                fi
            done
            echo "$mounts"
            ;;
        "custom")
            # Custom host path mount
            if [[ -n "$host_path_or_name" ]]; then
                echo "-v ${host_path_or_name}:${container_path}:${mode}"
            fi
            ;;
        *)
            log::warn "Unknown volume type: $volume_type"
            return 1
            ;;
    esac
}

#######################################
# Start multiple related containers in correct order
# Args: $1 - containers_config (JSON array of container configurations)
# Returns: 0 if all started successfully, 1 on failure
#
# Configuration Schema:
# [
#   {
#     "name": "database",
#     "image": "postgres:13",
#     "priority": 1,
#     "depends_on": [],
#     ...standard container config...
#   },
#   {
#     "name": "app",
#     "image": "myapp:latest",
#     "priority": 2,
#     "depends_on": ["database"],
#     ...standard container config...
#   }
# ]
#######################################
docker::start_container_stack() {
    local containers_config="$1"
    
    # Validate configuration
    if ! echo "$containers_config" | jq empty 2>/dev/null; then
        log::error "Invalid containers configuration (not valid JSON)"
        return 1
    fi
    
    # Sort containers by priority
    local sorted_configs
    sorted_configs=$(echo "$containers_config" | jq 'sort_by(.priority // 10)')
    
    local container_count
    container_count=$(echo "$sorted_configs" | jq 'length')
    
    log::info "Starting container stack ($container_count containers)"
    
    # Start each container in order
    local i=0
    while [ $i -lt $container_count ]; do
        local container_config
        container_config=$(echo "$sorted_configs" | jq ".[$i]")
        
        local container_name
        container_name=$(echo "$container_config" | jq -r '.name')
        
        # Check dependencies
        if ! docker::_check_container_dependencies "$container_config"; then
            log::error "Dependencies not met for container: $container_name"
            return 1
        fi
        
        # Start container
        if ! docker::start_with_config "$container_config"; then
            log::error "Failed to start container in stack: $container_name"
            return 1
        fi
        
        i=$((i + 1))
    done
    
    log::success "Container stack started successfully"
    return 0
}

#######################################
# Stop multiple containers in reverse dependency order
# Args: $1 - containers_config (JSON array) or container_names (space-separated string)
#       $2 - cleanup_volumes (optional, "true" to remove volumes)
# Returns: 0 on success
#######################################
docker::stop_container_stack() {
    local config_or_names="$1"
    local cleanup_volumes="${2:-false}"
    
    local container_names=()
    
    # Handle both config JSON and simple container names
    if echo "$config_or_names" | jq empty 2>/dev/null; then
        # It's JSON config - extract names and reverse order for proper shutdown
        local sorted_configs
        sorted_configs=$(echo "$config_or_names" | jq 'sort_by(.priority // 10) | reverse')
        
        local container_count
        container_count=$(echo "$sorted_configs" | jq 'length')
        
        local i=0
        while [ $i -lt $container_count ]; do
            local name
            name=$(echo "$sorted_configs" | jq -r ".[$i].name")
            container_names+=("$name")
            i=$((i + 1))
        done
    else
        # It's space-separated container names
        read -ra container_names <<< "$config_or_names"
    fi
    
    log::info "Stopping container stack (${#container_names[@]} containers)"
    
    # Stop each container
    for container_name in "${container_names[@]}"; do
        if docker::is_running "$container_name"; then
            docker::stop_container "$container_name"
        fi
        
        if docker::container_exists "$container_name"; then
            docker::remove_container "$container_name" "true"
        fi
    done
    
    # Clean up volumes if requested
    if [[ "$cleanup_volumes" == "true" ]]; then
        for container_name in "${container_names[@]}"; do
            docker::_cleanup_container_volumes "$container_name"
        done
    fi
    
    log::success "Container stack stopped successfully"
    return 0
}

#######################################
# Get comprehensive container stack status
# Args: $1 - containers_config (JSON array) or container_names (space-separated)
# Returns: Status information via stdout
#######################################
docker::get_stack_status() {
    local config_or_names="$1"
    
    local container_names=()
    
    # Extract container names
    if echo "$config_or_names" | jq empty 2>/dev/null; then
        local container_count
        container_count=$(echo "$config_or_names" | jq 'length')
        
        local i=0
        while [ $i -lt $container_count ]; do
            local name
            name=$(echo "$config_or_names" | jq -r ".[$i].name")
            container_names+=("$name")
            i=$((i + 1))
        done
    else
        read -ra container_names <<< "$config_or_names"
    fi
    
    echo "Stack Status (${#container_names[@]} containers):"
    
    for container_name in "${container_names[@]}"; do
        local status="❌ Not found"
        
        if docker::container_exists "$container_name"; then
            if docker::is_running "$container_name"; then
                local health_status
                case $(docker::check_health "$container_name") in
                    0) health_status="✅ Healthy" ;;
                    1) health_status="⚠️  Unhealthy" ;;
                    *) health_status="❓ No health check" ;;
                esac
                status="🟢 Running ($health_status)"
            else
                status="🟡 Stopped"
            fi
        fi
        
        echo "  $container_name: $status"
    done
}

################################################################################
# INTERNAL HELPER FUNCTIONS
# Private functions used by the lifecycle management system
################################################################################

#######################################
# Internal: Validate container configuration
# Args: $1 - container_config (JSON)
# Returns: 0 if valid, 1 if invalid
#######################################
docker::_validate_container_config() {
    local config="$1"
    
    # Check if it's valid JSON
    if ! echo "$config" | jq empty 2>/dev/null; then
        return 1
    fi
    
    # Check required fields
    local required_fields=(".name" ".image")
    for field in "${required_fields[@]}"; do
        if ! echo "$config" | jq -e "$field" >/dev/null 2>&1; then
            log::error "Missing required field in container config: $field"
            return 1
        fi
    done
    
    return 0
}

#######################################
# Internal: Build docker run command from configuration
# Args: $1 - container_config (JSON)
# Returns: Docker run command via stdout
#######################################
docker::_build_run_command() {
    local config="$1"
    
    local container_name
    local image
    local network
    local restart_policy
    
    container_name=$(echo "$config" | jq -r '.name')
    image=$(echo "$config" | jq -r '.image')
    network=$(echo "$config" | jq -r '.network // empty')
    restart_policy=$(echo "$config" | jq -r '.restart_policy // "unless-stopped"')
    
    local cmd="docker run -d"
    cmd+=" --name $container_name"
    
    # Add network
    if [[ -n "$network" ]]; then
        cmd+=" --network $network"
    fi
    
    # Add restart policy
    cmd+=" --restart $restart_policy"
    
    # Add ports
    if echo "$config" | jq -e '.ports' >/dev/null 2>&1; then
        local ports
        ports=$(echo "$config" | jq -r '.ports | to_entries[] | "-p \(.value):\(.key)"' | tr '\n' ' ')
        cmd+=" $ports"
    fi
    
    # Add volumes
    if echo "$config" | jq -e '.volumes' >/dev/null 2>&1; then
        local volumes
        volumes=$(echo "$config" | jq -r '.volumes | to_entries[] | "-v \(.key):\(.value)"' | tr '\n' ' ')
        cmd+=" $volumes"
    fi
    
    # Add environment variables
    if echo "$config" | jq -e '.environment' >/dev/null 2>&1; then
        local env_vars
        env_vars=$(echo "$config" | jq -r '.environment | to_entries[] | "-e \(.key)=\(.value)"' | tr '\n' ' ')
        cmd+=" $env_vars"
    fi
    
    # Add extra arguments
    if echo "$config" | jq -e '.extra_args' >/dev/null 2>&1; then
        local extra_args
        extra_args=$(echo "$config" | jq -r '.extra_args[]' | tr '\n' ' ')
        cmd+=" $extra_args"
    fi
    
    # Add image
    cmd+=" $image"
    
    echo "$cmd"
}

#######################################
# Internal: Wait for container health using multiple methods
# Args: $1 - container_config (JSON)
# Returns: 0 if healthy, 1 if timeout
#######################################
docker::_wait_for_container_health() {
    local config="$1"
    
    local container_name
    local timeout
    local interval
    local endpoint
    
    container_name=$(echo "$config" | jq -r '.name')
    timeout=$(echo "$config" | jq -r '.health.timeout // 30')
    interval=$(echo "$config" | jq -r '.health.interval // 5')
    endpoint=$(echo "$config" | jq -r '.health.endpoint // empty')
    
    log::info "Waiting for container health: $container_name (timeout: ${timeout}s)"
    
    local elapsed=0
    
    while [ $elapsed -lt $timeout ]; do
        # Check Docker health first
        case $(docker::check_health "$container_name") in
            0)
                log::success "Container is healthy: $container_name"
                return 0
                ;;
            1)
                # Unhealthy - continue waiting
                ;;
            2)
                # No Docker health check - try HTTP endpoint if configured
                if [[ -n "$endpoint" ]]; then
                    local port
                    port=$(echo "$config" | jq -r '.ports | to_entries[0].value // empty')
                    if [[ -n "$port" ]]; then
                        local health_url="http://localhost:${port}${endpoint}"
                        if curl -f -s --max-time 5 "$health_url" >/dev/null 2>&1; then
                            log::success "Container is healthy (HTTP): $container_name"
                            return 0
                        fi
                    fi
                fi
                ;;
        esac
        
        sleep "$interval"
        elapsed=$((elapsed + interval))
        echo -n "."
    done
    
    echo
    log::warn "Container health check timeout after ${timeout}s: $container_name"
    return 1
}

#######################################
# Internal: Check if container dependencies are running
# Args: $1 - container_config (JSON)
# Returns: 0 if dependencies met, 1 if not
#######################################
docker::_check_container_dependencies() {
    local config="$1"
    
    if ! echo "$config" | jq -e '.depends_on' >/dev/null 2>&1; then
        return 0  # No dependencies
    fi
    
    local dependencies
    dependencies=$(echo "$config" | jq -r '.depends_on[]')
    
    for dep in $dependencies; do
        if ! docker::is_running "$dep"; then
            log::error "Dependency not running: $dep"
            return 1
        fi
    done
    
    return 0
}

#######################################
# Internal: Clean up volumes associated with container
# Args: $1 - container_name
# Returns: 0 on success
#######################################
docker::_cleanup_container_volumes() {
    local container_name="$1"
    
    # Get volumes associated with this container
    local volumes
    volumes=$(docker volume ls -q --filter "label=container=$container_name" 2>/dev/null || true)
    
    if [[ -n "$volumes" ]]; then
        log::info "Cleaning up volumes for container: $container_name"
        echo "$volumes" | xargs docker volume rm 2>/dev/null || true
    fi
    
    return 0
}

#######################################
# Remove network if it has no containers connected
# Args: $1 - network_name
# Returns: 0 on success, 1 on failure
#######################################
docker::remove_network_if_empty() {
    local network_name="$1"
    
    if ! docker network ls | grep -q "$network_name" 2>/dev/null; then
        return 0  # Network doesn't exist, nothing to do
    fi
    
    # Check if network has connected containers
    local connected_containers
    connected_containers=$(docker network inspect "$network_name" --format '{{len .Containers}}' 2>/dev/null || echo "0")
    
    if [[ "$connected_containers" == "0" ]]; then
        log::info "Removing empty network: $network_name"
        docker network rm "$network_name" 2>/dev/null || {
            log::warn "Failed to remove network: $network_name"
            return 1
        }
    else
        log::debug "Network $network_name has $connected_containers containers, not removing"
    fi
    
    return 0
}