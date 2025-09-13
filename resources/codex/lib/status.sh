#\!/usr/bin/env bash
# Codex Status Functions

# Set script directory for sourcing
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
CODEX_STATUS_DIR="${APP_ROOT}/resources/codex/lib"

# Source required utilities
# shellcheck disable=SC1091
source "${CODEX_STATUS_DIR}/common.sh"
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/format.sh"
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/resources/lib/status-args.sh"
# shellcheck disable=SC1091
source "${CODEX_STATUS_DIR}/orchestrator.sh" 2>/dev/null || true
# shellcheck disable=SC1091
source "${CODEX_STATUS_DIR}/tools/registry.sh" 2>/dev/null || true
# shellcheck disable=SC1091
source "${CODEX_STATUS_DIR}/workspace/manager.sh" 2>/dev/null || true
# shellcheck disable=SC1091
source "${CODEX_STATUS_DIR}/models/model-selector.sh" 2>/dev/null || true

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
    
    # Check Codex CLI status
    local cli_installed="false"
    local cli_version="not installed"
    if type -t codex::cli::is_installed &>/dev/null && codex::cli::is_installed; then
        cli_installed="true"
        cli_version=$(codex::cli::version 2>/dev/null || echo "unknown")
        
        # Update details to mention CLI availability
        if [[ "$cli_installed" == "true" ]] && [[ "$api_configured" == "true" ]]; then
            details="${details}. Codex CLI agent available (v${cli_version})"
        fi
    fi
    
    # Check new architecture components (skip in fast mode)
    local orchestrator_available="false"
    local tools_available="false" 
    local tools_count=0
    local workspace_available="false"
    local workspaces_count=0
    local models_available="false"
    
    if [[ "$fast" == "false" ]]; then
        # Check orchestrator
        if type -t orchestrator::execute &>/dev/null; then
            orchestrator_available="true"
        fi
        
        # Check tools registry
        if type -t tool_registry::list_tools &>/dev/null; then
            tools_available="true"
            local tools_result
            tools_result=$(tool_registry::list_tools 2>/dev/null || echo '[]')
            tools_count=$(echo "$tools_result" | jq 'length' 2>/dev/null || echo "0")
        fi
        
        # Check workspace manager
        if type -t workspace_manager::list &>/dev/null; then
            workspace_available="true"
            local workspace_result
            workspace_result=$(workspace_manager::list "active" 2>/dev/null || echo '[]')
            workspaces_count=$(echo "$workspace_result" | jq 'length' 2>/dev/null || echo "0")
        fi
        
        # Check model selector
        if type -t model_selector::get_best_model &>/dev/null; then
            models_available="true"
        fi
        
        # Update details with architecture info
        if [[ "$orchestrator_available" == "true" ]]; then
            details="${details}. New orchestrator architecture available"
        fi
    elif [[ "$fast" == "true" ]]; then
        # Fast mode - assume available if functions exist
        orchestrator_available=$(type -t orchestrator::execute &>/dev/null && echo "true" || echo "false")
        tools_available=$(type -t tool_registry::list_tools &>/dev/null && echo "true" || echo "false")
        workspace_available=$(type -t workspace_manager::list &>/dev/null && echo "true" || echo "false")
        models_available=$(type -t model_selector::get_best_model &>/dev/null && echo "true" || echo "false")
        tools_count="N/A"
        workspaces_count="N/A"
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
    echo "cli_installed"
    echo "${cli_installed}"
    echo "cli_version"
    echo "${cli_version}"
    echo "orchestrator_available"
    echo "${orchestrator_available}"
    echo "tools_available"
    echo "${tools_available}"
    echo "tools_count"
    echo "${tools_count}"
    echo "workspace_available"
    echo "${workspace_available}"
    echo "workspaces_count"
    echo "${workspaces_count}"
    echo "models_available" 
    echo "${models_available}"
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
    echo "CLI Installed: ${data[cli_installed]}"
    echo "CLI Version: ${data[cli_version]}"
    echo ""
    echo "=== New Architecture Status ==="
    echo "Orchestrator Available: ${data[orchestrator_available]}"
    echo "Tools Available: ${data[tools_available]}"
    echo "Tools Count: ${data[tools_count]}"
    echo "Workspace Available: ${data[workspace_available]}"
    echo "Active Workspaces: ${data[workspaces_count]}"
    echo "Models Available: ${data[models_available]}"
    echo ""
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
