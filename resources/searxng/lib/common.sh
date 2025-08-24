#!/usr/bin/env bash
# SearXNG Common Utilities
# Shared functions used by other SearXNG modules

# Source required utilities
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*/../../.." && builtin pwd)}"
SEARXNG_LIB_DIR="${APP_ROOT}/resources/searxng/lib"
# shellcheck disable=SC1091
source "${SEARXNG_LIB_DIR}/../../../lib/utils/var.sh" 2>/dev/null || true
# shellcheck disable=SC1091
source "${var_LIB_SYSTEM_DIR}/trash.sh" 2>/dev/null || true
# shellcheck disable=SC1091
source "${var_LIB_UTILS_DIR}/sudo.sh" 2>/dev/null || true

#######################################
# Check if SearXNG is installed
# Returns:
#   0 if installed, 1 otherwise
#######################################
searxng::is_installed() {
    # Check if Docker is available
    if ! command -v docker >/dev/null 2>&1; then
        return 1
    fi
    
    # Check if SearXNG container exists
    if docker container inspect "$SEARXNG_CONTAINER_NAME" >/dev/null 2>&1; then
        return 0
    fi
    
    return 1
}

#######################################
# Check if SearXNG is running
# Returns:
#   0 if running, 1 otherwise
#######################################
searxng::is_running() {
    if ! searxng::is_installed; then
        return 1
    fi
    
    # Check if container is running
    local status
    status=$(docker container inspect --format='{{.State.Status}}' "$SEARXNG_CONTAINER_NAME" 2>/dev/null)
    
    if [[ "$status" == "running" ]]; then
        return 0
    fi
    
    return 1
}

#######################################
# Check if SearXNG is healthy with detailed diagnostics
# Returns:
#   0 if healthy, 1 otherwise
#######################################
searxng::is_healthy() {
    if ! searxng::is_running; then
        return 1
    fi
    
    # Check Docker health status first
    local health_status
    health_status=$(docker container inspect --format='{{.State.Health.Status}}' "$SEARXNG_CONTAINER_NAME" 2>/dev/null)
    
    if [[ "$health_status" == "healthy" ]]; then
        return 0
    fi
    
    # Always try API check as fallback, regardless of Docker health status
    # Docker health checks can be misconfigured while API still works
    if resources::is_service_running "$SEARXNG_PORT"; then
        # Try to reach the stats endpoint
        if curl -sf --max-time 5 "${SEARXNG_BASE_URL}/stats" >/dev/null 2>&1; then
            return 0
        fi
    fi
    
    return 1
}

