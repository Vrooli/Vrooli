#!/usr/bin/env bash
# Lifecycle Engine - Configuration Module
# Handles loading and validation of service.json configurations
# 
# Modernized to use the standardized JSON utilities for consistent
# service.json parsing across the Vrooli codebase.

set -euo pipefail

# Source var.sh with relative path first
# shellcheck disable=SC1091
source "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/../../utils/var.sh"
# shellcheck disable=SC1091
source "${var_LOG_FILE}"
# shellcheck disable=SC1091
source "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/../../utils/json.sh"

# Guard against re-sourcing
[[ -n "${_CONFIG_MODULE_LOADED:-}" ]] && return 0
declare -gr _CONFIG_MODULE_LOADED=1

# Global variables for configuration
declare -g SERVICE_JSON=""
declare -g LIFECYCLE_CONFIG=""
declare -g PHASE_CONFIG=""
declare -g DEFAULTS_CONFIG=""

# Legacy search paths for backward compatibility
readonly CONFIG_SEARCH_PATHS=(
    "${var_SERVICE_JSON_FILE:-}"
    "./service.json"
    "${PROJECT_ROOT:-.}/.vrooli/service.json"
    "${PROJECT_ROOT:-.}/service.json"
)

#######################################
# Find service.json in standard locations
# Uses the standardized JSON utilities for path resolution
# Returns:
#   Path to service.json or empty if not found
#######################################
config::find_service_json() {
    # Use standardized JSON utility for path resolution
    json::find_service_config
}

#######################################
# Load service.json file
# Uses standardized JSON utilities for robust loading and validation
# Arguments:
#   $1 - Path to service.json (optional, uses path resolution if not provided)
# Returns:
#   0 on success, 1 on error
# Sets:
#   SERVICE_JSON global variable (for backward compatibility)
#   SERVICE_JSON_PATH global variable
#######################################
config::load_file() {
    local config_path="${1:-}"
    
    # Use standardized JSON utility for loading and validation
    if ! json::load_service_config "$config_path"; then
        return 1
    fi
    
    # Set global variables for backward compatibility
    SERVICE_JSON_PATH=$(json::get_cached_path)
    SERVICE_JSON=$(cat "$SERVICE_JSON_PATH" 2>/dev/null)
    
    export SERVICE_JSON_PATH
    export SERVICE_JSON
    
    return 0
}

#######################################
# Extract lifecycle configuration from service.json
# Uses standardized JSON utilities for robust extraction
# Returns:
#   0 on success, 1 on error
# Sets:
#   LIFECYCLE_CONFIG global variable
#   DEFAULTS_CONFIG global variable
#######################################
config::extract_lifecycle() {
    # Use standardized JSON utility to get lifecycle config
    LIFECYCLE_CONFIG=$(json::get_value '.lifecycle' '{}')
    
    if [[ "$LIFECYCLE_CONFIG" == "{}" ]]; then
        echo "Error: No lifecycle configuration found in service.json" >&2
        return 1
    fi
    
    # Extract defaults from root level first
    DEFAULTS_CONFIG=$(json::get_value '.defaults' '{}')
    
    # Also check for defaults inside lifecycle config as fallback
    if [[ "$DEFAULTS_CONFIG" == "{}" ]]; then
        DEFAULTS_CONFIG=$(echo "$LIFECYCLE_CONFIG" | jq -r '.defaults // {}' 2>/dev/null || echo '{}')
    fi
    
    return 0
}

