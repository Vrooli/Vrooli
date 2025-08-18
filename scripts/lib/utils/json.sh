#!/usr/bin/env bash
################################################################################
# json.sh - Unified JSON Utility Library for Service Configuration
# 
# Provides centralized, robust functions for parsing service.json files
# throughout the Vrooli codebase. Replaces 200+ lines of duplicate parsing
# logic with standardized, well-tested utilities.
#
# Usage:
#   source "$(dirname "${BASH_SOURCE[0]}")/../utils/json.sh"
#   config=$(json::load_service_config)
#   enabled_resources=$(json::get_enabled_resources "storage")
#
################################################################################

set -euo pipefail

# Source guard to prevent re-sourcing
[[ -n "${_JSON_SH_SOURCED:-}" ]] && return 0
_JSON_SH_SOURCED=1

# Get the directory where this script is located
JSON_LIB_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Source required utilities
# shellcheck disable=SC1091
source "${JSON_LIB_DIR}/var.sh"
# shellcheck disable=SC1091  
source "${var_LOG_FILE}"
# shellcheck disable=SC1091
source "${var_SYSTEM_COMMANDS_FILE}"
# shellcheck disable=SC1091
source "${var_LIB_SYSTEM_DIR}/trash.sh" 2>/dev/null || true

# Global variables for caching and configuration
declare -g JSON_CONFIG_CACHE=""
declare -g JSON_CONFIG_PATH=""
declare -g JSON_DEBUG="${JSON_DEBUG:-false}"

################################################################################
# Core Configuration Loading Functions
################################################################################

#######################################
# Find service.json using standardized path resolution
# Returns: Path to service.json file
# Exit codes: 0 on success, 1 if not found
#######################################
json::find_service_config() {
    local config_paths=()
    
    # 1. Explicitly set path (highest priority)
    [[ -n "${SERVICE_JSON_PATH:-}" ]] && config_paths+=("${SERVICE_JSON_PATH}")
    
    # 2. Current directory (project-specific)
    config_paths+=("$(pwd)/.vrooli/service.json")
    
    # 3. Script context (scenario/app specific)
    [[ -n "${SCENARIO_DIR:-}" ]] && config_paths+=("${SCENARIO_DIR}/.vrooli/service.json")
    [[ -n "${APP_ROOT:-}" ]] && config_paths+=("${APP_ROOT}/config/service.json")
    
    # 4. Project root (fallback)
    [[ -n "${var_SERVICE_JSON_FILE:-}" ]] && config_paths+=("${var_SERVICE_JSON_FILE}")
    
    # 5. Global config (lowest priority)
    [[ -n "${HOME:-}" ]] && config_paths+=("${HOME}/.vrooli/service.json")
    
    local path
    for path in "${config_paths[@]}"; do
        if [[ -n "$path" && -f "$path" ]]; then
            [[ "$JSON_DEBUG" == "true" ]] && log::debug "Found service.json: $path"
            echo "$path"
            return 0
        fi
        [[ "$JSON_DEBUG" == "true" && -n "$path" ]] && log::debug "Checked path: $path (not found)"
    done
    
    log::error "No service.json found. Searched paths:"
    for path in "${config_paths[@]}"; do
        [[ -n "$path" ]] && echo "  - $path" >&2
    done
    return 1
}

#######################################
# Load service configuration with caching and validation
# Arguments:
#   $1 - Config path (optional, uses path resolution if not provided)
# Returns: 0 on success, 1 on error
# Sets: JSON_CONFIG_CACHE, JSON_CONFIG_PATH
#######################################
json::load_service_config() {
    local config_path="${1:-}"
    
    # Use path resolution if no path provided
    if [[ -z "$config_path" ]]; then
        if ! config_path=$(json::find_service_config); then
            return 1
        fi
    fi
    
    # Return cached config if same path and still valid
    if [[ "$config_path" == "$JSON_CONFIG_PATH" && -n "$JSON_CONFIG_CACHE" ]]; then
        [[ "$JSON_DEBUG" == "true" ]] && log::debug "Using cached config: $config_path"
        return 0
    fi
    
    # Validate file exists and is readable
    if [[ ! -f "$config_path" ]]; then
        log::error "Configuration file not found: $config_path"
        return 1
    fi
    
    if [[ ! -r "$config_path" ]]; then
        log::error "Configuration file not readable: $config_path"
        return 1
    fi
    
    # Check for required JSON parser
    if ! system::is_command "jq"; then
        log::error "jq command not available. Please install jq to parse service.json"
        log::info "Install with: sudo apt-get install jq  # or equivalent for your system"
        return 1
    fi
    
    # Load and validate JSON syntax
    local config_content
    if ! config_content=$(cat "$config_path" 2>/dev/null); then
        log::error "Failed to read configuration file: $config_path"
        return 1
    fi
    
    if [[ -z "$config_content" ]]; then
        log::error "Configuration file is empty: $config_path"
        return 1
    fi
    
    # Validate JSON syntax
    if ! echo "$config_content" | jq empty 2>/dev/null; then
        log::error "Invalid JSON syntax in: $config_path"
        log::info "Use 'jq . $config_path' to check JSON syntax"
        return 1
    fi
    
    # Cache the loaded configuration
    JSON_CONFIG_CACHE="$config_content"
    JSON_CONFIG_PATH="$config_path"
    
    [[ "$JSON_DEBUG" == "true" ]] && log::debug "Loaded service config: $config_path"
    return 0
}

