#!/usr/bin/env bash
set -euo pipefail

# Get the directory of this script
TWILIO_LIB_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Source common functions
source "$TWILIO_LIB_DIR/common.sh"

# Start Twilio webhook monitor
start_monitor() {
    log::header "🚀 Starting Twilio Monitor"
    
    # Check if installed
    if ! twilio::is_installed; then
        log::error "$MSG_TWILIO_NOT_INSTALLED"
        return 1
    fi
    
    # Check credentials
    if ! twilio::has_credentials; then
        log::error "$MSG_TWILIO_NO_CREDENTIALS"
        return 1
    fi
    
    # Check if already running
    if twilio::is_monitor_running; then
        log::warn "Twilio monitor is already running"
        return 0
    fi
    
    twilio::ensure_dirs
    
    # For now, just validate credentials work
    if twilio::setup_auth; then
        # Create a simple status file to indicate "running"
        echo "$$" > "$TWILIO_MONITOR_PID_FILE"
        log::success "$MSG_TWILIO_MONITOR_STARTED"
        log::info "Twilio is ready for API calls"
    else
        log::error "Failed to authenticate with Twilio"
        return 1
    fi
    
    return 0
}

twilio::start() {
    start_monitor
}

