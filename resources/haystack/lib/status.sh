#!/usr/bin/env bash
set -euo pipefail

# Define directory using cached APP_ROOT
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
HAYSTACK_LIB_DIR="${APP_ROOT}/resources/haystack/lib"

# Source dependencies
source "${HAYSTACK_LIB_DIR}/common.sh"
source "${HAYSTACK_LIB_DIR}/../../../../lib/utils/log.sh"
source "${HAYSTACK_LIB_DIR}/../../../../lib/utils/format.sh"
source "${HAYSTACK_LIB_DIR}/../../../lib/status-args.sh"

#######################################
# Collect Haystack status data in format-agnostic structure
# Args: [--fast] - Skip expensive operations for faster response
# Returns: Key-value pairs ready for formatting
#######################################
haystack::status::collect_data() {
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
    
    # Basic information
    status_data+=("name" "haystack")
    status_data+=("category" "execution")
    status_data+=("description" "Haystack AI framework for semantic search and question answering")
    
    # Status checks
    local installed="false"
    local running="false"
    local healthy="false"
    local status="stopped"
    local message="Haystack is not running"
    local health="unhealthy"
    local details=""
    local port="unknown"
    
    if haystack::is_installed; then
        installed="true"
        
        if haystack::is_running; then
            running="true"
            status="running"
            port=$(haystack::get_port)
            
            # Check health endpoint (skip in fast mode)
            if [[ "$fast_mode" == "false" ]] && timeout 2s curl -sf "http://localhost:${port}/health" >/dev/null 2>&1; then
                healthy="true"
                health="healthy"
                message="Haystack is running and healthy on port ${port}"
                
                # Get stats in non-fast mode
                local stats
                stats=$(timeout 2s curl -sf "http://localhost:${port}/stats" 2>/dev/null || echo "N/A")
                details="Stats: ${stats}"
            elif [[ "$fast_mode" == "true" ]]; then
                healthy="true"
                health="healthy"
                message="Haystack is running (fast mode)"
                details="N/A"
            else
                health="unhealthy"
                message="Haystack is running but not responding on port ${port}"
                details="Health check failed"
            fi
        else
            message="Haystack is installed but not running"
        fi
    else
        message="Haystack is not installed"
    fi
    
    # Build status data
    status_data+=("installed" "$installed")
    status_data+=("running" "$running")
    status_data+=("healthy" "$healthy")
    status_data+=("status" "$status")
    status_data+=("health_message" "$message")
    status_data+=("port" "$port")
    status_data+=("details" "$details")
    
    # Service endpoints
    if [[ "$running" == "true" ]]; then
        status_data+=("api_url" "http://localhost:$port")
        status_data+=("health_url" "http://localhost:$port/health")
        status_data+=("stats_url" "http://localhost:$port/stats")
    fi
    
    # Return the collected data
    printf '%s\n' "${status_data[@]}"
}

# Get Haystack status
haystack::status() {
    # Use the standard wrapper
    status::run_standard "haystack" "haystack::status::collect_data" "haystack::status::display_text" "$@"
}

#######################################
# Display status in text format
#######################################
haystack::status::display_text() {
    local -A data
    
    # Convert array to associative array
    for ((i=1; i<=$#; i+=2)); do
        local key="${!i}"
        local value_idx=$((i+1))
        local value="${!value_idx}"
        data["$key"]="$value"
    done
    
    # Get verbose mode from environment or default to false
    local verbose="${STATUS_VERBOSE:-false}"
    
    # Header
    log::header "🔍 Haystack Status"
    echo
    
    # Basic status
    log::info "📊 Basic Status:"
    if [[ "${data[installed]:-false}" == "true" ]]; then
        log::success "   ✅ Installed: Yes"
    else
        log::error "   ❌ Installed: No"
        echo
        log::info "💡 Installation Required:"
        log::info "   To install Haystack, run: resource-haystack install"
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
    log::info "   📶 Port: ${data[port]:-unknown}"
    echo
    
    # Service information
    if [[ "${data[running]:-false}" == "true" ]]; then
        log::info "🌐 Service Endpoints:"
        log::info "   🔌 API: http://localhost:${data[port]:-unknown}"
        log::info "   🏥 Health: http://localhost:${data[port]:-unknown}/health"
        log::info "   📊 Stats: http://localhost:${data[port]:-unknown}/stats"
        echo
    fi
    
    # Status message
    log::info "📋 Status Message:"
    log::info "   ${data[health_message]:-No status message}"
    
    # Verbose details if provided
    if [[ -n "${data[details]:-}" ]] && [[ "$verbose" == "true" ]]; then
        echo
        log::info "📈 Additional Details:"
        log::info "   ${data[details]}"
    fi
    echo
}