################################################################################
# Safe Value Extraction Functions
################################################################################

#######################################
# Safely get a JSON value with fallback and validation
# Arguments:
#   $1 - JSON path (e.g., '.resources.storage.postgres.enabled')
#   $2 - Default value (optional, defaults to empty string)
#   $3 - Config path (optional, uses loaded config if not provided)
# Returns: The extracted value or default
# Exit codes: 0 on success, 1 on error
#######################################
json::get_value() {
    local json_path="${1:-}"
    local default_value="${2:-}"
    local config_path="${3:-}"
    
    if [[ -z "$json_path" ]]; then
        log::error "JSON path required for json::get_value"
        return 1
    fi
    
    # Load config if not cached or different path specified
    if [[ -n "$config_path" ]] || [[ -z "$JSON_CONFIG_CACHE" ]]; then
        if ! json::load_service_config "$config_path"; then
            echo "$default_value"
            return 1
        fi
    fi
    
    # Extract value with proper error handling
    local value
    if value=$(echo "$JSON_CONFIG_CACHE" | jq -r "$json_path // empty" 2>/dev/null); then
        if [[ -n "$value" && "$value" != "null" ]]; then
            echo "$value"
            return 0
        fi
    fi
    
    # Return default if path doesn't exist or value is null/empty
    echo "$default_value"
    return 0
}

#######################################
# Check if a JSON path exists
# Arguments:
#   $1 - JSON path to check
#   $2 - Config path (optional)
# Returns: 0 if exists, 1 if not
#######################################
json::path_exists() {
    local json_path="${1:-}"
    local config_path="${2:-}"
    
    if [[ -z "$json_path" ]]; then
        log::error "JSON path required for json::path_exists"
        return 1
    fi
    
    if [[ -n "$config_path" ]] || [[ -z "$JSON_CONFIG_CACHE" ]]; then
        if ! json::load_service_config "$config_path"; then
            return 1
        fi
    fi
    
    echo "$JSON_CONFIG_CACHE" | jq -e "$json_path" >/dev/null 2>&1
}

################################################################################
# Resource Management Functions
################################################################################

#######################################
# Get enabled resources by category with filtering
# Arguments:
#   $1 - Category (optional, e.g., 'storage', 'ai', 'automation')
#   $2 - Config path (optional)
# Returns: Space-separated list of enabled resource names
#######################################
json::get_enabled_resources() {
    local category="${1:-}"
    local config_path="${2:-}"
    
    if [[ -n "$config_path" ]] || [[ -z "$JSON_CONFIG_CACHE" ]]; then
        if ! json::load_service_config "$config_path"; then
            return 1
        fi
    fi
    
    local jq_filter
    if [[ -n "$category" ]]; then
        # Get enabled resources from specific category
        jq_filter=".resources.${category} | to_entries[] | select(.value.enabled == true) | .key"
    else
        # Get all enabled resources across all categories
        jq_filter='.resources | to_entries[] | .value | to_entries[] | select(.value.enabled == true) | .key'
    fi
    
    echo "$JSON_CONFIG_CACHE" | jq -r "$jq_filter" 2>/dev/null | tr '\n' ' ' | xargs
}

