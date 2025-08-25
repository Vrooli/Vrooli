#!/usr/bin/env bash
# n8n Recovery Functions - Minimal wrapper using recovery framework

# Source guard to prevent multiple sourcing
[[ -n "${_N8N_RECOVERY_SOURCED:-}" ]] && return 0
export _N8N_RECOVERY_SOURCED=1

# Source core and frameworks
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*/../../.." && builtin pwd)}"
N8N_LIB_DIR="${APP_ROOT}/resources/n8n/lib"

# shellcheck disable=SC1091
source "${N8N_LIB_DIR}/../../../lib/utils/var.sh"
# shellcheck disable=SC1091
source "${var_SCRIPTS_RESOURCES_LIB_DIR}/backup-framework.sh"
# shellcheck disable=SC1091
source "${N8N_LIB_DIR}/core.sh"

#######################################
# Automatic recovery from corruption (wrapper)
# Returns: 0 on success, 1 on failure
#######################################
n8n::auto_recover() {
    # Use the new backup framework-based recovery
    n8n::recover
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
    
    # Initialize backup framework (creates backup directories)
    backup::init >/dev/null 2>&1
    
    # Fix permissions
    chown -R 1000:1000 "$N8N_DATA_DIR" 2>/dev/null || true
    chmod -R u+rw "$N8N_DATA_DIR" 2>/dev/null || true
    
    return 0
}