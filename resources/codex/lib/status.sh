#\!/usr/bin/env bash
# Codex Status Functions

# Set script directory for sourcing
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*/../../.." && builtin pwd)}"
CODEX_STATUS_DIR="${APP_ROOT}/resources/codex/lib"

# Source required utilities
# shellcheck disable=SC1091
source "${CODEX_STATUS_DIR}/common.sh"
# shellcheck disable=SC1091
source "${CODEX_STATUS_DIR}/../../../../lib/utils/format.sh" 2>/dev/null || true
# shellcheck disable=SC1091
source "${CODEX_STATUS_DIR}/../../../lib/status-args.sh"

#######################################
# Collect Codex status data
# Arguments:
#   --fast: Skip expensive operations
# Returns:
#   Key-value pairs (one per line)
#######################################
codex::status::collect_data() {
    local fast="false"
    
    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --fast)
                fast="true"
                shift
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
        
        # Check API availability (skip in fast mode)
        if [[ "$fast" == "true" ]]; then
            # Assume available if configured in fast mode
            api_available="N/A"
            health="healthy"
            healthy="true"
            details="Codex API configured (fast mode)"
            status="running"
            running="true"
        elif codex::is_available; then
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
    
    # Count injected scripts (skip in fast mode)
    local script_count=0
    if [[ "$fast" == "false" ]] && [[ -d "${CODEX_SCRIPTS_DIR}" ]]; then
        script_count=$(find "${CODEX_SCRIPTS_DIR}" -name "*.py" 2>/dev/null | wc -l)
    elif [[ "$fast" == "true" ]]; then
        script_count="N/A"
    fi
    
    # Count outputs (skip in fast mode)
    local output_count=0
    if [[ "$fast" == "false" ]] && [[ -d "${CODEX_OUTPUT_DIR}" ]]; then
        output_count=$(find "${CODEX_OUTPUT_DIR}" -type f 2>/dev/null | wc -l)
    elif [[ "$fast" == "true" ]]; then
        output_count="N/A"
    fi
    
    # Output data as key-value pairs
    echo "name"
    echo "${CODEX_NAME}"
    echo "status"
    echo "${status}"
    echo "installed"
    echo "${installed}"
    echo "running"
    echo "${running}"
    echo "health"
    echo "${health}"
    echo "healthy"
    echo "${healthy}"
    echo "version"
    echo "${version}"
    echo "api_configured"
    echo "${api_configured}"
    echo "api_available"
    echo "${api_available}"
    echo "scripts"
    echo "${script_count}"
    echo "outputs"
    echo "${output_count}"
    echo "message"
    echo "${details}"
    echo "description"
    echo "${CODEX_DESCRIPTION}"
    echo "category"
    echo "${CODEX_CATEGORY}"
}

#######################################
# Display Codex status in text format
# Arguments:
#   data_array: Array of key-value pairs
#######################################
codex::status::display_text() {
    local -a data_array=("$@")
    
    # Convert array to associative array for easier access
    local -A data
    for ((i=0; i<${#data_array[@]}; i+=2)); do
        data["${data_array[i]}"]="${data_array[i+1]}"
    done
    
    echo "Name: ${data[name]}"
    echo "Status: ${data[status]}"
    echo "Installed: ${data[installed]}"
    echo "Running: ${data[running]}"
    echo "Health: ${data[health]}"
    echo "Healthy: ${data[healthy]}"
    echo "Version: ${data[version]}"
    echo "API Configured: ${data[api_configured]}"
    echo "API Available: ${data[api_available]}"
    echo "Scripts: ${data[scripts]}"
    echo "Outputs: ${data[outputs]}"
    echo "Message: ${data[message]}"
    echo "Description: ${data[description]}"
    echo "Category: ${data[category]}"
}

#######################################
# Check Codex status
# Arguments:
#   --format: Output format (text/json)
#   --fast: Skip expensive operations
# Returns:
#   0 if healthy, 1 otherwise
#######################################
codex::status() {
    status::run_standard "codex" "codex::status::collect_data" "codex::status::display_text" "$@"
}

# Main entry point
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    codex::status "$@"
fi
