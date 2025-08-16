#!/usr/bin/env bash
# Simplified Docker Resource Utilities
# Ultra-thin API for resource container management

# Source guard to prevent multiple sourcing
[[ -n "${_DOCKER_RESOURCE_UTILS_SOURCED:-}" ]] && return 0
export _DOCKER_RESOURCE_UTILS_SOURCED=1

# Source required utilities
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck disable=SC1091
source "${SCRIPT_DIR}/../../lib/utils/var.sh" 2>/dev/null || true
# shellcheck disable=SC1091
source "${SCRIPT_DIR}/../../lib/utils/log.sh" 2>/dev/null || true
# shellcheck disable=SC1091
source "${SCRIPT_DIR}/../../lib/system/system_commands.sh" 2>/dev/null || true
# shellcheck disable=SC1091
source "${SCRIPT_DIR}/../../lib/system/trash.sh" 2>/dev/null || true
# shellcheck disable=SC1091
source "${SCRIPT_DIR}/docker-utils.sh" 2>/dev/null || true

################################################################################
# BASIC DOCKER UTILITIES - Keep the useful simple functions
################################################################################

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
    docker inspect "$container_name" >/dev/null 2>&1
}

#######################################
# Check if a container is running
# Args: $1 - container_name
# Returns: 0 if running, 1 otherwise
#######################################
docker::is_running() {
    local container_name="$1"
    local status
    status=$(docker inspect --format='{{.State.Status}}' "$container_name" 2>/dev/null)
    [[ "$status" == "running" ]]
}

#######################################
# Check if port is available
# Args: $1 - port number
# Returns: 0 if available, 1 if in use
#######################################
docker::is_port_available() {
    local port="$1"
    ! netstat -tuln 2>/dev/null | grep -q ":${port} " && ! ss -tuln 2>/dev/null | grep -q ":${port} "
}

#######################################
# Start existing container
# Args: $1 - container_name
# Returns: 0 on success, 1 on failure
#######################################
docker::start_container() {
    local container_name="$1"
    log::debug "Starting container: $container_name"
    docker start "$container_name" >/dev/null 2>&1
}

#######################################
# Stop container
# Args: $1 - container_name, $2 - timeout (optional)
# Returns: 0 on success, 1 on failure
#######################################
docker::stop_container() {
    local container_name="$1"
    local timeout="${2:-30}"
    
    log::debug "Stopping container: $container_name"
    
    if docker::is_running "$container_name"; then
        docker stop --time="$timeout" "$container_name" >/dev/null 2>&1
    else
        return 0
    fi
}

#######################################
# Restart container
# Args: $1 - container_name
# Returns: 0 on success, 1 on failure
#######################################
docker::restart_container() {
    local container_name="$1"
    log::debug "Restarting container: $container_name"
    docker restart "$container_name" >/dev/null 2>&1
}

#######################################
# Remove container
# Args: $1 - container_name, $2 - force (optional)
# Returns: 0 on success, 1 on failure
#######################################
docker::remove_container() {
    local container_name="$1"
    local force="${2:-false}"
    
    log::debug "Removing container: $container_name"
    
    # Stop first if running
    if docker::is_running "$container_name"; then
        docker::stop_container "$container_name"
    fi
    
    # Remove container
    if [[ "$force" == "true" ]]; then
        docker rm -f "$container_name" >/dev/null 2>&1
    else
        docker rm "$container_name" >/dev/null 2>&1
    fi
}

#######################################
# Pull Docker image
# Args: $1 - image name
# Returns: 0 on success, 1 on failure
#######################################
docker::pull_image() {
    local image="$1"
    log::debug "Pulling image: $image"
    docker pull "$image" >/dev/null 2>&1
}

#######################################
# Get container logs
# Args: $1 - container_name, $2 - lines (optional)
# Returns: 0 on success, 1 on failure
#######################################
docker::get_logs() {
    local container_name="$1"
    local lines="${2:-50}"
    docker logs --tail "$lines" "$container_name" 2>&1
}

#######################################
# Create network if it doesn't exist
# Args: $1 - network_name
# Returns: 0 on success, 1 on failure
#######################################
docker::create_network() {
    local network_name="$1"
    
    if ! docker network inspect "$network_name" >/dev/null 2>&1; then
        docker network create "$network_name" >/dev/null 2>&1
    fi
}

