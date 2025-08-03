#!/usr/bin/env bash
# Resource Manager - Service lifecycle and dependency management
# Handles starting, stopping, health checking, and tracking test resources

# Prevent duplicate loading
if [[ "${VROOLI_RESOURCE_MANAGER_LOADED:-}" == "true" ]]; then
    return 0
fi
export VROOLI_RESOURCE_MANAGER_LOADED="true"

# Load dependencies
SHARED_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SHARED_DIR/logging.bash"
source "$SHARED_DIR/utils.bash"
source "$SHARED_DIR/port-manager.bash"

# Resource tracking
declare -A MANAGED_RESOURCES=()
declare -A RESOURCE_DEPENDENCIES=()
declare -A RESOURCE_HEALTH_CHECKS=()
readonly RESOURCE_STATE_DIR="${VROOLI_RESOURCE_STATE_DIR:-/tmp/vrooli-resource-state}"

#######################################
# Initialize resource manager
# Arguments: None
# Returns: 0 on success
#######################################
vrooli_resource_init() {
    vrooli_log_debug "Initializing resource manager"
    
    # Create state directory
    if ! mkdir -p "$RESOURCE_STATE_DIR" 2>/dev/null; then
        vrooli_log_error "Failed to create resource state directory: $RESOURCE_STATE_DIR"
        return 1
    fi
    
    # Clean up old state files (older than 1 hour)
    find "$RESOURCE_STATE_DIR" -name "*.state" -mmin +60 -delete 2>/dev/null || true
    
    vrooli_log_info "Resource manager initialized"
    return 0
}

#######################################
# Register a resource for management
# Arguments:
#   $1 - Resource name
#   $2 - Resource type (docker|process|service)
#   $3 - Start command/container name
#   $4 - Dependencies (comma-separated, optional)
# Returns: 0 on success
#######################################
vrooli_resource_register() {
    local name="$1"
    local type="$2"
    local start_cmd="$3"
    local deps="${4:-}"
    
    vrooli_log_info "Registering resource: $name (type: $type)"
    
    # Store resource information
    MANAGED_RESOURCES["$name"]="$type:$start_cmd"
    
    # Store dependencies if provided
    if [[ -n "$deps" ]]; then
        RESOURCE_DEPENDENCIES["$name"]="$deps"
        vrooli_log_debug "Resource $name depends on: $deps"
    fi
    
    # Create state file
    echo "registered:$(date +%s)" > "$RESOURCE_STATE_DIR/$name.state"
    
    return 0
}

#######################################
# Register a health check for a resource
# Arguments:
#   $1 - Resource name
#   $2 - Health check type (http|tcp|process|custom)
#   $3 - Health check details (URL/port/command)
# Returns: 0 on success
#######################################
vrooli_resource_register_health_check() {
    local name="$1"
    local check_type="$2"
    local check_details="$3"
    
    RESOURCE_HEALTH_CHECKS["$name"]="$check_type:$check_details"
    vrooli_log_debug "Registered health check for $name: $check_type ($check_details)"
    
    return 0
}

#######################################
# Start a resource with dependency resolution
# Arguments:
#   $1 - Resource name
# Returns: 0 on success, 1 on failure
#######################################
vrooli_resource_start() {
    local name="$1"
    
    # Check if resource is registered
    if [[ -z "${MANAGED_RESOURCES[$name]:-}" ]]; then
        vrooli_log_error "Resource not registered: $name"
        return 1
    fi
    
    # Check if already running
    if vrooli_resource_is_running "$name"; then
        vrooli_log_info "Resource $name is already running"
        return 0
    fi
    
    vrooli_log_info "Starting resource: $name"
    
    # Start dependencies first
    if [[ -n "${RESOURCE_DEPENDENCIES[$name]:-}" ]]; then
        local deps="${RESOURCE_DEPENDENCIES[$name]}"
        IFS=',' read -ra dep_array <<< "$deps"
        
        for dep in "${dep_array[@]}"; do
            dep=$(echo "$dep" | xargs) # Trim whitespace
            if ! vrooli_resource_start "$dep"; then
                vrooli_log_error "Failed to start dependency: $dep"
                return 1
            fi
        done
    fi
    
    # Parse resource info
    local resource_info="${MANAGED_RESOURCES[$name]}"
    local type="${resource_info%%:*}"
    local start_cmd="${resource_info#*:}"
    
    # Start the resource based on type
    case "$type" in
        docker)
            if ! vrooli_resource_start_docker "$name" "$start_cmd"; then
                return 1
            fi
            ;;
        process)
            if ! vrooli_resource_start_process "$name" "$start_cmd"; then
                return 1
            fi
            ;;
        service)
            if ! vrooli_resource_start_service "$name" "$start_cmd"; then
                return 1
            fi
            ;;
        *)
            vrooli_log_error "Unknown resource type: $type"
            return 1
            ;;
    esac
    
    # Update state
    echo "running:$(date +%s)" > "$RESOURCE_STATE_DIR/$name.state"
    
    # Wait for health check if registered
    if [[ -n "${RESOURCE_HEALTH_CHECKS[$name]:-}" ]]; then
        if ! vrooli_resource_wait_healthy "$name" 30; then
            vrooli_log_error "Resource $name failed health check"
            vrooli_resource_stop "$name"
            return 1
        fi
    fi
    
    vrooli_log_info "Resource $name started successfully"
    return 0
}

