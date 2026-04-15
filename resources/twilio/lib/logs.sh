#!/usr/bin/env bash
set -euo pipefail

# Get the directory of this script
SCRIPT_DIR="$(builtin cd "${BASH_SOURCE[0]%/*}" && builtin pwd)"
RESOURCE_DIR="$(builtin cd "${SCRIPT_DIR}/.." && builtin pwd)"
REPO_ROOT="$(builtin cd "${RESOURCE_DIR}/../.." && builtin pwd)"
TWILIO_LIB_DIR="${RESOURCE_DIR}/lib"

# Source common functions
source "$TWILIO_LIB_DIR/common.sh"

# Show logs
show_logs() {
    local lines="${1:-50}"
    
    log::header "📜 Twilio Logs"
    
    if [[ -f "$TWILIO_LOG_FILE" ]]; then
        tail -n "$lines" "$TWILIO_LOG_FILE"
    else
        log::info "No logs available"
    fi
}

twilio::logs() {
    show_logs "$@"
}

