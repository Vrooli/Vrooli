#!/usr/bin/env bash
# Simplified Docker Resource Utilities
# Ultra-thin API for resource container management

# Source guard - check if already sourced AND functions are available
if [[ -n "${_DOCKER_RESOURCE_UTILS_SOURCED:-}" ]]; then
    # Verify that required functions are actually available
    if type docker::check_daemon >/dev/null 2>&1; then
        return 0  # Already sourced and functions available
    fi
    # Guard was set but functions are missing - need to re-source dependencies
    unset _DOCKER_RESOURCE_UTILS_SOURCED _DOCKER_UTILS_SOURCED _SYSTEM_COMMANDS_SH_SOURCED
fi
export _DOCKER_RESOURCE_UTILS_SOURCED=1

# Source required utilities
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Source var.sh to get directory variables
# shellcheck disable=SC1091
if [[ -f "${SCRIPT_DIR}/../../lib/utils/var.sh" ]]; then
    source "${SCRIPT_DIR}/../../lib/utils/var.sh"
fi

# Source logging utilities
# shellcheck disable=SC1091
if [[ -f "${SCRIPT_DIR}/../../lib/utils/log.sh" ]]; then
    source "${SCRIPT_DIR}/../../lib/utils/log.sh"
fi

# Source system commands - try multiple paths
# shellcheck disable=SC1091
if [[ -n "${var_LIB_SYSTEM_DIR:-}" ]] && [[ -f "${var_LIB_SYSTEM_DIR}/system_commands.sh" ]]; then
    source "${var_LIB_SYSTEM_DIR}/system_commands.sh"
elif [[ -f "${SCRIPT_DIR}/../../lib/system/system_commands.sh" ]]; then
    source "${SCRIPT_DIR}/../../lib/system/system_commands.sh"
fi

# Source trash utilities - try multiple paths
# shellcheck disable=SC1091
if [[ -n "${var_LIB_SYSTEM_DIR:-}" ]] && [[ -f "${var_LIB_SYSTEM_DIR}/trash.sh" ]]; then
    source "${var_LIB_SYSTEM_DIR}/trash.sh"
elif [[ -f "${SCRIPT_DIR}/../../lib/system/trash.sh" ]]; then
    source "${SCRIPT_DIR}/../../lib/system/trash.sh"
fi

# Source docker-utils.sh
# shellcheck disable=SC1091
if [[ -f "${SCRIPT_DIR}/docker-utils.sh" ]]; then
    source "${SCRIPT_DIR}/docker-utils.sh"
else
    echo "ERROR: Cannot find docker-utils.sh" >&2
    return 1 2>/dev/null || exit 1
fi

################################################################################
# GENERIC RESOURCE HELPERS - Common patterns for all resources
################################################################################

#######################################
# Build standardized health check parameters
# Args: $1 - health_cmd, $2 - interval (optional), $3 - timeout (optional), $4 - retries (optional)
# Returns: Array of health check docker arguments via stdout
#######################################
docker_resource::build_health_check() {
    local health_cmd="$1"
    local interval="${2:-${DOCKER_HEALTH_INTERVAL:-30s}}"
    local timeout="${3:-${DOCKER_HEALTH_TIMEOUT:-10s}}"
    local retries="${4:-${DOCKER_HEALTH_RETRIES:-3}}"
    
    if [[ -n "$health_cmd" ]]; then
        echo "--health-cmd \"$health_cmd\" --health-interval $interval --health-timeout $timeout --health-retries $retries"
    fi
}