#######################################
# Get required resources by category
# Arguments:
#   $1 - Category (optional)
#   $2 - Config path (optional)
# Returns: Space-separated list of required resource names
#######################################
json::get_required_resources() {
    local category="${1:-}"
    local config_path="${2:-}"
    
    if [[ -n "$config_path" ]] || [[ -z "$JSON_CONFIG_CACHE" ]]; then
        if ! json::load_service_config "$config_path"; then
            return 1
        fi
    fi
    
    local jq_filter
    if [[ -n "$category" ]]; then
        # Get required resources from specific category
        jq_filter=".resources.${category} | to_entries[] | select(.value.required == true) | .key"
    else
        # Get all required resources across all categories  
        jq_filter='.resources | to_entries[] | .value | to_entries[] | select(.value.required == true) | .key'
    fi
    
    echo "$JSON_CONFIG_CACHE" | jq -r "$jq_filter" 2>/dev/null | tr '\n' ' ' | xargs
}

#######################################
# Get resource configuration
# Arguments:
#   $1 - Resource category (e.g., 'storage', 'ai')
#   $2 - Resource name (e.g., 'postgres', 'ollama')  
#   $3 - Config path (optional)
# Returns: JSON configuration object for the resource
#######################################
json::get_resource_config() {
    local category="${1:-}"
    local resource_name="${2:-}"
    local config_path="${3:-}"
    
    if [[ -z "$category" || -z "$resource_name" ]]; then
        log::error "Both category and resource name required for json::get_resource_config"
        return 1
    fi
    
    if [[ -n "$config_path" ]] || [[ -z "$JSON_CONFIG_CACHE" ]]; then
        if ! json::load_service_config "$config_path"; then
            return 1
        fi
    fi
    
    echo "$JSON_CONFIG_CACHE" | jq -c ".resources.${category}.${resource_name} // {}" 2>/dev/null
}

################################################################################
# Lifecycle Configuration Functions  
################################################################################

#######################################
# Get lifecycle phase configuration
# Arguments:
#   $1 - Phase name (e.g., 'setup', 'develop', 'build')
#   $2 - Config path (optional)
# Returns: JSON configuration for the phase
#######################################
json::get_lifecycle_phase() {
    local phase="${1:-}"
    local config_path="${2:-}"
    
    if [[ -z "$phase" ]]; then
        log::error "Phase name required for json::get_lifecycle_phase"
        return 1
    fi
    
    if [[ -n "$config_path" ]] || [[ -z "$JSON_CONFIG_CACHE" ]]; then
        if ! json::load_service_config "$config_path"; then
            return 1
        fi
    fi
    
    echo "$JSON_CONFIG_CACHE" | jq -c ".lifecycle.${phase} // {}" 2>/dev/null
}

#######################################
# List available lifecycle phases
# Arguments:
#   $1 - Config path (optional)
# Returns: List of available phase names
#######################################
json::list_lifecycle_phases() {
    local config_path="${1:-}"
    
    if [[ -n "$config_path" ]] || [[ -z "$JSON_CONFIG_CACHE" ]]; then
        if ! json::load_service_config "$config_path"; then
            return 1
        fi
    fi
    
    echo "$JSON_CONFIG_CACHE" | jq -r '.lifecycle | keys[]' 2>/dev/null
}

################################################################################
# Deployment and Testing Functions
################################################################################

#######################################
# Get deployment configuration value
# Arguments:
#   $1 - Deployment key (e.g., 'testing.ui.required', 'testing.timeout')
#   $2 - Default value (optional)
#   $3 - Config path (optional)
# Returns: Deployment configuration value
#######################################
json::get_deployment_config() {
    local key="${1:-}"
    local default_value="${2:-}"
    local config_path="${3:-}"
    
    if [[ -z "$key" ]]; then
        log::error "Deployment key required for json::get_deployment_config"
        return 1
    fi
    
    json::get_value ".deployment.${key}" "$default_value" "$config_path"
}

#######################################
# Get testing configuration (common pattern in scenarios)
# Arguments:
#   $1 - Config path (optional)
# Returns: JSON object with testing configuration
#######################################
json::get_testing_config() {
    local config_path="${1:-}"
    
    if [[ -n "$config_path" ]] || [[ -z "$JSON_CONFIG_CACHE" ]]; then
        if ! json::load_service_config "$config_path"; then
            return 1
        fi
    fi
    
    echo "$JSON_CONFIG_CACHE" | jq -c '.deployment.testing // {}' 2>/dev/null
}

################################################################################
# Validation and Health Check Functions
################################################################################

