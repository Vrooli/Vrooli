#!/usr/bin/env bash
set -euo pipefail

################################################################################
# Runtime Resource Data Injection Engine
# 
# This is an adapted version of engine.sh designed for runtime use by
# generated scenario apps. It delegates resource-specific injection logic
# to co-located adapters, but operates within the context of a running app
# rather than during deployment.
#
# Key differences from deploy-time engine.sh:
#   - No rollback tracking (app manages its own lifecycle)
#   - Simpler operation modes (inject only, no status/list)
#   - Uses injection manifest instead of scenarios.json
#   - Integrated with app's resource management
#
# Usage:
#   runtime-engine.sh <injection-manifest.json>
#
################################################################################

# Source var.sh first to get standardized paths
SCRIPTS_SCENARIOS_INJECTION_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck disable=SC1091
source "${SCRIPTS_SCENARIOS_INJECTION_DIR}/../../lib/utils/var.sh"

# Source common utilities if available
if [[ -f "${var_SCRIPTS_RESOURCES_DIR}/common.sh" ]]; then
    # shellcheck disable=SC1091
    source "${var_SCRIPTS_RESOURCES_DIR}/common.sh"
fi

# Simple logging if common.sh not available
if ! declare -f log::info >/dev/null 2>&1; then
    log::info() { echo "[INFO] $*"; }
    log::error() { echo "[ERROR] $*" >&2; }
    log::success() { echo "[SUCCESS] $*"; }
    log::warning() { echo "[WARNING] $*"; }
fi

################################################################################
# Check if resource supports injection
# Arguments:
#   $1 - resource name
# Returns:
#   0 if injection supported, 1 otherwise
################################################################################
runtime_injection::resource_supports_injection() {
    local resource="$1"
    
    # Search for inject.sh script for this resource
    local inject_script
    inject_script=$(find "$var_SCRIPTS_RESOURCES_DIR" -name "inject.sh" -path "*/${resource}/*" 2>/dev/null | head -1)
    
    [[ -n "$inject_script" && -x "$inject_script" ]]
}

################################################################################
# Inject data for a single resource
# Arguments:
#   $1 - resource info JSON object
# Returns:
#   0 if successful, 1 if failed
################################################################################
runtime_injection::inject_resource() {
    local resource_info="$1"
    
    local resource_name
    local inject_script_path
    local init_data
    
    resource_name=$(echo "$resource_info" | jq -r '.name')
    inject_script_path=$(echo "$resource_info" | jq -r '.inject_script // empty')
    init_data=$(echo "$resource_info" | jq -c '.initialization')
    
    log::info "Processing resource: $resource_name"
    
    # Handle empty inject script path
    if [[ -z "$inject_script_path" ]]; then
        # Try to find inject script dynamically
        inject_script_path=$(find "$var_SCRIPTS_RESOURCES_DIR" -name "inject.sh" -path "*/${resource_name}/*" 2>/dev/null | head -1)
        
        if [[ -z "$inject_script_path" ]]; then
            log::warning "No injection handler for resource: $resource_name"
            return 0  # Not a failure, just skip
        fi
    else
        # Resolve relative path
        if [[ ! "$inject_script_path" =~ ^/ ]]; then
            inject_script_path="${var_ROOT_DIR}/${inject_script_path}"
        fi
    fi
    
    # Verify inject script exists and is executable
    if [[ ! -x "$inject_script_path" ]]; then
        log::error "Injection script not found or not executable: $inject_script_path"
        return 1
    fi
    
    # Check if we have initialization data
    if [[ -z "$init_data" || "$init_data" == "null" || "$init_data" == "{}" ]]; then
        log::info "No initialization data for resource: $resource_name"
        return 0
    fi
    
    # Execute injection
    log::info "Injecting data using: $inject_script_path"
    
    if "$inject_script_path" --inject "$init_data"; then
        log::success "✓ Successfully injected data for: $resource_name"
        return 0
    else
        log::error "✗ Failed to inject data for: $resource_name"
        return 1
    fi
}

################################################################################
# Main injection function
# Arguments:
#   $1 - path to injection manifest JSON
# Returns:
#   0 if all injections successful, 1 if any failed
################################################################################
runtime_injection::inject_from_manifest() {
    local manifest_file="$1"
    
    if [[ ! -f "$manifest_file" ]]; then
        log::error "Injection manifest not found: $manifest_file"
        return 1
    fi
    
    log::info "Loading injection manifest: $manifest_file"
    
    # Validate manifest is valid JSON
    if ! jq empty "$manifest_file" 2>/dev/null; then
        log::error "Invalid JSON in manifest file"
        return 1
    fi
    
    # Extract resource count
    local resource_count
    resource_count=$(jq '.resources | length' "$manifest_file")
    
    if [[ "$resource_count" -eq 0 ]]; then
        log::info "No resources to inject"
        return 0
    fi
    
    log::info "Found $resource_count resources to process"
    
    local failed_count=0
    local success_count=0
    
    # Process each resource
    for ((i=0; i<resource_count; i++)); do
        local resource_info
        resource_info=$(jq -c ".resources[$i]" "$manifest_file")
        
        if runtime_injection::inject_resource "$resource_info"; then
            ((success_count++))
        else
            ((failed_count++))
        fi
    done
    
    # Report results
    log::info "Injection complete: $success_count succeeded, $failed_count failed"
    
    if [[ "$failed_count" -gt 0 ]]; then
        log::error "Some injections failed"
        return 1
    else
        log::success "All injections completed successfully"
        return 0
    fi
}

################################################################################
# Display usage information
################################################################################
runtime_injection::usage() {
    cat << EOF
Runtime Resource Data Injection Engine

USAGE:
    $0 <injection-manifest.json>

DESCRIPTION:
    Injects initialization data into resources at runtime based on a
    pre-validated injection manifest generated during app packaging.

ARGUMENTS:
    injection-manifest.json    Path to the injection manifest file

EXAMPLE:
    $0 /app/manifests/injection.json

EOF
}

################################################################################
# Main execution
################################################################################
main() {
    if [[ $# -eq 0 ]]; then
        runtime_injection::usage
        exit 1
    fi
    
    local manifest_file="$1"
    
    # Perform injection
    if runtime_injection::inject_from_manifest "$manifest_file"; then
        exit 0
    else
        exit 1
    fi
}

# Execute main function if script is run directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi