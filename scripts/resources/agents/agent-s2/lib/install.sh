#!/usr/bin/env bash
# Agent S2 Installation Logic
# Complete installation workflow and configuration

#######################################
# Update Vrooli configuration with Agent S2 settings
# Returns: 0 if successful, 1 if failed
#######################################
agents2::update_config() {
    # Create comprehensive configuration JSON
    local additional_config
    additional_config=$(cat <<EOF
{
    "features": {
        "screenshot": true,
        "guiAutomation": true,
        "planning": true,
        "multiStepTasks": true,
        "crossPlatform": true
    },
    "llm": {
        "provider": "$AGENTS2_LLM_PROVIDER",
        "model": "$AGENTS2_LLM_MODEL",
        "apiKeyEnvVar": "AGENTS2_API_KEY"
    },
    "display": {
        "type": "virtual",
        "display": "$AGENTS2_DISPLAY",
        "resolution": "$AGENTS2_SCREEN_RESOLUTION",
        "vncEnabled": true,
        "vncPort": $AGENTS2_VNC_PORT,
        "vncPasswordSet": true
    },
    "api": {
        "version": "1.0.0",
        "healthEndpoint": "/health",
        "capabilitiesEndpoint": "/capabilities",
        "screenshotEndpoint": "/screenshot",
        "executeEndpoint": "/execute",
        "tasksEndpoint": "/tasks",
        "mouseEndpoint": "/mouse/position"
    },
    "container": {
        "name": "$AGENTS2_CONTAINER_NAME",
        "image": "$AGENTS2_IMAGE_NAME",
        "network": "$AGENTS2_NETWORK_NAME"
    },
    "security": {
        "sandboxed": true,
        "hostDisplayAccess": $([[ "$AGENTS2_ENABLE_HOST_DISPLAY" == "yes" ]] && echo "true" || echo "false"),
        "restrictedApplications": ["passwords", "keychain", "1password", "bitwarden"],
        "runAsNonRoot": true,
        "user": "$AGENTS2_USER"
    },
    "resources": {
        "memory": "$AGENTS2_MEMORY_LIMIT",
        "cpus": "$AGENTS2_CPU_LIMIT",
        "shmSize": "$AGENTS2_SHM_SIZE"
    },
    "supportedTasks": [
        "screenshot",
        "click",
        "double_click",
        "right_click",
        "type_text",
        "key_press",
        "mouse_move",
        "drag_drop",
        "scroll",
        "automation_sequence"
    ]
}
EOF
)
    
    if resources::update_config "agents" "agent-s2" "$AGENTS2_BASE_URL" "$additional_config"; then
        log::success "$MSG_CONFIG_UPDATE_SUCCESS"
        return 0
    else
        log::warn "$MSG_CONFIG_UPDATE_FAILED"
        log::info "Agent S2 is installed but may need manual configuration in Vrooli"
        return 1
    fi
}

#######################################
# Display installation success information
#######################################
agents2::show_installation_success() {
    log::success "$MSG_INSTALL_SUCCESS"
    
    # Display access information
    echo
    log::header "🤖 Agent S2 Access Information"
    log::info "API URL: $AGENTS2_BASE_URL"
    log::info "VNC URL: $AGENTS2_VNC_URL"
    log::info "VNC Password: $AGENTS2_VNC_PASSWORD"
    log::info "Health Check: ${AGENTS2_BASE_URL}/health"
    log::info "Capabilities: ${AGENTS2_BASE_URL}/capabilities"
    
    echo
    log::header "🔧 Configuration"
    log::info "LLM Provider: $AGENTS2_LLM_PROVIDER"
    log::info "LLM Model: $AGENTS2_LLM_MODEL"
    log::info "Display: $AGENTS2_DISPLAY (Virtual)"
    log::info "Resolution: $AGENTS2_SCREEN_RESOLUTION"
    log::info "Host Display Access: $AGENTS2_ENABLE_HOST_DISPLAY"
    
    echo
    log::header "🎯 Next Steps"
    log::info "1. Access the API at: $AGENTS2_BASE_URL"
    log::info "2. Connect via VNC to view the virtual display: $AGENTS2_VNC_URL"
    log::info "3. Test with: $0 --action usage"
    log::info "4. View logs: $0 --action logs"
    log::info "5. Check the API docs: ${AGENTS2_BASE_URL}/docs"
}

