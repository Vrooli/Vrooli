#!/usr/bin/env bash
set -euo pipefail

# Define directory using cached APP_ROOT
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
PANDAS_AI_LIB_DIR="${APP_ROOT}/resources/pandas-ai/lib"

# Source dependencies
source "${PANDAS_AI_LIB_DIR}/common.sh"
source "${APP_ROOT}/scripts/lib/utils/log.sh"
source "${APP_ROOT}/scripts/lib/utils/format.sh"
source "${APP_ROOT}/scripts/resources/lib/status-args.sh"

# Check if Pandas AI is installed
pandas_ai::is_installed() {
    # Check if we have either venv or system Python with pandas
    if [[ -f "${PANDAS_AI_VENV_DIR}/bin/python" ]] && [[ -f "${PANDAS_AI_SCRIPTS_DIR}/server.py" ]]; then
        return 0
    fi
    
    # Check for system Python with pandas installed
    if [[ -f "${PANDAS_AI_SCRIPTS_DIR}/server.py" ]] && python3 -c "import pandas" 2>/dev/null; then
        return 0
    fi
    
    return 1
}

# Check if Pandas AI is running
pandas_ai::is_running() {
    if [[ -f "${PANDAS_AI_PID_FILE}" ]]; then
        local pid
        pid=$(cat "${PANDAS_AI_PID_FILE}")
        if kill -0 "${pid}" 2>/dev/null; then
            # Process exists, but also verify it's actually serving
            local port
            port=$(pandas_ai::get_port)
            if timeout 1 bash -c "echo > /dev/tcp/localhost/${port}" 2>/dev/null; then
                return 0
            fi
        else
            # PID file exists but process is dead - clean up stale PID file
            rm -f "${PANDAS_AI_PID_FILE}"
        fi
    fi
    
    # Also check if service is listening on port (in case PID file is missing)
    local port
    port=$(pandas_ai::get_port)
    if timeout 1 bash -c "echo > /dev/tcp/localhost/${port}" 2>/dev/null; then
        return 0
    fi
    
    return 1
}

# Get Pandas AI status
pandas_ai::status() {
    local format_type="text"
    local fast_mode="false"
    
    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --format)
                format_type="${2:-text}"
                shift 2
                ;;
            --json|json)
                format_type="json"
                shift
                ;;
            --fast)
                fast_mode="true"
                shift
                ;;
            *)
                shift
                ;;
        esac
    done
    
    local port
    port=$(pandas_ai::get_port)
    
    # Prepare status data
    local installed="false"
    local running="false"
    local health="unhealthy"
    local version="unknown"
    local details=""
    
    if pandas_ai::is_installed; then
        installed="true"
        # Skip version check in fast mode
        if [[ "$fast_mode" == "false" ]]; then
            # Try venv pip first, fallback to system pip
            if [[ -f "${PANDAS_AI_VENV_DIR}/bin/pip" ]]; then
                version=$("${PANDAS_AI_VENV_DIR}/bin/pip" show pandasai 2>/dev/null | grep Version | cut -d' ' -f2 || echo "unknown")
            else
                version=$(python3 -m pip show pandasai 2>/dev/null | grep Version | cut -d' ' -f2 || echo "1.0.0")
            fi
        else
            version="N/A"
        fi
    fi
    
    if pandas_ai::is_running; then
        running="true"
        # In fast mode, assume healthy if running (already verified port is responsive)
        if [[ "$fast_mode" == "true" ]]; then
            health="healthy"
        else
            # Check health endpoint with timeout for performance
            if timeout 1s curl -sf "http://localhost:${port}/health" >/dev/null 2>&1; then
                health="healthy"
            fi
        fi
    fi
    
    # Build details
    if [[ "${installed}" == "true" ]]; then
        details="Version: ${version}, Port: ${port}"
        if [[ "${running}" == "true" ]]; then
            details="${details}, URL: http://localhost:${port}"
        fi
    fi
    
    # Build key-value pairs for format.sh
    local status_data=(
        "name" "${PANDAS_AI_NAME}"
        "description" "${PANDAS_AI_DESC}"
        "category" "${PANDAS_AI_CATEGORY}"
        "installed" "${installed}"
        "running" "${running}"
        "health" "${health}"
        "version" "${version}"
        "port" "${port}"
    )
    
    if [[ "${running}" == "true" ]]; then
        status_data+=("url" "http://localhost:${port}")
    fi
    
    # Use format::output to handle both JSON and visual formats
    if [[ "${format_type}" == "json" ]]; then
        format::output "json" "kv" "${status_data[@]}"
    else
        log::header "Pandas AI Status"
        if [[ "${installed}" == "true" ]]; then
            log::success "✅ Installed: ${installed}"
        else
            log::error "❌ Installed: ${installed}"
        fi
        
        if [[ "${running}" == "true" ]]; then
            log::success "✅ Running: ${running}"
        else
            log::error "❌ Running: ${running}"
        fi
        
        if [[ "${health}" == "healthy" ]]; then
            log::success "✅ Health: ${health}"
        else
            log::error "❌ Health: ${health}"
        fi
        
        if [[ -n "${details}" ]]; then
            log::info "Details:"
            echo "  ${details}"
        fi
    fi
    
    return 0
}

