#!/usr/bin/env bash
# Browserless Common Utility Functions
# Shared utilities used across all modules

#######################################
# Check if Docker is installed and configured
# Returns: 0 if installed, 1 otherwise
#######################################
browserless::check_docker() {
    if ! system::is_command "docker"; then
        log::error "${MSG_DOCKER_NOT_FOUND}"
        log::info "${MSG_DOCKER_INSTALL_HINT}"
        return 1
    fi
    
    # Check if Docker daemon is running
    if ! docker info >/dev/null 2>&1; then
        log::error "${MSG_DOCKER_NOT_RUNNING}"
        log::info "${MSG_DOCKER_START_HINT}"
        return 1
    fi
    
    # Check if user has permissions
    if ! docker ps >/dev/null 2>&1; then
        log::error "${MSG_DOCKER_NO_PERMISSIONS}"
        log::info "${MSG_DOCKER_PERMISSIONS_HINT}"
        log::info "${MSG_DOCKER_LOGOUT_HINT}"
        return 1
    fi
    
    return 0
}

#######################################
# Check if Browserless container exists
# Returns: 0 if exists, 1 otherwise
#######################################
browserless::container_exists() {
    docker ps -a --format '{{.Names}}' | grep -q "^${BROWSERLESS_CONTAINER_NAME}$"
}

#######################################
# Check if Browserless is running
# Returns: 0 if running, 1 otherwise
#######################################
browserless::is_running() {
    docker ps --format '{{.Names}}' | grep -q "^${BROWSERLESS_CONTAINER_NAME}$"
}

#######################################
# Check if port is available
# Returns: 0 if available, 1 if in use
#######################################
browserless::check_port() {
    if system::is_port_in_use "${BROWSERLESS_PORT}"; then
        log::error "${MSG_PORT_IN_USE}"
        log::info "You can set a custom port with: export BROWSERLESS_CUSTOM_PORT=<port>"
        return 1
    fi
    return 0
}

#######################################
# Validate installation prerequisites
# Returns: 0 if all checks pass, 1 otherwise
#######################################
browserless::validate_prerequisites() {
    # Check Docker
    if ! browserless::check_docker; then
        return 1
    fi
    
    # Validate port assignment
    if ! resources::validate_port "browserless" "$BROWSERLESS_PORT"; then
        log::error "Port validation failed for Browserless"
        log::info "You can set a custom port with: export BROWSERLESS_CUSTOM_PORT=<port>"
        return 1
    fi
    
    return 0
}

#######################################
# Check if already installed and handle force flag
# Returns: 0 if should proceed, 1 if should skip installation
#######################################
browserless::check_existing_installation() {
    if browserless::container_exists && browserless::is_running && [[ "$FORCE" != "yes" ]]; then
        log::info "${MSG_ALREADY_INSTALLED}"
        log::info "Use --force yes to reinstall"
        return 1
    fi
    return 0
}

#######################################
# Wait for service to be ready
# Arguments:
#   $1 - Description of what we're waiting for
#   $2 - Max wait time in seconds (optional, defaults to BROWSERLESS_STARTUP_MAX_WAIT)
# Returns: 0 if ready, 1 if timeout
#######################################
browserless::wait_for_ready() {
    local description="${1:-Browserless to start}"
    local max_wait="${2:-$BROWSERLESS_STARTUP_MAX_WAIT}"
    
    log::info "Waiting for ${description}..."
    
    local wait_time=0
    while [ $wait_time -lt $max_wait ]; do
        if browserless::is_running && ss -tlnp 2>/dev/null | grep -q ":$BROWSERLESS_PORT"; then
            log::info "Container is running and port is bound"
            return 0
        fi
        sleep $BROWSERLESS_STARTUP_WAIT_INTERVAL
        wait_time=$((wait_time + BROWSERLESS_STARTUP_WAIT_INTERVAL))
        echo -n "."
    done
    echo
    
    log::error "${MSG_STARTUP_TIMEOUT}"
    return 1
}

#######################################
# Display container resource usage
#######################################
browserless::show_resource_usage() {
    local stats
    stats=$(docker stats "$BROWSERLESS_CONTAINER_NAME" --no-stream --format "CPU: {{.CPUPerc}} | Memory: {{.MemUsage}}" 2>/dev/null || echo "")
    if [[ -n "$stats" ]]; then
        log::info "Resource usage: $stats"
    fi
}

#######################################
# Create backup of data directory
# Arguments:
#   $1 - Description of backup reason (optional)
#######################################
browserless::backup_data() {
    local reason="${1:-manual backup}"
    
    if [[ -d "$BROWSERLESS_DATA_DIR" ]]; then
        local backup_dir="$HOME/browserless-backup-$(date +%Y%m%d-%H%M%S)"
        log::info "${MSG_BACKING_UP_DATA} $backup_dir"
        cp -r "$BROWSERLESS_DATA_DIR" "$backup_dir" 2>/dev/null || true
        log::info "Backup created for: $reason"
    fi
}