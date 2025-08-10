#!/usr/bin/env bash
# Mode Management Library for Agent S2
# Handles switching between sandbox and host modes

#######################################
# Get current operating mode
# Returns: mode name (sandbox/host)
#######################################
agents2::get_current_mode() {
    if docker ps --filter "name=${AGENTS2_CONTAINER_NAME}" --format "table {{.Names}}" | grep -q "${AGENTS2_CONTAINER_NAME}"; then
        # Check environment variable in running container
        local mode=$(docker exec "${AGENTS2_CONTAINER_NAME}" printenv AGENT_S2_MODE 2>/dev/null || echo "sandbox")
        echo "$mode"
    else
        echo "sandbox"  # Default mode
    fi
}

#######################################
# Validate host mode prerequisites
# Returns: 0 if valid, 1 if invalid
#######################################
agents2::validate_host_mode() {
    local errors=0
    
    # Check if host mode is enabled in configuration
    if [[ "${AGENT_S2_HOST_MODE_ENABLED:-false}" != "true" ]]; then
        log::error "Host mode is not enabled in configuration"
        log::info "Set AGENT_S2_HOST_MODE_ENABLED=true in environment or config"
        ((errors++))
    fi
    
    # Check X11 permissions if display access is requested
    if [[ "${AGENT_S2_HOST_DISPLAY_ACCESS:-false}" == "true" ]]; then
        if ! command -v xhost >/dev/null 2>&1; then
            log::warning "xhost not found - X11 permissions may not work"
        else
            # Test X11 access
            if ! xdpyinfo >/dev/null 2>&1; then
                log::error "Cannot access X11 display"
                log::info "Run: export DISPLAY=:0 or check X11 permissions"
                ((errors++))
            fi
        fi
    fi
    
    # Check for required directories
    local mount_dirs=(
        "${HOME}/Documents"
        "${HOME}/Downloads"
        "/tmp"
    )
    
    for dir in "${mount_dirs[@]}"; do
        if [[ ! -d "$dir" ]]; then
            log::warning "Mount directory does not exist: $dir"
        elif [[ ! -r "$dir" ]]; then
            log::error "Cannot read mount directory: $dir"
            ((errors++))
        fi
    done
    
    # Check AppArmor profile if available
    if command -v aa-status >/dev/null 2>&1; then
        if ! aa-status | grep -q "agent-s2-host" 2>/dev/null; then
            log::warning "AppArmor profile 'agent-s2-host' not found"
            log::info "Host mode will run with default security profile"
        fi
    fi
    
    return $errors
}

#######################################
# Setup X11 permissions for host mode
#######################################
agents2::setup_x11_permissions() {
    if [[ "${AGENT_S2_HOST_DISPLAY_ACCESS:-false}" != "true" ]]; then
        return 0
    fi
    
    log::info "Setting up X11 permissions for host mode..."
    
    # Allow docker containers to access X11
    if command -v xhost >/dev/null 2>&1; then
        xhost +local:docker >/dev/null 2>&1 || {
            log::warning "Failed to set X11 permissions with xhost"
        }
    fi
    
    # Create Xauthority file if needed
    if [[ -n "${XAUTHORITY:-}" ]] && [[ -f "$XAUTHORITY" ]]; then
        # Copy host Xauthority to temporary location for container
        cp "$XAUTHORITY" /tmp/.Xauthority 2>/dev/null || {
            log::warning "Failed to copy Xauthority file"
        }
        chmod 644 /tmp/.Xauthority 2>/dev/null
    fi
}

