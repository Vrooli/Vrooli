#!/usr/bin/env bash
set -euo pipefail

# Get the directory of this script
HAYSTACK_LIB_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Source dependencies
source "${HAYSTACK_LIB_DIR}/common.sh"
source "${HAYSTACK_LIB_DIR}/../../../../lib/utils/log.sh"
source "${HAYSTACK_LIB_DIR}/../../../../lib/utils/format.sh"

# Get Haystack status
haystack::status() {
    local format="text"
    local verbose="false"
    
    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --format)
                format="${2:-text}"
                shift 2
                ;;
            --verbose)
                verbose="true"
                shift
                ;;
            *)
                shift
                ;;
        esac
    done
    
    local status="stopped"
    local message="Haystack is not running"
    local health="unhealthy"
    local details=""
    
    if haystack::is_running; then
        status="running"
        local port
        port=$(haystack::get_port)
        
        # Check health endpoint
        if curl -sf "http://localhost:${port}/health" >/dev/null 2>&1; then
            health="healthy"
            message="Haystack is running and healthy on port ${port}"
            
            # Get document count if verbose
            if [[ "${verbose}" == "true" ]]; then
                local stats
                stats=$(curl -sf "http://localhost:${port}/stats" 2>/dev/null || echo '{"error": "Failed to get stats"}')
                details="Stats: ${stats}"
            fi
        else
            health="unhealthy"
            message="Haystack is running but not responding on port ${port}"
        fi
    elif haystack::is_installed; then
        message="Haystack is installed but not running"
    else
        message="Haystack is not installed"
    fi
    
    # Convert status to boolean for consistency
    local running_bool="false"
    [[ "${status}" == "running" ]] && running_bool="true"
    
    # Format output using the shared format utility
    if [[ "${format}" == "json" ]]; then
        format::output "json" "kv" \
            "name" "haystack" \
            "status" "${status}" \
            "running" "${running_bool}" \
            "health" "${health}" \
            "message" "${message}" \
            "details" "${details}"
    else
        # Text format with standardized structure
        haystack::status::display_text \
            "status" "${status}" \
            "running" "${running_bool}" \
            "health" "${health}" \
            "message" "${message}" \
            "details" "${details}" \
            "verbose" "${verbose}"
    fi
}

#######################################
# Display status in text format
#######################################
haystack::status::display_text() {
    local -A data
    
    # Parse key-value pairs
    while [[ $# -gt 0 ]]; do
        data["$1"]="$2"
        shift 2 || break
    done
    
    # Header
    log::header "📊 HAYSTACK Status"
    log::info "📝 Description: End-to-end framework for question answering and search"
    log::info "📂 Category: execution"
    echo
    
    # Basic status
    log::info "📊 Basic Status:"
    
    # Installation status
    if haystack::is_installed; then
        log::success "   ✅ Installed: Yes"
    else
        log::error "   ❌ Installed: No"
        echo
        log::info "💡 To install Haystack, run: resource-haystack install"
        return
    fi
    
    # Running status
    if [[ "${data[running]:-false}" == "true" ]]; then
        log::success "   ✅ Running: Yes"
    else
        log::warn "   ⚠️  Running: No"
    fi
    
    # Health status
    if [[ "${data[health]:-unknown}" == "healthy" ]]; then
        log::success "   ✅ Health: Healthy"
    else
        log::warn "   ⚠️  Health: ${data[health]:-Unknown}"
    fi
    echo
    
    # Service information
    if [[ "${data[status]:-stopped}" == "running" ]]; then
        local port
        port=$(haystack::get_port)
        
        log::info "🌐 Service Endpoints:"
        log::info "   🔌 API: http://localhost:${port}"
        log::info "   🏥 Health: http://localhost:${port}/health"
        log::info "   📊 Stats: http://localhost:${port}/stats"
        echo
    fi
    
    # Status message
    log::info "📋 Status Message:"
    log::info "   ${data[message]:-No status message}"
    
    # Verbose details if provided
    if [[ -n "${data[details]:-}" ]] && [[ "${data[verbose]:-false}" == "true" ]]; then
        echo
        log::info "📈 Additional Details:"
        log::info "   ${data[details]}"
    fi
}