#!/usr/bin/env bash

# Ollama Installation Functions
# This file contains all installation, uninstallation, and verification functions

#######################################
# Install Ollama binary with rollback support
#######################################
ollama::install_binary() {
    if ollama::is_installed && [[ "$FORCE" != "yes" ]]; then
        log::info "$MSG_OLLAMA_ALREADY_INSTALLED"
        return 0
    fi
    
    log::info "$MSG_OLLAMA_INSTALLING"
    
    # Download and install using official installer
    local install_script
    install_script=$(mktemp) || {
        log::error "$MSG_DOWNLOAD_FAILED"
        return 1
    }
    
    # Add rollback action for cleanup
    resources::add_rollback_action \
        "Clean up Ollama installer script" \
        "rm -f \"$install_script\"" \
        1
    
    if ! resources::download_file "https://ollama.com/install.sh" "$install_script"; then
        log::error "$MSG_DOWNLOAD_FAILED"
        return 1
    fi
    
    # Verify the installer is valid
    if [[ ! -s "$install_script" ]]; then
        log::error "$MSG_INSTALLER_EMPTY"
        return 1
    fi
    
    # Make executable and run
    chmod +x "$install_script"
    
    if resources::can_sudo; then
        if sudo bash "$install_script"; then
            log::success "$MSG_INSTALLER_SUCCESS"
        else
            log::error "$MSG_INSTALLER_FAILED"
            return 1
        fi
    else
        log::error "$MSG_SUDO_REQUIRED"
        return 1
    fi
    
    # Clean up installer
    rm -f "$install_script"
    
    # Verify installation
    if ollama::is_installed; then
        log::success "$MSG_BINARY_INSTALL_SUCCESS"
        
        # Add rollback action for binary removal
        resources::add_rollback_action \
            "Remove Ollama binary" \
            "sudo rm -f /usr/local/bin/ollama" \
            20
        
        return 0
    else
        log::error "$MSG_BINARY_INSTALL_FAILED"
        return 1
    fi
}

#######################################
# Create Ollama user if it doesn't exist
#######################################
ollama::create_user() {
    if id "$OLLAMA_USER" &>/dev/null; then
        log::info "User $OLLAMA_USER already exists"
        return 0
    fi
    
    if ! resources::can_sudo; then
        log::error "$MSG_USER_SUDO_REQUIRED"
        return 1
    fi
    
    log::info "Creating $OLLAMA_USER user..."
    
    if sudo useradd -r -s /bin/false -d /usr/share/ollama -c "Ollama service user" "$OLLAMA_USER"; then
        log::success "$MSG_USER_CREATE_SUCCESS"
        
        # Add rollback action for user removal
        resources::add_rollback_action \
            "Remove Ollama user" \
            "sudo userdel $OLLAMA_USER 2>/dev/null || true" \
            15
        
        return 0
    else
        log::error "$MSG_USER_CREATE_FAILED"
        return 1
    fi
}

#######################################
# Install Ollama systemd service with rollback support
#######################################
ollama::install_service() {
    # Check if systemd service already exists (installed by official installer)
    if systemctl list-unit-files | grep -q "^${OLLAMA_SERVICE_NAME}.service"; then
        log::info "Ollama systemd service already exists (installed by official installer)"
        
        # Just ensure it's enabled
        if resources::can_sudo; then
            sudo systemctl enable "$OLLAMA_SERVICE_NAME" 2>/dev/null || true
        fi
        
        return 0
    fi
    
    # Only create service if it doesn't exist
    local service_content="[Unit]
Description=Ollama Service
After=network-online.target

[Service]
ExecStart=/usr/local/bin/ollama serve
User=$OLLAMA_USER
Group=$OLLAMA_USER
Restart=always
RestartSec=3
Environment=\"PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin\"
Environment=\"OLLAMA_HOST=0.0.0.0:$OLLAMA_PORT\"

[Install]
WantedBy=default.target"
    
    log::info "Installing Ollama systemd service..."
    
    if ! resources::can_sudo; then
        resources::handle_error \
            "Sudo privileges required to install systemd service" \
            "permission" \
            "Run 'sudo -v' to authenticate and retry"
        return 1
    fi
    
    if resources::install_systemd_service "$OLLAMA_SERVICE_NAME" "$service_content"; then
        log::success "$MSG_SERVICE_INSTALL_SUCCESS"
        
        # Add rollback action for service removal
        resources::add_rollback_action \
            "Remove Ollama systemd service" \
            "sudo systemctl stop $OLLAMA_SERVICE_NAME 2>/dev/null || true; sudo systemctl disable $OLLAMA_SERVICE_NAME 2>/dev/null || true; sudo rm -f /etc/systemd/system/${OLLAMA_SERVICE_NAME}.service; sudo systemctl daemon-reload" \
            18
        
        return 0
    else
        resources::handle_error \
            "Failed to install Ollama systemd service" \
            "system" \
            "Check systemd status and permissions: systemctl status systemd"
        return 1
    fi
}

