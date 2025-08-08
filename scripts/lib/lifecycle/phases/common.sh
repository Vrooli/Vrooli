#!/usr/bin/env bash
################################################################################
# Common utilities for lifecycle phase handlers
# 
# Provides shared functions used across all universal phase handlers.
# This ensures consistency and reduces duplication.
################################################################################

set -euo pipefail

# Get the phases directory
LIB_LIFECYCLE_PHASES_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Source var.sh first to get all directory variables
# shellcheck disable=SC1091
source "${LIB_LIFECYCLE_PHASES_DIR}/../../utils/var.sh"

# Source essential utilities
# shellcheck disable=SC1091
source "${var_LIB_UTILS_DIR}/log.sh"
# shellcheck disable=SC1091
source "${var_LIB_UTILS_DIR}/flow.sh"

################################################################################
# Phase Handler Functions
################################################################################

#######################################
# Initialize a phase handler
# Sets up common environment and logging
# Globals:
#   PHASE_NAME
#   PHASE_START_TIME
# Arguments:
#   $1 - Phase name
#######################################
phase::init() {
    local phase_name="${1:-unknown}"
    export PHASE_NAME="$phase_name"
    export PHASE_START_TIME=$(date +%s)
    
    log::header "🚀 Starting $PHASE_NAME phase"
    log::debug "Phase handler: ${BASH_SOURCE[1]:-unknown}"
    log::debug "Project root: ${var_ROOT_DIR:-not set}"
    log::debug "Context: ${VROOLI_CONTEXT:-not set}"
}

#######################################
# Complete a phase handler
# Logs completion time and success
# Globals:
#   PHASE_NAME
#   PHASE_START_TIME
#######################################
phase::complete() {
    local end_time=$(date +%s)
    local duration=$((end_time - ${PHASE_START_TIME:-0}))
    
    log::success "✅ $PHASE_NAME phase completed in ${duration}s"
}

#######################################
# Check if running in CI environment
# Returns:
#   0 if in CI, 1 otherwise
#######################################
phase::is_ci() {
    [[ "${CI:-}" == "true" ]] || \
    [[ "${IS_CI:-}" == "yes" ]] || \
    [[ "${GITHUB_ACTIONS:-}" == "true" ]] || \
    [[ "${GITLAB_CI:-}" == "true" ]]
}

#######################################
# Check if running in monorepo context
# Returns:
#   0 if monorepo, 1 otherwise
#######################################
phase::is_monorepo() {
    [[ "${VROOLI_CONTEXT:-}" == "monorepo" ]] || \
    [[ -d "${var_PACKAGES_DIR:-}" ]]
}

#######################################
# Check if running in standalone context
# Returns:
#   0 if standalone, 1 otherwise
#######################################
phase::is_standalone() {
    [[ "${VROOLI_CONTEXT:-}" == "standalone" ]] || \
    ( [[ ! -d "${var_PACKAGES_DIR:-}" ]] && [[ -f "${var_SERVICE_JSON_FILE:-}" ]] )
}

#######################################
# Get configuration value from service.json
# Arguments:
#   $1 - JSON path (e.g., ".lifecycle.setup.description")
# Returns:
#   Configuration value or empty string
#######################################
phase::get_config() {
    local path="${1:-}"
    local config_file="${var_SERVICE_JSON_FILE:-}"
    
    if [[ -z "$path" ]] || [[ ! -f "$config_file" ]]; then
        echo ""
        return
    fi
    
    if command -v jq &> /dev/null; then
        jq -r "$path // empty" "$config_file" 2>/dev/null || echo ""
    else
        echo ""
    fi
}

#######################################
# Execute a command with proper error handling
# Arguments:
#   $@ - Command and arguments
# Returns:
#   Command exit code
#######################################
phase::execute() {
    local cmd="$*"
    log::debug "Executing: $cmd"
    
    if eval "$cmd"; then
        return 0
    else
        local exit_code=$?
        log::error "Command failed with exit code $exit_code: $cmd"
        return $exit_code
    fi
}