#######################################
# Show container logs with optional follow mode
# Args: $1 - container_name, $2 - lines (default 50), $3 - follow (true/false)
# Returns: 0 on success, 1 on failure
#######################################
docker_resource::show_logs_with_follow() {
    local container_name="$1"
    local lines="${2:-50}"
    local follow="${3:-false}"
    
    if ! docker::container_exists "$container_name"; then
        log::error "Container not found: $container_name"
        return 1
    fi
    
    if [[ "$follow" == "true" || "$follow" == "follow" ]]; then
        log::info "Following logs for $container_name (last $lines lines):"
        docker logs --timestamps --tail "$lines" --follow "$container_name"
    else
        docker::get_logs "$container_name" "$lines"
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
        # Use standardized health check builder
        local health_args
        health_args=$(docker_resource::build_health_check "$health_cmd" "30s" "5s" "3")
        # shellcheck disable=SC2086
        cmd+=($health_args)
    fi
    
    # Add image
    cmd+=("$image")
    
    # Execute command with error checking
    local output
    log::debug "Executing: ${cmd[*]}"
    if output=$("${cmd[@]}" 2>&1); then
        log::debug "Container created successfully with ID: ${output:0:12}"
        return 0
    else
        log::error "Failed to create service container: $output"
        log::debug "Failed command was: ${cmd[*]}"
        return 1
    fi
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
        # Use standardized health check builder
        local health_args
        health_args=$(docker_resource::build_health_check "$health_cmd" "30s" "5s" "3")
        # shellcheck disable=SC2086
        cmd+=($health_args)
    fi
    
    # Add image and command
    cmd+=("$image")
    if [[ ${#command_args[@]} -gt 0 ]]; then
        cmd+=("${command_args[@]}")
    fi
    
    # Execute command with error checking
    local output
    log::debug "Executing: ${cmd[*]}"
    if output=$("${cmd[@]}" 2>&1); then
        log::debug "Container created successfully with ID: ${output:0:12}"
        return 0
    else
        log::error "Failed to create service with command: $output"
        log::debug "Failed command was: ${cmd[*]}"
        return 1
    fi
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
    
    # Find available port atomically to prevent race conditions
    local client_port
    client_port=$(docker_resource::allocate_port_atomic "$port_start" "$port_end" "${resource_name}-${client_id}")
    
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
    
    # Get port before removing container (for cleanup)
    local client_port
    if docker::container_exists "$client_container"; then
        client_port=$(docker port "$client_container" 2>/dev/null | head -1 | cut -d':' -f2 | cut -d'-' -f1)
    fi
    
    # Remove container
    docker::remove_container "$client_container" "true"
    
    # Release the allocated port if we found it
    if [[ -n "$client_port" ]]; then
        docker_resource::release_port "$client_port" "${resource_name}-${client_id}"
    fi
    
    # Remove network if empty (ignore error if network is in use)
    if ! docker network rm "$client_network" >/dev/null 2>&1; then
        log::debug "Network $client_network may still be in use or already removed"
    fi
    
    # Remove data if requested
    if [[ "$remove_data" == "true" ]]; then
        local client_dir="${data_dir_base}/clients/${client_id}/${resource_name}"
        if [[ -d "$client_dir" ]]; then
            trash::safe_remove "$client_dir" --no-confirm 2>/dev/null || true
        fi
    fi
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
# Atomically allocate port with file locking to prevent race conditions
# Args: $1 - start_port, $2 - end_port, $3 - lock_id (optional, for identifying lock owner)
# Returns: Available port number via stdout, or empty if none found
#######################################
docker_resource::allocate_port_atomic() {
    local start_port="$1"
    local end_port="$2"
    local lock_id="${3:-$$}"  # Default to current PID
    local lock_dir="/tmp/vrooli_port_locks"
    local allocated_port=""
    
    # Create lock directory if it doesn't exist
    mkdir -p "$lock_dir" 2>/dev/null || true
    
    # Use a global lock file for port allocation
    local global_lock="${lock_dir}/global.lock"
    
    # Create lock file if it doesn't exist
    touch "$global_lock" 2>/dev/null || true
    
    # Try to acquire exclusive lock with timeout
    if command -v flock &>/dev/null; then
        # Use flock for atomic locking if available
        (
            flock -x -w 5 200 || {
                log::error "Failed to acquire port allocation lock"
                exit 1
            }
            
            # Find available port while holding lock
            for ((port=start_port; port<=end_port; port++)); do
                local port_lock="${lock_dir}/port_${port}.lock"
                
                # Check if port is available and not locked
                if docker::is_port_available "$port" && [[ ! -f "$port_lock" ]]; then
                    # Claim this port
                    echo "$lock_id" > "$port_lock"
                    allocated_port="$port"
                    break
                fi
            done
        ) 200>"$global_lock"
    else
        # Fallback to simple locking without flock
        log::warn "flock not available, using simple port allocation (may have race conditions)"
        allocated_port=$(docker_resource::find_available_port "$start_port" "$end_port")
    fi
    
    if [[ -n "$allocated_port" ]]; then
        log::debug "Allocated port $allocated_port for lock_id $lock_id"
        echo "$allocated_port"
        return 0
    else
        log::error "No available ports in range $start_port-$end_port"
        return 1
    fi
}

#######################################
# Release a previously allocated port
# Args: $1 - port_number, $2 - lock_id (optional, must match allocation)
# Returns: 0 on success, 1 on failure
#######################################
docker_resource::release_port() {
    local port="$1"
    local lock_id="${2:-$$}"
    local lock_dir="/tmp/vrooli_port_locks"
    local port_lock="${lock_dir}/port_${port}.lock"
    
    if [[ -f "$port_lock" ]]; then
        local current_owner
        current_owner=$(cat "$port_lock" 2>/dev/null || echo "")
        
        if [[ "$current_owner" == "$lock_id" ]] || [[ -z "$lock_id" ]]; then
            rm -f "$port_lock" 2>/dev/null || true
            log::debug "Released port $port (was locked by $current_owner)"
            return 0
        else
            log::warn "Cannot release port $port: owned by $current_owner, not $lock_id"
            return 1
        fi
    else
        log::debug "Port $port was not locked"
        return 0
    fi
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
        # Use standardized health check builder
        local health_args
        health_args=$(docker_resource::build_health_check "$health_cmd" "30s" "5s" "3")
        # shellcheck disable=SC2086
        cmd+=($health_args)
    fi
    
    # Add image
    cmd+=("$image")
    
    # Execute command with error checking
    local output
    log::debug "Executing: ${cmd[*]}"
    if output=$("${cmd[@]}" 2>&1); then
        log::debug "Container created successfully with ID: ${output:0:12}"
        return 0
    else
        log::error "Failed to create multi-port service: $output"
        log::debug "Failed command was: ${cmd[*]}"
        return 1
    fi
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
    if [[ -t 0 ]]; then
        docker exec -it "$container_name" "$@"
    else
        docker exec "$container_name" "$@"
    fi
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
    
    # Execute with error checking
    local output
    if output=$("${cmd[@]}" 2>&1); then
        echo "$allocated_ports"
        return 0
    else
        log::error "Failed to create multi-port client: $output"
        return 1
    fi
}

################################################################################
# ADVANCED CONTAINER CREATION - For complex Docker configurations
################################################################################

#######################################
# Create service with advanced Docker options and custom entrypoint
# Args: $1 - name, $2 - image, $3 - port_mappings, $4 - network,
#       $5 - volumes, $6 - env_vars (array name), $7 - docker_opts (array name),
#       $8 - health_cmd, $9 - entrypoint_cmd (array name)
# Returns: 0 on success, 1 on failure
#######################################
docker_resource::create_service_advanced() {
    local name="$1"
    local image="$2"
    local port_mappings="$3"
    local network="$4"
    local volumes="$5"
    local env_vars_name="$6"
    local docker_opts_name="$7"
    local health_cmd="$8"
    local entrypoint_name="$9"
    
    log::debug "Creating advanced service container: $name"
    
    # Validate Docker daemon
    if ! docker::check_daemon; then
        return 1
    fi
    
    # Pull image first with error checking
    if ! docker::pull_image "$image"; then
        log::error "Failed to pull image: $image"
        return 1
    fi
    
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
    
    # Add environment variables from array
    if [[ -n "$env_vars_name" ]]; then
        # Use eval for Bash 4.2 compatibility instead of nameref
        eval "local env_array=(\"\${${env_vars_name}[@]}\")"
        for env_var in "${env_array[@]}"; do
            cmd+=("-e" "$env_var")
        done
    fi
    
    # Add Docker options from array
    if [[ -n "$docker_opts_name" ]]; then
        # Use eval for Bash 4.2 compatibility instead of nameref
        eval "local opts_array=(\"\${${docker_opts_name}[@]}\")"
        cmd+=("${opts_array[@]}")
    fi
    
    # Add health check with configurable timeout
    if [[ -n "$health_cmd" ]]; then
        # Use standardized health check builder with environment variable defaults
        local health_args
        health_args=$(docker_resource::build_health_check "$health_cmd")
        # shellcheck disable=SC2086
        cmd+=($health_args)
    fi
    
    # Add image
    cmd+=("$image")
    
    # Add entrypoint/command from array
    if [[ -n "$entrypoint_name" ]]; then
        # Use eval for Bash 4.2 compatibility instead of nameref
        eval "local entrypoint_array=(\"\${${entrypoint_name}[@]}\")"
        cmd+=("${entrypoint_array[@]}")
    fi
    
    # Execute command with detailed error output
    local output
    log::debug "Executing: ${cmd[*]}"
    if output=$("${cmd[@]}" 2>&1); then
        log::debug "Container created successfully with ID: ${output:0:12}"
        return 0
    else
        log::error "Failed to create advanced service container: $output"
        log::debug "Failed command was: ${cmd[*]}"
        return 1
    fi
}

#######################################
# Allocate multiple ports atomically for multi-port services
# Args: $1 - num_ports, $2 - start_range, $3 - end_range
# Returns: Space-separated list of allocated ports via stdout
# Example: ports=$(docker_resource::allocate_port_range 3 9000 9100)
#######################################
docker_resource::allocate_port_range() {
    local num_ports="$1"
    local start_range="$2"
    local end_range="$3"
    
    local allocated_ports=()
    local port="$start_range"
    
    while [[ ${#allocated_ports[@]} -lt $num_ports ]] && [[ $port -le $end_range ]]; do
        if docker::is_port_available "$port"; then
            allocated_ports+=("$port")
        fi
        ((port++))
    done
    
    if [[ ${#allocated_ports[@]} -eq $num_ports ]]; then
        echo "${allocated_ports[*]}"
        return 0
    else
        log::error "Could not allocate $num_ports ports in range $start_range-$end_range"
        return 1
    fi
}

#######################################
# Merge JSON objects using jq safely
# Args: $1 - base_json, $2 - additional_json
# Returns: Merged JSON via stdout
#######################################
docker_resource::merge_json() {
    local base_json="$1"
    local additional_json="$2"
    
    if command -v jq &>/dev/null; then
        echo "$base_json" | jq --argjson add "$additional_json" '. + $add'
    else
        log::error "jq is required for JSON operations"
        return 1
    fi
}

#######################################
# Validate required environment variables
# Args: Variable names to check
# Returns: 0 if all present, 1 if any missing
#######################################
docker_resource::validate_env_vars() {
    local missing=()
    
    for var_name in "$@"; do
        if [[ -z "${!var_name:-}" ]]; then
            missing+=("$var_name")
        fi
    done
    
    if [[ ${#missing[@]} -gt 0 ]]; then
        log::error "Missing required environment variables: ${missing[*]}"
        return 1
    fi
    
    return 0
}

################################################################################
# DOCKER COMPOSE UTILITIES - For multi-service orchestration
################################################################################

#######################################
# Run docker-compose command with standard options
# Args: $1 - compose_file, $@ - compose arguments
# Returns: Compose command exit code
#######################################
docker_resource::compose_cmd() {
    local compose_file="$1"
    shift
    
    if [[ ! -f "$compose_file" ]]; then
        log::error "Docker Compose file not found: $compose_file"
        return 1
    fi
    
    # Check for docker-compose or docker compose command
    local compose_cmd
    if command -v docker-compose &>/dev/null; then
        compose_cmd="docker-compose"
    elif docker compose version &>/dev/null 2>&1; then
        compose_cmd="docker compose"
    else
        log::error "Docker Compose is not installed"
        return 1
    fi
    
    # Execute compose command
    $compose_cmd -f "$compose_file" "$@"
}

#######################################
# Start services using docker-compose
# Args: $1 - compose_file, $2 - profiles (optional)
# Returns: 0 on success, 1 on failure
#######################################
docker_resource::compose_up() {
    local compose_file="$1"
    local profiles="${2:-}"
    
    log::info "Starting services with Docker Compose..."
    
    local cmd_args=("up" "-d")
    if [[ -n "$profiles" ]]; then
        for profile in $profiles; do
            cmd_args=("--profile" "$profile" "${cmd_args[@]}")
        done
    fi
    
    if docker_resource::compose_cmd "$compose_file" "${cmd_args[@]}"; then
        log::success "Services started successfully"
        return 0
    else
        log::error "Failed to start services"
        return 1
    fi
}

#######################################
# Stop services using docker-compose
# Args: $1 - compose_file, $2 - remove_volumes (true/false)
# Returns: 0 on success, 1 on failure
#######################################
docker_resource::compose_down() {
    local compose_file="$1"
    local remove_volumes="${2:-false}"
    
    log::info "Stopping services with Docker Compose..."
    
    local cmd_args=("down")
    if [[ "$remove_volumes" == "true" ]]; then
        cmd_args+=("-v")
    fi
    
    if docker_resource::compose_cmd "$compose_file" "${cmd_args[@]}"; then
        log::success "Services stopped successfully"
        return 0
    else
        log::error "Failed to stop services"
        return 1
    fi
}

#######################################
# Restart services using docker-compose
# Args: $1 - compose_file, $2 - service_name (optional)
# Returns: 0 on success, 1 on failure
#######################################
docker_resource::compose_restart() {
    local compose_file="$1"
    local service_name="${2:-}"
    
    log::info "Restarting services with Docker Compose..."
    
    local cmd_args=("restart")
    if [[ -n "$service_name" ]]; then
        cmd_args+=("$service_name")
    fi
    
    if docker_resource::compose_cmd "$compose_file" "${cmd_args[@]}"; then
        log::success "Services restarted successfully"
        return 0
    else
        log::error "Failed to restart services"
        return 1
    fi
}

#######################################
# Scale service to specified count
# Args: $1 - compose_file, $2 - service_name, $3 - count
# Returns: 0 on success, 1 on failure
#######################################
docker_resource::compose_scale() {
    local compose_file="$1"
    local service_name="$2"
    local count="$3"
    
    if ! [[ "$count" =~ ^[0-9]+$ ]]; then
        log::error "Invalid count: $count (must be a number)"
        return 1
    fi
    
    log::info "Scaling $service_name to $count instances..."
    
    if docker_resource::compose_cmd "$compose_file" scale "$service_name=$count"; then
        log::success "Service scaled successfully"
        return 0
    else
        log::error "Failed to scale service"
        return 1
    fi
}

#######################################
# Get compose service logs
# Args: $1 - compose_file, $2 - service_name (optional), $3 - lines (optional)
# Returns: 0 on success, 1 on failure
#######################################
docker_resource::compose_logs() {
    local compose_file="$1"
    local service_name="${2:-}"
    local lines="${3:-50}"
    
    local cmd_args=("logs" "--tail=$lines")
    if [[ -n "$service_name" ]]; then
        cmd_args+=("$service_name")
    fi
    
    docker_resource::compose_cmd "$compose_file" "${cmd_args[@]}"
}

#######################################
# Execute command in database container with automatic CLI detection
# Args: $1 - container_name, $2 - db_type (postgres/mysql/redis/mongo), $3 - command
# Returns: Command exit code
#######################################
docker_resource::exec_database_command() {
    local container_name="$1"
    local db_type="$2"
    local command="$3"
    
    if ! docker::is_running "$container_name"; then
        log::error "Container is not running: $container_name"
        return 1
    fi
    
    case "$db_type" in
        postgres)
            docker exec "$container_name" psql -U "${POSTGRES_USER:-postgres}" -c "$command"
            ;;
        mysql)
            docker exec "$container_name" mysql -u "${MYSQL_USER:-root}" -p"${MYSQL_PASSWORD:-}" -e "$command"
            ;;
        redis)
            if [[ -n "${REDIS_PASSWORD:-}" ]]; then
                docker exec "$container_name" redis-cli -a "$REDIS_PASSWORD" "$command"
            else
                docker exec "$container_name" redis-cli "$command"
            fi
            ;;
        mongo)
            docker exec "$container_name" mongosh --eval "$command"
            ;;
        *)
            log::error "Unknown database type: $db_type"
            return 1
            ;;
    esac
}

#######################################
# Get database-specific environment variables
# Args: $1 - db_type, $2 - password
# Returns: Environment variable arguments via stdout
#######################################
docker_resource::get_database_env_vars() {
    local db_type="$1"
    local password="${2:-}"
    
    case "$db_type" in
        postgres)
            echo "-e POSTGRES_PASSWORD=${password:-postgres} -e POSTGRES_USER=${POSTGRES_USER:-postgres} -e POSTGRES_DB=${POSTGRES_DB:-postgres}"
            ;;
        mysql)
            echo "-e MYSQL_ROOT_PASSWORD=${password:-mysql} -e MYSQL_DATABASE=${MYSQL_DATABASE:-mysql}"
            ;;
        redis)
            if [[ -n "$password" ]]; then
                echo "-e REDIS_PASSWORD=$password"
            fi
            ;;
        mongo)
            if [[ -n "$password" ]]; then
                echo "-e MONGO_INITDB_ROOT_PASSWORD=$password -e MONGO_INITDB_ROOT_USERNAME=${MONGO_USER:-admin}"
            fi
            ;;
        *)
            log::warn "Unknown database type for env vars: $db_type"
            ;;
    esac
}

#######################################
# Create database client instance with unified pattern
# Args: $1 - db_type, $2 - client_id, $3 - image, $4 - internal_port, 
#       $5 - port_start, $6 - port_end, $7 - password
# Returns: Allocated port via stdout, 1 on failure
#######################################
docker_resource::create_database_client_instance() {
    local db_type="$1"
    local client_id="$2"
    local image="$3"
    local internal_port="$4"
    local port_start="$5"
    local port_end="$6"
    local password="${7:-}"
    
    local client_container="${db_type}-client-${client_id}"
    local client_network="vrooli-${client_id}-network"
    
    # Find available port atomically to prevent race conditions
    local client_port
    client_port=$(docker_resource::allocate_port_atomic "$port_start" "$port_end" "${db_type}-${client_id}")
    
    if [[ -z "$client_port" ]]; then
        log::error "No available ports in range $port_start-$port_end"
        return 1
    fi
    
    # Create network
    docker::create_network "$client_network"
    
    # Get database-specific environment variables
    local env_vars
    env_vars=$(docker_resource::get_database_env_vars "$db_type" "$password")
    
    # Create container
    local cmd=("docker" "run" "-d")
    cmd+=("--name" "$client_container")
    cmd+=("--network" "$client_network")
    cmd+=("-p" "${client_port}:${internal_port}")
    cmd+=("--restart" "unless-stopped")
    
    # Add environment variables if any
    if [[ -n "$env_vars" ]]; then
        # shellcheck disable=SC2086
        cmd+=($env_vars)
    fi
    
    cmd+=("$image")
    
    # Execute with error checking
    local output
    log::debug "Creating database client: ${cmd[*]}"
    if output=$("${cmd[@]}" 2>&1); then
        log::debug "Database client created with ID: ${output:0:12}"
        echo "$client_port"
        return 0
    else
        log::error "Failed to create $db_type client instance: $output"
        log::debug "Failed command was: ${cmd[*]}"
        return 1
    fi
}