#######################################
# Validate service configuration schema
# Arguments:
#   $1 - Config path (optional)
# Returns: 0 if valid, 1 if invalid
#######################################
json::validate_config() {
    local config_path="${1:-}"
    
    if [[ -n "$config_path" ]] || [[ -z "$JSON_CONFIG_CACHE" ]]; then
        if ! json::load_service_config "$config_path"; then
            return 1
        fi
    fi
    
    # Basic structural validation
    local required_sections=(".service" ".resources")
    local section
    
    for section in "${required_sections[@]}"; do
        if ! echo "$JSON_CONFIG_CACHE" | jq -e "$section" >/dev/null 2>&1; then
            log::error "Missing required section: $section"
            return 1
        fi
    done
    
    [[ "$JSON_DEBUG" == "true" ]] && log::debug "Configuration validation passed"
    return 0
}

################################################################################
# Utility and Helper Functions
################################################################################

#######################################
# Clear cached configuration (useful for testing)
#######################################
json::clear_cache() {
    JSON_CONFIG_CACHE=""
    JSON_CONFIG_PATH=""
    if [[ "${JSON_DEBUG:-false}" == "true" ]]; then
        log::debug "Cleared JSON configuration cache"
    fi
}

#######################################
# Get current cached configuration path
# Returns: Path to currently cached config, or empty if none
#######################################
json::get_cached_path() {
    echo "$JSON_CONFIG_PATH"
}

#######################################
# Enable or disable debug logging
# Arguments:
#   $1 - true/false for debug mode
#######################################
json::set_debug() {
    local debug_mode="${1:-false}"
    JSON_DEBUG="$debug_mode"
    if [[ "$debug_mode" == "true" ]]; then
        log::debug "JSON utility debug logging enabled"
    fi
}

################################################################################
# Backward Compatibility Functions (Phase 2 - Temporary)
################################################################################

# Note: These functions provide temporary backward compatibility during migration
# They will be REMOVED in Phase 5 cleanup

#######################################
# Legacy wrapper for direct jq calls (DEPRECATED)
# Use json::get_value instead
#######################################
json::legacy_extract() {
    local json_path="$1"
    local file_path="${2:-}"
    
    log::warning "json::legacy_extract is deprecated. Use json::get_value instead."
    json::get_value "$json_path" "" "$file_path"
}

################################################################################
# Export Functions
################################################################################

# Export all public functions for use by other scripts
export -f json::find_service_config
export -f json::load_service_config
export -f json::get_value
export -f json::path_exists
export -f json::get_enabled_resources
export -f json::get_required_resources
export -f json::get_resource_config
export -f json::get_lifecycle_phase
export -f json::list_lifecycle_phases
export -f json::get_deployment_config
export -f json::get_testing_config
export -f json::validate_config
export -f json::clear_cache
export -f json::get_cached_path
export -f json::set_debug

# Temporary backward compatibility exports (will be removed in Phase 5)
export -f json::legacy_extract

################################################################################
# Self-Test Function (for development and debugging)
################################################################################

#######################################
# Run basic self-tests of JSON utilities
# Returns: 0 if all tests pass, 1 if any fail
#######################################
json::self_test() {
    log::info "Running JSON utility self-tests..."
    
    local test_config='{"service":{"name":"test"},"resources":{"storage":{"postgres":{"enabled":true,"required":true}}}}'
    local temp_file
    temp_file=$(mktemp)
    
    echo "$test_config" > "$temp_file"
    
    # Test 1: Load config
    if ! json::load_service_config "$temp_file"; then
        log::error "Self-test failed: Could not load test config"
        trash::safe_remove "$temp_file" --temp
        return 1
    fi
    
    # Test 2: Get value
    local service_name
    service_name=$(json::get_value '.service.name' 'default' "$temp_file")
    if [[ "$service_name" != "test" ]]; then
        log::error "Self-test failed: Expected 'test', got '$service_name'"
        trash::safe_remove "$temp_file" --temp
        return 1
    fi
    
    # Test 3: Get enabled resources
    local enabled_resources
    enabled_resources=$(json::get_enabled_resources 'storage' "$temp_file")
    if [[ "$enabled_resources" != "postgres" ]]; then
        log::error "Self-test failed: Expected 'postgres', got '$enabled_resources'"
        trash::safe_remove "$temp_file" --temp
        return 1
    fi
    
    # Test 4: Path exists
    if ! json::path_exists '.service.name' "$temp_file"; then
        log::error "Self-test failed: Path should exist"
        trash::safe_remove "$temp_file" --temp
        return 1
    fi
    
    # Clean up
    rm -f "$temp_file"
    json::clear_cache
    
    log::success "All JSON utility self-tests passed"
    return 0
}

# Allow running self-test directly
if [[ "${1:-}" == "--self-test" ]]; then
    json::self_test
fi