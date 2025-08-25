#!/usr/bin/env bash
set -euo pipefail

# Get the directory of this script
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
TWILIO_LIB_DIR="${APP_ROOT}/resources/twilio/lib"

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

