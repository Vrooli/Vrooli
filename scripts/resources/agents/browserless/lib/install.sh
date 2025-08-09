#!/usr/bin/env bash
# Browserless Installation Logic
# Complete installation workflow and configuration

#######################################
# Create Browserless data directories
# Returns: 0 if successful, 1 if failed
#######################################
browserless::create_directories() {
    log::info "${MSG_CREATING_DIRS}"
    
    if mkdir -p "$BROWSERLESS_DATA_DIR"; then
        log::success "${MSG_DIRECTORIES_CREATED}"
        
        # Add rollback action
        resources::add_rollback_action \
            "Remove Browserless data directory" \
            "rm -rf $BROWSERLESS_DATA_DIR 2>/dev/null || true" \
            10
        
        return 0
    else
        log::error "${MSG_CREATE_DIRS_FAILED}"
        return 1
    fi
}

#######################################
# Update Vrooli configuration with Browserless settings
# Returns: 0 if successful, 1 if failed
#######################################
browserless::update_config() {
    # Create JSON with proper escaping
    local additional_config
    additional_config=$(cat <<EOF
{
    "features": {
        "screenshots": true,
        "pdf": true,
        "scraping": true,
        "automation": true
    },
    "browser": {
        "maxConcurrency": "$BROWSERLESS_MAX_BROWSERS",
        "headless": $([[ "$BROWSERLESS_HEADLESS" == "yes" ]] && echo "true" || echo "false"),
        "timeout": "$BROWSERLESS_TIMEOUT"
    },
    "api": {
        "version": "v1",
        "statusEndpoint": "/pressure",
        "screenshotEndpoint": "/screenshot",
        "pdfEndpoint": "/pdf",
        "contentEndpoint": "/content",
        "functionEndpoint": "/function",
        "scrapeEndpoint": "/scrape"
    },
    "container": {
        "name": "$BROWSERLESS_CONTAINER_NAME",
        "image": "$BROWSERLESS_IMAGE"
    }
}
EOF
)
    
    if resources::update_config "agents" "browserless" "$BROWSERLESS_BASE_URL" "$additional_config"; then
        return 0
    else
        log::warn "${MSG_CONFIG_UPDATE_FAILED}"
        log::info "Browserless is installed but may need manual configuration in Vrooli"
        return 1
    fi
}

#######################################
# Display installation success information
#######################################
browserless::show_installation_success() {
    log::success "✅ Browserless is running and healthy on port $BROWSERLESS_PORT"
    
    # Display access information
    echo
    log::header "🌐 Browserless Access Information"
    log::info "URL: $BROWSERLESS_BASE_URL"
    log::info "Status Check: $BROWSERLESS_BASE_URL/pressure"
    log::info "Max Browsers: $BROWSERLESS_MAX_BROWSERS"
    log::info "Headless Mode: $BROWSERLESS_HEADLESS"
    log::info "Timeout: ${BROWSERLESS_TIMEOUT}ms"
    
    echo
    log::header "🎯 Next Steps"
    log::info "1. Access Browserless at: $BROWSERLESS_BASE_URL"
    log::info "2. Use the API endpoints for browser automation"
    log::info "3. Test with: $0 --action usage"
    log::info "4. Check the docs: https://www.browserless.io/docs/"
}

#######################################
# Complete Browserless installation
# Returns: 0 if successful, 1 if failed
#######################################
browserless::install_service() {
    log::header "🎭 Installing Browserless Browser Automation (Docker)"
    
    # Start rollback context
    resources::start_rollback_context "install_browserless_docker"
    
    # Check if already installed
    if ! browserless::check_existing_installation; then
        return 0
    fi
    
    # Validate prerequisites
    if ! browserless::validate_prerequisites; then
        return 1
    fi
    
    # Create directories
    if ! browserless::create_directories; then
        resources::handle_error \
            "${MSG_CREATE_DIRS_FAILED}" \
            "system" \
            "Check directory permissions"
        return 1
    fi
    
    # Create Docker network
    browserless::create_network
    
    # Start Browserless container
    if ! browserless::docker_run "$BROWSERLESS_MAX_BROWSERS" "$BROWSERLESS_TIMEOUT" "$BROWSERLESS_HEADLESS"; then
        resources::handle_error \
            "${MSG_START_CONTAINER_FAILED}" \
            "system" \
            "Check Docker logs: docker logs $BROWSERLESS_CONTAINER_NAME"
        return 1
    fi
    
    # Wait for service to be ready
    if ! browserless::wait_for_ready "Browserless to start" "$BROWSERLESS_STARTUP_MAX_WAIT"; then
        resources::handle_error \
            "${MSG_STARTUP_TIMEOUT}" \
            "system" \
            "Check container logs for errors"
        return 1
    fi
    
    # Give Browserless time to initialize
    sleep $BROWSERLESS_INITIALIZATION_WAIT
    
    # Verify health
    if browserless::wait_for_healthy 30; then
        # Update Vrooli configuration
        browserless::update_config
        
        # Show success information
        browserless::show_installation_success
        
        # Clear rollback context on success
        ROLLBACK_ACTIONS=()
        OPERATION_ID=""
        
        return 0
    else
        log::warn "${MSG_STARTED_NOT_HEALTHY}"
        log::info "Check logs: docker logs $BROWSERLESS_CONTAINER_NAME"
        return 0
    fi
}

#######################################
# Uninstall Browserless completely
# Returns: 0 if successful, 1 if failed
#######################################
browserless::uninstall_service() {
    log::header "🗑️  Uninstalling Browserless"
    
    if ! flow::is_yes "$YES"; then
        log::warn "${MSG_UNINSTALL_WARNING}"
        read -p "Are you sure you want to continue? (y/N): " -r
        if [[ ! $REPLY =~ ^[Yy]$ ]]; then
            log::info "Uninstall cancelled"
            return 0
        fi
    fi
    
    # Backup data before removal
    browserless::backup_data "uninstall"
    
    # Ask about data directory removal
    local remove_data="no"
    if [[ -d "$BROWSERLESS_DATA_DIR" ]]; then
        read -p "Remove Browserless data directory? (y/N): " -r
        if [[ $REPLY =~ ^[Yy]$ ]]; then
            remove_data="yes"
        fi
    fi
    
    # Remove Docker container and network
    browserless::docker_remove "$remove_data"
    
    # Remove from Vrooli config
    resources::remove_config "agents" "browserless"
    
    log::success "${MSG_UNINSTALL_SUCCESS}"
    return 0
}

# Export functions for subshell availability
export -f browserless::create_directories
export -f browserless::update_config
export -f browserless::show_installation_success
export -f browserless::install_service
export -f browserless::uninstall_service