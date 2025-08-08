#!/usr/bin/env bash
# Lifecycle Engine - Configuration Module
# Handles loading and validation of service.json configurations

set -euo pipefail

# Get script directory
LIB_LIFECYCLE_LIB_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)

# Source common environment variables
source "$LIB_LIFECYCLE_LIB_DIR/../../utils/var.sh"

# Guard against re-sourcing
[[ -n "${_CONFIG_MODULE_LOADED:-}" ]] && return 0
declare -gr _CONFIG_MODULE_LOADED=1

# Global variables for configuration
declare -g SERVICE_JSON=""
declare -g LIFECYCLE_CONFIG=""
declare -g PHASE_CONFIG=""
declare -g DEFAULTS_CONFIG=""

# Default paths to search for service.json
readonly CONFIG_SEARCH_PATHS=(
    "${var_SERVICE_JSON_FILE:-}"
    "./service.json"
    "${PROJECT_ROOT:-.}/.vrooli/service.json"
    "${PROJECT_ROOT:-.}/service.json"
)

#######################################
# Find service.json in standard locations
# Returns:
#   Path to service.json or empty if not found
#######################################
config::find_service_json() {
    local config_path="${SERVICE_JSON_PATH:-}"
    
    # If path was explicitly provided, use it
    if [[ -n "$config_path" ]]; then
        if [[ -f "$config_path" ]]; then
            echo "$config_path"
            return 0
        else
            return 1
        fi
    fi
    
    # Search standard locations
    for path in "${CONFIG_SEARCH_PATHS[@]}"; do
        if [[ -f "$path" ]]; then
            echo "$path"
            return 0
        fi
    done
    
    return 1
}

#######################################
# Load service.json file
# Arguments:
#   $1 - Path to service.json
# Returns:
#   0 on success, 1 on error
# Sets:
#   SERVICE_JSON global variable
#######################################
config::load_file() {
    local config_path="${1:-}"
    
    if [[ -z "$config_path" ]] || [[ ! -f "$config_path" ]]; then
        echo "Error: Configuration file not found: $config_path" >&2
        return 1
    fi
    
    # Load and validate JSON
    if ! SERVICE_JSON=$(cat "$config_path" 2>/dev/null); then
        echo "Error: Failed to read configuration file: $config_path" >&2
        return 1
    fi
    
    # Validate JSON syntax
    if ! echo "$SERVICE_JSON" | jq empty 2>/dev/null; then
        echo "Error: Invalid JSON in configuration file: $config_path" >&2
        return 1
    fi
    
    SERVICE_JSON_PATH="$config_path"
    export SERVICE_JSON_PATH
    
    return 0
}

#######################################
# Extract lifecycle configuration from service.json
# Returns:
#   0 on success, 1 on error
# Sets:
#   LIFECYCLE_CONFIG global variable
#######################################
config::extract_lifecycle() {
    if [[ -z "$SERVICE_JSON" ]]; then
        echo "Error: No service.json loaded" >&2
        return 1
    fi
    
    LIFECYCLE_CONFIG=$(echo "$SERVICE_JSON" | jq -r '.lifecycle // {}')
    
    if [[ "$LIFECYCLE_CONFIG" == "{}" ]] || [[ "$LIFECYCLE_CONFIG" == "null" ]]; then
        echo "Error: No lifecycle configuration found in service.json" >&2
        return 1
    fi
    
    # Extract defaults if present
    DEFAULTS_CONFIG=$(echo "$LIFECYCLE_CONFIG" | jq -r '.defaults // {}')
    
    return 0
}

#######################################
# Get configuration for specific phase
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
    
    # Try direct phase first
    PHASE_CONFIG=$(echo "$LIFECYCLE_CONFIG" | jq -r ".\"$phase\" // empty")
    
    # If not found, try in hooks
    if [[ -z "$PHASE_CONFIG" ]] || [[ "$PHASE_CONFIG" == "null" ]]; then
        PHASE_CONFIG=$(echo "$LIFECYCLE_CONFIG" | jq -r ".hooks.\"$phase\" // empty")
    fi
    
    # Check if phase was found
    if [[ -z "$PHASE_CONFIG" ]] || [[ "$PHASE_CONFIG" == "null" ]]; then
        return 1
    fi
    
    return 0
}

#######################################
# List available phases
# Returns:
#   List of phase names
#######################################
config::list_phases() {
    if [[ -z "$LIFECYCLE_CONFIG" ]]; then
        echo "Error: Lifecycle configuration not loaded" >&2
        return 1
    fi
    
    # Get direct phases
    local direct_phases
    direct_phases=$(echo "$LIFECYCLE_CONFIG" | jq -r 'keys[] | select(. != "version" and . != "defaults" and . != "hooks")')
    
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
# Returns:
#   0 if valid, 1 if invalid
#######################################
config::validate_schema() {
    if [[ -z "$SERVICE_JSON" ]]; then
        echo "Error: No configuration loaded" >&2
        return 1
    fi
    
    # Check required fields
    local version
    version=$(echo "$SERVICE_JSON" | jq -r '.version // empty')
    
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

# If sourced for testing, don't auto-execute
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    echo "This script should be sourced, not executed directly" >&2
    exit 1
fi