#######################################
# Execute a command conditionally based on context
# Arguments:
#   $1 - Context (monorepo|standalone|all)
#   $@ - Command and arguments
# Returns:
#   0 if skipped or successful, command exit code otherwise
#######################################
phase::execute_if() {
    local context="$1"
    shift
    
    case "$context" in
        monorepo)
            if ! phase::is_monorepo; then
                log::debug "Skipping monorepo-only command: $*"
                return 0
            fi
            ;;
        standalone)
            if ! phase::is_standalone; then
                log::debug "Skipping standalone-only command: $*"
                return 0
            fi
            ;;
        all)
            # Execute for all contexts
            ;;
        *)
            log::warning "Unknown context: $context"
            return 1
            ;;
    esac
    
    phase::execute "$@"
}

#######################################
# Source a script if it exists
# Arguments:
#   $1 - Script path
# Returns:
#   0 if sourced or not found, 1 on error
#######################################
phase::source_if_exists() {
    local script="$1"
    
    if [[ -f "$script" ]]; then
        log::debug "Sourcing: $script"
        # shellcheck disable=SC1090
        source "$script" || {
            log::error "Failed to source: $script"
            return 1
        }
    else
        log::debug "Script not found (skipping): $script"
    fi
    
    return 0
}

#######################################
# Run a hook if defined in service.json
# Arguments:
#   $1 - Hook name (e.g., "preSetup", "postBuild")
# Returns:
#   0 on success or if hook not defined
#######################################
phase::run_hook() {
    local hook_name="$1"
    local hook_cmd
    
    # Try to get hook from lifecycle.hooks.<hookName>
    hook_cmd=$(phase::get_config ".lifecycle.hooks.${hook_name}")
    
    if [[ -z "$hook_cmd" ]]; then
        # Try legacy location
        hook_cmd=$(phase::get_config ".repository.hooks.${hook_name}")
    fi
    
    if [[ -n "$hook_cmd" ]]; then
        log::info "Running $hook_name hook..."
        if phase::execute "$hook_cmd"; then
            log::success "✅ $hook_name hook completed"
        else
            log::warning "⚠️  $hook_name hook failed"
            return 1
        fi
    else
        log::debug "No $hook_name hook defined"
    fi
    
    return 0
}

#######################################
# Export common environment variables
# Sets up standard variables for phase execution
#######################################
phase::export_env() {
    # Ensure PROJECT_ROOT is set for backward compatibility
    if [[ -z "${PROJECT_ROOT:-}" ]]; then
        export PROJECT_ROOT="${var_ROOT_DIR}"
    fi
    
    # Export standard paths for backward compatibility
    export SCRIPTS_DIR="${var_SCRIPTS_DIR}"
    export LIB_DIR="${var_LIB_DIR}"
    export APP_DIR="${var_APP_DIR}"
    
    # Export context if not set
    if [[ -z "${VROOLI_CONTEXT:-}" ]]; then
        if [[ -d "${var_PACKAGES_DIR:-}" ]]; then
            export VROOLI_CONTEXT="monorepo"
        else
            export VROOLI_CONTEXT="standalone"
        fi
    fi
    
    log::debug "Environment exported:"
    log::debug "  PROJECT_ROOT=${var_ROOT_DIR}"
    log::debug "  VROOLI_CONTEXT=$VROOLI_CONTEXT"
    log::debug "  SCRIPTS_DIR=${var_SCRIPTS_DIR}"
}

# Export functions for use by phase handlers
export -f phase::init
export -f phase::complete
export -f phase::is_ci
export -f phase::is_monorepo
export -f phase::is_standalone
export -f phase::get_config
export -f phase::execute
export -f phase::execute_if
export -f phase::source_if_exists
export -f phase::run_hook
export -f phase::export_env