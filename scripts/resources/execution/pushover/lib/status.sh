#!/bin/bash
# Pushover status functionality

# Get script directory
PUSHOVER_STATUS_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Source dependencies
source "${PUSHOVER_STATUS_DIR}/core.sh"
source "${PUSHOVER_STATUS_DIR}/../../../lib/status-args.sh"
source "${PUSHOVER_STATUS_DIR}/../../../../lib/utils/format.sh"

# Collect Pushover status data in format-agnostic structure
pushover::status::collect_data() {
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
    
    pushover::init false
    
    local installed="false"
    local running="false" 
    local healthy="false"
    local health_message="Pushover not configured"
    
    # Check if installed (Python dependencies)
    if pushover::is_installed; then
        installed="true"
        health_message="Dependencies installed, awaiting API credentials"
    fi
    
    # Check if configured and running
    if pushover::is_configured; then
        running="true"
        healthy="true"
        health_message="Pushover is configured and ready"
    else
        # Not configured but installed - this is a valid pending state
        running="false"
        healthy="pending"  # Use pending instead of false to indicate awaiting configuration
        health_message="Awaiting API credentials (check Vault or set PUSHOVER_APP_TOKEN and PUSHOVER_USER_KEY)"
    fi
    
    # Build status data array
    local status_data=(
        "name" "pushover"
        "category" "execution"
        "description" "Real-time notifications for Android, iPhone, iPad and Desktop"
        "installed" "$installed"
        "running" "$running"
        "healthy" "$healthy"
        "health_message" "$health_message"
        "api_url" "https://api.pushover.net/1/messages.json"
    )
    
    # Return the collected data
    printf '%s\n' "${status_data[@]}"
}

# Display status in text format
pushover::status::display_text() {
    local -A data
    
    # Convert array to associative array
    for ((i=1; i<=$#; i+=2)); do
        local key="${!i}"
        local value_idx=$((i+1))
        local value="${!value_idx}"
        data["$key"]="$value"
    done
    
    # Header
    log::header "📱 Pushover Status"
    echo
    
    # Basic status
    log::info "📊 Basic Status:"
    if [[ "${data[installed]:-false}" == "true" ]]; then
        log::success "   ✅ Installed: Yes"
        if [[ "${data[running]:-false}" == "true" ]]; then
            log::success "   ✅ Configured: Yes"
            log::success "   ✅ Running: Yes"
        else
            log::warning "   ⚠️ Configured: No"
            log::info "   ⚠️ Running: No (awaiting credentials)"
        fi
        log::info "   📋 Status: ${data[health_message]:-Unknown}"
    else
        log::error "   ❌ Installed: No"
        echo
        log::info "💡 Installation Required:"
        log::info "   To install Pushover, run: resource-pushover install"
        return
    fi
    echo
    
    log::info "🌐 Service Info:"
    log::info "   🔌 API: ${data[api_url]:-unknown}"
    echo
}

# New main status function using standard wrapper
pushover::status::new() {
    status::run_standard "pushover" "pushover::status::collect_data" "pushover::status::display_text" "$@"
}

# Main status function - delegate to new implementation
pushover::status() {
    pushover::status::new "$@"
}