#######################################
# Get configuration for specific phase
# Uses standardized JSON utilities with enhanced phase resolution
# Arguments:
#   $1 - Phase name
# Returns:
#   0 if phase exists, 1 if not found
# Sets:
#   PHASE_CONFIG global variable
#######################################
config::get_phase() {
    local phase="${1:-}"
    
    if [[ -z "$phase" ]]; then
        echo "Error: Phase name required" >&2
        return 1
    fi
    
    if [[ -z "$LIFECYCLE_CONFIG" ]]; then
        echo "Error: Lifecycle configuration not loaded" >&2
        return 1
    fi
    
    # Try to use standardized JSON utility first
    PHASE_CONFIG=$(json::get_lifecycle_phase "$phase" 2>/dev/null || echo "")
    
    # If standardized utility didn't find it, try legacy locations for compatibility
    if [[ -z "$PHASE_CONFIG" ]] || [[ "$PHASE_CONFIG" == "{}" ]]; then
        # Try under phases key (for test compatibility)
        PHASE_CONFIG=$(echo "$LIFECYCLE_CONFIG" | jq -r ".phases.\"$phase\" // empty" 2>/dev/null)
        
        # If still not found, try in hooks
        if [[ -z "$PHASE_CONFIG" ]] || [[ "$PHASE_CONFIG" == "null" ]]; then
            PHASE_CONFIG=$(echo "$LIFECYCLE_CONFIG" | jq -r ".hooks.\"$phase\" // empty" 2>/dev/null)
        fi
    fi
    
    # If still not found, try to inherit from parent configuration
    if [[ -z "$PHASE_CONFIG" ]] || [[ "$PHASE_CONFIG" == "null" ]]; then
        if config::try_inherit_phase "$phase"; then
            return 0
        fi
    fi
    
    # Check if phase was found
    if [[ -z "$PHASE_CONFIG" ]] || [[ "$PHASE_CONFIG" == "null" ]]; then
        return 1
    fi
    
    # For test compatibility, also echo the phase config
    echo "$PHASE_CONFIG"
    
    return 0
}