# Collect Pandas AI status data in format-agnostic structure
pandas_ai::status::collect_data() {
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
    
    local port=$(pandas_ai::get_port)
    local installed="false"
    local running="false" 
    local healthy="false"
    local version="unknown"
    local health_message="Pandas AI not installed"
    
    if pandas_ai::is_installed; then
        installed="true"
        
        if pandas_ai::is_running; then
            running="true"
            healthy="true"
            health_message="Pandas AI is running and operational"
        else
            health_message="Pandas AI is installed but not running"
        fi
        
        # Skip version check in fast mode
        if [[ "$fast_mode" == "false" ]]; then
            if [[ -f "${PANDAS_AI_VENV_DIR}/bin/pip" ]]; then
                version=$("${PANDAS_AI_VENV_DIR}/bin/pip" show pandasai 2>/dev/null | grep Version | cut -d' ' -f2 || echo "unknown")
            else
                version=$(python3 -m pip show pandasai 2>/dev/null | grep Version | cut -d' ' -f2 || echo "1.0.0")
            fi
        else
            version="N/A"
        fi
    fi
    
    # Build status data array
    local status_data=(
        "name" "pandas-ai"
        "category" "execution"
        "description" "Conversational AI for data analysis"
        "installed" "$installed"
        "running" "$running"
        "healthy" "$healthy"
        "health_message" "$health_message"
        "version" "$version"
        "port" "$port"
        "venv_dir" "${PANDAS_AI_VENV_DIR:-unknown}"
        "scripts_dir" "${PANDAS_AI_SCRIPTS_DIR:-unknown}"
    )
    
    # Return the collected data
    printf '%s\n' "${status_data[@]}"
}

# Display status in text format
pandas_ai::status::display_text() {
    local -A data
    
    # Convert array to associative array
    for ((i=1; i<=$#; i+=2)); do
        local key="${!i}"
        local value_idx=$((i+1))
        local value="${!value_idx}"
        data["$key"]="$value"
    done
    
    # Header
    log::header "🐼 Pandas AI Status"
    echo
    
    # Basic status
    log::info "📊 Basic Status:"
    if [[ "${data[installed]:-false}" == "true" ]]; then
        log::success "   ✅ Installed: Yes"
    else
        log::error "   ❌ Installed: No"
        echo
        log::info "💡 Installation Required:"
        log::info "   To install Pandas AI, run: resource-pandas-ai install"
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
    log::info "   🐍 Venv: ${data[venv_dir]:-unknown}"
    log::info "   📁 Scripts: ${data[scripts_dir]:-unknown}"
    echo
    
    log::info "📋 Status Message:"
    log::info "   ${data[health_message]:-No status message}"
    echo
}

# New main status function using standard wrapper
pandas_ai::status::new() {
    status::run_standard "pandas-ai" "pandas_ai::status::collect_data" "pandas_ai::status::display_text" "$@"
}

# Export functions for use by CLI
export -f pandas_ai::is_installed
export -f pandas_ai::is_running
export -f pandas_ai::status
export -f pandas_ai::status::new