#######################################
# Start Agent S2 in specified mode
# Arguments:
#   $1 - mode (sandbox/host)
#######################################
agents2::start_in_mode() {
    local mode="${1:-sandbox}"
    local compose_files=("-f" "${SCRIPT_DIR}/docker/compose/docker-compose.yml")
    
    log::info "Starting Agent S2 in $mode mode..."
    
    case "$mode" in
        "sandbox")
            # Use default compose file only
            export AGENT_S2_MODE=sandbox
            ;;
        "host")
            # Validate host mode prerequisites
            if ! agents2::validate_host_mode; then
                log::error "Host mode validation failed"
                return 1
            fi
            
            # Setup host mode environment
            agents2::setup_x11_permissions
            
            # Add host mode compose override
            compose_files+=("-f" "${SCRIPT_DIR}/docker/compose/docker-compose.host.yml")
            export AGENT_S2_MODE=host
            export AGENT_S2_HOST_MODE_ENABLED=true
            ;;
        *)
            log::error "Unknown mode: $mode"
            return 1
            ;;
    esac
    
    # Stop any existing container
    agents2::docker_stop >/dev/null 2>&1
    
    # Load environment variables from resources.local.json
    agents2::load_environment_from_config
    
    # Start with appropriate compose files
    log::debug "Using compose files: ${compose_files[*]}"
    docker-compose "${compose_files[@]}" up -d
    
    local exit_code=$?
    if [[ $exit_code -eq 0 ]]; then
        log::success "Agent S2 started successfully in $mode mode"
        
        # Wait for service to be ready
        log::info "Waiting for Agent S2 to be ready..."
        agents2::wait_for_health 60
        
        # Display mode information
        agents2::show_mode_info
    else
        log::error "Failed to start Agent S2 in $mode mode"
        return $exit_code
    fi
}

#######################################
# Switch between modes
# Arguments:
#   $1 - new mode (sandbox/host)
#   $2 - force flag (optional)
#######################################
agents2::switch_mode() {
    local new_mode="$1"
    local force="${2:-false}"
    
    if [[ -z "$new_mode" ]]; then
        log::error "Mode not specified"
        return 1
    fi
    
    local current_mode=$(agents2::get_current_mode)
    
    if [[ "$current_mode" == "$new_mode" ]] && [[ "$force" != "true" ]]; then
        log::info "Already running in $new_mode mode"
        return 0
    fi
    
    log::info "Switching from $current_mode mode to $new_mode mode..."
    
    # Graceful shutdown
    if agents2::is_running; then
        log::info "Stopping current Agent S2 instance..."
        agents2::docker_stop
        
        # Wait for complete shutdown
        sleep 3
    fi
    
    # Start in new mode
    agents2::start_in_mode "$new_mode"
}

#######################################
# Show current mode information
#######################################
agents2::show_mode_info() {
    if ! agents2::is_running; then
        log::warning "Agent S2 is not running"
        return 1
    fi
    
    local current_mode=$(agents2::get_current_mode)
    
    echo
    log::info "Agent S2 Mode Information:"
    echo "  Current Mode: $current_mode"
    echo "  Host Mode Enabled: ${AGENT_S2_HOST_MODE_ENABLED:-false}"
    
    if [[ "$current_mode" == "host" ]]; then
        echo "  Host Display Access: ${AGENT_S2_HOST_DISPLAY_ACCESS:-false}"
        echo "  Security Profile: ${AGENT_S2_HOST_SECURITY_PROFILE:-agent-s2-host}"
        echo "  Audit Logging: ${AGENT_S2_HOST_AUDIT_LOGGING:-true}"
    fi
    
    echo "  API Endpoint: http://localhost:${AGENT_S2_PORT:-4113}"
    echo "  VNC Endpoint: vnc://localhost:${AGENT_S2_VNC_PORT:-5900}"
    
    # Get mode details from API
    if command -v curl >/dev/null 2>&1; then
        local api_response=$(curl -s "http://localhost:${AGENT_S2_PORT:-4113}/modes/current" 2>/dev/null)
        if [[ -n "$api_response" ]]; then
            local apps_count=$(echo "$api_response" | jq -r '.applications_count // 0' 2>/dev/null || echo "unknown")
            local security_level=$(echo "$api_response" | jq -r '.security_level // "unknown"' 2>/dev/null || echo "unknown")
            
            echo "  Available Applications: $apps_count"
            echo "  Security Level: $security_level"
        fi
    fi
    echo
}

