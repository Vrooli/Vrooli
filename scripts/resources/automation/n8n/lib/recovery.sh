#!/usr/bin/env bash
# n8n Recovery Functions - Minimal wrapper using recovery framework

# Source core and frameworks
N8N_LIB_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)

# shellcheck disable=SC1091
source "${N8N_LIB_DIR}/../../../../lib/utils/var.sh"
# shellcheck disable=SC1091
source "${var_SCRIPTS_RESOURCES_LIB_DIR}/recovery-framework.sh"
# shellcheck disable=SC1091
source "${N8N_LIB_DIR}/core.sh"

#######################################
# Automatic recovery from corruption (wrapper)
# Returns: 0 on success, 1 on failure
#######################################
n8n::auto_recover() {
    local config
    config=$(n8n::get_recovery_config)
    recovery::auto_recover "$config"
}

#######################################
# Create directories with proper permissions
# Returns: 0 on success, 1 on failure
#######################################
n8n::create_directories() {
    # Create main data directory
    if [[ ! -d "$N8N_DATA_DIR" ]]; then
        mkdir -p "$N8N_DATA_DIR" || return 1
    fi
    
    # Create backup directory
    if [[ ! -d "${N8N_DATA_DIR}/backups" ]]; then
        mkdir -p "${N8N_DATA_DIR}/backups" || return 1
    fi
    
    # Fix permissions
    recovery::fix_permissions "$N8N_DATA_DIR" "1000:1000"
    
    return 0
}