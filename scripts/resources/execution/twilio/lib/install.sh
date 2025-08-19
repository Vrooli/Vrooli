#!/usr/bin/env bash
set -euo pipefail

# Get the directory of this script
TWILIO_LIB_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

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
main() {
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

main "$@"