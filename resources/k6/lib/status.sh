#!/usr/bin/env bash
# K6 Status Functions

# Source format utility for JSON support
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
K6_STATUS_SCRIPT_DIR="${APP_ROOT}/resources/k6/lib"
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/format.sh"

# Source status-args library
source "${APP_ROOT}/scripts/resources/lib/status-args.sh"

# Collect K6 status data in format-agnostic structure
k6::status::collect_data() {
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
    
    # Initialize if needed
    k6::core::init 2>/dev/null || true
    
    # Check installation
    local installed="false"
    local running="false"
    local health="false"
    local message="K6 not installed"
    local version="unknown"
    
    # Check native installation
    if k6::core::is_installed_native; then
        installed="true"
        version=$(k6 version 2>/dev/null | grep -oP 'k6 v\K[0-9.]+' || echo "unknown")
        running="true"  # CLI tool is always "running" when installed
        health="true"
        message="K6 CLI installed and ready"
    fi
    
    # Check Docker installation
    if k6::core::is_running; then
        installed="true"
        running="true"
        health="true"
        message="K6 container running"
        if [[ "$version" == "unknown" ]]; then
            version=$(docker exec "$K6_CONTAINER_NAME" k6 version 2>/dev/null | grep -oP 'k6 v\K[0-9.]+' || echo "unknown")
        fi
    fi
    
    # Count test scripts (skip in fast mode)
    local test_count=0
    if [[ -d "$K6_SCRIPTS_DIR" ]] && [[ "$fast_mode" == "false" ]]; then
        test_count=$(find "$K6_SCRIPTS_DIR" -name "*.js" 2>/dev/null | wc -l)
    elif [[ -d "$K6_SCRIPTS_DIR" ]] && [[ "$fast_mode" == "true" ]]; then
        test_count="N/A"
    fi
    
    # Count results (skip in fast mode)
    local results_count=0
    if [[ -d "$K6_RESULTS_DIR" ]] && [[ "$fast_mode" == "false" ]]; then
        results_count=$(find "$K6_RESULTS_DIR" -name "*.json" -o -name "*.csv" 2>/dev/null | wc -l)
    elif [[ -d "$K6_RESULTS_DIR" ]] && [[ "$fast_mode" == "true" ]]; then
        results_count="N/A"
    fi
    
    # Build status data array
    local status_data=(
        "name" "k6"
        "category" "${K6_CATEGORY:-execution}"
        "description" "${K6_DESCRIPTION:-Modern load testing tool}"
        "installed" "$installed"
        "running" "$running"
        "healthy" "$health"
        "health_message" "$message"
        "version" "$version"
        "port" "${K6_PORT:-unknown}"
        "test_scripts" "$test_count"
        "results" "$results_count"
        "scripts_dir" "${K6_SCRIPTS_DIR:-unknown}"
        "results_dir" "${K6_RESULTS_DIR:-unknown}"
    )
    
    # Return the collected data
    printf '%s\n' "${status_data[@]}"
}

# Display status in text format
k6::status::display_text() {
    local -A data
    
    # Convert array to associative array
    for ((i=1; i<=$#; i+=2)); do
        local key="${!i}"
        local value_idx=$((i+1))
        local value="${!value_idx}"
        data["$key"]="$value"
    done
    
    # Header
    log::header "🚀 K6 Status"
    echo
    
    # Basic status
    log::info "📊 Basic Status:"
    if [[ "${data[installed]:-false}" == "true" ]]; then
        log::success "   ✅ Installed: Yes"
    else
        log::error "   ❌ Installed: No"
        echo
        log::info "💡 Installation Required:"
        log::info "   Install k6: https://k6.io/docs/getting-started/installation/"
        return
    fi
    
    if [[ "${data[running]:-false}" == "true" ]]; then
        log::success "   ✅ Running: Yes"
    else
        log::warn "   ⚠️  Running: No"
    fi
    
    if [[ "${data[healthy]:-false}" == "true" ]]; then
        log::success "   ✅ Health: ${data[health_message]:-Healthy}"
    else
        log::warn "   ⚠️  Health: ${data[health_message]:-Unknown}"
    fi
    echo
    
    # Configuration
    log::info "⚙️  Configuration:"
    log::info "   📦 Version: ${data[version]:-unknown}"
    log::info "   📶 Port: ${data[port]:-unknown}"
    log::info "   📝 Test Scripts: ${data[test_scripts]:-0} in ${data[scripts_dir]:-unknown}"
    log::info "   📊 Results: ${data[results]:-0} in ${data[results_dir]:-unknown}"
    echo
}

# Main status function using standard wrapper
k6::status() {
    status::run_standard "k6" "k6::status::collect_data" "k6::status::display_text" "$@"
}

# Legacy function for backward compatibility
k6::status::check() {
    k6::status "$@"
}