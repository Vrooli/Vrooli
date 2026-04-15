#!/usr/bin/env bash
set -euo pipefail

# Get the directory of this script
SCRIPT_DIR="$(builtin cd "${BASH_SOURCE[0]%/*}" && builtin pwd)"
RESOURCE_DIR="$(builtin cd "${SCRIPT_DIR}/.." && builtin pwd)"
REPO_ROOT="$(builtin cd "${RESOURCE_DIR}/../.." && builtin pwd)"
TWILIO_LIB_DIR="${RESOURCE_DIR}/lib"

# Source common functions
source "$TWILIO_LIB_DIR/common.sh"

# Stop Twilio monitor
stop_monitor() {
    log::header "🛑 Stopping Twilio Monitor"
    
    if twilio::is_monitor_running; then
        rm -f "$TWILIO_MONITOR_PID_FILE"
        log::success "$MSG_TWILIO_MONITOR_STOPPED"
    else
        log::info "Twilio monitor is not running"
    fi
    
    return 0
}

twilio::stop() {
    stop_monitor
}

