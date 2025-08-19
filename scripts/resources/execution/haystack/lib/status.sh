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
    
    # Format output using the shared format utility
    if [[ "${format}" == "json" ]]; then
        format::output "json" "kv" \
            "name" "haystack" \
            "status" "${status}" \
            "health" "${health}" \
            "message" "${message}" \
            "details" "${details}"
    else
        echo "Status: ${status}"
        echo "Health: ${health}"
        echo "Message: ${message}"
        [[ -n "${details}" ]] && echo "Details: ${details}"
    fi
}