#######################################
# Complete Ollama installation with comprehensive error handling
#######################################
ollama::install() {
    log::header "🤖 Installing Ollama"
    
    # Start rollback context
    resources::start_rollback_context "install_ollama"
    
    # Check if already installed and running
    if ollama::is_installed && ollama::is_running && [[ "$FORCE" != "yes" ]]; then
        log::info "Ollama is already installed and running"
        log::info "Use --force yes to reinstall, or --action status to check current state"
        return 0
    fi
    
    # Validate prerequisites
    if ! resources::can_sudo; then
        resources::handle_error \
            "Ollama installation requires sudo privileges for system service management" \
            "permission" \
            "Run 'sudo -v' to authenticate, then retry installation"
        return 1
    fi
    
    # Validate port assignment
    local port_validation_result
    resources::validate_port "ollama" "$OLLAMA_PORT" "$FORCE"
    port_validation_result=$?
    
    case "$port_validation_result" in
        0)
            log::info "Port $OLLAMA_PORT validated successfully for Ollama"
            ;;
        1)
            log::error "$MSG_PORT_CONFLICT"
            log::info "You can set a custom port with: export OLLAMA_CUSTOM_PORT=<port>"
            return 1
            ;;
        2)
            log::warn "$MSG_PORT_WARNING"
            log::info "Installation may succeed if the existing service is compatible"
            ;;
        *)
            log::warn "$MSG_PORT_UNEXPECTED"
            ;;
    esac
    
    # Check network connectivity
    if ! curl -s --max-time 5 https://ollama.com > /dev/null; then
        resources::handle_error \
            "Cannot connect to ollama.com for installation" \
            "network" \
            "Check internet connection and firewall settings"
        return 1
    fi
    
    # Install binary with error handling
    if ! ollama::install_binary; then
        resources::handle_error \
            "Failed to install Ollama binary" \
            "system" \
            "Check system requirements and available disk space"
        return 1
    fi
    
    # Create user with rollback
    if ! ollama::create_user; then
        resources::handle_error \
            "Failed to create ollama user" \
            "system" \
            "Check user creation permissions and existing user conflicts"
        return 1
    fi
    
    # Install service with rollback
    if ! ollama::install_service; then
        resources::handle_error \
            "Failed to install Ollama systemd service" \
            "system" \
            "Check systemd status and service file permissions"
        return 1
    fi
    
    # Start service with error handling
    if ! ollama::start; then
        resources::handle_error \
            "Failed to start Ollama service" \
            "system" \
            "Check service logs: journalctl -u ollama -n 20"
        return 1
    fi
    
    # Install models with error handling
    if ! ollama::install_models; then
        log::warn "$MSG_MODELS_INSTALL_FAILED"
        log::info "You can install models manually later with: ollama pull <model-name>"
    fi
    
    # At this point, Ollama is successfully installed
    # Clear rollback context to prevent removing it if config update fails
    log::info "Ollama core installation completed successfully"
    ROLLBACK_ACTIONS=()
    OPERATION_ID=""
    
    # Update Vrooli configuration (non-critical)
    if ! ollama::update_config; then
        log::warn "$MSG_CONFIG_UPDATE_FAILED"
        log::info "You may need to configure Vrooli manually to use this Ollama instance"
    fi
    
    # Validate installation (informational only, don't fail)
    if ! ollama::verify_installation; then
        log::warn "$MSG_VERIFICATION_FAILED"
        log::info "Check service status with: systemctl status ollama"
    else
        log::success "$MSG_OLLAMA_INSTALL_SUCCESS"
    fi
    
    # Show status
    echo
    ollama::status
}