#######################################
# Start a Docker container resource
# Arguments:
#   $1 - Resource name
#   $2 - Container name or run command
# Returns: 0 on success
#######################################
vrooli_resource_start_docker() {
    local name="$1"
    local container="$2"
    
    # Check if it's a container name or run command
    if [[ "$container" =~ ^docker\ run ]]; then
        # Execute run command
        if eval "$container"; then
            vrooli_log_info "Started Docker container for $name"
            return 0
        fi
    else
        # Start existing container
        if docker start "$container" 2>/dev/null; then
            vrooli_log_info "Started Docker container: $container"
            return 0
        fi
    fi
    
    vrooli_log_error "Failed to start Docker container for $name"
    return 1
}

#######################################
# Start a process resource
# Arguments:
#   $1 - Resource name
#   $2 - Command to execute
# Returns: 0 on success
#######################################
vrooli_resource_start_process() {
    local name="$1"
    local cmd="$2"
    
    # Start process in background and save PID
    eval "$cmd" &
    local pid=$!
    
    # Save PID for tracking
    echo "$pid" > "$RESOURCE_STATE_DIR/$name.pid"
    
    # Check if process started successfully
    sleep 1
    if kill -0 "$pid" 2>/dev/null; then
        vrooli_log_info "Started process for $name (PID: $pid)"
        return 0
    else
        vrooli_log_error "Process for $name failed to start"
        return 1
    fi
}

#######################################
# Start a system service resource
# Arguments:
#   $1 - Resource name
#   $2 - Service name
# Returns: 0 on success
#######################################
vrooli_resource_start_service() {
    local name="$1"
    local service="$2"
    
    # Try systemctl first
    if command -v systemctl >/dev/null 2>&1; then
        if sudo systemctl start "$service" 2>/dev/null; then
            vrooli_log_info "Started systemd service: $service"
            return 0
        fi
    fi
    
    # Try service command
    if command -v service >/dev/null 2>&1; then
        if sudo service "$service" start 2>/dev/null; then
            vrooli_log_info "Started service: $service"
            return 0
        fi
    fi
    
    vrooli_log_error "Failed to start service: $service"
    return 1
}

#######################################
# Stop a resource
# Arguments:
#   $1 - Resource name
# Returns: 0 on success
#######################################
vrooli_resource_stop() {
    local name="$1"
    
    # Check if resource is registered
    if [[ -z "${MANAGED_RESOURCES[$name]:-}" ]]; then
        vrooli_log_debug "Resource not registered: $name"
        return 0
    fi
    
    vrooli_log_info "Stopping resource: $name"
    
    # Parse resource info
    local resource_info="${MANAGED_RESOURCES[$name]}"
    local type="${resource_info%%:*}"
    local identifier="${resource_info#*:}"
    
    # Stop based on type
    case "$type" in
        docker)
            docker stop "$identifier" 2>/dev/null || true
            docker rm "$identifier" 2>/dev/null || true
            ;;
        process)
            if [[ -f "$RESOURCE_STATE_DIR/$name.pid" ]]; then
                local pid
                pid=$(cat "$RESOURCE_STATE_DIR/$name.pid")
                kill "$pid" 2>/dev/null || true
                sleep 1
                kill -9 "$pid" 2>/dev/null || true
                rm -f "$RESOURCE_STATE_DIR/$name.pid"
            fi
            ;;
        service)
            if command -v systemctl >/dev/null 2>&1; then
                sudo systemctl stop "$identifier" 2>/dev/null || true
            elif command -v service >/dev/null 2>&1; then
                sudo service "$identifier" stop 2>/dev/null || true
            fi
            ;;
    esac
    
    # Update state
    echo "stopped:$(date +%s)" > "$RESOURCE_STATE_DIR/$name.state"
    
    vrooli_log_info "Resource $name stopped"
    return 0
}

#######################################
# Check if a resource is running
# Arguments:
#   $1 - Resource name
# Returns: 0 if running, 1 if not
#######################################
vrooli_resource_is_running() {
    local name="$1"
    
    # Check state file
    if [[ ! -f "$RESOURCE_STATE_DIR/$name.state" ]]; then
        return 1
    fi
    
    local state
    state=$(cut -d: -f1 "$RESOURCE_STATE_DIR/$name.state")
    
    if [[ "$state" != "running" ]]; then
        return 1
    fi
    
    # Verify resource is actually running
    if [[ -n "${RESOURCE_HEALTH_CHECKS[$name]:-}" ]]; then
        vrooli_resource_check_health "$name" >/dev/null 2>&1
        return $?
    fi
    
    return 0
}