#######################################
# Wait for Agent S2 health check
# Arguments:
#   $1 - timeout in seconds (default: 30)
#######################################
agents2::wait_for_health() {
    local timeout="${1:-30}"
    local count=0
    
    while [[ $count -lt $timeout ]]; do
        if curl -sf "http://localhost:${AGENT_S2_PORT:-4113}/health" >/dev/null 2>&1; then
            log::success "Agent S2 is healthy and ready"
            return 0
        fi
        
        sleep 1
        ((count++))
        
        if [[ $((count % 10)) -eq 0 ]]; then
            log::info "Still waiting for Agent S2 to be ready... (${count}s)"
        fi
    done
    
    log::warning "Agent S2 health check timeout after ${timeout}s"
    return 1
}

#######################################
# Get available modes
#######################################
agents2::list_modes() {
    echo "Available Agent S2 modes:"
    echo
    echo "  sandbox - Secure, isolated environment (default)"
    echo "    • Pre-installed applications only"
    echo "    • Virtual display (Xvfb + Fluxbox)"
    echo "    • No host filesystem access"
    echo "    • High security isolation"
    echo
    
    if [[ "${AGENT_S2_HOST_MODE_ENABLED:-false}" == "true" ]]; then
        echo "  host - Extended host system access"
        echo "    • Native desktop integration"
        echo "    • Host filesystem mounts"
        echo "    • All installed applications"
        echo "    • Medium security isolation"
        echo
    else
        echo "  host - Extended host system access (DISABLED)"
        echo "    • Enable with: AGENT_S2_HOST_MODE_ENABLED=true"
        echo
    fi
    
    local current_mode=$(agents2::get_current_mode)
    echo "Current mode: $current_mode"
}

#######################################
# Test mode functionality
# Arguments:
#   $1 - mode to test (optional, defaults to current)
#######################################
agents2::test_mode() {
    local mode="${1:-$(agents2::get_current_mode)}"
    
    log::info "Testing Agent S2 $mode mode functionality..."
    
    # Test API connectivity
    if ! curl -sf "http://localhost:${AGENT_S2_PORT:-4113}/health" >/dev/null; then
        log::error "API health check failed"
        return 1
    fi
    log::success "✓ API connectivity"
    
    # Test mode-specific endpoints
    if ! curl -sf "http://localhost:${AGENT_S2_PORT:-4113}/modes/current" >/dev/null; then
        log::error "Mode API endpoint failed"
        return 1
    fi
    log::success "✓ Mode management API"
    
    # Test screenshot capability
    if curl -sf "http://localhost:${AGENT_S2_PORT:-4113}/screenshot" >/dev/null; then
        log::success "✓ Screenshot capability"
    else
        log::warning "⚠ Screenshot test failed"
    fi
    
    # Test applications endpoint
    if curl -sf "http://localhost:${AGENT_S2_PORT:-4113}/modes/applications" >/dev/null; then
        log::success "✓ Applications discovery"
    else
        log::warning "⚠ Applications discovery failed"
    fi
    
    log::success "Mode testing completed for $mode mode"
}

#######################################
# Export mode configuration
#######################################
agents2::export_mode_config() {
    echo "# Agent S2 Mode Configuration"
    echo "export AGENT_S2_MODE=${AGENT_S2_MODE:-sandbox}"
    echo "export AGENT_S2_HOST_MODE_ENABLED=${AGENT_S2_HOST_MODE_ENABLED:-false}"
    echo "export AGENT_S2_HOST_DISPLAY_ACCESS=${AGENT_S2_HOST_DISPLAY_ACCESS:-false}"
    echo "export AGENT_S2_HOST_MOUNTS='${AGENT_S2_HOST_MOUNTS:-[]}'"
    echo "export AGENT_S2_HOST_APPS='${AGENT_S2_HOST_APPS:-*}'"
    echo "export AGENT_S2_HOST_SECURITY_PROFILE=${AGENT_S2_HOST_SECURITY_PROFILE:-agent-s2-host}"
    echo "export AGENT_S2_HOST_AUDIT_LOGGING=${AGENT_S2_HOST_AUDIT_LOGGING:-true}"
}

# Export functions for subshell availability
export -f agents2::get_current_mode
export -f agents2::validate_host_mode
export -f agents2::setup_x11_permissions
export -f agents2::start_in_mode
export -f agents2::switch_mode
export -f agents2::show_mode_info
export -f agents2::wait_for_health
export -f agents2::list_modes
export -f agents2::test_mode
export -f agents2::export_mode_config