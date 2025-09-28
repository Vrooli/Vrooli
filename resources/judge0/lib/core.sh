#!/usr/bin/env bash
################################################################################
# Judge0 Core Library - v2.0 Contract Compliance
# 
# Core functionality for Judge0 resource management
################################################################################

set -euo pipefail

# Get script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
JUDGE0_ROOT="${SCRIPT_DIR}/.."

# Source configuration
source "${JUDGE0_ROOT}/config/defaults.sh"

# Show credentials for integration
judge0::core::credentials() {
    log::header "Judge0 Integration Credentials"
    
    local api_url="http://localhost:${JUDGE0_PORT:-2358}"
    local api_key="${JUDGE0_API_KEY:-}"
    
    log::info "API Configuration:"
    log::info "  • Endpoint: ${api_url}"
    log::info "  • Port: ${JUDGE0_PORT:-2358}"
    
    if [[ -n "${api_key}" ]]; then
        log::info "  • API Key: ${api_key:0:8}..."
    else
        log::info "  • API Key: (not configured)"
    fi
    
    log::info ""
    log::info "Integration Example:"
    cat << EOF
    # Python
    import requests
    
    response = requests.post(
        "${api_url}/submissions",
        json={
            "source_code": "print('Hello, World!')",
            "language_id": 92
        }
    )
    token = response.json()['token']
    
    # JavaScript
    const response = await fetch("${api_url}/submissions", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
            source_code: "console.log('Hello, World!')",
            language_id: 93
        })
    });
    const { token } = await response.json();
EOF
    
    log::info ""
    log::info "Available Languages: ${api_url}/languages"
    log::info "System Info: ${api_url}/system_info"
    
    return 0
}

# Get resource info
judge0::core::info() {
    local runtime_config="${JUDGE0_ROOT}/config/runtime.json"
    
    if [[ -f "$runtime_config" ]]; then
        cat "$runtime_config"
    else
        echo '{"name": "judge0", "version": "1.13.1", "status": "unknown"}'
    fi
}

# Main status function (delegates to status.sh)
judge0::core::status() {
    if declare -f judge0::status &>/dev/null; then
        judge0::status "$@"
    else
        log::error "Status function not available"
        return 1
    fi
}

# Health monitoring wrapper functions
judge0::health::monitor::quick() {
    "${SCRIPT_DIR}/health-monitor.sh" quick
}

judge0::health::monitor::detailed() {
    "${SCRIPT_DIR}/health-monitor.sh" detailed
}

judge0::health::monitor::trends() {
    "${SCRIPT_DIR}/health-monitor.sh" trends
}

judge0::health::monitor::start() {
    "${SCRIPT_DIR}/health-monitor.sh" monitor "${1:-30}"
}

judge0::health::monitor::stop() {
    "${SCRIPT_DIR}/health-monitor.sh" stop-monitor
}

judge0::health::monitor::cache_clear() {
    "${SCRIPT_DIR}/health-monitor.sh" clear-cache "${1:-all}"
}

# Export functions for CLI
export -f judge0::core::credentials
export -f judge0::core::info
export -f judge0::core::status
export -f judge0::health::monitor::quick
export -f judge0::health::monitor::detailed
export -f judge0::health::monitor::trends
export -f judge0::health::monitor::start
export -f judge0::health::monitor::stop
export -f judge0::health::monitor::cache_clear