#######################################
# Enhanced health check with detailed diagnostics
# Returns:
#   0 if healthy, 1 otherwise
# Outputs detailed diagnostic information
#######################################
searxng::health_check_detailed() {
    local issues=()
    local warnings=()
    local status="healthy"
    
    log::info "Running detailed SearXNG health check..."
    
    # 1. Check if container exists
    if ! searxng::is_installed; then
        issues+=("Container not installed")
        status="critical"
    else
        log::success "✅ Container is installed"
    fi
    
    # 2. Check if container is running
    if ! searxng::is_running; then
        issues+=("Container not running")
        status="critical"
        
        # Check container exit code and logs if not running
        local exit_code
        exit_code=$(docker container inspect --format='{{.State.ExitCode}}' "$SEARXNG_CONTAINER_NAME" 2>/dev/null || echo "unknown")
        if [[ "$exit_code" != "0" && "$exit_code" != "unknown" ]]; then
            issues+=("Container exited with code $exit_code")
            
            # Get recent logs to help diagnose
            log::info "Recent container logs:"
            docker logs --tail 10 "$SEARXNG_CONTAINER_NAME" 2>/dev/null || true
        fi
    else
        log::success "✅ Container is running"
    fi
    
    # 3. Check configuration files and permissions
    if [[ -d "$SEARXNG_DATA_DIR" ]]; then
        log::success "✅ Data directory exists: $SEARXNG_DATA_DIR"
        
        # Check configuration files
        local config_file="$SEARXNG_DATA_DIR/settings.yml"
        if [[ -f "$config_file" ]]; then
            log::success "✅ Configuration file exists"
            
            # Check file permissions
            local current_uid current_gid
            current_uid=$(id -u)
            current_gid=$(id -g)
            
            local file_uid file_gid
            if command -v stat >/dev/null 2>&1; then
                file_uid=$(stat -c %u "$config_file" 2>/dev/null || echo "unknown")
                file_gid=$(stat -c %g "$config_file" 2>/dev/null || echo "unknown")
                
                if [[ "$file_uid" != "$current_uid" ]] || [[ "$file_gid" != "$current_gid" ]]; then
                    issues+=("Configuration file has incorrect ownership (expected $current_uid:$current_gid, got $file_uid:$file_gid)")
                    status="critical"
                else
                    log::success "✅ Configuration file has correct ownership"
                fi
            fi
            
            # Check if file is readable
            if [[ ! -r "$config_file" ]]; then
                issues+=("Configuration file is not readable")
                status="critical"
            else
                log::success "✅ Configuration file is readable"
            fi
        else
            issues+=("Configuration file missing: $config_file")
            status="critical"
        fi
    else
        issues+=("Data directory missing: $SEARXNG_DATA_DIR")
        status="critical"
    fi
    
    # 4. Check port availability
    if resources::is_service_running "$SEARXNG_PORT"; then
        log::success "✅ Port $SEARXNG_PORT is listening"
        
        # 5. Check HTTP health
        if curl -sf --max-time 5 "${SEARXNG_BASE_URL}/stats" >/dev/null 2>&1; then
            log::success "✅ HTTP health check passed"
        else
            issues+=("HTTP health check failed - service not responding")
            status="degraded"
        fi
    else
        if searxng::is_running; then
            issues+=("Container running but port $SEARXNG_PORT not accessible")
            status="degraded"
        fi
    fi
    
    # 6. Check Docker health status
    if searxng::is_running; then
        local docker_health
        docker_health=$(docker container inspect --format='{{.State.Health.Status}}' "$SEARXNG_CONTAINER_NAME" 2>/dev/null || echo "no-health-check")
        
        case "$docker_health" in
            "healthy")
                log::success "✅ Docker health check: healthy"
                ;;
            "unhealthy")
                issues+=("Docker health check reports unhealthy")
                status="degraded"
                ;;
            "starting")
                warnings+=("Docker health check still starting")
                ;;
            "no-health-check")
                log::info "ℹ️  No Docker health check configured"
                ;;
        esac
    fi
    
    # Output summary
    echo
    log::header "Health Check Summary"
    echo "Status: $status"
    
    if [[ ${#issues[@]} -gt 0 ]]; then
        echo "Issues found:"
        for issue in "${issues[@]}"; do
            echo "  ❌ $issue"
        done
    fi
    
    if [[ ${#warnings[@]} -gt 0 ]]; then
        echo "Warnings:"
        for warning in "${warnings[@]}"; do
            echo "  ⚠️  $warning"
        done
    fi
    
    if [[ ${#issues[@]} -eq 0 ]]; then
        echo "✅ All checks passed"
        return 0
    else
        return 1
    fi
}

#######################################
# Get SearXNG container status
# Outputs: status string
#######################################
searxng::get_status() {
    if ! searxng::is_installed; then
        echo "not_installed"
        return 0
    fi
    
    if ! searxng::is_running; then
        echo "stopped"
        return 0
    fi
    
    if searxng::is_healthy; then
        echo "healthy"
        return 0
    fi
    
    echo "unhealthy"
}

#######################################
# Wait for SearXNG to become healthy
# Arguments:
#   $1 - timeout in seconds (optional, default: 60)
# Returns:
#   0 if healthy, 1 if timeout
#######################################
searxng::wait_for_health() {
    local timeout="${1:-60}"
    local counter=0
    local interval=5
    
    log::info "Waiting for SearXNG to become healthy (timeout: ${timeout}s)..."
    
    while [[ $counter -lt $timeout ]]; do
        if searxng::is_healthy; then
            log::success "SearXNG is healthy!"
            return 0
        fi
        
        log::info "  Waiting... (${counter}/${timeout}s)"
        sleep $interval
        counter=$((counter + interval))
    done
    
    log::error "Timeout waiting for SearXNG to become healthy"
    return 1
}

#######################################
# Get SearXNG version
# Outputs: version string
#######################################
searxng::get_version() {
    if ! searxng::is_running; then
        echo "unknown"
        return 1
    fi
    
    # Try to get version from API or container
    local version
    version=$(docker exec "$SEARXNG_CONTAINER_NAME" python -c "import searx; print(searx.__version__)" 2>/dev/null)
    
    if [[ -n "$version" ]]; then
        echo "$version"
    else
        echo "unknown"
    fi
}

#######################################
# Get SearXNG container logs
# Arguments:
#   $1 - number of lines (optional, default: from config)
#######################################
searxng::get_logs() {
    local lines="${1:-$SEARXNG_LOG_LINES}"
    
    if ! searxng::is_installed; then
        log::error "SearXNG is not installed"
        return 1
    fi
    
    docker logs --tail "$lines" "$SEARXNG_CONTAINER_NAME"
}

#######################################
# Ensure data directory exists with proper permissions
#######################################
searxng::ensure_data_dir() {
    if [[ ! -d "$SEARXNG_DATA_DIR" ]]; then
        log::info "Creating SearXNG data directory: $SEARXNG_DATA_DIR"
        if ! mkdir -p "$SEARXNG_DATA_DIR"; then
            log::error "Failed to create SearXNG data directory: $SEARXNG_DATA_DIR"
            return 1
        fi
    fi
    
    # Fix Docker volume permissions
    log::info "Setting up Docker volume permissions..."
    
    if [[ -d "$SEARXNG_DATA_DIR" ]]; then
        # SearXNG runs as user 977:977 inside the container
        # Instead of changing ownership, make directory accessible to both host user and container
        
        # Ensure current user owns the directory for host operations
        if [[ -O "$SEARXNG_DATA_DIR" ]]; then
            log::success "✅ Directory already owned by current user"
        else
            log::info "Attempting to claim ownership of directory..."
            # Try to change ownership to current user - if this fails, we'll use permission fallback
            if chown -R "$(id -u):$(id -g)" "$SEARXNG_DATA_DIR" 2>/dev/null; then
                log::success "✅ Changed ownership to current user"
            else
                log::warn "Could not change ownership - will use permission fallback"
            fi
        fi
        
        # Set permissions to allow both host user and container to read/write
        # Use 777 permissions to ensure container can write regardless of ownership
        if chmod -R 777 "$SEARXNG_DATA_DIR" 2>/dev/null; then
            log::success "✅ Set directory permissions for container access"
        else
            log::warn "Could not set full permissions - SearXNG may have issues"
            # Try at least to make it readable
            chmod -R 755 "$SEARXNG_DATA_DIR" 2>/dev/null || true
        fi
        
        return 0
    else
        log::error "SearXNG data directory does not exist after creation attempt"
        return 1
    fi
}

#######################################
# Clean up SearXNG resources
# Arguments:
#   $1 - cleanup level: "containers" or "all"
#######################################
searxng::cleanup() {
    local level="${1:-containers}"
    
    log::info "Cleaning up SearXNG resources (level: $level)..."
    
    # Stop and remove containers
    if docker container inspect "$SEARXNG_CONTAINER_NAME" >/dev/null 2>&1; then
        log::info "Removing SearXNG container..."
        docker container rm -f "$SEARXNG_CONTAINER_NAME" >/dev/null 2>&1
    fi
    
    # Remove Redis container if it exists
    if docker container inspect "searxng-redis" >/dev/null 2>&1; then
        log::info "Removing SearXNG Redis container..."
        docker container rm -f "searxng-redis" >/dev/null 2>&1
    fi
    
    # Remove network
    if docker network inspect "$SEARXNG_NETWORK_NAME" >/dev/null 2>&1; then
        log::info "Removing SearXNG network..."
        docker network rm "$SEARXNG_NETWORK_NAME" >/dev/null 2>&1
    fi
    
    # Remove data directory if requested
    if [[ "$level" == "all" ]] && [[ -d "$SEARXNG_DATA_DIR" ]]; then
        log::info "Removing SearXNG data directory..."
        trash::safe_remove "$SEARXNG_DATA_DIR" --production
    fi
    
    log::success "SearXNG cleanup completed"
}

#######################################
# Validate SearXNG configuration
# Returns:
#   0 if valid, 1 otherwise
#######################################
searxng::validate_config() {
    # Check if required configuration variables are set
    local required_vars=(
        "SEARXNG_PORT"
        "SEARXNG_BASE_URL"
        "SEARXNG_CONTAINER_NAME"
        "SEARXNG_IMAGE"
        "SEARXNG_DATA_DIR"
        "SEARXNG_SECRET_KEY"
    )
    
    for var in "${required_vars[@]}"; do
        if [[ -z "${!var}" ]]; then
            log::error "Required configuration variable $var is not set"
            return 1
        fi
    done
    
    # Validate port
    if ! ports::validate_port "$SEARXNG_PORT"; then
        log::error "Invalid port configuration: $SEARXNG_PORT"
        return 1
    fi
    
    # Check for port conflicts
    if resources::is_service_running "$SEARXNG_PORT" && ! searxng::is_running; then
        log::error "Port $SEARXNG_PORT is already in use by another service"
        return 1
    fi
    
    return 0
}

#######################################
# Show SearXNG information
#######################################
searxng::show_info() {
    local status
    status=$(searxng::get_status)
    
    log::info "SearXNG Information:"
    echo "  Status: $status"
    echo "  Port: $SEARXNG_PORT"
    echo "  Base URL: $SEARXNG_BASE_URL"
    echo "  Container: $SEARXNG_CONTAINER_NAME"
    echo "  Data Directory: $SEARXNG_DATA_DIR"
    
    if [[ "$status" == "healthy" ]] || [[ "$status" == "unhealthy" ]]; then
        local version
        version=$(searxng::get_version)
        echo "  Version: $version"
        
        # Show access information
        echo
        searxng::message "info" "MSG_SEARXNG_ACCESS_INFO"
        searxng::message "info" "MSG_SEARXNG_API_INFO"
        searxng::message "info" "MSG_SEARXNG_STATS_INFO"
    fi
}