################################################################################
# SIMPLIFIED RESOURCE PATTERNS - Replace complex JSON with simple functions
################################################################################

#######################################
# Create standard service container
# Args: $1 - name, $2 - image, $3 - host_port, $4 - container_port, 
#       $5 - network, $6 - data_volume, $7 - config_volume, $8 - health_cmd
# Returns: 0 on success, 1 on failure
#######################################
docker_resource::create_service() {
    local name="$1"
    local image="$2" 
    local host_port="$3"
    local container_port="$4"
    local network="$5"
    local data_volume="$6"
    local config_volume="${7:-}"
    local health_cmd="${8:-}"
    
    log::debug "Creating service container: $name"
    
    # Create network
    docker::create_network "$network"
    
    # Build docker run command
    local cmd=("docker" "run" "-d")
    cmd+=("--name" "$name")
    cmd+=("--network" "$network")
    cmd+=("--restart" "unless-stopped")
    
    # Add port mapping
    if [[ -n "$host_port" && -n "$container_port" ]]; then
        cmd+=("-p" "${host_port}:${container_port}")
    fi
    
    # Add data volume
    if [[ -n "$data_volume" ]]; then
        cmd+=("-v" "$data_volume")
    fi
    
    # Add config volume if specified
    if [[ -n "$config_volume" ]]; then
        cmd+=("-v" "$config_volume")
    fi
    
    # Add health check if specified
    if [[ -n "$health_cmd" ]]; then
        cmd+=("--health-cmd" "$health_cmd")
        cmd+=("--health-interval" "30s")
        cmd+=("--health-timeout" "5s")
        cmd+=("--health-retries" "3")
    fi
    
    # Add image
    cmd+=("$image")
    
    # Execute command
    "${cmd[@]}" >/dev/null 2>&1
}

