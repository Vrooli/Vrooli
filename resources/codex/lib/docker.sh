#!/usr/bin/env bash
# Codex Docker Functions (simulated for API-based service)

# Set script directory for sourcing
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
CODEX_DOCKER_DIR="${APP_ROOT}/resources/codex/lib"

# Source required utilities
# shellcheck disable=SC1091
source "${CODEX_DOCKER_DIR}/common.sh"

#######################################
# Start Codex service (mark as running and verify API)
# Returns:
#   0 on success, 1 on failure
#######################################
codex::docker::start() {
    log::info "Starting Codex service..."
    
    if ! codex::is_configured; then
        log::error "Codex not configured. Please set OPENAI_API_KEY or configure Vault"
        return 1
    fi
    
    codex::save_status "running"
    log::success "Codex service marked as running"
    
    # Verify API is accessible
    if codex::is_available; then
        log::success "Codex API is accessible and responding"
    else
        log::warn "Codex API not responding. Check your API key and network connection"
    fi
    
    return 0
}

#######################################
# Stop Codex service (mark as stopped)
# Returns:
#   0 on success, 1 on failure
#######################################
codex::docker::stop() {
    log::info "Stopping Codex service..."
    codex::save_status "stopped"
    log::success "Codex service marked as stopped"
    return 0
}

#######################################
# Restart Codex service
# Returns:
#   0 on success, 1 on failure
#######################################
codex::docker::restart() {
    log::info "Restarting Codex service..."
    codex::docker::stop
    sleep 1
    codex::docker::start
    return $?
}

#######################################
# Show Codex logs (simulated - shows recent API calls/status)
# Arguments:
#   --follow|-f: Follow logs (simulated)
#   --tail N: Show last N entries
# Returns:
#   0 on success
#######################################
codex::docker::logs() {
    local follow="false"
    local tail_lines="50"
    
    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --follow|-f)
                follow="true"
                shift
                ;;
            --tail)
                tail_lines="${2:-50}"
                shift 2
                ;;
            *)
                shift
                ;;
        esac
    done
    
    log::header "Codex Service Logs"
    
    # Show current status
    echo "=== Current Status ==="
    codex::status --format text
    echo ""
    
    # Show recent API activity (if available)
    echo "=== Recent Activity ==="
    if [[ -d "${CODEX_OUTPUT_DIR}" ]]; then
        local recent_files
        recent_files=$(find "${CODEX_OUTPUT_DIR}" -type f -name "*.py" 2>/dev/null | head -"${tail_lines}")
        if [[ -n "${recent_files}" ]]; then
            echo "Recent Codex outputs:"
            echo "${recent_files}" | while read -r file; do
                local timestamp
                timestamp=$(stat -c %y "${file}" 2>/dev/null | cut -d. -f1)
                local basename
                basename=$(basename "${file}")
                echo "  [${timestamp}] Generated: ${basename}"
            done
        else
            echo "No recent Codex outputs found"
        fi
    else
        echo "No output directory found"
    fi
    
    echo ""
    echo "=== Configuration ==="
    echo "API Endpoint: ${CODEX_API_ENDPOINT}"
    echo "Default Model: ${CODEX_DEFAULT_MODEL}"
    echo "Scripts Directory: ${CODEX_SCRIPTS_DIR}"
    echo "Output Directory: ${CODEX_OUTPUT_DIR}"
    
    if [[ "$follow" == "true" ]]; then
        log::info "Log following not supported for API-based service"
        log::info "Use 'resource-codex status' to check current state"
    fi
    
    return 0
}

# Main entry point
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    case "$1" in
        start)
            codex::docker::start
            ;;
        stop)
            codex::docker::stop
            ;;
        restart)
            codex::docker::restart
            ;;
        logs)
            shift
            codex::docker::logs "$@"
            ;;
    esac
fi