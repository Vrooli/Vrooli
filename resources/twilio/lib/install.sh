#!/usr/bin/env bash
set -euo pipefail

# Get the directory of this script
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
TWILIO_LIB_DIR="${APP_ROOT}/resources/twilio/lib"

# Source common functions
source "$TWILIO_LIB_DIR/common.sh"

# Install Twilio CLI
install_twilio() {
    log::header "📦 Installing Twilio CLI"
    
    # Check if already installed
    if twilio::is_installed && [[ "${FORCE:-false}" != "true" ]]; then
        local version
        version=$(twilio::get_version)
        log::warn "$MSG_TWILIO_ALREADY_INSTALLED (version: $version)"
        log::info "Use --force to reinstall"
        return 0
    fi
    
    log::info "$MSG_TWILIO_INSTALLING"
    
    # Install via npm (local installation in data directory)
    if command -v npm >/dev/null 2>&1; then
        twilio::ensure_dirs
        cd "$TWILIO_DATA_DIR"
        
        # Initialize package.json if it doesn't exist
        if [[ ! -f package.json ]]; then
            npm init -y >/dev/null 2>&1
        fi
        
        if ! timeout 120 npm install twilio-cli 2>&1; then
            log::error "$MSG_TWILIO_INSTALL_FAILED"
            return 1
        fi
        
        # Create symlink for easy access
        if [[ -f "$TWILIO_DATA_DIR/node_modules/.bin/twilio" ]]; then
            ln -sf "$TWILIO_DATA_DIR/node_modules/.bin/twilio" "$TWILIO_CONFIG_DIR/twilio-cli"
        fi
    else
        log::error "npm not found. Please install Node.js first"
        return 1
    fi
    
    log::success "$MSG_TWILIO_INSTALLED"
    
    # Set up initial configuration
    setup_config
    
    return 0
}

# Set up initial configuration
setup_config() {
    twilio::ensure_dirs
    
    # Create sample credentials file if it doesn't exist
    if [[ ! -f "$TWILIO_CREDENTIALS_FILE" ]]; then
        cat > "$TWILIO_CREDENTIALS_FILE" << 'EOF'
{
  "account_sid": "",
  "auth_token": "",
  "default_from_number": "",
  "webhook_url": "",
  "status_callback_url": ""
}
EOF
        chmod 600 "$TWILIO_CREDENTIALS_FILE"
        log::info "Created credentials template at $TWILIO_CREDENTIALS_FILE"
        log::warn "Please add your Twilio credentials to this file"
    fi
    
    # Create sample phone numbers file
    if [[ ! -f "$TWILIO_PHONE_NUMBERS_FILE" ]]; then
        cat > "$TWILIO_PHONE_NUMBERS_FILE" << 'EOF'
{
  "numbers": [],
  "verified_callers": [],
  "messaging_services": []
}
EOF
        log::info "Created phone numbers template at $TWILIO_PHONE_NUMBERS_FILE"
    fi
    
    return 0
}

# Parse command line arguments
twilio::install::execute() {
    local force=false
    
    while [[ $# -gt 0 ]]; do
        case $1 in
            --force|-f)
                force=true
                shift
                ;;
            *)
                shift
                ;;
        esac
    done
    
    FORCE=$force install_twilio
}

# Uninstall Twilio CLI and clean up data
twilio::install::uninstall() {
    log::header "🗑️ Uninstalling Twilio CLI"
    
    # Stop any running monitor first
    if twilio::is_monitor_running; then
        log::info "Stopping Twilio monitor..."
        twilio::stop
    fi
    
    # Remove global CLI if installed via npm
    if command -v twilio >/dev/null 2>&1 && command -v npm >/dev/null 2>&1; then
        log::info "Removing global Twilio CLI..."
        npm uninstall -g twilio-cli 2>/dev/null || true
    fi
    
    # Remove local installation
    if [[ -d "$TWILIO_DATA_DIR/node_modules" ]]; then
        log::info "Removing local Twilio CLI installation..."
        rm -rf "$TWILIO_DATA_DIR/node_modules"
        rm -f "$TWILIO_DATA_DIR/package.json"
        rm -f "$TWILIO_DATA_DIR/package-lock.json"
    fi
    
    # Remove symlink
    if [[ -L "$TWILIO_CONFIG_DIR/twilio-cli" ]]; then
        rm -f "$TWILIO_CONFIG_DIR/twilio-cli"
    fi
    
    # Optionally remove config files (ask user)
    if [[ -d "$TWILIO_CONFIG_DIR" ]] && [[ "${FORCE:-false}" != "true" ]]; then
        read -p "Remove Twilio configuration files? (y/N) " -n 1 -r
        echo
        if [[ $REPLY =~ ^[Yy]$ ]]; then
            rm -rf "$TWILIO_CONFIG_DIR"
            log::info "Removed Twilio configuration directory"
        fi
    elif [[ "${FORCE:-false}" == "true" ]]; then
        rm -rf "$TWILIO_CONFIG_DIR"
        log::info "Removed Twilio configuration directory"
    fi
    
    # Remove data directory
    if [[ -d "$TWILIO_DATA_DIR" ]]; then
        rm -rf "$TWILIO_DATA_DIR"
        log::info "Removed Twilio data directory"
    fi
    
    log::success "Twilio CLI uninstalled successfully"
    return 0
}