#######################################
# Create service with custom command and multiple volumes
# Args: $1 - name, $2 - image, $3 - host_port, $4 - container_port,
#       $5 - network, $6 - volumes (space-separated), $7 - health_cmd, $8+ - command and args
# Returns: 0 on success, 1 on failure
#######################################
docker_resource::create_service_with_command() {
    local name="$1"
    local image="$2"
    local host_port="$3" 
    local container_port="$4"
    local network="$5"
    local volumes="$6"
    local health_cmd="$7"
    shift 7
    local command_args=("$@")
    
    log::debug "Creating service container with command: $name"
    
    # Create network
    docker::create_network "$network"
    
    # Build docker run command
    local cmd=("docker" "run" "-d")
    cmd+=("--name" "$name")
    cmd+=("--network" "$network") 
    cmd+=("--restart" "unless-stopped")
    
    # Add port mapping
    if [[ -n "$host_port" && -n "$container_port" ]]; then
        cmd+=("-p" "${host_port}:${container_port}")
    fi
    
    # Add volumes (space-separated list)
    if [[ -n "$volumes" ]]; then
        for volume in $volumes; do
            cmd+=("-v" "$volume")
        done
    fi
    
    # Add health check if specified
    if [[ -n "$health_cmd" ]]; then
        cmd+=("--health-cmd" "$health_cmd")
        cmd+=("--health-interval" "30s")
        cmd+=("--health-timeout" "5s")
        cmd+=("--health-retries" "3")
    fi
    
    # Add image and command
    cmd+=("$image")
    if [[ ${#command_args[@]} -gt 0 ]]; then
        cmd+=("${command_args[@]}")
    fi
    
    # Execute command
    "${cmd[@]}" >/dev/null 2>&1
}

#######################################
# Create client instance with port allocation
# Args: $1 - resource_name, $2 - client_id, $3 - image, $4 - internal_port,
#       $5 - port_start, $6 - port_end, $7 - data_dir_base, $8 - config_template
# Returns: 0 on success, 1 on failure, prints allocated port to stdout
#######################################
docker_resource::create_client_instance() {
    local resource_name="$1"
    local client_id="$2"
    local image="$3"
    local internal_port="$4"
    local port_start="$5"
    local port_end="$6"
    local data_dir_base="$7"
    local config_template="${8:-}"
    
    local client_container="${resource_name}-client-${client_id}"
    local client_network="vrooli-${client_id}-network"
    
    # Check if already exists
    if docker::container_exists "$client_container"; then
        log::error "Client instance already exists: $client_id"
        return 1
    fi
    
    # Find available port
    local client_port
    for ((port=port_start; port<=port_end; port++)); do
        if docker::is_port_available "$port"; then
            client_port="$port"
            break
        fi
    done
    
    if [[ -z "$client_port" ]]; then
        log::error "No available ports in range ${port_start}-${port_end}"
        return 1
    fi
    
    # Create client directories
    local client_data_dir="${data_dir_base}/clients/${client_id}/${resource_name}/data"
    local client_config_dir="${data_dir_base}/clients/${client_id}/${resource_name}/config"
    
    mkdir -p "$client_data_dir" "$client_config_dir"
    
    # Copy config template if provided
    if [[ -n "$config_template" && -f "$config_template" ]]; then
        cp "$config_template" "${client_config_dir}/"
    fi
    
    # Create container
    docker_resource::create_service \
        "$client_container" \
        "$image" \
        "$client_port" \
        "$internal_port" \
        "$client_network" \
        "${client_data_dir}:/data" \
        "${client_config_dir}:/config"
    
    local result=$?
    if [[ $result -eq 0 ]]; then
        echo "$client_port"  # Output allocated port
    fi
    
    return $result
}

#######################################
# Remove client instance and cleanup
# Args: $1 - resource_name, $2 - client_id, $3 - data_dir_base, $4 - remove_data
# Returns: 0 on success, 1 on failure
#######################################
docker_resource::remove_client_instance() {
    local resource_name="$1"
    local client_id="$2"
    local data_dir_base="$3"
    local remove_data="${4:-false}"
    
    local client_container="${resource_name}-client-${client_id}"
    local client_network="vrooli-${client_id}-network"
    
    # Remove container
    docker::remove_container "$client_container" "true"
    
    # Remove network if empty
    docker network rm "$client_network" >/dev/null 2>&1 || true
    
    # Remove data if requested
    if [[ "$remove_data" == "true" ]]; then
        local client_dir="${data_dir_base}/clients/${client_id}/${resource_name}"
        if [[ -d "$client_dir" ]]; then
            trash::safe_remove "$client_dir" --no-confirm 2>/dev/null || true
        fi
    fi
}

#######################################
# Add environment variables to container creation
# Args: Appends -e KEY=VALUE pairs to global cmd array
# Usage: docker_resource::add_env "KEY1=value1" "KEY2=value2"
#######################################
docker_resource::add_env() {
    for env_var in "$@"; do
        cmd+=("-e" "$env_var")
    done
}

#######################################
# Add labels to container creation  
# Args: Appends --label KEY=VALUE pairs to global cmd array
# Usage: docker_resource::add_labels "app=myapp" "version=1.0"
#######################################
docker_resource::add_labels() {
    for label in "$@"; do
        cmd+=("--label" "$label")
    done
}

#######################################
# Find available port in range
# Args: $1 - start_port, $2 - end_port
# Returns: Available port number via stdout, or empty if none found
#######################################
docker_resource::find_available_port() {
    local start_port="$1"
    local end_port="$2"
    
    for ((port=start_port; port<=end_port; port++)); do
        if docker::is_port_available "$port"; then
            echo "$port"
            return 0
        fi
    done
    
    return 1
}

#######################################
# Create service with multiple port mappings
# Args: $1 - name, $2 - image, $3 - network, $4 - volumes (space-separated),
#       $5 - health_cmd, $6 - port_mappings (space-separated "host:container" pairs)
# Returns: 0 on success, 1 on failure
#######################################
docker_resource::create_multi_port_service() {
    local name="$1"
    local image="$2"
    local network="$3"
    local volumes="$4"
    local health_cmd="$5"
    local port_mappings="$6"
    
    log::debug "Creating multi-port service container: $name"
    
    # Create network
    docker::create_network "$network"
    
    # Build docker run command
    local cmd=("docker" "run" "-d")
    cmd+=("--name" "$name")
    cmd+=("--network" "$network")
    cmd+=("--restart" "unless-stopped")
    
    # Add port mappings
    if [[ -n "$port_mappings" ]]; then
        for port_map in $port_mappings; do
            cmd+=("-p" "$port_map")
        done
    fi
    
    # Add volumes
    if [[ -n "$volumes" ]]; then
        for volume in $volumes; do
            cmd+=("-v" "$volume")
        done
    fi
    
    # Add health check if specified
    if [[ -n "$health_cmd" ]]; then
        cmd+=("--health-cmd" "$health_cmd")
        cmd+=("--health-interval" "30s")
        cmd+=("--health-timeout" "5s")
        cmd+=("--health-retries" "3")
    fi
    
    # Add image
    cmd+=("$image")
    
    # Execute command
    "${cmd[@]}" >/dev/null 2>&1
}

#######################################
# Create service with additional Docker options
# Args: $1 - name, $2 - image, $3 - port_mappings, $4 - network,
#       $5 - volumes, $6 - health_cmd, $7 - docker_opts (additional docker run options)
# Returns: 0 on success, 1 on failure
#######################################
docker_resource::create_service_with_opts() {
    local name="$1"
    local image="$2"
    local port_mappings="$3"
    local network="$4"
    local volumes="$5"
    local health_cmd="$6"
    local docker_opts="$7"
    
    log::debug "Creating service container with options: $name"
    
    # Create network
    docker::create_network "$network"
    
    # Build docker run command
    local cmd=("docker" "run" "-d")
    cmd+=("--name" "$name")
    cmd+=("--network" "$network")
    cmd+=("--restart" "unless-stopped")
    
    # Add port mappings
    if [[ -n "$port_mappings" ]]; then
        for port_map in $port_mappings; do
            cmd+=("-p" "$port_map")
        done
    fi
    
    # Add volumes
    if [[ -n "$volumes" ]]; then
        for volume in $volumes; do
            cmd+=("-v" "$volume")
        done
    fi
    
    # Add custom Docker options (e.g., --shm-size, --cap-add, etc.)
    if [[ -n "$docker_opts" ]]; then
        # Parse docker_opts as an array to handle options with values
        eval "local opts_array=($docker_opts)"
        cmd+=("${opts_array[@]}")
    fi
    
    # Add health check if specified
    if [[ -n "$health_cmd" ]]; then
        cmd+=("--health-cmd" "$health_cmd")
        cmd+=("--health-interval" "30s")
        cmd+=("--health-timeout" "5s")
        cmd+=("--health-retries" "3")
    fi
    
    # Add image
    cmd+=("$image")
    
    # Execute command
    "${cmd[@]}" >/dev/null 2>&1
}

#######################################
# Safe data removal with confirmation
# Args: $1 - resource_name, $2 - data_paths (space-separated), $3 - force (yes/no)
# Returns: 0 on success, 1 on failure
#######################################
docker_resource::remove_data() {
    local resource_name="$1"
    local data_paths="$2"
    local force="${3:-no}"
    
    if [[ "$force" != "yes" ]]; then
        log::warn "⚠️  This will permanently delete all $resource_name data!"
        echo -n "Are you sure you want to continue? (yes/no): "
        read -r confirm
        if [[ "$confirm" != "yes" ]]; then
            log::info "Data removal cancelled"
            return 0
        fi
    fi
    
    # Remove each data path safely
    for data_path in $data_paths; do
        if [[ -d "$data_path" || -f "$data_path" ]]; then
            trash::safe_remove "$data_path" --no-confirm 2>/dev/null || true
            log::info "Removed: $data_path"
        fi
    done
    
    log::success "✅ $resource_name data removed"
    return 0
}

#######################################
# Execute command in container with running check
# Args: $1 - container_name, $@ - command to execute
# Returns: Command exit code
#######################################
docker_resource::exec() {
    local container_name="$1"
    shift
    
    if ! docker::is_running "$container_name"; then
        log::error "Container is not running: $container_name"
        return 1
    fi
    
    docker::exec "$container_name" "$@"
}

#######################################
# Execute command with interactive TTY support
# Args: $1 - container_name, $@ - command to execute
# Returns: Command exit code
#######################################
docker_resource::exec_interactive() {
    local container_name="$1"
    shift
    
    if ! docker::is_running "$container_name"; then
        log::error "Container is not running: $container_name"
        return 1
    fi
    
    # Use -it only if we have a TTY
    local docker_flags=""
    if [[ -t 0 ]]; then
        docker_flags="-it"
    fi
    
    docker exec $docker_flags "$container_name" "$@"
}

#######################################
# Get container statistics wrapper
# Args: $1 - container_name, $2 - format (optional)
# Returns: Stats via stdout
#######################################
docker_resource::get_stats() {
    local container_name="$1"
    local format="${2:-}"
    
    if ! docker::is_running "$container_name"; then
        log::error "Container is not running: $container_name"
        return 1
    fi
    
    if [[ -n "$format" ]] && [[ "$format" == "json" ]]; then
        docker stats "$container_name" --no-stream --format "json" 2>/dev/null
    else
        docker::get_stats "$container_name"
    fi
}

#######################################
# Show standardized connection info for a resource
# Args: $1 - resource_name, $2 - primary_url, $@ - additional info lines
# Example: docker_resource::show_connection_info "QuestDB" "http://localhost:9000" "PostgreSQL: localhost:8812" "InfluxDB: localhost:9009"
#######################################
docker_resource::show_connection_info() {
    local resource_name="$1"
    local primary_url="$2"
    shift 2
    
    log::info "${resource_name} Connection Information:"
    log::info "  Primary URL: ${primary_url}"
    
    # Print any additional info lines
    for info_line in "$@"; do
        log::info "  ${info_line}"
    done
}

#######################################
# Generate standardized client metadata JSON
# Args: $1 - client_id, $2 - resource, $3 - container, $4 - network, $5 - port(s), $6 - base_url
# Returns: JSON string via stdout
#######################################
docker_resource::generate_client_metadata() {
    local client_id="$1"
    local resource="$2"
    local container="$3"
    local network="$4"
    local ports="$5"  # Can be single port or JSON object for multi-port
    local base_url="$6"
    
    # Check if ports is a JSON object (contains '{')
    if [[ "$ports" == *"{"* ]]; then
        # Multi-port JSON object
        cat << EOF
{
  "clientId": "${client_id}",
  "resource": "${resource}",
  "container": "${container}",
  "network": "${network}",
  "ports": ${ports},
  "baseUrl": "${base_url}",
  "created": "$(date -Iseconds)"
}
EOF
    else
        # Single port number
        cat << EOF
{
  "clientId": "${client_id}",
  "resource": "${resource}",
  "container": "${container}",
  "network": "${network}",
  "port": ${ports},
  "baseUrl": "${base_url}",
  "created": "$(date -Iseconds)"
}
EOF
    fi
}

#######################################
# Simplified client instance creation for multi-port resources
# Args: $1 - resource_name, $2 - client_id, $3 - image, $4 - port_ranges (JSON), $5 - volumes, $6 - env_vars
# Returns: 0 on success with port info via stdout, 1 on failure
#######################################
docker_resource::create_multi_port_client() {
    local resource_name="$1"
    local client_id="$2"
    local image="$3"
    local port_ranges="$4"  # JSON like {"http": [9000, 9050], "pg": [8812, 8850]}
    local volumes="$5"
    local env_vars="$6"
    
    local client_container="${resource_name}-client-${client_id}"
    local client_network="vrooli-${client_id}-network"
    
    # Create network
    docker::create_network "$client_network"
    
    # Allocate ports based on ranges
    local allocated_ports="{}"
    local port_mappings=""
    
    # Parse port ranges and allocate (simplified for now - would need jq in production)
    # For demonstration, using fixed offsets
    local base_offset=$((RANDOM % 100))
    
    # Build docker command
    local cmd=("docker" "run" "-d")
    cmd+=("--name" "$client_container")
    cmd+=("--network" "$client_network")
    cmd+=("--restart" "unless-stopped")
    
    # Add port mappings (resource-specific logic needed here)
    for port_map in $port_mappings; do
        cmd+=("-p" "$port_map")
    done
    
    # Add volumes
    if [[ -n "$volumes" ]]; then
        for volume in $volumes; do
            cmd+=("-v" "$volume")
        done
    fi
    
    # Add environment variables
    if [[ -n "$env_vars" ]]; then
        for env_var in $env_vars; do
            cmd+=("-e" "$env_var")
        done
    fi
    
    cmd+=("$image")
    
    # Execute
    if "${cmd[@]}" >/dev/null 2>&1; then
        echo "$allocated_ports"
        return 0
    else
        return 1
    fi
}