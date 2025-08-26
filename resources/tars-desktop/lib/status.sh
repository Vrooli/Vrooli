#!/bin/bash
# TARS-desktop status functionality

# Get script directory
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
TARS_DESKTOP_STATUS_DIR="${APP_ROOT}/resources/tars-desktop/lib"

# Source dependencies
source "${TARS_DESKTOP_STATUS_DIR}/core.sh"
source "${APP_ROOT}/scripts/resources/lib/status-args.sh"
source "${APP_ROOT}/scripts/lib/utils/format.sh"

# Collect TARS-desktop status data in format-agnostic structure
tars_desktop::status::collect_data() {
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
    
    local installed="false"
    local running="false" 
    local healthy="false"
    local health_message="TARS-desktop not installed"
    
    if tars_desktop::is_installed; then
        installed="true"
        if tars_desktop::is_running; then
            running="true"
            healthy="true"
            health_message="TARS-desktop is running"
        else
            health_message="TARS-desktop installed but not running"
        fi
    fi
    
    # Build status data array
    local status_data=(
        "name" "tars-desktop"
        "category" "execution"
        "description" "Virtual desktop environment for AI agents"
        "installed" "$installed"
        "running" "$running"
        "healthy" "$healthy"
        "health_message" "$health_message"
        "port" "${TARS_DESKTOP_PORT:-6080}"
        "vnc_port" "${TARS_DESKTOP_VNC_PORT:-5901}"
    )
    
    # Return the collected data
    printf '%s\n' "${status_data[@]}"
}

# Display status in text format
tars_desktop::status::display_text() {
    local -A data
    
    # Convert array to associative array
    for ((i=1; i<=$#; i+=2)); do
        local key="${!i}"
        local value_idx=$((i+1))
        local value="${!value_idx}"
        data["$key"]="$value"
    done
    
    # Header
    log::header "🖥️ TARS-desktop Status"
    echo
    
    # Basic status
    log::info "📊 Basic Status:"
    if [[ "${data[installed]:-false}" == "true" ]]; then
        log::success "   ✅ Installed: Yes"
    else
        log::error "   ❌ Installed: No"
        echo
        log::info "💡 Installation Required:"
        log::info "   To install TARS-desktop, run: resource-tars-desktop install"
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
    log::info "   📶 Web Port: ${data[port]:-unknown}"
    log::info "   📺 VNC Port: ${data[vnc_port]:-unknown}"
    echo
    
    if [[ "${data[running]:-false}" == "true" ]]; then
        log::info "🌐 Service Endpoints:"
        log::info "   🖥️  Web Desktop: http://localhost:${data[port]:-unknown}"
        log::info "   📺 VNC: vnc://localhost:${data[vnc_port]:-unknown}"
        echo
    fi
}

# New main status function using standard wrapper
tars_desktop::status::new() {
    status::run_standard "tars-desktop" "tars_desktop::status::collect_data" "tars_desktop::status::display_text" "$@"
}

# Main status function
tars_desktop::status() {
    local verbose="false"
    local format="text"
    local fast="false"
    
    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --verbose|-v)
                verbose="true"
                shift
                ;;
            --format)
                format="${2:-text}"
                shift 2
                ;;
            --fast)
                fast="true"
                shift
                ;;
            *)
                shift
                ;;
        esac
    done
    
    # Initialize
    tars_desktop::init >/dev/null 2>&1
    
    local status="unknown"
    local message=""
    local health="unhealthy"
    
    # Check if installed
    if ! tars_desktop::is_installed; then
        status="not_installed"
        message="TARS-desktop is not installed"
    elif ! tars_desktop::is_running; then
        status="stopped"
        message="TARS-desktop is installed but not running"
    elif [[ "$fast" == "true" ]]; then
        # Skip health check in fast mode
        status="running"
        health="healthy"
        message="TARS-desktop is running (fast mode)"
    elif tars_desktop::health_check; then
        status="running"
        health="healthy"
        message="TARS-desktop is running and healthy"
    else
        status="running"
        health="unhealthy"
        message="TARS-desktop is running but not responding"
    fi
    
    # Build output data
    local running="false"
    if [[ "$status" == "running" ]]; then
        running="true"
    fi
    
    local -a output_data=(
        "status" "$status"
        "running" "$running"
        "health" "$health"
        "message" "$message"
        "api_base" "$TARS_DESKTOP_API_BASE"
        "port" "$TARS_DESKTOP_PORT"
    )
    
    # Add verbose details if requested (skip expensive ones in fast mode)
    if [[ "$verbose" == "true" && "$status" == "running" ]]; then
        if [[ "$fast" == "false" ]]; then
            # Get capabilities
            local capabilities
            capabilities=$(tars_desktop::get_capabilities 2>/dev/null)
            if [[ -n "$capabilities" ]]; then
                local screen_width screen_height
                screen_width=$(echo "$capabilities" | jq -r '.screen.width' 2>/dev/null || echo "unknown")
                screen_height=$(echo "$capabilities" | jq -r '.screen.height' 2>/dev/null || echo "unknown")
                output_data+=(
                    "screen_width" "$screen_width"
                    "screen_height" "$screen_height"
                )
            fi
            
            # Check process info
            local pid
            pid=$(pgrep -f "tars_desktop.*server" | head -1)
            if [[ -n "$pid" ]]; then
                output_data+=("pid" "$pid")
            fi
        else
            # Fast mode - skip expensive operations
            output_data+=(
                "screen_width" "N/A"
                "screen_height" "N/A"
                "pid" "N/A"
            )
        fi
        
        # Installation info
        output_data+=(
            "install_dir" "$TARS_DESKTOP_INSTALL_DIR"
            "venv_dir" "$TARS_DESKTOP_VENV_DIR"
            "model" "$TARS_DESKTOP_MODEL"
            "provider" "$TARS_DESKTOP_MODEL_PROVIDER"
        )
    fi
    
    # Format output using standard formatter
    format::key_value "$format" "${output_data[@]}"
}

# Export function
export -f tars_desktop::status
# Wrapper function for compatibility
tars_desktop_status() {
    tars_desktop::status "$@"
}
export -f tars_desktop_status
