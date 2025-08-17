#!/usr/bin/env bash
################################################################################
# Unified Configuration Utilities
# 
# Minimal configuration handling for service.json
# Consolidated from system/config.sh, service/config.sh, service/service_config.sh
################################################################################

set -euo pipefail

# Source dependencies
UTILS_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
source "$UTILS_DIR/var.sh"
source "$UTILS_DIR/log.sh"

################################################################################
# Basic Configuration Functions (from system/config.sh)
################################################################################

#######################################
# Load configuration from file
# Arguments:
#   $1 - config file path
# Returns:
#   0 on success, 1 on failure
#######################################
config::load() {
    local config_file="${1:-}"
    
    if [[ -z "$config_file" || ! -f "$config_file" ]]; then
        return 1
    fi
    
    # Source the config file
    # shellcheck disable=SC1090
    source "$config_file"
    return 0
}

#######################################
# Set configuration value
# Arguments:
#   $1 - key
#   $2 - value
#######################################
config::set() {
    local key="$1"
    local value="$2"
    export "$key=$value"
}

#######################################
# Get configuration value
# Arguments:
#   $1 - key
#   $2 - default value (optional)
# Returns:
#   Configuration value or default
#######################################
config::get() {
    local key="$1"
    local default="${2:-}"
    echo "${!key:-$default}"
}

#######################################
# Check if configuration exists
# Arguments:
#   $1 - key
# Returns:
#   0 if exists, 1 otherwise
#######################################
config::exists() {
    local key="$1"
    [[ -n "${!key:-}" ]]
}

################################################################################
# Service Configuration Functions (from service/config.sh)
################################################################################

#######################################
# Check if in development environment
# Returns:
#   0 if development, 1 otherwise
#######################################
config::is_development() {
    [[ "${ENVIRONMENT:-development}" == "development" ]]
}

#######################################
# Initialize configuration
# The main entry point used by setup.sh
#######################################
config::init() {
    log::info "Initializing configuration..."
    
    local vrooli_dir="${var_VROOLI_CONFIG_DIR}"
    
    # Ensure .vrooli directory exists
    if [[ ! -d "${vrooli_dir}" ]]; then
        log::debug "Creating .vrooli directory..."
        mkdir -p "${vrooli_dir}"
    fi
    
    # Check for service.json
    local service_json="${var_SERVICE_JSON_FILE}"
    if [[ -f "${service_json}" ]]; then
        log::debug "service.json found at ${service_json}"
    else
        log::warning "No service.json found at ${service_json}"
        log::info "Consider creating one to define lifecycle phases"
    fi
    
    return 0
}

# Export functions
export -f config::load
export -f config::set
export -f config::get
export -f config::exists
export -f config::is_development
export -f config::init