#######################################
# Verify Ollama installation is complete and functional
#######################################
ollama::verify_installation() {
    log::info "Verifying Ollama installation..."
    
    local verification_errors=()
    local verification_warnings=()
    
    # Check binary installation
    if ! ollama::is_installed; then
        verification_errors+=("Ollama binary not found in PATH")
    else
        log::success "$MSG_STATUS_BINARY_OK"
    fi
    
    # Check user exists
    if ! id "$OLLAMA_USER" &>/dev/null; then
        verification_errors+=("Ollama user '$OLLAMA_USER' does not exist")
    else
        log::success "$MSG_STATUS_USER_OK"
    fi
    
    # Check systemd service (check if service exists anywhere in systemd)
    if ! systemctl status "$OLLAMA_SERVICE_NAME" &>/dev/null && ! systemctl list-unit-files | grep -q "^${OLLAMA_SERVICE_NAME}.service"; then
        verification_errors+=("Ollama systemd service not found")
    else
        log::success "$MSG_STATUS_SERVICE_OK"
        
        # Check if service is enabled
        if ! systemctl is-enabled "$OLLAMA_SERVICE_NAME" &>/dev/null; then
            verification_warnings+=("Ollama service is not enabled for auto-start")
        else
            log::success "$MSG_STATUS_SERVICE_ENABLED"
        fi
    fi
    
    # Check service is running
    if ! resources::is_service_active "$OLLAMA_SERVICE_NAME"; then
        verification_errors+=("Ollama service is not running")
    else
        log::success "$MSG_STATUS_SERVICE_ACTIVE"
    fi
    
    # Check port accessibility
    if ! resources::is_service_running "$OLLAMA_PORT"; then
        verification_errors+=("Ollama is not listening on port $OLLAMA_PORT")
    else
        log::success "$MSG_STATUS_PORT_OK"
    fi
    
    # Check API health
    if ! ollama::is_healthy; then
        verification_warnings+=("Ollama API is not responding to health checks")
    else
        log::success "$MSG_STATUS_API_OK"
    fi
    
    # Check models are available
    local model_count=0
    if system::is_command "ollama"; then
        # Count lines that contain model names (skip header)
        model_count=$(ollama list 2>/dev/null | tail -n +2 | wc -l || echo "0")
        if [[ "$model_count" =~ ^[0-9]+$ ]] && [[ $model_count -gt 0 ]]; then
            log::success "$MSG_MODELS_COUNT"
        else
            verification_warnings+=("No models are installed")
        fi
    fi
    
    # Check configuration
    if [[ -f "$VROOLI_RESOURCES_CONFIG" ]]; then
        if system::is_command "jq"; then
            local config_exists
            config_exists=$(jq -r '.services.ai.ollama // empty' "$VROOLI_RESOURCES_CONFIG" 2>/dev/null)
            if [[ -n "$config_exists" && "$config_exists" != "null" ]]; then
                log::success "✅ Ollama configured in Vrooli resources"
            else
                verification_warnings+=("Ollama not found in Vrooli resource configuration")
            fi
        fi
    else
        verification_warnings+=("Vrooli resource configuration file not found")
    fi
    
    # Print verification summary
    echo
    log::header "🔍 Installation Verification Summary"
    
    if [[ ${#verification_errors[@]} -eq 0 ]]; then
        log::success "✅ Ollama installation verification passed!"
        
        if [[ ${#verification_warnings[@]} -gt 0 ]]; then
            echo
            log::warn "⚠️  Warnings found:"
            for warning in "${verification_warnings[@]}"; do
                log::warn "  • $warning"
            done
            
            echo
            log::info "💡 These warnings don't prevent Ollama from working but may affect functionality"
        fi
        
        echo
        log::info "🚀 Ollama is ready to use!"
        log::info "   Base URL: $OLLAMA_BASE_URL"
        log::info "   Test API: curl $OLLAMA_BASE_URL/api/tags"
        if [[ $model_count -gt 0 ]]; then
            log::info "   Chat with a model: ollama run llama3.1:8b"
        else
            log::info "   Install a model: ollama pull llama3.1:8b"
        fi
        
        return 0
    else
        log::error "❌ Ollama installation verification failed!"
        echo
        log::error "Errors found:"
        for error in "${verification_errors[@]}"; do
            log::error "  • $error"
        done
        
        if [[ ${#verification_warnings[@]} -gt 0 ]]; then
            echo
            log::warn "Additional warnings:"
            for warning in "${verification_warnings[@]}"; do
                log::warn "  • $warning"
            done
        fi
        
        echo
        log::info "💡 Try reinstalling with: $0 --action install --force yes"
        
        return 1
    fi
}

#######################################
# Uninstall Ollama
#######################################
ollama::uninstall() {
    log::header "🗑️  Uninstalling Ollama"
    
    if ! flow::is_yes "$YES"; then
        log::warn "This will completely remove Ollama, including all models and data"
        read -p "Are you sure you want to continue? (y/N): " -r
        if [[ ! $REPLY =~ ^[Yy]$ ]]; then
            log::info "Uninstall cancelled"
            return 0
        fi
    fi
    
    # Stop service
    if resources::is_service_active "$OLLAMA_SERVICE_NAME"; then
        ollama::stop
    fi
    
    # Remove systemd service
    if resources::can_sudo && [[ -f "/etc/systemd/system/${OLLAMA_SERVICE_NAME}.service" ]]; then
        sudo systemctl disable "$OLLAMA_SERVICE_NAME" 2>/dev/null || true
        sudo rm -f "/etc/systemd/system/${OLLAMA_SERVICE_NAME}.service"
        sudo systemctl daemon-reload
        log::info "Systemd service removed"
    fi
    
    # Remove binary
    if resources::can_sudo && [[ -f "${OLLAMA_INSTALL_DIR}/ollama" ]]; then
        sudo rm -f "${OLLAMA_INSTALL_DIR}/ollama"
        log::info "Ollama binary removed"
    fi
    
    # Remove user (optional - may have other uses)
    if id "$OLLAMA_USER" &>/dev/null && resources::can_sudo; then
        read -p "Remove $OLLAMA_USER user? (y/N): " -r
        if [[ $REPLY =~ ^[Yy]$ ]]; then
            sudo userdel "$OLLAMA_USER" 2>/dev/null || true
            log::info "User $OLLAMA_USER removed"
        fi
    fi
    
    # Remove from Vrooli config
    resources::remove_config "ai" "ollama"
    
    log::success "✅ Ollama uninstalled successfully"
}