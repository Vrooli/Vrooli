#!/usr/bin/env bash
################################################################################
# OpenRouter Info Library - v2.0 Universal Contract Compliant
# 
# Displays runtime information from runtime.json
################################################################################

set -euo pipefail

# Define directories using cached APP_ROOT
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
OPENROUTER_RESOURCE_DIR="${APP_ROOT}/resources/openrouter"

# Source dependencies
source "${APP_ROOT}/scripts/lib/utils/log.sh"
# Check for format.sh in multiple locations
if [[ -f "${APP_ROOT}/scripts/lib/utils/format.sh" ]]; then
    source "${APP_ROOT}/scripts/lib/utils/format.sh"
else
    # Define fallback format functions if format.sh not found
    format::bold() { echo -e "\033[1m$*\033[0m"; }
    format::dim() { echo -e "\033[2m$*\033[0m"; }
fi

# Show resource runtime information
openrouter::info() {
    local json_output="${1:-false}"
    local runtime_file="${OPENROUTER_RESOURCE_DIR}/config/runtime.json"
    
    if [[ ! -f "$runtime_file" ]]; then
        log::error "Runtime configuration not found at $runtime_file"
        return 1
    fi
    
    # If JSON output requested, just output the file
    if [[ "$json_output" == "--json" ]] || [[ "$json_output" == "true" ]]; then
        cat "$runtime_file"
        return 0
    fi
    
    # Parse and display formatted output
    local startup_order startup_timeout startup_time dependencies recovery priority
    
    startup_order=$(jq -r '.startup_order // "N/A"' "$runtime_file")
    startup_timeout=$(jq -r '.startup_timeout // "N/A"' "$runtime_file")
    startup_time=$(jq -r '.startup_time_estimate // "N/A"' "$runtime_file")
    dependencies=$(jq -r '.dependencies | if length > 0 then join(", ") else "none" end' "$runtime_file")
    recovery=$(jq -r '.recovery_attempts // "N/A"' "$runtime_file")
    priority=$(jq -r '.priority // "N/A"' "$runtime_file")
    
    echo -e "\033[1mOpenRouter Runtime Information\033[0m"
    echo -e "\033[2m================================\033[0m"
    echo
    echo -e "\033[1mStartup Configuration:\033[0m"
    echo "  Startup Order:     $startup_order"
    echo "  Startup Timeout:   ${startup_timeout}s"
    echo "  Startup Time Est:  $startup_time"
    echo "  Recovery Attempts: $recovery"
    echo "  Priority:          $priority"
    echo
    echo -e "\033[1mDependencies:\033[0m"
    echo "  $dependencies"
    echo
    echo -e "\033[1mResource Details:\033[0m"
    echo "  Type:              API Service (External)"
    echo "  Category:          AI/ML"
    echo "  Port Allocation:   None (External API)"
    echo "  Container:         None (API Proxy)"
    echo
    echo -e "\033[2mConfiguration file: $runtime_file\033[0m"
    
    return 0
}