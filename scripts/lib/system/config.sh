#!/usr/bin/env bash
################################################################################
# System Configuration Utilities
# 
# Handles system configuration and environment setup
################################################################################

LIB_SYSTEM_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)

# shellcheck disable=SC1091
source "$LIB_SYSTEM_DIR/../utils/var.sh"

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
# Check if configuration key exists
# Arguments:
#   $1 - key
# Returns:
#   0 if exists, 1 otherwise
#######################################
config::exists() {
    local key="$1"
    [[ -n "${!key:-}" ]]
}

# Export functions
export -f config::load
export -f config::set
export -f config::get
export -f config::exists