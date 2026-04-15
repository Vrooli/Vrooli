#!/usr/bin/env bash
################################################################################
# OpenRouter Info Library - v2.0 Universal Contract Compliant
# 
# Displays runtime information from resource.json
################################################################################

set -euo pipefail

# Resolve local resource paths
SCRIPT_DIR="$(builtin cd "${BASH_SOURCE[0]%/*}" && builtin pwd)"
RESOURCE_DIR="$(builtin cd "${SCRIPT_DIR}/.." && builtin pwd)"
REPO_ROOT="$(builtin cd "${RESOURCE_DIR}/../.." && builtin pwd)"
OPENROUTER_RESOURCE_DIR="${RESOURCE_DIR}"

# Source dependencies
source "${REPO_ROOT}/scripts/lib/utils/log.sh"
# Check for format.sh in multiple locations
if [[ -f "${REPO_ROOT}/scripts/lib/utils/format.sh" ]]; then
    source "${REPO_ROOT}/scripts/lib/utils/format.sh"
else
    # Define fallback format functions if format.sh not found
    format::bold() { echo -e "\033[1m$*\033[0m"; }
    format::dim() { echo -e "\033[2m$*\033[0m"; }
fi

# Show resource runtime information
openrouter::info() {
    local json_output="${1:-false}"
    local manifest_file="${OPENROUTER_RESOURCE_DIR}/resource.json"
    
    if [[ ! -f "$manifest_file" ]]; then
        log::error "Resource manifest not found at $manifest_file"
        return 1
    fi
    
    # If JSON output requested, just output the file
    if [[ "$json_output" == "--json" ]] || [[ "$json_output" == "true" ]]; then
        jq '.orchestration // {}' "$manifest_file"
        return 0
    fi
    
    # Parse and display formatted output
    local startup_order startup_timeout startup_time dependencies recovery priority
    
    startup_order=$(jq -r '.orchestration.startup_order // "N/A"' "$manifest_file")
    startup_timeout=$(jq -r '.orchestration.startup_timeout_seconds // "N/A"' "$manifest_file")
    startup_time=$(jq -r '.orchestration.startup_time_estimate // "N/A"' "$manifest_file")
    dependencies=$(jq -r '.orchestration.dependencies | if length > 0 then join(", ") else "none" end' "$manifest_file")
    recovery=$(jq -r '.orchestration.recovery_attempts // "N/A"' "$manifest_file")
    priority=$(jq -r '.orchestration.priority // "N/A"' "$manifest_file")
    
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
    echo -e "\033[2mConfiguration file: $manifest_file\033[0m"
    
    return 0
}
