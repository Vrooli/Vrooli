#!/bin/bash

# Status functions for Mail-in-a-Box resource

MAILINABOX_STATUS_LIB_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Source dependencies
source "$MAILINABOX_STATUS_LIB_DIR/core.sh"
source "$MAILINABOX_STATUS_LIB_DIR/../../../lib/status-args.sh"
source "$MAILINABOX_STATUS_LIB_DIR/../../../../lib/utils/format.sh"

# Collect Mail-in-a-Box status data in format-agnostic structure
mailinabox::status::collect_data() {
    local fast_mode="false"
    
    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --fast)
                fast_mode="true"
                shift
                ;;
            *)
                shift
                ;;
        esac
    done
    
    # Gather status information
    local installed=$(mailinabox_is_installed && echo "true" || echo "false")
    local running=$(mailinabox_is_running && echo "true" || echo "false")
    
    # Skip expensive operations in fast mode
    local health version details
    if [[ "$fast_mode" == "true" ]]; then
        if [[ "$running" == "true" ]]; then
            health="true"
            version="N/A"
            details="Mail-in-a-Box running (fast mode)"
        else
            health="false"
            version="N/A"
            details="Status check skipped (fast mode)"
        fi
    else
        local health_status=$(mailinabox_get_health)
        health=$([[ "$health_status" == "healthy" ]] && echo "true" || echo "false")
        version=$(mailinabox_get_version)
        details=$(mailinabox_get_status_details)
    fi
    
    # Build status data array
    local status_data=(
        "name" "${MAILINABOX_NAME:-mail-in-a-box}"
        "category" "${MAILINABOX_CATEGORY:-execution}"
        "description" "${MAILINABOX_DESCRIPTION:-Complete email server solution}"
        "installed" "$installed"
        "running" "$running"
        "healthy" "$health"
        "health_message" "$details"
        "version" "$version"
        "port" "${MAILINABOX_PORT_ADMIN:-443}"
        "admin_url" "https://${MAILINABOX_BIND_ADDRESS:-localhost}:${MAILINABOX_PORT_ADMIN:-443}/admin"
        "webmail_url" "https://${MAILINABOX_BIND_ADDRESS:-localhost}/mail"
    )
    
    # Return the collected data
    printf '%s\n' "${status_data[@]}"
}

# Display status in text format
mailinabox::status::display_text() {
    local -A data
    
    # Convert array to associative array
    for ((i=1; i<=$#; i+=2)); do
        local key="${!i}"
        local value_idx=$((i+1))
        local value="${!value_idx}"
        data["$key"]="$value"
    done
    
    # Header
    log::header "📧 Mail-in-a-Box Status"
    echo
    
    # Basic status
    log::info "📊 Basic Status:"
    if [[ "${data[installed]:-false}" == "true" ]]; then
        log::success "   ✅ Installed: Yes"
    else
        log::error "   ❌ Installed: No"
        echo
        log::info "💡 Installation Required:"
        log::info "   To install Mail-in-a-Box, run: resource-mail-in-a-box install"
        return
    fi
    
    if [[ "${data[running]:-false}" == "true" ]]; then
        log::success "   ✅ Running: Yes"
    else
        log::warn "   ⚠️  Running: No"
    fi
    
    if [[ "${data[healthy]:-false}" == "true" ]]; then
        log::success "   ✅ Health: Healthy"
    else
        log::warn "   ⚠️  Health: ${data[health_message]:-Unknown}"
    fi
    echo
    
    # Configuration
    log::info "⚙️  Configuration:"
    log::info "   📦 Version: ${data[version]:-unknown}"
    log::info "   📶 Port: ${data[port]:-unknown}"
    echo
    
    if [[ "${data[running]:-false}" == "true" ]]; then
        log::info "🌐 Service Endpoints:"
        log::info "   🔧 Admin: ${data[admin_url]:-unknown}"
        log::info "   📧 Webmail: ${data[webmail_url]:-unknown}"
        echo
    fi
    
    log::info "📋 Status Message:"
    log::info "   ${data[health_message]:-No status message}"
    echo
}

# Main status function using standard wrapper
mailinabox::status() {
    status::run_standard "mail-in-a-box" "mailinabox::status::collect_data" "mailinabox::status::display_text" "$@"
}

# Legacy function for backward compatibility
mailinabox_status() {
    mailinabox::status "$@"
}