#######################################
# Perform health check on a resource
# Arguments:
#   $1 - Resource name
# Returns: 0 if healthy, 1 if not
#######################################
vrooli_resource_check_health() {
    local name="$1"
    
    # Get health check info
    local check_info="${RESOURCE_HEALTH_CHECKS[$name]:-}"
    if [[ -z "$check_info" ]]; then
        vrooli_log_debug "No health check registered for $name"
        return 0
    fi
    
    local check_type="${check_info%%:*}"
    local check_details="${check_info#*:}"
    
    case "$check_type" in
        http)
            curl -sf --max-time 5 "$check_details" >/dev/null 2>&1
            return $?
            ;;
        tcp)
            nc -z -w1 localhost "$check_details" 2>/dev/null
            return $?
            ;;
        process)
            pgrep -f "$check_details" >/dev/null 2>&1
            return $?
            ;;
        custom)
            eval "$check_details"
            return $?
            ;;
        *)
            vrooli_log_warn "Unknown health check type: $check_type"
            return 1
            ;;
    esac
}

#######################################
# Wait for a resource to become healthy
# Arguments:
#   $1 - Resource name
#   $2 - Timeout in seconds (optional, default 30)
# Returns: 0 if healthy, 1 if timeout
#######################################
vrooli_resource_wait_healthy() {
    local name="$1"
    local timeout="${2:-30}"
    local elapsed=0
    
    vrooli_log_info "Waiting for $name to be healthy (timeout: ${timeout}s)"
    
    while [[ $elapsed -lt $timeout ]]; do
        if vrooli_resource_check_health "$name"; then
            vrooli_log_info "Resource $name is healthy"
            return 0
        fi
        
        sleep 1
        ((elapsed++))
        
        # Progress indication
        if [[ $((elapsed % 5)) -eq 0 ]]; then
            vrooli_log_debug "Still waiting for $name... (${elapsed}s/${timeout}s)"
        fi
    done
    
    vrooli_log_error "Timeout waiting for $name to be healthy"
    return 1
}

#######################################
# Stop all managed resources
# Call this in cleanup/teardown
# Arguments: None
# Returns: 0
#######################################
vrooli_resource_stop_all() {
    vrooli_log_info "Stopping all managed resources"
    
    # Stop in reverse order (respecting dependencies)
    local resources=()
    for name in "${!MANAGED_RESOURCES[@]}"; do
        resources+=("$name")
    done
    
    # Simple reverse order (could be improved with topological sort)
    for ((i=${#resources[@]}-1; i>=0; i--)); do
        vrooli_resource_stop "${resources[$i]}"
    done
    
    # Clear tracking
    MANAGED_RESOURCES=()
    RESOURCE_DEPENDENCIES=()
    RESOURCE_HEALTH_CHECKS=()
    
    return 0
}

#######################################
# List all managed resources and their status
# Arguments: None
# Returns: 0
# Outputs: Resource status to stdout
#######################################
vrooli_resource_list() {
    vrooli_log_info "Managed resources:"
    
    if [[ ${#MANAGED_RESOURCES[@]} -eq 0 ]]; then
        echo "No resources currently managed"
        return 0
    fi
    
    for name in "${!MANAGED_RESOURCES[@]}"; do
        local status="stopped"
        if vrooli_resource_is_running "$name"; then
            status="running"
        fi
        
        local info="${MANAGED_RESOURCES[$name]}"
        local type="${info%%:*}"
        
        echo "  - $name ($type): $status"
        
        # Show dependencies if any
        if [[ -n "${RESOURCE_DEPENDENCIES[$name]:-}" ]]; then
            echo "    Dependencies: ${RESOURCE_DEPENDENCIES[$name]}"
        fi
        
        # Show health check if registered
        if [[ -n "${RESOURCE_HEALTH_CHECKS[$name]:-}" ]]; then
            local health="unhealthy"
            if vrooli_resource_check_health "$name" 2>/dev/null; then
                health="healthy"
            fi
            echo "    Health: $health"
        fi
    done
}

#######################################
# Export functions for use in tests
#######################################
export -f vrooli_resource_init vrooli_resource_register
export -f vrooli_resource_register_health_check vrooli_resource_start
export -f vrooli_resource_stop vrooli_resource_is_running
export -f vrooli_resource_check_health vrooli_resource_wait_healthy
export -f vrooli_resource_stop_all vrooli_resource_list

# Initialize on load
vrooli_resource_init

vrooli_log_info "Resource manager loaded successfully"