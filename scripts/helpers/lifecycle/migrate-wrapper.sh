#!/usr/bin/env bash
set -euo pipefail

################################################################################
# Migration Wrapper for Lifecycle System
#
# This wrapper allows existing scripts (setup.sh, develop.sh, etc.) to
# gradually migrate to the lifecycle engine while maintaining backward
# compatibility.
#
# Usage:
#   source migrate-wrapper.sh
#   lifecycle::migrate_or_fallback "setup" "$@" || setup::monorepo "$@"
################################################################################

WRAPPER_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
LIFECYCLE_ENGINE="${WRAPPER_DIR}/engine.sh"

#######################################
# Check if lifecycle configuration exists for phase
# Arguments:
#   $1 - Phase name (setup, develop, build, etc.)
# Returns:
#   0 if lifecycle config exists, 1 otherwise
#######################################
lifecycle::has_config() {
    local phase="$1"
    local service_json_path
    
    # Find service.json
    if [[ -f "${PROJECT_ROOT:-$(pwd)}/.vrooli/service.json" ]]; then
        service_json_path="${PROJECT_ROOT:-$(pwd)}/.vrooli/service.json"
    elif [[ -f "${PROJECT_ROOT:-$(pwd)}/service.json" ]]; then
        service_json_path="${PROJECT_ROOT:-$(pwd)}/service.json"
    else
        return 1
    fi
    
    # Check if lifecycle phase exists
    local phase_exists
    phase_exists=$(jq -r ".lifecycle.$phase // .lifecycle.hooks.$phase // empty" "$service_json_path" 2>/dev/null)
    
    [[ -n "$phase_exists" ]] && return 0 || return 1
}

#######################################
# Try to use lifecycle engine, fall back if not available
# Arguments:
#   $1 - Phase name
#   $@ - All arguments to pass
# Returns:
#   0 if lifecycle succeeded, 1 to trigger fallback
#######################################
lifecycle::migrate_or_fallback() {
    local phase="$1"
    shift
    
    # Check if lifecycle engine exists
    if [[ ! -f "$LIFECYCLE_ENGINE" ]]; then
        log::debug "Lifecycle engine not found, using fallback"
        return 1
    fi
    
    # Check if lifecycle configuration exists
    if ! lifecycle::has_config "$phase"; then
        log::debug "No lifecycle config for $phase, using fallback"
        return 1
    fi
    
    # Check if explicitly disabled
    if [[ "${USE_LIFECYCLE:-true}" == "false" ]]; then
        log::debug "Lifecycle explicitly disabled, using fallback"
        return 1
    fi
    
    # Use lifecycle engine
    log::info "Using lifecycle engine for: $phase"
    "$LIFECYCLE_ENGINE" "$phase" "$@"
    return $?
}

#######################################
# Check if we should use lifecycle for this invocation
# Returns:
#   0 if should use lifecycle, 1 otherwise
#######################################
lifecycle::should_use() {
    # Check environment variable
    [[ "${USE_LIFECYCLE:-true}" == "false" ]] && return 1
    
    # Check if engine is available
    [[ ! -f "$LIFECYCLE_ENGINE" ]] && return 1
    
    # Check if service.json has lifecycle config
    [[ ! -f "${PROJECT_ROOT:-$(pwd)}/.vrooli/service.json" ]] && return 1
    
    return 0
}