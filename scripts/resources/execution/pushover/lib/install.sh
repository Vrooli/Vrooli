#!/bin/bash
# Pushover installation functionality

# Get script directory
PUSHOVER_INSTALL_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Source dependencies
source "${PUSHOVER_INSTALL_DIR}/core.sh"

# Install Pushover
pushover::install() {
    local verbose="${1:-false}"
    
    log::header "Installing Pushover"
    
    # Initialize
    pushover::init "$verbose"
    
    # Check if already installed
    if pushover::is_installed; then
        log::success "Pushover dependencies already installed"
    else
        # Install Python requests library
        log::info "Installing Python dependencies..."
        python3 -m pip install --user requests >/dev/null 2>&1
        
        if pushover::is_installed; then
            log::success "Python dependencies installed successfully"
        else
            log::error "Failed to install Python dependencies"
            return 1
        fi
    fi
    
    # Create default configuration
    if [[ ! -f "$PUSHOVER_CONFIG_FILE" ]]; then
        log::info "Creating default configuration..."
        cat > "$PUSHOVER_CONFIG_FILE" <<EOF
{
    "enabled": true,
    "default_priority": 0,
    "default_sound": "pushover",
    "retry_interval": 60,
    "expire_time": 3600,
    "templates": {
        "success": {
            "sound": "cosmic",
            "priority": 0
        },
        "warning": {
            "sound": "falling",
            "priority": 1
        },
        "error": {
            "sound": "siren",
            "priority": 2
        }
    }
}
EOF
        log::success "Configuration created"
    fi
    
    # Create sample templates
    log::info "Creating notification templates..."
    
    # Success template
    cat > "${PUSHOVER_TEMPLATES_DIR}/success.json" <<EOF
{
    "title": "Task Completed Successfully",
    "message": "{{task_name}} has completed successfully.\n\nDuration: {{duration}}\nDetails: {{details}}",
    "priority": 0,
    "sound": "cosmic"
}
EOF
    
    # Error template
    cat > "${PUSHOVER_TEMPLATES_DIR}/error.json" <<EOF
{
    "title": "Task Failed",
    "message": "{{task_name}} has failed.\n\nError: {{error_message}}\nTime: {{timestamp}}",
    "priority": 2,
    "sound": "siren"
}
EOF
    
    # Workflow template
    cat > "${PUSHOVER_TEMPLATES_DIR}/workflow.json" <<EOF
{
    "title": "Workflow Update",
    "message": "Workflow: {{workflow_name}}\nStatus: {{status}}\nProgress: {{progress}}%",
    "priority": 0,
    "sound": "pushover"
}
EOF
    
    log::success "Templates created"
    
    # Register CLI with vrooli
    if command -v install-resource-cli >/dev/null 2>&1; then
        log::info "Registering Pushover CLI..."
        install-resource-cli pushover "${PUSHOVER_INSTALL_DIR}/../cli.sh" execution
        log::success "CLI registered"
    fi
    
    # Check for credentials
    if ! pushover::is_configured; then
        log::warning "Pushover installed but not configured"
        log::info ""
        log::info "To configure Pushover:"
        log::info "1. Create an account at https://pushover.net"
        log::info "2. Create an application to get an App Token"
        log::info "3. Note your User Key from the dashboard"
        log::info "4. Save credentials in one of these ways:"
        log::info "   a) Copy and edit the example credentials file:"
        log::info "      cp ${PUSHOVER_INSTALL_DIR}/../config/credentials.example.json ${PUSHOVER_CREDENTIALS_FILE}"
        log::info "      Then edit ${PUSHOVER_CREDENTIALS_FILE} with your actual credentials"
        log::info "   b) Store in Vault (recommended for production):"
        log::info "      resource-vault set pushover '{\"app_token\":\"...\",\"user_key\":\"...\"}'"
        log::info "   c) Set environment variables:"
        log::info "      export PUSHOVER_APP_TOKEN=..."
        log::info "      export PUSHOVER_USER_KEY=..."
    else
        log::success "Pushover is installed and configured"
        
        # Test connection
        if pushover::health_check; then
            log::success "Successfully connected to Pushover API"
        else
            log::error "Failed to connect to Pushover API"
        fi
    fi
    
    return 0
}

# Uninstall Pushover
pushover::uninstall() {
    local verbose="${1:-false}"
    
    log::header "Uninstalling Pushover"
    
    # We don't remove Python packages as they might be used by other resources
    # Just remove our data directory
    if [[ -d "$PUSHOVER_DATA_DIR" ]]; then
        log::info "Removing Pushover data directory..."
        rm -rf "$PUSHOVER_DATA_DIR"
        log::success "Data directory removed"
    fi
    
    log::success "Pushover uninstalled"
    return 0
}