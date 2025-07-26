#!/usr/bin/env bash
# Agent S2 Status Functions
# Status checking and reporting functionality

#######################################
# Show comprehensive Agent S2 status
# Returns: 0 always
#######################################
agents2::show_status() {
    log::header "$MSG_STATUS_HEADER"
    
    # Check Docker availability
    if ! system::is_command "docker"; then
        log::error "Docker is not installed"
        return 0
    fi
    
    if ! docker info >/dev/null 2>&1; then
        log::error "Docker daemon is not running"
        return 0
    fi
    
    # Check container status
    if agents2::container_exists; then
        if agents2::is_running; then
            log::success "$MSG_SERVICE_RUNNING"
            
            # Get container stats
            echo
            agents2::docker_stats
            
            # Check health
            local health_status
            health_status=$(agents2::get_health_status)
            echo
            case "$health_status" in
                "healthy")
                    log::success "✅ Container health: healthy"
                    ;;
                "unhealthy")
                    log::error "❌ Container health: unhealthy"
                    ;;
                "starting")
                    log::warn "⏳ Container health: starting"
                    ;;
                *)
                    log::info "Container health: $health_status"
                    ;;
            esac
            
            # API health check
            if agents2::is_healthy; then
                log::success "✅ API is healthy and responding"
                
                # Get detailed health info
                if system::is_command "jq"; then
                    local health_info
                    health_info=$(curl -s "${AGENTS2_BASE_URL}/health" 2>/dev/null)
                    if [[ -n "$health_info" ]]; then
                        echo
                        log::info "Health Details:"
                        echo "$health_info" | jq . 2>/dev/null || echo "$health_info"
                    fi
                fi
            else
                log::warn "⚠️  API health check failed"
            fi
            
            # Show access information
            echo
            agents2::show_access_info
            
            # Check ports
            echo
            agents2::check_ports
            
        else
            log::warn "⚠️  Agent S2 container exists but is not running"
            log::info "Start it with: $0 --action start"
        fi
    else
        log::info "$MSG_SERVICE_NOT_INSTALLED"
        log::info "Install with: $0 --action install"
    fi
    
    # Show data directory info
    if [[ -d "$AGENTS2_DATA_DIR" ]]; then
        echo
        log::info "Data Directory: $AGENTS2_DATA_DIR"
        local dir_size
        dir_size=$(du -sh "$AGENTS2_DATA_DIR" 2>/dev/null | cut -f1)
        [[ -n "$dir_size" ]] && log::info "Disk Usage: $dir_size"
    fi
    
    return 0
}

#######################################
# Show access information
#######################################
agents2::show_access_info() {
    log::info "Access Information:"
    log::info "  API URL: $AGENTS2_BASE_URL"
    log::info "  VNC URL: $AGENTS2_VNC_URL"
    log::info "  API Docs: ${AGENTS2_BASE_URL}/docs"
    log::info "  Container: $AGENTS2_CONTAINER_NAME"
    
    # Show VNC connection command
    echo
    log::info "Connect via VNC:"
    log::info "  Using viewer: vncviewer localhost:$AGENTS2_VNC_PORT"
    log::info "  Using browser: Open VNC client and connect to localhost:$AGENTS2_VNC_PORT"
    log::info "  Password: $AGENTS2_VNC_PASSWORD"
}

#######################################
# Check port availability
#######################################
agents2::check_ports() {
    log::info "Port Status:"
    
    # Check API port
    if resources::is_service_running "$AGENTS2_PORT"; then
        log::success "$MSG_PORT_LISTENING $AGENTS2_PORT (API)"
    else
        log::warn "$MSG_PORT_NOT_ACCESSIBLE $AGENTS2_PORT (API)"
    fi
    
    # Check VNC port
    if resources::is_service_running "$AGENTS2_VNC_PORT"; then
        log::success "$MSG_PORT_LISTENING $AGENTS2_VNC_PORT (VNC)"
    else
        log::warn "$MSG_PORT_NOT_ACCESSIBLE $AGENTS2_VNC_PORT (VNC)"
    fi
}

#######################################
# Show detailed Agent S2 information
#######################################
agents2::show_info() {
    cat << EOF
=== Agent S2 Resource Information ===

ID: agent-s2
Category: agents
Display Name: Agent S2
Description: Autonomous computer interaction service with GUI automation

Service Details:
- Container Name: $AGENTS2_CONTAINER_NAME
- Image Name: $AGENTS2_IMAGE_NAME
- API Port: $AGENTS2_PORT
- VNC Port: $AGENTS2_VNC_PORT
- API URL: $AGENTS2_BASE_URL
- VNC URL: $AGENTS2_VNC_URL
- Data Directory: $AGENTS2_DATA_DIR

Display Configuration:
- Virtual Display: $AGENTS2_DISPLAY
- Screen Resolution: $AGENTS2_SCREEN_RESOLUTION
- VNC Access: Enabled (password protected)
- Host Display Access: $AGENTS2_ENABLE_HOST_DISPLAY

LLM Configuration:
- Provider: $AGENTS2_LLM_PROVIDER
- Model: $AGENTS2_LLM_MODEL
- API Key: $([ -n "$AGENTS2_API_KEY" ] && echo "Configured" || echo "Not configured")

Resource Limits:
- Memory: $AGENTS2_MEMORY_LIMIT
- CPUs: $AGENTS2_CPU_LIMIT
- Shared Memory: $AGENTS2_SHM_SIZE

Capabilities:
- Screenshot capture
- Mouse control (click, move, drag)
- Keyboard input (type, key press)
- GUI automation sequences
- Multi-step task planning
- Cross-platform support

Security Features:
- Runs as non-root user
- Isolated virtual display
- Sandboxed environment
- Restricted application access
- VNC password protection

API Endpoints:
- GET  /health          - Health check
- GET  /capabilities    - List capabilities
- POST /screenshot      - Capture screenshot
- POST /execute         - Execute automation task
- GET  /tasks/{id}      - Get task status
- GET  /mouse/position  - Get mouse position

Management Commands:
$0 --action status      # Check status
$0 --action logs        # View logs
$0 --action usage       # Run examples
$0 --action restart     # Restart service

For more information: https://www.simular.ai/agent-s
EOF
}

#######################################
# Get quick status (for scripts)
# Returns: JSON status object
#######################################
agents2::get_status_json() {
    local status="not_installed"
    local health="unknown"
    local api_responsive=false
    local vnc_accessible=false
    
    if agents2::container_exists; then
        if agents2::is_running; then
            status="running"
            health=$(agents2::get_health_status)
            
            # Check API
            if agents2::is_healthy; then
                api_responsive=true
            fi
            
            # Check VNC port
            if resources::is_service_running "$AGENTS2_VNC_PORT"; then
                vnc_accessible=true
            fi
        else
            status="stopped"
        fi
    fi
    
    cat << EOF
{
    "status": "$status",
    "health": "$health",
    "api_responsive": $api_responsive,
    "vnc_accessible": $vnc_accessible,
    "ports": {
        "api": $AGENTS2_PORT,
        "vnc": $AGENTS2_VNC_PORT
    },
    "urls": {
        "api": "$AGENTS2_BASE_URL",
        "vnc": "$AGENTS2_VNC_URL"
    }
}
EOF
}