#######################################
# Complete Agent S2 installation
# Returns: 0 if successful, 1 if failed
#######################################
agents2::install_service() {
    log::header "$MSG_INSTALLING"
    
    # Start rollback context
    resources::start_rollback_context "install_agent_s2"
    
    # Check if already installed
    if agents2::container_exists && agents2::is_running && [[ "$FORCE" != "yes" ]]; then
        log::info "$MSG_ALREADY_INSTALLED"
        log::info "$MSG_USE_FORCE"
        return 0
    fi
    
    # Validate Docker is available
    if ! resources::ensure_docker; then
        log::error "$MSG_DOCKER_REQUIRED"
        return 1
    fi
    
    # Validate port availability
    if ! resources::validate_port "agent-s2" "$AGENTS2_PORT" "$FORCE"; then
        log::error "Port validation failed for Agent S2 API port: $AGENTS2_PORT"
        return 1
    fi
    
    if ! resources::validate_port "agent-s2-vnc" "$AGENTS2_VNC_PORT" "$FORCE"; then
        log::error "Port validation failed for Agent S2 VNC port: $AGENTS2_VNC_PORT"
        return 1
    fi
    
    # Create directories
    if ! agents2::create_directories; then
        resources::execute_rollback
        return 1
    fi
    
    # Add rollback for directories
    resources::add_rollback_action \
        "Remove Agent S2 directories" \
        "rm -rf \"$AGENTS2_DATA_DIR\"" \
        10
    
    # Build Docker image
    if ! agents2::docker_build; then
        resources::execute_rollback
        return 1
    fi
    
    # Add rollback for Docker image
    resources::add_rollback_action \
        "Remove Docker image" \
        "docker rmi \"$AGENTS2_IMAGE_NAME\" 2>/dev/null || true" \
        20
    
    # Create network
    if ! agents2::docker_create_network; then
        resources::execute_rollback
        return 1
    fi
    
    # Add rollback for network
    resources::add_rollback_action \
        "Remove Docker network" \
        "docker network rm \"$AGENTS2_NETWORK_NAME\" 2>/dev/null || true" \
        5
    
    # Start container
    if ! agents2::docker_start; then
        resources::execute_rollback
        return 1
    fi
    
    # Add rollback for container
    resources::add_rollback_action \
        "Stop and remove container" \
        "docker stop \"$AGENTS2_CONTAINER_NAME\" 2>/dev/null || true; docker rm \"$AGENTS2_CONTAINER_NAME\" 2>/dev/null || true" \
        25
    
    # Clear rollback since core installation succeeded
    log::info "Agent S2 core installation completed successfully"
    ROLLBACK_ACTIONS=()
    OPERATION_ID=""
    
    # Update Vrooli configuration
    agents2::update_config
    
    # Show success information
    agents2::show_installation_success
    
    # Show status
    echo
    agents2::show_status
    
    return 0
}

#######################################
# Uninstall Agent S2
# Returns: 0 if successful, 1 if failed
#######################################
agents2::uninstall_service() {
    log::header "$MSG_UNINSTALLING"
    
    if ! flow::is_yes "$YES"; then
        log::warn "$MSG_UNINSTALL_WARNING"
        read -p "Are you sure you want to continue? (y/N): " -r
        if [[ ! $REPLY =~ ^[Yy]$ ]]; then
            log::info "$MSG_UNINSTALL_CANCELLED"
            return 0
        fi
    fi
    
    # Clean up Docker resources
    agents2::docker_cleanup
    
    # Ask about data removal
    if [[ -d "$AGENTS2_DATA_DIR" ]]; then
        if ! flow::is_yes "$YES"; then
            read -p "Remove Agent S2 data directory? (y/N): " -r
            if [[ $REPLY =~ ^[Yy]$ ]]; then
                rm -rf "$AGENTS2_DATA_DIR"
                log::info "Data directory removed"
            fi
        fi
    fi
    
    # Remove from Vrooli config
    resources::remove_config "agents" "agent-s2"
    
    log::success "$MSG_UNINSTALL_SUCCESS"
    return 0
}