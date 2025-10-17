#!/usr/bin/env bash
################################################################################
# Docker operations for Odoo resource
################################################################################

# Source common functions
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
source "${APP_ROOT}/resources/odoo/lib/common.sh"

# Start Odoo services
odoo::docker::start() {
    odoo_start "$@"
}

# Stop Odoo services
odoo::docker::stop() {
    odoo_stop "$@"
}

# Restart Odoo services - required by Universal Contract
odoo::docker::restart() {
    log::info "Restarting Odoo..."
    odoo_stop
    sleep 2
    odoo_start
}

# Show Odoo logs - required by Universal Contract
odoo::docker::logs() {
    local lines="${1:-50}"
    local follow="${2:-false}"
    
    log::info "Showing Odoo logs (last $lines lines)..."
    
    if [[ "$follow" == "true" || "$follow" == "-f" ]]; then
        if odoo_is_running; then
            docker logs -f --tail "$lines" "$ODOO_CONTAINER_NAME" 2>&1
        else
            log::warn "Container not running. Showing static logs..."
            odoo_logs "$lines"
        fi
    else
        odoo_logs "$lines"
    fi
}

# Export functions
export -f odoo::docker::start
export -f odoo::docker::stop
export -f odoo::docker::restart
export -f odoo::docker::logs