#######################################
# Try to inherit phase configuration from parent
# Arguments:
#   $1 - Phase name
# Returns:
#   0 if phase found in parent, 1 if not found
# Sets:
#   PHASE_CONFIG global variable
#######################################
config::try_inherit_phase() {
    local phase="${1:-}"
    
    # Check if inheritance is configured using standardized utilities
    local inheritance_config
    inheritance_config=$(json::get_value '.inheritance' '')
    
    if [[ -z "$inheritance_config" ]]; then
        return 1
    fi
    
    # Get the extends path using standardized utilities
    local extends_path
    extends_path=$(json::get_value '.inheritance.extends' '')
    
    if [[ -z "$extends_path" ]] || [[ "$extends_path" == "null" ]]; then
        return 1
    fi
    
    # Resolve the parent service.json path
    local parent_path=""
    if [[ "$extends_path" == /* ]]; then
        # Absolute path
        parent_path="$extends_path"
    else
        # Relative path from service.json location
        local service_dir
        service_dir=$(dirname "$SERVICE_JSON_PATH")
        parent_path="$service_dir/$extends_path"
    fi
    
    # Normalize path
    parent_path=$(cd "$(dirname "$parent_path")" 2>/dev/null && pwd)/$(basename "$parent_path") || return 1
    
    if [[ ! -f "$parent_path" ]]; then
        echo "Warning: Parent service.json not found at: $parent_path" >&2
        return 1
    fi
    
    # Load parent configuration
    local parent_json
    if ! parent_json=$(cat "$parent_path" 2>/dev/null); then
        return 1
    fi
    
    # Validate parent JSON
    if ! echo "$parent_json" | jq empty 2>/dev/null; then
        echo "Warning: Invalid JSON in parent configuration: $parent_path" >&2
        return 1
    fi
    
    # Extract parent lifecycle configuration
    local parent_lifecycle
    parent_lifecycle=$(echo "$parent_json" | jq -r '.lifecycle // {}')
    
    if [[ "$parent_lifecycle" == "{}" ]] || [[ "$parent_lifecycle" == "null" ]]; then
        return 1
    fi
    
    # Try to get phase from parent
    local parent_phase_config
    parent_phase_config=$(echo "$parent_lifecycle" | jq -r ".\"$phase\" // empty")
    
    # If not found in direct phases, try hooks
    if [[ -z "$parent_phase_config" ]] || [[ "$parent_phase_config" == "null" ]]; then
        parent_phase_config=$(echo "$parent_lifecycle" | jq -r ".hooks.\"$phase\" // empty")
    fi
    
    if [[ -z "$parent_phase_config" ]] || [[ "$parent_phase_config" == "null" ]]; then
        return 1
    fi
    
    # Set the phase config from parent
    PHASE_CONFIG="$parent_phase_config"
    
    # Log inheritance if verbose
    echo "Info: Phase '$phase' inherited from parent configuration at: $parent_path" >&2
    
    return 0
}

#######################################
# List available phases
# Uses standardized JSON utilities with legacy compatibility
# Returns:
#   List of phase names
#######################################
config::list_phases() {
    if [[ -z "$LIFECYCLE_CONFIG" ]]; then
        echo "Error: Lifecycle configuration not loaded" >&2
        return 1
    fi
    
    # Use standardized JSON utility for consistent phase listing
    local standard_phases
    standard_phases=$(json::list_lifecycle_phases 2>/dev/null || echo "")
    
    # If standard utility found phases, use them
    if [[ -n "$standard_phases" ]]; then
        echo "$standard_phases"
        return 0
    fi
    
    # Fallback to legacy logic for compatibility
    # Get direct phases
    local direct_phases
    direct_phases=$(echo "$LIFECYCLE_CONFIG" | jq -r 'keys[] | select(. != "version" and . != "defaults" and . != "hooks")' 2>/dev/null || echo "")
    
    # Get hook phases
    local hook_phases
    hook_phases=$(echo "$LIFECYCLE_CONFIG" | jq -r '.hooks // {} | keys[]' 2>/dev/null || echo "")
    
    # Combine and deduplicate
    {
        echo "$direct_phases"
        echo "$hook_phases"
    } | sort -u | grep -v '^$' || true
}

#######################################
# Get phase metadata
# Arguments:
#   $1 - Metadata key (description, requires, confirm, etc.)
# Returns:
#   Metadata value or empty string
#######################################
config::get_phase_metadata() {
    local key="${1:-}"
    
    if [[ -z "$key" ]] || [[ -z "$PHASE_CONFIG" ]]; then
        echo ""
        return 1
    fi
    
    echo "$PHASE_CONFIG" | jq -r ".\"$key\" // empty"
}

#######################################
# Get phase steps
# Arguments:
#   $1 - Step type (steps, pre_steps, post_steps)
# Returns:
#   JSON array of steps
#######################################
config::get_phase_steps() {
    local step_type="${1:-steps}"
    
    if [[ -z "$PHASE_CONFIG" ]]; then
        echo "[]"
        return 1
    fi
    
    echo "$PHASE_CONFIG" | jq -r ".\"$step_type\" // []"
}

#######################################
# Get targets configuration
# Returns:
#   JSON object of targets
#######################################
config::get_targets() {
    if [[ -z "$PHASE_CONFIG" ]]; then
        echo "{}"
        return 1
    fi
    
    echo "$PHASE_CONFIG" | jq -r '.targets // {}'
}

#######################################
# Check if target exists
# Arguments:
#   $1 - Target name
# Returns:
#   0 if exists, 1 if not
#######################################
config::target_exists() {
    local target="${1:-}"
    
    if [[ -z "$target" ]] || [[ -z "$PHASE_CONFIG" ]]; then
        return 1
    fi
    
    local targets
    targets=$(config::get_targets)
    
    if echo "$targets" | jq -e ".\"$target\"" >/dev/null 2>&1; then
        return 0
    fi
    
    return 1
}

#######################################
# Get default configuration values
# Arguments:
#   $1 - Config key (timeout, shell, error, etc.)
# Returns:
#   Default value or empty string
#######################################
config::get_default() {
    local key="${1:-}"
    
    if [[ -z "$key" ]] || [[ -z "$DEFAULTS_CONFIG" ]]; then
        echo ""
        return 1
    fi
    
    echo "$DEFAULTS_CONFIG" | jq -r ".\"$key\" // empty"
}

#######################################
# Validate configuration schema
# Uses standardized JSON utilities for robust validation
# Returns:
#   0 if valid, 1 if invalid
#######################################
config::validate_schema() {
    # Use standardized JSON validation
    if ! json::validate_config; then
        echo "Error: Configuration validation failed" >&2
        return 1
    fi
    
    # Check version field using standardized utilities
    local version
    version=$(json::get_value '.version' '')
    
    if [[ -z "$version" ]]; then
        echo "Warning: No version field in service.json" >&2
    fi
    
    # Validate lifecycle structure
    if [[ -z "$LIFECYCLE_CONFIG" ]] || [[ "$LIFECYCLE_CONFIG" == "{}" ]]; then
        echo "Error: No lifecycle configuration found" >&2
        return 1
    fi
    
    # Check for at least one phase
    local phase_count
    phase_count=$(config::list_phases | wc -l)
    
    if [[ $phase_count -eq 0 ]]; then
        echo "Error: No lifecycle phases defined" >&2
        return 1
    fi
    
    return 0
}

#######################################
# Export configuration for use by other modules
#######################################
config::export() {
    export SERVICE_JSON
    export LIFECYCLE_CONFIG
    export PHASE_CONFIG
    export DEFAULTS_CONFIG
}

#######################################
# Test-compatible wrapper functions
# These provide backward compatibility with existing tests
#######################################

#######################################
# Load service.json (wrapper for config::load_file)
# Arguments:
#   $1 - Path to service.json
# Returns:
#   0 on success, 1 on error
#######################################
config::load() {
    local config_path="${1:-}"
    
    if [[ -z "$config_path" ]]; then
        config_path=$(config::find_service_json) || return 1
    fi
    
    if ! config::load_file "$config_path"; then
        return 1
    fi
    
    # Also extract lifecycle configuration for tests
    config::extract_lifecycle || return 1
    
    # Export SERVICE_JSON for test compatibility
    export SERVICE_JSON
    
    return 0
}

#######################################
# Get lifecycle configuration (wrapper)
# Returns:
#   Lifecycle configuration JSON
#######################################
config::get_lifecycle() {
    config::extract_lifecycle >/dev/null 2>&1 || return 1
    echo "$LIFECYCLE_CONFIG"
}

#######################################
# Get defaults configuration
# Returns:
#   Defaults configuration JSON
#######################################
config::get_defaults() {
    if [[ -z "$DEFAULTS_CONFIG" ]] || [[ "$DEFAULTS_CONFIG" == "{}" ]]; then
        echo "{}"
    else
        echo "$DEFAULTS_CONFIG"
    fi
}

#######################################
# Set active phase for subsequent operations
# Arguments:
#   $1 - Phase name
# Returns:
#   0 on success, 1 on error
#######################################
config::set_phase() {
    local phase="${1:-}"
    
    if [[ -z "$phase" ]]; then
        echo "Error: Phase name required" >&2
        return 1
    fi
    
    if ! config::get_phase "$phase"; then
        return 1
    fi
    
    # Phase config is now set in PHASE_CONFIG global
    export PHASE_CONFIG
    return 0
}

#######################################
# Validate that a phase exists
# Arguments:
#   $1 - Phase name
# Returns:
#   0 if phase exists, 1 if not
#######################################
config::validate_phase() {
    local phase="${1:-}"
    
    if [[ -z "$phase" ]]; then
        echo "Error: Phase name required" >&2
        return 1
    fi
    
    # Try to get the phase - will return 1 if not found
    config::get_phase "$phase" >/dev/null 2>&1
}


# If sourced for testing, don't auto-execute
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    echo "This script should be sourced, not executed directly" >&2
    exit 1
fi