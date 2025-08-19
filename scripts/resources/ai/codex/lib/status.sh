#\!/usr/bin/env bash
# Codex Status Functions

# Set script directory for sourcing
CODEX_STATUS_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)

# Source required utilities
# shellcheck disable=SC1091
source "${CODEX_STATUS_DIR}/common.sh"
# shellcheck disable=SC1091
source "${CODEX_STATUS_DIR}/../../../../lib/utils/format.sh" 2>/dev/null || true

#######################################
# Check Codex status
# Arguments:
#   --format: Output format (text/json)
# Returns:
#   0 if healthy, 1 otherwise
#######################################
codex::status() {
    local format="text"
    
    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --format)
                format="${2:-text}"
                shift 2
                ;;
            *)
                shift
                ;;
        esac
    done
    
    local status="stopped"
    local health="unhealthy"
    local version="${CODEX_DEFAULT_MODEL}"
    local api_configured="false"
    local api_available="false"
    local details=""
    local running="false"
    local installed="false"
    local healthy="false"
    
    # Check status file
    if [[ -f "${CODEX_STATUS_FILE}" ]]; then
        local file_status
        file_status=$(cat "${CODEX_STATUS_FILE}" 2>/dev/null || echo "stopped")
        if [[ "${file_status}" == "running" ]]; then
            status="running"
            running="true"
        fi
    fi
    
    # Check configuration
    if codex::is_configured; then
        api_configured="true"
        installed="true"
        
        # Check API availability
        if codex::is_available; then
            api_available="true"
            health="healthy"
            healthy="true"
            details="Codex API is available and configured"
            
            # Mark as running if healthy
            if [[ "${status}" != "running" ]]; then
                status="running"
                running="true"
                codex::save_status "running"
            fi
        else
            health="partial"
            details="Codex configured but API not responding"
        fi
    else
        details="OpenAI API key not configured"
    fi
    
    # Count injected scripts
    local script_count=0
    if [[ -d "${CODEX_SCRIPTS_DIR}" ]]; then
        script_count=$(find "${CODEX_SCRIPTS_DIR}" -name "*.py" 2>/dev/null | wc -l)
    fi
    
    # Count outputs
    local output_count=0
    if [[ -d "${CODEX_OUTPUT_DIR}" ]]; then
        output_count=$(find "${CODEX_OUTPUT_DIR}" -type f 2>/dev/null | wc -l)
    fi
    
    # Build output data array for format utility
    local -a output_data=(
        "name" "${CODEX_NAME}"
        "status" "${status}"
        "installed" "${installed}"
        "running" "${running}"
        "health" "${health}"
        "healthy" "${healthy}"
        "version" "${version}"
        "api_configured" "${api_configured}"
        "api_available" "${api_available}"
        "scripts" "${script_count}"
        "outputs" "${output_count}"
        "message" "${details}"
        "description" "${CODEX_DESCRIPTION}"
        "category" "${CODEX_CATEGORY}"
    )
    
    # Use standard formatter if available, fallback to text
    if type -t format::key_value &>/dev/null; then
        format::key_value "${format}" "${output_data[@]}"
    else
        # Fallback to text output
        echo "Name: ${CODEX_NAME}"
        echo "Status: ${status}"
        echo "Installed: ${installed}"
        echo "Running: ${running}"
        echo "Health: ${health}"
        echo "Healthy: ${healthy}"
        echo "Version: ${version}"
        echo "API Configured: ${api_configured}"
        echo "API Available: ${api_available}"
        echo "Scripts: ${script_count}"
        echo "Outputs: ${output_count}"
        echo "Message: ${details}"
        echo "Description: ${CODEX_DESCRIPTION}"
        echo "Category: ${CODEX_CATEGORY}"
    fi
    
    # Return appropriate exit code
    if [[ "${healthy}" == "true" ]]; then
        return 0
    else
        return 1
    fi
}

# Main entry point
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    codex::status "$@"
fi
