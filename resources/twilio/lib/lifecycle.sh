#!/usr/bin/env bash
################################################################################
# Twilio Lifecycle Management
# 
# Start/stop/restart implementations for v2.0 contract
################################################################################

set -euo pipefail

# Get the directory of this script
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
TWILIO_LIB_DIR="${APP_ROOT}/resources/twilio/lib"

# Source required libraries
source "$TWILIO_LIB_DIR/common.sh"
source "$TWILIO_LIB_DIR/core.sh"

################################################################################
# Lifecycle Functions
################################################################################

# Start Twilio (validate configuration)
twilio::lifecycle::start() {
    log::header "🚀 Starting Twilio Resource"
    
    # Check if installed
    if ! twilio::is_installed; then
        log::error "Twilio CLI not installed. Run: resource-twilio manage install"
        return 1
    fi
    
    # Initialize and load secrets
    twilio::core::init
    
    # Check credentials
    if ! twilio::core::has_valid_credentials; then
        log::error "Twilio credentials not configured"
        log::info "Configure with: resource-vault secrets init twilio"
        return 1
    fi
    
    # Test API connection
    log::info "Testing Twilio API connection..."
    if twilio::core::test_api_connection; then
        log::success "✅ Twilio API connection successful"
        
        # Create a marker file to indicate "running" state
        echo "$(date +%s)" > "$TWILIO_DATA_DIR/.running"
        
        # Log account info
        local account_sid="${TWILIO_ACCOUNT_SID:-unknown}"
        log::info "Account: ${account_sid:0:10}..."
        
        log::success "Twilio resource started successfully"
        return 0
    else
        log::error "Failed to connect to Twilio API"
        log::info "Check your credentials and network connection"
        return 1
    fi
}

# Stop Twilio (cleanup state)
twilio::lifecycle::stop() {
    log::header "🛑 Stopping Twilio Resource"
    
    # Clean up any state files
    if [[ -f "$TWILIO_DATA_DIR/.running" ]]; then
        rm -f "$TWILIO_DATA_DIR/.running"
    fi
    
    # Clean up any PID files from legacy monitor
    if [[ -f "$TWILIO_MONITOR_PID_FILE" ]]; then
        rm -f "$TWILIO_MONITOR_PID_FILE"
    fi
    
    log::success "Twilio resource stopped"
    return 0
}

# Restart Twilio
twilio::lifecycle::restart() {
    log::header "🔄 Restarting Twilio Resource"
    
    # Stop
    twilio::lifecycle::stop
    
    # Wait a moment
    sleep 1
    
    # Start
    twilio::lifecycle::start
}

# Check if Twilio is running
twilio::lifecycle::is_running() {
    # Check for running marker
    if [[ -f "$TWILIO_DATA_DIR/.running" ]]; then
        # Check if marker is recent (within last hour)
        local timestamp=$(cat "$TWILIO_DATA_DIR/.running" 2>/dev/null || echo "0")
        local now=$(date +%s)
        local age=$((now - timestamp))
        
        if [[ $age -lt 3600 ]]; then
            return 0
        fi
    fi
    
    return 1
}

################################################################################
# Docker compatibility shims (for CLI framework)
################################################################################

twilio::docker::start() {
    twilio::lifecycle::start "$@"
}

twilio::docker::stop() {
    twilio::lifecycle::stop "$@"
}

twilio::docker::restart() {
    twilio::lifecycle::restart "$@"
}

twilio::docker::logs() {
    if [[ -f "$TWILIO_LOG_FILE" ]]; then
        tail -f "$TWILIO_LOG_FILE"
    else
        log::info "No Twilio logs available (stateless service)"
    fi
}

################################################################################
# Export Functions
################################################################################

export -f twilio::lifecycle::start
export -f twilio::lifecycle::stop
export -f twilio::lifecycle::restart
export -f twilio::lifecycle::is_running
export -f twilio::docker::start
export -f twilio::docker::stop
export -f twilio::docker::restart
export -f twilio::docker::logs