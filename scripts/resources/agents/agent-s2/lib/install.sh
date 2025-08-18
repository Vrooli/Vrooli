#!/usr/bin/env bash
# Agent S2 Installation Logic

# Source var.sh first to get proper directory variables
AGENT_S2_LIB_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
# shellcheck disable=SC1091
source "${AGENT_S2_LIB_DIR}/../../../../lib/utils/var.sh" 2>/dev/null || true

# Source trash system for safe removal using var_ variables
# shellcheck disable=SC1091
source "${var_LIB_SYSTEM_DIR}/trash.sh" 2>/dev/null || true
# Complete installation workflow and configuration

#######################################
# Update Vrooli configuration with Agent S2 settings
# Returns: 0 if successful, 1 if failed
#######################################
agents2::update_config() {
    # Create comprehensive dual-mode configuration JSON
    local additional_config
    additional_config=$(cat <<EOF
{
    "features": {
        "screenshot": true,
        "guiAutomation": true,
        "planning": true,
        "multiStepTasks": true,
        "crossPlatform": true,
        "dualMode": true,
        "hostModeSupport": $([[ "${AGENT_S2_HOST_MODE_ENABLED:-false}" == "true" ]] && echo "true" || echo "false"),
        "securityMonitoring": true,
        "auditLogging": $([[ "$AGENT_S2_HOST_AUDIT_LOGGING" == "true" ]] && echo "true" || echo "false")
    },
    "modes": {
        "current": "${MODE:-sandbox}",
        "available": ["sandbox", "host"],
        "sandbox": {
            "enabled": true,
            "description": "Secure isolated environment",
            "applications": ["firefox-esr", "mousepad", "gedit", "gnome-calculator", "xterm"],
            "security": "high",
            "hostAccess": false
        },
        "host": {
            "enabled": $([[ "${AGENT_S2_HOST_MODE_ENABLED:-false}" == "true" ]] && echo "true" || echo "false"),
            "description": "Extended host system access",
            "applications": "${AGENT_S2_HOST_APPS:-*}",
            "security": "medium",
            "hostAccess": true,
            "displayAccess": $([[ "$AGENT_S2_HOST_DISPLAY_ACCESS" == "true" ]] && echo "true" || echo "false"),
            "mounts": "${AGENT_S2_HOST_MOUNTS:-[]}",
            "securityProfile": "${AGENT_S2_HOST_SECURITY_PROFILE:-agent-s2-host}"
        }
    },
    "llm": {
        "provider": "$AGENTS2_LLM_PROVIDER",
        "model": "$AGENTS2_LLM_MODEL",
        "apiKeyEnvVar": "AGENTS2_API_KEY"
    },
    "display": {
        "type": "$([[ "${MODE:-sandbox}" == "host" && "$AGENT_S2_HOST_DISPLAY_ACCESS" == "true" ]] && echo "host" || echo "virtual")",
        "display": "$AGENTS2_DISPLAY",
        "resolution": "$AGENTS2_SCREEN_RESOLUTION",
        "vncEnabled": true,
        "vncPort": $AGENTS2_VNC_PORT,
        "vncPasswordSet": true
    },
    "api": {
        "version": "2.0.0",
        "healthEndpoint": "/health",
        "capabilitiesEndpoint": "/capabilities",
        "screenshotEndpoint": "/screenshot",
        "executeEndpoint": "/execute",
        "tasksEndpoint": "/tasks",
        "mouseEndpoint": "/mouse/position",
        "modesEndpoint": "/modes",
        "switchModeEndpoint": "/modes/switch",
        "securityEndpoint": "/modes/security"
    },
    "container": {
        "name": "$AGENTS2_CONTAINER_NAME",
        "image": "$AGENTS2_IMAGE_NAME",
        "network": "$AGENTS2_NETWORK_NAME"
    },
    "security": {
        "sandboxed": $([[ "${MODE:-sandbox}" == "sandbox" ]] && echo "true" || echo "false"),
        "hostDisplayAccess": $([[ "$AGENTS2_ENABLE_HOST_DISPLAY" == "yes" ]] && echo "true" || echo "false"),
        "restrictedApplications": ["passwords", "keychain", "1password", "bitwarden"],
        "runAsNonRoot": true,
        "user": "$AGENTS2_USER",
        "appArmorProfile": "$([[ "${MODE:-sandbox}" == "host" ]] && echo "docker-agent-s2-host" || echo "docker-default")",
        "auditLogging": $([[ "$AGENT_S2_HOST_AUDIT_LOGGING" == "true" ]] && echo "true" || echo "false"),
        "threatDetection": true,
        "securityMonitoring": true
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
        "automation_sequence",
        "mode_switch",
        "environment_discovery",
        "application_launch",
        $([[ "${MODE:-sandbox}" == "host" ]] && echo '"host_integration",' || echo "")
        $([[ "${MODE:-sandbox}" == "host" ]] && echo '"file_system_access",' || echo "")
        "security_validation"
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
    log::info "Operating Mode: ${MODE:-sandbox}"
    log::info "Host Mode Enabled: ${AGENT_S2_HOST_MODE_ENABLED:-false}"
    # Show actual provider based on what will be used
    local actual_provider="$AGENTS2_LLM_PROVIDER"
    if [[ -z "${AGENTS2_OPENAI_API_KEY}" && -z "${AGENTS2_ANTHROPIC_API_KEY}" ]]; then
        actual_provider="ollama (auto-detected)"
    fi
    log::info "LLM Provider: $actual_provider"
    log::info "LLM Model: $AGENTS2_LLM_MODEL"
    log::info "Display: $AGENTS2_DISPLAY ($([[ "${MODE:-sandbox}" == "host" && "$AGENT_S2_HOST_DISPLAY_ACCESS" == "true" ]] && echo "Host" || echo "Virtual"))"
    log::info "Resolution: $AGENTS2_SCREEN_RESOLUTION"
    log::info "Host Display Access: $AGENTS2_ENABLE_HOST_DISPLAY"
    log::info "Security Profile: $([[ "${MODE:-sandbox}" == "host" ]] && echo "${AGENT_S2_HOST_SECURITY_PROFILE:-docker-agent-s2-host}" || echo "docker-default")"
    log::info "Audit Logging: ${AGENT_S2_HOST_AUDIT_LOGGING:-false}"
    
    echo
    log::header "🎯 Next Steps"
    log::info "1. Access the API at: $AGENTS2_BASE_URL"
    log::info "2. Connect via VNC to view the virtual display: $AGENTS2_VNC_URL"
    log::info "3. View current mode: $0 --action mode"
    log::info "4. Test functionality: $0 --action usage"
    log::info "5. Check mode capabilities: ${AGENTS2_BASE_URL}/modes/current"
    log::info "6. View logs: $0 --action logs"
    log::info "7. Check the API docs: ${AGENTS2_BASE_URL}/docs"
    
    if [[ "${AGENT_S2_HOST_MODE_ENABLED:-false}" == "true" ]]; then
        echo
        log::header "🔐 Host Mode Available"
        log::info "Switch to host mode: $0 --action switch-mode --target-mode host"
        log::info "Test host mode: $0 --action test-mode --mode host"
        log::info "Security monitoring: ${AGENTS2_BASE_URL}/modes/security"
        
        if [[ "${MODE:-sandbox}" == "host" ]]; then
            log::warning "⚠️  Currently running in HOST MODE with elevated privileges"
            log::info "Audit logs: /var/log/agent-s2-audit/"
            log::info "Switch to sandbox: $0 --action switch-mode --target-mode sandbox"
        fi
    else
        echo
        log::header "🔒 Security Note"
        log::info "Running in SANDBOX MODE (secure isolation)"
        log::info "To enable host mode: Set AGENT_S2_HOST_MODE_ENABLED=true"
    fi
}

#######################################
# Install security components for host mode
# Returns: 0 if successful, 1 if failed
#######################################
agents2::install_security_components() {
    local security_script="${SCRIPT_DIR}/security/install-security.sh"
    
    if [[ ! -f "$security_script" ]]; then
        log::warning "Security installation script not found: $security_script"
        log::info "Host mode will run without enhanced security features"
        return 0
    fi
    
    # Check if we can run with sudo
    if [[ $EUID -eq 0 ]]; then
        log::info "Installing security components as root..."
        bash "$security_script" install
    elif command -v sudo >/dev/null 2>&1; then
        log::info "Installing security components with sudo..."
        sudo bash "$security_script" install
    else
        log::warning "Cannot install security components (no root/sudo access)"
        log::info "Host mode will run with basic Docker security only"
        log::info "To install security features manually:"
        log::info "  sudo $security_script install"
        return 0
    fi
    
    local exit_code=$?
    if [[ $exit_code -eq 0 ]]; then
        log::success "Security components installed successfully"
        
        # Set security environment variables for current session
        export AGENT_S2_HOST_SECURITY_PROFILE="docker-agent-s2-host"
        export AGENT_S2_HOST_AUDIT_LOGGING="true"
        
        return 0
    else
        log::warning "Security component installation failed (exit code: $exit_code)"
        log::info "Host mode will continue with basic security"
        return 0  # Don't fail the entire installation
    fi
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
        "if command -v trash::safe_remove >/dev/null 2>&1; then trash::safe_remove '$AGENTS2_DATA_DIR' --no-confirm; else rm -rf '$AGENTS2_DATA_DIR'; fi" \
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
    
    # Install security components if host mode is enabled
    if [[ "${AGENT_S2_HOST_MODE_ENABLED:-false}" == "true" ]]; then
        log::info "Installing security components for host mode..."
        agents2::install_security_components
    fi
    
    # Update Vrooli configuration
    agents2::update_config
    
    # Show success information
    agents2::show_installation_success
    
    # Show status
    echo
    agents2::show_status
    
    # Auto-install CLI if available
    # shellcheck disable=SC1091
    "${var_SCRIPTS_RESOURCES_LIB_DIR}/cli/install-resource-cli.sh" "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)" 2>/dev/null || true
    
    return 0
}

#######################################
# Uninstall security components
# Returns: 0 if successful, 1 if failed
#######################################
agents2::uninstall_security_components() {
    local security_script="${SCRIPT_DIR}/security/install-security.sh"
    
    if [[ ! -f "$security_script" ]]; then
        log::debug "Security installation script not found, skipping security cleanup"
        return 0
    fi
    
    log::info "Removing security components..."
    
    # Check if we can run with sudo
    if [[ $EUID -eq 0 ]]; then
        bash "$security_script" uninstall 2>/dev/null || true
    elif command -v sudo >/dev/null 2>&1; then
        sudo bash "$security_script" uninstall 2>/dev/null || true
    else
        log::warning "Cannot remove security components (no root/sudo access)"
        log::info "You may need to manually remove:"
        log::info "  - AppArmor profiles: /etc/apparmor.d/docker-agent-s2-*"
        log::info "  - Audit logs: /var/log/agent-s2-audit/"
        log::info "  - Security configs: /etc/agent-s2-security.conf"
    fi
    
    log::info "Security components cleanup attempted"
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
    
    # CRITICAL: Clean up NAT redirection rules to prevent traffic hijacking
    log::info "Cleaning up NAT redirection rules..."
    agents2::cleanup_nat_rules || log::warn "Failed to clean NAT rules - manual cleanup may be required"
    
    # Uninstall security components if they were installed
    agents2::uninstall_security_components
    
    # Ask about data removal
    if [[ -d "$AGENTS2_DATA_DIR" ]]; then
        if ! flow::is_yes "$YES"; then
            read -p "Remove Agent S2 data directory? (y/N): " -r
            if [[ $REPLY =~ ^[Yy]$ ]]; then
                if command -v trash::safe_remove >/dev/null 2>&1; then
                    trash::safe_remove "$AGENTS2_DATA_DIR" --no-confirm
                else
                    rm -rf "$AGENTS2_DATA_DIR"
                fi
                log::info "Data directory removed"
            fi
        fi
    fi
    
    # Remove from Vrooli config
    resources::remove_config "agents" "agent-s2"
    
    log::success "$MSG_UNINSTALL_SUCCESS"
    return 0
}

# Export functions for subshell availability
export -f agents2::update_config
export -f agents2::show_installation_success
export -f agents2::install_security_components
export -f agents2::install_service
export -f agents2::uninstall_security_components
export -f agents2::uninstall_service