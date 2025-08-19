#!/bin/bash
# ERPNext Status Functions

# Source format utilities and config
ERPNEXT_STATUS_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck disable=SC1091
source "${ERPNEXT_STATUS_DIR}/../../../../lib/utils/format.sh"
# shellcheck disable=SC1091
source "${ERPNEXT_STATUS_DIR}/../../../lib/status-args.sh"
# shellcheck disable=SC1091
source "${ERPNEXT_STATUS_DIR}/../config/defaults.sh" 2>/dev/null || true
# shellcheck disable=SC1091
source "${ERPNEXT_STATUS_DIR}/common.sh" 2>/dev/null || true

#######################################
# Collect ERPNext status data in format-agnostic structure
# Args: [--fast] - Skip expensive operations for faster response
# Returns: Key-value pairs ready for formatting
#######################################
erpnext::status::collect_data() {
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
    
    local status_data=()
    
    # Basic status checks
    local installed="false"
    local running="false"
    local healthy="false"
    local health_message="ERPNext not installed"
    
    if erpnext::is_installed; then
        installed="true"
        if erpnext::is_running; then
            running="true"
            # Skip health check in fast mode
            if [[ "$fast_mode" == "false" ]]; then
                if erpnext::is_healthy; then
                    healthy="true"
                    health_message="ERPNext is running and healthy on port ${ERPNEXT_PORT}"
                else
                    health_message="ERPNext is running but not responding to health checks"
                fi
            else
                healthy="N/A"
                health_message="ERPNext is running on port ${ERPNEXT_PORT}"
            fi
        else
            health_message="ERPNext is installed but not running"
        fi
    fi
    
    # Basic resource information
    status_data+=("name" "erpnext")
    status_data+=("category" "automation")
    status_data+=("description" "Complete open-source ERP suite")
    status_data+=("installed" "$installed")
    status_data+=("running" "$running")
    status_data+=("healthy" "$healthy")
    status_data+=("health_message" "$health_message")
    status_data+=("version" "${ERPNEXT_VERSION:-unknown}")
    status_data+=("port" "${ERPNEXT_PORT:-8020}")
    status_data+=("site" "${ERPNEXT_SITE_NAME:-vrooli.local}")
    
    # Return the collected data
    printf '%s\n' "${status_data[@]}"
}

#######################################
# Display status in text format
#######################################
erpnext::status::display_text() {
    local -A data
    
    # Convert array to associative array
    for ((i=1; i<=$#; i+=2)); do
        local key="${!i}"
        local value_idx=$((i+1))
        local value="${!value_idx}"
        data["$key"]="$value"
    done
    
    # Header
    log::header "📊 ERPNext Status"
    echo
    
    # Basic status
    log::info "📊 Basic Status:"
    if [[ "${data[installed]:-false}" == "true" ]]; then
        log::success "   ✅ Installed: Yes"
    else
        log::error "   ❌ Installed: No"
        echo
        log::info "💡 Installation Required:"
        log::info "   To install ERPNext, run: ./manage.sh --action install"
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
    
    # Service information
    log::info "🌐 Service Information:"
    log::info "   📶 Port: ${data[port]:-unknown}"
    log::info "   🏢 Site: ${data[site]:-unknown}"
    log::info "   🔖 Version: ${data[version]:-unknown}"
    echo
    
    # Quick access info
    if [[ "${data[running]:-false}" == "true" ]]; then
        log::info "🎯 Quick Actions:"
        log::info "   🌐 Access ERPNext: http://localhost:${data[port]:-8020}"
        log::info "   📄 View logs: ./manage.sh --action logs"
        log::info "   🛑 Stop service: ./manage.sh --action stop"
    fi
}

# Status function
erpnext::status() {
    status::run_standard "erpnext" "erpnext::status::collect_data" "erpnext::status::display_text" "$@"
}