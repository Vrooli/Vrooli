#!/usr/bin/env bash
set -euo pipefail

# Get the directory of this script
PANDAS_AI_LIB_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Source dependencies
source "${PANDAS_AI_LIB_DIR}/common.sh"
source "${PANDAS_AI_LIB_DIR}/../../../../lib/utils/log.sh"
source "${PANDAS_AI_LIB_DIR}/../../../../lib/utils/format.sh"

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
        # Try venv pip first, fallback to system pip
        if [[ -f "${PANDAS_AI_VENV_DIR}/bin/pip" ]]; then
            version=$("${PANDAS_AI_VENV_DIR}/bin/pip" show pandasai 2>/dev/null | grep Version | cut -d' ' -f2 || echo "unknown")
        else
            version=$(python3 -m pip show pandasai 2>/dev/null | grep Version | cut -d' ' -f2 || echo "1.0.0")
        fi
    fi
    
    if pandas_ai::is_running; then
        running="true"
        # Check health endpoint
        if curl -sf "http://localhost:${port}/health" >/dev/null 2>&1; then
            health="healthy"
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

# Export functions for use by CLI
export -f pandas_ai::is_installed
export -f pandas_ai::is_running
export -f pandas_ai::status